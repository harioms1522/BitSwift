# BitSwift

A lightweight, CLI-first BitTorrent client (BitTorrent v1.0). Implemented in Go.

## Phase 1: Torrent parsing and CLI

**Goal:** Read and validate `.torrent` files; expose metadata via the CLI. No network, no download.

### Build

```bash
go build -o bitswift ./cmd/bitswift
```

### Run

```bash
bitswift <path_to_torrent>
```

- **Valid torrent:** Exits 0; prints name, info hash, piece count, piece length, file count, total size.
- **Missing file:** Exits non-zero; prints "file not found: &lt;path&gt;".
- **Invalid/corrupt file:** Exits non-zero; prints "invalid torrent" or parse error.

### Test

```bash
go test ./...
```

### Layout

- `cmd/bitswift` — CLI entrypoint
- `internal/bencode` — Bencode parser (integers, strings, lists, dictionaries)
- `internal/torrent` — Torrent parser (announce, announce-list, info, info hash)
- `testdata/` — Sample .torrent files for manual testing

See [docs/IMPLEMENTATION_PHASES.md](docs/IMPLEMENTATION_PHASES.md) and [docs/PRODUCT_SPEC.md](docs/PRODUCT_SPEC.md) for the full spec.
