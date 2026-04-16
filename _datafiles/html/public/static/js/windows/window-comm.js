/**
 * window-comm.js
 *
 * Virtual window: Communications (tabbed chat channels).
 *
 * Responds to GMCP namespace:
 *   Comm  - incoming channel message
 *
 * Reads: Client.GMCPStructs.Comm.Channel
 *
 * Channels are defined in the CHANNELS constant below. Add entries there
 * to expose additional tabs without touching any other file.
 */

'use strict';

(function() {

    injectStyles(`
        .tabs {
            width: 100%;
            display: flex;
            flex-direction: column;
            height: 100%;
        }

        .tab-buttons {
            display: flex;
            border-bottom: 1px solid #0f3333;
        }

        .tab-button {
            flex: 1;
            padding: 0.75em;
            background: #279888;
            border: none;
            cursor: pointer;
            font: inherit;
            font-size: small;
            transition: background 0.2s;
        }

        .tab-button:not(.active):hover {
            background: #0f3333;
            color: #dffbd1;
        }

        .tab-button.active {
            background: #dffbd1;
            border-bottom: 2px solid #0f3333;
        }

        .tab-contents {
            flex: 1;
            padding: 0.25em;
            background: #000;
        }

        .tab-content {
            display: none;
            height: 100%;
        }

        .tab-content.active {
            display: block;
        }

        .chat-window {
            overflow: scroll;
            background-color: #000;
            color: #fff;
        }

        .chat-window.broadcast { color: #d700d7; }
        .chat-window.whisper   { color: #737670; }

        .text-name.mob    { color: #00ffff; }
        .text-name.player { color: #fce94f; }
    `);

    // -----------------------------------------------------------------------
    // Channel configuration
    // Add or remove entries here to change which tabs appear.
    // -----------------------------------------------------------------------
    const CHANNELS = [
        { id: 'say',       label: 'Say',        cssClass: 'say',       active: true  },
        { id: 'whisper',   label: 'Whisper',     cssClass: 'whisper',   active: false },
        { id: 'party',     label: 'Party',       cssClass: 'party',     active: false },
        { id: 'broadcast', label: 'Broadcasts',  cssClass: 'broadcast', active: false },
    ];

    // -----------------------------------------------------------------------
    // DOM factory
    // Builds the full tabbed comm UI and appends it to document.body.
    // Returns the root element for WinBox to mount.
    // -----------------------------------------------------------------------
    function createDOM() {
        const root = document.createElement('div');
        root.id        = 'comm-output';
        root.className = 'tabs';

        // Tab button row
        const buttonRow = document.createElement('div');
        buttonRow.className = 'tab-buttons';

        // Tab panel container
        const panelContainer = document.createElement('div');
        panelContainer.className = 'tab-contents';

        CHANNELS.forEach(ch => {
            // Button
            const btn = document.createElement('button');
            btn.id               = 'comm-tab-' + ch.id;
            btn.className        = 'tab-button' + (ch.active ? ' active' : '');
            btn.dataset.tab      = 'comm-' + ch.id;
            btn.dataset.label    = ch.label;
            btn.dataset.unread   = '0';
            btn.textContent      = ch.label;
            buttonRow.appendChild(btn);

            // Panel
            const panel = document.createElement('div');
            panel.id        = 'comm-' + ch.id;
            panel.className = 'chat-window ' + ch.cssClass + ' tab-content' + (ch.active ? ' active' : '');
            panelContainer.appendChild(panel);
        });

        root.appendChild(buttonRow);
        root.appendChild(panelContainer);
        document.body.appendChild(root);

        // Wire up tab switching within this window's root element
        const buttons = buttonRow.querySelectorAll('.tab-button');
        const panels  = panelContainer.querySelectorAll('.tab-content');

        buttons.forEach(btn => {
            btn.addEventListener('click', () => {
                const target = btn.dataset.tab;

                buttons.forEach(b => b.classList.remove('active'));
                panels.forEach(p => p.classList.remove('active'));

                btn.classList.add('active');
                btn.dataset.unread = '0';
                btn.textContent    = btn.dataset.label;
                document.getElementById(target).classList.add('active');
            });
        });

        return root;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow instance
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Comm', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  500,
        factory() {
            const el = createDOM();
            return {
                title:      'Communications',
                mount:      el,
                background: '#1c6b60',
                border:     1,
                x:          'right',
                y:          450,
                width:      363,
                height:     20 + 290,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Message rendering
    // -----------------------------------------------------------------------
    function postMessage(channelName, fromName, fromSource, message) {
        const tab   = document.getElementById('comm-tab-' + channelName);
        const panel = document.getElementById('comm-' + channelName);

        if (!tab || !panel) {
            return;
        }

        // Update unread badge on inactive tabs
        if (tab.classList.contains('active')) {
            tab.dataset.unread = '0';
            tab.textContent    = tab.dataset.label;
        } else {
            tab.dataset.unread = String(parseInt(tab.dataset.unread) + 1);
            tab.textContent    = tab.dataset.label + '(' + tab.dataset.unread + ')';
        }

        const p = document.createElement('p');
        p.innerHTML =
            '<span class="text-name ' + fromSource + '">' + fromName + '</span>: ' +
            '<span class="text-body ' + fromSource + '">' + message + '</span>';
        panel.appendChild(p);

        // Trim overflow: remove oldest messages when content exceeds window height
        const winBox = win.get();
        if (winBox) {
            const winContainer = winBox.window;
            while (panel.scrollHeight > winContainer.clientHeight - 58) {
                if (panel.childElementCount < 1) {
                    break;
                }
                panel.removeChild(panel.firstElementChild);
            }
        }
    }

    // -----------------------------------------------------------------------
    // Update logic
    // -----------------------------------------------------------------------
    function updateComm() {
        const obj = Client.GMCPStructs.Comm;
        if (!obj || !obj.Channel) {
            return;
        }

        win.open();
        if (!win.isOpen()) {
            return;
        }

        const ch = obj.Channel;
        postMessage(ch.channel, ch.sender, ch.source, ch.text);
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Comm'],
        onGMCP(namespace, body) {
            updateComm();
        },
    });

})();
