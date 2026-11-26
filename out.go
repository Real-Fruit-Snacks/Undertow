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

// Terminal output in the RFS Cyclorama Dawn palette (BRAND.md §11).
// Every status line starts with a bracketed ASCII tag — colour is never
// the only signal, and output stays meaningful under NO_COLOR or a pipe.
package main

import (
	"fmt"
	"os"
	"sync/atomic"

	"golang.org/x/term"
)

var (
	outNoColor atomic.Bool
	outANSI16  atomic.Bool
	outVerbose atomic.Bool
	outQuiet   atomic.Bool
)

// outInit wires the gates. verbose enables the ok/info/warn status lines
// (the old log package gate); quiet suppresses everything, including
// failures, for NOCLI and daemonized payloads on a target.
func outInit(verbose, quiet bool) {
	outVerbose.Store(verbose)
	outQuiet.Store(quiet)

	disabled := false
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		disabled = true
	}
	if os.Getenv("TERM") == "dumb" {
		disabled = true
	}
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		disabled = true
	}
	outNoColor.Store(disabled)

	ct := os.Getenv("COLORTERM")
	outANSI16.Store(ct != "truecolor" && ct != "24bit")
}

// Cyclorama Dawn tokens (BRAND.md §11), truecolour dark row.
const (
	cBeacon = "\x1b[38;2;255;146;84m"  // #FF9254
	cMint   = "\x1b[38;2;61;220;151m"  // #3DDC97
	cAmber  = "\x1b[38;2;255;194;71m"  // #FFC247
	cFault  = "\x1b[38;2;255;92;108m"  // #FF5C6C Signal Crimson
	cSteel  = "\x1b[38;2;116;184;232m" // #74B8E8
	cMute   = "\x1b[38;2;179;181;192m" // #B3B5C0
	// §11's 16-colour row, for terminals without truecolour.
	cBeacon16 = "\x1b[91m" // bright red
	cMint16   = "\x1b[32m" // green
	cAmber16  = "\x1b[33m" // yellow
	cFault16  = "\x1b[31m" // red
	cSteel16  = "\x1b[36m" // cyan
	cMute16   = "\x1b[90m" // bright black; §11 defines no 16-colour mute
	cReset    = "\x1b[0m"
)

func outPaint(truecolour, basic, tag, msg string) string {
	if outNoColor.Load() {
		return tag + " " + msg
	}
	if outANSI16.Load() {
		return basic + tag + cReset + " " + msg
	}
	return truecolour + tag + cReset + " " + msg
}

// outOkf reports something that succeeded. Mint is only ever "OK".
func outOkf(format string, args ...any) {
	if outQuiet.Load() || !outVerbose.Load() {
		return
	}
	fmt.Fprintln(os.Stdout, outPaint(cMint, cMint16, "[ok]", fmt.Sprintf(format, args...)))
}

// outWarnf reports something the operator should see before continuing.
// Amber is caution, not failure.
func outWarnf(format string, args ...any) {
	if outQuiet.Load() || !outVerbose.Load() {
		return
	}
	fmt.Fprintln(os.Stderr, outPaint(cAmber, cAmber16, "[!]", fmt.Sprintf(format, args...)))
}

// outInfof reports neutral progress detail. Steel.
func outInfof(format string, args ...any) {
	if outQuiet.Load() || !outVerbose.Load() {
		return
	}
	fmt.Fprintln(os.Stdout, outPaint(cSteel, cSteel16, "[i]", fmt.Sprintf(format, args...)))
}

// outPlanf reports a step the tool will take. Muted.
func outPlanf(format string, args ...any) {
	if outQuiet.Load() || !outVerbose.Load() {
		return
	}
	fmt.Fprintln(os.Stdout, outPaint(cMute, cMute16, "[i]", fmt.Sprintf(format, args...)))
}

// outFailf reports something that failed. Signal Crimson, never red — it
// cannot be mistaken for Beacon. Always printed unless quiet.
func outFailf(format string, args ...any) {
	if outQuiet.Load() {
		return
	}
	fmt.Fprintln(os.Stderr, outPaint(cFault, cFault16, "[x]", fmt.Sprintf(format, args...)))
}

// outFatalf reports the failure and exits 1 (runtime error).
func outFatalf(format string, args ...any) {
	outFailf(format, args...)
	os.Exit(1)
}

// outPromptf prints a beacon prompt-style line, used sparingly.
func outPromptf(format string, args ...any) {
	if outQuiet.Load() {
		return
	}
	fmt.Fprintln(os.Stdout, outPaint(cBeacon, cBeacon16, "$", fmt.Sprintf(format, args...)))
}
