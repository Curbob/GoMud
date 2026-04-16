/**
 * window-vitals.js
 *
 * Virtual window: Vitals (HP / MP bars).
 *
 * Responds to GMCP namespaces:
 *   Char.Vitals  - direct vitals update
 *   Char         - top-level Char update; delegates to Char.Vitals if present
 *
 * Reads: Client.GMCPStructs.Char.Vitals
 */

'use strict';

(function() {

    injectStyles(`
        #vitals-bars {
            height: 100%;
        }

        #health-bar {
            position: relative;
            width: 100%;
            height: 50%;
            background: linear-gradient(to right, #f44336 0%, #ffeb3b 50%, #4caf50 100%);
            border-radius: 12px;
            box-shadow: inset 0 2px 4px rgba(0,0,0,0.6);
            overflow: hidden;
            margin: 0 2px;
        }

        #health-bar .health-fill {
            height: 100%;
            width: 0%;
            background: #333;
            float: right;
            transition: width 0.4s ease-out;
        }

        #health-bar .health-text {
            position: absolute;
            top: 0; left: 0; right: 0; bottom: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            font-family: monospace;
            font-size: 0.85em;
            color: white;
            text-shadow: 0 1px 2px rgba(0,0,0,0.8);
            pointer-events: none;
        }

        #mana-bar {
            position: relative;
            width: 100%;
            height: 50%;
            background: linear-gradient(to right, #1e108b 0%, #3a20fe 100%);
            border-radius: 12px;
            box-shadow: inset 0 2px 4px rgba(0,0,0,0.6);
            overflow: hidden;
            margin: 0;
        }

        #mana-bar .mana-fill {
            height: 100%;
            width: 0%;
            background: #333;
            float: right;
            transition: width 0.4s ease-out;
        }

        #mana-bar .mana-text {
            position: absolute;
            top: 0; left: 0; right: 0; bottom: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            font-family: monospace;
            font-size: 0.85em;
            color: white;
            text-shadow: 0 1px 2px rgba(0,0,0,0.8);
            pointer-events: none;
        }
    `);

    // -----------------------------------------------------------------------
    // DOM factory
    // Creates the vitals bar elements and appends them to document.body
    // (hidden until WinBox mounts them).
    // -----------------------------------------------------------------------
    function createDOM() {
        const container = document.createElement('div');
        container.id = 'vitals-bars';

        const healthBar = document.createElement('div');
        healthBar.id = 'health-bar';
        healthBar.innerHTML =
            '<div class="health-fill" style="width:100%;"></div>' +
            '<span class="health-text">100%</span>';

        const manaBar = document.createElement('div');
        manaBar.id = 'mana-bar';
        manaBar.innerHTML =
            '<div class="mana-fill" style="width:100%;"></div>' +
            '<span class="mana-text">100%</span>';

        container.appendChild(healthBar);
        container.appendChild(manaBar);

        // Must be in the DOM before WinBox mounts it
        document.body.appendChild(container);
        return container;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow instance
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Char.Vitals', {
        dock:          'left',
        defaultDocked: true,
        dockedHeight:  60,
        factory() {
            const el = createDOM();
            return {
                title:      'Vitals',
                mount:      el,
                background: '#1c6b60',
                border:     1,
                x:          0,
                y:          0,
                width:      300,
                height:     20 + 40,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Update logic
    // -----------------------------------------------------------------------
    function updateBars() {
        const vitals = Client.GMCPStructs.Char && Client.GMCPStructs.Char.Vitals;
        if (!vitals) {
            return;
        }

        win.open();
        if (!win.isOpen()) {
            return;
        }

        const hp  = Math.max(0, Math.min(100, Math.floor(vitals.hp  / vitals.hp_max  * 100)));
        const sp  = Math.max(0, Math.min(100, Math.floor(vitals.sp  / vitals.sp_max  * 100)));

        const healthFill = document.querySelector('#health-bar .health-fill');
        const healthText = document.querySelector('#health-bar .health-text');
        healthFill.style.width  = (100 - hp) + '%';
        healthText.textContent  = vitals.hp + '/' + vitals.hp_max + ' hp';

        const manaFill = document.querySelector('#mana-bar .mana-fill');
        const manaText = document.querySelector('#mana-bar .mana-text');
        manaFill.style.width  = (100 - sp) + '%';
        manaText.textContent  = vitals.sp + '/' + vitals.sp_max + ' mp';
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Char.Vitals', 'Char'],
        onGMCP(namespace) {
            // Both Char and Char.Vitals route here.
            // The GMCP store is already updated before onGMCP is called,
            // so we just check for Vitals data and render.
            updateBars();
        },
    });

})();
