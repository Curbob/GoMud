package events

import (
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/stats"
)

// EVENT DEFINITIONS FOLLOW
// NOTE: If you give an event the following receiver function: `UniqueID() string`
//
//	      It will become a "unique event", meaning only one can be in the event queue
//			 at a time matching the string return value.
//		 Example: See `RedrawPrompt`
//
// Used to apply or remove buffs
type Buff struct {
	UserId        int
	MobInstanceId int
	BuffId        int
	Source        string // optional source such as spell,
}

func (b Buff) Type() string { return `Buff` }

type BuffsTriggered struct {
	UserId        int
	MobInstanceId int
	BuffIds       []int
}

func (b BuffsTriggered) Type() string { return `BuffsTriggered` }

// Used for giving/taking quest progress
type Quest struct {
	UserId     int
	QuestToken string
}

func (q Quest) Type() string { return `Quest` }

// For special room-targetting actions
type RoomAction struct {
	RoomId       int
	SourceUserId int
	SourceMobId  int
	Action       string
	Details      any
	ReadyTurn    uint64
}

func (r RoomAction) Type() string { return `RoomAction` }

// Used for Input from players/mobs
type Input struct {
	UserId        int
	MobInstanceId int
	InputText     string
	ReadyTurn     uint64
	Flags         EventFlag
}

func (i Input) Type() string { return `Input` }

// When a skill is used by a player
type SkillUsed struct {
	UserId  int
	Skill   skills.SkillTag
	Details string // usually the specific sub-command of the skill
}

func (i SkillUsed) Type() string { return `SkillUsed` }

// Messages that are intended to reach all users on the system
type Broadcast struct {
	Text             string
	TextScreenReader string // optional text for screenreader friendliness
	IsCommunication  bool
	SourceIsMod      bool
	SkipLineRefresh  bool
}

func (b Broadcast) Type() string { return `Broadcast` }

type Message struct {
	UserId          int
	ExcludeUserIds  []int
	RoomId          int
	Text            string
	IsQuiet         bool // whether it can only be heard by superior "hearing"
	IsCommunication bool // If true, this is a communication such as "say" or "emote"
}

func (m Message) Type() string { return `Message` }

type Communication struct {
	SourceUserId        int    // User that sent the message
	SourceMobInstanceId int    // Mob that sent the message
	TargetUserId        int    // Sent to only 1 person
	CommType            string // say, party, broadcast, whisper, shout
	Name                string
	Message             string
}

func (m Communication) Type() string { return `Communication` }

// Special commands that only the webclient is equipped to handle
type WebClientCommand struct {
	ConnectionId uint64
	Text         string
}

func (w WebClientCommand) Type() string { return `WebClientCommand` }

// Messages that are intended to reach all users on the system
type System struct {
	Command     string
	Data        any
	Description string
}

func (s System) Type() string { return `System` }

// Payloads describing sound/music to play
type MSP struct {
	UserId    int
	SoundType string // SOUND or MUSIC
	SoundFile string
	Volume    int    // 1-100
	Category  string // special category/type for MSP string
}

func (m MSP) Type() string { return `MSP` }

// Fired whenever a mob or player changes rooms
type RoomChange struct {
	UserId        int
	MobInstanceId int
	FromRoomId    int
	ToRoomId      int
	Unseen        bool
}

func (r RoomChange) Type() string { return `RoomChange` }

// Fired every new round
type NewRound struct {
	RoundNumber uint64
	TimeNow     time.Time
}

func (n NewRound) Type() string { return `NewRound` }

// Each new turn (TurnMs in config.yaml)
type NewTurn struct {
	TurnNumber uint64
	TimeNow    time.Time
}

func (n NewTurn) Type() string { return `NewTurn` }

// Anytime a mob is idle
type MobIdle struct {
	MobInstanceId int
}

func (i MobIdle) Type() string { return `MobIdle` }

// Gained or lost an item
type EquipmentChange struct {
	UserId        int
	MobInstanceId int
	GoldChange    int
	BankChange    int
	ItemsWorn     []items.Item
	ItemsRemoved  []items.Item
}

func (i EquipmentChange) Type() string { return `EquipmentChange` }

// Gained or lost an item
type ItemOwnership struct {
	UserId        int
	MobInstanceId int
	Item          items.Item
	Gained        bool
}

func (i ItemOwnership) Type() string { return `ItemOwnership` }

// Triggered by a script
type ScriptedEvent struct {
	Name string
	Data map[string]any
}

func (s ScriptedEvent) Type() string { return `ScriptedEvent` }

// Entered the world
type PlayerSpawn struct {
	UserId        int
	ConnectionId  uint64
	RoomId        int
	Username      string
	CharacterName string
}

func (p PlayerSpawn) Type() string { return `PlayerSpawn` }

// Left the world
type PlayerDespawn struct {
	UserId        int
	RoomId        int
	Username      string
	CharacterName string
	TimeOnline    string
}

func (p PlayerDespawn) Type() string { return `PlayerDespawn` }

type Log struct {
	FollowAdd    connections.ConnectionId
	FollowRemove connections.ConnectionId
	Level        string
	Data         []any
}

func (l Log) Type() string { return `Log` }

type GainExperience struct {
	UserId     int
	Experience int
	Scale      int
}

func (l GainExperience) Type() string { return `GainExperience` }

type LevelUp struct {
	UserId         int
	RoomId         int
	Username       string
	CharacterName  string
	LevelsGained   int
	NewLevel       int
	StatsDelta     stats.Statistics
	TrainingPoints int
	StatPoints     int
	LivesGained    int
}

func (l LevelUp) Type() string { return `LevelUp` }

type PlayerDrop struct {
	UserId int
	RoomId int
}

func (l PlayerDrop) Type() string { return `PlayerDrop` }

type PlayerDeath struct {
	UserId        int
	RoomId        int
	Username      string
	CharacterName string
	Permanent     bool
	KilledByUsers []int
}

func (l PlayerDeath) Type() string { return `PlayerDeath` }

type MobDeath struct {
	MobId         int
	InstanceId    int
	RoomId        int
	CharacterName string
	Level         int
	PlayerDamage  map[int]int
}

func (l MobDeath) Type() string { return `MobDeath` }

type DayNightCycle struct {
	IsSunrise bool
	Day       int
	Month     int
	Year      int
	Time      string
}

func (l DayNightCycle) Type() string { return `DayNightCycle` }

type Looking struct {
	UserId int
	RoomId int
	Target string
	Hidden bool
}

func (l Looking) Type() string { return `Looking` }

// Fired after creating a new character and giving the character a name.
type CharacterCreated struct {
	UserId        int
	CharacterName string
}

func (p CharacterCreated) Type() string { return `CharacterCreated` }

// Fired when a character alt change has occured.
type CharacterChanged struct {
	UserId            int
	LastCharacterName string
	CharacterName     string
}

func (p CharacterChanged) Type() string { return `CharacterChanged` }

type UserSettingChanged struct {
	UserId int
	Name   string
}

func (i UserSettingChanged) Type() string { return `UserSettingChanged` }

// Health, mana, etc.
type CharacterVitalsChanged struct {
	UserId int
}

func (p CharacterVitalsChanged) Type() string { return `CharacterVitalsChanged` }

// Health, mana, etc.
type CharacterTrained struct {
	UserId int
}

func (p CharacterTrained) Type() string { return `CharacterTrained` }

// any stats or healthmax etc. have changed
type CharacterStatsChanged struct {
	UserId int
}

func (p CharacterStatsChanged) Type() string { return `CharacterStatsChanged` }

// any stats or healthmax etc. have changed
type PartyUpdated struct {
	Action  string // create, disband, membership
	UserIds []int
}

func (p PartyUpdated) Type() string { return `PartyUpdated` }

type Party struct {
	LeaderUserId  int
	UserIds       []int
	InviteUserIds []int
	AutoAttackers []int
	Position      map[int]string
}

func (p Party) Type() string { return `Party` }

// Rebuilds mapper for a given RoomId
// NOTE: RoomId should USUALLY be the Room's Zone.RootRoomId
type RebuildMap struct {
	MapRootRoomId int
	SkipIfExists  bool
}

func (r RebuildMap) Type() string { return `RebuildMap` }
func (r RebuildMap) UniqueID() string {
	return `RebuildMap-` + strconv.Itoa(r.MapRootRoomId) + `-` + strconv.FormatBool(r.SkipIfExists)
}

type RedrawPrompt struct {
	UserId        int
	OnlyIfChanged bool
}

func (l RedrawPrompt) Type() string     { return `RedrawPrompt` }
func (l RedrawPrompt) UniqueID() string { return `RedrawPrompt-` + strconv.Itoa(l.UserId) }

// Combat State Management Events

// CombatStarted fires when combat begins between entities
type CombatStarted struct {
	AttackerId   int
	AttackerType string // "player" or "mob"
	AttackerName string
	DefenderId   int
	DefenderType string // "player" or "mob"
	DefenderName string
	RoomId       int
	InitiatedBy  string // command/action that started combat
}

func (c CombatStarted) Type() string { return `CombatStarted` }

// CombatEnded fires when combat ends (not due to death)
type CombatEnded struct {
	EntityId   int
	EntityType string // "player" or "mob"
	EntityName string
	Reason     string // "fled", "broke", "peace", "distance", etc.
	RoomId     int
	Duration   int // Combat duration in seconds
}

func (c CombatEnded) Type() string { return `CombatEnded` }

// Damage and Healing Tracking Events

// DamageDealt fires immediately after damage calculation and application
type DamageDealt struct {
	SourceId      int
	SourceType    string // "player" or "mob"
	SourceName    string
	TargetId      int
	TargetType    string // "player" or "mob"
	TargetName    string
	Amount        int
	DamageType    string // "physical", "magical", "fire", etc.
	WeaponName    string // Name of weapon used (if applicable)
	SpellName     string // Name of spell used (if applicable)
	IsCritical    bool
	IsKillingBlow bool
}

func (d DamageDealt) Type() string { return `DamageDealt` }

// HealingReceived fires whenever health is restored
type HealingReceived struct {
	SourceId   int
	SourceType string // "player", "mob", "item", "regen"
	SourceName string
	TargetId   int
	TargetType string // "player" or "mob"
	TargetName string
	Amount     int
	HealType   string // "spell", "potion", "regen", "item", etc.
	SpellName  string // Name of healing spell (if applicable)
	IsOverheal bool
}

func (h HealingReceived) Type() string { return `HealingReceived` }

// Target and Aggro Management Events

// TargetChanged fires when a combatant's primary target changes
type TargetChanged struct {
	EntityId      int
	EntityType    string // "player" or "mob"
	EntityName    string
	OldTargetId   int
	OldTargetType string
	OldTargetName string
	NewTargetId   int
	NewTargetType string
	NewTargetName string
	Reason        string // "manual", "death", "fled", "taunt", etc.
}

func (t TargetChanged) Type() string { return `TargetChanged` }

// AggroGained fires when a mob becomes hostile to someone
type AggroGained struct {
	MobId       int
	MobName     string
	TargetId    int
	TargetType  string // "player" or "mob"
	TargetName  string
	IsInitial   bool // True if this is the first aggro
	ThreatLevel int
}

func (a AggroGained) Type() string { return `AggroGained` }

// AggroLost fires when a mob stops being hostile
type AggroLost struct {
	MobId      int
	MobName    string
	TargetId   int
	TargetType string // "player" or "mob"
	TargetName string
	Reason     string // "death", "distance", "reset", "peace", etc.
}

func (a AggroLost) Type() string { return `AggroLost` }

// Mob-Specific Events

// MobVitalsChanged fires whenever a mob's health or mana changes
type MobVitalsChanged struct {
	MobId      int
	OldHealth  int
	NewHealth  int
	OldMana    int
	NewMana    int
	ChangeType string // "damage", "heal", "regen", etc.
}

func (m MobVitalsChanged) Type() string { return `MobVitalsChanged` }

// MobStatusChanged fires when status effects are applied/removed from mobs
type MobStatusChanged struct {
	MobId    int
	Status   string // "stunned", "blinded", "slowed", etc.
	Added    bool   // True if added, false if removed
	Duration int    // Duration in seconds (0 for permanent)
	SourceId int    // Who applied the status
}

func (m MobStatusChanged) Type() string { return `MobStatusChanged` }

// Combat Action Events

// CombatActionStarted fires when combat actions begin (casting, channeling, etc.)
type CombatActionStarted struct {
	EntityId      int
	EntityType    string // "player" or "mob"
	EntityName    string
	Action        string // "spell", "ability", "item", etc.
	ActionName    string
	TargetId      int
	TargetName    string
	CastTime      float64 // Time to complete in seconds
	Interruptible bool
}

func (c CombatActionStarted) Type() string { return `CombatActionStarted` }

// CombatActionCompleted fires when an action finishes (successfully or not)
type CombatActionCompleted struct {
	EntityId      int
	EntityType    string // "player" or "mob"
	EntityName    string
	Action        string
	ActionName    string
	Success       bool
	FailureReason string
}

func (c CombatActionCompleted) Type() string { return `CombatActionCompleted` }

// CombatActionInterrupted fires when an action is interrupted before completion
type CombatActionInterrupted struct {
	EntityId          int
	EntityType        string // "player" or "mob"
	EntityName        string
	Action            string
	ActionName        string
	InterruptedById   int
	InterruptedByType string // Type of interrupter
	InterruptedByName string
	InterruptType     string // "damage", "stun", "silence", etc.
}

func (c CombatActionInterrupted) Type() string { return `CombatActionInterrupted` }

// Defense and Avoidance Events

// AttackAvoided fires when an attack fails to connect
type AttackAvoided struct {
	AttackerId   int
	AttackerType string // "player" or "mob"
	AttackerName string
	DefenderId   int
	DefenderType string // "player" or "mob"
	DefenderName string
	AvoidType    string // "miss", "dodge", "parry", "block"
	WeaponName   string
}

func (a AttackAvoided) Type() string { return `AttackAvoided` }

// Special Combat Events

// CombatEffectTriggered fires when DoTs, bleeds, or other effects tick
type CombatEffectTriggered struct {
	SourceId       int
	SourceName     string
	TargetId       int
	TargetType     string // "player" or "mob"
	TargetName     string
	Effect         string // "bleed", "poison", "burn", etc.
	Damage         int
	TicksRemaining int
}

func (c CombatEffectTriggered) Type() string { return `CombatEffectTriggered` }

// CombatantFled fires when someone attempts to flee
type CombatantFled struct {
	EntityId    int
	EntityType  string // "player" or "mob"
	EntityName  string
	Direction   string
	Success     bool
	PreventedBy string // What prevented it (if failed)
}

func (c CombatantFled) Type() string { return `CombatantFled` }

// Room Events

// ExitLockChanged fires when an exit's lock state changes
type ExitLockChanged struct {
	Event
	RoomId   int
	ExitName string
	Locked   bool // true if now locked, false if now unlocked
}

func (e ExitLockChanged) Type() string { return `ExitLockChanged` }
