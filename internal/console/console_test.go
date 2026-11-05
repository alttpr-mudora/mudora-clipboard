package console_test

import (
	"strings"
	"testing"

	"github.com/alttpr-mudora/mudora-clipboard/internal/console"
	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod/copyrodtest"
)

func TestRenderByEnemy(t *testing.T) {
	s := copyrodtest.ValidSnapshot()
	s.MarkSeen(0x4B)

	hidden := console.RenderByEnemy(&s, false)
	if !strings.HasPrefix(hidden, "Enemies found: 1 / 78\n") {
		t.Errorf("missing progress line:\n%s", hidden)
	}
	if !strings.Contains(hidden, "Green Knife Guard") || strings.Contains(hidden, "Raven") {
		t.Errorf("only found enemies should be listed:\n%s", hidden)
	}
	if !strings.Contains(hidden, "(77 not found yet)") {
		t.Errorf("missing remaining count:\n%s", hidden)
	}

	shown := console.RenderByEnemy(&s, true)
	if !strings.Contains(shown, "Raven") || !strings.Contains(shown, "* Green Knife Guard") {
		t.Errorf("reveal should list everything and mark found enemies:\n%s", shown)
	}
}

func TestRenderByItem(t *testing.T) {
	s := copyrodtest.ValidSnapshot()
	// give two enemies the same item and find both
	s.EnemyMap[0x00] = 0x10 // Hammer
	s.EnemyMap[0x0D] = 0x10
	s.MarkSeen(0x00)
	s.MarkSeen(0x0D)

	out := console.RenderByItem(&s, false)
	if !strings.Contains(out, "Hammer") || !strings.Contains(out, "Raven, Buzzblob") {
		t.Errorf("items should group their enemies in table order:\n%s", out)
	}
	if strings.Contains(out, "Octorok") {
		t.Errorf("unfound enemies must stay hidden:\n%s", out)
	}
}
