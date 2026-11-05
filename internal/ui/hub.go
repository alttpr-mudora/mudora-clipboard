package ui

import (
	"reflect"
	"sync"
	"time"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	"github.com/alttpr-mudora/mudora-clipboard/internal/icons"
)

// maxRecent is how many "just found" entries the UI keeps.
const maxRecent = 8

// EnemyView is one enemy as the web UI sees it. The item of an enemy that has not been found is left out
// unless reveal mode is on, so the page cannot spoil the seed (even through the browser's dev tools).
type EnemyView struct {
	Type     int    `json:"type"`
	Name     string `json:"name"`
	Found    bool   `json:"found"`
	Item     int    `json:"item,omitempty"`
	ItemName string `json:"itemName,omitempty"`
	// Sprite is the enemy's own sprite (a file under /img/enemies/). It is not a spoiler, so it is sent for every
	// enemy. ItemSprite is the item's sprite (under /img/); like the item it is left out until the enemy is found.
	Sprite     string `json:"sprite,omitempty"`
	ItemSprite string `json:"itemSprite,omitempty"`
}

// RecentView is one entry of the "just found" list, newest first.
type RecentView struct {
	Name        string `json:"name"`
	EnemySprite string `json:"enemySprite,omitempty"`
	ItemName    string `json:"itemName"`
	Sprite      string `json:"sprite,omitempty"` // the item's sprite
	At          string `json:"at"`
}

// ItemEnemy is an enemy listed under an item.
type ItemEnemy struct {
	Name   string `json:"name"`
	Found  bool   `json:"found"`
	Sprite string `json:"sprite,omitempty"` // the enemy's sprite
}

// ItemView is one slot of the item list: an item and the enemies known to carry it. Without reveal mode only
// enemies the rod has hit are listed, so an item nobody has been found with has an empty list.
type ItemView struct {
	Item    int         `json:"item"`
	Name    string      `json:"name"`
	Sprite  string      `json:"sprite,omitempty"`
	Enemies []ItemEnemy `json:"enemies"`
}

// StateView is everything the web UI renders.
type StateView struct {
	Version int          `json:"version"`
	Device  string       `json:"device"` // empty when SNI or the device cannot be reached
	Ready   bool         `json:"ready"`  // the Copy Rod tables were read
	Notice  string       `json:"notice"` // what is wrong or being waited for; empty while tracking
	Reveal  bool         `json:"reveal"`
	Found   int          `json:"found"`
	Total   int          `json:"total"`
	Enemies []EnemyView  `json:"enemies"`
	Items   []ItemView   `json:"items"` // every possible item in id order, each with the enemies known to carry it
	Recent  []RecentView `json:"recent"`
}

// buildEnemies lists every known enemy; snap may be nil (nothing read yet).
func buildEnemies(snap *copyrod.Snapshot, reveal bool) []EnemyView {
	out := make([]EnemyView, 0, len(copyrod.Enemies))
	for _, e := range copyrod.Enemies {
		v := EnemyView{Type: int(e.Type), Name: e.Name, Sprite: icons.Enemy(e.Type)}
		if snap != nil {
			v.Found = snap.IsSeen(e.Type)
			if v.Found || reveal {
				id := snap.EnemyMap[e.Type]
				v.Item = int(id)
				v.ItemName = copyrod.ItemName(id)
				v.ItemSprite = icons.Item(id)
			}
		}
		out = append(out, v)
	}
	return out
}

// buildItems lists every item that can be assigned to an enemy (ids copyrod.FirstItemID..copyrod.LastItemID) in id order, each
// with the enemies known to carry it, in table order. An id outside that range that shows up on a visible
// enemy gets a slot of its own at the end. snap may be nil (nothing read yet).
func buildItems(snap *copyrod.Snapshot, reveal bool) []ItemView {
	byID := map[byte]*ItemView{}
	var order []byte
	slot := func(id byte) *ItemView {
		if v, ok := byID[id]; ok {
			return v
		}
		v := &ItemView{Item: int(id), Name: copyrod.ItemName(id), Sprite: icons.Item(id), Enemies: []ItemEnemy{}}
		byID[id] = v
		order = append(order, id)
		return v
	}
	for id := byte(copyrod.FirstItemID); id <= copyrod.LastItemID; id++ {
		slot(id)
	}
	if snap != nil {
		for _, e := range copyrod.Enemies {
			found := snap.IsSeen(e.Type)
			if !found && !reveal {
				continue
			}
			v := slot(snap.EnemyMap[e.Type])
			v.Enemies = append(v.Enemies, ItemEnemy{Name: e.Name, Found: found, Sprite: icons.Enemy(e.Type)})
		}
	}
	out := make([]ItemView, 0, len(order))
	for _, id := range order {
		out = append(out, *byID[id])
	}
	return out
}

// Hub holds the latest state for the web UI and wakes the event streams when it changes.
// It is a Sink.
type Hub struct {
	mu     sync.Mutex
	reveal bool
	state  StateView
	subs   map[chan struct{}]struct{}
	now    func() time.Time
}

// NewHub returns a hub in the "not connected yet" state.
func NewHub(reveal bool) *Hub {
	return &Hub{
		reveal: reveal,
		subs:   map[chan struct{}]struct{}{},
		now:    time.Now,
		state: StateView{
			Reveal:  reveal,
			Notice:  "starting",
			Total:   len(copyrod.Enemies),
			Enemies: buildEnemies(nil, reveal),
			Items:   buildItems(nil, reveal),
		},
	}
}

// State returns a copy of the current state. Its slices are never modified in place, so it is safe to
// encode while the hub keeps changing.
func (h *Hub) State() StateView {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.state
}

// Subscribe returns a channel that receives a signal after every state change (signals coalesce), and a
// function that ends the subscription.
func (h *Hub) Subscribe() (<-chan struct{}, func()) {
	ch := make(chan struct{}, 1)
	h.mu.Lock()
	h.subs[ch] = struct{}{}
	h.mu.Unlock()
	return ch, func() {
		h.mu.Lock()
		delete(h.subs, ch)
		h.mu.Unlock()
	}
}

// mutate applies f to a copy of the state. f must replace slices rather than modify them. Nothing happens
// (and no one is woken) if the result equals the old state, so polling an unchanged game stays silent.
func (h *Hub) mutate(f func(s *StateView)) {
	h.mu.Lock()
	next := h.state
	f(&next)
	next.Version = h.state.Version
	if reflect.DeepEqual(next, h.state) {
		h.mu.Unlock()
		return
	}
	next.Version++
	h.state = next
	subs := make([]chan struct{}, 0, len(h.subs))
	for c := range h.subs {
		subs = append(subs, c)
	}
	h.mu.Unlock()

	for _, c := range subs {
		select {
		case c <- struct{}{}:
		default: // a wake-up is already pending
		}
	}
}

// Connected implements Sink.
func (h *Hub) Connected(device string) {
	h.mutate(func(s *StateView) {
		s.Device = device
		s.Ready = false
		s.Notice = "connected, reading memory"
	})
}

// Notice implements Sink.
func (h *Hub) Notice(msg string) {
	h.mutate(func(s *StateView) {
		s.Ready = false
		s.Notice = msg
	})
}

// Disconnected implements Sink.
func (h *Hub) Disconnected(msg string) {
	h.mutate(func(s *StateView) {
		s.Device = ""
		s.Ready = false
		s.Notice = msg
	})
}

// Update implements Sink.
func (h *Hub) Update(snap *copyrod.Snapshot, u copyrod.Update) {
	enemies := buildEnemies(snap, h.reveal)
	items := buildItems(snap, h.reveal)
	found, total := copyrod.Progress(snap)
	at := h.now().Format("15:04:05")
	h.mutate(func(s *StateView) {
		s.Ready = true
		s.Notice = ""
		s.Enemies = enemies
		s.Items = items
		s.Found, s.Total = found, total
		if u.Reset {
			s.Recent = nil
		}
		for _, d := range u.New {
			r := RecentView{Name: d.Enemy.Name, EnemySprite: icons.Enemy(d.Enemy.Type), ItemName: copyrod.ItemName(d.Item),
				Sprite: icons.Item(d.Item), At: at}
			s.Recent = append([]RecentView{r}, s.Recent...)
		}
		if len(s.Recent) > maxRecent {
			s.Recent = s.Recent[:maxRecent]
		}
	})
}
