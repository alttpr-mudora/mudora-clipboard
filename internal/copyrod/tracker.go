package copyrod

// Update is what changed after feeding the tracker a snapshot.
type Update struct {
	Ready bool        // the enemy table looks valid
	Reset bool        // the first game, or a different one than before: show the whole table again
	New   []Discovery // enemies found for the first time since the last update (only when !Reset)
	Snap  Snapshot    // what to show: the game's enemy table with every enemy found so far (only when Ready)
}

// Tracker turns a stream of snapshots into updates. It remembers the game it is tracking while the tables are not
// in memory (save and quit, the file select screen, a soft reset) and when the game's found bits are cleared, so
// nothing already found is forgotten. It starts over only when a different enemy table appears. The poll loop
// makes a new Tracker for every SNI connection, so losing SNI also starts over.
type Tracker struct {
	known bool                 // a game is being tracked
	table [EnemyMapSize]byte   // its enemy table
	found [SpriteSeenSize]byte // every found bit seen since tracking began (only ever grows)

	pending    [EnemyMapSize]byte // a different table seen once, adopted if the next valid read agrees
	hasPending bool
}

// sameTable reports whether a and b agree on every known enemy.
func sameTable(a, b *[EnemyMapSize]byte) bool {
	for _, e := range Enemies {
		if a[e.Type] != b[e.Type] {
			return false
		}
	}
	return true
}

func (t *Tracker) snapshot() Snapshot { return Snapshot{EnemyMap: t.table, Seen: t.found} }

// Update feeds the next snapshot to the tracker.
func (t *Tracker) Update(s Snapshot) Update {
	if !s.Plausible() {
		return Update{} // the game is not loaded right now; keep everything
	}

	if !t.known || !sameTable(&t.table, &s.EnemyMap) {
		// The first game, or a different table. A different table has to show up on two reads in a row before
		// it replaces the current game, so one glitched or half-copied read cannot wipe what was found.
		if t.known && !(t.hasPending && sameTable(&t.pending, &s.EnemyMap)) {
			t.pending, t.hasPending = s.EnemyMap, true
			return Update{}
		}
		t.known, t.hasPending = true, false
		t.table, t.found = s.EnemyMap, s.Seen
		return Update{Ready: true, Reset: true, Snap: t.snapshot()}
	}
	t.hasPending = false

	var fresh []Discovery
	for _, e := range Enemies {
		mask := byte(0x80 >> (e.Type & 7))
		if s.IsSeen(e.Type) && t.found[e.Type>>3]&mask == 0 {
			t.found[e.Type>>3] |= mask
			fresh = append(fresh, Discovery{e, t.table[e.Type]})
		}
	}
	return Update{Ready: true, New: fresh, Snap: t.snapshot()}
}
