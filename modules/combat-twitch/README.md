# Combat-Twitch Module

Action-based combat system with cooldowns/balance mechanics for GoMud.

## Configuration

This module uses the main game configuration for timing:
- Base round duration is derived from `Timing.RoundSeconds` in the main config
- Combat style selection is controlled by `GamePlay.CombatStyle`

No module-specific configuration is required.

## Features

- Action-based combat with balance/unbalance mechanics
- Weapon speed affects balance recovery time
- Speed stat reduces balance recovery time (up to 50%)
- Real-time GMCP balance updates
- "You are balanced" notifications

## Commands

- `attack <target>` - Attack a target (requires balance)
- `flee` - Attempt to escape from combat
- `consider <target>` - Evaluate a potential opponent

## Balance Mechanics

- Base balance recovery: 2 seconds for unarmed, varies by weapon
- Weapon wait rounds add 4 seconds per round to recovery time
- Speed stat can reduce recovery time by up to 50%
- GMCP updates sent every 100ms during unbalance