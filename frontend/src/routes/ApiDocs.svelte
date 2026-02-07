<script lang="ts">
  const endpoints = [
    {
      category: 'Configuration',
      items: [
        {
          method: 'GET',
          path: '/api/v2/config.json',
          description: 'Get enabled services configuration. Returns which services (Movies, TV Shows, Live TV) are active.',
          params: [],
        },
      ],
    },
    {
      category: 'Movies',
      items: [
        {
          method: 'GET',
          path: '/api/v2/list_movies.json',
          description: 'List all movies with filtering and pagination',
          params: [
            { name: 'limit', type: 'int', description: 'Number of results (default: 20, max: 50)' },
            { name: 'page', type: 'int', description: 'Page number (default: 1)' },
            { name: 'quality', type: 'string', description: 'Filter by quality (720p, 1080p, 2160p)' },
            { name: 'minimum_rating', type: 'float', description: 'Minimum IMDb rating' },
            { name: 'query_term', type: 'string', description: 'Search term for title' },
            { name: 'genre', type: 'string', description: 'Filter by genre' },
            { name: 'sort_by', type: 'string', description: 'Sort field (title, year, rating, date_added)' },
            { name: 'order_by', type: 'string', description: 'Sort order (asc, desc)' },
            { name: 'year', type: 'int', description: 'Filter by exact year' },
            { name: 'minimum_year', type: 'int', description: 'Minimum release year' },
            { name: 'maximum_year', type: 'int', description: 'Maximum release year' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/movie_details.json',
          description: 'Get detailed information about a specific movie',
          params: [
            { name: 'movie_id', type: 'int', description: 'Movie ID (required)' },
            { name: 'with_cast', type: 'bool', description: 'Include cast information' },
            { name: 'with_images', type: 'bool', description: 'Include additional images' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/movie_suggestions.json',
          description: 'Get movie suggestions based on a movie',
          params: [
            { name: 'movie_id', type: 'int', description: 'Movie ID (required)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/franchise_movies.json',
          description: 'Get all movies in a franchise/series (e.g. all Harry Potter movies)',
          params: [
            { name: 'movie_id', type: 'int', description: 'Movie ID (required)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/check_availability',
          description: 'Check if a movie exists by IMDB code',
          params: [
            { name: 'imdb_code', type: 'string', description: 'IMDB code (required)' },
          ],
        },
      ],
    },
    {
      category: 'TV Series',
      items: [
        {
          method: 'GET',
          path: '/api/v2/list_series.json',
          description: 'List all TV series with filtering and pagination',
          params: [
            { name: 'limit', type: 'int', description: 'Number of results (default: 20)' },
            { name: 'page', type: 'int', description: 'Page number (default: 1)' },
            { name: 'query_term', type: 'string', description: 'Search term' },
            { name: 'genre', type: 'string', description: 'Filter by genre' },
            { name: 'sort_by', type: 'string', description: 'Sort field (title, year, rating)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/series_details.json',
          description: 'Get detailed information about a TV series',
          params: [
            { name: 'series_id', type: 'int', description: 'Series ID (required)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/season_episodes.json',
          description: 'Get episodes for a specific season',
          params: [
            { name: 'series_id', type: 'int', description: 'Series ID (required)' },
            { name: 'season', type: 'int', description: 'Season number (required)' },
          ],
        },
      ],
    },
    {
      category: 'Live TV / Channels',
      items: [
        {
          method: 'GET',
          path: '/api/v2/list_channels.json',
          description: 'List channels with filtering and pagination',
          params: [
            { name: 'limit', type: 'int', description: 'Number of results (default: 50)' },
            { name: 'page', type: 'int', description: 'Page number (default: 1)' },
            { name: 'country', type: 'string', description: 'Filter by country code (e.g. US, AL)' },
            { name: 'category', type: 'string', description: 'Filter by category' },
            { name: 'query', type: 'string', description: 'Search by channel name' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/channel_details.json',
          description: 'Get details for a specific channel',
          params: [
            { name: 'id', type: 'string', description: 'Channel ID (required)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/channel_countries.json',
          description: 'List all countries with channel counts',
          params: [],
        },
        {
          method: 'GET',
          path: '/api/v2/channel_categories.json',
          description: 'List all channel categories with counts',
          params: [],
        },
        {
          method: 'GET',
          path: '/api/v2/channels_by_country.json',
          description: 'Get channels for a specific country',
          params: [
            { name: 'country', type: 'string', description: 'Country code (required)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/channel_epg.json',
          description: 'Get EPG (Electronic Program Guide) for a channel',
          params: [
            { name: 'channel_id', type: 'string', description: 'Channel ID (required)' },
          ],
        },
      ],
    },
    {
      category: 'Subtitles',
      items: [
        {
          method: 'GET',
          path: '/api/v2/subtitles/search',
          description: 'Search subtitles by IMDB ID. Returns from local DB first, falls back to external API.',
          params: [
            { name: 'imdb_id', type: 'string', description: 'IMDB code (required)' },
            { name: 'languages', type: 'string', description: 'Comma-separated language codes (e.g. en,es)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/subtitles/search_by_filename',
          description: 'Search subtitles by release filename',
          params: [
            { name: 'filename', type: 'string', description: 'Release filename (required)' },
            { name: 'languages', type: 'string', description: 'Comma-separated language codes' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/subtitles/stored/{id}',
          description: 'Serve a stored subtitle as VTT content directly from DB',
          params: [
            { name: 'id', type: 'int', description: 'Stored subtitle ID (required, in URL)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/subtitles/download',
          description: 'Download and convert a subtitle from external URL to VTT',
          params: [
            { name: 'url', type: 'string', description: 'Subtitle download URL (required, URL-encoded)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/subtitle_languages',
          description: 'Get list of supported subtitle languages',
          params: [],
        },
      ],
    },
    {
      category: 'Streaming',
      items: [
        {
          method: 'POST',
          path: '/api/v2/stream/start',
          description: 'Start streaming a torrent. Loads the torrent and returns a stream URL. Also extracts embedded subtitles.',
          params: [
            { name: 'hash', type: 'string', description: 'Torrent info hash (required, in body)' },
            { name: 'file_index', type: 'int', description: 'File index (optional, auto-selects largest video)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/stream/status',
          description: 'Get download progress for an active stream',
          params: [
            { name: 'hash', type: 'string', description: 'Torrent info hash (required)' },
          ],
        },
        {
          method: 'POST',
          path: '/api/v2/stream/stop',
          description: 'Stop streaming and remove torrent',
          params: [
            { name: 'hash', type: 'string', description: 'Torrent info hash (required, in body)' },
          ],
        },
        {
          method: 'GET',
          path: '/stream/{info_hash}/{file_index}',
          description: 'Stream a video file from a torrent (supports Range requests)',
          params: [
            { name: 'info_hash', type: 'string', description: 'Torrent info hash (in URL)' },
            { name: 'file_index', type: 'int', description: 'File index in the torrent (in URL)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/torrent_files',
          description: 'List files in a torrent (video, subtitle, etc.)',
          params: [
            { name: 'hash', type: 'string', description: 'Torrent info hash (required)' },
          ],
        },
      ],
    },
    {
      category: 'Search',
      items: [
        {
          method: 'GET',
          path: '/api/v2/search.json',
          description: 'Unified search across movies, series, and channels',
          params: [
            { name: 'query', type: 'string', description: 'Search term (required)' },
          ],
        },
      ],
    },
    {
      category: 'Home & Curated',
      items: [
        {
          method: 'GET',
          path: '/api/v2/home.json',
          description: 'Get home page sections with content',
          params: [],
        },
        {
          method: 'GET',
          path: '/api/v2/curated_lists.json',
          description: 'Get all active curated lists',
          params: [],
        },
        {
          method: 'GET',
          path: '/api/v2/curated_list.json',
          description: 'Get a curated list with movies',
          params: [
            { name: 'slug', type: 'string', description: 'List slug (required)' },
            { name: 'limit', type: 'int', description: 'Number of movies to return' },
          ],
        },
      ],
    },
    {
      category: 'IMDB',
      items: [
        {
          method: 'GET',
          path: '/api/v2/imdb/search',
          description: 'Search IMDB for movies/series',
          params: [
            { name: 'q', type: 'string', description: 'Search query (required)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/imdb/title/{imdbCode}',
          description: 'Get IMDB title details',
          params: [
            { name: 'imdbCode', type: 'string', description: 'IMDB code (in URL, e.g. tt0111161)' },
          ],
        },
        {
          method: 'GET',
          path: '/api/v2/imdb/images/{imdbCode}',
          description: 'Get images for an IMDB title',
          params: [
            { name: 'imdbCode', type: 'string', description: 'IMDB code (in URL)' },
          ],
        },
      ],
    },
    {
      category: 'Sync & Ratings',
      items: [
        {
          method: 'POST',
          path: '/api/v2/sync_movie',
          description: 'Sync a movie by IMDB code (fetches metadata, torrents, subtitles)',
          params: [
            { name: 'imdb_code', type: 'string', description: 'IMDB code (required, in body)' },
          ],
        },
        {
          method: 'POST',
          path: '/api/v2/sync_movies',
          description: 'Batch sync multiple movies',
          params: [
            { name: 'imdb_codes', type: 'string[]', description: 'Array of IMDB codes (required, in body)' },
          ],
        },
        {
          method: 'POST',
          path: '/api/v2/sync_series',
          description: 'Sync a TV series by IMDB code',
          params: [
            { name: 'imdb_code', type: 'string', description: 'IMDB code (required, in body)' },
          ],
        },
        {
          method: 'POST',
          path: '/api/v2/refresh_movie',
          description: 'Refresh metadata and torrents for an existing movie',
          params: [
            { name: 'imdb_code', type: 'string', description: 'IMDB code (required, in body)' },
          ],
        },
        {
          method: 'POST',
          path: '/api/v2/refresh_series',
          description: 'Refresh metadata for an existing series',
          params: [
            { name: 'imdb_code', type: 'string', description: 'IMDB code (required, in body)' },
          ],
        },
        {
          method: 'POST',
          path: '/api/v2/get_ratings',
          description: 'Get ratings for a list of IMDB codes',
          params: [
            { name: 'imdb_codes', type: 'string[]', description: 'Array of IMDB codes (in body)' },
          ],
        },
      ],
    },
    {
      category: 'Analytics',
      items: [
        {
          method: 'POST',
          path: '/api/v2/analytics/view',
          description: 'Record a content view',
          params: [
            { name: 'content_id', type: 'int', description: 'Movie/series ID (in body)' },
            { name: 'content_type', type: 'string', description: 'Type: movie or series (in body)' },
          ],
        },
        {
          method: 'POST',
          path: '/api/v2/analytics/stream/start',
          description: 'Record stream start event',
          params: [],
        },
        {
          method: 'POST',
          path: '/api/v2/analytics/stream/heartbeat',
          description: 'Send stream heartbeat (keep-alive)',
          params: [],
        },
        {
          method: 'POST',
          path: '/api/v2/analytics/stream/end',
          description: 'Record stream end event',
          params: [],
        },
        {
          method: 'GET',
          path: '/api/v2/analytics/top-movies',
          description: 'Get most viewed movies',
          params: [],
        },
      ],
    },
  ];

  let selectedEndpoint: any = $state(null);
</script>

<div class="page-header">
  <h1>API Documentation</h1>
  <div class="header-actions">
    <span class="base-url">Base URL: <code>{window.location.origin}</code></span>
  </div>
</div>

<div class="docs-container">
  <aside class="docs-sidebar">
    {#each endpoints as category}
      <div class="category">
        <h3>{category.category}</h3>
        {#each category.items as endpoint}
          <button
            class="endpoint-link"
            class:active={selectedEndpoint === endpoint}
            onclick={() => selectedEndpoint = endpoint}
          >
            <span class="method {endpoint.method.toLowerCase()}">{endpoint.method}</span>
            <span class="path">{endpoint.path}</span>
          </button>
        {/each}
      </div>
    {/each}
  </aside>

  <main class="docs-content">
    {#if selectedEndpoint}
      <div class="endpoint-detail">
        <div class="endpoint-header">
          <span class="method large {selectedEndpoint.method.toLowerCase()}">{selectedEndpoint.method}</span>
          <code class="path-code">{selectedEndpoint.path}</code>
        </div>

        <p class="description">{selectedEndpoint.description}</p>

        {#if selectedEndpoint.params.length > 0}
          <h3>Parameters</h3>
          <table class="params-table">
            <thead>
              <tr>
                <th>Name</th>
                <th>Type</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              {#each selectedEndpoint.params as param}
                <tr>
                  <td><code>{param.name}</code></td>
                  <td><span class="type">{param.type}</span></td>
                  <td>{param.description}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        {/if}

        <h3>Example Request</h3>
        <pre class="code-block">curl "{window.location.origin}{selectedEndpoint.path}{selectedEndpoint.params.length > 0 ? '?' + selectedEndpoint.params.filter(p => !p.description.includes('in body') && !p.description.includes('in URL')).slice(0, 2).map(p => p.name + '=...').join('&') : ''}"</pre>
      </div>
    {:else}
      <div class="welcome-message">
        <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1">
          <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
          <polyline points="14 2 14 8 20 8"/>
          <line x1="16" y1="13" x2="8" y2="13"/>
          <line x1="16" y1="17" x2="8" y2="17"/>
          <polyline points="10 9 9 9 8 9"/>
        </svg>
        <h2>Select an endpoint</h2>
        <p>Choose an endpoint from the sidebar to view its documentation.</p>
      </div>
    {/if}
  </main>
</div>

<style>
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 24px;
  }

  .page-header h1 {
    font-size: 24px;
    font-weight: 600;
  }

  .base-url {
    font-size: 14px;
    color: var(--text-secondary);
  }

  .base-url code {
    background: var(--bg-tertiary);
    padding: 4px 8px;
    border-radius: 4px;
    font-family: monospace;
  }

  .docs-container {
    display: flex;
    gap: 24px;
    height: calc(100vh - 140px);
  }

  .docs-sidebar {
    width: 320px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    padding: 16px;
    overflow-y: auto;
  }

  .category {
    margin-bottom: 24px;
  }

  .category h3 {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
    padding: 0 8px;
  }

  .endpoint-link {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 8px;
    border: none;
    background: transparent;
    border-radius: 6px;
    cursor: pointer;
    text-align: left;
    transition: background 0.2s;
  }

  .endpoint-link:hover {
    background: var(--bg-tertiary);
  }

  .endpoint-link.active {
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
  }

  .method {
    font-size: 10px;
    font-weight: 600;
    padding: 2px 6px;
    border-radius: 4px;
    text-transform: uppercase;
    flex-shrink: 0;
  }

  .method.get {
    background: rgba(76, 175, 80, 0.2);
    color: #4caf50;
  }

  .method.post {
    background: rgba(33, 150, 243, 0.2);
    color: #2196f3;
  }

  .method.put {
    background: rgba(255, 152, 0, 0.2);
    color: #ff9800;
  }

  .method.delete {
    background: rgba(244, 67, 54, 0.2);
    color: #f44336;
  }

  .method.large {
    font-size: 12px;
    padding: 4px 10px;
  }

  .path {
    font-size: 12px;
    color: var(--text-secondary);
    font-family: monospace;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .docs-content {
    flex: 1;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    padding: 24px;
    overflow-y: auto;
  }

  .welcome-message {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-muted);
    text-align: center;
  }

  .welcome-message svg {
    margin-bottom: 16px;
    opacity: 0.5;
  }

  .welcome-message h2 {
    font-size: 18px;
    margin-bottom: 8px;
    color: var(--text-secondary);
  }

  .endpoint-detail h3 {
    font-size: 14px;
    font-weight: 600;
    margin: 24px 0 12px;
    color: var(--text-secondary);
  }

  .endpoint-header {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }

  .path-code {
    font-size: 16px;
    font-family: monospace;
    color: var(--text-primary);
  }

  .description {
    color: var(--text-secondary);
    font-size: 14px;
    line-height: 1.6;
  }

  .params-table {
    width: 100%;
    border-collapse: collapse;
  }

  .params-table th,
  .params-table td {
    padding: 12px;
    text-align: left;
    border-bottom: 1px solid var(--border-color);
  }

  .params-table th {
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
  }

  .params-table td code {
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 13px;
  }

  .type {
    font-size: 12px;
    color: var(--accent-blue);
  }

  .code-block {
    background: var(--bg-tertiary);
    padding: 16px;
    border-radius: 8px;
    font-family: monospace;
    font-size: 13px;
    overflow-x: auto;
    color: var(--text-secondary);
  }
</style>
