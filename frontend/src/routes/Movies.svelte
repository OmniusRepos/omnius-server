<script lang="ts">
  import { link } from 'svelte-spa-router';
  import { onMount } from 'svelte';
  import { getMovies, getMovie, deleteMovie, updateMovie, getMovieByIMDB, type Movie, type Torrent } from '../lib/api/client';
  import Modal from '../lib/components/Modal.svelte';

  let movies: Movie[] = [];
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
  let showTorrentModal = false;
  let selectedMovie: Movie | null = null;

  // Form data
  let movieForm = {
    imdb_code: '',
    title: '',
    year: new Date().getFullYear(),
    rating: 0,
    runtime: 0,
    genres: '',
    language: 'en',
    summary: '',
    yt_trailer_code: '',
    medium_cover_image: '',
    background_image: '',
    status: 'available',     // 'available' or 'coming_soon'
    release_date: '',        // YYYY-MM-DD format
    franchise: '',           // e.g., "Lord of the Rings", "Star Wars"
  };

  let torrentForm = {
    hash: '',
    quality: '1080p',
    type: 'web',
    size: '',
  };

  // YTS torrent search
  let ytsTorrents: Array<{hash: string, quality: string, type: string, size: string, seeds: number, peers: number}> = [];
  let selectedYtsTorrents: Set<string> = new Set();
  let ytsLoading = false;
  let ytsError = '';
  let addingTorrents = false;

  onMount(async () => {
    await loadMovies();
  });

  async function loadMovies() {
    loading = true;
    try {
      const result = await getMovies({ page, limit, search: search || undefined });
      movies = result.movies;
      total = result.total;
    } catch (err) {
      console.error('Failed to load movies:', err);
    } finally {
      loading = false;
    }
  }

  function handleSearch() {
    page = 1;
    loadMovies();
  }

  function goToPage(p: number) {
    if (p >= 1 && p <= totalPages) {
      page = p;
      loadMovies();
    }
  }

  function changePerPage(newLimit: number) {
    limit = newLimit;
    page = 1;
    loadMovies();
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
    movieForm = {
      imdb_code: '',
      title: '',
      year: new Date().getFullYear(),
      rating: 0,
      runtime: 0,
      genres: '',
      language: 'en',
      summary: '',
      yt_trailer_code: '',
      medium_cover_image: '',
      background_image: '',
      status: 'available',
      release_date: '',
      franchise: '',
    };
    showAddModal = true;
  }

  function openEditModal(movie: Movie) {
    selectedMovie = movie;
    movieForm = {
      imdb_code: movie.imdb_code || '',
      title: movie.title,
      year: movie.year,
      rating: movie.rating || 0,
      runtime: movie.runtime || 0,
      genres: movie.genres?.join(', ') || '',
      language: movie.language || 'en',
      summary: movie.summary || '',
      yt_trailer_code: movie.yt_trailer_code || '',
      medium_cover_image: movie.medium_cover_image || '',
      background_image: movie.background_image || '',
      status: movie.status || 'available',
      release_date: movie.release_date || '',
      franchise: movie.franchise || '',
    };
    showEditModal = true;
  }

  function openDeleteModal(movie: Movie) {
    selectedMovie = movie;
    showDeleteModal = true;
  }

  async function openTorrentModal(movie: Movie) {
    selectedMovie = movie;
    torrentForm = {
      hash: '',
      quality: '1080p',
      type: 'web',
      size: '',
    };
    ytsTorrents = [];
    selectedYtsTorrents = new Set();
    ytsError = '';
    showTorrentModal = true;

    // Search YTS for available torrents
    if (movie.imdb_code) {
      ytsLoading = true;
      try {
        const res = await fetch(`/admin/api/yts/search?imdb=${encodeURIComponent(movie.imdb_code)}`);
        if (res.ok) {
          const data = await res.json();
          if (data.status === 'error') {
            ytsError = 'YTS unavailable - enter torrent manually';
          } else {
            const ytsMovie = data.data?.movies?.[0];
            if (ytsMovie?.torrents) {
              ytsTorrents = ytsMovie.torrents.map((t: any) => ({
                hash: t.hash,
                quality: t.quality,
                type: t.type || 'web',
                size: t.size,
                seeds: t.seeds || 0,
                peers: t.peers || 0,
              }));
            } else {
              ytsError = 'No torrents found on YTS for this movie';
            }
          }
        }
      } catch (err) {
        console.error('Failed to search YTS:', err);
        ytsError = 'YTS unavailable - enter torrent manually';
      }
      ytsLoading = false;
    }
  }

  function toggleYtsTorrent(hash: string) {
    if (selectedYtsTorrents.has(hash)) {
      selectedYtsTorrents.delete(hash);
    } else {
      selectedYtsTorrents.add(hash);
    }
    selectedYtsTorrents = selectedYtsTorrents; // trigger reactivity
  }

  function selectAllYtsTorrents() {
    if (selectedYtsTorrents.size === ytsTorrents.length) {
      selectedYtsTorrents = new Set();
    } else {
      selectedYtsTorrents = new Set(ytsTorrents.map(t => t.hash));
    }
  }

  async function handleAddSelectedTorrents() {
    if (!selectedMovie || selectedYtsTorrents.size === 0) return;
    addingTorrents = true;
    try {
      const torrentsToAdd = ytsTorrents.filter(t => selectedYtsTorrents.has(t.hash));
      for (const torrent of torrentsToAdd) {
        await fetch(`/admin/movies/${selectedMovie.id}/torrent`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
            'Accept': 'application/json'
          },
          body: new URLSearchParams({
            hash: torrent.hash,
            quality: torrent.quality,
            type: torrent.type,
            size: torrent.size,
          }),
        });
      }
      showTorrentModal = false;
      loadMovies();
    } catch (err) {
      console.error('Failed to add torrents:', err);
    } finally {
      addingTorrents = false;
    }
  }

  async function handleAddMovie() {
    try {
      const res = await fetch('/admin/movies', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'Accept': 'application/json'
        },
        body: new URLSearchParams({
          imdb_code: movieForm.imdb_code,
          title: movieForm.title,
          year: String(movieForm.year),
          rating: String(movieForm.rating),
          runtime: String(movieForm.runtime),
          genres: movieForm.genres,
          language: movieForm.language,
          summary: movieForm.summary,
          yt_trailer_code: movieForm.yt_trailer_code,
          medium_cover_image: movieForm.medium_cover_image,
          background_image: movieForm.background_image,
          status: movieForm.status,
          release_date: movieForm.release_date,
          franchise: movieForm.franchise,
        }),
      });
      if (res.ok) {
        showAddModal = false;
        loadMovies();
      }
    } catch (err) {
      console.error('Failed to add movie:', err);
    }
  }

  async function handleEditMovie() {
    if (!selectedMovie) return;
    try {
      await updateMovie(selectedMovie.id, {
        imdb_code: movieForm.imdb_code,
        title: movieForm.title,
        year: movieForm.year,
        rating: movieForm.rating,
        runtime: movieForm.runtime,
        genres: movieForm.genres,
        language: movieForm.language,
        summary: movieForm.summary,
        yt_trailer_code: movieForm.yt_trailer_code,
        medium_cover_image: movieForm.medium_cover_image,
        background_image: movieForm.background_image,
        status: movieForm.status,
        release_date: movieForm.release_date,
        franchise: movieForm.franchise,
      });
      showEditModal = false;
      loadMovies();
    } catch (err) {
      console.error('Failed to update movie:', err);
    }
  }

  async function handleDeleteMovie() {
    if (!selectedMovie) return;
    try {
      await deleteMovie(selectedMovie.id);
      showDeleteModal = false;
      loadMovies();
    } catch (err) {
      console.error('Failed to delete movie:', err);
    }
  }

  // Sync featured / top 250 / latest / scan torrents
  let syncingFeatured = false;
  let syncingTop250 = false;
  let syncingLatest = false;
  let scanningTorrents = false;
  let syncResult: {type: string, imported: number, skipped: number, comingSoon?: number, added?: number, scanned?: number, total?: number} | null = null;

  async function syncYTSFeatured() {
    syncingFeatured = true;
    syncResult = null;
    try {
      const res = await fetch('/admin/api/yts/sync-featured', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        syncResult = { type: 'YTS Featured', imported: data.imported, skipped: data.skipped, total: data.total };
        if (data.imported > 0) {
          loadMovies();
        }
      }
    } catch (err) {
      console.error('Failed to sync featured:', err);
    } finally {
      syncingFeatured = false;
    }
  }

  async function syncIMDBTop250() {
    syncingTop250 = true;
    syncResult = null;
    try {
      const res = await fetch('/admin/api/imdb/sync-top250', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        syncResult = { type: 'IMDB Top 250', imported: data.imported, skipped: data.skipped };
        if (data.imported > 0) {
          loadMovies();
        }
      }
    } catch (err) {
      console.error('Failed to sync top 250:', err);
    } finally {
      syncingTop250 = false;
    }
  }

  async function syncLatest() {
    syncingLatest = true;
    syncResult = null;
    try {
      const res = await fetch('/admin/api/imdb/sync-latest', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        syncResult = { type: 'Latest Movies', imported: data.imported, skipped: data.skipped, comingSoon: data.coming_soon };
        if (data.imported > 0 || data.coming_soon > 0) {
          loadMovies();
        }
      }
    } catch (err) {
      console.error('Failed to sync latest:', err);
    } finally {
      syncingLatest = false;
    }
  }

  async function scanYTSTorrents() {
    scanningTorrents = true;
    syncResult = null;
    try {
      const res = await fetch('/admin/api/yts/scan-torrents', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        syncResult = { type: 'YTS Torrent Scan', imported: 0, skipped: data.skipped, added: data.added, scanned: data.scanned };
        if (data.added > 0) {
          loadMovies();
        }
      }
    } catch (err) {
      console.error('Failed to scan torrents:', err);
    } finally {
      scanningTorrents = false;
    }
  }

  let fetchingImdb = false;
  let imdbError: string | null = null;
  let titleSuggestions: Array<{id: string, title: string, year: string, poster: string, inLibrary?: boolean, movieId?: number}> = [];
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
        // Search both local DB and IMDB in parallel
        const [localRes, imdbRes] = await Promise.all([
          fetch(`/api/v2/list_movies.json?query_term=${encodeURIComponent(query)}&limit=5`),
          fetch(`/admin/api/imdb/search?query=${encodeURIComponent(query)}`)
        ]);

        const suggestions: typeof titleSuggestions = [];

        // Add local results first (marked as in library)
        if (localRes.ok) {
          const localData = await localRes.json();
          const localMovies = localData.data?.movies || [];
          for (const m of localMovies) {
            suggestions.push({
              id: m.imdb_code || `local-${m.id}`,
              title: m.title,
              year: String(m.year),
              poster: m.medium_cover_image || '',
              inLibrary: true,
              movieId: m.id,
            });
          }
        }

        // Add IMDB results (skip if already in local results)
        if (imdbRes.ok) {
          const imdbData = await imdbRes.json();
          const localIds = new Set(suggestions.map(s => s.id));
          for (const r of (imdbData.titles || []).slice(0, 8)) {
            if (!localIds.has(r.id)) {
              suggestions.push({
                id: r.id,
                title: r.primaryTitle,
                year: r.startYear ? String(r.startYear) : '',
                poster: r.primaryImage?.url || '',
                inLibrary: false,
              });
            }
          }
        }

        titleSuggestions = suggestions.slice(0, 10);
        showSuggestions = titleSuggestions.length > 0;
      } catch (err) {
        console.error('Search failed:', err);
      }
    }, 300);
  }

  async function selectSuggestion(suggestion: {id: string, title: string, year: string, inLibrary?: boolean, movieId?: number}) {
    showSuggestions = false;
    titleSuggestions = [];

    // If it's already in library, open edit modal directly
    if (suggestion.inLibrary && suggestion.movieId) {
      try {
        const movie = await getMovie(suggestion.movieId);
        if (movie) {
          showAddModal = false;
          selectedMovie = movie;
          movieForm = {
            imdb_code: movie.imdb_code || '',
            title: movie.title,
            year: movie.year,
            rating: movie.rating || 0,
            runtime: 0,
            genres: movie.genres?.join(', ') || '',
            language: 'en',
            summary: movie.summary || '',
            yt_trailer_code: '',
            medium_cover_image: movie.medium_cover_image || '',
            background_image: '',
          };
          showEditModal = true;
          return;
        }
      } catch (err) {
        console.error('Failed to load movie:', err);
      }
    }

    // Check if movie exists by IMDB code (in case it wasn't in local search results)
    if (suggestion.id.startsWith('tt')) {
      try {
        const result = await getMovieByIMDB(suggestion.id);
        if (result.exists && result.movie) {
          showAddModal = false;
          selectedMovie = result.movie;
          movieForm = {
            imdb_code: result.movie.imdb_code || '',
            title: result.movie.title,
            year: result.movie.year,
            rating: result.movie.rating || 0,
            runtime: 0,
            genres: result.movie.genres?.join(', ') || '',
            language: 'en',
            summary: result.movie.summary || '',
            yt_trailer_code: '',
            medium_cover_image: result.movie.medium_cover_image || '',
            background_image: '',
          };
          showEditModal = true;
          return;
        }
      } catch (err) {
        console.error('Failed to check existing movie:', err);
      }
    }

    // Movie doesn't exist - continue with add flow
    movieForm.imdb_code = suggestion.id;
    movieForm.title = suggestion.title;
    movieForm.year = parseInt(suggestion.year) || new Date().getFullYear();
    // Auto-fetch full details from IMDB
    fetchFromImdb();
  }

  async function fetchFromImdb() {
    if (!movieForm.imdb_code) return;
    fetchingImdb = true;
    imdbError = null;
    try {
      const res = await fetch(`/admin/api/imdb/title/${movieForm.imdb_code}`);
      if (res.ok) {
        const data = await res.json();
        console.log('IMDB data:', data);

        // Check if it's a TV series
        const titleType = (data.type || data.titleType || '').toLowerCase();
        if (titleType.includes('series') || titleType.includes('tv')) {
          imdbError = `"${data.primaryTitle || data.title}" is a TV Series. Add it in the Series section instead.`;
          fetchingImdb = false;
          return;
        }

        // Clear any previous error
        imdbError = null;

        // Basic info
        movieForm.title = data.primaryTitle || movieForm.title;
        movieForm.year = data.startYear || movieForm.year;
        movieForm.rating = data.rating?.aggregateRating || movieForm.rating;
        movieForm.runtime = data.runtimeSeconds ? Math.round(data.runtimeSeconds / 60) : movieForm.runtime;
        movieForm.genres = data.genres?.join(', ') || movieForm.genres;

        // Description fields
        movieForm.summary = data.plot || movieForm.summary;

        // Images
        const posterUrl = data.primaryImage?.url || '';
        movieForm.medium_cover_image = posterUrl;

        // Language (default to English if not specified)
        movieForm.language = 'en';
      }
    } catch (err) {
      console.error('Failed to fetch from IMDB:', err);
      imdbError = 'Failed to fetch from IMDB';
    } finally {
      fetchingImdb = false;
    }
  }

  async function handleAddTorrent() {
    if (!selectedMovie) return;
    try {
      const res = await fetch(`/admin/movies/${selectedMovie.id}/torrent`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'Accept': 'application/json'
        },
        body: new URLSearchParams({
          hash: torrentForm.hash,
          quality: torrentForm.quality,
          type: torrentForm.type,
          size: torrentForm.size,
        }),
      });
      if (res.ok) {
        showTorrentModal = false;
        loadMovies();
      }
    } catch (err) {
      console.error('Failed to add torrent:', err);
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

<div class="movies-page">
  <header class="page-header">
    <h1 class="page-title">MOVIES</h1>
    <div class="page-actions">
      <div class="search-box">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <circle cx="11" cy="11" r="8"/>
          <path d="m21 21-4.35-4.35"/>
        </svg>
        <input
          type="text"
          placeholder="Search movies..."
          bind:value={search}
          on:keydown={(e) => e.key === 'Enter' && handleSearch()}
        />
      </div>
      <button class="btn btn-secondary" on:click={syncYTSFeatured} disabled={syncingFeatured || syncingTop250 || syncingLatest || scanningTorrents}>
        {syncingFeatured ? 'SYNCING...' : 'YTS FEATURED'}
      </button>
      <button class="btn btn-secondary" on:click={syncIMDBTop250} disabled={syncingTop250 || syncingFeatured || syncingLatest || scanningTorrents}>
        {syncingTop250 ? 'SYNCING...' : 'IMDB TOP 250'}
      </button>
      <button class="btn btn-secondary" on:click={syncLatest} disabled={syncingLatest || syncingFeatured || syncingTop250 || scanningTorrents}>
        {syncingLatest ? 'SYNCING...' : 'LATEST MOVIES'}
      </button>
      <button class="btn btn-secondary" on:click={scanYTSTorrents} disabled={scanningTorrents || syncingFeatured || syncingTop250 || syncingLatest}>
        {scanningTorrents ? 'SCANNING...' : 'SCAN TORRENTS'}
      </button>
      <button class="btn btn-primary" on:click={openAddModal}>ADD</button>
    </div>
  </header>

  {#if syncResult}
    <div class="sync-result">
      {#if syncResult.added !== undefined}
        Scanned {syncResult.scanned} movies, found {syncResult.added} new torrent{syncResult.added !== 1 ? 's' : ''}
      {:else}
        Imported {syncResult.imported} new movie{syncResult.imported !== 1 ? 's' : ''} from {syncResult.type}
        {#if syncResult.comingSoon}
          + {syncResult.comingSoon} as coming soon
        {/if}
        ({syncResult.skipped} already in library)
      {/if}
      <button class="btn-dismiss" on:click={() => syncResult = null}>&times;</button>
    </div>
  {/if}

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if movies.length === 0}
    <div class="empty-state">
      <p>No movies found</p>
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
            <th>Ratings</th>
            <th>Torrents</th>
            <th style="text-align: right;">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each movies as movie}
            <tr>
              <td>
                {#if movie.medium_cover_image}
                  <img src={movie.medium_cover_image} alt={movie.title} class="poster" />
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
                <a href="/movies/{movie.id}" use:link class="movie-title">{movie.title}</a>
                {#if movie.status === 'coming_soon'}
                  <span class="status-badge status-coming-soon">COMING SOON</span>
                {/if}
              </td>
              <td>{movie.year}</td>
              <td>
                <div class="ratings">
                  {#if movie.imdb_rating}
                    <div class="rating-badge rating-imdb">
                      <span class="source">IMDb</span>
                      <span class="score">{movie.imdb_rating}/10</span>
                    </div>
                  {/if}
                  {#if movie.rotten_tomatoes}
                    <div class="rating-badge rating-rt">
                      <span class="score">{movie.rotten_tomatoes}%</span>
                    </div>
                  {/if}
                </div>
              </td>
              <td>
                <div class="torrents">
                  {#each movie.torrents || [] as torrent}
                    <span class="badge badge-quality badge-{torrent.quality}">{torrent.quality}</span>
                  {/each}
                  <button class="btn-add-torrent" title="Add Torrent" on:click={() => openTorrentModal(movie)}>
                    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="12" y1="5" x2="12" y2="19"/>
                      <line x1="5" y1="12" x2="19" y2="12"/>
                    </svg>
                  </button>
                </div>
              </td>
              <td>
                <div class="actions">
                  <button class="btn btn-sm btn-secondary" on:click={() => openEditModal(movie)}>EDIT</button>
                  <button class="btn btn-sm btn-danger" on:click={() => openDeleteModal(movie)}>DELETE</button>
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

<!-- Add Movie Modal -->
<Modal bind:open={showAddModal} title="Add Movie" size="lg" on:close={() => showAddModal = false}>
  <form on:submit|preventDefault={handleAddMovie}>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="imdb_code">IMDB Code</label>
        <div class="input-with-button">
          <input type="text" id="imdb_code" class="form-input" bind:value={movieForm.imdb_code} placeholder="tt1234567" />
          <button type="button" class="btn btn-sm btn-secondary" on:click={fetchFromImdb} disabled={fetchingImdb || !movieForm.imdb_code}>
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
            bind:value={movieForm.title}
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
        <input type="number" id="year" class="form-input" bind:value={movieForm.year} />
      </div>
      <div class="form-group">
        <label class="form-label" for="rating">Rating</label>
        <input type="number" id="rating" class="form-input" bind:value={movieForm.rating} step="0.1" min="0" max="10" />
      </div>
      <div class="form-group">
        <label class="form-label" for="runtime">Runtime (min)</label>
        <input type="number" id="runtime" class="form-input" bind:value={movieForm.runtime} min="0" />
      </div>
      <div class="form-group">
        <label class="form-label" for="language">Language</label>
        <select id="language" class="form-input" bind:value={movieForm.language}>
          <option value="en">English</option>
          <option value="es">Spanish</option>
          <option value="fr">French</option>
          <option value="de">German</option>
          <option value="it">Italian</option>
          <option value="ja">Japanese</option>
          <option value="ko">Korean</option>
          <option value="zh">Chinese</option>
          <option value="hi">Hindi</option>
          <option value="other">Other</option>
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="genres">Genres</label>
      <input type="text" id="genres" class="form-input" bind:value={movieForm.genres} placeholder="Action, Drama, Thriller" />
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="cover">Cover Image URL</label>
        <input type="text" id="cover" class="form-input" bind:value={movieForm.medium_cover_image} placeholder="https://..." />
      </div>
      <div class="form-group">
        <label class="form-label" for="background">Background Image URL</label>
        <input type="text" id="background" class="form-input" bind:value={movieForm.background_image} placeholder="https://..." />
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="yt_trailer">YouTube Trailer Code</label>
      <input type="text" id="yt_trailer" class="form-input" bind:value={movieForm.yt_trailer_code} placeholder="dQw4w9WgXcQ" />
    </div>
    <div class="form-group">
      <label class="form-label" for="franchise">Franchise</label>
      <input type="text" id="franchise" class="form-input" bind:value={movieForm.franchise} placeholder="e.g., Lord of the Rings, Star Wars" />
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label checkbox-label">
          <input type="checkbox" checked={movieForm.status === 'coming_soon'} on:change={(e) => movieForm.status = e.currentTarget.checked ? 'coming_soon' : 'available'} />
          Coming Soon
        </label>
        <p class="form-hint">Mark as upcoming release (no torrents yet)</p>
      </div>
      {#if movieForm.status === 'coming_soon'}
        <div class="form-group">
          <label class="form-label" for="release_date">Release Date</label>
          <input type="date" id="release_date" class="form-input" bind:value={movieForm.release_date} />
        </div>
      {/if}
    </div>
    <div class="form-group">
      <label class="form-label" for="summary">Summary</label>
      <textarea id="summary" class="form-input form-textarea" bind:value={movieForm.summary} rows="3"></textarea>
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showAddModal = false}>Cancel</button>
    <button class="btn btn-primary" on:click={handleAddMovie}>Add Movie</button>
  </svelte:fragment>
</Modal>

<!-- Edit Movie Modal -->
<Modal bind:open={showEditModal} title="Edit Movie" size="lg" on:close={() => showEditModal = false}>
  <form on:submit|preventDefault={handleEditMovie}>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="edit_imdb_code">IMDB Code</label>
        <div class="input-with-button">
          <input type="text" id="edit_imdb_code" class="form-input" bind:value={movieForm.imdb_code} placeholder="tt1234567" />
          <button type="button" class="btn btn-sm btn-secondary" on:click={fetchFromImdb} disabled={fetchingImdb || !movieForm.imdb_code}>
            {fetchingImdb ? 'Updating...' : 'Update'}
          </button>
        </div>
        {#if imdbError}
          <p class="form-error">{imdbError}</p>
        {/if}
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_title">Title *</label>
        <input type="text" id="edit_title" class="form-input" bind:value={movieForm.title} required />
      </div>
    </div>
    <div class="form-row form-row-4">
      <div class="form-group">
        <label class="form-label" for="edit_year">Year</label>
        <input type="number" id="edit_year" class="form-input" bind:value={movieForm.year} />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_rating">Rating</label>
        <input type="number" id="edit_rating" class="form-input" bind:value={movieForm.rating} step="0.1" min="0" max="10" />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_runtime">Runtime (min)</label>
        <input type="number" id="edit_runtime" class="form-input" bind:value={movieForm.runtime} min="0" />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_language">Language</label>
        <select id="edit_language" class="form-input" bind:value={movieForm.language}>
          <option value="en">English</option>
          <option value="es">Spanish</option>
          <option value="fr">French</option>
          <option value="de">German</option>
          <option value="other">Other</option>
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="edit_genres">Genres</label>
      <input type="text" id="edit_genres" class="form-input" bind:value={movieForm.genres} placeholder="Action, Drama" />
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="edit_cover">Cover Image URL</label>
        <input type="text" id="edit_cover" class="form-input" bind:value={movieForm.medium_cover_image} placeholder="https://..." />
      </div>
      <div class="form-group">
        <label class="form-label" for="edit_background">Background Image URL</label>
        <input type="text" id="edit_background" class="form-input" bind:value={movieForm.background_image} placeholder="https://..." />
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="edit_franchise">Franchise</label>
      <input type="text" id="edit_franchise" class="form-input" bind:value={movieForm.franchise} placeholder="e.g., Lord of the Rings, Star Wars" />
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label checkbox-label">
          <input type="checkbox" checked={movieForm.status === 'coming_soon'} on:change={(e) => movieForm.status = e.currentTarget.checked ? 'coming_soon' : 'available'} />
          Coming Soon
        </label>
        <p class="form-hint">Mark as upcoming release</p>
      </div>
      {#if movieForm.status === 'coming_soon'}
        <div class="form-group">
          <label class="form-label" for="edit_release_date">Release Date</label>
          <input type="date" id="edit_release_date" class="form-input" bind:value={movieForm.release_date} />
        </div>
      {/if}
    </div>
    <div class="form-group">
      <label class="form-label" for="edit_summary">Summary</label>
      <textarea id="edit_summary" class="form-input form-textarea" bind:value={movieForm.summary} rows="3"></textarea>
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showEditModal = false}>Cancel</button>
    <button class="btn btn-primary" on:click={handleEditMovie}>Save Changes</button>
  </svelte:fragment>
</Modal>

<!-- Delete Confirmation Modal -->
<Modal bind:open={showDeleteModal} title="Delete Movie" size="sm" on:close={() => showDeleteModal = false}>
  <p class="delete-warning">
    Are you sure you want to delete <strong>{selectedMovie?.title}</strong>?
  </p>
  <p class="text-muted">This action cannot be undone. All associated torrents will also be removed.</p>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger" on:click={handleDeleteMovie}>Delete</button>
  </svelte:fragment>
</Modal>

<!-- Add Torrent Modal -->
<Modal bind:open={showTorrentModal} title="Add Torrent" size="lg" on:close={() => showTorrentModal = false}>
  <p class="modal-subtitle">Adding torrent to: <strong>{selectedMovie?.title}</strong></p>

  <!-- YTS Torrents Section -->
  {#if selectedMovie?.imdb_code}
    <div class="yts-section">
      <div class="section-header">
        <h4 class="section-title">Available on YTS</h4>
        {#if ytsTorrents.length > 1}
          <button type="button" class="btn btn-sm btn-secondary" on:click={selectAllYtsTorrents}>
            {selectedYtsTorrents.size === ytsTorrents.length ? 'Deselect All' : 'Select All'}
          </button>
        {/if}
      </div>
      {#if ytsLoading}
        <div class="loading-inline">
          <div class="spinner-small"></div>
          <span>Searching YTS...</span>
        </div>
      {:else if ytsTorrents.length > 0}
        <div class="yts-torrents">
          {#each ytsTorrents as torrent}
            <label
              class="yts-torrent-item"
              class:selected={selectedYtsTorrents.has(torrent.hash)}
            >
              <input
                type="checkbox"
                checked={selectedYtsTorrents.has(torrent.hash)}
                on:change={() => toggleYtsTorrent(torrent.hash)}
              />
              <span class="torrent-quality-badge badge-{torrent.quality}">{torrent.quality}</span>
              <span class="torrent-type">{torrent.type}</span>
              <span class="torrent-size">{torrent.size}</span>
              <span class="torrent-seeds">🌱 {torrent.seeds}</span>
            </label>
          {/each}
        </div>
        {#if selectedYtsTorrents.size > 0}
          <button
            type="button"
            class="btn btn-primary mt-3"
            on:click={handleAddSelectedTorrents}
            disabled={addingTorrents}
          >
            {addingTorrents ? 'Adding...' : `Add ${selectedYtsTorrents.size} Torrent${selectedYtsTorrents.size > 1 ? 's' : ''}`}
          </button>
        {/if}
      {:else if ytsError}
        <p class="text-muted">{ytsError}</p>
      {/if}
    </div>
    <div class="divider">or enter manually</div>
  {/if}

  <form on:submit|preventDefault={handleAddTorrent}>
    <div class="form-group">
      <label class="form-label" for="torrent_hash">Torrent Hash / Magnet Link *</label>
      <input type="text" id="torrent_hash" class="form-input" bind:value={torrentForm.hash} placeholder="Magnet link or info hash" required />
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="torrent_quality">Quality</label>
        <select id="torrent_quality" class="form-input" bind:value={torrentForm.quality}>
          <option value="720p">720p</option>
          <option value="1080p">1080p</option>
          <option value="2160p">2160p (4K)</option>
          <option value="480p">480p</option>
        </select>
      </div>
      <div class="form-group">
        <label class="form-label" for="torrent_type">Type</label>
        <select id="torrent_type" class="form-input" bind:value={torrentForm.type}>
          <option value="web">Web</option>
          <option value="bluray">BluRay</option>
          <option value="hdtv">HDTV</option>
          <option value="webrip">WebRip</option>
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="torrent_size">Size</label>
      <input type="text" id="torrent_size" class="form-input" bind:value={torrentForm.size} placeholder="1.5 GB" />
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showTorrentModal = false}>Cancel</button>
    <button class="btn btn-primary" on:click={handleAddTorrent} disabled={!torrentForm.hash}>Add Torrent</button>
  </svelte:fragment>
</Modal>

<style>
  .sync-result {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: rgba(34, 197, 94, 0.1);
    border: 1px solid rgba(34, 197, 94, 0.3);
    border-radius: 8px;
    color: var(--accent-green, #22c55e);
    font-size: 14px;
    margin-bottom: 16px;
  }

  .btn-dismiss {
    margin-left: auto;
    background: none;
    border: none;
    color: var(--text-muted);
    font-size: 18px;
    cursor: pointer;
    padding: 0 4px;
  }

  .btn-dismiss:hover {
    color: var(--text-primary);
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

  .movie-title {
    font-weight: 500;
    color: var(--text-primary);
  }

  .movie-title:hover {
    color: var(--accent-blue);
  }

  .ratings {
    display: flex;
    gap: 8px;
  }

  .torrents {
    display: flex;
    gap: 4px;
    flex-wrap: wrap;
    align-items: center;
  }

  .btn-add-torrent {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    padding: 0;
    background: var(--bg-tertiary);
    border: 1px dashed var(--border-color);
    border-radius: 4px;
    color: var(--text-muted);
    cursor: pointer;
    transition: all var(--transition-fast);
  }

  .btn-add-torrent:hover {
    border-color: var(--accent-green);
    color: var(--accent-green);
    border-style: solid;
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

  .modal-subtitle {
    color: var(--text-secondary);
    margin-bottom: 20px;
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

  /* Loading */
  .loading-inline {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 16px 0;
    color: var(--text-muted);
  }

  .spinner-small {
    width: 20px;
    height: 20px;
    border: 2px solid var(--border-color);
    border-top-color: var(--accent-blue);
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  /* YTS Torrents */
  .yts-section {
    margin-bottom: 16px;
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .section-title {
    font-size: 14px;
    font-weight: 600;
    margin: 0;
    color: var(--text-secondary);
  }

  .yts-torrents {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
  }

  .yts-torrent-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 14px;
    background: var(--bg-tertiary);
    border: 2px solid var(--border-color);
    border-radius: 8px;
    cursor: pointer;
    transition: all 0.15s;
    color: var(--text-primary);
    font-size: 13px;
  }

  .yts-torrent-item input[type="checkbox"] {
    width: 18px;
    height: 18px;
    accent-color: var(--accent-green, #22c55e);
    cursor: pointer;
  }

  .yts-torrent-item:hover {
    border-color: var(--accent-blue);
    background: var(--bg-secondary);
  }

  .yts-torrent-item.selected {
    border-color: var(--accent-green, #22c55e);
    background: rgba(34, 197, 94, 0.1);
  }

  .mt-3 {
    margin-top: 12px;
  }

  .torrent-quality-badge {
    font-weight: 700;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
  }

  .badge-720p {
    background: #3b82f6;
    color: white;
  }

  .badge-1080p {
    background: #8b5cf6;
    color: white;
  }

  .badge-2160p {
    background: #f59e0b;
    color: white;
  }

  .torrent-type {
    color: var(--text-muted);
  }

  .torrent-size {
    color: var(--text-secondary);
  }

  .torrent-seeds {
    color: var(--accent-green, #22c55e);
    font-weight: 500;
  }

  .divider {
    display: flex;
    align-items: center;
    text-align: center;
    margin: 20px 0;
    color: var(--text-muted);
    font-size: 12px;
  }

  .divider::before,
  .divider::after {
    content: '';
    flex: 1;
    border-bottom: 1px solid var(--border-color);
  }

  .divider::before {
    margin-right: 12px;
  }

  .divider::after {
    margin-left: 12px;
  }

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }

  .checkbox-label input[type="checkbox"] {
    width: 18px;
    height: 18px;
    accent-color: var(--accent-primary);
  }

  .form-hint {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  .status-badge {
    display: inline-block;
    font-size: 9px;
    font-weight: 700;
    padding: 3px 6px;
    border-radius: 3px;
    margin-left: 8px;
    vertical-align: middle;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .status-coming-soon {
    background: linear-gradient(135deg, #f59e0b, #d97706);
    color: white;
    animation: pulse 2s ease-in-out infinite;
  }

  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.7; }
  }
</style>
