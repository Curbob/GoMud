/**
 * window-map.js
 *
 * Virtual window: Map (room grid SVG).
 *
 * Responds to GMCP namespace:
 *   Room  - room info and exit data
 *
 * Reads: Client.GMCPStructs.Room.Info
 */

'use strict';

(function() {

    // -----------------------------------------------------------------------
    // Module state
    // -----------------------------------------------------------------------
    let gr = null;  // RoomGridSVG instance, created when window first opens

    const allRooms = {
        currentZoneKey: '',
        roomZones: {},
    };

    // -----------------------------------------------------------------------
    // DOM factory
    // -----------------------------------------------------------------------
    function createDOM() {
        const el = document.createElement('div');
        el.id = 'map-render';
        el.style.width  = '100%';
        el.style.height = '100%';
        document.body.appendChild(el);
        return el;
    }

    // -----------------------------------------------------------------------
    // VirtualWindow instance
    // -----------------------------------------------------------------------
    const win = new VirtualWindow('Map', {
        dock:          'right',
        defaultDocked: true,
        dockedHeight:  363,
        factory() {
            const el = createDOM();
            return {
                title:      'Map',
                mount:      el,
                background: '#1c6b60',
                border:     1,
                x:          'right',
                y:          66,
                width:      363,
                height:     20 + 363,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -----------------------------------------------------------------------
    // Update logic
    // -----------------------------------------------------------------------
    function ensureGrid() {
        if (!gr) {
            gr = new RoomGridSVG('#map-render', {
                cellSize:    80,
                cellMargin:  80,
                initialZoom: 0.5,
            });
        }
    }

    function updateMap() {
        const obj = Client.GMCPStructs.Room;
        if (!obj || !obj.Info) {
            return;
        }

        win.open();
        if (!win.isOpen()) {
            return;
        }

        ensureGrid();

        const info   = obj.Info;
        const winBox = win.get();
        if (winBox) {
            winBox.setTitle('Map (' + info.area + ')');
        }

        // Parse coordinate string: "zoneName, x, y, z"
        const coords = info.coords.split(',').map(s => s.trim());
        const zoneName = coords[0];
        const zoneKey  = zoneName + '/z:' + coords[3];

        if (allRooms.currentZoneKey !== zoneKey) {
            gr.reset();
            allRooms.currentZoneKey = zoneKey;
            if (!allRooms.roomZones[zoneKey] || !Array.isArray(allRooms.roomZones[zoneKey])) {
                allRooms.roomZones[zoneKey] = [];
            }
            gr.setRooms(allRooms.roomZones[zoneKey]);
        }

        const r = {
            RoomId: info.num,
            Text:   '',
            x:      parseInt(coords[1]),
            y:      parseInt(coords[2]),
            z:      parseInt(coords[3]),
            Exits:  [],
        };

        // Annotate room based on NPC adjectives
        if (info.Contents && info.Contents.Npcs) {
            for (const i in info.Contents.Npcs) {
                if (info.Contents.Npcs[i].adjectives.indexOf('shop') !== -1) {
                    r.Color = '#AFE1AF';
                    r.Text  = '💰';
                    break;
                }
            }
        }

        // Annotate room based on room details flags
        if (info.details) {
            if (info.details.indexOf('bank') !== -1) {
                r.Color = '#56d44a';
                r.Text  = '🏛️';
            } else if (info.details.indexOf('trainer') !== -1) {
                r.Color = '#fdd';
                r.Text  = '🧠';
            } else if (info.details.indexOf('storage') !== -1) {
                r.Color = '#a88100';
                r.Text  = '🗄️';
            } else if (info.details.indexOf('character') !== -1) {
                r.Color = '#757575';
                r.Text  = '👤';
            }
        }

        // Build exits (2D only — skip exits with a z-delta)
        for (const p in info.exitsv2) {
            const exitInfo = info.exitsv2[p];
            if (exitInfo.dz === 0) {
                r.Exits.push({
                    RoomId: exitInfo.num,
                    x:      exitInfo.dx + parseInt(coords[1]),
                    y:      exitInfo.dy + parseInt(coords[2]),
                    Text:   '',
                });
            }
        }

        allRooms.roomZones[zoneKey].push(r);
        gr.addRoom(r);
        gr.centerOnRoom(r.RoomId);
    }

    // -----------------------------------------------------------------------
    // Registration
    // -----------------------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Room'],
        onGMCP(namespace, body) {
            updateMap();
        },
    });

})();
