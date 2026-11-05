package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod/copyrodtest"
	"github.com/alttpr-mudora/mudora-clipboard/internal/icons"
)

// hubWith returns a hub that has seen one valid read in which the given enemy types are found.
func hubWith(t *testing.T, reveal bool, found ...byte) (*Hub, copyrod.Snapshot) {
	t.Helper()
	h := NewHub(reveal)
	h.now = func() time.Time { return time.Date(2026, 11, 5, 21, 4, 12, 0, time.UTC) }
	s := copyrodtest.ValidSnapshot()
	for _, f := range found {
		s.MarkSeen(f)
	}
	h.Update(&s, copyrod.Update{Ready: true, Reset: true})
	return h, s
}

func findEnemy(t *testing.T, st StateView, name string) EnemyView {
	t.Helper()
	for _, e := range st.Enemies {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("enemy %q not in state", name)
	return EnemyView{}
}

func TestHubHidesUnfoundItems(t *testing.T) {
	h, s := hubWith(t, false, 0x4B)
	st := h.State()
	if !st.Ready || st.Found != 1 || st.Total != 78 || len(st.Enemies) != 78 {
		t.Fatalf("unexpected state: ready=%v found=%d total=%d enemies=%d", st.Ready, st.Found, st.Total, len(st.Enemies))
	}
	knife := findEnemy(t, st, "Green Knife Guard")
	if !knife.Found || knife.ItemName != copyrod.ItemName(s.EnemyMap[0x4B]) {
		t.Errorf("found enemy should show its item, got %+v", knife)
	}
	raven := findEnemy(t, st, "Raven")
	if raven.Found || raven.Item != 0 || raven.ItemName != "" {
		t.Errorf("unfound enemy must not carry its item, got %+v", raven)
	}
	// nothing about unfound items may appear in the JSON either (dev tools must not spoil the seed)
	b, _ := json.Marshal(st)
	for _, e := range copyrod.Enemies {
		if e.Type == 0x4B {
			continue
		}
		// an unfound enemy may carry its name and its own sprite (neither is a spoiler), nothing about its item
		want := `"name":"` + e.Name + `","found":false}`
		if sprite := icons.Enemy(e.Type); sprite != "" {
			want = `"name":"` + e.Name + `","found":false,"sprite":"` + sprite + `"}`
		}
		if !strings.Contains(string(b), want) {
			t.Fatalf("JSON for %s leaks more than its name and sprite: %s", e.Name, b)
		}
	}
}

func TestHubReveal(t *testing.T) {
	h, _ := hubWith(t, true, 0x4B)
	st := h.State()
	if !st.Reveal {
		t.Error("state should report reveal mode")
	}
	if r := findEnemy(t, st, "Raven"); r.Found || r.ItemName == "" {
		t.Errorf("reveal mode shows the item of an unfound enemy, got %+v", r)
	}
}

func TestHubRecentAndSilence(t *testing.T) {
	h, s := hubWith(t, false)
	v0 := h.State().Version

	// an identical read changes nothing and wakes no one
	ch, cancel := h.Subscribe()
	defer cancel()
	h.Update(&s, copyrod.Update{Ready: true})
	if h.State().Version != v0 {
		t.Error("an unchanged read must not bump the version")
	}
	select {
	case <-ch:
		t.Error("an unchanged read must not wake subscribers")
	default:
	}

	// a discovery does, and lands at the top of the recent list
	s.MarkSeen(0x08)
	h.Update(&s, copyrod.Update{Ready: true, New: []copyrod.Discovery{{Enemy: copyrod.Enemies[2], Item: s.EnemyMap[0x08]}}})
	select {
	case <-ch:
	default:
		t.Error("a discovery must wake subscribers")
	}
	st := h.State()
	if st.Version != v0+1 || len(st.Recent) != 1 || st.Recent[0].Name != "Octorok" || st.Recent[0].At != "21:04:12" {
		t.Errorf("unexpected state after a discovery: %+v", st)
	}

	// the list is capped, newest first
	for i := 0; i < maxRecent+3; i++ {
		s.MarkSeen(copyrod.Enemies[3+i].Type)
		h.Update(&s, copyrod.Update{Ready: true, New: []copyrod.Discovery{{Enemy: copyrod.Enemies[3+i], Item: 2}}})
	}
	st = h.State()
	if len(st.Recent) != maxRecent || st.Recent[0].Name != copyrod.Enemies[3+maxRecent+2].Name {
		t.Errorf("recent list should be capped to %d, newest first: %+v", maxRecent, st.Recent)
	}

	// a reset (new file) clears it
	h.Update(&s, copyrod.Update{Ready: true, Reset: true})
	if len(h.State().Recent) != 0 {
		t.Error("a reset must clear the recent list")
	}
}

func TestHubConnectionStates(t *testing.T) {
	h := NewHub(false)
	if st := h.State(); st.Ready || st.Device != "" || st.Notice == "" {
		t.Errorf("initial state: %+v", st)
	}
	h.Connected("fake (fake://1)")
	if st := h.State(); st.Device != "fake (fake://1)" || st.Ready {
		t.Errorf("connected: %+v", st)
	}
	h.Notice("waiting for the Copy Rod tables")
	if st := h.State(); st.Notice != "waiting for the Copy Rod tables" || st.Device == "" {
		t.Errorf("waiting: %+v", st)
	}
	h.Disconnected("cannot reach SNI")
	if st := h.State(); st.Device != "" || st.Ready || st.Notice != "cannot reach SNI" {
		t.Errorf("disconnected: %+v", st)
	}
}

func TestWebStateAndIndex(t *testing.T) {
	h, _ := hubWith(t, false, 0x4B)
	srv := httptest.NewServer(h.Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("state content type = %q", ct)
	}
	var st StateView
	if err := json.NewDecoder(resp.Body).Decode(&st); err != nil {
		t.Fatal(err)
	}
	if st.Found != 1 || !st.Ready {
		t.Errorf("decoded state: %+v", st)
	}

	for _, path := range []string{"/", "/app.js", "/style.css", "/mudora.png", "/favicon.ico", "/img/" + icons.Item(0x02), "/img/" + icons.Enemy(0x00)} {
		r, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(r.Body)
		r.Body.Close()
		if r.StatusCode != 200 || len(body) == 0 {
			t.Errorf("GET %s: status %d, %d bytes", path, r.StatusCode, len(body))
		}
		if r.Header.Get("Cache-Control") != "no-store" {
			t.Errorf("GET %s should not be cached", path)
		}
	}
}

func TestWebEventStream(t *testing.T) {
	h, s := hubWith(t, false)
	srv := httptest.NewServer(h.Handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", srv.URL+"/api/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content type = %q", ct)
	}

	events := make(chan StateView, 4)
	go func() {
		sc := bufio.NewScanner(resp.Body)
		sc.Buffer(make([]byte, 1<<20), 1<<20)
		for sc.Scan() {
			if data, ok := strings.CutPrefix(sc.Text(), "data: "); ok {
				var st StateView
				if json.Unmarshal([]byte(data), &st) == nil {
					events <- st
				}
			}
		}
	}()
	next := func() StateView {
		select {
		case st := <-events:
			return st
		case <-ctx.Done():
			t.Fatal("timed out waiting for an event")
			return StateView{}
		}
	}

	if first := next(); first.Found != 0 || !first.Ready {
		t.Errorf("the stream should start with the current state, got %+v", first)
	}
	s.MarkSeen(0x08)
	h.Update(&s, copyrod.Update{Ready: true, New: []copyrod.Discovery{{Enemy: copyrod.Enemies[2], Item: s.EnemyMap[0x08]}}})
	if second := next(); second.Found != 1 || len(second.Recent) != 1 || second.Recent[0].Name != "Octorok" {
		t.Errorf("a discovery should be pushed, got %+v", second)
	}
}

func TestStartWeb(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h := NewHub(false)
	h.Connected("fake (fake://1)")
	url, err := Start(ctx, "127.0.0.1:0", h)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(url + "/api/state")
	if err != nil {
		t.Fatal(err)
	}
	var st StateView
	err = json.NewDecoder(resp.Body).Decode(&st)
	resp.Body.Close()
	if err != nil || st.Device != "fake (fake://1)" {
		t.Fatalf("state = %+v, err = %v", st, err)
	}

	// cancelling the context stops the server
	cancel()
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := http.Get(url + "/api/state"); err != nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("the server kept serving after the context was cancelled")
}

func TestBuildItems(t *testing.T) {
	s := copyrodtest.ValidSnapshot()
	s.EnemyMap[0x4B] = 0x0F // Green Knife Guard -> Lamp
	s.EnemyMap[0x22] = 0x0F // Ropa -> Lamp
	s.EnemyMap[0x00] = 0x10 // Raven -> Hammer
	s.MarkSeen(0x4B)
	s.MarkSeen(0x00)

	items := buildItems(&s, false)
	if len(items) != copyrod.LastItemID-copyrod.FirstItemID+1 {
		t.Fatalf("got %d item slots, want %d", len(items), copyrod.LastItemID-copyrod.FirstItemID+1)
	}
	for i, it := range items {
		if it.Item != copyrod.FirstItemID+i {
			t.Fatalf("slot %d holds item %02X, want id order", i, it.Item)
		}
		if it.Enemies == nil {
			t.Errorf("%s: Enemies must be an empty list, not nil, so it encodes as []", it.Name)
		}
	}
	lamp, hammer := items[0x0F-copyrod.FirstItemID], items[0x10-copyrod.FirstItemID]
	if lamp.Name != "Lamp" || len(lamp.Enemies) != 1 || lamp.Enemies[0] != (ItemEnemy{Name: "Green Knife Guard", Found: true, Sprite: icons.Enemy(0x4B)}) {
		t.Errorf("Lamp should list only the found enemy, got %+v", lamp)
	}
	if hammer.Name != "Hammer" || len(hammer.Enemies) != 1 || hammer.Enemies[0].Name != "Raven" {
		t.Errorf("Hammer: %+v", hammer)
	}
	if lamp.Sprite != "items/lamp.png" {
		t.Errorf("Lamp sprite = %q", lamp.Sprite)
	}
	for _, it := range items {
		if it.Item != 0x0F && it.Item != 0x10 && len(it.Enemies) != 0 {
			// the filler table gives most enemies some item, but none of them has been found
			t.Errorf("%s lists enemies that were not found: %+v", it.Name, it.Enemies)
		}
	}

	// reveal lists every enemy, found ones marked
	rev := buildItems(&s, true)
	lampR := rev[0x0F-copyrod.FirstItemID]
	if len(lampR.Enemies) < 2 {
		t.Fatalf("reveal should list unfound enemies too, got %+v", lampR.Enemies)
	}
	for _, e := range lampR.Enemies {
		if e.Name == "Green Knife Guard" && !e.Found || e.Name == "Ropa" && e.Found {
			t.Errorf("found flags wrong in reveal mode: %+v", e)
		}
	}

	// an id outside the normal range gets its own slot at the end
	s.EnemyMap[0x08] = 0x25
	s.MarkSeen(0x08)
	extra := buildItems(&s, false)
	if len(extra) != len(items)+1 || extra[len(extra)-1].Item != 0x25 || extra[len(extra)-1].Enemies[0].Name != "Octorok" {
		t.Errorf("unknown id slot: %+v", extra[len(extra)-1])
	}
}

func TestStateJSONDoesNotLeakItems(t *testing.T) {
	h, s := hubWith(t, false, 0x4B)
	b, err := json.Marshal(h.State())
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	// the only enemy named anywhere under an item is the found one
	for _, e := range copyrod.Enemies {
		if e.Type == 0x4B {
			continue
		}
		if strings.Contains(js, `{"name":"`+e.Name+`","found":`) {
			t.Fatalf("%s appears under an item although it was not found: %s", e.Name, js)
		}
	}
	if !strings.Contains(js, `{"name":"Green Knife Guard","found":true,"sprite":"enemies/Green-Sword-Soldier-Sprite.png"}`) {
		t.Errorf("the found enemy should be listed under its item: %s", js)
	}
	if !strings.Contains(js, `"enemies":[]`) {
		t.Error("items without enemies should encode an empty list")
	}
	want := icons.Item(s.EnemyMap[0x4B])
	if !strings.Contains(js, `"sprite":"`+want+`"`) {
		t.Errorf("sprite %q missing from the state", want)
	}
}

// An enemy's own sprite is not a spoiler and is always sent; its item (and the item's sprite) only once found.
func TestEnemySpritesInState(t *testing.T) {
	h, s := hubWith(t, false, 0x4B)
	st := h.State()

	raven := findEnemy(t, st, "Raven")
	if raven.Found || raven.Sprite != "enemies/Crow.png" || raven.ItemSprite != "" || raven.ItemName != "" {
		t.Errorf("an unfound enemy shows its sprite but not its item: %+v", raven)
	}
	knife := findEnemy(t, st, "Green Knife Guard")
	if knife.Sprite != "enemies/Green-Sword-Soldier-Sprite.png" || knife.ItemSprite != icons.Item(s.EnemyMap[0x4B]) {
		t.Errorf("a found enemy shows both: %+v", knife)
	}

	// the same enemy under its item, and in the feed
	for _, it := range st.Items {
		for _, e := range it.Enemies {
			if e.Name == "Green Knife Guard" && e.Sprite != knife.Sprite {
				t.Errorf("sprite under the item = %q", e.Sprite)
			}
		}
	}
	h.Update(&s, copyrod.Update{Ready: true, New: []copyrod.Discovery{{Enemy: copyrod.Enemies[2], Item: s.EnemyMap[0x08]}}})
	if r := h.State().Recent; len(r) != 1 || r[0].EnemySprite != "enemies/Octorok-Sprite-3.png" || r[0].Sprite != icons.Item(s.EnemyMap[0x08]) {
		t.Errorf("feed entry: %+v", r)
	}
}
