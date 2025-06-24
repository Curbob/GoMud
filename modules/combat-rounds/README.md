# Combat-Rounds Module

Traditional round-based combat system for GoMud.

## Configuration

This module uses the main game configuration for timing:
- Round duration is controlled by `Timing.RoundSeconds` in the main config
- Combat style selection is controlled by `GamePlay.CombatStyle`

No module-specific configuration is required.

## Features

- Traditional MUD round-based combat
- Multiple attacks per round based on speed difference
- Flee mechanics with speed-based success chance
- Consider command for evaluating opponents
- GMCP round countdown display

## Commands

- `attack <target>` - Initiate combat with a target
- `flee` - Attempt to escape from combat
- `consider <target>` - Evaluate a potential opponent