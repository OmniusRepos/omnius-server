# Torrent Server - Development Notes

## Build Process

**Important**: This project uses Go's `//go:embed` to embed static files into the binary at compile time.

After making changes to the frontend (`frontend/`), you must:

1. Build the frontend: `cd frontend && npm run build`
2. **Rebuild the Go binary**: `go build -o torrent-server .`

Just rebuilding the frontend is NOT enough - the Go binary contains the old embedded files until recompiled.

## Project Structure

- `frontend/` - Svelte 5 admin UI, builds to `static/admin/`
- `static/admin/` - Built frontend assets (embedded into Go binary)
- `templates/` - Go HTML templates (also embedded)
- `data/omnius.db` - SQLite database (movies, torrents, series, channels, etc.)

## Database

The main database is `data/omnius.db` (SQLite). Key tables:
- `movies` - Movie metadata including `franchise` field for grouping series
- `torrents` - Torrent info linked to movies
- `series` - TV series metadata
- `home_sections` - Admin-configurable home page sections

## Quick Commands

```bash
# Full rebuild after frontend changes
cd frontend && npm run build && cd .. && go build -o torrent-server . && ./torrent-server

# Or with go run (also re-embeds)
cd frontend && npm run build && cd .. && go run .
```
