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
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

const testPassword = "undertow-test-password"

func TestMain(m *testing.M) {
	// Keep test output readable: silence all status lines.
	outInit(false, true)
	os.Exit(m.Run())
}

// newTestServer wires a server exactly the way main() does and starts it on
// a random loopback port. The returned address is its "host:port".
func newTestServer(t *testing.T, noShell bool) (*ssh.Server, string) {
	t.Helper()

	forwardHandler := &ssh.ForwardedTCPHandler{}
	server := &ssh.Server{
		Version:                       "OpenSSH_8.9",
		Handler:                       createSSHSessionHandler(defaultShell),
		PasswordHandler:               createPasswordHandler(testPassword),
		LocalPortForwardingCallback:   createLocalPortForwardingCallback(noShell),
		ReversePortForwardingCallback: createReversePortForwardingCallback(),
		SessionRequestCallback:        createSessionRequestCallback(noShell),
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": ssh.DirectTCPIPHandler,
			"session":      ssh.DefaultSessionHandler,
			"rs-info":      createExtraInfoHandler(),
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": createSFTPHandler(),
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { server.Serve(ln) }()
	t.Cleanup(func() {
		server.Close()
		ln.Close()
	})
	return server, ln.Addr().String()
}

func dialTestServer(t *testing.T, addr string) *gossh.Client {
	t.Helper()
	config := &gossh.ClientConfig{
		User:            "operator",
		Auth:            []gossh.AuthMethod{gossh.Password(testPassword)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

// echoCommand returns a command line that prints the marker on the test
// platform ("echo" is a shell builtin on Windows, not an executable).
func echoCommand(marker string) string {
	if runtime.GOOS == "windows" {
		return "cmd /c echo " + marker
	}
	return "echo " + marker
}

func TestRemoveDFlag(t *testing.T) {
	got := removeDFlag([]string{"-d", "-l", "-p", "4444", "-d"})
	want := []string{"-l", "-p", "4444"}
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("removeDFlag = %v, want %v", got, want)
	}
	if got := removeDFlag(nil); len(got) != 0 {
		t.Fatalf("removeDFlag(nil) = %v, want empty", got)
	}
}

func TestShouldDetach(t *testing.T) {
	oldDaemonize := DAEMONIZE
	t.Cleanup(func() { DAEMONIZE = oldDaemonize })

	// The detached child (env key set) must never detach again.
	t.Setenv(detachEnvKey, "1")
	DAEMONIZE = "1"
	if shouldDetach() {
		t.Fatal("detached child tried to detach again")
	}

	// Baked-in DAEMONIZE detaches on a fresh launch.
	t.Setenv(detachEnvKey, "")
	os.Unsetenv(detachEnvKey)
	if !shouldDetach() {
		t.Fatal("DAEMONIZE build did not detach on launch")
	}

	DAEMONIZE = ""
	if shouldDetach() {
		t.Fatal("plain launch detached (test binary has no -d in argv)")
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	cert, err := generateSelfSignedCert("login.example.com")
	if err != nil {
		t.Fatalf("generateSelfSignedCert: %v", err)
	}
	if len(cert.Certificate) == 0 {
		t.Fatal("no certificate in chain")
	}
	parsed, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.DNSNames) != 1 || parsed.DNSNames[0] != "login.example.com" {
		t.Fatalf("DNSNames = %v, want [login.example.com]", parsed.DNSNames)
	}
	if parsed.NotAfter.Before(time.Now().Add(300 * 24 * time.Hour)) {
		t.Fatalf("certificate lifetime too short: %v", parsed.NotAfter)
	}
	priv, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
	if !ok {
		t.Fatalf("private key type %T, want *ecdsa.PrivateKey", cert.PrivateKey)
	}
	if priv.Curve != elliptic.P256() {
		t.Fatalf("curve = %v, want P-256", priv.Curve.Params().Name)
	}
}

func TestPasswordHandler(t *testing.T) {
	_, addr := newTestServer(t, false)

	// Wrong password must be rejected over the wire.
	config := &gossh.ClientConfig{
		User:            "operator",
		Auth:            []gossh.AuthMethod{gossh.Password("wrong-password")},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	if client, err := gossh.Dial("tcp", addr, config); err == nil {
		client.Close()
		t.Fatal("wrong password authenticated")
	}

	// Right password must pass (dialTestServer fails the test otherwise).
	dialTestServer(t, addr)
}

// newPubKeyServer wires a server that only accepts the given authorized key.
func newPubKeyServer(t *testing.T, authorizedKey string) string {
	t.Helper()

	server := &ssh.Server{
		Version:          "OpenSSH_8.9",
		Handler:          createSSHSessionHandler(defaultShell),
		PublicKeyHandler: createPublicKeyHandler(authorizedKey),
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"session": ssh.DefaultSessionHandler,
		},
	}

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	go func() { server.Serve(ln) }()
	t.Cleanup(func() {
		server.Close()
		ln.Close()
	})
	return ln.Addr().String()
}

func TestPublicKeyHandler(t *testing.T) {
	if createPublicKeyHandler("") != nil {
		t.Fatal("empty authorized key should disable pubkey auth")
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	sshPub, err := gossh.NewPublicKey(pub)
	if err != nil {
		t.Fatalf("wrap key: %v", err)
	}
	addr := newPubKeyServer(t, string(gossh.MarshalAuthorizedKey(sshPub)))

	// The authorized key must authenticate.
	signer, err := gossh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatalf("signer: %v", err)
	}
	config := &gossh.ClientConfig{
		User:            "operator",
		Auth:            []gossh.AuthMethod{gossh.PublicKeys(signer)},
		HostKeyCallback: gossh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}
	client, err := gossh.Dial("tcp", addr, config)
	if err != nil {
		t.Fatalf("authorized key rejected: %v", err)
	}
	client.Close()

	// Any other key must be rejected.
	_, otherPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	otherSigner, err := gossh.NewSignerFromKey(otherPriv)
	if err != nil {
		t.Fatalf("other signer: %v", err)
	}
	config.Auth = []gossh.AuthMethod{gossh.PublicKeys(otherSigner)}
	if client, err := gossh.Dial("tcp", addr, config); err == nil {
		client.Close()
		t.Fatal("non-authorized key authenticated")
	}
}

func TestExecCommand(t *testing.T) {
	_, addr := newTestServer(t, false)
	client := dialTestServer(t, addr)

	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	defer session.Close()

	marker := "ut-exec-ok"
	out, err := session.CombinedOutput(echoCommand(marker))
	if err != nil {
		t.Fatalf("exec: %v (%s)", err, out)
	}
	if !strings.Contains(string(out), marker) {
		t.Fatalf("output %q missing %q", out, marker)
	}
}

func TestExecCommandDeniedInNoShellMode(t *testing.T) {
	_, addr := newTestServer(t, true)
	client := dialTestServer(t, addr)

	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	defer session.Close()
	if err := session.Run(echoCommand("nope")); err == nil {
		t.Fatal("exec succeeded with sessions denied")
	}
}

func TestPtyShell(t *testing.T) {
	_, addr := newTestServer(t, false)
	client := dialTestServer(t, addr)

	session, err := client.NewSession()
	if err != nil {
		t.Fatalf("session: %v", err)
	}
	defer session.Close()

	stdin, err := session.StdinPipe()
	if err != nil {
		t.Fatalf("stdin: %v", err)
	}
	var buf bytes.Buffer
	session.Stdout = &buf
	session.Stderr = &buf

	modes := gossh.TerminalModes{
		gossh.ECHO:          0,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 24, 80, modes); err != nil {
		t.Fatalf("request pty: %v", err)
	}
	if err := session.Shell(); err != nil {
		t.Fatalf("shell: %v", err)
	}

	marker := "ut-pty-ok"
	fmt.Fprintf(stdin, "echo %s\n", marker)

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if strings.Contains(buf.String(), marker) {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !strings.Contains(buf.String(), marker) {
		t.Fatalf("pty output missing %q; got %q", marker, buf.String())
	}
	fmt.Fprintln(stdin, "exit")
}

func TestSFTPRoundTrip(t *testing.T) {
	_, addr := newTestServer(t, false)
	client := dialTestServer(t, addr)

	sc, err := sftp.NewClient(client)
	if err != nil {
		t.Fatalf("sftp client: %v", err)
	}
	defer sc.Close()

	payload := []byte("undertow sftp round trip\n\x00\x01\x02")
	path := filepath.Join(t.TempDir(), "payload.bin")

	f, err := sc.Create(path)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.Write(payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	rf, err := sc.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	got, err := io.ReadAll(rf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	rf.Close()
	if !bytes.Equal(got, payload) {
		t.Fatalf("round trip mismatch: got %d bytes, want %d", len(got), len(payload))
	}
}

func TestDirectTCPIPForward(t *testing.T) {
	// Local echo server stands in for the service being tunnelled to.
	echoLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("echo listen: %v", err)
	}
	defer echoLn.Close()
	go func() {
		for {
			conn, err := echoLn.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()

	_, addr := newTestServer(t, false)
	client := dialTestServer(t, addr)

	// direct-tcpip through the SSH connection to the echo server.
	conn, err := client.Dial("tcp", echoLn.Addr().String())
	if err != nil {
		t.Fatalf("direct-tcpip dial: %v", err)
	}
	defer conn.Close()

	msg := []byte("ut-direct-tcpip")
	if _, err := conn.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	got := make([]byte, len(msg))
	if _, err := io.ReadFull(conn, got); err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, msg) {
		t.Fatalf("echo mismatch: got %q, want %q", got, msg)
	}
}

func TestDirectTCPIPDeniedInNoShellMode(t *testing.T) {
	_, addr := newTestServer(t, true)
	client := dialTestServer(t, addr)

	conn, err := client.Dial("tcp", "127.0.0.1:80")
	if err == nil {
		conn.Close()
		t.Fatal("direct-tcpip succeeded with forwarding denied")
	}
}

func TestReversePortForward(t *testing.T) {
	_, addr := newTestServer(t, false)
	client := dialTestServer(t, addr)

	// Ask the server to bind a random port and forward it back to us.
	remoteLn, err := client.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("remote listen: %v", err)
	}
	defer remoteLn.Close()

	// Connecting to the bound port on the server side must reach us.
	serverSide, err := net.Dial("tcp", remoteLn.Addr().String())
	if err != nil {
		t.Fatalf("dial forwarded port: %v", err)
	}
	defer serverSide.Close()

	type acceptResult struct {
		conn net.Conn
		err  error
	}
	accepted := make(chan acceptResult, 1)
	go func() {
		c, err := remoteLn.Accept()
		accepted <- acceptResult{c, err}
	}()

	select {
	case res := <-accepted:
		if res.err != nil {
			t.Fatalf("accept: %v", res.err)
		}
		defer res.conn.Close()
		msg := []byte("ut-reverse")
		if _, err := serverSide.Write(msg); err != nil {
			t.Fatalf("write: %v", err)
		}
		got := make([]byte, len(msg))
		res.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		if _, err := io.ReadFull(res.conn, got); err != nil {
			t.Fatalf("read: %v", err)
		}
		if !bytes.Equal(got, msg) {
			t.Fatalf("mismatch: got %q, want %q", got, msg)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("forwarded connection never arrived")
	}
}

func TestExtraInfoChannel(t *testing.T) {
	_, addr := newTestServer(t, false)
	client := dialTestServer(t, addr)

	info := ExtraInfo{CurrentUser: "svc", Hostname: "web01", ListeningAddress: "127.0.0.1:4444"}
	ch, _, err := client.OpenChannel("rs-info", gossh.Marshal(&info))
	if err == nil {
		ch.Close()
		t.Fatal("rs-info channel was accepted; the handler must reject with th4nkz")
	}
	if !strings.Contains(err.Error(), "th4nkz") {
		t.Fatalf("rejection reason %q missing th4nkz", err)
	}
}

func TestDialHomeAndListen(t *testing.T) {
	// dialHomeAndListen authenticates with the baked localPassword; point it
	// at the test password the way ldflags would at build time.
	oldPassword := localPassword
	localPassword = testPassword
	t.Cleanup(func() { localPassword = oldPassword })

	_, addr := newTestServer(t, false)

	ln, err := dialHomeAndListen("svc", addr, 0, false, false, "")
	if err != nil {
		t.Fatalf("dialHomeAndListen: %v", err)
	}
	defer ln.Close()

	bound := ln.Addr().String()
	if strings.HasSuffix(bound, ":0") {
		t.Fatalf("no real port allocated: %s", bound)
	}

	// Sessions through the forwarded listener must reach the payload side.
	go func() {
		conn, err := ln.Accept()
		if err == nil {
			conn.Close()
		}
	}()
	home, err := net.Dial("tcp", bound)
	if err != nil {
		t.Fatalf("dial home-bound port: %v", err)
	}
	home.Close()
}

func TestSetupParametersWithoutCLI(t *testing.T) {
	p := setupParametersWithoutCLI("1", "login.example.com")
	if p.listen {
		t.Fatal("NOCLI build must not listen for flags")
	}
	if p.shell != defaultShell {
		t.Fatalf("shell = %q, want %q", p.shell, defaultShell)
	}
	if !p.tlsWrap {
		t.Fatal("tlsWrap not picked up")
	}
	if p.tlsSNI != "login.example.com" {
		t.Fatalf("tlsSNI = %q", p.tlsSNI)
	}
	if p.LPORT == 0 || p.LPORT > 65535 {
		t.Fatalf("implausible LPORT %d", p.LPORT)
	}
}

func TestOutTokens(t *testing.T) {
	outNoColor.Store(true)
	if got := outPaint(cMint, cMint16, "[ok]", "done"); got != "[ok] done" {
		t.Fatalf("no-colour paint = %q", got)
	}
	outNoColor.Store(false)
	outANSI16.Store(true)
	if got := outPaint(cMint, cMint16, "[ok]", "done"); got != cMint16+"[ok]"+cReset+" done" {
		t.Fatalf("16-colour paint = %q", got)
	}
	outANSI16.Store(false)
	if got := outPaint(cMint, cMint16, "[ok]", "done"); got != cMint+"[ok]"+cReset+" done" {
		t.Fatalf("truecolour paint = %q", got)
	}
	// Quiet mode suppresses everything, including failures.
	outQuiet.Store(true)
	defer outQuiet.Store(false)

	// Verbose gate: ok/info/warn require verbose; fail requires only !quiet.
	outVerbose.Store(false)
	if outVerbose.Load() {
		t.Fatal("verbose gate stuck")
	}
}

func TestVersionString(t *testing.T) {
	// The release workflow greps main.go for this exact shape.
	if version == "" || strings.HasSuffix(version, " ") {
		t.Fatalf("implausible version %q", version)
	}
}
