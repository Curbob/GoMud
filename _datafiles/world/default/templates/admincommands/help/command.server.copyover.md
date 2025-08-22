# Server Copyover Command

Performs a hot-reload of the server with rebuild, preserving all player connections.

## Usage:

  ~server copyover now~  
  Immediate copyover with rebuild

  ~server copyover [seconds]~  
  Schedule copyover after countdown (e.g., ~server copyover 30~)

  ~server copyover cancel~  
  Cancel a scheduled copyover

  ~server copyover status~  
  Check if a copyover is scheduled

## What it does:

- Rebuilds the server executable with latest code changes
- Saves all game state (users, rooms, mobs, items)
- Restarts the server process
- Preserves all player connections (no disconnects!)
- Restores game state after restart

## Note:

Copyover is the recommended way to apply code updates without disrupting gameplay.