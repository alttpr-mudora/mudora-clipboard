package icons_test

import (
	"image"
	_ "image/png"
	"io/fs"
	"path"
	"testing"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	"github.com/alttpr-mudora/mudora-clipboard/internal/icons"
)

// The icon boxes in the web UI (static/style.css), and the size the sprite files were scaled to (their longest side).
const (
	itemBox     = 36
	enemyBox    = 44
	itemTarget  = 32
	enemyTarget = 40
	// Sprites were scaled by a whole number where that landed within 10% of the target, so the longest side
	// is within that of the target.
	sizeSlack = 4
)

func isPNG(t *testing.T, p string) bool {
	t.Helper()
	data, err := fs.ReadFile(icons.FS(), p)
	if err != nil {
		t.Errorf("%s is not embedded: %v", p, err)
		return false
	}
	if len(data) < 8 || string(data[1:4]) != "PNG" {
		t.Errorf("%s is not a PNG", p)
		return false
	}
	return true
}

func TestEveryItemHasASprite(t *testing.T) {
	for id := byte(copyrod.FirstItemID); id <= copyrod.LastItemID; id++ {
		p := icons.Item(id)
		if p == "" {
			t.Errorf("item %02X (%s) has no sprite", id, copyrod.ItemName(id))
			continue
		}
		isPNG(t, p)
	}
	if icons.Item(0x01) != "" || icons.Item(0x00) != "" {
		t.Error("the plain Copy Rod and 'no item' have no sprite")
	}
}

func TestEnemySprites(t *testing.T) {
	for _, e := range copyrod.Enemies {
		p := icons.Enemy(e.Type)
		if p == "" {
			t.Errorf("%s (%02X) has no sprite", e.Name, e.Type)
			continue
		}
		isPNG(t, p)
	}

	// no entry for a type the hack does not know
	known := map[byte]bool{}
	for _, e := range copyrod.Enemies {
		known[e.Type] = true
	}
	for typ := 0; typ < 256; typ++ {
		if icons.Enemy(byte(typ)) != "" && !known[byte(typ)] {
			t.Errorf("there is a sprite for unknown enemy type %02X", typ)
		}
	}
}

func TestNoStraySprites(t *testing.T) {
	used := map[string]bool{}
	for i := 0; i < 256; i++ {
		used[icons.Item(byte(i))] = true
		used[icons.Enemy(byte(i))] = true
	}
	for _, dir := range []string{"items", "enemies"} {
		entries, err := fs.ReadDir(icons.FS(), dir)
		if err != nil {
			t.Fatal(err)
		}
		for _, e := range entries {
			if p := path.Join(dir, e.Name()); !used[p] {
				t.Errorf("%s is embedded but nothing uses it", p)
			}
		}
	}
}

func size(t *testing.T, p string) (w, h int) {
	t.Helper()
	f, err := icons.FS().Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		t.Fatalf("%s: %v", p, err)
	}
	return cfg.Width, cfg.Height
}

// Every sprite is shown at its natural size, so the files themselves must be the right size. If this fails after
// adding or changing a sprite, scale it so its longest side is about the target.
func TestSpritesAreScaledToASharedSize(t *testing.T) {
	check := func(p string, target, box int) {
		w, h := size(t, p)
		if longest := max(w, h); longest < target-sizeSlack || longest > target+sizeSlack {
			t.Errorf("%s is %dx%d: its longest side should be about %d", p, w, h, target)
		}
		if w > box || h > box {
			t.Errorf("%s is %dx%d and does not fit its %d px box", p, w, h, box)
		}
	}
	for i := 0; i < 256; i++ {
		if p := icons.Item(byte(i)); p != "" {
			check(p, itemTarget, itemBox)
		}
		if p := icons.Enemy(byte(i)); p != "" {
			check(p, enemyTarget, enemyBox)
		}
	}
}
