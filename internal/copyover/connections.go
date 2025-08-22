package copyover

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sort"
	"strconv"
	"syscall"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/gorilla/websocket"
)

// ConnectionState represents a saved connection
type ConnectionState struct {
	ConnectionId connections.ConnectionId `json:"connection_id"`
	IsWebsocket  bool                     `json:"is_websocket"`
	FdIndex      int                      `json:"fd_index"` // Index in ExtraFiles array
}

// ListenerState represents a saved listener
type ListenerState struct {
	Network string `json:"network"`  // "tcp", "tcp4", "tcp6"
	Address string `json:"address"`  // e.g., ":33333"
	FdIndex int    `json:"fd_index"` // Index in ExtraFiles array
}

var (
	// Maps to store connections and listeners during gathering
	gatheredConnections map[connections.ConnectionId]*connections.ConnectionDetails
	gatheredListeners   map[string]net.Listener

	// Maps to store FDs right before exec
	connectionFds map[string]*os.File
	listenerFds   map[string]*os.File

	// Preserved listeners and connections after recovery
	preservedListeners   map[string]net.Listener
	preservedConnections map[connections.ConnectionId]net.Conn
)

func init() {
	gatheredConnections = make(map[connections.ConnectionId]*connections.ConnectionDetails)
	gatheredListeners = make(map[string]net.Listener)
	connectionFds = make(map[string]*os.File)
	listenerFds = make(map[string]*os.File)
	preservedListeners = make(map[string]net.Listener)
	preservedConnections = make(map[connections.ConnectionId]net.Conn)

	// Register the subsystems - listeners must be registered before connections
	// so that listeners are gathered first and connections can calculate correct FD indices
	Register("listeners", gatherListeners, restoreListeners)
	Register("connections", gatherConnections, restoreConnections)
}

// gatherConnections saves all active connections
func gatherConnections() (interface{}, error) {
	activeConns := connections.GetActiveConnections()
	states := make([]ConnectionState, 0, len(activeConns))

	// Connections come after listeners in the FD array
	// We need to account for how many listeners we have
	fdIndex := len(gatheredListeners)
	for _, conn := range activeConns {
		// Get the underlying connection
		var isWebsocket bool

		if conn.GetWebsocket() != nil {
			// Skip websockets for now - they need special handling
			// TODO: Implement websocket copyover
			mudlog.Info("Copyover", "skipping", "Websocket connection", "id", conn.ConnectionId())
			continue
		} else {
			isWebsocket = false
		}

		if conn.GetConn() == nil {
			continue
		}

		// Store connection reference for later FD extraction
		connId := conn.ConnectionId()
		gatheredConnections[connId] = conn

		// Create state
		state := ConnectionState{
			ConnectionId: connId,
			IsWebsocket:  isWebsocket,
			FdIndex:      fdIndex,
		}

		states = append(states, state)
		fdIndex++
	}

	mudlog.Info("Copyover", "connections", "Gathered connections", "count", len(states))
	return states, nil
}

// restoreConnections rebuilds connections from saved state
func restoreConnections(data interface{}) error {
	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal connection data: %w", err)
	}

	var states []ConnectionState
	if err := json.Unmarshal(jsonData, &states); err != nil {
		return fmt.Errorf("failed to unmarshal connection states: %w", err)
	}

	// Get extra files from environment
	extraFiles := getExtraFiles()

	for _, state := range states {
		// The FdIndex should already account for listeners coming first
		if state.FdIndex >= len(extraFiles) {
			mudlog.Error("Copyover", "error", "Invalid FD index", "index", state.FdIndex, "max", len(extraFiles))
			continue
		}

		mudlog.Info("Copyover", "restore", "Restoring connection", "id", state.ConnectionId, "fdIndex", state.FdIndex, "totalFds", len(extraFiles))
		file := extraFiles[state.FdIndex]

		// Recreate network connection
		conn, err := net.FileConn(file)
		if err != nil {
			mudlog.Error("Copyover", "error", "Failed to restore connection", "err", err)
			continue
		}

		// Verify the connection is valid
		if conn == nil {
			mudlog.Error("Copyover", "error", "Restored connection is nil", "id", state.ConnectionId)
			continue
		}

		// Store preserved connection
		preservedConnections[state.ConnectionId] = conn

		// Add the connection back to the connection manager with its original ID
		var wsConn *websocket.Conn = nil
		if state.IsWebsocket {
			// For websockets, we'd need special handling - skip for now
			mudlog.Info("Copyover", "connection", "Skipping websocket reattachment", "id", state.ConnectionId)
		} else {
			// Re-add the connection to the connection manager
			cd := connections.AddWithId(state.ConnectionId, conn, wsConn)

			// Set state to LoggedIn
			cd.SetState(connections.LoggedIn)

			// Input handlers will be added by the user restoration process
			// to avoid import cycles
		}

		mudlog.Info("Copyover", "connection", "Restored", "id", state.ConnectionId)
	}

	mudlog.Info("Copyover", "connections", "Restored connections", "count", len(preservedConnections))
	return nil
}

// gatherListeners saves all active listeners
func gatherListeners() (interface{}, error) {
	listeners := connections.GetListeners()
	states := make(map[string]ListenerState)

	fdIndex := 0
	for name, listener := range listeners {
		if listener == nil {
			continue
		}

		// Store listener reference for later FD extraction
		gatheredListeners[name] = listener

		// Get listener address
		addr := listener.Addr()

		// Create state
		state := ListenerState{
			Network: addr.Network(),
			Address: addr.String(),
			FdIndex: fdIndex,
		}

		states[name] = state
		fdIndex++
	}

	mudlog.Info("Copyover", "listeners", "Gathered listeners", "count", len(states))
	return states, nil
}

// restoreListeners rebuilds listeners from saved state
func restoreListeners(data interface{}) error {
	// Type assertion with JSON remarshal for safety
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal listener data: %w", err)
	}

	var states map[string]ListenerState
	if err := json.Unmarshal(jsonData, &states); err != nil {
		return fmt.Errorf("failed to unmarshal listener states: %w", err)
	}

	// Get extra files from environment
	extraFiles := getExtraFiles()

	// Sort listener names for consistent ordering (same as when gathering)
	var listenerNames []string
	for name := range states {
		listenerNames = append(listenerNames, name)
	}
	sort.Strings(listenerNames)

	for _, name := range listenerNames {
		state := states[name]
		if state.FdIndex >= len(extraFiles) {
			mudlog.Error("Copyover", "error", "Invalid listener FD index", "index", state.FdIndex, "max", len(extraFiles))
			continue
		}

		file := extraFiles[state.FdIndex]

		// Recreate listener
		listener, err := net.FileListener(file)
		if err != nil {
			mudlog.Error("Copyover", "error", "Failed to restore listener", "name", name, "err", err)
			continue
		}

		preservedListeners[name] = listener
		mudlog.Info("Copyover", "listener", "Restored", "name", name, "address", state.Address)
	}

	mudlog.Info("Copyover", "listeners", "Restored listeners", "count", len(preservedListeners))
	return nil
}

// GetPreservedListeners returns listeners restored from copyover
func GetPreservedListeners() map[string]net.Listener {
	return preservedListeners
}

// GetPreservedConnections returns connections restored from copyover
func GetPreservedConnections() map[connections.ConnectionId]net.Conn {
	return preservedConnections
}

// PrepareFileDescriptors collects all FDs for passing to the new process
// This should be called right before exec to minimize the time connections are broken
func PrepareFileDescriptors() ([]*os.File, error) {
	var extraFiles []*os.File

	// Extract listener FDs first (they must come first to match the FD indices)
	// Sort listener names for consistent ordering
	var listenerNames []string
	for name := range gatheredListeners {
		listenerNames = append(listenerNames, name)
	}
	// Sort to ensure consistent ordering
	sort.Strings(listenerNames)

	for _, name := range listenerNames {
		listener := gatheredListeners[name]
		if listener == nil {
			continue
		}

		file, err := extractListenerFd(listener)
		if err != nil {
			mudlog.Error("Copyover", "error", "Failed to extract listener FD", "name", name, "err", err)
			continue
		}

		listenerFds[name] = file
		extraFiles = append(extraFiles, file)
		mudlog.Info("Copyover", "fd", "Added listener", "name", name, "fd", file.Fd(), "index", len(extraFiles)-1)
	}

	// Extract connection FDs (they come after listeners)
	// Sort connection IDs for consistent ordering
	var connIds []connections.ConnectionId
	for connId := range gatheredConnections {
		connIds = append(connIds, connId)
	}
	// Sort to ensure consistent ordering
	sort.Slice(connIds, func(i, j int) bool {
		return connIds[i] < connIds[j]
	})

	for _, connId := range connIds {
		conn := gatheredConnections[connId]
		if conn == nil || conn.GetConn() == nil {
			continue
		}

		file, err := extractFd(conn.GetConn())
		if err != nil {
			mudlog.Error("Copyover", "error", "Failed to extract FD", "connId", connId, "err", err)
			continue
		}

		connectionFds[fmt.Sprintf("%d", connId)] = file
		extraFiles = append(extraFiles, file)
		mudlog.Info("Copyover", "fd", "Added connection", "id", connId, "fd", file.Fd(), "index", len(extraFiles)-1)
	}

	return extraFiles, nil
}

// Helper functions

// extractFd extracts a file descriptor from a net.Conn
func extractFd(conn net.Conn) (*os.File, error) {
	// Use reflection to get the underlying file descriptor
	switch c := conn.(type) {
	case *net.TCPConn:
		// Get file from TCPConn - this duplicates the FD
		file, err := c.File()
		if err != nil {
			return nil, fmt.Errorf("failed to get file from TCPConn: %w", err)
		}
		// Note: TCPConn.File() already duplicates the FD, so we get a new FD
		// that won't affect the original connection when we use it
		return file, nil

	case interface{ File() (*os.File, error) }:
		// Generic interface for connections with File() method
		file, err := c.File()
		if err != nil {
			return nil, fmt.Errorf("failed to get file from connection: %w", err)
		}
		return file, nil

	default:
		return nil, fmt.Errorf("unsupported connection type: %T", conn)
	}
}

// extractListenerFd extracts a file descriptor from a net.Listener
func extractListenerFd(listener net.Listener) (*os.File, error) {
	// Use reflection to get the underlying file descriptor
	switch l := listener.(type) {
	case *net.TCPListener:
		// Get file from TCPListener - this duplicates the FD
		file, err := l.File()
		if err != nil {
			return nil, fmt.Errorf("failed to get file from TCPListener: %w", err)
		}
		// Note: TCPListener.File() already duplicates the FD
		return file, nil

	case interface{ File() (*os.File, error) }:
		// Generic interface for listeners with File() method
		file, err := l.File()
		if err != nil {
			return nil, fmt.Errorf("failed to get file from listener: %w", err)
		}
		return file, nil

	default:
		return nil, fmt.Errorf("unsupported listener type: %T", listener)
	}
}

// getExtraFiles retrieves file descriptors passed from parent process
func getExtraFiles() []*os.File {
	var files []*os.File

	// ExtraFiles start at FD 3 (after stdin, stdout, stderr)
	fdStart := 3

	// Check environment for FD count
	fdCountStr := os.Getenv("GOMUD_FD_COUNT")
	if fdCountStr == "" {
		return files
	}

	fdCount, err := strconv.Atoi(fdCountStr)
	if err != nil {
		mudlog.Error("Copyover", "error", "Invalid FD count", "value", fdCountStr)
		return files
	}

	// Create os.File for each FD
	for i := 0; i < fdCount; i++ {
		fd := fdStart + i
		file := os.NewFile(uintptr(fd), fmt.Sprintf("preserved-fd-%d", fd))
		if file != nil {
			// Set non-blocking mode
			syscall.SetNonblock(int(file.Fd()), true)
			files = append(files, file)
		}
	}

	mudlog.Info("Copyover", "fds", "Retrieved extra files", "count", len(files))
	return files
}

// CleanupConnections closes all preserved FDs after they've been processed
func CleanupConnections() {
	// Clear maps
	gatheredConnections = make(map[connections.ConnectionId]*connections.ConnectionDetails)
	gatheredListeners = make(map[string]net.Listener)
	connectionFds = make(map[string]*os.File)
	listenerFds = make(map[string]*os.File)
	preservedListeners = make(map[string]net.Listener)
	preservedConnections = make(map[connections.ConnectionId]net.Conn)
}
