<script lang="ts">
  import { onMount } from 'svelte';
  import { link } from 'svelte-spa-router';
  import { getSeries, deleteSeries, updateSeries, getSeriesByIMDB, type Series } from '../lib/api/client';
  import Modal from '../lib/components/Modal.svelte';

  let series: Series[] = [];
  let total = 0;
  let loading = true;
  let search = '';
  let page = 1;
  let limit = 20;

  const perPageOptions = [10, 20, 50, 100];

  $: totalPages = Math.ceil(total / limit);
  $: startItem = total > 0 ? (page - 1) * limit + 1 : 0;
  $: endItem = Math.min(page * limit, total);

  // Modal states
  let showAddModal = false;
  let showEditModal = false;
  let showDeleteModal = false;
  let selectedSeries: Series | null = null;

  // Form data
  let seriesForm = {
    imdb_code: '',
    title: '',
    year: new Date().getFullYear(),
    rating: 0,
    runtime: 0,
    genres: '',
    summary: '',
    poster_image: '',
    background_image: '',
    total_seasons: 1,
    status: 'Continuing',
    network: '',
  };

  onMount(async () => {
    await loadSeries();
  });

  async function loadSeries() {
    loading = true;
    try {
      const result = await getSeries({ page, limit, search: search || undefined });
      series = result.series;
      total = result.total;
    } catch (err) {
      console.error('Failed to load series:', err);
    } finally {
      loading = false;
    }
  }

  function handleSearch() {
    page = 1;
    loadSeries();
  }

  function goToPage(p: number) {
    if (p >= 1 && p <= totalPages) {
      page = p;
      loadSeries();
    }
  }

  function changePerPage(newLimit: number) {
    limit = newLimit;
    page = 1;
    loadSeries();
  }

  function getPageNumbers(): number[] {
    const pages: number[] = [];
    const maxVisible = 5;
    let start = Math.max(1, page - Math.floor(maxVisible / 2));
    let end = Math.min(totalPages, start + maxVisible - 1);

    if (end - start + 1 < maxVisible) {
      start = Math.max(1, end - maxVisible + 1);
    }

    for (let i = start; i <= end; i++) {
      pages.push(i);
    }
    return pages;
  }

  // Modal functions
  function openAddModal() {
    seriesForm = {
      imdb_code: '',
      title: '',
      year: new Date().getFullYear(),
      rating: 0,
      runtime: 0,
      genres: '',
      summary: '',
      poster_image: '',
      background_image: '',
      total_seasons: 1,
      status: 'Continuing',
      network: '',
    };
    showAddModal = true;
  }

  function openEditModal(s: Series) {
    selectedSeries = s;
    seriesForm = {
      imdb_code: s.imdb_code || '',
      title: s.title,
      year: s.year,
      rating: s.rating || 0,
      runtime: s.runtime || 0,
      genres: s.genres?.join(', ') || '',
      summary: s.summary || '',
      poster_image: s.poster_image || '',
      background_image: s.background_image || '',
      total_seasons: s.total_seasons || 1,
      status: s.status || 'Continuing',
      network: s.network || '',
    };
    showEditModal = true;
  }

  function openDeleteModal(s: Series) {
    selectedSeries = s;
    showDeleteModal = true;
  }

  async function handleAddSeries() {
    try {
      const res = await fetch('/admin/series', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'Accept': 'application/json'
        },
        body: new URLSearchParams({
          imdb_code: seriesForm.imdb_code,
          title: seriesForm.title,
          year: String(seriesForm.year),
          rating: String(seriesForm.rating),
          runtime: String(seriesForm.runtime),
          genres: seriesForm.genres,
          summary: seriesForm.summary,
          poster_image: seriesForm.poster_image,
          background_image: seriesForm.background_image,
          total_seasons: String(seriesForm.total_seasons),
          status: seriesForm.status,
          network: seriesForm.network,
        }),
      });
      if (res.ok) {
        showAddModal = false;
        page = 1; // Reset to first page to see new series at top
        loadSeries();
      }
    } catch (err) {
      console.error('Failed to add series:', err);
    }
  }

  async function handleEditSeries() {
    if (!selectedSeries) return;
    try {
      await updateSeries(selectedSeries.id, {
        imdb_code: seriesForm.imdb_code,
        title: seriesForm.title,
        year: seriesForm.year,
        rating: seriesForm.rating,
        runtime: seriesForm.runtime,
        genres: seriesForm.genres,
        summary: seriesForm.summary,
        poster_image: seriesForm.poster_image,
        background_image: seriesForm.background_image,
        total_seasons: seriesForm.total_seasons,
        status: seriesForm.status,
        network: seriesForm.network,
      });
      showEditModal = false;
      loadSeries();
    } catch (err) {
      console.error('Failed to update series:', err);
    }
  }

  async function handleDeleteSeries() {
    if (!selectedSeries) return;
    try {
      await deleteSeries(selectedSeries.id);
      showDeleteModal = false;
      loadSeries();
    } catch (err) {
      console.error('Failed to delete series:', err);
    }
  }

  let fetchingImdb = false;
  let imdbError: string | null = null;
  let titleSuggestions: Array<{id: string, title: string, year: string, poster: string, inLibrary?: boolean, seriesId?: number}> = [];
  let showSuggestions = false;
  let searchTimeout: ReturnType<typeof setTimeout>;

  async function searchTitles(query: string) {
    if (query.length < 2) {
      titleSuggestions = [];
      return;
    }

    clearTimeout(searchTimeout);
    searchTimeout = setTimeout(async () => {
      try {
        const queryLower = query.toLowerCase();

        // Search both local DB and IMDB in parallel
        const [localRes, imdbRes] = await Promise.all([
          fetch(`/api/v2/list_series.json?query_term=${encodeURIComponent(query)}&limit=5`),
          fetch(`/admin/api/imdb/search?query=${encodeURIComponent(query)}`)
        ]);

        const imdbSuggestions: typeof titleSuggestions = [];
        const librarySuggestions: typeof titleSuggestions = [];
        const libraryImdbIds = new Set<string>();

        // Collect local results (to filter out from IMDB and show at end)
        if (localRes.ok) {
          const localData = await localRes.json();
          const localSeries = localData.data?.series || [];
          for (const s of localSeries) {
            if (s.imdb_code) libraryImdbIds.add(s.imdb_code);
            librarySuggestions.push({
              id: s.imdb_code || `local-${s.id}`,
              title: s.title,
              year: String(s.year),
              poster: s.poster_image || '',
              inLibrary: true,
              seriesId: s.id,
            });
          }
        }

        // Add IMDB results first (only TV series types, skip if already in library)
        if (imdbRes.ok) {
          const imdbData = await imdbRes.json();
          for (const r of (imdbData.titles || []).slice(0, 10)) {
            // Only include TV series types
            const titleType = (r.type || r.titleType || '').toLowerCase();
            if (!titleType.includes('series') && !titleType.includes('tv')) continue;

            // Skip if already in library
            if (libraryImdbIds.has(r.id)) continue;

            imdbSuggestions.push({
              id: r.id,
              title: r.primaryTitle,
              year: r.startYear ? String(r.startYear) : '',
              poster: r.primaryImage?.url || '',
              inLibrary: false,
            });
          }
        }

        // Sort IMDB results: exact matches first, then starts-with, then contains
        imdbSuggestions.sort((a, b) => {
          const aLower = a.title.toLowerCase();
          const bLower = b.title.toLowerCase();
          const aExact = aLower === queryLower;
          const bExact = bLower === queryLower;
          const aStarts = aLower.startsWith(queryLower);
          const bStarts = bLower.startsWith(queryLower);

          if (aExact && !bExact) return -1;
          if (bExact && !aExact) return 1;
          if (aStarts && !bStarts) return -1;
          if (bStarts && !aStarts) return 1;
          return 0;
        });

        // IMDB results first, then library results at the end
        titleSuggestions = [...imdbSuggestions, ...librarySuggestions].slice(0, 10);
        showSuggestions = titleSuggestions.length > 0;
      } catch (err) {
        console.error('Search failed:', err);
      }
    }, 300);
  }

  async function selectSuggestion(suggestion: {id: string, title: string, year: string, inLibrary?: boolean, seriesId?: number}) {
    showSuggestions = false;
    titleSuggestions = [];

    // If it's already in library, open edit modal directly
    if (suggestion.inLibrary && suggestion.seriesId) {
      const s = series.find(ser => ser.id === suggestion.seriesId);
      if (s) {
        showAddModal = false;
        openEditModal(s);
        return;
      }
    }

    // Check if series exists by IMDB code (in case it wasn't in local search results)
    if (suggestion.id.startsWith('tt')) {
      try {
        const result = await getSeriesByIMDB(suggestion.id);
        if (result.exists && result.series) {
          showAddModal = false;
          openEditModal(result.series);
          return;
        }
      } catch (err) {
        console.error('Failed to check existing series:', err);
      }
    }

    // Series doesn't exist - continue with add flow
    seriesForm.imdb_code = suggestion.id;
    seriesForm.title = suggestion.title;
    seriesForm.year = parseInt(suggestion.year) || new Date().getFullYear();
    // Auto-fetch full details from IMDB
    fetchFromImdb();
  }

  async function fetchFromImdb() {
    if (!seriesForm.imdb_code) return;
    fetchingImdb = true;
    imdbError = null;
    try {
      const res = await fetch(`/admin/api/imdb/title/${seriesForm.imdb_code}`);
      if (res.ok) {
        const data = await res.json();
        console.log('IMDB data:', data);

        // Check if it's a movie (reject movies)
        const titleType = (data.type || data.titleType || '').toLowerCase();
        if (titleType === 'movie' || titleType === 'short' || titleType === 'video') {
          imdbError = `"${data.primaryTitle || data.title}" is a Movie. Add it in the Movies section instead.`;
          fetchingImdb = false;
          return;
        }

        // Clear any previous error
        imdbError = null;

        // Basic info
        seriesForm.title = data.primaryTitle || seriesForm.title;
        seriesForm.year = data.startYear || seriesForm.year;
        seriesForm.rating = data.rating?.aggregateRating || seriesForm.rating;
        seriesForm.runtime = data.runtimeSeconds ? Math.round(data.runtimeSeconds / 60) : seriesForm.runtime;
        seriesForm.genres = data.genres?.join(', ') || seriesForm.genres;

        // Description
        seriesForm.summary = data.plot || seriesForm.summary;

        // Images
        seriesForm.poster_image = data.primaryImage?.url || seriesForm.poster_image;

        // TV-specific fields
        if (data.totalSeasons) {
          seriesForm.total_seasons = data.totalSeasons;
        }
        if (data.endYear) {
          seriesForm.status = 'Ended';
        } else {
          seriesForm.status = 'Continuing';
        }
      }
    } catch (err) {
      console.error('Failed to fetch from IMDB:', err);
      imdbError = 'Failed to fetch from IMDB';
    } finally {
      fetchingImdb = false;
    }
  }
</script>

{#snippet paginationBar()}
  <div class="pagination">
    <div class="pagination-left">
      <span class="pagination-info">
        Showing {startItem}-{endItem} of {total.toLocaleString()}
      </span>
      <div class="per-page">
        <span>per page:</span>
        <select bind:value={limit} on:change={() => changePerPage(limit)}>
          {#each perPageOptions as opt}
            <option value={opt}>{opt}</option>
          {/each}
        </select>
      </div>
    </div>
    {#if totalPages > 1}
      <div class="pagination-controls">
        <button
          class="pagination-btn"
          disabled={page === 1}
          on:click={() => goToPage(1)}
          title="First page"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="11 17 6 12 11 7"/>
            <polyline points="18 17 13 12 18 7"/>
          </svg>
        </button>
        <button
          class="pagination-btn"
          disabled={page === 1}
          on:click={() => goToPage(page - 1)}
          title="Previous page"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="15 18 9 12 15 6"/>
          </svg>
        </button>

        {#each getPageNumbers() as p}
          <button
            class="pagination-btn"
            class:active={p === page}
            on:click={() => goToPage(p)}
          >
            {p}
          </button>
        {/each}

        <button
          class="pagination-btn"
          disabled={page === totalPages}
          on:click={() => goToPage(page + 1)}
          title="Next page"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="9 18 15 12 9 6"/>
          </svg>
        </button>
        <button
          class="pagination-btn"
          disabled={page === totalPages}
          on:click={() => goToPage(totalPages)}
          title="Last page"
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <polyline points="13 17 18 12 13 7"/>
            <polyline points="6 17 11 12 6 7"/>
          </svg>
        </button>
      </div>
    {/if}
  </div>
{/snippet}

<div class="tvshows-page">
  <header class="page-header">
    <h1 class="page-title">TV SHOWS</h1>
    <div class="page-actions">
      <div class="search-box">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <path d="m21 21-4.35-4.35"/>
        </svg>
        <input
          type="text"
          placeholder="Search series..."
          bind:value={search}
          on:keydown={(e) => e.key === 'Enter' && handleSearch()}
        />
      </div>
      <button class="btn btn-primary" on:click={openAddModal}>ADD</button>
    </div>
  </header>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if series.length === 0}
    <div class="empty-state">
      <p>No TV shows found</p>
    </div>
  {:else}
    {@render paginationBar()}

    <div class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th>Poster</th>
            <th>Title</th>
            <th>Year</th>
            <th>Seasons</th>
            <th>Rating</th>
            <th>Status</th>
            <th style="text-align: right;">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each series as show}
            <tr>
              <td>
                {#if show.poster_image}
                  <img src={show.poster_image} alt={show.title} class="poster" />
                {:else}
                  <div class="poster poster-placeholder">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                      <rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/>
                      <line x1="7" y1="2" x2="7" y2="22"/>
                      <line x1="17" y1="2" x2="17" y2="22"/>
                      <line x1="2" y1="12" x2="22" y2="12"/>
                    </svg>
                  </div>
                {/if}
              </td>
              <td>
                <a href="/tvshows/{show.id}" use:link class="series-title">{show.title}</a>
              </td>
              <td>{show.year}</td>
              <td>{show.total_seasons}</td>
              <td>
                {#if show.rating}
                  <div class="rating-badge">
                    <span class="score">{show.rating.toFixed(1)}</span>
                  </div>
                {:else}
                  <span class="text-muted">-</span>
                {/if}
              </td>
              <td>
                <span class="status status-{show.status?.toLowerCase()}">{show.status}</span>
              </td>
              <td>
                <div class="actions">
                  <button class="btn btn-sm btn-secondary" on:click={() => openEditModal(show)}>EDIT</button>
                  <button class="btn btn-sm btn-danger" on:click={() => openDeleteModal(show)}>DELETE</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>

    {@render paginationBar()}
  {/if}
</div>

<!-- Add Series Modal -->
<Modal bind:open={showAddModal} title="Add TV Series" size="lg" on:close={() => showAddModal = false}>
  <form on:submit|preventDefault={handleAddSeries}>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="imdb_code">IMDB Code</label>
        <div class="input-with-button">
          <input type="text" id="imdb_code" class="form-input" bind:value={seriesForm.imdb_code} placeholder="tt1234567" />
          <button type="button" class="btn btn-sm btn-secondary" on:click={fetchFromImdb} disabled={fetchingImdb || !seriesForm.imdb_code}>
            {fetchingImdb ? 'Fetching...' : 'Fetch'}
          </button>
        </div>
        {#if imdbError}
          <p class="form-error">{imdbError}</p>
        {/if}
      </div>
      <div class="form-group">
        <label class="form-label" for="title">Title *</label>
        <div class="autocomplete-wrapper">
          <input
            type="text"
            id="title"
            class="form-input"
            bind:value={seriesForm.title}
            on:input={(e) => searchTitles(e.currentTarget.value)}
            on:focus={() => showSuggestions = titleSuggestions.length > 0}
            on:blur={() => setTimeout(() => showSuggestions = false, 200)}
            autocomplete="off"
            required
          />
          {#if showSuggestions && titleSuggestions.length > 0}
            <div class="suggestions-dropdown">
              {#each titleSuggestions as suggestion}
                <button
                  type="button"
                  class="suggestion-item"
                  class:in-library={suggestion.inLibrary}
                  on:mousedown={() => selectSuggestion(suggestion)}
                >
                  {#if suggestion.poster}
                    <img src={suggestion.poster} alt="" class="suggestion-poster" />
                  {:else}
                    <div class="suggestion-poster suggestion-poster-empty"></div>
                  {/if}
                  <div class="suggestion-info">
                    <span class="suggestion-title">
                      {suggestion.title}
                      {#if suggestion.inLibrary}
                        <span class="library-badge">IN LIBRARY</span>
                      {/if}
                    </span>
                    <span class="suggestion-year">{suggestion.year} {suggestion.inLibrary ? '• Click to edit' : ''}</span>
                  </div>
                </button>
              {/each}
            </div>
          {/if}
        </div>
      </div>
    </div>
    <div class="form-row form-row-4">
      <div class="form-group">
        <label class="form-label" for="year">Year</label>
        <input type="number" id="year" class="form-input" bind:value={seriesForm.year} />
      </div>
      <div class="form-group">
        <label class="form-label" for="rating">Rating</label>
        <input type="number" id="rating" class="form-input" bind:value={seriesForm.rating} step="0.1" min="0" max="10" />
      </div>
      <div class="form-group">
        <label class="form-label" for="runtime">Runtime (min)</label>
        <input type="number" id="runtime" class="form-input" bind:value={seriesForm.runtime} min="0" />
      </div>
      <div class="form-group">
        <label class="form-label" for="total_seasons">Seasons</label>
        <input type="number" id="total_seasons" class="form-input" bind:value={seriesForm.total_seasons} min="1" />
      </div>
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="genres">Genres</label>
        <input type="text" id="genres" class="form-input" bind:value={seriesForm.genres} placeholder="Drama, Thriller, Crime" />
      </div>
      <div class="form-group">
        <label class="form-label" for="status">Status</label>
        <select id="status" class="form-input" bind:value={seriesForm.status}>
          <option value="Continuing">Continuing</option>
          <option value="Ended">Ended</option>
        </select>
      </div>
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="poster">Poster Image URL</label>
        <input type="text" id="poster" class="form-input" bind:value={seriesForm.poster_image} placeholder="https://..." />
      </div>
      <div class="form-group">
        <label class="form-label" for="background">Background Image URL</label>
        <input type="text" id="background" class="form-input" bind:value={seriesForm.background_image} placeholder="https://..." />
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="network">Network</label>
      <input type="text" id="network" class="form-input" bind:value={seriesForm.network} placeholder="HBO, Netflix, AMC..." />
    </div>
    <div class="form-group">
      <label class="form-label" for="summary">Summary</label>
      <textarea id="summary" class="form-input form-textarea" bind:value={seriesForm.summary} rows="3"></textarea>
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showAddModal = false}>Cancel</button>
    <button class="btn btn-primary" on:click={handleAddSeries}>Add Series</button>
  </svelte:fragment>
</Modal>

<!-- Edit Series Modal -->
<Modal bind:open={showEditModal} title="Edit TV Series" size="lg" on:close={() => showEditModal = false}>
  <form on:submit|preventDefault={handleEditSeries}>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="edit_imdb_code">IMDB Code</label>
        <div class="input-with-button">
          <input type="text" id="edit_imdb_code" class="form-input" bind:value={seriesForm.imdb_code} placeholder="tt1234567" />
          <button type="button" class="btn btn-sm btn-secondary" on:click={fetchFromImdb} disabled={fetchingImdb || !seriesForm.imdb_code}>
            {fetchingImdb ? 'Updating...' : 'Update'}
          </button>
        </div>
        {#if imdbError}
          <p class="form-error">{imdbError}</p>
        {/if}
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_title">Title *</label>
        <input type="text" id="edit_title" class="form-input" bind:value={seriesForm.title} required />
      </div>
    </div>
    <div class="form-row form-row-4">
      <div class="form-group">
        <label class="form-label" for="edit_year">Year</label>
        <input type="number" id="edit_year" class="form-input" bind:value={seriesForm.year} />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_rating">Rating</label>
        <input type="number" id="edit_rating" class="form-input" bind:value={seriesForm.rating} step="0.1" min="0" max="10" />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_runtime">Runtime (min)</label>
        <input type="number" id="edit_runtime" class="form-input" bind:value={seriesForm.runtime} min="0" />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_total_seasons">Seasons</label>
        <input type="number" id="edit_total_seasons" class="form-input" bind:value={seriesForm.total_seasons} min="1" />
      </div>
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="edit_genres">Genres</label>
        <input type="text" id="edit_genres" class="form-input" bind:value={seriesForm.genres} placeholder="Drama, Thriller" />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_status">Status</label>
        <select id="edit_status" class="form-input" bind:value={seriesForm.status}>
          <option value="Continuing">Continuing</option>
          <option value="Ended">Ended</option>
        </select>
      </div>
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="edit_poster">Poster Image URL</label>
        <input type="text" id="edit_poster" class="form-input" bind:value={seriesForm.poster_image} placeholder="https://..." />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_background">Background Image URL</label>
        <input type="text" id="edit_background" class="form-input" bind:value={seriesForm.background_image} placeholder="https://..." />
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="edit_network">Network</label>
      <input type="text" id="edit_network" class="form-input" bind:value={seriesForm.network} placeholder="HBO, Netflix, AMC..." />
    </div>
    <div class="form-group">
      <label class="form-label" for="edit_summary">Summary</label>
      <textarea id="edit_summary" class="form-input form-textarea" bind:value={seriesForm.summary} rows="3"></textarea>
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showEditModal = false}>Cancel</button>
    <button class="btn btn-primary" on:click={handleEditSeries}>Save Changes</button>
  </svelte:fragment>
</Modal>

<!-- Delete Confirmation Modal -->
<Modal bind:open={showDeleteModal} title="Delete Series" size="sm" on:close={() => showDeleteModal = false}>
  <p class="delete-warning">
    Are you sure you want to delete <strong>{selectedSeries?.title}</strong>?
  </p>
  <p class="text-muted">This action cannot be undone. All episodes and torrents will also be removed.</p>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger" on:click={handleDeleteSeries}>Delete</button>
  </svelte:fragment>
</Modal>

<style>
  .status {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    text-transform: capitalize;
  }

  .status-continuing,
  .status-ongoing {
    background: rgba(46, 160, 67, 0.2);
    color: var(--accent-green);
  }

  .status-ended {
    background: rgba(139, 148, 158, 0.2);
    color: var(--text-muted);
  }

  .form-error {
    color: #ef4444;
    font-size: 0.85rem;
    margin-top: 0.5rem;
    margin-bottom: 0;
  }

  .poster-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-tertiary);
    color: var(--text-muted);
  }

  .series-title {
    font-weight: 500;
    color: var(--text-primary);
    text-decoration: none;
  }

  .series-title:hover {
    color: var(--accent-blue);
  }

  .rating-badge {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 2px 8px;
    background: rgba(245, 158, 11, 0.2);
    border-radius: 4px;
  }

  .rating-badge .score {
    font-weight: 600;
    color: #f59e0b;
  }

  .pagination {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 12px 0;
  }

  .pagination-left {
    display: flex;
    align-items: center;
    gap: 16px;
  }

  .pagination-info {
    color: var(--text-muted);
    font-size: 14px;
  }

  .per-page {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    color: var(--text-muted);
  }

  .per-page select {
    padding: 4px 8px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 14px;
    cursor: pointer;
  }

  .pagination-controls {
    display: flex;
    gap: 4px;
  }

  .pagination-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    min-width: 36px;
    height: 36px;
    padding: 0 8px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-secondary);
    font-size: 14px;
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .pagination-btn:hover:not(:disabled) {
    background: var(--bg-tertiary);
    color: var(--text-primary);
    border-color: var(--text-muted);
  }

  .pagination-btn.active {
    background: var(--accent-red);
    border-color: var(--accent-red);
    color: white;
  }

  .pagination-btn:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }

  /* Form styles */
  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .form-row-4 {
    grid-template-columns: repeat(4, 1fr);
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-label {
    display: block;
    margin-bottom: 6px;
    font-size: 14px;
    font-weight: 500;
    color: var(--text-secondary);
  }

  .form-input {
    width: 100%;
    padding: 10px 12px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 14px;
  }

  .form-input:focus {
    outline: none;
    border-color: var(--accent-blue);
  }

  .form-textarea {
    resize: vertical;
    min-height: 80px;
  }

  .delete-warning {
    margin-bottom: 12px;
  }

  .input-with-button {
    display: flex;
    gap: 8px;
  }

  .input-with-button .form-input {
    flex: 1;
  }

  .input-with-button .btn {
    white-space: nowrap;
  }

  /* Autocomplete */
  .autocomplete-wrapper {
    position: relative;
  }

  .suggestions-dropdown {
    position: absolute;
    top: 100%;
    left: 0;
    right: 0;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    margin-top: 4px;
    max-height: 320px;
    overflow-y: auto;
    z-index: 100;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
  }

  .suggestion-item {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 10px 12px;
    border: none;
    background: none;
    color: var(--text-primary);
    text-align: left;
    cursor: pointer;
    transition: background var(--transition-fast);
  }

  .suggestion-item:hover {
    background: var(--bg-secondary);
  }

  .suggestion-item.in-library {
    background: rgba(34, 197, 94, 0.1);
    border-left: 3px solid var(--accent-green, #22c55e);
  }

  .library-badge {
    display: inline-block;
    font-size: 9px;
    font-weight: 600;
    padding: 2px 6px;
    background: var(--accent-green, #22c55e);
    color: white;
    border-radius: 3px;
    margin-left: 8px;
    vertical-align: middle;
  }

  .suggestion-poster {
    width: 32px;
    height: 48px;
    object-fit: cover;
    border-radius: 4px;
    flex-shrink: 0;
  }

  .suggestion-poster-empty {
    background: var(--bg-secondary);
  }

  .suggestion-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    overflow: hidden;
  }

  .suggestion-title {
    font-size: 14px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .suggestion-year {
    font-size: 12px;
    color: var(--text-muted);
  }
</style>
