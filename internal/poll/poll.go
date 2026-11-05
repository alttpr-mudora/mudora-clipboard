// Package poll polls SNI for the Copy Rod tables and reports what it finds to one or more Sinks (the console, the
// web UI). It reconnects when SNI or the device goes away.
package poll

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/alttpr-mudora/mudora-clipboard/internal/copyrod"
	"github.com/alttpr-mudora/mudora-clipboard/internal/sni"
)

// retryDelay is how long to wait before reconnecting after an error.
const retryDelay = 2 * time.Second

// ErrNotReady is returned in Once mode when the Copy Rod tables are not in memory.
var ErrNotReady = errors.New("the Copy Rod tables were not found (is the Copy Rod ROM running with a save loaded?)")

// Config says what to poll and how often.
type Config struct {
	Addr     string        // SNI's gRPC address
	Device   string        // a device URI or part of one; empty means the first device SNI lists
	Interval time.Duration // how often to poll
	Once     bool          // read once and stop, instead of following the game
}

// Sink receives what the poll loop observes. The console and the web UI are both sinks.
type Sink interface {
	// Connected is called when a device was found and reads are starting.
	Connected(device string)
	// Notice reports that the device is connected but the Copy Rod tables are not in memory yet.
	Notice(msg string)
	// Disconnected reports that the device or SNI cannot be reached.
	Disconnected(msg string)
	// Update is called for every read in which the tables look valid; u says what changed.
	Update(snap *copyrod.Snapshot, u copyrod.Update)
}

// Multi fans events out to several sinks.
type Multi []Sink

// Connected implements Sink.
func (m Multi) Connected(device string) {
	for _, s := range m {
		s.Connected(device)
	}
}

// Notice implements Sink.
func (m Multi) Notice(msg string) {
	for _, s := range m {
		s.Notice(msg)
	}
}

// Disconnected implements Sink.
func (m Multi) Disconnected(msg string) {
	for _, s := range m {
		s.Disconnected(msg)
	}
}

// Update implements Sink.
func (m Multi) Update(snap *copyrod.Snapshot, u copyrod.Update) {
	for _, s := range m {
		s.Update(snap, u)
	}
}

// Run polls until the context ends (or, with cfg.Once, until one table has been reported). It returns a process
// exit code, and writes errors that stop a Once run to errOut.
func Run(ctx context.Context, cfg Config, sink Sink, errOut io.Writer) int {
	for ctx.Err() == nil {
		err := session(ctx, cfg, sink)
		if cfg.Once {
			if err != nil {
				fmt.Fprintln(errOut, "error:", err)
				return 1
			}
			return 0
		}
		if err != nil && ctx.Err() == nil {
			sink.Disconnected(err.Error() + " - retrying")
		}
		select {
		case <-ctx.Done():
		case <-time.After(retryDelay):
		}
	}
	return 0
}

// session connects to SNI and tracks until an error occurs (or, with cfg.Once, one table is reported).
func session(ctx context.Context, cfg Config, sink Sink) error {
	c, err := sni.Dial(cfg.Addr)
	if err != nil {
		return err
	}
	defer c.Close()
	return RunSession(ctx, c, cfg, sink)
}

// RunSession picks a device on an already connected client and tracks it until an error occurs or the context
// ends (or, with cfg.Once, until one table is reported). Each call starts with a new Tracker, so reconnecting
// starts over.
func RunSession(ctx context.Context, c *sni.Client, cfg Config, sink Sink) error {
	dev, err := c.PickDevice(ctx, cfg.Device)
	if err != nil {
		return err
	}
	sink.Connected(fmt.Sprintf("%s (%s)", dev.DisplayName, dev.Uri))

	var tr copyrod.Tracker
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()
	for {
		snap, err := c.Read(ctx, dev.Uri)
		if err != nil {
			return err
		}
		u := tr.Update(snap)
		if u.Ready {
			sink.Update(&u.Snap, u)
		} else {
			if cfg.Once {
				return ErrNotReady
			}
			sink.Notice("waiting for the Copy Rod tables (is a save loaded?)")
		}
		if cfg.Once {
			return nil
		}
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}
	}
}
