<script lang="ts">
  const endpoints = [
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
      ],
    },
    {
      category: 'Curated Lists',
      items: [
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
      category: 'Streaming',
      items: [
        {
          method: 'GET',
          path: '/stream/{info_hash}/{file_index}',
          description: 'Stream a video file from a torrent',
          params: [
            { name: 'info_hash', type: 'string', description: 'Torrent info hash' },
            { name: 'file_index', type: 'int', description: 'File index in the torrent' },
          ],
        },
      ],
    },
    {
      category: 'Stremio Addon',
      items: [
        {
          method: 'GET',
          path: '/manifest.json',
          description: 'Stremio addon manifest',
          params: [],
        },
        {
          method: 'GET',
          path: '/catalog/{type}/{id}.json',
          description: 'Stremio catalog endpoint',
          params: [
            { name: 'type', type: 'string', description: 'Content type (movie, series)' },
            { name: 'id', type: 'string', description: 'Catalog ID' },
          ],
        },
        {
          method: 'GET',
          path: '/stream/{type}/{id}.json',
          description: 'Stremio stream endpoint',
          params: [
            { name: 'type', type: 'string', description: 'Content type' },
            { name: 'id', type: 'string', description: 'IMDb ID' },
          ],
        },
      ],
    },
  ];

  let selectedEndpoint: any = null;
</script>

<div class="page-header">
  <h1>API Documentation</h1>
  <div class="header-actions">
    <span class="base-url">Base URL: <code>http://localhost:8080</code></span>
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
            on:click={() => selectedEndpoint = endpoint}
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
        <pre class="code-block">curl "http://localhost:8080{selectedEndpoint.path}{selectedEndpoint.params.length > 0 ? '?' + selectedEndpoint.params.slice(0, 2).map(p => p.name + '=...').join('&') : ''}"</pre>
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
