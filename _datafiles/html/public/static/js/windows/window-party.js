/**
 * window-party.js
 *
 * Virtual window: Party.
 * Shows all party members with their level, role/rank, current location,
 * and a health bar. Auto-hides the window content when not in a party.
 * Docks to the left column so it doesn't compete with the character panels.
 *
 * Responds to GMCP namespaces:
 *   Party         - full party update (roster + vitals)
 *   Party.Vitals  - lightweight vitals-only update
 *
 * Reads:
 *   Client.GMCPStructs.Party        (Leader, Members, Invited, Vitals)
 *   Client.GMCPStructs['Party.Vitals']  (Vitals map only)
 */

'use strict';

(function() {

    injectStyles(`
        #party-panel {
            height: 100%;
            overflow-y: auto;
            padding: 4px 6px;
            background: #000;
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        #party-panel::-webkit-scrollbar       { width: 4px; }
        #party-panel::-webkit-scrollbar-track  { background: #111; }
        #party-panel::-webkit-scrollbar-thumb  { background: #1c6b60; border-radius: 2px; }

        .party-empty {
            color: #444;
            font-size: 0.78em;
            font-style: italic;
            text-align: center;
            padding: 12px 0;
        }

        .party-member {
            background: #0a1e1a;
            border: 1px solid #1c6b60;
            border-radius: 4px;
            padding: 5px 7px;
            display: flex;
            flex-direction: column;
            gap: 4px;
        }

        .party-member.is-leader {
            border-color: #3ad4b8;
        }

        .party-member.is-invited {
            border-color: #555;
            background: #0a0a0a;
            opacity: 0.7;
        }

        .party-member-header {
            display: flex;
            align-items: center;
            gap: 6px;
        }

        .party-member-name {
            flex: 1;
            font-size: 0.82em;
            color: #dffbd1;
            font-weight: bold;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .party-member.is-leader .party-member-name {
            color: #3ad4b8;
        }

        .party-member.is-invited .party-member-name {
            color: #888;
        }

        .party-member-level {
            font-size: 0.7em;
            color: #7ab8a0;
            flex-shrink: 0;
        }

        .party-member-rank {
            font-size: 0.65em;
            color: #555;
            flex-shrink: 0;
            text-transform: capitalize;
        }

        .party-member-location {
            font-size: 0.68em;
            color: #4a8070;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
        }

        .party-hp-track {
            width: 100%;
            height: 5px;
            background: #1a1a1a;
            border-radius: 3px;
            overflow: hidden;
            border: 1px solid #1a2e28;
        }

        .party-hp-fill {
            height: 100%;
            border-radius: 3px;
            transition: width 0.3s ease-out;
        }

        /* Colour shifts from green → yellow → red as HP drops */
        .party-hp-fill[data-pct="high"]   { background: #3ad4b8; }
        .party-hp-fill[data-pct="medium"] { background: #d4a843; }
        .party-hp-fill[data-pct="low"]    { background: #d43a3a; }

        .party-invited-label {
            font-size: 0.65em;
            color: #555;
            font-style: italic;
        }
    `);

    function hpClass(pct) {
        if (pct >= 60) { return 'high'; }
        if (pct >= 30) { return 'medium'; }
        return 'low';
    }

    function createDOM() {
        const el = document.createElement('div');
        el.id = 'party-panel';
        el.innerHTML = '<div class="party-empty">Not in a party</div>';
        document.body.appendChild(el);
        return el;
    }

    const win = new VirtualWindow('Party', {
        dock:          'left',
        defaultDocked: true,
        dockedHeight:  200,
        factory() {
            const el = createDOM();
            return {
                title:      'Party',
                mount:      el,
                background: '#1c6b60',
                border:     1,
                x:          0,
                y:          0,
                width:      280,
                height:     240,
                header:     20,
                bottom:     60,
            };
        },
    });

    function update() {
        // Accept either a full Party payload or a Party.Vitals-only payload.
        // GMCPStructs stores Party.Vitals under the key 'Party' → 'Vitals'
        // because _applyGMCPPayload splits on '.' and nests accordingly.
        const partyData  = Client.GMCPStructs.Party;
        const vitalsOnly = partyData && partyData.Vitals && !partyData.Members;

        // Determine the vitals map regardless of which payload arrived
        let vitals  = {};
        let members = [];
        let invited = [];
        let leader  = '';

        if (partyData) {
            vitals  = partyData.Vitals  || {};
            members = partyData.Members || [];
            invited = partyData.Invited || [];
            leader  = partyData.Leader  || '';
        }

        win.open();
        if (!win.isOpen()) { return; }

        const panel = document.getElementById('party-panel');

        // If no members and no vitals entries, show the empty state
        const hasMembers = members.length > 0 || Object.keys(vitals).length > 0;
        if (!hasMembers) {
            panel.innerHTML = '<div class="party-empty">Not in a party</div>';
            return;
        }

        // Build a merged member list.
        // When we only have vitals data, synthesise member entries from it.
        let allMembers = members.length > 0 ? members : Object.keys(vitals).map(name => ({ Name: name, Status: 'In Party', Position: '' }));

        panel.innerHTML = '';

        allMembers.forEach(m => {
            const name     = m.Name || m.name || '';
            const rank     = m.Position || m.position || '';
            const isLeader = name === leader;
            const v        = vitals[name] || {};
            const hpPct    = Math.max(0, Math.min(100, v.health || 0));
            const level    = v.level || 0;
            const location = v.location || '';

            const div = document.createElement('div');
            div.className = 'party-member' + (isLeader ? ' is-leader' : '');

            div.innerHTML =
                '<div class="party-member-header">' +
                    '<span class="party-member-name">' + name + (isLeader ? ' ★' : '') + '</span>' +
                    (level ? '<span class="party-member-level">Lv ' + level + '</span>' : '') +
                    (rank   ? '<span class="party-member-rank">' + rank + '</span>' : '') +
                '</div>' +
                (location ? '<div class="party-member-location">' + location + '</div>' : '') +
                '<div class="party-hp-track">' +
                    '<div class="party-hp-fill" data-pct="' + hpClass(hpPct) + '" style="width:' + hpPct + '%"></div>' +
                '</div>';

            panel.appendChild(div);
        });

        // Invited members (no vitals available)
        invited.forEach(m => {
            const name = m.Name || m.name || '';
            const div  = document.createElement('div');
            div.className = 'party-member is-invited';
            div.innerHTML =
                '<div class="party-member-header">' +
                    '<span class="party-member-name">' + name + '</span>' +
                    '<span class="party-invited-label">invited</span>' +
                '</div>';
            panel.appendChild(div);
        });
    }

    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Party', 'Party.Vitals'],
        onGMCP() { update(); },
    });

})();
