# Database Schema Documentation

The torrent-server uses SQLite with the `modernc.org/sqlite` pure Go driver.

Database location: `data/omnius.db`

---

## Tables Overview

| Table | Purpose |
|-------|---------|
| movies | Movie metadata and info |
| torrents | Torrent files linked to movies |
| series | TV series metadata |
| episodes | TV series episodes |
| episode_torrents | Torrent files for episodes |
| season_packs | Full season torrent packs |
| seasons | Season metadata |
| channels | IPTV live channels |
| channel_countries | Country lookup table |
| channel_categories | Category lookup table |
| curated_lists | Admin-created movie lists |
| curated_list_movies | Movies in curated lists |
| home_sections | Home page section config |
| content_views | Analytics - view tracking |
| content_stats_daily | Analytics - daily aggregates |
| active_streams | Analytics - active viewers |

---

## Movies Table

```sql
CREATE TABLE movies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    imdb_code TEXT UNIQUE,
    title TEXT NOT NULL,
    title_english TEXT,
    title_long TEXT,
    slug TEXT,
    year INTEGER,
    rating REAL DEFAULT 0,
    runtime INTEGER DEFAULT 0,
    genres TEXT,                    -- JSON array
    summary TEXT,
    description_full TEXT,
    synopsis TEXT,
    yt_trailer_code TEXT,
    language TEXT DEFAULT 'en',
    background_image TEXT,
    small_cover_image TEXT,
    medium_cover_image TEXT,
    large_cover_image TEXT,
    date_uploaded TEXT,
    date_uploaded_unix INTEGER,

    -- Extended fields
    imdb_rating REAL,
    rotten_tomatoes INTEGER,
    metacritic INTEGER,
    mpa_rating TEXT,
    url TEXT,
    background_image_original TEXT,
    like_count INTEGER DEFAULT 0,
    download_count INTEGER DEFAULT 0,
    ratings_updated_at TEXT,
    state TEXT DEFAULT 'ok',
    franchise TEXT,                 -- Franchise grouping (e.g., "Avengers")
    imdb_votes TEXT,
    content_type TEXT DEFAULT 'movie',
    provider TEXT,

    -- Rich metadata from IMDB
    director TEXT,
    writers TEXT,                   -- JSON array
    cast_json TEXT,                 -- JSON array with cast info
    budget TEXT,
    box_office_gross TEXT,
    country TEXT,
    awards TEXT,
    all_images TEXT,                -- JSON array of image URLs

    -- Coming soon support
    status TEXT DEFAULT 'available', -- 'available' or 'coming_soon'
    release_date TEXT,              -- YYYY-MM-DD format

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

### Key Fields

- **franchise**: Groups related movies (e.g., "Avengers", "Star Wars")
- **status**: "available" for released movies, "coming_soon" for upcoming
- **genres**: Stored as JSON array `["Action", "Drama"]`

---

## Torrents Table

```sql
CREATE TABLE torrents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    movie_id INTEGER NOT NULL,
    url TEXT,
    hash TEXT NOT NULL,
    quality TEXT,                   -- 720p, 1080p, 2160p
    type TEXT DEFAULT 'web',        -- web, bluray, hdtv
    video_codec TEXT,               -- x264, x265, HEVC
    seeds INTEGER DEFAULT 0,
    peers INTEGER DEFAULT 0,
    size TEXT,                      -- Human readable (1.5 GB)
    size_bytes INTEGER,
    date_uploaded TEXT,
    date_uploaded_unix INTEGER,
    FOREIGN KEY (movie_id) REFERENCES movies(id) ON DELETE CASCADE
);
```

---

## Series Table

```sql
CREATE TABLE series (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    imdb_code TEXT UNIQUE,
    title TEXT NOT NULL,
    title_slug TEXT,
    year INTEGER,
    rating REAL DEFAULT 0,
    genres TEXT,                    -- JSON array
    summary TEXT,
    poster_image TEXT,
    background_image TEXT,
    total_seasons INTEGER DEFAULT 0,
    status TEXT DEFAULT 'ongoing',  -- ongoing, ended
    date_added TEXT,
    date_added_unix INTEGER,

    -- Extended fields
    tvdb_id INTEGER,
    end_year INTEGER,
    runtime INTEGER DEFAULT 0,
    network TEXT,                   -- HBO, Netflix, etc.
    total_episodes INTEGER DEFAULT 0,
    imdb_rating REAL,
    rotten_tomatoes INTEGER,
    franchise TEXT,

    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Episodes Table

```sql
CREATE TABLE episodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    series_id INTEGER NOT NULL,
    season INTEGER NOT NULL,
    episode INTEGER NOT NULL,
    title TEXT,
    overview TEXT,
    air_date TEXT,
    imdb_code TEXT,
    summary TEXT,
    runtime INTEGER,
    still_image TEXT,
    FOREIGN KEY (series_id) REFERENCES series(id) ON DELETE CASCADE,
    UNIQUE(series_id, season, episode)
);
```

---

## Episode Torrents Table

```sql
CREATE TABLE episode_torrents (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    episode_id INTEGER NOT NULL,
    series_id INTEGER,
    season_number INTEGER,
    episode_number INTEGER,
    hash TEXT NOT NULL,
    quality TEXT,
    video_codec TEXT,
    seeds INTEGER DEFAULT 0,
    peers INTEGER DEFAULT 0,
    size TEXT,
    size_bytes INTEGER,
    source TEXT,
    release_group TEXT,
    date_uploaded TEXT,
    date_uploaded_unix INTEGER,
    FOREIGN KEY (episode_id) REFERENCES episodes(id) ON DELETE CASCADE
);
```

---

## Channels Table (IPTV)

```sql
CREATE TABLE channels (
    id TEXT PRIMARY KEY,            -- e.g., "BBCOne.uk"
    name TEXT NOT NULL,
    country TEXT,                   -- Country code (US, UK, DE)
    languages TEXT,                 -- JSON array
    categories TEXT,                -- JSON array
    logo TEXT,
    stream_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE channel_countries (
    code TEXT PRIMARY KEY,          -- ISO country code
    name TEXT NOT NULL,
    flag TEXT                       -- Flag emoji or URL
);

CREATE TABLE channel_categories (
    id TEXT PRIMARY KEY,            -- e.g., "news", "sports"
    name TEXT NOT NULL
);
```

---

## Home Sections Table

```sql
CREATE TABLE home_sections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    section_id TEXT UNIQUE NOT NULL,
    title TEXT NOT NULL,
    section_type TEXT NOT NULL DEFAULT 'query',
    display_type TEXT NOT NULL DEFAULT 'carousel',
    query_type TEXT,
    genre TEXT,
    curated_list_id INTEGER,
    content_type TEXT,              -- movie, series, channel
    content_id INTEGER,
    sort_by TEXT DEFAULT 'rating',
    order_by TEXT DEFAULT 'desc',
    minimum_rating REAL DEFAULT 0,
    limit_count INTEGER DEFAULT 10,
    is_active INTEGER DEFAULT 1,
    display_order INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (curated_list_id) REFERENCES curated_lists(id) ON DELETE SET NULL
);
```

### Display Types
- `hero` - Large featured banner
- `carousel` - Horizontal scrollable row
- `grid` - Grid layout
- `featured` - Featured content section
- `banner` - Promotional banner

---

## Analytics Tables

```sql
-- Individual view tracking
CREATE TABLE content_views (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content_type TEXT NOT NULL,     -- movie, series, episode
    content_id INTEGER NOT NULL,
    imdb_code TEXT,
    device_id TEXT,
    view_date DATE NOT NULL,
    view_count INTEGER DEFAULT 1,
    watch_duration INTEGER DEFAULT 0,
    completed INTEGER DEFAULT 0,
    quality TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(content_type, content_id, device_id, view_date)
);

-- Daily aggregates for fast Top 10 queries
CREATE TABLE content_stats_daily (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    content_type TEXT NOT NULL,
    content_id INTEGER NOT NULL,
    stat_date DATE NOT NULL,
    view_count INTEGER DEFAULT 0,
    unique_viewers INTEGER DEFAULT 0,
    total_watch_time INTEGER DEFAULT 0,
    completions INTEGER DEFAULT 0,
    UNIQUE(content_type, content_id, stat_date)
);

-- Real-time active streams
CREATE TABLE active_streams (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id TEXT NOT NULL UNIQUE,
    content_type TEXT NOT NULL,
    content_id INTEGER,
    imdb_code TEXT,
    quality TEXT,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_heartbeat DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

---

## Indexes

```sql
CREATE INDEX idx_movies_imdb ON movies(imdb_code);
CREATE INDEX idx_movies_year ON movies(year);
CREATE INDEX idx_movies_rating ON movies(rating);
CREATE INDEX idx_torrents_movie ON torrents(movie_id);
CREATE INDEX idx_torrents_hash ON torrents(hash);
CREATE INDEX idx_series_imdb ON series(imdb_code);
CREATE INDEX idx_episodes_series ON episodes(series_id);
CREATE INDEX idx_home_sections_order ON home_sections(display_order);
CREATE INDEX idx_content_views_date ON content_views(view_date);
CREATE INDEX idx_content_views_content ON content_views(content_type, content_id);
CREATE INDEX idx_content_stats_date ON content_stats_daily(stat_date);
CREATE INDEX idx_content_stats_content ON content_stats_daily(content_type, content_id);
CREATE INDEX idx_channels_country ON channels(country);
CREATE INDEX idx_channels_name ON channels(name);
```

---

## Migrations

Migrations are applied automatically on startup in `database/sqlite.go`. New columns are added using `ALTER TABLE` with error suppression for existing columns.

Example migration pattern:
```go
migrations := []string{
    "ALTER TABLE movies ADD COLUMN franchise TEXT",
    "ALTER TABLE movies ADD COLUMN status TEXT DEFAULT 'available'",
}
for _, m := range migrations {
    d.Exec(m) // Ignore errors if column exists
}
```
