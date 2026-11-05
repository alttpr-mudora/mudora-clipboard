package copyrod_test

import (
	"strings"
	"testing"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod/copyrodtest"
)

func TestSeenBitOrder(t *testing.T) {
	cases := []struct {
		typ  byte
		idx  int
		mask byte
	}{
		{0x00, 0, 0x80}, // first bit of the first byte is MSB
		{0x07, 0, 0x01},
		{0x08, 1, 0x80},
		{0x4B, 9, 0x10}, // Green Knife Guard: the Lua test run reported SpriteSeen byte 9
		{0xFF, 31, 0x01},
	}
	for _, c := range cases {
		var s copyrod.Snapshot
		s.Seen[c.idx] = c.mask
		if !s.IsSeen(c.typ) {
			t.Errorf("type %02X: want seen with byte %d = %02X", c.typ, c.idx, c.mask)
		}
		// no other type may read as seen
		for other := 0; other < 256; other++ {
			if byte(other) != c.typ && s.IsSeen(byte(other)) {
				t.Errorf("type %02X: type %02X also reads as seen", c.typ, other)
			}
		}
	}
}

func TestPlausible(t *testing.T) {
	var zero copyrod.Snapshot
	if zero.Plausible() {
		t.Error("all-zero memory must not look like a table")
	}
	var junk copyrod.Snapshot
	for i := range junk.EnemyMap {
		junk.EnemyMap[i] = 0xAA
	}
	if junk.Plausible() {
		t.Error("0xAA junk must not look like a table")
	}
	s := copyrodtest.ValidSnapshot()
	if !s.Plausible() {
		t.Error("a full table must look valid")
	}
	// a seed may leave a few enemies without an item
	for _, e := range copyrod.Enemies[:5] {
		s.EnemyMap[e.Type] = 0
	}
	if !s.Plausible() {
		t.Error("a table with a few empty entries must still look valid")
	}
}

func TestTrackerFlow(t *testing.T) {
	var tr copyrod.Tracker

	// nothing valid in memory yet
	if u := tr.Update(copyrod.Snapshot{}); u.Ready || u.Reset || len(u.New) != 0 {
		t.Fatalf("empty memory: got %+v", u)
	}

	// the table appears: the whole table is reported once
	s := copyrodtest.ValidSnapshot()
	if u := tr.Update(s); !u.Ready || !u.Reset {
		t.Fatalf("first valid read: got %+v", u)
	}
	// unchanged
	if u := tr.Update(s); !u.Ready || u.Reset || len(u.New) != 0 {
		t.Fatalf("unchanged read: got %+v", u)
	}

	// two enemies are hit
	s.MarkSeen(0x4B)
	s.MarkSeen(0x08)
	u := tr.Update(s)
	if !u.Ready || u.Reset || len(u.New) != 2 {
		t.Fatalf("after two hits: got %+v", u)
	}
	if u.New[0].Enemy.Name != "Octorok" || u.New[1].Enemy.Name != "Green Knife Guard" {
		t.Errorf("discoveries should come in table order, got %q then %q", u.New[0].Enemy.Name, u.New[1].Enemy.Name)
	}
	if want := s.EnemyMap[0x08]; u.New[0].Item != want {
		t.Errorf("Octorok item = %02X, want %02X", u.New[0].Item, want)
	}
	if !u.Snap.IsSeen(0x4B) || !u.Snap.IsSeen(0x08) {
		t.Error("the update's snapshot should include the found enemies")
	}
	// the same hits again are not new
	if u := tr.Update(s); len(u.New) != 0 || u.Reset {
		t.Fatalf("repeat read: got %+v", u)
	}
}

// Save and quit (the tables vanish and come back) must not forget anything.
func TestTrackerSurvivesSaveAndQuit(t *testing.T) {
	var tr copyrod.Tracker
	s := copyrodtest.ValidSnapshot()
	tr.Update(s)
	s.MarkSeen(0x4B)
	tr.Update(s)

	// the file select screen: memory holds no table, for any number of reads
	for i := 0; i < 5; i++ {
		if u := tr.Update(copyrod.Snapshot{}); u.Ready || u.Reset || len(u.New) != 0 {
			t.Fatalf("gap read %d: got %+v", i, u)
		}
	}

	// the same game comes back, with found bits that are fewer (nothing saved yet) or none at all
	back := copyrodtest.ValidSnapshot()
	u := tr.Update(back)
	if !u.Ready || u.Reset {
		t.Fatalf("same game back: got %+v (a reset would clear the feed)", u)
	}
	if !u.Snap.IsSeen(0x4B) {
		t.Error("the enemy found before quitting must still be found")
	}
	if len(u.New) != 0 {
		t.Errorf("nothing is newly found: %+v", u.New)
	}

	// and it keeps counting from there
	back.MarkSeen(0x08)
	u = tr.Update(back)
	if len(u.New) != 1 || u.New[0].Enemy.Name != "Octorok" || !u.Snap.IsSeen(0x4B) || !u.Snap.IsSeen(0x08) {
		t.Fatalf("after coming back: got %+v", u)
	}
}

func TestTrackerNeverUnfinds(t *testing.T) {
	var tr copyrod.Tracker
	s := copyrodtest.ValidSnapshot()
	s.MarkSeen(0x4B)
	tr.Update(s)
	// the game clears its found bits (a new file, or quitting without saving)
	u := tr.Update(copyrodtest.ValidSnapshot())
	if u.Reset || !u.Snap.IsSeen(0x4B) {
		t.Fatalf("cleared bits must not remove a find: got %+v", u)
	}
}

func TestTrackerDifferentGameResets(t *testing.T) {
	var tr copyrod.Tracker
	s := copyrodtest.ValidSnapshot()
	s.MarkSeen(0x4B)
	tr.Update(s)

	other := copyrodtest.ValidSnapshot()
	other.EnemyMap[copyrod.Enemies[0].Type] ^= 0x01
	other.MarkSeen(0x00)

	// one read of a different table is not enough (it could be a glitched read)
	if u := tr.Update(other); u.Ready || u.Reset {
		t.Fatalf("first read of a different table: got %+v", u)
	}
	// the original table coming back in between cancels it
	if u := tr.Update(s); !u.Ready || u.Reset || !u.Snap.IsSeen(0x4B) {
		t.Fatalf("original table back: got %+v", u)
	}
	if u := tr.Update(other); u.Ready || u.Reset {
		t.Fatalf("different table again, first read: got %+v", u)
	}
	// two reads in a row: a different game, so everything starts over
	u := tr.Update(other)
	if !u.Ready || !u.Reset {
		t.Fatalf("second read of the different table: got %+v", u)
	}
	if u.Snap.IsSeen(0x4B) || !u.Snap.IsSeen(0x00) {
		t.Error("the new game must start from its own found bits, not the old game's")
	}
	if u.Snap.EnemyMap != other.EnemyMap {
		t.Error("the new game's table should be shown")
	}
}

func TestItemName(t *testing.T) {
	if got := copyrod.ItemName(0x10); got != "Hammer" {
		t.Errorf("ItemName(0x10) = %q", got)
	}
	if got := copyrod.ItemName(0x25); !strings.Contains(got, "0x25") {
		t.Errorf("unknown item should show its id, got %q", got)
	}
	if got := copyrod.ItemName(0); got != "(no item)" {
		t.Errorf("ItemName(0) = %q", got)
	}
}
