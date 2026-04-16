/**
 * window-character.js
 *
 * Virtual window: Character — left dock, tabbed.
 *
 * Tabs:
 *   Overview   — name, race/class, level, alignment, stats grid,
 *                HP/MP bars, equipment slots (with hover tooltips)
 *   Backpack   — carried items with carry capacity, hover tooltips
 *   Quests     — in-progress quest log, click to expand
 *
 * Responds to GMCP namespaces:
 *   Char                    — full character update
 *   Char.Info               — name, race, class, level, alignment
 *   Char.Vitals             — HP / MP
 *   Char.Stats              — six core stats
 *   Char.Inventory          — worn equipment + backpack
 *   Char.Inventory.Backpack — backpack items only
 *   Char.Quests             — quest progress
 *
 * Reads:
 *   Client.GMCPStructs.Char.Info
 *   Client.GMCPStructs.Char.Vitals
 *   Client.GMCPStructs.Char.Stats
 *   Client.GMCPStructs.Char.Inventory.Worn
 *   Client.GMCPStructs.Char.Inventory.Backpack
 *   Client.GMCPStructs.Char.Quests
 */

'use strict';

(function() {

    injectStyles(`
        /* ---- shared tab chrome ---- */
        #character-window {
            height: 100%;
            display: flex;
            flex-direction: column;
            background: #000;
        }

        #character-window .cw-tab-bar {
            display: flex;
            flex-shrink: 0;
            border-bottom: 1px solid #0f3333;
        }

        #character-window .cw-tab-btn {
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

        #character-window .cw-tab-btn:last-child {
            border-right: none;
        }

        #character-window .cw-tab-btn:hover {
            background: #0f3333;
            color: #dffbd1;
        }

        #character-window .cw-tab-btn.active {
            background: #000;
            color: #dffbd1;
            border-bottom: 2px solid #3ad4b8;
        }

        #character-window .cw-tab-panel {
            display: none;
            flex: 1;
            overflow-y: auto;
        }

        #character-window .cw-tab-panel::-webkit-scrollbar       { width: 4px; }
        #character-window .cw-tab-panel::-webkit-scrollbar-track  { background: #111; }
        #character-window .cw-tab-panel::-webkit-scrollbar-thumb  { background: #1c6b60; border-radius: 2px; }

        #character-window .cw-tab-panel.active {
            display: flex;
            flex-direction: column;
        }

        /* ---- Overview tab ---- */
        #cw-overview {
            padding: 8px 10px;
            gap: 5px;
        }

        #cw-char-name {
            font-size: 0.88em;
            color: #dffbd1;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        #cw-char-level {
            font-size: 0.74em;
            color: #aaa;
        }

        #cw-char-alignment {
            font-size: 0.7em;
            font-style: italic;
            margin-bottom: 2px;
        }

        .cw-align-good    { color: #7ecfff; }
        .cw-align-neutral { color: #666;    }
        .cw-align-evil    { color: #e06060; }


        /* ---- Equipment section (inside Overview) ---- */
        #cw-equip-section {
            display: flex;
            flex-direction: column;
            gap: 2px;
            margin-top: 4px;
            padding-top: 6px;
            border-top: 1px solid #0f3333;
        }

        .cw-equip-row {
            display: flex;
            align-items: center;
            gap: 6px;
            min-height: 18px;
            border-bottom: 1px solid #0a1a16;
            padding-bottom: 2px;
            cursor: default;
        }

        .cw-equip-row:last-child { border-bottom: none; }

        .cw-equip-row:hover { background: #0a1e1a; }

        .cw-equip-slot {
            width: 54px;
            font-size: 0.66em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.03em;
            flex-shrink: 0;
        }

        .cw-equip-name {
            flex: 1;
            font-size: 0.76em;
            color: #dffbd1;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .cw-equip-name.empty  { color: #2a2a2a; font-style: italic; }
        .cw-equip-name.cursed { color: #e06060; }
        .cw-equip-name.quest  { color: #d4a843; }

        .cw-equip-badge {
            font-size: 0.58em;
            padding: 1px 3px;
            border-radius: 3px;
            flex-shrink: 0;
        }

        .cw-equip-badge.cursed { background:#3d0f0f; color:#e06060; border:1px solid #6b1c1c; }
        .cw-equip-badge.quest  { background:#2e2000; color:#d4a843; border:1px solid #6b5010; }

        /* ---- Stats grid (inside Overview, above vitals) ---- */
        #cw-stats-grid {
            display: grid;
            grid-template-columns: 1fr 1fr 1fr;
            gap: 3px 6px;
            padding: 4px 0 2px;
            border-top: 1px solid #0f3333;
            border-bottom: 1px solid #0f3333;
        }

        .cw-stat-cell {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 3px;
        }

        .cw-stat-abbr {
            font-size: 0.64em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.04em;
            flex-shrink: 0;
        }

        .cw-stat-num {
            font-size: 0.78em;
            color: #dffbd1;
            font-weight: bold;
        }

        /* ---- Equipment tooltip ---- */
        #cw-equip-tooltip {
            position: fixed;
            z-index: 99999;
            pointer-events: none;
            background: #0d2e28;
            border: 1px solid #1c6b60;
            border-radius: 6px;
            box-shadow: 0 4px 16px rgba(0,0,0,0.7);
            padding: 8px 10px;
            min-width: 160px;
            max-width: 260px;
            display: none;
        }

        .cw-tt-name {
            font-size: 0.85em;
            font-weight: bold;
            color: #dffbd1;
            margin-bottom: 4px;
            line-height: 1.3;
        }

        .cw-tt-name .cw-tt-details {
            font-weight: normal;
            font-style: italic;
            color: #7ab8a0;
        }

        .cw-tt-name .cw-tt-details.cursed { color: #e06060; }
        .cw-tt-name .cw-tt-details.quest  { color: #d4a843; }

        .cw-tt-divider {
            border: none;
            border-top: 1px solid #1c6b60;
            margin: 5px 0;
        }

        .cw-tt-row {
            display: flex;
            justify-content: space-between;
            align-items: baseline;
            gap: 8px;
            font-size: 0.75em;
            line-height: 1.6;
        }

        .cw-tt-row-label {
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.04em;
            font-size: 0.88em;
            flex-shrink: 0;
        }

        .cw-tt-row-value {
            color: #dffbd1;
            text-align: right;
        }

        /* ---- Quests tab ---- */
        #cw-quests {
            padding: 4px 6px;
            gap: 5px;
        }

        #cw-quests .cq-empty {
            color: #444;
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        #cw-quests .cq-item {
            background: #0a1e1a;
            border: 1px solid #1c6b60;
            border-radius: 4px;
            padding: 5px 7px;
            display: flex;
            flex-direction: column;
            gap: 4px;
            cursor: pointer;
            transition: background 0.15s;
            flex-shrink: 0;
        }

        #cw-quests .cq-item:hover,
        #cw-quests .cq-item.expanded {
            background: #0d2e28;
        }

        #cw-quests .cq-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            gap: 6px;
        }

        #cw-quests .cq-name {
            font-size: 0.82em;
            color: #dffbd1;
            font-weight: bold;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        #cw-quests .cq-pct {
            font-size: 0.7em;
            color: #7ab8a0;
            flex-shrink: 0;
        }

        #cw-quests .cq-bar-track {
            width: 100%;
            height: 5px;
            background: #1a1a1a;
            border-radius: 3px;
            overflow: hidden;
            border: 1px solid #1a2e28;
        }

        #cw-quests .cq-bar-fill {
            height: 100%;
            border-radius: 3px;
            background: linear-gradient(to right, #1c6b60, #3ad4b8);
            transition: width 0.4s ease-out;
        }

        #cw-quests .cq-bar-fill.complete {
            background: #3ad4b8;
        }

        #cw-quests .cq-desc {
            font-size: 0.73em;
            color: #7ab8a0;
            line-height: 1.4;
            display: none;
            padding-top: 2px;
            border-top: 1px solid #0f3333;
        }

        #cw-quests .cq-item.expanded .cq-desc {
            display: block;
        }

        /* ---- Backpack tab ---- */
        #cw-backpack {
            padding: 4px 6px;
            gap: 3px;
        }

        #cw-bp-header {
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 3px 2px 5px;
            border-bottom: 1px solid #0f3333;
            margin-bottom: 2px;
            flex-shrink: 0;
        }

        #cw-bp-title {
            font-size: 0.68em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.04em;
        }

        #cw-bp-count {
            font-size: 0.68em;
            color: #aaa;
        }

        #cw-bp-count .bp-count-num {
            color: #dffbd1;
        }

        #cw-bp-count .bp-count-num.full {
            color: #e06060;
        }

        #cw-bp-list {
            display: flex;
            flex-direction: column;
            gap: 2px;
            flex: 1;
        }

        .cw-bp-empty {
            color: #444;
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        .cw-bp-row {
            display: flex;
            align-items: center;
            gap: 6px;
            min-height: 18px;
            border-bottom: 1px solid #0a1a16;
            padding-bottom: 2px;
            cursor: default;
            flex-shrink: 0;
        }

        .cw-bp-row:last-child { border-bottom: none; }

        .cw-bp-row:hover { background: #0a1e1a; }

        .cw-bp-type {
            width: 54px;
            font-size: 0.66em;
            color: #7ab8a0;
            text-transform: uppercase;
            letter-spacing: 0.03em;
            flex-shrink: 0;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .cw-bp-name {
            flex: 1;
            font-size: 0.76em;
            color: #dffbd1;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .cw-bp-name.cursed { color: #e06060; }
        .cw-bp-name.quest  { color: #d4a843; }

        .cw-bp-badge {
            font-size: 0.58em;
            padding: 1px 3px;
            border-radius: 3px;
            flex-shrink: 0;
        }

        .cw-bp-badge.cursed { background:#3d0f0f; color:#e06060; border:1px solid #6b1c1c; }
        .cw-bp-badge.quest  { background:#2e2000; color:#d4a843; border:1px solid #6b5010; }
    `);

    // -----------------------------------------------------------------------
    // Tooltip
    // Created lazily on first use so document.body is guaranteed to exist.
    // -----------------------------------------------------------------------
    let tooltip    = null;
    let hideTimer  = null;
    const rowItemData = new Map();

    function ensureTooltip() {
        if (tooltip) { return; }
        tooltip = document.createElement('div');
        tooltip.id = 'cw-equip-tooltip';
        document.body.appendChild(tooltip);
    }

    function showTooltip(rowEl, item) {
        ensureTooltip();
        clearTimeout(hideTimer);

        // Build content
        const details = (item.details && item.details.length > 0)
            ? item.details.join(', ')
            : null;

        const detailClass = item.details && item.details.includes('cursed') ? 'cursed'
                          : item.details && item.details.includes('quest')  ? 'quest'
                          : '';

        let html = '<div class="cw-tt-name">' + item.name;
        if (details) {
            html += ' <span class="cw-tt-details ' + detailClass + '">(' + details + ')</span>';
        }
        html += '</div>';

        const rows = [];
        if (item.type)    { rows.push({ label: 'Type',    value: item.type    }); }
        if (item.subtype) { rows.push({ label: 'Subtype', value: item.subtype }); }
        if (item.uses > 0){ rows.push({ label: 'Uses',    value: item.uses    }); }

        if (rows.length > 0) {
            html += '<hr class="cw-tt-divider">';
            rows.forEach(r => {
                html += '<div class="cw-tt-row">' +
                    '<span class="cw-tt-row-label">' + r.label + '</span>' +
                    '<span class="cw-tt-row-value">' + r.value + '</span>' +
                '</div>';
            });
        }

        tooltip.innerHTML = html;
        tooltip.style.display = 'block';

        positionTooltip(rowEl);
    }

    function positionTooltip(rowEl) {
        if (!tooltip) { return; }
        const rect = rowEl.getBoundingClientRect();
        const ttW  = tooltip.offsetWidth;
        const ttH  = tooltip.offsetHeight;
        const vw   = window.innerWidth;
        const vh   = window.innerHeight;

        // Try to place to the right of the row; flip left if it would overflow
        let left = rect.right + 8;
        if (left + ttW > vw - 8) {
            left = rect.left - ttW - 8;
        }
        left = Math.max(8, left);

        // Align top with the row; shift up if it would overflow the bottom
        let top = rect.top;
        if (top + ttH > vh - 8) {
            top = vh - ttH - 8;
        }
        top = Math.max(8, top);

        tooltip.style.left = left + 'px';
        tooltip.style.top  = top  + 'px';
    }

    function hideTooltip() {
        if (!tooltip) { return; }
        hideTimer = setTimeout(() => {
            tooltip.style.display = 'none';
        }, 80);
    }

    function attachTooltip(rowEl) {
        rowEl.addEventListener('mouseenter', () => {
            const item = rowItemData.get(rowEl);
            if (item) { showTooltip(rowEl, item); }
        });
        rowEl.addEventListener('mouseleave', hideTooltip);
        rowEl.addEventListener('mousemove',  () => {
            if (tooltip.style.display === 'block') {
                positionTooltip(rowEl);
            }
        });
    }

    // -----------------------------------------------------------------------
    // Tab switching
    // -----------------------------------------------------------------------
    function makeTabSwitcher(root) {
        const btns   = root.querySelectorAll('.cw-tab-btn');
        const panels = root.querySelectorAll('.cw-tab-panel');
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
    // Data definitions
    // -----------------------------------------------------------------------
    const STAT_DEFS = [
        { key: 'strength',   abbr: 'STR' },
        { key: 'speed',      abbr: 'SPD' },
        { key: 'smarts',     abbr: 'SMT' },
        { key: 'vitality',   abbr: 'VIT' },
        { key: 'mysticism',  abbr: 'MYS' },
        { key: 'perception', abbr: 'PER' },
    ];

    const EQUIP_SLOTS = [
        { key: 'head',    label: 'Head'    },
        { key: 'neck',    label: 'Neck'    },
        { key: 'body',    label: 'Body'    },
        { key: 'weapon',  label: 'Weapon'  },
        { key: 'offhand', label: 'Offhand' },
        { key: 'gloves',  label: 'Gloves'  },
        { key: 'belt',    label: 'Belt'    },
        { key: 'ring',    label: 'Ring'    },
        { key: 'legs',    label: 'Legs'    },
        { key: 'feet',    label: 'Feet'    },
    ];

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function buildStatsGrid() {
        const cells = STAT_DEFS.map(d =>
            '<div class="cw-stat-cell">' +
                '<span class="cw-stat-abbr">' + d.abbr + '</span>' +
                '<span class="cw-stat-num" id="cw-stat-' + d.key + '">—</span>' +
            '</div>'
        ).join('');
        return '<div id="cw-stats-grid">' + cells + '</div>';
    }

    function buildEquipSection() {
        const rows = EQUIP_SLOTS.map(s =>
            '<div class="cw-equip-row" id="cw-eqrow-' + s.key + '">' +
                '<span class="cw-equip-slot">' + s.label + '</span>' +
                '<span class="cw-equip-name empty" id="cw-eq-' + s.key + '">empty</span>' +
                '<span class="cw-equip-badge" id="cw-eqb-' + s.key + '" style="display:none"></span>' +
            '</div>'
        ).join('');
        return '<div id="cw-equip-section">' + rows + '</div>';
    }

    function createDOM() {
        const el = document.createElement('div');
        el.id = 'character-window';
        el.innerHTML =
            '<div class="cw-tab-bar">' +
                '<button class="cw-tab-btn active" data-panel="cw-overview">Overview</button>' +
                '<button class="cw-tab-btn"        data-panel="cw-backpack">Backpack</button>' +
                '<button class="cw-tab-btn"        data-panel="cw-quests">Quests</button>' +
            '</div>' +

            '<div class="cw-tab-panel active" id="cw-overview">' +
                '<div id="cw-char-name">—</div>' +
                '<div id="cw-char-level">Level —</div>' +
                '<div id="cw-char-alignment"></div>' +
                buildStatsGrid() +
                buildEquipSection() +
            '</div>' +

            '<div class="cw-tab-panel" id="cw-backpack">' +
                '<div id="cw-bp-header">' +
                    '<span id="cw-bp-title">Carried Items</span>' +
                    '<span id="cw-bp-count"><span class="bp-count-num" id="cw-bp-num">0</span> / <span id="cw-bp-max">—</span></span>' +
                '</div>' +
                '<div id="cw-bp-list"><div class="cw-bp-empty">Empty</div></div>' +
            '</div>' +

            '<div class="cw-tab-panel" id="cw-quests">' +
                '<div class="cq-empty">No active quests</div>' +
            '</div>';

        document.body.appendChild(el);
        makeTabSwitcher(el);

        // Attach tooltip listeners to all equipment rows
        EQUIP_SLOTS.forEach(s => {
            const rowEl = el.querySelector('#cw-eqrow-' + s.key);
            if (rowEl) { attachTooltip(rowEl); }
        });

        return el;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Character', {
        dock:          'left',
        defaultDocked: true,
        dockedHeight:  390,
        factory() {
            const el = createDOM();
            return {
                title:      'Character',
                mount:      el,
                background: '#1c6b60',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     580,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Update functions
    // -----------------------------------------------------------------------
    function updateOverview() {
        const info = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Info;

        if (info) {
            document.getElementById('cw-char-name').textContent = [info.name, info.race, info.class].filter(Boolean).join(' · ') || '—';
            document.getElementById('cw-char-level').textContent = info.level ? 'Level ' + info.level : 'Level —';

            const alignEl = document.getElementById('cw-char-alignment');
            alignEl.textContent = info.alignment || '';
            const a = (info.alignment || '').toLowerCase();
            alignEl.className = 'cw-char-alignment ' +
                (a.includes('good') ? 'cw-align-good' : a.includes('evil') ? 'cw-align-evil' : 'cw-align-neutral');
        }
    }

    function updateStats() {
        const stats = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Stats;
        if (!stats) { return; }

        STAT_DEFS.forEach(def => {
            const el = document.getElementById('cw-stat-' + def.key);
            if (el) { el.textContent = stats[def.key] || '—'; }
        });
    }

    function updateEquipment() {
        const inv = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Inventory;
        if (!inv || !inv.Worn) { return; }

        const worn = inv.Worn;
        EQUIP_SLOTS.forEach(slot => {
            const item    = worn[slot.key];
            const rowEl   = document.getElementById('cw-eqrow-' + slot.key);
            const nameEl  = document.getElementById('cw-eq-'    + slot.key);
            const badgeEl = document.getElementById('cw-eqb-'   + slot.key);

            if (!item || !item.name) {
                nameEl.textContent = 'empty';
                nameEl.className   = 'cw-equip-name empty';
                badgeEl.style.display = 'none';
                rowItemData.delete(rowEl);
                return;
            }

            // Store full item data on the row for the tooltip
            rowItemData.set(rowEl, item);

            const isCursed = item.details && item.details.includes('cursed');
            const isQuest  = item.details && item.details.includes('quest');

            nameEl.textContent = item.name;
            nameEl.className   = 'cw-equip-name' + (isCursed ? ' cursed' : isQuest ? ' quest' : '');

            if (isCursed) {
                badgeEl.textContent = 'cursed'; badgeEl.className = 'cw-equip-badge cursed'; badgeEl.style.display = '';
            } else if (isQuest) {
                badgeEl.textContent = 'quest';  badgeEl.className = 'cw-equip-badge quest';  badgeEl.style.display = '';
            } else {
                badgeEl.style.display = 'none';
            }
        });
    }

    function updateBackpack() {
        const inv = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Inventory;
        if (!inv || !inv.Backpack) { return; }

        const bp      = inv.Backpack;
        const items   = bp.items   || [];
        const summary = bp.Summary || {};
        const count   = summary.count !== undefined ? summary.count : items.length;
        const max     = summary.max   || 0;

        // Update carry capacity header
        const numEl = document.getElementById('cw-bp-num');
        const maxEl = document.getElementById('cw-bp-max');
        if (numEl) {
            numEl.textContent = count;
            numEl.classList.toggle('full', max > 0 && count >= max);
        }
        if (maxEl) { maxEl.textContent = max || '—'; }

        const list = document.getElementById('cw-bp-list');
        if (!list) { return; }

        // Remove old tooltip registrations for rows about to be replaced
        list.querySelectorAll('.cw-bp-row').forEach(r => rowItemData.delete(r));
        list.innerHTML = '';

        if (items.length === 0) {
            list.innerHTML = '<div class="cw-bp-empty">Empty</div>';
            return;
        }

        // Sort: quest items first, then cursed, then alphabetical by name
        const sorted = [...items].sort((a, b) => {
            const aq = a.details && a.details.includes('quest');
            const bq = b.details && b.details.includes('quest');
            if (aq !== bq) { return aq ? -1 : 1; }
            const ac = a.details && a.details.includes('cursed');
            const bc = b.details && b.details.includes('cursed');
            if (ac !== bc) { return ac ? -1 : 1; }
            return (a.name || '').localeCompare(b.name || '');
        });

        sorted.forEach(item => {
            const isCursed = item.details && item.details.includes('cursed');
            const isQuest  = item.details && item.details.includes('quest');

            const row = document.createElement('div');
            row.className = 'cw-bp-row';

            const typeEl  = document.createElement('span');
            typeEl.className   = 'cw-bp-type';
            typeEl.textContent = item.type || '';

            const nameEl  = document.createElement('span');
            nameEl.className   = 'cw-bp-name' + (isCursed ? ' cursed' : isQuest ? ' quest' : '');
            nameEl.textContent = item.name || '';

            const badgeEl = document.createElement('span');
            badgeEl.className = 'cw-bp-badge';
            if (isCursed) {
                badgeEl.textContent = 'cursed';
                badgeEl.classList.add('cursed');
            } else if (isQuest) {
                badgeEl.textContent = 'quest';
                badgeEl.classList.add('quest');
            } else {
                badgeEl.style.display = 'none';
            }

            row.appendChild(typeEl);
            row.appendChild(nameEl);
            row.appendChild(badgeEl);
            list.appendChild(row);

            // Register item data and attach tooltip — same mechanism as equipment
            rowItemData.set(row, item);
            attachTooltip(row);
        });
    }

    function updateQuests() {
        const quests = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Quests;
        if (!quests) { return; }

        const panel = document.getElementById('cw-quests');
        if (!panel) { return; }

        // Preserve expanded state by quest name
        const expanded = new Set();
        panel.querySelectorAll('.cq-item.expanded').forEach(el => {
            expanded.add(el.dataset.questName);
        });

        panel.innerHTML = '';

        if (!Array.isArray(quests) || quests.length === 0) {
            panel.innerHTML = '<div class="cq-empty">No active quests</div>';
            return;
        }

        // Sort: least complete first, then alphabetical
        const sorted = [...quests].sort((a, b) => {
            if (a.completion !== b.completion) { return a.completion - b.completion; }
            return (a.name || '').localeCompare(b.name || '');
        });

        sorted.forEach(q => {
            const pct        = Math.max(0, Math.min(100, q.completion || 0));
            const complete   = pct >= 100;
            const isExpanded = expanded.has(q.name);

            const item = document.createElement('div');
            item.className       = 'cq-item' + (isExpanded ? ' expanded' : '');
            item.dataset.questName = q.name || '';
            item.innerHTML =
                '<div class="cq-header">' +
                    '<span class="cq-name">' + (q.name || 'Unknown Quest') + '</span>' +
                    '<span class="cq-pct">' + pct + '%</span>' +
                '</div>' +
                '<div class="cq-bar-track">' +
                    '<div class="cq-bar-fill' + (complete ? ' complete' : '') + '" style="width:' + pct + '%"></div>' +
                '</div>' +
                '<div class="cq-desc">' + (q.description || '') + '</div>';

            item.addEventListener('click', () => item.classList.toggle('expanded'));
            panel.appendChild(item);
        });
    }

    function update() {
        win.open();
        if (!win.isOpen()) { return; }
        updateOverview();
        updateStats();
        updateEquipment();
        updateBackpack();
        updateQuests();
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char.Info', 'Char.Stats', 'Char.Inventory', 'Char.Inventory.Backpack', 'Char.Quests', 'Char'],
        onGMCP() { update(); },
    });

})();
