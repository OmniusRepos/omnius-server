# Torrent Server API Documentation

Base URL: `http://localhost:8080` (local) or `https://api.omnius.lol` (production)

## Authentication

No authentication required for public API endpoints.
Admin endpoints require basic auth or session cookie.

---

## Movies API

### List Movies
```
GET /api/v2/list_movies.json
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Results per page (max 50) |
| page | int | 1 | Page number |
| quality | string | - | Filter by quality (720p, 1080p, 2160p) |
| minimum_rating | float | 0 | Minimum IMDB rating |
| query_term | string | - | Search by title |
| genre | string | - | Filter by genre |
| sort_by | string | date_added | Sort field (date_added, rating, year, title) |
| order_by | string | desc | Sort order (asc, desc) |
| year | int | - | Filter by exact year |
| status | string | - | Filter by status (available, coming_soon) |

**Response:**
```json
{
  "status": "ok",
  "status_message": "Query was successful",
  "data": {
    "movie_count": 503,
    "limit": 20,
    "page_number": 1,
    "movies": [
      {
        "id": 1,
        "imdb_code": "tt1234567",
        "title": "Movie Title",
        "year": 2024,
        "rating": 8.5,
        "genres": ["Action", "Drama"],
        "torrents": [...],
        "franchise": "Franchise Name"
      }
    ]
  }
}
```

### Movie Details
```
GET /api/v2/movie_details.json?movie_id={id}
```

**Query Parameters:**
| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| movie_id | int | Yes | Movie ID |
| with_cast | bool | No | Include cast information |
| with_images | bool | No | Include additional images |

### Movie Suggestions
```
GET /api/v2/movie_suggestions.json?movie_id={id}
```

Returns similar movies based on genre and rating.

### Franchise Movies
```
GET /api/v2/franchise_movies.json?movie_id={id}
```

Returns all movies in the same franchise (e.g., all Avengers movies).

---

## Series API

### List Series
```
GET /api/v2/list_series.json
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 20 | Results per page |
| page | int | 1 | Page number |
| query_term | string | - | Search by title |
| genre | string | - | Filter by genre |
| status | string | - | Filter by status (Continuing, Ended) |
| network | string | - | Filter by network (HBO, Netflix, etc.) |
| minimum_rating | float | 0 | Minimum rating |
| sort_by | string | date_added | Sort field |
| order_by | string | desc | Sort order |

**Response:**
```json
{
  "status": "ok",
  "data": {
    "series_count": 50,
    "limit": 20,
    "page_number": 1,
    "series": [
      {
        "id": 1,
        "imdb_code": "tt1234567",
        "title": "Series Title",
        "year": 2020,
        "total_seasons": 5,
        "status": "Continuing"
      }
    ]
  }
}
```

### Series Details
```
GET /api/v2/series_details.json?series_id={id}
```

### Season Episodes
```
GET /api/v2/season_episodes.json?series_id={id}&season={num}
```

Returns all episodes for a specific season with torrent information.

---

## Channels API (IPTV)

### List Channels
```
GET /api/v2/list_channels.json
```

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| limit | int | 50 | Results per page (max 100) |
| page | int | 1 | Page number |
| country | string | - | Filter by country code (US, UK, DE) |
| category | string | - | Filter by category (news, sports, movies) |
| query_term | string | - | Search by channel name |

**Response:**
```json
{
  "status": "ok",
  "data": {
    "channel_count": 39006,
    "limit": 50,
    "page_number": 1,
    "channels": [
      {
        "id": "BBCOne.uk",
        "name": "BBC One",
        "country": "UK",
        "categories": ["general", "entertainment"],
        "logo": "https://...",
        "stream_url": "https://..."
      }
    ]
  }
}
```

### Channel Countries
```
GET /api/v2/channel_countries.json
```

Returns list of all countries with available channels.

### Channel Categories
```
GET /api/v2/channel_categories.json
```

Returns list of all channel categories.

### Channels by Country
```
GET /api/v2/channels_by_country.json?country={code}
```

---

## Unified Search

### Search All Content
```
GET /api/v2/search.json?query={term}
```

Searches across movies, series, and channels simultaneously.

**Query Parameters:**
| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| query | string | Required | Search term |
| limit | int | 10 | Results per category (max 50) |

**Response:**
```json
{
  "status": "ok",
  "data": {
    "query": "news",
    "movies": [...],
    "series": [...],
    "channels": [...]
  }
}
```

---

## Analytics API

### Record View
```
POST /api/v2/analytics/view
```

**Body:**
```json
{
  "content_type": "movie",
  "content_id": 123,
  "imdb_code": "tt1234567",
  "device_id": "unique_device_id",
  "duration": 3600,
  "completed": true,
  "quality": "1080p"
}
```

### Stream Start
```
POST /api/v2/analytics/stream/start
```

### Stream Heartbeat
```
POST /api/v2/analytics/stream/heartbeat
```

Call every 30-60 seconds during active streaming.

### Stream End
```
POST /api/v2/analytics/stream/end
```

### Top Movies
```
GET /api/v2/analytics/top-movies?days=7&limit=10
```

---

## Home API

### Get Home Data
```
GET /api/v2/home.json
```

Returns configured home sections with content.

**Response:**
```json
{
  "status": "ok",
  "data": {
    "hero_slider": [...],
    "sections": [
      {
        "id": "recently_added",
        "title": "Recently Added",
        "type": "recent",
        "display_type": "carousel",
        "movies": [...]
      }
    ]
  }
}
```

---

## Sync API

### Sync Movie
```
POST /api/v2/sync_movie
```

Syncs a movie from external sources (YTS, IMDB) to local database.

**Body:**
```json
{
  "imdb_code": "tt1234567",
  "franchise": "Optional Franchise Name"
}
```

### Refresh Movie
```
POST /api/v2/refresh_movie
```

Refreshes movie data from external sources.

**Body:**
```json
{
  "movie_id": 123
}
```

### Torrent Stats
```
GET /api/v2/torrent_stats?hash={hash}
```

Gets real-time seed/peer information for a torrent.

---

## Curated Lists

### List Curated Lists
```
GET /api/v2/curated_lists.json
```

### Get Curated List
```
GET /api/v2/curated_list.json?slug={slug}
```

---

## Coming Soon / Reminders

### Check Availability
```
GET /api/v2/check_availability?imdb_codes=tt123,tt456
```

Checks if "coming soon" movies are now available.

**Response:**
```json
{
  "status": "ok",
  "data": {
    "tt123": {
      "available": true,
      "title": "Movie Title",
      "id": 123,
      "poster": "https://..."
    },
    "tt456": {
      "available": false
    }
  }
}
```
