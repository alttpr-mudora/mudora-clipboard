package poll_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/alttpr-mudora/mudora-clipboard/internal/console"
	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod/copyrodtest"
	"github.com/alttpr-mudora/mudora-clipboard/internal/poll"
	"github.com/alttpr-mudora/mudora-clipboard/internal/sni/snitest"
	"github.com/alttpr-mudora/mudora-clipboard/internal/ui"
)

func enemyFound(st ui.StateView, name string) bool {
	for _, e := range st.Enemies {
		if e.Name == name {
			return e.Found
		}
	}
	return false
}

// TestSessionOnce runs the real session code in -once mode against the fake, checking the printed table.
func TestSessionOnce(t *testing.T) {
	f := &snitest.Fake{Devices: []string{"fake://one"}}
	s := copyrodtest.ValidSnapshot()
	s.MarkSeen(0x4B)
	f.Put(copyrod.AddrEnemyMap, s.EnemyMap[:])
	f.Put(copyrod.AddrSpriteSeen, s.Seen[:])

	c := snitest.Start(t, f)
	var out bytes.Buffer
	cfg := poll.Config{Once: true, Interval: time.Millisecond}
	if err := poll.RunSession(context.Background(), c, cfg, console.New(&out, false, false)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Green Knife Guard") || !strings.Contains(out.String(), "Enemies found: 1 / 78") {
		t.Errorf("unexpected output:\n%s", out.String())
	}

	// memory without a table: -once must fail instead of printing junk
	g := &snitest.Fake{Devices: []string{"fake://one"}}
	empty := snitest.Start(t, g)
	out.Reset()
	if err := poll.RunSession(context.Background(), empty, cfg, console.New(&out, false, false)); err != poll.ErrNotReady {
		t.Errorf("want poll.ErrNotReady, got %v", err)
	}
}

// TestSessionKeepsFindsAcrossSaveAndQuit plays the scenario that used to wipe the feed: an enemy is found,
// the game saves and quits (the tables vanish from memory), and the same game comes back.
func TestSessionKeepsFindsAcrossSaveAndQuit(t *testing.T) {
	f := &snitest.Fake{Devices: []string{"fake://one"}}
	game := copyrodtest.ValidSnapshot()
	putTables := func(s copyrod.Snapshot) {
		f.Put(copyrod.AddrEnemyMap, s.EnemyMap[:])
		f.Put(copyrod.AddrSpriteSeen, s.Seen[:])
	}
	putTables(game)

	hub := ui.NewHub(false)
	c := snitest.Start(t, f)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- poll.RunSession(ctx, c, poll.Config{Interval: 2 * time.Millisecond}, hub) }()
	defer func() {
		cancel()
		<-done
	}()

	waitFor := func(what string, cond func(ui.StateView) bool) ui.StateView {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if st := hub.State(); cond(st) {
				return st
			}
			time.Sleep(2 * time.Millisecond)
		}
		t.Fatalf("timed out waiting for: %s (state %+v)", what, hub.State())
		return ui.StateView{}
	}

	waitFor("tracking", func(st ui.StateView) bool { return st.Ready })

	// the rod hits a Green Knife Guard
	game.MarkSeen(0x4B)
	putTables(game)
	st := waitFor("the find", func(st ui.StateView) bool { return st.Found == 1 && len(st.Recent) == 1 })
	if st.Recent[0].Name != "Green Knife Guard" {
		t.Fatalf("feed: %+v", st.Recent)
	}

	// save and quit: the tables are gone
	putTables(copyrod.Snapshot{})
	st = waitFor("unloaded", func(st ui.StateView) bool { return !st.Ready })
	if st.Found != 1 || len(st.Recent) != 1 || !enemyFound(st, "Green Knife Guard") {
		t.Fatalf("nothing may be dropped while the game is unloaded: %+v", st)
	}

	// the same game is loaded again, with no found bits yet
	putTables(copyrodtest.ValidSnapshot())
	st = waitFor("tracking again", func(st ui.StateView) bool { return st.Ready })
	if st.Found != 1 || len(st.Recent) != 1 || st.Recent[0].Name != "Green Knife Guard" {
		t.Fatalf("the feed and the find must survive save and quit: %+v", st)
	}
}
