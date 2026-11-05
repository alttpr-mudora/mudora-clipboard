// Package copyrodtest has helpers for the tests of packages that use copyrod.
package copyrodtest

import "github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"

// ValidSnapshot returns a snapshot whose enemy table looks like a real seed (every known enemy has an item) and
// in which no enemy has been found.
func ValidSnapshot() copyrod.Snapshot {
	var s copyrod.Snapshot
	for i, e := range copyrod.Enemies {
		s.EnemyMap[e.Type] = byte(2 + i%20)
	}
	return s
}
