# Go-Stop LAN (고스톱)

A portable, zero-install-on-client, browser-based **Go-Stop (Hwatu)** multiplayer game hosted on your local Wi-Fi/LAN.

## Quick Start (Mac & Windows)

1. Install [Go 1.18+](https://go.dev/dl/) on the host computer.
2. Clone this repository and run:

```bash
go run ./cmd/gostop
```

3. A **LAN URL** (e.g. `http://192.168.1.x:8080`) and a **QR code** will appear in your terminal.
4. Scan the QR code with your phone camera (or open the URL in any browser on the same Wi-Fi) — **no app installation required**.

### Optional: Build a Single Standalone Binary

```bash
# macOS / Linux
go build -o gostop ./cmd/gostop

# Windows
go build -o gostop.exe ./cmd/gostop
```

You can also specify a custom port:
```bash
go run ./cmd/gostop -port 3000
```

## Architecture

- `internal/game/` — Pure, deterministic Go-Stop engine (deck, rules, turn phases, scoring, settlement) with zero networking dependencies.
- `internal/server/` — WebSocket room manager, session reconnection tokens, LAN IP detection, and terminal QR code banner.
- `web/` — Embedded static frontend (`go:embed`) using native ES modules and CSS custom properties (zero Node/npm build step).

## Card Artwork Attribution

The 48 Korean Hwatu SVG card illustrations in `web/assets/cards/` are from [Wikimedia Commons `Category:SVG Hwatu`](https://commons.wikimedia.org/wiki/Category:SVG_Hwatu) (individual SVGs by Spenĉjo, Korean Hwatu design by Marcus Richert, based on Hanafuda graphics by Louie Mantia, Jr.), licensed under **CC BY-SA 4.0**. See `web/assets/cards/ATTRIBUTION.txt` for details.
