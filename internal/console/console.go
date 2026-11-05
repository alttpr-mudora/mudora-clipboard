// Package console prints what the tracker sees as plain text: the whole table when a game is first seen, then one
// line per discovery.
package console

import (
	"fmt"
	"io"
	"time"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
)

// Sink prints to a writer: the whole table when it is (re)established, then one line per discovery.
// Status messages print only when they differ from the previous one, so retry loops stay quiet.
type Sink struct {
	out    io.Writer
	byItem bool
	reveal bool
	last   string
}

// New returns a Sink that prints to out. byItem groups the table by item instead of by enemy, and reveal lists the
// items of enemies that have not been found.
func New(out io.Writer, byItem, reveal bool) *Sink {
	return &Sink{out: out, byItem: byItem, reveal: reveal}
}

func (t *Sink) line(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	if msg == t.last {
		return
	}
	t.last = msg
	fmt.Fprintf(t.out, "[%s] %s\n", time.Now().Format("15:04:05"), msg)
}

// Connected implements the poll loop's sink.
func (t *Sink) Connected(device string) { t.line("connected to %s", device) }

// Notice reports that the device is connected but the tables are not in memory.
func (t *Sink) Notice(msg string) { t.line("%s", msg) }

// Disconnected reports that the device or SNI cannot be reached.
func (t *Sink) Disconnected(msg string) { t.line("%s", msg) }

// Update prints the whole table when u.Reset is set, otherwise a line for each new discovery.
func (t *Sink) Update(snap *copyrod.Snapshot, u copyrod.Update) {
	if u.Reset {
		t.last = "" // let the next status message print even if it repeats an earlier one
		if t.byItem {
			fmt.Fprint(t.out, RenderByItem(snap, t.reveal))
		} else {
			fmt.Fprint(t.out, RenderByEnemy(snap, t.reveal))
		}
		return
	}
	found, total := copyrod.Progress(snap)
	for i, d := range u.New {
		fmt.Fprintf(t.out, "[%s] NEW  %s -> %s  (%d/%d)\n", time.Now().Format("15:04:05"),
			d.Enemy.Name, copyrod.ItemName(d.Item), found-len(u.New)+i+1, total)
	}
}
