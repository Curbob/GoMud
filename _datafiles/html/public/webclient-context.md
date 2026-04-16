# GoMud Web Client — Developer Context

This document describes the architecture of the web client and explains how to
create new virtual windows that respond to GMCP payloads.

---

## File Map

```
webclient-pure.html                               Page shell, layout, init bridge
static/js/webclient-core.js                      Core infrastructure (loaded first)
static/js/windows/window-character.js            Character window (left dock)
static/js/windows/window-vitals.js               Vitals window (left dock)
static/js/windows/window-status.js               Status window (left dock)
static/js/windows/window-party.js                Party window (left dock)
static/js/windows/window-map.js                  Map window (right dock)
static/js/windows/window-comm.js                 Communications window (right dock)
static/css/windows.css                           Shared dock/panel styles
```

`webclient-pure.html` is a Go template. All static asset paths must be prefixed
with `{{ .CONFIG.FilePaths.WebCDNLocation }}`. To add a new window module, add
one `<script>` tag in the appropriate dock comment block.

---

## Current Window Layout

### Left dock

| File | Title | Tabs | GMCP namespaces |
|---|---|---|---|
| `window-character.js` | Character | Overview, Pack, Quests | `Char.Info`, `Char.Stats`, `Char.Inventory`, `Char.Inventory.Backpack`, `Char.Quests`, `Char` |
| `window-vitals.js` | Vitals | — | `Char.Vitals`, `Char` |
| `window-status.js` | Status | Worth, Effects | `Char.Worth`, `Char.Affects`, `Char` |
| `window-party.js` | Party | — | `Party`, `Party.Vitals` |

### Right dock

| File | Title | Tabs | GMCP namespaces |
|---|---|---|---|
| `window-map.js` | Map | — | `Room` |
| `window-comm.js` | Communications | Say, Whisper, Party, Broadcasts | `Comm` |

---

## Architecture Overview

`webclient-core.js` defines these globals, available to all window modules:

```
injectStyles(css)    Append a <style> block to <head>
VirtualWindows       Registry: register(), handleGMCP(), openAll()
VirtualWindow        Class: lifecycle for a single panel
DockSlots            Object: { left, right } DockSlot instances (populated by Client.init())
Client               Shared state and services
```

### Load and init sequence

1. `webclient-core.js` executes — all globals are defined.
2. Window module scripts execute in order — each calls `VirtualWindows.register(...)`,
   recording the window instance and its GMCP handlers.
3. The page `onload` fires, calling `Client.init()`.
4. `Client.init()` initialises `DockSlots`, mounts the terminal, sets up the
   WebSocket and volume sliders, then calls `VirtualWindows.openAll()` — every
   registered window opens immediately on page load.
5. When the WebSocket receives a `!!GMCP(...)` message, `VirtualWindows.handleGMCP()`
   dispatches it to all matching handlers. Handlers whose associated window has
   been closed by the user are silently skipped.

### Terminal font size

`resizeTerminal()` automatically adjusts the xterm.js font size based on how
many dock slots are occupied:

| Dock state | Font size |
|---|---|
| Neither slot has panels | 20px |
| One slot has panels | 18px |
| Both slots have panels | 16px |

---

## VirtualWindow Lifecycle

A `VirtualWindow` has four states:

| `_win` value | Meaning |
|---|---|
| `undefined` | Never opened (before first `open()` call) |
| `'docked'` | Content is in a dock slot panel; no WinBox exists |
| WinBox instance | Floating, visible |
| `false` | Closed by the user; will not reopen for this session |

`open()` is idempotent — safe to call on every GMCP update. It is a no-op if
the window is already open or has been closed by the user.

### Constructor options

```js
new VirtualWindow(id, {
    factory(),        // required — called once on first open; returns WinBox opts
    dock,             // optional — 'left' | 'right'; enables docking
    defaultDocked,    // optional — boolean; start docked instead of floating
    dockedHeight,     // optional — number (px); preferred panel height when docked
})
```

The `factory()` function must:
- Create the content DOM element
- Append it to `document.body` (required before WinBox can mount it)
- Return a WinBox options object with at minimum `{ title, mount: el }`

### Methods

```js
win.open()      // Open (first call) or no-op (already open or closed)
win.isOpen()    // true when floating or docked
win.get()       // Returns WinBox instance when floating, null when docked/closed
win.dock()      // Move from floating to docked
win.undock()    // Move from docked to floating
```

---

## VirtualWindows Registry

```js
VirtualWindows.register({
    window:       win,            // VirtualWindow instance (enables openAll + GMCP skip)
    gmcpHandlers: ['Foo.Bar'],    // GMCP namespaces this module handles
    onGMCP(namespace, body) {     // called on matching GMCP payload
        // ...
    },
});
```

**Dispatch rules:**
- Namespaces are matched from most-specific to least-specific. A payload of
  `Char.Vitals` matches `Char.Vitals` before `Char`.
- Multiple modules may register for the same namespace — all matching handlers
  are called.
- If a handler's associated `window` is closed (`_win === false`), it is
  skipped entirely.

**`Client.GMCPStructs`** is updated before `onGMCP` is called, so handlers
always read the latest value from it directly rather than from the `body`
argument.

---

## Client API (available to window modules)

```js
Client.GMCPStructs          // Object tree of all received GMCP data
Client.sliderValues         // Current volume levels by category key
Client.MusicPlayer          // MP3Player instance (background music)
Client.SoundPlayer          // MP3Player instance (sound effects)
Client.term                 // xterm.js Terminal instance
Client.sendData(str)        // Send a string over the WebSocket; returns bool
Client.debugLog(msg)        // Log only when Client.debug === true
Client.debug                // get/set — enable verbose logging from console
Client.registerCommand(name, description, fn)   // Add a !terminal command
Client.registerShortcut(code, command)          // Add a keyboard shortcut
```

---

## Creating a New Window Module

### 1. Create the file

`static/js/windows/window-example.js`

Every module is an IIFE to keep all state private. The pattern is:
- `injectStyles()` at the top for CSS
- `createDOM()` builds and appends the content element
- A `VirtualWindow` instance with `factory()` returning WinBox opts
- One or more update functions that read from `Client.GMCPStructs`
- `VirtualWindows.register()` at the bottom

```js
'use strict';

(function() {

    // -- Styles (injected once at script load time) --------------------------
    injectStyles(`
        #example-content {
            color: #fff;
            padding: 8px;
            font-family: monospace;
        }
    `);

    // -- DOM factory --------------------------------------------------------
    // Called once on first open. Must append to document.body.
    function createDOM() {
        const el = document.createElement('div');
        el.id = 'example-content';
        el.textContent = 'Waiting for data...';
        document.body.appendChild(el);
        return el;
    }

    // -- VirtualWindow instance ---------------------------------------------
    const win = new VirtualWindow('Example', {
        dock:          'right',   // 'left' | 'right' — enables docking
        defaultDocked: true,      // start docked on page load
        dockedHeight:  200,       // preferred height (px) when docked
        factory() {
            const el = createDOM();
            return {
                title:      'Example',
                mount:      el,
                background: '#1c6b60',
                border:     1,
                x:          'right',
                y:          0,
                width:      363,
                height:     220,
                header:     20,
                bottom:     60,
            };
        },
    });

    // -- Update logic -------------------------------------------------------
    function update() {
        const data = Client.GMCPStructs.Some && Client.GMCPStructs.Some.Namespace;
        if (!data) { return; }

        win.open();
        if (!win.isOpen()) { return; }

        document.getElementById('example-content').textContent = data.someField;
    }

    // -- Registration -------------------------------------------------------
    VirtualWindows.register({
        window:       win,
        gmcpHandlers: ['Some.Namespace', 'Some'],
        onGMCP() { update(); },
    });

})();
```

### 2. Register the script in `webclient-pure.html`

Add one `<script>` tag in the appropriate dock comment block:

```html
<!-- Left dock: character identity, party, communications -->
<script src="{{ .CONFIG.FilePaths.WebCDNLocation }}/static/js/windows/window-example.js"></script>
```

That is all that is required. The window will open on page load, respond to its
GMCP namespaces, and stop receiving payloads if the user closes it.

---

## GMCP Namespace Conventions

Namespaces follow a dot-separated hierarchy. Register the most specific
namespace you need. If you need to handle both a parent and a child namespace,
list both — the dispatch walks from most-specific to least-specific and stops at
the first level that has any registered handlers, calling all of them.

```js
// Receives Char.Worth payloads directly, and also bare Char payloads.
gmcpHandlers: ['Char.Worth', 'Char'],
```

Always read from `Client.GMCPStructs` inside `onGMCP` rather than from the
`body` argument. The store is the source of truth and is always current.

---

## Dock Slot Behaviour

- Two slots exist: `#dock-left` and `#dock-right`, both flex children of
  `#main-container`.
- A slot is zero-width when empty and expands to its stored width when it
  contains panels. A drag handle appears between the slot and the terminal
  when panels are present, allowing the slot width to be resized.
- Panels within a slot can be resized vertically by dragging the handle
  between them.
- Each docked panel has a titlebar with:
  - A pop-out arrow — undocks the panel to a floating WinBox
  - An X — closes the window for the session
- When a floating window has `dock` configured, its WinBox header contains a
  dock button (↓ arrow) that moves it back into the slot.
- Closing a window (from either floating or docked state) sets `_win = false`
  and deregisters it from GMCP dispatch for the remainder of the session.

---

## Extension Points

### Adding a terminal command

```js
Client.registerCommand('!example', 'Print example info', (input) => {
    Client.term.writeln('Example command ran.');
    return true;  // return true to consume the input (clears the field)
});
```

Commands are matched against the exact input string before it is sent to the
server. Return `true` to prevent the string from being sent.

### Adding a keyboard shortcut

```js
// Sends 'look' when the user presses Tab with an empty input field.
Client.registerShortcut('Tab', 'look');
```

The `code` value is a `KeyboardEvent.code` string (e.g. `'KeyM'`, `'Tab'`,
`'F11'`). Shortcuts only fire when the command input field is empty.

### Adding a comm channel tab

In `window-comm.js`, add an entry to the `CHANNELS` array:

```js
{ id: 'guild', label: 'Guild', cssClass: 'guild', active: false },
```

No other changes are required. The tab and its panel are created automatically.
