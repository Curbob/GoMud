# Server Restart Command

Performs a hot-reload of the server without rebuild, preserving all player connections.

## Usage:

  ~server restart now~  
  Immediate restart without rebuild

  ~server restart [seconds]~  
  Schedule restart after countdown (e.g., ~server restart 30~)

  ~server restart cancel~  
  Cancel a scheduled restart

  ~server restart status~  
  Check if a restart is scheduled

## What it does:

- Uses existing server executable (no rebuild)
- Saves all game state (users, rooms, mobs, items)
- Restarts the server process
- Preserves all player connections (no disconnects!)
- Restores game state after restart

## Note:

Restart is faster than copyover since it doesn't rebuild. Use when code hasn't changed.