package combatrounds

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
)

func createTestCharacter(name string) *characters.Character {
	char := &characters.Character{
		Name: name,
	}
	// Set basic stats
	char.Stats.Strength.Value = 10
	char.Stats.Speed.Value = 10
	char.Stats.Smarts.Value = 10
	char.Stats.Vitality.Value = 10
	char.Stats.Mysticism.Value = 10
	char.Stats.Perception.Value = 10
	char.Level = 1
	
	return char
}

func TestCalculateHitChance(t *testing.T) {
	calc := &RoundCalculator{}
	
	tests := []struct {
		name           string
		attackerLevel  int
		attackerSpeed  int
		defenderLevel  int
		defenderSpeed  int
		expectHit      bool
		minChance      float64
	}{
		{
			name:          "Equal level and speed",
			attackerLevel: 10,
			attackerSpeed: 15,
			defenderLevel: 10,
			defenderSpeed: 15,
			expectHit:     true, // Should have decent chance
			minChance:     0.4,
		},
		{
			name:          "Higher level attacker",
			attackerLevel: 15,
			attackerSpeed: 15,
			defenderLevel: 10,
			defenderSpeed: 15,
			expectHit:     true,
			minChance:     0.6,
		},
		{
			name:          "Lower level attacker",
			attackerLevel: 5,
			attackerSpeed: 15,
			defenderLevel: 10,
			defenderSpeed: 15,
			expectHit:     true, // Still possible
			minChance:     0.2,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attacker := createTestCharacter("Attacker")
			defender := createTestCharacter("Defender")
			
			attacker.Level = tt.attackerLevel
			attacker.Stats.Speed.Value = tt.attackerSpeed
			defender.Level = tt.defenderLevel
			defender.Stats.Speed.Value = tt.defenderSpeed
			
			// Test hit chance calculation multiple times
			hits := 0
			attempts := 1000
			for i := 0; i < attempts; i++ {
				if calc.CalculateHitChance(attacker, defender) {
					hits++
				}
			}
			
			hitRate := float64(hits) / float64(attempts)
			if hitRate < tt.minChance {
				t.Errorf("Hit rate too low: got %f, want at least %f", hitRate, tt.minChance)
			}
		})
	}
}

func TestCalculateDamage(t *testing.T) {
	calc := &RoundCalculator{}
	
	tests := []struct {
		name         string
		strength     int
		weaponDamage string
		minDamage    int
		maxDamage    int
	}{
		{
			name:         "Basic attack no weapon",
			strength:     10,
			weaponDamage: "",
			minDamage:    1,
			maxDamage:    15,
		},
		{
			name:         "High strength no weapon",
			strength:     20,
			weaponDamage: "",
			minDamage:    5,
			maxDamage:    25,
		},
		{
			name:         "With weapon",
			strength:     10,
			weaponDamage: "2d6",
			minDamage:    3,  // 2 + str bonus
			maxDamage:    20, // 12 + str bonus
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attacker := createTestCharacter("Attacker")
			defender := createTestCharacter("Defender")
			attacker.Stats.Strength.Value = tt.strength
			
			var weapon *items.Item
			if tt.weaponDamage != "" {
				weapon = &items.Item{
					Spec: items.ItemSpec{
						Damage: items.DiceRoll{
							DiceRoll: tt.weaponDamage,
						},
					},
				}
			}
			
			// Test damage calculation multiple times
			minSeen := 999999
			maxSeen := 0
			for i := 0; i < 100; i++ {
				damage := calc.CalculateDamage(attacker, defender, weapon)
				if damage < minSeen {
					minSeen = damage
				}
				if damage > maxSeen {
					maxSeen = damage
				}
			}
			
			if minSeen < tt.minDamage {
				t.Errorf("Damage too low: got min %d, want at least %d", minSeen, tt.minDamage)
			}
			if maxSeen > tt.maxDamage {
				t.Errorf("Damage too high: got max %d, want at most %d", maxSeen, tt.maxDamage)
			}
		})
	}
}

func TestCalculateCriticalChance(t *testing.T) {
	calc := &RoundCalculator{}
	
	tests := []struct {
		name      string
		speed     int
		level     int
		minCrits  int
		maxCrits  int
	}{
		{
			name:     "Low speed and level",
			speed:    5,
			level:    1,
			minCrits: 0,
			maxCrits: 100, // Up to 10% crit
		},
		{
			name:     "High speed and level",
			speed:    20,
			level:    20,
			minCrits: 50,
			maxCrits: 250, // Higher crit chance
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attacker := createTestCharacter("Attacker")
			defender := createTestCharacter("Defender")
			attacker.Stats.Speed.Value = tt.speed
			attacker.Level = tt.level
			
			// Test crit chance
			crits := 0
			attempts := 1000
			for i := 0; i < attempts; i++ {
				if calc.CalculateCriticalChance(attacker, defender) {
					crits++
				}
			}
			
			if crits < tt.minCrits {
				t.Errorf("Too few crits: got %d, want at least %d", crits, tt.minCrits)
			}
			if crits > tt.maxCrits {
				t.Errorf("Too many crits: got %d, want at most %d", crits, tt.maxCrits)
			}
		})
	}
}

func TestCalculateDefense(t *testing.T) {
	calc := &RoundCalculator{}
	
	tests := []struct {
		name          string
		vitality      int
		armorDefense  int
		expectDefense int
	}{
		{
			name:          "No armor low vitality",
			vitality:      5,
			armorDefense:  0,
			expectDefense: 5, // vitality only
		},
		{
			name:          "With armor",
			vitality:      10,
			armorDefense:  15,
			expectDefense: 25, // vitality + armor
		},
		{
			name:          "High vitality and armor",
			vitality:      20,
			armorDefense:  30,
			expectDefense: 50,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defender := createTestCharacter("Defender")
			defender.Stats.Vitality.Value = tt.vitality
			defender.Equipment.Armor = items.Item{
				Spec: items.ItemSpec{
					Defense: tt.armorDefense,
				},
			}
			
			defense := calc.CalculateDefense(defender)
			if defense != tt.expectDefense {
				t.Errorf("Wrong defense: got %d, want %d", defense, tt.expectDefense)
			}
		})
	}
}

func TestCalculateInitiative(t *testing.T) {
	calc := &RoundCalculator{}
	
	// Test that higher speed gives better initiative
	fastChar := createTestCharacter("Fast")
	fastChar.Stats.Speed.Value = 20
	
	slowChar := createTestCharacter("Slow")
	slowChar.Stats.Speed.Value = 5
	
	// Run multiple times to account for randomness
	fastWins := 0
	for i := 0; i < 100; i++ {
		fastInit := calc.CalculateInitiative(fastChar)
		slowInit := calc.CalculateInitiative(slowChar)
		if fastInit > slowInit {
			fastWins++
		}
	}
	
	// Fast character should win most of the time
	if fastWins < 70 {
		t.Errorf("Fast character not winning enough: %d/100", fastWins)
	}
}

func TestCalculateAttackCount(t *testing.T) {
	calc := &RoundCalculator{}
	
	tests := []struct {
		name         string
		level        int
		speed        int
		expectCount  int
	}{
		{
			name:        "Low level",
			level:       1,
			speed:       10,
			expectCount: 1,
		},
		{
			name:        "Mid level",
			level:       10,
			speed:       15,
			expectCount: 1, // Might get 2
		},
		{
			name:        "High level high speed",
			level:       20,
			speed:       25,
			expectCount: 2, // Should get multiple
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attacker := createTestCharacter("Attacker")
			defender := createTestCharacter("Defender")
			attacker.Level = tt.level
			attacker.Stats.Speed.Value = tt.speed
			
			count := calc.CalculateAttackCount(attacker, defender)
			if count < tt.expectCount {
				t.Errorf("Attack count too low: got %d, want at least %d", count, tt.expectCount)
			}
			if count > 5 { // Sanity check
				t.Errorf("Attack count too high: %d", count)
			}
		})
	}
}

func TestPowerRanking(t *testing.T) {
	calc := &RoundCalculator{}
	
	// Test relative power
	strongChar := createTestCharacter("Strong")
	strongChar.Level = 20
	strongChar.HealthMax.Value = 200
	strongChar.Stats.Strength.Value = 20
	strongChar.Stats.Speed.Value = 20
	
	weakChar := createTestCharacter("Weak")
	weakChar.Level = 1
	weakChar.HealthMax.Value = 20
	weakChar.Stats.Strength.Value = 5
	weakChar.Stats.Speed.Value = 5
	
	avgChar := createTestCharacter("Average")
	avgChar.Level = 10
	avgChar.HealthMax.Value = 100
	
	// Test power rankings
	strongVsWeak := calc.PowerRanking(strongChar, weakChar)
	weakVsStrong := calc.PowerRanking(weakChar, strongChar)
	avgVsAvg := calc.PowerRanking(avgChar, avgChar)
	
	// Strong should have high ratio vs weak
	if strongVsWeak < 2.0 {
		t.Errorf("Strong vs weak ratio too low: %f", strongVsWeak)
	}
	
	// Weak should have low ratio vs strong
	if weakVsStrong > 0.5 {
		t.Errorf("Weak vs strong ratio too high: %f", weakVsStrong)
	}
	
	// Equal characters should be near 1.0
	if avgVsAvg < 0.9 || avgVsAvg > 1.1 {
		t.Errorf("Equal power ranking not near 1.0: %f", avgVsAvg)
	}
}