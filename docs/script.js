// shell-tunnel — Cyclorama Dawn v2.1.0 interactive controller
// Theme toggle (LX 263 Dark default / LX 512 Light opt-in, dual-key storage sync, View Transition),
// Live Pivot Topology & Binary Baker, ControlMaster Multiplexer tabs, and Invoker Command copy + toast.
(function () {
    'use strict';

    var root = document.documentElement;
    var reducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)');
    var themeBtn = document.getElementById('theme');
    var toastEl = document.getElementById('rfs-toast');
    var toastBody = document.getElementById('rfs-toast-body');
    var toastTimer = null;
    var lastToastTrigger = null;
    var lastToastMsg = '';

    // ==================== TOAST NOTIFICATIONS (popover="manual") ====================
    function hideToast() {
        clearTimeout(toastTimer);
        if (!toastEl) return;
        var hadFocus = toastEl.contains(document.activeElement);
        try {
            if (typeof toastEl.hidePopover === 'function' && toastEl.matches(':popover-open')) {
                toastEl.hidePopover();
            } else {
                toastEl.classList.remove(':popover-open');
                toastEl.hidden = true;
            }
        } catch (e) {
            toastEl.classList.remove(':popover-open');
            toastEl.hidden = true;
        }
        if (hadFocus && lastToastTrigger && typeof lastToastTrigger.focus === 'function') {
            lastToastTrigger.focus({ preventScroll: true });
        }
    }

    function scheduleToastDismiss() {
        clearTimeout(toastTimer);
        toastTimer = setTimeout(function () {
            if (toastEl && toastEl.contains(document.activeElement)) return;
            hideToast();
        }, 2600);
    }

    function showToast(msg, kind, triggerEl) {
        if (!toastEl || !toastBody) return;
        lastToastTrigger = triggerEl || document.activeElement;
        var map = { ok: '[ok]', fail: '[x]', warn: '[!]', info: '[i]' };
        var tone = map[kind] ? kind : 'ok';
        var tag = map[tone];
        var announced = msg;
        if (announced === lastToastMsg) announced += '\u00A0';
        lastToastMsg = announced;
        toastEl.className = 'rfs-toast is-' + tone;
        toastBody.textContent = '';
        var tagSpan = document.createElement('span');
        tagSpan.className = 'k';
        tagSpan.textContent = tag;
        toastBody.appendChild(tagSpan);
        toastBody.appendChild(document.createTextNode(' ' + announced));

        try {
            if (typeof toastEl.showPopover === 'function') {
                if (toastEl.matches(':popover-open')) toastEl.hidePopover();
                toastEl.showPopover();
            } else {
                toastEl.hidden = false;
                toastEl.classList.add(':popover-open');
            }
        } catch (e) {
            toastEl.hidden = false;
            toastEl.classList.add(':popover-open');
        }

        if (typeof navigator.ariaNotify === 'function') {
            try { navigator.ariaNotify(tag + ' ' + msg, { priority: 'normal' }); } catch (e) {}
        }
        scheduleToastDismiss();
    }

    if (toastEl) {
        toastEl.addEventListener('focusin', function () { clearTimeout(toastTimer); });
        toastEl.addEventListener('focusout', function (ev) {
            if (!toastEl.contains(ev.relatedTarget)) scheduleToastDismiss();
        });
        toastEl.addEventListener('click', function (ev) {
            if (ev.target.closest('[popovertargetaction="hide"]')) hideToast();
        });
    }

    // ==================== THEME: LX 263 DARK DEFAULT / LX 512 LIGHT OPT-IN ====================
    function isLight() {
        return root.dataset.theme === 'light';
    }

    function readTheme() {
        try {
            return localStorage.getItem('rfs-theme') || localStorage.getItem('color-scheme');
        } catch (e) {
            return null;
        }
    }

    function syncThemeMeta() {
        var light = isLight();
        var mc = document.querySelector('meta[name="theme-color"]');
        if (mc) mc.content = light ? '#F5F7F9' : '#050507';
        var cs = document.querySelector('meta[name="color-scheme"]');
        if (cs) cs.content = light ? 'light' : 'dark';
        if (themeBtn) {
            themeBtn.setAttribute('aria-pressed', String(light));
            themeBtn.setAttribute('aria-label', light ? 'Switch to LX 263 dark theme' : 'Switch to LX 512 light theme');
        }
    }

    function applyTheme(next) {
        if (next === 'light' || next === 'dayshift') {
            root.dataset.theme = 'light';
        } else {
            delete root.dataset.theme;
        }
        syncThemeMeta();
    }

    if (themeBtn) {
        themeBtn.addEventListener('click', function () {
            var next = isLight() ? 'dark' : 'light';
            var commit = function () {
                applyTheme(next);
                try {
                    localStorage.setItem('rfs-theme', next);
                    localStorage.setItem('color-scheme', next);
                } catch (e) {}
            };
            if (!document.startViewTransition || reducedMotion.matches) {
                commit();
                return;
            }
            var rect = themeBtn.getBoundingClientRect();
            var x = rect.left + rect.width / 2;
            var y = rect.top + rect.height / 2;
            var endRadius = Math.hypot(Math.max(x, window.innerWidth - x), Math.max(y, window.innerHeight - y));
            var transition = document.startViewTransition(commit);
            var animated = transition.ready.then(function () {
                root.animate(
                    {
                        clipPath: [
                            'circle(0px at ' + x + 'px ' + y + 'px)',
                            'circle(' + endRadius + 'px at ' + x + 'px ' + y + 'px)'
                        ]
                    },
                    {
                        duration: 520,
                        easing: 'cubic-bezier(.22, .61, .36, 1)',
                        pseudoElement: '::view-transition-new(root)'
                    }
                );
            });
            Promise.allSettled([animated, transition.updateCallbackDone, transition.finished]);
        });
    }

    window.addEventListener('storage', function (ev) {
        if (ev.key === 'rfs-theme' || ev.key === 'color-scheme' || ev.key === null) {
            applyTheme(ev.newValue);
        }
    });

    var initialTheme = readTheme();
    if (initialTheme) applyTheme(initialTheme);
    else syncThemeMeta();

    // ==================== COPY BUTTONS (Invoker Commands + Fallback) ====================
    var copyTimers = new WeakMap();

    function legacyCopy(text) {
        var prevFocus = document.activeElement;
        var field = document.createElement('textarea');
        field.value = text;
        field.setAttribute('readonly', '');
        field.style.cssText = 'position:fixed;inset-block-start:0;opacity:0';
        document.body.appendChild(field);
        field.select();
        try {
            return document.execCommand('copy');
        } catch (e) {
            return false;
        } finally {
            field.remove();
            if (prevFocus && typeof prevFocus.focus === 'function') {
                prevFocus.focus({ preventScroll: true });
            }
        }
    }

    function reportCopy(button, ok) {
        clearTimeout(copyTimers.get(button));
        button.removeAttribute('data-copied');
        button.removeAttribute('data-failed');
        var labelEl = button.querySelector('.copy-label');
        if (labelEl && !button.dataset.origLabel) {
            button.dataset.origLabel = labelEl.textContent;
        }
        button.setAttribute(ok ? 'data-copied' : 'data-failed', '');
        if (labelEl) labelEl.textContent = ok ? 'Copied' : 'Failed';
        showToast(ok ? 'Command copied to clipboard.' : 'Clipboard access denied; copy manually.', ok ? 'ok' : 'fail', button);

        copyTimers.set(button, setTimeout(function () {
            button.removeAttribute('data-copied');
            button.removeAttribute('data-failed');
            if (labelEl && button.dataset.origLabel) {
                labelEl.textContent = button.dataset.origLabel;
            }
        }, 1800));
    }

    function copyFromTarget(button, target) {
        if (!target) return;
        var text = (target.innerText || target.textContent || '').trim();
        if (!text) return;
        if (navigator.clipboard && typeof navigator.clipboard.writeText === 'function') {
            navigator.clipboard.writeText(text).then(
                function () { reportCopy(button, true); },
                function () { reportCopy(button, legacyCopy(text)); }
            );
        } else {
            reportCopy(button, legacyCopy(text));
        }
    }

    if ('CommandEvent' in window) {
        document.addEventListener('command', function (ev) {
            if (ev.command === '--copy') {
                copyFromTarget(ev.source, ev.target);
            }
        }, true);
    } else {
        document.addEventListener('click', function (ev) {
            if (ev.defaultPrevented) return;
            var btn = ev.target.closest('button[command="--copy"]');
            if (!btn) return;
            var targetId = btn.getAttribute('commandfor');
            var target = targetId ? document.getElementById(targetId) : null;
            if (target) copyFromTarget(btn, target);
        });
    }

    // ==================== INTERACTIVE PIVOT TOPOLOGY & BINARY BAKER ====================
    var workbench = document.getElementById('workbench');
    if (workbench) {
        var state = {
            mode: 'reverse',
            tls: false,
            nocli: false,
            daemonize: false,
            os: 'linux/amd64'
        };

        var OS_META = {
            'linux/amd64': {
                sub: 'linux/amd64 · static binary · openpty',
                shortArtifact: 'artifact: *-linux-x64',
                revFile: 'shell-tunnel-10-10-14-5-4444-linux-x64',
                bindFile: 'shell-tunnel-listen-8888-linux-x64',
                extraNote: '(build.sh also emits linux-x86, windows-x64/x86)'
            },
            'windows/amd64': {
                sub: 'windows/amd64 · static · ConPTY',
                shortArtifact: 'artifact: *-windows-x64.exe',
                revFile: 'shell-tunnel-10-10-14-5-4444-windows-x64.exe',
                bindFile: 'shell-tunnel-listen-8888-windows-x64.exe',
                extraNote: '(build.sh also emits windows-x86.exe, linux-x64/x86)'
            },
            'darwin/arm64': {
                sub: 'darwin/arm64 · static · unsigned',
                shortArtifact: 'artifact: *-darwin-arm64',
                revFile: 'shell-tunnel-10-10-14-5-4444-darwin-arm64',
                bindFile: 'shell-tunnel-listen-8888-darwin-arm64',
                extraNote: '(direct GOOS=darwin GOARCH=arm64 go build; first run needs xattr -d com.apple.quarantine)'
            }
        };

        var els = {
            routeLabel: document.getElementById('topo-route-label'),
            modeBadge: document.getElementById('topo-mode-badge'),
            tlsBadge: document.getElementById('topo-tls-badge'),
            flowReverse: document.getElementById('svg-flow-reverse'),
            flowBind: document.getElementById('svg-flow-bind'),
            opSlot1Label: document.getElementById('svg-op-slot1-label'),
            opSlot1Port: document.getElementById('svg-op-slot1-port'),
            opSlot2Label: document.getElementById('svg-op-slot2-label'),
            opSlot2Port: document.getElementById('svg-op-slot2-port'),
            opSlot2Meta: document.getElementById('svg-op-slot2-meta'),
            opDotTop: document.getElementById('svg-op-dot-top'),
            tgtOsSub: document.getElementById('svg-target-os-sub'),
            tgtSlot1Label: document.getElementById('svg-tgt-slot1-label'),
            tgtSlot1Port: document.getElementById('svg-tgt-slot1-port'),
            tgtProfileLine: document.getElementById('svg-tgt-profile-line'),
            tgtBinaryName: document.getElementById('svg-tgt-binary-name'),
            tgtDotTop: document.getElementById('svg-tgt-dot-top'),
            conduitTitle: document.getElementById('svg-conduit-title'),
            revTopLabel: document.getElementById('svg-rev-top-label'),
            bindTopLabel: document.getElementById('svg-bind-top-label'),
            cmdMeta: document.getElementById('baker-cmd-meta'),
            cmdCode: document.getElementById('baker-cmd'),
            statusLines: document.getElementById('baker-status-lines'),
            outHead: document.getElementById('baker-out-head'),
            artifacts: document.getElementById('baker-artifacts')
        };

        function tok(cls, text) {
            var span = document.createElement('span');
            span.className = cls;
            span.textContent = text;
            return span;
        }

        function buildHighlightedCommand() {
            var frag = document.createDocumentFragment();
            if (state.os === 'darwin/arm64') {
                frag.appendChild(tok('rfs-tok-kw', 'CGO_ENABLED=0 GOOS=darwin GOARCH=arm64'));
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-fn', 'go build'));
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-kw', '-ldflags='));

                var ld = ['-s -w'];
                if (state.mode === 'reverse') {
                    ld.push("-X 'main.LHOST=10.10.14.5'");
                    ld.push("-X 'main.LPORT=4444'");
                    ld.push("-X 'main.BPORT=0'");
                    ld.push("-X 'main.LUSER=svc'");
                } else {
                    ld.push("-X 'main.LPORT=8888'");
                }
                ld.push("-X 'main.localPassword=$(openssl rand -hex 8)'");
                if (state.tls) ld.push("-X 'main.TLS_WRAP=1'");
                if (state.nocli) ld.push("-X 'main.NOCLI=1'");
                if (state.daemonize) ld.push("-X 'main.DAEMONIZE=1'");

                frag.appendChild(tok('rfs-tok-str', '"' + ld.join(' ') + '"'));
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-kw', '-o'));
                frag.appendChild(document.createTextNode(' '));
                var outBin = state.mode === 'reverse' ? OS_META['darwin/arm64'].revFile : OS_META['darwin/arm64'].bindFile;
                frag.appendChild(tok('rfs-tok-str', 'bin/' + outBin));
                frag.appendChild(document.createTextNode(' .'));
                return frag;
            }

            frag.appendChild(tok('rfs-tok-fn', './build.sh'));
            frag.appendChild(document.createTextNode(' '));
            if (state.mode === 'reverse') {
                frag.appendChild(tok('rfs-tok-kw', 'reverse'));
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-num', '10.10.14.5:4444'));
            } else {
                frag.appendChild(tok('rfs-tok-kw', 'listen'));
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-num', '8888'));
            }

            if (state.tls) {
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-kw', '--tls'));
            }
            if (state.nocli) {
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-kw', '--nocli'));
            }
            if (state.daemonize) {
                frag.appendChild(document.createTextNode(' '));
                frag.appendChild(tok('rfs-tok-kw', '--daemonize'));
            }
            return frag;
        }

        function setStatusLine(container, tagClass, tagText, bodyText) {
            var p = document.createElement('p');
            p.className = 'status-line';
            var s = document.createElement('span');
            s.className = tagClass;
            s.textContent = tagText;
            p.appendChild(s);
            p.appendChild(document.createTextNode(' ' + bodyText));
            container.appendChild(p);
        }

        function setArtifactLine(container, codeText, noteText) {
            var p = document.createElement('p');
            p.className = 'artifact-line';
            var c = document.createElement('code');
            c.textContent = codeText;
            p.appendChild(c);
            if (noteText) {
                p.appendChild(document.createTextNode(' '));
                var n = document.createElement('span');
                n.className = 'artifact-note';
                n.textContent = noteText;
                p.appendChild(n);
            }
            container.appendChild(p);
        }

        function renderWorkbench() {
            var isRev = state.mode === 'reverse';
            var osInfo = OS_META[state.os];

            workbench.dataset.mode = state.mode;
            workbench.dataset.tls = String(state.tls);
            workbench.dataset.nocli = String(state.nocli);
            workbench.dataset.daemonize = String(state.daemonize);
            workbench.dataset.os = state.os;

            // Header telemetry
            els.routeLabel.textContent = isRev
                ? 'Target dials 10.10.14.5:4444 · Operator connects to 127.0.0.1:41827'
                : 'Target listens on target:8888 · Operator connects directly via ssh';

            els.modeBadge.className = 'rfs-badge ' + (isRev ? 'b-ok' : 'b-steel');
            els.modeBadge.textContent = '';
            var mbKey = document.createElement('span');
            mbKey.className = 'k';
            mbKey.textContent = isRev ? '[rev]' : '[bind]';
            els.modeBadge.appendChild(mbKey);
            els.modeBadge.appendChild(document.createTextNode(isRev ? ' REVERSE CALLBACK' : ' BIND LISTENER'));

            els.tlsBadge.hidden = !state.tls;

            // SVG dynamic arrows & ports
            els.flowReverse.classList.toggle('svg-hidden', !isRev);
            els.flowBind.classList.toggle('svg-hidden', isRev);
            els.conduitTitle.textContent = state.tls ? 'TLS-WRAPPED SSH TUNNEL' : 'ENCRYPTED SSH TUNNEL';

            if (isRev) {
                els.opSlot1Label.textContent = 'HANDLER CALLBACK LISTENER';
                els.opSlot1Port.textContent = '0.0.0.0:4444';
                els.opSlot1Port.setAttribute('class', 'svg-port svg-fill-beacon');
                els.opSlot2Label.textContent = 'ALLOCATED LOOPBACK BIND';
                els.opSlot2Port.textContent = '127.0.0.1:41827';
                els.opSlot2Meta.textContent = '/tmp/shell-tunnel-41827.sock';
                els.opDotTop.setAttribute('class', 'svg-port-dot svg-dot-beacon');
                els.tgtDotTop.setAttribute('class', 'svg-port-dot svg-dot-beacon');

                els.tgtSlot1Label.textContent = 'OUTBOUND CALLBACK TARGET';
                els.tgtSlot1Port.textContent = '10.10.14.5:4444';
                els.tgtSlot1Port.setAttribute('class', 'svg-port svg-fill-beacon');
                els.revTopLabel.textContent = state.tls
                    ? 'TLS+SSH dial :4444 (SNI www.microsoft.com)'
                    : 'dial home :4444 (retry 5s · keepalive 15s)';
            } else {
                els.opSlot1Label.textContent = 'STOCK SSH CLIENT';
                els.opSlot1Port.textContent = 'ssh -p 8888';
                els.opSlot1Port.setAttribute('class', 'svg-port svg-fill-steel');
                els.opSlot2Label.textContent = 'DIRECT TARGET SESSION';
                els.opSlot2Port.textContent = 'svc@target:8888';
                els.opSlot2Meta.textContent = 'Direct connect · no handler';
                els.opDotTop.setAttribute('class', 'svg-port-dot svg-dot-steel');
                els.tgtDotTop.setAttribute('class', 'svg-port-dot svg-dot-steel');

                els.tgtSlot1Label.textContent = 'INBOUND SSH LISTENER';
                els.tgtSlot1Port.textContent = 'target:8888';
                els.tgtSlot1Port.setAttribute('class', 'svg-port svg-fill-steel');
                els.bindTopLabel.textContent = state.tls
                    ? 'TLS listener :8888 (SNI www.microsoft.com)'
                    : 'ssh -p 8888 svc@target (direct TCP)';
            }

            els.tgtOsSub.textContent = osInfo.sub;
            var profileParts = [
                state.nocli ? '--nocli' : 'CLI on',
                state.daemonize ? '--daemonize' : 'foreground'
            ];
            els.tgtProfileLine.textContent = profileParts.join(' · ');
            els.tgtBinaryName.textContent = osInfo.shortArtifact;

            // Update button pressed states
            workbench.querySelectorAll('[data-set-mode]').forEach(function (b) {
                b.setAttribute('aria-pressed', String(b.dataset.setMode === state.mode));
            });
            workbench.querySelectorAll('[data-toggle-opt]').forEach(function (b) {
                b.setAttribute('aria-pressed', String(Boolean(state[b.dataset.toggleOpt])));
            });
            workbench.querySelectorAll('[data-set-os]').forEach(function (b) {
                b.setAttribute('aria-pressed', String(b.dataset.setOs === state.os));
            });

            // Command card
            var toolName = state.os === 'darwin/arm64' ? 'go build' : 'build.sh';
            els.cmdMeta.textContent = toolName + ' · ' + (isRev ? 'reverse' : 'listen') + ' · ' + state.os;
            els.cmdCode.textContent = '';
            els.cmdCode.appendChild(buildHighlightedCommand());

            // Verbatim status lines (exact core.go strings)
            els.statusLines.textContent = '';
            if (isRev) {
                setStatusLine(
                    els.statusLines,
                    'st-ok',
                    '[ok]',
                    state.tls ? 'TLS listening on 0.0.0.0:4444' : 'listening on 0.0.0.0:4444'
                );
                setStatusLine(els.statusLines, 'st-ok', '[ok]', 'listening at home on 127.0.0.1:41827');
            } else {
                setStatusLine(
                    els.statusLines,
                    'st-ok',
                    '[ok]',
                    state.tls ? 'TLS listening on 0.0.0.0:8888' : 'listening on 0.0.0.0:8888'
                );
                setStatusLine(
                    els.statusLines,
                    state.tls ? 'st-warn' : 'st-ok',
                    state.tls ? '[!]' : '[ok]',
                    state.tls
                        ? 'Self-signed runtime cert active (SNI: www.microsoft.com)'
                        : 'Successful authentication with password from svc@10.10.14.5'
                );
            }

            // Artifact lines
            els.outHead.textContent = state.os === 'darwin/arm64'
                ? 'Cross-compiled output in bin/'
                : 'Generated artifacts in bin/';
            els.artifacts.textContent = '';
            var mainFile = isRev ? osInfo.revFile : osInfo.bindFile;
            setArtifactLine(els.artifacts, 'bin/' + mainFile, osInfo.extraNote);
            if (isRev && state.os !== 'darwin/arm64') {
                setArtifactLine(
                    els.artifacts,
                    'bin/handler-10-10-14-5-4444',
                    '(chmod 700 · starts listener + prints ControlMaster commands)'
                );
            } else if (isRev && state.os === 'darwin/arm64') {
                setArtifactLine(
                    els.artifacts,
                    './shell-tunnel -l -v -p 4444',
                    '(start handler on operator host with matching localPassword)'
                );
            } else {
                setArtifactLine(
                    els.artifacts,
                    'ssh -p 8888 svc@<target-ip>',
                    '(connect directly with baked password or ED25519 key)'
                );
            }
        }

        workbench.addEventListener('click', function (ev) {
            var modeBtn = ev.target.closest('[data-set-mode]');
            if (modeBtn) {
                state.mode = modeBtn.dataset.setMode;
                renderWorkbench();
                return;
            }
            var optBtn = ev.target.closest('[data-toggle-opt]');
            if (optBtn) {
                var key = optBtn.dataset.toggleOpt;
                state[key] = !state[key];
                renderWorkbench();
                return;
            }
            var osBtn = ev.target.closest('[data-set-os]');
            if (osBtn) {
                state.os = osBtn.dataset.setOs;
                renderWorkbench();
            }
        });

        renderWorkbench();
    }

    // ==================== CONTROLMASTER MULTIPLEXER TABS ====================
    var muxBay = document.getElementById('mux-bay');
    if (muxBay) {
        var tablist = muxBay.querySelector('[role="tablist"]');
        var tabs = [].slice.call(muxBay.querySelectorAll('[role="tab"]'));
        var panels = [].slice.call(muxBay.querySelectorAll('[role="tabpanel"]'));

        function selectTab(targetTab, shouldFocus) {
            var idx = tabs.indexOf(targetTab);
            if (idx < 0) return;
            tabs.forEach(function (tab, i) {
                var active = i === idx;
                tab.setAttribute('aria-selected', String(active));
                tab.tabIndex = active ? 0 : -1;
                if (active) tab.setAttribute('focusgroupstart', '');
                else tab.removeAttribute('focusgroupstart');
            });
            panels.forEach(function (panel, i) {
                if (i === idx) {
                    panel.removeAttribute('hidden');
                } else {
                    panel.setAttribute('hidden', 'until-found');
                }
            });
            if (shouldFocus) {
                targetTab.focus();
            }
            if (typeof targetTab.scrollIntoView === 'function') {
                try {
                    targetTab.scrollIntoView({ block: 'nearest', inline: 'nearest', container: 'nearest' });
                } catch (e) {
                    targetTab.scrollIntoView({ block: 'nearest', inline: 'nearest' });
                }
            }
        }

        tabs.forEach(function (tab) {
            tab.addEventListener('click', function () {
                selectTab(tab, false);
            });
        });

        if (tablist) {
            tablist.addEventListener('keydown', function (ev) {
                var current = tabs.indexOf(document.activeElement);
                if (current < 0) return;
                var next = current;
                if (ev.key === 'ArrowRight') next = (current + 1) % tabs.length;
                else if (ev.key === 'ArrowLeft') next = (current - 1 + tabs.length) % tabs.length;
                else if (ev.key === 'Home') next = 0;
                else if (ev.key === 'End') next = tabs.length - 1;
                else return;
                ev.preventDefault();
                selectTab(tabs[next], true);
            });
        }

        // Sync tab selection when Find-in-Page matches inside a hidden="until-found" panel
        muxBay.addEventListener('beforematch', function (ev) {
            var matchedPanel = ev.target.closest('[role="tabpanel"]');
            if (!matchedPanel) return;
            var labelId = matchedPanel.getAttribute('aria-labelledby');
            var ownerTab = labelId ? document.getElementById(labelId) : null;
            if (ownerTab) selectTab(ownerTab, false);
        });
    }
})();
