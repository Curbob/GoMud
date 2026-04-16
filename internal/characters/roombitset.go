package characters

import (
	"fmt"
	"math/bits"
	"strconv"

	"gopkg.in/yaml.v2"
)

// RoomBitset is a memory-efficient set of visited room IDs using a chunked
// bitset. The map key is roomId/64 (the block index) and the value is a uint64
// where bit (roomId%64) represents that room. Blocks are only allocated for
// room ID ranges that have been visited, so sparse zones cost nothing for
// unvisited regions.
//
// YAML serialization uses hex strings ("0x...") so the data is human-readable
// in character save files.
type RoomBitset map[uint16]uint64

// Set marks a room as visited. Room IDs must be positive; non-positive IDs
// are silently ignored because they represent special sentinel values (e.g.
// -1 for the character-creation room, 0 for StartRoomIdAlias) that are
// always considered visited.
func (rb RoomBitset) Set(roomId int) {
	if roomId < 1 {
		return
	}
	block := uint16(roomId / 64)
	bit := uint64(1) << (roomId % 64)
	rb[block] |= bit
}

// Has reports whether a room has been visited. Non-positive room IDs always
// return true because they are sentinel values that are considered visited
// by definition.
func (rb RoomBitset) Has(roomId int) bool {
	if roomId < 1 {
		return true
	}
	block := uint16(roomId / 64)
	bit := uint64(1) << (roomId % 64)
	return rb[block]&bit != 0
}

// Count returns the total number of visited rooms across all blocks.
func (rb RoomBitset) Count() int {
	total := 0
	for _, word := range rb {
		total += bits.OnesCount64(word)
	}
	return total
}

// CountIn returns how many rooms from the provided set have been visited.
func (rb RoomBitset) CountIn(roomIds map[int]struct{}) int {
	count := 0
	for roomId := range roomIds {
		if rb.Has(roomId) {
			count++
		}
	}
	return count
}

// IsComplete reports whether every room in the provided set has been visited.
func (rb RoomBitset) IsComplete(roomIds map[int]struct{}) bool {
	return rb.CountIn(roomIds) == len(roomIds)
}

// Prune clears any bits that do not correspond to a live room in validRoomIds,
// then removes blocks that become empty. This handles the case where rooms are
// deleted from a zone after a player has already visited them.
func (rb RoomBitset) Prune(validRoomIds map[int]struct{}) {
	// Build a valid-bits mask per block from the live room set.
	validMasks := make(map[uint16]uint64, len(validRoomIds)/32+1)
	for roomId := range validRoomIds {
		block := uint16(roomId / 64)
		bit := uint64(1) << (roomId % 64)
		validMasks[block] |= bit
	}

	for block, word := range rb {
		masked := word & validMasks[block]
		if masked == 0 {
			delete(rb, block)
		} else {
			rb[block] = masked
		}
	}
}

// ToSet expands the bitset into a map[int]struct{} of all visited room IDs.
func (rb RoomBitset) ToSet() map[int]struct{} {
	out := make(map[int]struct{}, rb.Count())
	for block, word := range rb {
		base := int(block) * 64
		for bit := 0; bit < 64; bit++ {
			if word&(uint64(1)<<bit) != 0 {
				out[base+bit] = struct{}{}
			}
		}
	}
	return out
}

// MarshalYAML serialises the bitset as a map of block-index to hex string so
// that character save files remain human-readable.
//
// Example output:
//
//	0: "0x0000000000000003"
//	9: "0xFFFFFFFFFFFFFFFF"
func (rb RoomBitset) MarshalYAML() (interface{}, error) {
	out := make(map[string]string, len(rb))
	for block, word := range rb {
		out[strconv.Itoa(int(block))] = fmt.Sprintf("0x%016X", word)
	}
	return out, nil
}

// UnmarshalYAML deserialises the hex-string map produced by MarshalYAML.
func (rb *RoomBitset) UnmarshalYAML(unmarshal func(interface{}) error) error {
	raw := make(map[string]string)
	if err := unmarshal(&raw); err != nil {
		// Tolerate an empty / null node.
		*rb = make(RoomBitset)
		return nil
	}

	result := make(RoomBitset, len(raw))
	for blockStr, hexStr := range raw {
		block, err := strconv.ParseUint(blockStr, 10, 16)
		if err != nil {
			return fmt.Errorf("roombitset: invalid block key %q: %w", blockStr, err)
		}
		word, err := strconv.ParseUint(hexStr, 0, 64)
		if err != nil {
			return fmt.Errorf("roombitset: invalid word value %q: %w", hexStr, err)
		}
		result[uint16(block)] = word
	}
	*rb = result
	return nil
}

// Ensure RoomBitset satisfies the yaml interfaces at compile time.
var _ yaml.Marshaler = RoomBitset{}
var _ yaml.Unmarshaler = (*RoomBitset)(nil)
