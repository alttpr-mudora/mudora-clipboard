// mudora-clipboard is an auto-tracker for the Copy Rod ALttP hack ("A Copy to the Past"). It reads the hack's
// enemy->item table and "enemy found" bits from a running emulator or flash cart through SNI, and shows which
// enemies carry which items as you hit them with the rod, in the console and optionally in a small web UI.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"time"

	"github.com/alttpr-mudora/mudora-clipboard/internal"
	"github.com/alttpr-mudora/mudora-clipboard/internal/console"
	"github.com/alttpr-mudora/mudora-clipboard/internal/poll"
	"github.com/alttpr-mudora/mudora-clipboard/internal/ui"
)

// options are the command line flags.
type options struct {
	poll   poll.Config
	reveal bool   // list the items of enemies that have not been found
	byItem bool   // group the console table by item
	gui    bool   // serve the web UI
	listen string // address for the web UI
	noOpen bool   // do not open a browser window
}

func main() {
	var o options
	var version bool

	flag.StringVar(&o.poll.Addr, "addr", "localhost:8191", "SNI gRPC address")
	flag.StringVar(&o.poll.Device, "device", "", "device URI, or part of it (default: the first device SNI lists)")
	flag.DurationVar(&o.poll.Interval, "interval", 500*time.Millisecond, "how often to poll")
	flag.BoolVar(&o.poll.Once, "once", false, "read once, print the table and exit")
	flag.BoolVar(&o.reveal, "reveal", false, "list every enemy's item, including enemies not found yet (spoils the seed)")
	flag.BoolVar(&o.byItem, "by-item", false, "group the console table by item instead of by enemy")
	flag.BoolVar(&o.gui, "gui", false, "serve the web UI and open it in a browser")
	flag.StringVar(&o.listen, "listen", "127.0.0.1:8765", "address for the web UI")
	flag.BoolVar(&o.noOpen, "no-open", false, "with -gui: do not open a browser window")
	flag.BoolVar(&version, "version", false, "print the mudora-clipboard version")
	flag.BoolVar(&version, "v", false, "shorthand for -version")
	flag.Parse()

	if version {
		fmt.Println("mudora-clipboard", internal.Version)
		return
	}
	if o.poll.Once && o.gui {
		fmt.Fprintln(os.Stderr, "mudora-clipboard: -once cannot be combined with -gui")
		os.Exit(2)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	os.Exit(run(ctx, o, os.Stdout, os.Stderr))
}

// run wires the sinks together and polls until the context ends. It returns the process exit code.
func run(ctx context.Context, o options, out, errOut io.Writer) int {
	var sink poll.Sink = console.New(out, o.byItem, o.reveal)

	if o.gui {
		hub := ui.NewHub(o.reveal)
		url, err := ui.Start(ctx, o.listen, hub)
		if err != nil {
			fmt.Fprintln(errOut, "error:", err)
			return 1
		}
		fmt.Fprintf(out, "web UI: %s\n", url)
		if !o.noOpen {
			if err := ui.OpenBrowser(url); err != nil {
				fmt.Fprintln(errOut, "could not open a browser:", err)
			}
		}
		sink = poll.Multi{sink, hub}
	}

	return poll.Run(ctx, o.poll, sink, errOut)
}
