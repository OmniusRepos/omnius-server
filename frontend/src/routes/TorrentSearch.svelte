<script lang="ts">
  interface TorrentResult {
    Title: string;
    Hash: string;
    MagnetURL: string;
    Quality: string;
    Type: string;
    Seeds: number;
    Peers: number;
    Size: string;
    SizeBytes: number;
    Source: string;
    Season?: number;
    Episode?: number;
  }

  let query = '';
  let searchType: 'movie' | 'series' = 'movie';
  let year = '';
  let imdb = '';
  let results: TorrentResult[] = [];
  let loading = false;
  let searched = false;
  let resultCount = 0;

  // Source filter
  let sourceFilter: string | null = null;
  $: sources = [...new Set(results.map(r => r.Source))].sort();
  $: filteredResults = sourceFilter ? results.filter(r => r.Source === sourceFilter) : results;

  async function handleSearch() {
    if (!query.trim() && !imdb.trim()) return;
    loading = true;
    searched = false;
    results = [];

    try {
      const params = new URLSearchParams();
      if (query.trim()) params.set('query', query.trim());
      if (searchType) params.set('type', searchType);
      if (year.trim()) params.set('year', year.trim());
      if (imdb.trim()) params.set('imdb', imdb.trim());

      const res = await fetch(`/admin/api/torrents/search?${params}`);
      if (res.ok) {
        const data = await res.json();
        results = data.results || [];
        resultCount = data.count || 0;
      }
    } catch (err) {
      console.error('Search failed:', err);
    } finally {
      loading = false;
      searched = true;
      sourceFilter = null;
    }
  }

  function copyHash(hash: string) {
    navigator.clipboard.writeText(hash);
  }

  function copyMagnet(magnet: string) {
    navigator.clipboard.writeText(magnet);
  }

  function getSourceColor(source: string): string {
    const colors: Record<string, string> = {
      'YTS': '#2ecc71',
      'EZTV': '#3498db',
      '1337x': '#e74c3c',
      'TPB': '#f39c12',
    };
    return colors[source] || '#8b5cf6';
  }
</script>

<div class="page">
  <header class="page-header">
    <h1>Torrent Search</h1>
    <p class="page-subtitle">Search across YTS, EZTV, 1337x, and ThePirateBay</p>
  </header>

  <div class="search-card">
    <form on:submit|preventDefault={handleSearch}>
      <div class="search-row">
        <div class="search-type-toggle">
          <button type="button" class="type-btn" class:active={searchType === 'movie'} on:click={() => searchType = 'movie'}>
            Movies
          </button>
          <button type="button" class="type-btn" class:active={searchType === 'series'} on:click={() => searchType = 'series'}>
            TV Series
          </button>
        </div>

        <input
          type="text"
          class="search-input"
          bind:value={query}
          placeholder="Search title..."
        />

        {#if searchType === 'movie'}
          <input
            type="text"
            class="search-input search-input-sm"
            bind:value={year}
            placeholder="Year"
          />
        {/if}

        <input
          type="text"
          class="search-input search-input-sm"
          bind:value={imdb}
          placeholder="IMDB code"
        />

        <button type="submit" class="btn btn-primary" disabled={loading}>
          {loading ? 'Searching...' : 'Search'}
        </button>
      </div>
    </form>
  </div>

  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Searching all providers...</p>
    </div>
  {:else if searched}
    <div class="results-header">
      <div class="results-count">
        {resultCount} result{resultCount !== 1 ? 's' : ''} found
      </div>

      {#if sources.length > 1}
        <div class="source-filters">
          <button class="source-btn" class:active={sourceFilter === null} on:click={() => sourceFilter = null}>
            All ({results.length})
          </button>
          {#each sources as source}
            <button
              class="source-btn"
              class:active={sourceFilter === source}
              on:click={() => sourceFilter = sourceFilter === source ? null : source}
              style="--source-color: {getSourceColor(source)}"
            >
              {source} ({results.filter(r => r.Source === source).length})
            </button>
          {/each}
        </div>
      {/if}
    </div>

    {#if filteredResults.length === 0}
      <div class="empty-state">
        <p>No torrents found. Try a different search term.</p>
      </div>
    {:else}
      <div class="results-table">
        <div class="table-header">
          <span class="col-source">Source</span>
          {#if searchType === 'series'}
            <span class="col-episode">Episode</span>
          {/if}
          <span class="col-quality">Quality</span>
          <span class="col-title">Title</span>
          <span class="col-size">Size</span>
          <span class="col-seeds">Seeds</span>
          <span class="col-peers">Peers</span>
          <span class="col-actions">Actions</span>
        </div>

        {#each filteredResults as torrent (torrent.Hash)}
          <div class="table-row">
            <span class="col-source">
              <span class="source-badge" style="background: {getSourceColor(torrent.Source)}">{torrent.Source}</span>
            </span>
            {#if searchType === 'series'}
              <span class="col-episode">
                {#if torrent.Season && torrent.Episode}
                  S{String(torrent.Season).padStart(2, '0')}E{String(torrent.Episode).padStart(2, '0')}
                {:else if torrent.Season}
                  S{String(torrent.Season).padStart(2, '0')}
                {:else}
                  -
                {/if}
              </span>
            {/if}
            <span class="col-quality">
              <span class="quality-badge badge-{torrent.Quality}">{torrent.Quality}</span>
              {#if torrent.Type && torrent.Type !== 'web'}
                <span class="type-label">{torrent.Type}</span>
              {/if}
            </span>
            <span class="col-title" title={torrent.Title}>{torrent.Title}</span>
            <span class="col-size">{torrent.Size}</span>
            <span class="col-seeds seeds">{torrent.Seeds}</span>
            <span class="col-peers">{torrent.Peers}</span>
            <span class="col-actions">
              <button class="btn btn-xs btn-secondary" on:click={() => copyHash(torrent.Hash)} title="Copy hash">
                Hash
              </button>
              <button class="btn btn-xs btn-secondary" on:click={() => copyMagnet(torrent.MagnetURL || `magnet:?xt=urn:btih:${torrent.Hash}`)} title="Copy magnet">
                Magnet
              </button>
            </span>
          </div>
        {/each}
      </div>
    {/if}
  {/if}
</div>

<style>
  .page {
    padding: 0;
  }

  .page-header {
    margin-bottom: 24px;
  }

  .page-header h1 {
    font-size: 24px;
    font-weight: 700;
    margin: 0 0 4px 0;
  }

  .page-subtitle {
    font-size: 14px;
    color: var(--text-muted);
    margin: 0;
  }

  .search-card {
    background: var(--bg-secondary);
    border-radius: 8px;
    padding: 20px;
    margin-bottom: 24px;
  }

  .search-row {
    display: flex;
    gap: 12px;
    align-items: center;
    flex-wrap: wrap;
  }

  .search-type-toggle {
    display: flex;
    background: var(--bg-tertiary);
    border-radius: 6px;
    overflow: hidden;
    flex-shrink: 0;
  }

  .type-btn {
    padding: 10px 16px;
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: 14px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .type-btn.active {
    background: var(--accent-red);
    color: white;
  }

  .type-btn:hover:not(.active) {
    color: var(--text-primary);
  }

  .search-input {
    flex: 1;
    min-width: 200px;
    padding: 10px 14px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 14px;
  }

  .search-input:focus {
    outline: none;
    border-color: var(--accent-blue);
  }

  .search-input-sm {
    flex: 0;
    min-width: 100px;
    width: 120px;
  }

  /* Results */
  .results-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
    flex-wrap: wrap;
    gap: 12px;
  }

  .results-count {
    font-size: 14px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  .source-filters {
    display: flex;
    gap: 6px;
    flex-wrap: wrap;
  }

  .source-btn {
    padding: 4px 12px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 16px;
    color: var(--text-muted);
    font-size: 12px;
    cursor: pointer;
    transition: all 0.15s;
  }

  .source-btn:hover {
    border-color: var(--source-color, var(--accent-blue));
    color: var(--source-color, var(--accent-blue));
  }

  .source-btn.active {
    background: var(--source-color, var(--accent-blue));
    border-color: var(--source-color, var(--accent-blue));
    color: white;
  }

  /* Table */
  .results-table {
    background: var(--bg-secondary);
    border-radius: 8px;
    overflow: hidden;
  }

  .table-header {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: var(--bg-tertiary);
    font-size: 12px;
    font-weight: 600;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .table-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 16px;
    border-bottom: 1px solid var(--border-color);
    font-size: 13px;
    transition: background 0.1s;
  }

  .table-row:last-child {
    border-bottom: none;
  }

  .table-row:hover {
    background: var(--bg-tertiary);
  }

  .col-source { width: 70px; flex-shrink: 0; }
  .col-episode { width: 70px; flex-shrink: 0; font-weight: 600; color: var(--text-secondary); }
  .col-quality { width: 110px; flex-shrink: 0; display: flex; align-items: center; gap: 6px; }
  .col-title { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--text-secondary); }
  .col-size { width: 80px; flex-shrink: 0; color: var(--text-secondary); }
  .col-seeds { width: 60px; flex-shrink: 0; }
  .col-peers { width: 60px; flex-shrink: 0; color: var(--text-muted); }
  .col-actions { width: 120px; flex-shrink: 0; display: flex; gap: 4px; }

  .seeds {
    color: #22c55e;
    font-weight: 600;
  }

  .source-badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 700;
    color: white;
  }

  .quality-badge {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
    color: white;
  }

  .badge-720p { background: #3b82f6; }
  .badge-1080p { background: #8b5cf6; }
  .badge-2160p { background: #f59e0b; }
  .badge-480p { background: #6b7280; }

  .type-label {
    font-size: 11px;
    color: var(--text-muted);
  }

  .btn-xs {
    padding: 4px 8px;
    font-size: 11px;
    line-height: 1;
  }

  .loading-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 16px;
    padding: 60px 0;
    color: var(--text-muted);
  }

  .spinner {
    width: 32px;
    height: 32px;
    border: 3px solid var(--border-color);
    border-top-color: var(--accent-red);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .empty-state {
    text-align: center;
    padding: 48px;
    color: var(--text-muted);
  }
</style>
