/**
 * webclient-core.js
 *
 * Core infrastructure for the GoMud web client. Provides:
 *   - Client namespace (shared state accessible by window modules)
 *   - VirtualWindow class (lifecycle management for WinBox panels)
 *   - VirtualWindows registry (GMCP handler dispatch)
 *   - WebSocket connection management
 *   - Terminal (xterm.js) setup
 *   - MSP audio (music + sound)
 *   - Volume slider UI
 *
 * Window modules call VirtualWindows.register(...) to add themselves.
 * The HTML file calls Client.init() on page load.
 */

'use strict';

// ---------------------------------------------------------------------------
// injectStyles
//
// Appends a <style> block to <head>. Called by window modules at load time
// so each module owns and ships its own CSS alongside its JS.
// ---------------------------------------------------------------------------
function injectStyles(css) {
    const style = document.createElement('style');
    style.textContent = css;
    document.head.appendChild(style);
}

// ---------------------------------------------------------------------------
// DockSlot
//
// Manages one side's dock column (#dock-left or #dock-right).
// Handles:
//   - adding / removing panels
//   - showing / hiding the slot (zero-width when empty)
//   - the slot-width drag handle
//   - the per-panel vertical resize handles
// ---------------------------------------------------------------------------
class DockSlot {
    constructor(side) {
        this.side    = side;
        this.el      = document.getElementById('dock-' + side);
        this._panels = [];
        if (!this.el) {
            console.error('DockSlot: #dock-' + side + ' not found. Check that webclient-pure.html contains <div id="dock-' + side + '"> inside #main-container.');
            return;
        }
        this._initSlotResize();
    }

    // Add a content element as a new panel with the given title.
    // height (optional) sets the preferred panel height in px.
    // onClose (optional) called when the panel's X button is clicked.
    // onMoveTo (optional) called with (newSide, dropIdx) when dragged to the opposite slot.
    // Returns the panel wrapper element.
    addPanel(contentEl, title, onPopout, height, onClose, onMoveTo) {
        if (!this.el) { return null; }
        const panel = document.createElement('div');
        panel.className = 'dock-panel';

        // Apply preferred height as a fixed flex-basis so the panel does not
        // grow to fill the slot. The user can still drag the resize handle to
        // redistribute space between panels.
        if (height) {
            panel.style.flex      = '0 0 ' + height + 'px';
            panel.style.flexBasis = height + 'px';
        }

        const titlebar = document.createElement('div');
        titlebar.className = 'dock-panel-titlebar';

        const titleSpan = document.createElement('span');
        titleSpan.className   = 'dock-panel-title';
        titleSpan.textContent = title;

        const popoutBtn = document.createElement('span');
        popoutBtn.className   = 'dock-panel-popout';
        popoutBtn.title       = 'Pop out';
        popoutBtn.textContent = this.side === 'left' ? '\u2197' : '\u2196';  // NE / NW arrow
        popoutBtn.addEventListener('click', onPopout);

        const closeBtn = document.createElement('span');
        closeBtn.className   = 'dock-panel-close';
        closeBtn.title       = 'Close';
        closeBtn.textContent = '\u00d7';  // ×
        closeBtn.addEventListener('click', () => {
            this.removePanel(contentEl);
            if (typeof onClose === 'function') { onClose(); }
        });

        titlebar.appendChild(titleSpan);
        titlebar.appendChild(popoutBtn);
        titlebar.appendChild(closeBtn);

        const content = document.createElement('div');
        content.className = 'dock-panel-content';
        content.appendChild(contentEl);

        panel.appendChild(titlebar);
        panel.appendChild(content);

        // Wire up drag-to-reorder on the titlebar, with cross-slot transfer support
        this._initPanelDrag(titlebar, panel, (newSide, dropIdx) => {
            if (typeof onMoveTo === 'function') { onMoveTo(newSide, dropIdx); }
        });

        // Insert a vertical resize handle before this panel if there are others
        let resizeHandle = null;
        if (this._panels.length > 0) {
            resizeHandle = document.createElement('div');
            resizeHandle.className = 'dock-panel-resize';
            this.el.appendChild(resizeHandle);
            this._initPanelResize(resizeHandle);
        }

        this.el.appendChild(panel);
        this._panels.push({ panel, contentEl, resizeHandle });
        this._setActive(true);
        return panel;
    }

    // Remove a panel by its content element. Returns the content element.
    removePanel(contentEl) {
        if (!this.el) { return contentEl; }
        const idx = this._panels.findIndex(p => p.contentEl === contentEl);
        if (idx === -1) { return contentEl; }

        const { panel, resizeHandle } = this._panels[idx];

        // Remove the resize handle that was inserted before this panel,
        // or the one after it if this is the first panel.
        if (resizeHandle) {
            resizeHandle.remove();
        } else if (this._panels.length > 1) {
            // This was the first panel; remove the handle that was after it
            const next = this._panels[1];
            if (next.resizeHandle) {
                next.resizeHandle.remove();
                next.resizeHandle = null;
            }
        }

        // Move content back out before removing the panel
        document.body.appendChild(contentEl);
        panel.remove();
        this._panels.splice(idx, 1);

        if (this._panels.length === 0) {
            this._setActive(false);
        }
        return contentEl;
    }

    hasPanel(contentEl) {
        return this._panels.some(p => p.contentEl === contentEl);
    }

    _setActive(active) {
        if (active) {
            this.el.classList.add('has-panels');
        } else {
            this.el.classList.remove('has-panels');
        }
        // Defer until after the browser has completed its layout pass so
        // fitAddon.fit() measures the terminal at its new settled dimensions.
        requestAnimationFrame(() => window.dispatchEvent(new Event('resize')));
    }

    // Slot-width drag handle — inserted as a sibling of the slot in
    // #main-container so it is never clipped by the slot's overflow:hidden.
    // Hidden when the slot is empty, shown when it has panels.
    _initSlotResize() {
        if (!this.el) { return; }
        const handle = document.createElement('div');
        handle.className = 'dock-slot-resize dock-slot-resize-' + this.side;
        // Insert adjacent to the slot inside #main-container
        const container = this.el.parentNode;
        if (this.side === 'right') {
            container.insertBefore(handle, this.el);
        } else {
            this.el.insertAdjacentElement('afterend', handle);
        }

        // Keep visibility in sync with the slot's active state
        const observer = new MutationObserver(() => {
            handle.style.display = this.el.classList.contains('has-panels') ? '' : 'none';
        });
        observer.observe(this.el, { attributes: true, attributeFilter: ['class'] });
        handle.style.display = 'none';  // hidden until first panel is added

        let startX, startWidth, _rafPending = false;
        const onMove = (e) => {
            const dx    = (e.clientX || e.touches[0].clientX) - startX;
            const width = Math.max(80, startWidth + (this.side === 'right' ? -dx : dx));
            this.el.style.setProperty('--dock-' + this.side + '-width', width + 'px');
            this.el.style.width = width + 'px';
            if (!_rafPending) {
                _rafPending = true;
                requestAnimationFrame(() => {
                    _rafPending = false;
                    window.dispatchEvent(new Event('resize'));
                });
            }
        };
        const onUp = () => {
            handle.classList.remove('dragging');
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup',   onUp);
            document.removeEventListener('touchmove', onMove);
            document.removeEventListener('touchend',  onUp);
        };
        handle.addEventListener('mousedown', (e) => {
            e.preventDefault();
            handle.classList.add('dragging');
            startX     = e.clientX;
            startWidth = this.el.offsetWidth;
            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup',   onUp);
        });
        handle.addEventListener('touchstart', (e) => {
            startX     = e.touches[0].clientX;
            startWidth = this.el.offsetWidth;
            document.addEventListener('touchmove', onMove, { passive: false });
            document.addEventListener('touchend',  onUp);
        }, { passive: true });
    }

    // Vertical resize handle between two stacked panels
    _initPanelResize(handle) {
        let startY, prevHeight, nextHeight, prevPanel, nextPanel;

        const onMove = (e) => {
            const dy   = (e.clientY || e.touches[0].clientY) - startY;
            const newPrev = Math.max(40, prevHeight + dy);
            const newNext = Math.max(40, nextHeight - dy);
            prevPanel.style.flexBasis = newPrev + 'px';
            prevPanel.style.flex      = '0 0 ' + newPrev + 'px';
            nextPanel.style.flexBasis = newNext + 'px';
            nextPanel.style.flex      = '0 0 ' + newNext + 'px';
        };
        const onUp = () => {
            document.removeEventListener('mousemove', onMove);
            document.removeEventListener('mouseup',   onUp);
            document.removeEventListener('touchmove', onMove);
            document.removeEventListener('touchend',  onUp);
        };
        handle.addEventListener('mousedown', (e) => {
            e.preventDefault();
            startY      = e.clientY;
            prevPanel   = handle.previousElementSibling;
            nextPanel   = handle.nextElementSibling;
            prevHeight  = prevPanel.offsetHeight;
            nextHeight  = nextPanel.offsetHeight;
            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup',   onUp);
        });
        handle.addEventListener('touchstart', (e) => {
            startY      = e.touches[0].clientY;
            prevPanel   = handle.previousElementSibling;
            nextPanel   = handle.nextElementSibling;
            prevHeight  = prevPanel.offsetHeight;
            nextHeight  = nextPanel.offsetHeight;
            document.addEventListener('touchmove', onMove, { passive: false });
            document.addEventListener('touchend',  onUp);
        }, { passive: true });
    }

    // Drag-to-reorder on a panel's titlebar.
    // Shows a ghost label following the cursor and a drop indicator line
    // between panels. On drop, reorders the panel in the DOM and _panels array,
    // or calls onMoveTo(newSide, dropIdx) if dropped into the opposite slot.
    _initPanelDrag(titlebar, panel, onMoveTo) {
        titlebar.addEventListener('mousedown', (e) => {
            // Ignore clicks on the action buttons
            if (e.target.classList.contains('dock-panel-popout') ||
                e.target.classList.contains('dock-panel-close')) {
                return;
            }
            e.preventDefault();

            const srcIdx = this._panels.findIndex(p => p.panel === panel);
            if (srcIdx === -1) { return; }

            const oppSide = this.side === 'left' ? 'right' : 'left';
            const oppSlot = DockSlots[oppSide];

            // Ghost label that follows the cursor
            const ghost = document.createElement('div');
            ghost.className = 'dock-drag-ghost';
            ghost.textContent = titlebar.querySelector('.dock-panel-title').textContent;
            document.body.appendChild(ghost);

            // Two drop indicators — one per slot
            const ownIndicator = document.createElement('div');
            ownIndicator.className = 'dock-drop-indicator';
            ownIndicator.style.display = 'none';
            this.el.appendChild(ownIndicator);

            let oppIndicator = null;
            if (oppSlot && oppSlot.el) {
                oppIndicator = document.createElement('div');
                oppIndicator.className = 'dock-drop-indicator';
                oppIndicator.style.display = 'none';
                oppSlot.el.appendChild(oppIndicator);
            }

            panel.classList.add('dock-dragging');

            let dropSide = this.side;
            let dropIdx  = srcIdx;

            const _calcDropIdx = (slot, clientY) => {
                const panels = slot._panels;
                let idx = panels.length;
                for (let i = 0; i < panels.length; i++) {
                    if (panels[i].panel === panel) { continue; }
                    const r = panels[i].panel.getBoundingClientRect();
                    if (clientY < r.top + r.height / 2) { idx = i; break; }
                }
                return idx;
            };

            const _showIndicator = (indicator, slot, idx) => {
                if (!indicator || !slot.el) { return; }
                const panels    = slot._panels;
                const slotRect  = slot.el.getBoundingClientRect();
                indicator.style.display = 'block';
                if (panels.length === 0 || idx >= panels.length) {
                    const last = panels.length > 0 ? panels[panels.length - 1].panel : null;
                    indicator.style.top = last
                        ? (last.getBoundingClientRect().bottom - slotRect.top + 2) + 'px'
                        : '4px';
                } else {
                    const r = panels[idx].panel.getBoundingClientRect();
                    indicator.style.top = (r.top - slotRect.top - 2) + 'px';
                }
            };

            const onMove = (e) => {
                ghost.style.top = e.clientY + 'px';

                const ownRect = this.el.getBoundingClientRect();
                const oppRect = oppSlot && oppSlot.el ? oppSlot.el.getBoundingClientRect() : null;

                // Determine which slot the cursor is over
                const overOpp = oppRect &&
                    e.clientX >= oppRect.left && e.clientX <= oppRect.right &&
                    oppRect.width > 0;

                if (overOpp) {
                    dropSide = oppSide;
                    dropIdx  = _calcDropIdx(oppSlot, e.clientY);
                    ghost.style.left  = oppRect.left + 'px';
                    ghost.style.width = oppRect.width + 'px';
                    ownIndicator.style.display = 'none';
                    _showIndicator(oppIndicator, oppSlot, dropIdx);
                } else {
                    dropSide = this.side;
                    dropIdx  = _calcDropIdx(this, e.clientY);
                    ghost.style.left  = ownRect.left + 'px';
                    ghost.style.width = ownRect.width + 'px';
                    if (oppIndicator) { oppIndicator.style.display = 'none'; }
                    // Hide own indicator when drop would not change order
                    if (dropIdx === srcIdx || dropIdx === srcIdx + 1) {
                        ownIndicator.style.display = 'none';
                    } else {
                        _showIndicator(ownIndicator, this, dropIdx);
                    }
                }
            };

            // Set initial ghost position
            const initRect = this.el.getBoundingClientRect();
            ghost.style.left  = initRect.left + 'px';
            ghost.style.width = initRect.width + 'px';
            ghost.style.top   = e.clientY + 'px';

            const onUp = () => {
                document.removeEventListener('mousemove', onMove);
                document.removeEventListener('mouseup',   onUp);

                ghost.remove();
                ownIndicator.remove();
                if (oppIndicator) { oppIndicator.remove(); }
                panel.classList.remove('dock-dragging');

                if (dropSide !== this.side) {
                    // Dropped into the opposite slot
                    if (typeof onMoveTo === 'function') {
                        onMoveTo(dropSide, dropIdx);
                    }
                } else if (dropIdx !== srcIdx && dropIdx !== srcIdx + 1) {
                    // Reorder within own slot
                    this._movePanel(srcIdx, dropIdx);
                }
            };

            document.addEventListener('mousemove', onMove);
            document.addEventListener('mouseup',   onUp);
        });
    }

    // Reorder a panel from fromIdx to toIdx (insert-before semantics).
    // Rebuilds the DOM order and the resize handles between panels.
    _movePanel(fromIdx, toIdx) {
        if (fromIdx === toIdx) { return; }

        // Remove all resize handles from the DOM — we'll rebuild them
        this._panels.forEach(p => {
            if (p.resizeHandle) {
                p.resizeHandle.remove();
                p.resizeHandle = null;
            }
        });

        // Reorder the _panels array
        const moved = this._panels.splice(fromIdx, 1)[0];
        const insertAt = toIdx > fromIdx ? toIdx - 1 : toIdx;
        this._panels.splice(insertAt, 0, moved);

        // Re-append panels to the slot in the new order
        this._panels.forEach(p => this.el.appendChild(p.panel));

        // Rebuild resize handles between adjacent panels
        for (let i = 1; i < this._panels.length; i++) {
            const handle = document.createElement('div');
            handle.className = 'dock-panel-resize';
            // Insert before the panel at index i
            this.el.insertBefore(handle, this._panels[i].panel);
            this._panels[i].resizeHandle = handle;
            this._initPanelResize(handle);
        }
    }
}

// Singleton slot instances, populated by Client.init() once the DOM is ready.
const DockSlots = {};

// ---------------------------------------------------------------------------
// VirtualWindow
//
// Wraps a WinBox instance with a well-defined lifecycle and optional docking.
//
// States:
//   undefined  -> never opened
//   'docked'   -> content is in a dock slot panel; no WinBox exists
//   WinBox obj -> floating
//   false      -> user closed it from floating state; will not reopen
//
// Constructor options (passed as the second argument to VirtualWindow):
//   factory()         required  Returns WinBox opts object. Must append
//                               the content element to document.body.
//   dock              optional  'left' | 'right'  — which slot to use.
//                               If omitted the window is float-only.
//   defaultDocked     optional  boolean — start docked instead of floating.
//   dockedHeight      optional  number (px) — preferred panel height when docked.
//                               Defaults to the height from the factory opts.
//
// Usage:
//   const win = new VirtualWindow('id', {
//       factory() { ... return { title, mount: el, ... }; },
//       dock: 'right',
//       defaultDocked: true,
//   });
//   win.open();       creates on first call, no-op if closed by user
//   win.isOpen()      true when floating or docked
//   win.get()         returns WinBox instance, or null when docked/closed
//   win.dock()        move from floating -> docked
//   win.undock()      move from docked   -> floating
// ---------------------------------------------------------------------------
class VirtualWindow {
    constructor(id, options) {
        this._id             = id;
        this._factory        = options.factory;
        this._dockSide       = options.dock || null;
        this._defaultDocked  = options.defaultDocked || false;
        this._dockedHeight   = options.dockedHeight || null;  // preferred px height when docked
        this._win            = undefined;
        this._contentEl      = null;
        this._winboxOpts     = null;
    }

    // Open the window. On first call, honours defaultDocked.
    // Subsequent calls are no-ops unless the window is not yet open.
    open() {
        if (this._win === false)      { return; }  // user closed it
        if (this._win !== undefined)  { return; }  // already open (float or docked)

        // First open: run the factory to get opts + content element
        const opts = this._factory();
        if (!opts) { return; }
        this._winboxOpts = opts;
        this._contentEl  = opts.mount;

        if (this._dockSide && this._defaultDocked) {
            this._dockNow();
        } else {
            this._floatNow();
        }
    }

    isOpen() {
        return this._win === 'docked' || (!!this._win && this._win !== false);
    }

    // Returns the WinBox instance when floating, null when docked or closed.
    get() {
        return (this._win && this._win !== false && this._win !== 'docked')
            ? this._win : null;
    }

    // Move from floating to docked. Safe to call when already docked.
    dock() {
        if (!this._dockSide)          { return; }
        if (this._win === 'docked')   { return; }
        if (!this._win || this._win === false) { return; }

        // Destroy the WinBox without triggering the user-close state
        const wb = this._win;
        wb.onclose = null;
        wb.close();
        this._win = undefined;

        this._dockNow();
    }

    // Move from docked to floating. Safe to call when already floating.
    undock() {
        if (this._win !== 'docked') { return; }

        const slot = DockSlots[this._dockSide];

        // Capture the panel's vertical position before it is removed so the
        // floating window can appear at roughly the same screen position.
        const panel = slot.el.querySelector('.dock-panel');
        let spawnY = null;
        if (panel) {
            const rect = panel.getBoundingClientRect();
            spawnY = Math.round(rect.top);
        }

        slot.removePanel(this._contentEl);
        this._win = undefined;

        this._floatNow(spawnY);
    }

    // -----------------------------------------------------------------------
    // Private
    // -----------------------------------------------------------------------

    _floatNow(spawnY) {
        const opts = Object.assign({}, this._winboxOpts);

        // If spawning from a docked position, place the window at the same
        // vertical position, inset 50px from the edge of the viewport.
        if (spawnY !== undefined && spawnY !== null) {
            opts.y = spawnY;
            opts.x = this._dockSide === 'right'
                ? window.innerWidth  - (opts.width  || 363) - 50
                : 50;
        }

        // Re-attach content to body if it was moved by the dock slot
        if (this._contentEl && !document.body.contains(this._contentEl)) {
            document.body.appendChild(this._contentEl);
        }
        opts.mount = this._contentEl;

        // Inject close handler — sets state to false and removes the content
        // element from the DOM so WinBox's unmount() doesn't leave it visible
        // as a bare element on document.body.
        const userOnClose = opts.onclose;
        opts.onclose = (force) => {
            this._win = false;
            if (this._contentEl && this._contentEl.parentNode) {
                this._contentEl.parentNode.removeChild(this._contentEl);
            }
            if (typeof userOnClose === 'function') { return userOnClose(force); }
            return false;
        };

        // Wrap oncreate to add the dock button.
        // IMPORTANT: this._win must be set before oncreate fires because WinBox
        // calls oncreate synchronously inside its constructor, before the
        // assignment `this._win = new WinBox(opts)` completes. We use a
        // placeholder object so addControl can be called safely, then replace
        // it with the real WinBox instance immediately after construction.
        const existingOncreate = opts.oncreate;
        if (this._dockSide) {
            const self = this;
            opts.oncreate = function(o) {
                if (existingOncreate) { existingOncreate.call(this, o); }
                this.addControl({
                    index: 0,
                    class: 'wb-dock-btn',
                    click: () => self.dock(),
                });
            };
        } else if (existingOncreate) {
            opts.oncreate = existingOncreate;
        }

        this._win = new WinBox(opts);
    }

    _dockNow() {
        const slot   = DockSlots[this._dockSide];
        const height = this._dockedHeight || (this._winboxOpts && this._winboxOpts.height) || null;
        slot.addPanel(
            this._contentEl,
            this._winboxOpts.title,
            () => this.undock(),
            height,
            () => {
                // User clicked X on the docked panel — same semantics as
                // closing a floating window: remove content and deregister.
                if (this._contentEl && this._contentEl.parentNode) {
                    this._contentEl.parentNode.removeChild(this._contentEl);
                }
                this._win = false;
            },
            (newSide) => {
                // User dragged the panel to the opposite slot.
                // Remove from the current slot, update _dockSide, re-dock.
                slot.removePanel(this._contentEl);
                this._dockSide = newSide;
                this._win = undefined;
                this._dockNow();
            }
        );
        this._win = 'docked';
    }
}

// ---------------------------------------------------------------------------
// VirtualWindows registry
//
// Window modules call VirtualWindows.register(descriptor) where descriptor is:
//   {
//       window:       VirtualWindow instance (required for openAll)
//       gmcpHandlers: ['Char.Vitals', 'Char'],   // GMCP namespaces this handles
//       onGMCP(namespace, data) { ... }           // called when any listed namespace updates
//   }
//
// Multiple modules may register for the same namespace — all handlers are called.
// handleGMCP(namespace, body) walks from the most-specific to least-specific
// namespace segment and calls every handler registered at the first level that
// has any handlers.
// openAll() opens every registered window immediately — called by Client.init().
// ---------------------------------------------------------------------------
const VirtualWindows = (() => {
    // Map<gmcpNamespace, Array<handler function>>
    const _handlers = {};
    // Ordered list of all registered VirtualWindow instances
    const _windows  = [];

    function register(descriptor) {
        if (!descriptor || !Array.isArray(descriptor.gmcpHandlers)) {
            console.error('VirtualWindows.register: descriptor must have gmcpHandlers array');
            return;
        }
        if (typeof descriptor.onGMCP !== 'function') {
            console.error('VirtualWindows.register: descriptor must have onGMCP function');
            return;
        }
        const win = (descriptor.window instanceof VirtualWindow) ? descriptor.window : null;
        descriptor.gmcpHandlers.forEach(ns => {
            if (!_handlers[ns]) {
                _handlers[ns] = [];
            }
            // Store the handler alongside its window so dispatch can skip
            // handlers whose window has been closed by the user.
            _handlers[ns].push({ fn: descriptor.onGMCP.bind(descriptor), win });
        });
        if (win) {
            _windows.push(win);
        }
    }

    function handleGMCP(namespace, body) {
        // Walk from most-specific to least-specific namespace segment.
        // Call all handlers registered at the first matching level,
        // skipping any whose associated window has been closed.
        const parts = namespace.split('.');
        for (let i = parts.length; i >= 1; i--) {
            const path = parts.slice(0, i).join('.');
            if (_handlers[path] && _handlers[path].length > 0) {
                _handlers[path].forEach(entry => {
                    if (entry.win && !entry.win.isOpen()) { return; }
                    entry.fn(namespace, body);
                });
                return;
            }
        }
        console.log('GMCP (unhandled):', namespace, body);
    }

    function openAll() {
        _windows.forEach(win => win.open());
    }

    return { register, handleGMCP, openAll };
})();

// ---------------------------------------------------------------------------
// Client namespace
//
// Shared state and services that window modules may read or call.
// Nothing here is truly private — window modules are trusted collaborators.
// ---------------------------------------------------------------------------
const Client = (() => {

    // -----------------------------------------------------------------------
    // Audio
    // -----------------------------------------------------------------------
    let baseMp3Url = '';
    const MusicPlayer = new MP3Player(false);
    const SoundPlayer = new MP3Player(true);

    // -----------------------------------------------------------------------
    // Terminal
    // -----------------------------------------------------------------------
    const term = new window.Terminal({
        cols: 80,
        rows: 60,
        cursorBlink: true,
        fontSize: 20,
    });
    const fitAddon = new window.FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    function resizeTerminal() {
        const hasLeft  = DockSlots.left  && DockSlots.left.el  && DockSlots.left.el.classList.contains('has-panels');
        const hasRight = DockSlots.right && DockSlots.right.el && DockSlots.right.el.classList.contains('has-panels');
        const fontSize = (hasLeft && hasRight) ? 16 : (hasLeft || hasRight) ? 18 : 20;
        if (term.options.fontSize !== fontSize) {
            term.options.fontSize = fontSize;
        }
        fitAddon.fit();
    }

    // -----------------------------------------------------------------------
    // Networking stats
    // -----------------------------------------------------------------------
    let payloadsReceived = 0;
    let totalBytesReceived = 0;
    let payloadsSent      = 0;
    let totalBytesSent    = 0;

    // -----------------------------------------------------------------------
    // Command history
    // -----------------------------------------------------------------------
    let commandHistory          = [];
    let historyPosition         = 0;
    const commandHistoryMaxLength = 30;

    // -----------------------------------------------------------------------
    // GMCP state store
    //
    // GMCPStructs holds the most-recently-received value for every namespace.
    // Window modules read from it inside their onGMCP callbacks.
    // -----------------------------------------------------------------------
    const GMCPStructs = {};

    function _applyGMCPPayload(namespace, body) {
        const parts        = namespace.split('.');
        const lastProperty = parts.pop();
        let cursor         = GMCPStructs;
        for (const seg of parts) {
            if (!cursor[seg]) {
                cursor[seg] = {};
            }
            cursor = cursor[seg];
        }
        cursor[lastProperty] = body;
    }

    // -----------------------------------------------------------------------
    // WebSocket
    // -----------------------------------------------------------------------
    let socket               = null;
    let pendingReconnectToken = null;
    let debugOutput           = false;  // set Client.debug = true from the console to enable

    function debugLog(msg) {
        if (debugOutput) {
            console.log(msg);
        }
    }

    function sendData(dataToSend) {
        if (!socket || socket.readyState !== WebSocket.OPEN) {
            return false;
        }
        payloadsSent++;
        totalBytesSent += dataToSend.length;
        socket.send(dataToSend);
        return true;
    }

    function _parseMSPProps(parts, startIndex) {
        const props = {};
        for (let i = startIndex; i < parts.length; i++) {
            const eq = parts[i].indexOf('=');
            if (eq !== -1) {
                props[parts[i].slice(0, eq)] = parts[i].slice(eq + 1);
            }
        }
        return props;
    }

    function _handleMusicCommand(raw) {
        const inner  = raw.slice(8, raw.length - 1);
        const parts  = inner.split(' ');
        const fileName = parts[0];
        const obj    = _parseMSPProps(parts, 1);

        if (fileName === 'Off') {
            if (obj.U) {
                baseMp3Url = obj.U;
                if (baseMp3Url[baseMp3Url.length - 1] !== '/') {
                    baseMp3Url += '/';
                }
            } else {
                MusicPlayer.stop();
            }
            return;
        }

        let loopMusic  = true;
        let soundLevel = 1.0;
        if (obj.L && obj.L !== '-1') { loopMusic  = false; }
        if (obj.V)                    { soundLevel = Number(obj.V) / 100; }

        if (!MusicPlayer.isPlaying(baseMp3Url + fileName)) {
            MusicPlayer.play(baseMp3Url + fileName, loopMusic, soundLevel * (sliderValues['music'] / 100));
        }
    }

    function _handleSoundCommand(raw) {
        const inner    = raw.slice(8, raw.length - 1);
        const parts    = inner.split(' ');
        const fileName = parts[0];
        const obj      = _parseMSPProps(parts, 1);

        if (fileName === 'Off') {
            if (obj.U) {
                baseMp3Url = obj.U;
                if (baseMp3Url[baseMp3Url.length - 1] !== '/') {
                    baseMp3Url += '/';
                }
            } else {
                SoundPlayer.stop();
            }
            return;
        }

        let soundLevel = 1.0;
        let loopSound  = true;
        if (obj.L && obj.L !== '-1') { loopSound  = false; }
        if (obj.V)                    { soundLevel = Number(obj.V) / 100; }

        const typeKey = ((obj.T || 'other').toLowerCase()) + ' sounds';
        SoundPlayer.play(baseMp3Url + fileName, false, soundLevel * (sliderValues[typeKey] / 100));
    }

    function _handleWebclientCommand(data) {
        if (data.startsWith('TEXTMASK:')) {
            debugLog(data);
            textInput.type = data.substring(9) === 'true' ? 'password' : 'text';
            return true;
        }
        if (data.startsWith('RELOGTKN:')) {
            pendingReconnectToken = data.substring(9);
            return true;
        }
        return false;
    }

    function _onMessage(event) {
        payloadsReceived++;
        totalBytesReceived += event.data.length;

        // Webclient protocol commands (TEXTMASK:, RELOGTKN:)
        if (_handleWebclientCommand(event.data)) {
            return;
        }

        // MSP / GMCP commands (all start with "!!")
        if (event.data.length > 2 && event.data.slice(0, 2) === '!!') {

            if (event.data.slice(0, 7) === '!!GMCP(') {
                const gmcpPayload = event.data.trim().slice(7, event.data.length - 1).trim();
                const lastChar    = gmcpPayload[gmcpPayload.length - 1];
                const jsonIndex   = (lastChar === '}') ? gmcpPayload.indexOf('{') : gmcpPayload.indexOf('[');
                if (jsonIndex === -1) {
                    return;
                }
                const gmcpNamespace = gmcpPayload.slice(0, jsonIndex).trim();
                const gmcpBody      = JSON.parse(gmcpPayload.slice(jsonIndex).trim());
                _applyGMCPPayload(gmcpNamespace, gmcpBody);
                VirtualWindows.handleGMCP(gmcpNamespace, gmcpBody);
                return;
            }

            if (event.data.slice(0, 8) === '!!MUSIC(') {
                _handleMusicCommand(event.data);
                return;
            }

            if (event.data.slice(0, 8) === '!!SOUND(') {
                _handleSoundCommand(event.data);
                return;
            }
        }

        term.write(event.data);
    }

    function requestInitialGMCPData() {
        sendData('!!GMCP(Core.Hello {"Client":"WebClient","Version":"1.0.0"})');
        sendData('!!GMCP(Core.Supports.Set ["Char 1","Char.Info 1","Char.Stats 1","Char.Vitals 1","Char.Worth 1","Char.Affects 1","Char.Inventory 1","Char.Inventory.Backpack 1","Char.Quests 1","Room 1","Room.Info 1","Comm 1","Party 1","Party.Vitals 1"])');
    }

    function attachSocketHandlers(openMessage, clearOnOpen) {
        socket.onopen = function() {
            if (clearOnOpen) { term.clear(); }
            term.writeln(openMessage);
            connectButton.style.display = 'none';
            connectButton.disabled = true;
            textInput.focus();
            requestInitialGMCPData();
        };

        socket.onmessage = _onMessage;

        socket.onerror = function(error) {
            term.writeln('Error: ' + (error.message || 'unknown'));
        };

        socket.onclose = function(event) {
            if (event.wasClean) {
                term.writeln('Connection closed cleanly, code=' + event.code + ', reason=' + event.reason);
            } else {
                term.writeln('Connection died');
            }
            connectButton.style.display = 'block';
            connectButton.disabled = false;

            if (textInput.type === 'password') {
                textInput.value = '';
                textInput.type  = 'text';
            }

            if (pendingReconnectToken) {
                const token = pendingReconnectToken;
                pendingReconnectToken = null;
                setTimeout(() => reconnectWithToken(token), 500);
            }
        };
    }

    function reconnectWithToken(token) {
        debugLog('Reconnecting with copyover token');
        const wsUrl = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws';
        socket = new WebSocket(wsUrl);
        attachSocketHandlers('Reconnected after server reboot.', false);
        const origOnOpen = socket.onopen;
        socket.onopen = function() {
            origOnOpen();
            socket.send(token);
        };
    }

    // -----------------------------------------------------------------------
    // Volume sliders
    // -----------------------------------------------------------------------
    const defaultSliders = {
        'music':               75,
        'combat sounds':       75,
        'movement sounds':     75,
        'environment sounds':  75,
        'other sounds':        75,
    };

    let sliderValues        = { ...defaultSliders };
    let unmutedSliderValues = null;

    function getSpeakerIcon(value) {
        value = Number(value);
        if (value === 0)       { return '🔇'; }
        if (value < 33)        { return '🔈'; }
        if (value < 66)        { return '🔉'; }
        return '🔊';
    }

    function buildSliders() {
        const container = document.getElementById('sliders-container');
        container.innerHTML = '';

        Object.keys(sliderValues).forEach(key => {
            const wrapper = document.createElement('div');
            wrapper.className = 'slider-container';

            const label = document.createElement('label');
            label.textContent = key.toLowerCase().split(' ').map(w => w.charAt(0).toUpperCase() + w.slice(1)).join(' ');

            const slider = document.createElement('input');
            slider.type  = 'range';
            slider.min   = 0;
            slider.max   = 100;
            slider.value = sliderValues[key];

            const iconSpan = document.createElement('span');
            iconSpan.className   = 'slider-icon';
            iconSpan.textContent = getSpeakerIcon(sliderValues[key]);

            slider.addEventListener('input', e => {
                const val = Number(e.target.value);
                sliderValues[key] = val;
                iconSpan.textContent = getSpeakerIcon(val);
                localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
                MusicPlayer.setGlobalVolume(sliderValues['music'] / 100);

                const muteCheckbox = document.getElementById('mute-checkbox');
                if (muteCheckbox.checked && val > 0) {
                    muteCheckbox.checked = false;
                    localStorage.setItem('muteAllSound', JSON.stringify(false));
                    document.getElementById('mute-icon').textContent = '🔊';
                }
            });

            wrapper.appendChild(label);
            wrapper.appendChild(slider);
            wrapper.appendChild(iconSpan);
            container.appendChild(wrapper);
        });
    }

    function toggleMuteAll() {
        const muteCheckbox = document.getElementById('mute-checkbox');
        const muteIcon     = document.getElementById('mute-icon');
        const isChecked    = muteCheckbox.checked;

        if (isChecked) {
            unmutedSliderValues = { ...sliderValues };
            localStorage.setItem('unmutedSliderValues', JSON.stringify(unmutedSliderValues));
            Object.keys(sliderValues).forEach(k => { sliderValues[k] = 0; });
            localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
            buildSliders();
            muteIcon.textContent = '🔇';
            MusicPlayer.setGlobalVolume(0);
            localStorage.setItem('muteAllSound', JSON.stringify(true));
        } else {
            const savedUnmuted = localStorage.getItem('unmutedSliderValues');
            if (savedUnmuted) {
                let loaded = JSON.parse(savedUnmuted) || {};
                loaded = { ...defaultSliders, ...loaded };
                unmutedSliderValues = { ...loaded };
                sliderValues = { ...unmutedSliderValues };
                localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
            }
            buildSliders();
            muteIcon.textContent = '🔊';
            MusicPlayer.setGlobalVolume(sliderValues['music'] / 100);
            localStorage.setItem('muteAllSound', JSON.stringify(false));
        }
    }

    function toggleMenu() {
        const menu = document.getElementById('floating-menu');
        menu.style.display = (menu.style.display === 'none' || menu.style.display === '') ? 'block' : 'none';
    }

    // -----------------------------------------------------------------------
    // Net stats
    // -----------------------------------------------------------------------
    function printNetStats() {
        term.writeln('');
        term.writeln(' Request Ct: ' + String(payloadsSent));
        term.writeln(' Bytes Sent: ' + String(Math.round(totalBytesSent    / 1024 * 100) / 100) + 'kb');
        term.writeln('Response Ct: ' + String(payloadsReceived));
        term.writeln(' Bytes Rcvd: ' + String(Math.round(totalBytesReceived / 1024 * 100) / 100) + 'kb');
        term.writeln('');
    }

    // -----------------------------------------------------------------------
    // Keyboard shortcuts
    //
    // Window modules may call Client.registerShortcut(code, command) to add
    // their own bindings, e.g. Client.registerShortcut('KeyM', 'map').
    // -----------------------------------------------------------------------
    const codeShortcuts = {
        Numpad1: 'southwest', Numpad2: 'south',  Numpad3: 'southeast',
        Numpad4: 'west',      Numpad5: 'default', Numpad6: 'east',
        Numpad7: 'northwest', Numpad8: 'north',   Numpad9: 'northeast',
        F1: '=1', F2: '=2', F3: '=3',  F4: '=4',  F5: '=5',
        F6: '=6', F7: '=7', F8: '=8',  F9: '=9',  F10: '=10',
        ArrowUp: 'north', ArrowDown: 'south', ArrowLeft: 'west', ArrowRight: 'east',
    };

    function registerShortcut(code, command) {
        codeShortcuts[code] = command;
    }

    // -----------------------------------------------------------------------
    // Terminal commands
    //
    // Window modules may call Client.registerCommand(name, description, fn)
    // to add their own !commands processed before sending to the server.
    // fn receives the full input string and returns true if it handled it.
    // -----------------------------------------------------------------------
    const specialCommands = {
        '!net': { description: 'Print out network traffic stats', fn: () => { printNetStats(); return true; } },
    };

    function registerCommand(name, description, fn) {
        specialCommands[name] = { description, fn };
    }

    // -----------------------------------------------------------------------
    // DOM references (resolved at init time)
    // -----------------------------------------------------------------------
    let connectButton, textOutput, textInput;

    // -----------------------------------------------------------------------
    // init()
    // -----------------------------------------------------------------------
    function init() {
        // Initialise dock slots first — VirtualWindows.openAll() depends on them.
        DockSlots.left  = new DockSlot('left');
        DockSlots.right = new DockSlot('right');

        connectButton = document.getElementById('connect-button');
        textOutput    = document.getElementById('terminal');
        textInput     = document.getElementById('command-input');

        // Mount terminal
        term.open(textOutput);
        window.addEventListener('resize', resizeTerminal);
        resizeTerminal();

        // Keep focus on terminal on click (not drag)
        let isDragging = false;
        textOutput.addEventListener('mousedown', () => { isDragging = false; });
        textOutput.addEventListener('mousemove', () => { isDragging = true; });
        textOutput.addEventListener('mouseup', () => {
            const selected = window.getSelection().toString();
            if (!isDragging && !selected) { textInput.focus(); }
            isDragging = false;
        });

        // Connect button
        connectButton.addEventListener('click', () => {
            if (socket && socket.readyState === WebSocket.OPEN) {
                socket.close();
                return;
            }
            const wsUrl = (location.protocol === 'https:' ? 'wss://' : 'ws://') + location.host + '/ws';
            debugLog('Connecting to: ' + wsUrl);
            socket = new WebSocket(wsUrl);
            attachSocketHandlers('Connected to the server!', true);
        });

        // Input keydown
        textInput.addEventListener('keydown', function(event) {
            // F-key macros
            if (event.key.substring(0, 1) === 'F' && event.key.length === 2) {
                sendData('=' + event.key.substring(1));
                if (event.preventDefault) { event.preventDefault(); }
                return false;
            }

            // Command history
            if (event.key === 'ArrowUp' || event.key === 'ArrowDown') {
                historyPosition += (event.key === 'ArrowUp') ? 1 : -1;
                if (historyPosition < 1) { historyPosition = 1; }
                if (historyPosition > commandHistory.length) { historyPosition = commandHistory.length; }
                event.target.value = commandHistory[commandHistory.length - historyPosition];
                return;
            }

            // Numpad / arrow shortcuts when input is empty
            if (textInput.value.length === 0 && codeShortcuts[event.code]) {
                sendData(codeShortcuts[event.code]);
                if (event.preventDefault) { event.preventDefault(); }
                return false;
            }

            // Enter
            if (event.key === 'Enter') {
                const currentValue = event.target.value;
                const allowBlankSubmit = textInput.type === 'password' || term.buffer.active.cursorY < 20;

                // Ignore accidental blank submits from the webclient during normal
                // gameplay, but allow them for login/signup prompt flows where
                // Enter can intentionally mean "skip".
                if (currentValue === '' && !allowBlankSubmit) {
                    if (event.preventDefault) { event.preventDefault(); }
                    return false;
                }

                if (currentValue !== '' && textInput.type !== 'password') {
                    commandHistory.push(currentValue);
                    historyPosition = 0;
                    if (commandHistory.length > commandHistoryMaxLength) {
                        commandHistory = commandHistory.slice(commandHistory.length - commandHistoryMaxLength);
                    }
                }

                const cmd = specialCommands[currentValue];
                if (cmd) {
                    if (cmd.fn(currentValue)) {
                        event.target.value = '';
                        return;
                    }
                }

                if (sendData(currentValue)) {
                    event.target.value = '';
                } else {
                    term.writeln('Not connected to the server. Did you click the Connect button?');
                }
            }
        });

        // Volume sliders: load from localStorage
        const savedValues = localStorage.getItem('sliderValues');
        if (savedValues) {
            try {
                sliderValues = { ...defaultSliders, ...JSON.parse(savedValues) };
            } catch (e) {
                console.warn('Could not parse saved sliderValues, using defaults.');
            }
        } else {
            localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
        }

        const savedMute = localStorage.getItem('muteAllSound');
        if (savedMute) {
            try {
                document.getElementById('mute-checkbox').checked = JSON.parse(savedMute);
            } catch (e) {
                console.warn('Could not parse muteAllSound, ignoring.');
            }
        }

        buildSliders();

        const muteCheckbox = document.getElementById('mute-checkbox');
        const muteIcon     = document.getElementById('mute-icon');

        if (muteCheckbox.checked) {
            const savedUnmuted = localStorage.getItem('unmutedSliderValues');
            if (savedUnmuted) {
                try {
                    unmutedSliderValues = { ...defaultSliders, ...JSON.parse(savedUnmuted) };
                } catch (e) {
                    console.warn('Could not parse unmutedSliderValues.');
                }
            }
            Object.keys(sliderValues).forEach(k => { sliderValues[k] = 0; });
            localStorage.setItem('sliderValues', JSON.stringify(sliderValues));
            buildSliders();
            muteIcon.textContent = '🔇';
            MusicPlayer.setGlobalVolume(0);
        } else {
            MusicPlayer.setGlobalVolume(sliderValues['music'] / 100);
            muteIcon.textContent = '🔊';
        }

        // Log available commands to console
        console.log('%cterminal commands:', 'font-weight:bold;');
        let longest = 0;
        for (const k in specialCommands) { if (k.length > longest) { longest = k.length; } }
        for (const k in specialCommands) { console.log('  ' + k.padEnd(longest) + ' - ' + specialCommands[k].description); }
        console.log('%cconsole commands:', 'font-weight:bold;');
        console.log('  Client.debug = true   - enable debug logging');
        console.log('  Client.registerCommand(name, description, fn)  - add a terminal command');
        console.log('  Client.registerShortcut(code, command)          - add a keyboard shortcut');

        // Open all registered virtual windows immediately so they are present
        // on page load rather than waiting for the first GMCP payload.
        VirtualWindows.openAll();
    }

    // -----------------------------------------------------------------------
    // Public surface
    // -----------------------------------------------------------------------
    return {
        // Services
        get term()         { return term; },
        get MusicPlayer()  { return MusicPlayer; },
        get SoundPlayer()  { return SoundPlayer; },

        // Shared state (read by window modules)
        get GMCPStructs()  { return GMCPStructs; },
        // sliderValues is a `let` that gets reassigned on mute/unmute, so the
        // getter captures the variable binding, not a snapshot of the object.
        get sliderValues() { return sliderValues; },

        // Debug toggle: set Client.debug = true from the browser console
        get debug()        { return debugOutput; },
        set debug(v)       { debugOutput = !!v; },

        // Extension points for window modules
        registerCommand,
        registerShortcut,

        // Functions called from HTML event handlers
        init,
        toggleMenu,
        toggleMuteAll,

        // Utility
        sendData,
        debugLog,
    };
})();
