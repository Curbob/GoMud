/**
 * window-status.js
 *
 * Virtual window: Status — right dock, tabbed.
 *
 * Tabs:
 *   Worth    — XP progress bar, gold (carried + bank), skill/training points
 *   Effects  — active buffs/debuffs with duration bars
 *
 * Responds to GMCP namespaces:
 *   Char.Worth    — XP, gold, points
 *   Char.Affects  — active buffs/debuffs
 *   Char          — full character update
 *
 * Reads:
 *   Client.GMCPStructs.Char.Worth
 *   Client.GMCPStructs.Char.Affects
 */

'use strict';

(function() {

    injectStyles(`
        /* ---- shared tab chrome ---- */
        #status-window {
            height: 100%;
            display: flex;
            flex-direction: column;
            background: #000;
        }

        #status-window .sw-tab-bar {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid #0f3333;
        }

        #status-window .sw-tab-btn {
            flex: 1;
            padding: 5px 4px;
            background: #0d2e28;
            border: none;
            cursor: pointer;
            font: inherit;
            font-size: 0.7em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.04em;
            transition: background 0.15s, color 0.15s;
            border-right: 1px solid #0f3333;
        }

        #status-window .sw-tab-btn:last-child { border-right: none; }

        #status-window .sw-tab-btn:hover {
            background: #0f3333;
            color: #dffbd1;
        }

        #status-window .sw-tab-btn.active {
            background: #000;
            color: #dffbd1;
            border-bottom: 2px solid #3ad4b8;
        }

        #status-window .sw-tab-panel {
            display: none;
            flex: 1;
            overflow-y: auto;
        }

        #status-window .sw-tab-panel::-webkit-scrollbar       { width: 4px; }
        #status-window .sw-tab-panel::-webkit-scrollbar-track  { background: #111; }
        #status-window .sw-tab-panel::-webkit-scrollbar-thumb  { background: #1c6b60; border-radius: 2px; }

        #status-window .sw-tab-panel.active {
            display: flex;
            flex-direction: column;
        }

        /* ---- Worth tab ---- */
        #sw-worth {
            padding: 8px 10px;
            gap: 8px;
            justify-content: flex-start;
        }

        .sw-xp-section {
            display: flex;
            flex-direction: column;
            gap: 3px;
        }

        .sw-xp-label-row {
            display: flex;
            justify-content: space-between;
            font-size: 0.7em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .sw-xp-track {
            width: 100%;
            height: 10px;
            background: #1a1a1a;
            border-radius: 5px;
            overflow: hidden;
            border: 1px solid #222;
        }

        .sw-xp-fill {
            height: 100%;
            border-radius: 5px;
            background: linear-gradient(to right, #1c6b60, #3ad4b8);
            transition: width 0.4s ease-out;
        }

        .sw-worth-grid {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 6px 10px;
        }

        .sw-worth-cell {
            display: flex;
            flex-direction: column;
            gap: 1px;
        }

        .sw-worth-cell-label {
            font-size: 0.66em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .sw-worth-cell-value {
            font-size: 0.85em;
            color: #dffbd1;
        }

        .sw-points-row {
            display: flex;
            gap: 8px;
        }

        .sw-point-badge {
            flex: 1;
            display: flex;
            flex-direction: column;
            align-items: center;
            padding: 4px 6px;
            background: #0d2e28;
            border: 1px solid #1c6b60;
            border-radius: 4px;
            gap: 1px;
        }

        .sw-point-badge-label {
            font-size: 0.63em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        .sw-point-badge-value {
            font-size: 1em;
            color: #dffbd1;
            font-weight: bold;
        }

        .sw-point-badge.has-points {
            border-color: #3ad4b8;
            background: #0d3d35;
        }

        .sw-point-badge.has-points .sw-point-badge-value {
            color: #3ad4b8;
        }

        /* ---- Effects tab ---- */
        #sw-effects {
            padding: 4px 6px;
            gap: 4px;
        }

        .sw-affect-empty {
            color: #444;
            font-size: 0.76em;
            font-style: italic;
            text-align: center;
            padding: 14px 0;
        }

        .sw-affect-item {
            background: #0a1e1a;
            border: 1px solid #1c6b60;
            border-radius: 4px;
            padding: 4px 6px;
            display: flex;
            flex-direction: column;
            gap: 3px;
            flex-shrink: 0;
        }

        .sw-affect-item.debuff {
            border-color: #6b1c1c;
            background: #1e0a0a;
        }

        .sw-affect-header {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 4px;
        }

        .sw-affect-name {
            font-size: 0.8em;
            color: #dffbd1;
            font-weight: bold;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .sw-affect-item.debuff .sw-affect-name { color: #f4a0a0; }

        .sw-affect-source {
            font-size: 0.63em;
            color: #7ab8a0;
            white-space: nowrap;
            flex-shrink: 0;
        }

        .sw-affect-item.debuff .sw-affect-source { color: #b87a7a; }

        .sw-affect-mods {
            font-size: 0.66em;
            color: #7ab8a0;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .sw-affect-item.debuff .sw-affect-mods { color: #b87a7a; }

        .sw-affect-dur-track {
            width: 100%;
            height: 4px;
            background: #1a1a1a;
            border-radius: 2px;
            overflow: hidden;
        }

        .sw-affect-dur-fill {
            height: 100%;
            border-radius: 2px;
            background: #1c6b60;
            transition: width 1s linear;
        }

        .sw-affect-item.debuff .sw-affect-dur-fill { background: #6b1c1c; }

        .sw-affect-dur-fill.permanent {
            background: #3ad4b8;
            width: 100% !important;
        }

        .sw-affect-item.debuff .sw-affect-dur-fill.permanent { background: #d43a3a; }
    `);

    // -----------------------------------------------------------------------
    // Tab switching
    // -----------------------------------------------------------------------
    function makeTabSwitcher(root) {
        const btns   = root.querySelectorAll('.sw-tab-btn');
        const panels = root.querySelectorAll('.sw-tab-panel');
        btns.forEach(btn => {
            btn.addEventListener('click', () => {
                btns.forEach(b   => b.classList.remove('active'));
                panels.forEach(p => p.classList.remove('active'));
                btn.classList.add('active');
                root.querySelector('#' + btn.dataset.panel).classList.add('active');
            });
        });
    }

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function createDOM() {
        const el = document.createElement('div');
        el.id = 'status-window';
        el.innerHTML =
            '<div class="sw-tab-bar">' +
                '<button class="sw-tab-btn active" data-panel="sw-worth">Worth</button>' +
                '<button class="sw-tab-btn"        data-panel="sw-effects">Effects</button>' +
            '</div>' +

            '<div class="sw-tab-panel active" id="sw-worth">' +
                '<div class="sw-xp-section">' +
                    '<div class="sw-xp-label-row"><span>Experience</span><span id="sw-xp-text">— / —</span></div>' +
                    '<div class="sw-xp-track"><div class="sw-xp-fill" id="sw-xp-fill" style="width:0%"></div></div>' +
                '</div>' +
                '<div class="sw-worth-grid">' +
                    '<div class="sw-worth-cell"><span class="sw-worth-cell-label">Gold (on hand)</span><span class="sw-worth-cell-value" id="sw-gold">—</span></div>' +
                    '<div class="sw-worth-cell"><span class="sw-worth-cell-label">Gold (bank)</span><span class="sw-worth-cell-value" id="sw-bank">—</span></div>' +
                '</div>' +
                '<div class="sw-points-row">' +
                    '<div class="sw-point-badge" id="sw-badge-sp"><span class="sw-point-badge-label">Skill Pts</span><span class="sw-point-badge-value" id="sw-sp">—</span></div>' +
                    '<div class="sw-point-badge" id="sw-badge-tp"><span class="sw-point-badge-label">Train Pts</span><span class="sw-point-badge-value" id="sw-tp">—</span></div>' +
                '</div>' +
            '</div>' +

            '<div class="sw-tab-panel" id="sw-effects">' +
                '<div class="sw-affect-empty" id="sw-effects-empty">No active effects</div>' +
            '</div>';

        document.body.appendChild(el);
        makeTabSwitcher(el);
        return el;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Status', {
        dock:          'left',
        defaultDocked: true,
        dockedHeight:  190,
        factory() {
            const el = createDOM();
            return {
                title:      'Status',
                mount:      el,
                background: '#1c6b60',
                border:     1,
                x:          0,
                y:          0,
                width:      363,
                height:     260,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Worth update
    // -----------------------------------------------------------------------
    function fmt(n) {
        if (n === undefined || n === null) { return '—'; }
        return Number(n).toLocaleString();
    }

    function updateWorth() {
        const worth = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Worth;
        if (!worth) { return; }

        const xp  = worth.xp  || 0;
        const tnl = worth.tnl || 0;
        const pct = tnl > 0 ? Math.min(100, Math.round((xp / tnl) * 100)) : 0;

        document.getElementById('sw-xp-fill').style.width = pct + '%';
        document.getElementById('sw-xp-text').textContent = fmt(xp) + ' / ' + fmt(tnl);
        document.getElementById('sw-gold').textContent    = fmt(worth.gold_carry);
        document.getElementById('sw-bank').textContent    = fmt(worth.gold_bank);

        const sp = worth.skillpoints    || 0;
        const tp = worth.trainingpoints || 0;
        document.getElementById('sw-sp').textContent = sp;
        document.getElementById('sw-tp').textContent = tp;
        document.getElementById('sw-badge-sp').classList.toggle('has-points', sp > 0);
        document.getElementById('sw-badge-tp').classList.toggle('has-points', tp > 0);
    }

    // -----------------------------------------------------------------------
    // Effects update
    // -----------------------------------------------------------------------
    function isDebuff(mods) {
        if (!mods) { return false; }
        return Object.values(mods).some(v => v < 0);
    }

    function formatMods(mods) {
        if (!mods || Object.keys(mods).length === 0) { return ''; }
        return Object.entries(mods)
            .map(([k, v]) => (v >= 0 ? '+' : '') + v + ' ' + k)
            .join('  ');
    }

    function updateEffects() {
        const affects = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Affects;
        if (!affects) { return; }

        const panel = document.getElementById('sw-effects');
        panel.innerHTML = '';

        const keys = Object.keys(affects);
        if (keys.length === 0) {
            panel.innerHTML = '<div class="sw-affect-empty">No active effects</div>';
            return;
        }

        keys.sort((a, b) => {
            const da = isDebuff(affects[a].affects);
            const db = isDebuff(affects[b].affects);
            if (da !== db) { return da ? 1 : -1; }
            const pa = affects[a].duration_max === -1;
            const pb = affects[b].duration_max === -1;
            if (pa !== pb) { return pa ? 1 : -1; }
            return a.localeCompare(b);
        });

        keys.forEach(key => {
            const aff     = affects[key];
            const debuff  = isDebuff(aff.affects);
            const perma   = aff.duration_max === -1;
            const modText = formatMods(aff.affects);

            let durPct = 100;
            if (!perma && aff.duration_max > 0) {
                durPct = Math.max(0, Math.min(100, Math.round((aff.duration_cur / aff.duration_max) * 100)));
            }

            const item = document.createElement('div');
            item.className = 'sw-affect-item' + (debuff ? ' debuff' : '');
            item.innerHTML =
                '<div class="sw-affect-header">' +
                    '<span class="sw-affect-name">' + (aff.name || key) + '</span>' +
                    '<span class="sw-affect-source">' + (aff.type || '') + '</span>' +
                '</div>' +
                (modText ? '<div class="sw-affect-mods">' + modText + '</div>' : '') +
                '<div class="sw-affect-dur-track">' +
                    '<div class="sw-affect-dur-fill' + (perma ? ' permanent' : '') + '" style="width:' + durPct + '%"></div>' +
                '</div>';

            panel.appendChild(item);
        });
    }

    // -----------------------------------------------------------------------
    // Combined update
    // -----------------------------------------------------------------------
    function update() {
        win.open();
        if (!win.isOpen()) { return; }
        updateWorth();
        updateEffects();
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char.Worth', 'Char.Affects', 'Char'],
        onGMCP() { update(); },
    });

})();
