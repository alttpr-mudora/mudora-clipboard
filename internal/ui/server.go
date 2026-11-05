package ui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"time"

	"github.com/alttpr-mudora/mudora-clipboard/internal/icons"
)

//go:embed static
var staticFS embed.FS

// Handler serves the UI files, the sprites (under /img/) and the state API:
//
//	GET /api/state   the current state as JSON
//	GET /api/events  a server-sent-events stream; each event is the full state as JSON
func (h *Hub) Handler() http.Handler {
	static, err := fs.Sub(staticFS, "static")
	if err != nil {
		panic(err) // the embedded directory is part of the build
	}
	mux := http.NewServeMux()
	mux.Handle("/", noStore(http.FileServer(http.FS(static))))
	mux.Handle("/img/", http.StripPrefix("/img/", noStore(http.FileServer(http.FS(icons.FS())))))
	mux.Handle("/api/state", noStore(http.HandlerFunc(h.serveState)))
	mux.HandleFunc("/api/events", h.serveEvents)
	return mux
}

func noStore(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}

func (h *Hub) serveState(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(h.State())
}

func (h *Hub) serveEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	changed, cancel := h.Subscribe()
	defer cancel()

	send := func() error {
		b, err := json.Marshal(h.State())
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", b); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}
	if send() != nil {
		return
	}
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-changed:
			if send() != nil {
				return
			}
		case <-keepalive.C:
			if _, err := fmt.Fprint(w, ": keepalive\n\n"); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

// Start listens on addr and serves the UI in the background until ctx ends. It returns the URL.
func Start(ctx context.Context, addr string, h *Hub) (string, error) {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("web UI: %w", err)
	}
	srv := &http.Server{Handler: h.Handler(), ReadHeaderTimeout: 5 * time.Second}
	go srv.Serve(ln)
	go func() {
		<-ctx.Done()
		srv.Close()
	}()
	return "http://" + ln.Addr().String(), nil
}
