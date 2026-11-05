# mudora-clipboard

An auto-tracker for the Copy Rod ALttP hack ("A Copy to the Past"). It reads the hack's enemy->item table and its "enemy found" bits through [SNI](https://github.com/alttpo/sni) and shows which enemies carry which items as you hit them with the rod. Nothing is written to the game.

# Usage

SNI must be running, with a device for the Copy Rod ROM: snes9x-nwa (NWA), an FX Pak Pro, or an emulator connected through SNI's Lua bridge. Only the first matching device is used (`-device` matches part of its URI). By default only enemies you have hit are shown; the rest stay hidden.

```sh
go run ./cmd/mudora-clipboard <flags>
```

Compile a binary from source and place it on `PATH` if you prefer:

```sh
go build -o mudora-clipboard.exe ./cmd/mudora-clipboard
```

```
mudora-clipboard                      # follow along: prints the table, then one line per new discovery
mudora-clipboard -once                # print the table once and exit
mudora-clipboard -by-item             # group the console table by item instead of by enemy
mudora-clipboard -reveal              # list every enemy's item (spoils the seed); found enemies are marked *
mudora-clipboard -addr host:8191 -device COM4 -interval 250ms
mudora-clipboard -gui                 # also serve the web UI (http://127.0.0.1:8765) and open it in a browser
mudora-clipboard -version             # print the version (-v for short)
```

Example follow output:

```
[21:04:12] connected to snes9x-nwa (emunwa://localhost:48879)
Enemies found: 2 / 78
   Octorok            Blue Boomerang
   Green Knife Guard  Lamp
   (76 not found yet)
[21:05:40] NEW  Raven -> Hammer  (3/78)
```

## Web UI

`-gui` serves a page (default `http://127.0.0.1:8765`, change it with `-listen`; `-no-open` skips opening a browser).

- **Left:** every item the hack can hand out, in item order, each with its sprite and the enemies found carrying it underneath. Items nobody has been found with are dimmed ("not found yet"). A "By enemy" view, a filter box and a "Found only" checkbox are above the list. `/?view=enemy` (or `item`) opens a given view.
- **Right:** the "Just found" feed, newest first.
- Every enemy name shows its in-game sprite, and every item its icon. Unfound enemies show their sprite dimmed; it reveals nothing.
- Unfound enemies never have their item sent to the page, so the browser's dev tools cannot spoil the seed (`-reveal` is the only way to see them).
- `/?overlay` gives a transparent page with only the feed, for an OBS browser source.
- The page updates live (server-sent events from `/api/events`; `/api/state` returns the same JSON once).

The sprites are mudora's item icons and the Zelda Dungeon wiki's enemy sprites, scaled so that every item icon is the same size and every enemy icon is the same size; see [internal/icons](internal/icons/README.md) for credits.

## How it works

One `MultiRead` per poll, in SNI's FX Pak Pro address space (WRAM `$7E:0000` = `0xF50000`):

| Address | WRAM | Meaning |
|---|---|---|
| `0xF66518` (256 bytes) | `$7F:6518` | copied-item id for each sprite type |
| `0xF66628` (32 bytes) | `$7F:6628` | bit per sprite type, set when the rod hit it (MSB first) |

Enemy names and item names are compiled in (`internal/copyrod/names.go`).

Behaviour worth knowing:

- Until a save is loaded the WRAM does not hold the table. The tracker only trusts memory in which nearly every known enemy has an item id of 1..0x27, and says "waiting" otherwise.
- Nothing is forgotten when the game goes away and comes back. Save and quit, the file select screen, a soft reset, or the game clearing its found bits all leave the feed and everything found in place; the tracker keeps counting from there. Found enemies are never un-found.
- The tracker starts over only when a different enemy table (a different seed or ROM) shows up on two reads in a row, or when the connection to SNI is lost and re-established. Restart the program to clear it by hand.
- "Found" means the rod has hit that enemy type, not that you picked up anything. Enemies found before the tracker started appear in the item list but not in the feed, which only lists discoveries made while it is running.

## Status

- Tested: the tracker logic, the bit order, the rendering, the web server, and the whole SNI client and poll loop (including save and quit) against an in-process fake SNI server (`go test ./...`).
- Run once against a live SNI with Snes9x 1.63-nwa: the tables read correctly (`-once` and `-gui` showed the real game's found enemies and items). New discoveries and save and quit have only been exercised against the fake.
- Item ids `0x20`-`0x27` were never seen in a seed and have no names (they print as `item 0x2X`).

## Layout

```
cmd/mudora-clipboard/   the command: flags and wiring only
internal/
  copyrod/              the hack's data (addresses, enemy and item names) and the Tracker; copyrodtest/ has test helpers
  sni/                  the SNI client; pb/ is the vendored gRPC stub; snitest/ is a fake SNI for tests
  poll/                 the poll loop and the Sink interface it reports to
  console/              console output (tables and discovery lines)
  ui/                   the web UI: state hub, HTTP server, and the page itself (static/, embedded in the binary)
  icons/                the sprites (embedded) and their credits
  version.go            the version, set at release time
```

- `internal/sni/pb/` is the unmodified generated gRPC stub from SNI v0.0.102 (MIT, see its `LICENSE`). SNI's own Go module cannot be imported: every version, including `main`, declares its path as `sni`, hence the copy. There is no submodule.
