// Package copyrod describes the data of the Copy Rod ALttP hack ("A Copy to the Past") that mudora-clipboard reads
// from a running game, and turns a stream of reads into discoveries.
//
// The hack gives every enemy type an item. Hitting an enemy with the rod "finds" it, and the player copies that
// item. The game keeps two tables in WRAM that this package decodes: which item each enemy type carries, and
// which enemy types have been found.
package copyrod

// Memory the tracker reads, in SNI's default (FX Pak Pro) address space, where WRAM $7E:0000 is 0xF50000.
// They were found by reading the hack's ROM.
const (
	AddrEnemyMap   = 0xF66518 // $7F:6518, 256 bytes: copied item id per sprite type (0 = none)
	AddrSpriteSeen = 0xF66628 // $7F:6628, 32 bytes: one bit per sprite type, MSB first, set when the rod hit that type
	EnemyMapSize   = 256
	SpriteSeenSize = 32

	// Item ids that can be assigned to enemies, in id order (which is roughly the game's item menu order).
	FirstItemID = 0x02
	LastItemID  = 0x1F

	// Item ids in the enemy table are 0x01..MaxItemID; anything above that means the bytes are not the table.
	MaxItemID = 0x27
)

// Snapshot is one read of the two Copy Rod tables.
type Snapshot struct {
	EnemyMap [EnemyMapSize]byte
	Seen     [SpriteSeenSize]byte
}

// IsSeen reports whether the rod has hit enemies of sprite type t.
// The handler sets the bit with the mask table at $00:98C1 (0x80, 0x40, ... 0x01): byte t>>3, MSB first.
func (s *Snapshot) IsSeen(t byte) bool {
	return s.Seen[t>>3]&(0x80>>(t&7)) != 0
}

// MarkSeen sets the found bit for sprite type t (used by tests).
func (s *Snapshot) MarkSeen(t byte) {
	s.Seen[t>>3] |= 0x80 >> (t & 7)
}

// Plausible reports whether the enemy table looks like real data. Before a save is loaded the WRAM holds
// whatever was there, and treating that as a table would invent discoveries. A real table assigns an item
// id of 1..MaxItemID to (nearly) every known enemy; random bytes almost never do.
func (s *Snapshot) Plausible() bool {
	good := 0
	for _, e := range Enemies {
		if id := s.EnemyMap[e.Type]; id >= 1 && id <= MaxItemID {
			good++
		}
	}
	return good*100 >= len(Enemies)*75
}

// Discovery is an enemy type the rod has hit, and the item that enemy carries.
type Discovery struct {
	Enemy Enemy
	Item  byte
}

// Discovered returns every known enemy that has been seen, in table order.
func (s *Snapshot) Discovered() []Discovery {
	var out []Discovery
	for _, e := range Enemies {
		if s.IsSeen(e.Type) {
			out = append(out, Discovery{e, s.EnemyMap[e.Type]})
		}
	}
	return out
}

// Progress returns how many of the known enemies have been found.
func Progress(s *Snapshot) (found, total int) {
	for _, e := range Enemies {
		if s.IsSeen(e.Type) {
			found++
		}
	}
	return found, len(Enemies)
}
