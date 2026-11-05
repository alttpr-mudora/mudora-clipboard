package console

import (
	"fmt"
	"sort"
	"strings"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
)

// RenderByEnemy lists enemies with their items. Without reveal only enemies the rod has hit are listed;
// with reveal every enemy is listed and the found ones are marked with '*'.
func RenderByEnemy(s *copyrod.Snapshot, reveal bool) string {
	var b strings.Builder
	found, total := copyrod.Progress(s)
	fmt.Fprintf(&b, "Enemies found: %d / %d\n", found, total)

	width := 0
	for _, e := range copyrod.Enemies {
		if len(e.Name) > width {
			width = len(e.Name)
		}
	}
	for _, e := range copyrod.Enemies {
		seen := s.IsSeen(e.Type)
		if !seen && !reveal {
			continue
		}
		mark := " "
		if reveal && seen {
			mark = "*"
		}
		fmt.Fprintf(&b, " %s %-*s  %s\n", mark, width, e.Name, copyrod.ItemName(s.EnemyMap[e.Type]))
	}
	if !reveal && found < total {
		fmt.Fprintf(&b, "   (%d not found yet)\n", total-found)
	}
	return b.String()
}

// RenderByItem groups enemies under the item they carry, items in id order. Without reveal only enemies
// the rod has hit appear (and items with no found enemy are omitted); with reveal everything appears and
// found enemies are marked with '*'.
func RenderByItem(s *copyrod.Snapshot, reveal bool) string {
	byItem := map[byte][]string{}
	for _, e := range copyrod.Enemies {
		seen := s.IsSeen(e.Type)
		if !seen && !reveal {
			continue
		}
		name := e.Name
		if reveal && seen {
			name += "*"
		}
		id := s.EnemyMap[e.Type]
		byItem[id] = append(byItem[id], name)
	}
	ids := make([]int, 0, len(byItem))
	for id := range byItem {
		ids = append(ids, int(id))
	}
	sort.Ints(ids)

	var b strings.Builder
	found, total := copyrod.Progress(s)
	fmt.Fprintf(&b, "Enemies found: %d / %d\n", found, total)
	width := 0
	for _, id := range ids {
		if n := len(copyrod.ItemName(byte(id))); n > width {
			width = n
		}
	}
	for _, id := range ids {
		fmt.Fprintf(&b, "  %-*s  %s\n", width, copyrod.ItemName(byte(id)), strings.Join(byItem[byte(id)], ", "))
	}
	if !reveal && found < total {
		fmt.Fprintf(&b, "  (%d not found yet)\n", total-found)
	}
	return b.String()
}
