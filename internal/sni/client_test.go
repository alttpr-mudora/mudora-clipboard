package sni_test

import (
	"context"
	"strings"
	"testing"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod/copyrodtest"
	"github.com/alttpr-mudora/mudora-clipboard/internal/sni"
	"github.com/alttpr-mudora/mudora-clipboard/internal/sni/snitest"
)

func TestClientRead(t *testing.T) {
	f := &snitest.Fake{Devices: []string{"fake://one"}}
	want := copyrodtest.ValidSnapshot()
	want.MarkSeen(0x4B)
	f.Put(copyrod.AddrEnemyMap, want.EnemyMap[:])
	f.Put(copyrod.AddrSpriteSeen, want.Seen[:])

	c := snitest.Start(t, f)
	ctx := context.Background()
	dev, err := c.PickDevice(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	got, err := c.Read(ctx, dev.Uri)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("snapshot mismatch\n got %x %x\nwant %x %x", got.EnemyMap[:16], got.Seen, want.EnemyMap[:16], want.Seen)
	}
	if f.Reads() != 1 {
		t.Errorf("a poll should be a single MultiRead, got %d", f.Reads())
	}
	if !got.IsSeen(0x4B) || got.IsSeen(0x4A) {
		t.Error("seen bits did not survive the round trip")
	}
}

func TestPickDevice(t *testing.T) {
	f := &snitest.Fake{Devices: []string{"emunwa://localhost:48879", "fxpakpro://./COM4"}}
	c := snitest.Start(t, f)
	ctx := context.Background()

	if d, err := c.PickDevice(ctx, ""); err != nil || d.Uri != "emunwa://localhost:48879" {
		t.Errorf("default should be the first device, got %v, %v", d, err)
	}
	if d, err := c.PickDevice(ctx, "COM4"); err != nil || d.Uri != "fxpakpro://./COM4" {
		t.Errorf("substring match failed, got %v, %v", d, err)
	}
	if _, err := c.PickDevice(ctx, "nothing"); err == nil || !strings.Contains(err.Error(), "emunwa://localhost:48879") {
		t.Errorf("a miss should list the available devices, got %v", err)
	}

	none := snitest.Start(t, &snitest.Fake{})
	if _, err := none.PickDevice(ctx, ""); err == nil || !strings.Contains(err.Error(), "no devices") {
		t.Errorf("no devices should be reported clearly, got %v", err)
	}
}

func TestUnreachableSNI(t *testing.T) {
	// nothing listens on this address
	c, err := sni.Dial("127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_, err = c.PickDevice(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "cannot reach SNI") {
		t.Errorf("want a 'cannot reach SNI' error, got %v", err)
	}
}
