<script lang="ts">
  import { onMount } from 'svelte';
  import { link, push } from 'svelte-spa-router';
  import { getMovie, deleteMovie, updateMovie, type Movie } from '../lib/api/client';
  import Modal from '../lib/components/Modal.svelte';

  export let params: { id: string };

  let movie: Movie | null = null;
  let loading = true;
  let fetchingImdb = false;

  // Modal states
  let showEditModal = false;
  let showDeleteModal = false;
  let showTorrentModal = false;

  // Forms
  let movieForm = {
    imdb_code: '',
    title: '',
    year: 0,
    rating: 0,
    runtime: 0,
    genres: '',
    language: 'en',
    summary: '',
    yt_trailer_code: '',
    medium_cover_image: '',
    background_image: '',
  };

  let torrentForm = {
    hash: '',
    quality: '1080p',
    type: 'web',
    size: '',
  };

  onMount(async () => {
    await loadMovie();
  });

  async function loadMovie() {
    loading = true;
    try {
      movie = await getMovie(parseInt(params.id));
    } catch (err) {
      console.error('Failed to load movie:', err);
    } finally {
      loading = false;
    }
  }

  function openEditModal() {
    if (!movie) return;
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
  }

  async function fetchFromImdb() {
    if (!movieForm.imdb_code) return;
    fetchingImdb = true;
    try {
      const res = await fetch(`/admin/api/imdb/title/${movieForm.imdb_code}`);
      if (res.ok) {
        const data = await res.json();
        movieForm.title = data.primaryTitle || movieForm.title;
        movieForm.year = data.startYear || movieForm.year;
        movieForm.rating = data.rating?.aggregateRating || movieForm.rating;
        movieForm.runtime = data.runtimeSeconds ? Math.round(data.runtimeSeconds / 60) : movieForm.runtime;
        movieForm.genres = data.genres?.join(', ') || movieForm.genres;
        movieForm.summary = data.plot || movieForm.summary;
        movieForm.medium_cover_image = data.primaryImage?.url || movieForm.medium_cover_image;
      }
    } catch (err) {
      console.error('Failed to fetch from IMDB:', err);
    } finally {
      fetchingImdb = false;
    }
  }

  async function handleEditMovie() {
    if (!movie) return;
    try {
      await updateMovie(movie.id, {
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
      });
      showEditModal = false;
      await loadMovie();
    } catch (err) {
      console.error('Failed to update movie:', err);
    }
  }

  async function handleDeleteMovie() {
    if (!movie) return;
    try {
      await deleteMovie(movie.id);
      showDeleteModal = false;
      push('/movies');
    } catch (err) {
      console.error('Failed to delete movie:', err);
    }
  }

  async function handleAddTorrent() {
    if (!movie) return;
    try {
      const res = await fetch(`/admin/movies/${movie.id}/torrent`, {
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
        torrentForm = { hash: '', quality: '1080p', type: 'web', size: '' };
        await loadMovie();
      }
    } catch (err) {
      console.error('Failed to add torrent:', err);
    }
  }
</script>

<div class="movie-detail">
  <header class="page-header">
    <div class="breadcrumb">
      <a href="/movies" use:link>MOVIES</a>
      <span>/</span>
      <span>{movie?.title || 'Loading...'}</span>
    </div>
  </header>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if movie}
    <div class="card movie-card">
      <div class="movie-header">
        <div class="movie-actions">
          <button class="btn btn-secondary" on:click={openEditModal}>EDIT</button>
          <button class="btn btn-danger" on:click={() => showDeleteModal = true}>DELETE</button>
        </div>
      </div>

      <div class="movie-info">
        {#if movie.medium_cover_image || movie.large_cover_image}
          <img src={movie.large_cover_image || movie.medium_cover_image} alt={movie.title} class="poster poster-lg" />
        {:else}
          <div class="poster poster-lg poster-placeholder">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
              <rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/>
              <line x1="7" y1="2" x2="7" y2="22"/>
              <line x1="17" y1="2" x2="17" y2="22"/>
              <line x1="2" y1="12" x2="22" y2="12"/>
            </svg>
          </div>
        {/if}

        <div class="movie-meta">
          <h2 class="movie-title">{movie.title}</h2>

          <div class="movie-details">
            <span class="movie-year">{movie.year}</span>
            {#if movie.runtime}
              <span class="movie-runtime">{movie.runtime} min</span>
            {/if}
            {#if movie.mpa_rating}
              <span class="mpa-rating">{movie.mpa_rating}</span>
            {/if}
            {#if movie.language}
              <span class="movie-language">{movie.language.toUpperCase()}</span>
            {/if}
            {#if movie.imdb_code}
              <a href="https://www.imdb.com/title/{movie.imdb_code}" target="_blank" rel="noopener" class="imdb-link">
                {movie.imdb_code}
              </a>
            {/if}
          </div>

          <!-- Ratings -->
          <div class="ratings-row">
            {#if movie.imdb_rating}
              <div class="rating-badge rating-imdb">
                <span class="rating-source">IMDb</span>
                <span class="rating-score">{movie.imdb_rating}/10</span>
                {#if movie.imdb_votes}
                  <span class="rating-votes">({movie.imdb_votes})</span>
                {/if}
              </div>
            {/if}
            {#if movie.rotten_tomatoes}
              <div class="rating-badge rating-rt" class:fresh={movie.rotten_tomatoes >= 60}>
                <span class="rating-source">RT</span>
                <span class="rating-score">{movie.rotten_tomatoes}%</span>
              </div>
            {/if}
            {#if movie.metacritic}
              <div class="rating-badge rating-mc" class:good={movie.metacritic >= 60}>
                <span class="rating-source">Meta</span>
                <span class="rating-score">{movie.metacritic}</span>
              </div>
            {/if}
          </div>

          {#if movie.genres && movie.genres.length > 0}
            <div class="movie-genres">
              {#each movie.genres as genre}
                <span class="genre-tag">{genre}</span>
              {/each}
            </div>
          {/if}

          {#if movie.summary || movie.description_full}
            <p class="movie-summary">{movie.description_full || movie.summary}</p>
          {/if}

          {#if movie.yt_trailer_code}
            <div class="trailer-link">
              <a href="https://www.youtube.com/watch?v={movie.yt_trailer_code}" target="_blank" rel="noopener" class="btn btn-sm btn-secondary">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                  <path d="M19.615 3.184c-3.604-.246-11.631-.245-15.23 0C.488 3.45.029 5.804 0 12c.029 6.185.484 8.549 4.385 8.816 3.6.245 11.626.246 15.23 0C23.512 20.55 23.971 18.196 24 12c-.029-6.185-.484-8.549-4.385-8.816zM9 16V8l8 3.993L9 16z"/>
                </svg>
                Watch Trailer
              </a>
            </div>
          {/if}
        </div>
      </div>
    </div>

    <!-- Cast -->
    {#if movie.cast && movie.cast.length > 0}
      <div class="card cast-card">
        <h3>Cast</h3>
        <div class="cast-grid">
          {#each movie.cast as actor}
            <div class="cast-item">
              {#if actor.url_small_image}
                <img src={actor.url_small_image} alt={actor.name} class="cast-image" />
              {:else}
                <div class="cast-image cast-placeholder">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <circle cx="12" cy="8" r="4"/>
                    <path d="M20 21a8 8 0 10-16 0"/>
                  </svg>
                </div>
              {/if}
              <div class="cast-info">
                <span class="cast-name">{actor.name}</span>
                {#if actor.character_name}
                  <span class="cast-character">{actor.character_name}</span>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}

    <div class="card torrents-card">
      <div class="torrents-header">
        <h3>Torrents</h3>
        <button class="btn btn-sm btn-primary" on:click={() => showTorrentModal = true}>ADD TORRENT</button>
      </div>
      {#if movie.torrents && movie.torrents.length > 0}
        {#each movie.torrents as torrent}
          <div class="torrent-row">
            <span class="torrent-quality badge badge-{torrent.quality}">{torrent.quality}</span>
            <span class="torrent-type">{torrent.type}</span>
            <span class="torrent-size">{torrent.size}</span>
            <span class="torrent-hash" title={torrent.hash}>{torrent.hash.substring(0, 8)}...</span>
            <div class="torrent-actions">
              <a href="/stream/{torrent.hash}/0" target="_blank" class="btn btn-sm btn-primary">PLAY</a>
            </div>
          </div>
        {/each}
      {:else}
        <p class="text-muted">No torrents yet</p>
      {/if}
    </div>
  {:else}
    <div class="empty-state">
      <p>Movie not found</p>
      <a href="/movies" use:link class="btn btn-primary">Back to Movies</a>
    </div>
  {/if}
</div>

<!-- Edit Movie Modal -->
<Modal bind:open={showEditModal} title="Edit Movie" size="lg" on:close={() => showEditModal = false}>
  <form on:submit|preventDefault={handleEditMovie}>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="imdb_code">IMDB Code</label>
        <div class="input-with-button">
          <input type="text" id="imdb_code" class="form-input" bind:value={movieForm.imdb_code} placeholder="tt1234567" />
          <button type="button" class="btn btn-sm btn-secondary" on:click={fetchFromImdb} disabled={fetchingImdb || !movieForm.imdb_code}>
            {fetchingImdb ? 'Updating...' : 'Update'}
          </button>
        </div>
      </div>
      <div class="form-group">
        <label class="form-label" for="title">Title *</label>
        <input type="text" id="title" class="form-input" bind:value={movieForm.title} required />
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
          <option value="sq">Albanian</option>
        </select>
      </div>
    </div>
    <div class="form-group">
      <label class="form-label" for="genres">Genres</label>
      <input type="text" id="genres" class="form-input" bind:value={movieForm.genres} placeholder="Action, Drama" />
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
      <label class="form-label" for="summary">Summary</label>
      <textarea id="summary" class="form-input form-textarea" bind:value={movieForm.summary} rows="3"></textarea>
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
    Are you sure you want to delete <strong>{movie?.title}</strong>?
  </p>
  <p class="text-muted">This action cannot be undone. All associated torrents will also be removed.</p>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger" on:click={handleDeleteMovie}>Delete</button>
  </svelte:fragment>
</Modal>

<!-- Add Torrent Modal -->
<Modal bind:open={showTorrentModal} title="Add Torrent" size="md" on:close={() => showTorrentModal = false}>
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
    <button class="btn btn-primary" on:click={handleAddTorrent}>Add Torrent</button>
  </svelte:fragment>
</Modal>

<style>
  .breadcrumb {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 24px;
    font-weight: 600;
  }

  .breadcrumb a {
    color: var(--text-muted);
  }

  .breadcrumb span:last-child {
    color: var(--text-primary);
  }

  .movie-card {
    margin-bottom: 24px;
  }

  .movie-header {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 16px;
  }

  .movie-actions {
    display: flex;
    gap: 8px;
  }

  .movie-info {
    display: flex;
    gap: 24px;
  }

  .poster-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-tertiary);
    color: var(--text-muted);
  }

  .movie-meta {
    flex: 1;
  }

  .movie-title {
    font-size: 28px;
    font-weight: 600;
    margin-bottom: 8px;
  }

  .movie-details {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 12px;
  }

  .movie-year {
    color: var(--text-muted);
    font-size: 16px;
  }

  .movie-runtime {
    color: var(--text-muted);
  }

  .mpa-rating {
    padding: 2px 8px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
  }

  .movie-language {
    color: var(--text-muted);
    font-size: 13px;
  }

  .ratings-row {
    display: flex;
    gap: 12px;
    margin-bottom: 16px;
  }

  .rating-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 6px 12px;
    background: var(--bg-tertiary);
    border-radius: 6px;
    font-size: 13px;
  }

  .rating-source {
    font-weight: 600;
    color: var(--text-muted);
  }

  .rating-score {
    font-weight: 700;
  }

  .rating-votes {
    font-size: 11px;
    color: var(--text-muted);
  }

  .rating-imdb .rating-score {
    color: #f5c518;
  }

  .rating-rt .rating-score {
    color: #fa320a;
  }

  .rating-rt.fresh .rating-score {
    color: #21d07a;
  }

  .rating-mc .rating-score {
    color: #fa320a;
  }

  .rating-mc.good .rating-score {
    color: #66cc33;
  }

  .trailer-link {
    margin-top: 16px;
  }

  .trailer-link .btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  /* Cast */
  .cast-card {
    margin-bottom: 24px;
  }

  .cast-card h3 {
    margin-bottom: 16px;
  }

  .cast-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 12px;
  }

  .cast-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px;
    background: var(--bg-tertiary);
    border-radius: 8px;
  }

  .cast-image {
    width: 48px;
    height: 48px;
    border-radius: 50%;
    object-fit: cover;
    flex-shrink: 0;
  }

  .cast-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-secondary);
    color: var(--text-muted);
  }

  .cast-info {
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  .cast-name {
    font-weight: 500;
    font-size: 13px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .cast-character {
    font-size: 12px;
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .imdb-link {
    color: var(--accent-blue);
    font-family: monospace;
    font-size: 13px;
    text-decoration: none;
  }

  .imdb-link:hover {
    text-decoration: underline;
  }

  .movie-genres {
    display: flex;
    gap: 8px;
    flex-wrap: wrap;
    margin-bottom: 16px;
  }

  .genre-tag {
    padding: 4px 10px;
    background: var(--bg-tertiary);
    border-radius: 4px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .movie-summary {
    color: var(--text-secondary);
    line-height: 1.6;
  }

  .torrents-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .torrents-header h3 {
    margin: 0;
  }

  .torrent-row {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 12px 0;
    border-bottom: 1px solid var(--border-color);
  }

  .torrent-row:last-child {
    border-bottom: none;
  }

  .torrent-quality {
    font-weight: 600;
    min-width: 60px;
  }

  .torrent-type {
    color: var(--text-muted);
    min-width: 60px;
  }

  .torrent-size {
    color: var(--text-secondary);
    min-width: 80px;
  }

  .torrent-hash {
    font-family: monospace;
    font-size: 12px;
    color: var(--text-muted);
  }

  .torrent-actions {
    margin-left: auto;
    display: flex;
    gap: 8px;
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
    font-size: 13px;
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

  .input-with-button {
    display: flex;
    gap: 8px;
  }

  .input-with-button .form-input {
    flex: 1;
  }

  .delete-warning {
    margin-bottom: 8px;
  }

  .empty-state {
    text-align: center;
    padding: 48px;
  }

  .empty-state p {
    margin-bottom: 16px;
    color: var(--text-muted);
  }
</style>
