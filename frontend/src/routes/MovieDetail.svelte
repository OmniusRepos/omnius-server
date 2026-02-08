<script lang="ts">
  import { onMount } from 'svelte';
  import { link, push } from 'svelte-spa-router';
  import { getMovie, deleteMovie, updateMovie, getMovies, getSubtitles, getSubtitlePreview, deleteSubtitle, syncSubtitles, type Movie, type StoredSubtitle, type SubtitlePreview } from '../lib/api/client';
  import Modal from '../lib/components/Modal.svelte';

  export let params: { id: string };

  let movie: Movie | null = null;
  let loading = true;
  let fetchingImdb = false;
  let refreshing = false;

  // Franchise movies
  let franchiseMovies: Movie[] = [];
  let loadingFranchise = false;

  // Subtitles
  let subtitles: StoredSubtitle[] = [];
  let loadingSubtitles = false;
  let showPreviewModal = false;
  let subtitlePreview: SubtitlePreview | null = null;
  let loadingPreview = false;
  let expandedTorrentHash: string | null = null;
  let syncingSubtitles = false;

  // Modal states
  let showEditModal = false;
  let showDeleteModal = false;
  let showTorrentModal = false;
  let showFranchiseModal = false;

  // Franchise search
  let franchiseSearch = '';
  let franchiseSearchResults: any[] = [];  // IMDB results
  let searchingFranchise = false;

  interface IMDBResult {
    id: string;
    type: string;
    primaryTitle: string;
    startYear: number;
    primaryImage?: { url: string };
    rating?: { aggregateRating: number };
  }

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
    status: 'available',
    release_date: '',
    franchise: '',
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
      // Load franchise movies and subtitles after main movie loads
      if (movie) {
        loadFranchiseMovies();
        loadSubtitles();
      }
    } catch (err) {
      console.error('Failed to load movie:', err);
    } finally {
      loading = false;
    }
  }

  async function loadFranchiseMovies() {
    if (!movie?.franchise) return;
    loadingFranchise = true;
    try {
      const res = await fetch(`/api/v2/franchise_movies.json?movie_id=${movie.id}`);
      if (res.ok) {
        const data = await res.json();
        franchiseMovies = data.data?.movies || [];
      }
    } catch (err) {
      console.error('Failed to load franchise movies:', err);
    } finally {
      loadingFranchise = false;
    }
  }

  function openFranchiseModal() {
    // Pre-fill with title prefix (e.g., "Avengers" from "Avengers: Endgame")
    // This is more specific than broad franchises like "Marvel"
    const titlePrefix = movie?.title?.split(/[:\-–]/)[0].trim() || '';
    franchiseSearch = titlePrefix;
    franchiseSearchResults = [];
    showFranchiseModal = true;
    searchFranchiseMovies();
  }

  async function searchFranchiseMovies() {
    if (!franchiseSearch.trim()) {
      franchiseSearchResults = [];
      return;
    }
    searchingFranchise = true;
    try {
      // Search IMDB API
      const res = await fetch(`/admin/api/imdb/search?query=${encodeURIComponent(franchiseSearch)}`);
      if (res.ok) {
        const data = await res.json();
        // Filter to only movies, exclude current movie
        franchiseSearchResults = (data.titles || [])
          .filter((t: IMDBResult) => t.type === 'movie' && t.id !== movie?.imdb_code)
          .slice(0, 12);
      }
    } catch (err) {
      console.error('Failed to search franchise:', err);
    } finally {
      searchingFranchise = false;
    }
  }

  async function setFranchiseForCurrentMovie() {
    if (!movie || !franchiseSearch.trim()) return;
    try {
      await updateMovie(movie.id, { franchise: franchiseSearch.trim() });
      showFranchiseModal = false;
      await loadMovie();
    } catch (err) {
      console.error('Failed to set franchise:', err);
    }
  }

  async function addIMDBMovieToFranchise(imdbResult: IMDBResult) {
    if (!franchiseSearch.trim()) return;
    try {
      // Sync movie from IMDB to local DB with franchise
      const res = await fetch('/api/v2/sync_movie', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          imdb_code: imdbResult.id,
          franchise: franchiseSearch.trim()
        }),
      });
      if (res.ok) {
        // Remove from results
        franchiseSearchResults = franchiseSearchResults.filter(r => r.id !== imdbResult.id);
        // Reload franchise movies
        loadFranchiseMovies();
      }
    } catch (err) {
      console.error('Failed to add movie:', err);
    }
  }

  async function refreshMovieData() {
    if (!movie) return;
    refreshing = true;
    try {
      const res = await fetch('/api/v2/refresh_movie', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ movie_id: movie.id }),
      });
      if (res.ok) {
        await loadMovie(); // Reload movie data
      } else {
        const data = await res.json();
        alert('Failed to refresh: ' + (data.status_message || 'Unknown error'));
      }
    } catch (err) {
      console.error('Failed to refresh movie:', err);
    } finally {
      refreshing = false;
    }
  }

  async function loadSubtitles() {
    if (!movie?.imdb_code) return;
    loadingSubtitles = true;
    try {
      const res = await getSubtitles(movie.imdb_code);
      subtitles = res.subtitles || [];
    } catch (err) {
      console.error('Failed to load subtitles:', err);
    } finally {
      loadingSubtitles = false;
    }
  }

  async function previewSubtitle(id: number) {
    loadingPreview = true;
    showPreviewModal = true;
    subtitlePreview = null;
    try {
      subtitlePreview = await getSubtitlePreview(id);
    } catch (err) {
      console.error('Failed to load preview:', err);
    } finally {
      loadingPreview = false;
    }
  }

  async function handleDeleteSubtitle(id: number) {
    try {
      await deleteSubtitle(id);
      subtitles = subtitles.filter(s => s.id !== id);
    } catch (err) {
      console.error('Failed to delete subtitle:', err);
    }
  }

  async function handleSyncSubtitles() {
    if (!movie?.imdb_code) return;
    syncingSubtitles = true;
    try {
      const res = await syncSubtitles(movie.imdb_code);
      await loadSubtitles();
    } catch (err) {
      console.error('Failed to sync subtitles:', err);
    } finally {
      syncingSubtitles = false;
    }
  }

  function openEditModal() {
    if (!movie) return;
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
        status: movieForm.status,
        release_date: movieForm.release_date,
        franchise: movieForm.franchise,
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
          <button class="btn btn-primary" on:click={refreshMovieData} disabled={refreshing || !movie?.imdb_code}>
            {refreshing ? 'REFRESHING...' : 'REFRESH DATA'}
          </button>
          <button class="btn btn-secondary" on:click={openFranchiseModal}>FIND FRANCHISE</button>
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

    <!-- Details Grid -->
    <div class="card details-card">
      <h3>Details</h3>
      <div class="details-grid">
        <div class="detail-row">
          <span class="detail-label">Title</span>
          <span class="detail-value">{movie.title}</span>
        </div>
        {#if movie.title_english && movie.title_english !== movie.title}
          <div class="detail-row">
            <span class="detail-label">English Title</span>
            <span class="detail-value">{movie.title_english}</span>
          </div>
        {/if}
        {#if movie.title_long}
          <div class="detail-row">
            <span class="detail-label">Full Title</span>
            <span class="detail-value">{movie.title_long}</span>
          </div>
        {/if}
        <div class="detail-row">
          <span class="detail-label">Year</span>
          <span class="detail-value">{movie.year}</span>
        </div>
        {#if movie.runtime}
          <div class="detail-row">
            <span class="detail-label">Runtime</span>
            <span class="detail-value">{movie.runtime} min ({Math.floor(movie.runtime / 60)}h {movie.runtime % 60}m)</span>
          </div>
        {/if}
        {#if movie.director}
          <div class="detail-row">
            <span class="detail-label">Director</span>
            <span class="detail-value">{movie.director}</span>
          </div>
        {/if}
        {#if movie.writers && movie.writers.length > 0}
          <div class="detail-row">
            <span class="detail-label">Writers</span>
            <span class="detail-value">{movie.writers.join(', ')}</span>
          </div>
        {/if}
        {#if movie.language}
          <div class="detail-row">
            <span class="detail-label">Language</span>
            <span class="detail-value">{movie.language.toUpperCase()}</span>
          </div>
        {/if}
        {#if movie.country}
          <div class="detail-row">
            <span class="detail-label">Country</span>
            <span class="detail-value">{movie.country}</span>
          </div>
        {/if}
        {#if movie.mpa_rating}
          <div class="detail-row">
            <span class="detail-label">Rating</span>
            <span class="detail-value">{movie.mpa_rating}</span>
          </div>
        {/if}
        {#if movie.genres && movie.genres.length > 0}
          <div class="detail-row">
            <span class="detail-label">Genres</span>
            <span class="detail-value">{movie.genres.join(', ')}</span>
          </div>
        {/if}
        {#if movie.franchise}
          <div class="detail-row">
            <span class="detail-label">Franchise</span>
            <span class="detail-value">{movie.franchise}</span>
          </div>
        {/if}
        {#if movie.awards}
          <div class="detail-row">
            <span class="detail-label">Awards</span>
            <span class="detail-value">{movie.awards}</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Franchise Section -->
    {#if movie.franchise && franchiseMovies.length > 0}
      <div class="card franchise-card">
        <h3>{movie.franchise} Franchise</h3>

        {#if loadingFranchise}
          <p class="text-muted">Loading...</p>
        {:else}
          <div class="franchise-section">
            <div class="franchise-grid">
              {#each franchiseMovies as fm}
                <a href="#/movies/{fm.id}" class="franchise-movie">
                  {#if fm.medium_cover_image}
                    <img src={fm.medium_cover_image} alt={fm.title} class="franchise-poster" />
                  {:else}
                    <div class="franchise-poster franchise-placeholder">
                      <span>{fm.title.charAt(0)}</span>
                    </div>
                  {/if}
                  <div class="franchise-info">
                    <span class="franchise-title">{fm.title}</span>
                    <span class="franchise-year">{fm.year}</span>
                  </div>
                </a>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    {/if}

    <!-- Box Office -->
    {#if movie.budget || movie.box_office_gross}
      <div class="card box-office-card">
        <h3>Box Office</h3>
        <div class="box-office-grid">
          {#if movie.budget}
            <div class="box-office-item">
              <span class="box-office-label">Budget</span>
              <span class="box-office-value">{movie.budget}</span>
            </div>
          {/if}
          {#if movie.box_office_gross}
            <div class="box-office-item">
              <span class="box-office-label">Worldwide Gross</span>
              <span class="box-office-value">{movie.box_office_gross}</span>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Ratings Card -->
    {#if movie.imdb_rating || movie.rotten_tomatoes || movie.metacritic || movie.rating}
      <div class="card ratings-card">
        <h3>Ratings</h3>
        <div class="ratings-grid">
          {#if movie.imdb_rating}
            <div class="rating-box rating-imdb">
              <div class="rating-icon">
                <svg width="32" height="32" viewBox="0 0 24 24" fill="#f5c518">
                  <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/>
                </svg>
              </div>
              <div class="rating-info">
                <span class="rating-source">IMDb</span>
                <span class="rating-value">{movie.imdb_rating}<span class="rating-max">/10</span></span>
                {#if movie.imdb_votes}
                  <span class="rating-votes">{movie.imdb_votes} votes</span>
                {/if}
              </div>
            </div>
          {/if}
          {#if movie.rotten_tomatoes}
            <div class="rating-box" class:rating-rt-fresh={movie.rotten_tomatoes >= 60} class:rating-rt-rotten={movie.rotten_tomatoes < 60}>
              <div class="rating-icon">
                {#if movie.rotten_tomatoes >= 60}
                  <span class="rt-icon fresh">🍅</span>
                {:else}
                  <span class="rt-icon rotten">🤢</span>
                {/if}
              </div>
              <div class="rating-info">
                <span class="rating-source">Rotten Tomatoes</span>
                <span class="rating-value">{movie.rotten_tomatoes}<span class="rating-max">%</span></span>
                <span class="rating-label">{movie.rotten_tomatoes >= 60 ? 'Fresh' : 'Rotten'}</span>
              </div>
            </div>
          {/if}
          {#if movie.metacritic}
            <div class="rating-box rating-mc" class:mc-good={movie.metacritic >= 60} class:mc-mixed={(movie.metacritic >= 40 && movie.metacritic < 60)} class:mc-bad={movie.metacritic < 40}>
              <div class="rating-icon mc-score" class:mc-good={movie.metacritic >= 60} class:mc-mixed={(movie.metacritic >= 40 && movie.metacritic < 60)} class:mc-bad={movie.metacritic < 40}>
                {movie.metacritic}
              </div>
              <div class="rating-info">
                <span class="rating-source">Metacritic</span>
                <span class="rating-label">
                  {#if movie.metacritic >= 60}
                    Generally Favorable
                  {:else if movie.metacritic >= 40}
                    Mixed Reviews
                  {:else}
                    Generally Unfavorable
                  {/if}
                </span>
              </div>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Stats Card -->
    {#if movie.like_count || movie.download_count}
      <div class="card stats-card">
        <h3>Statistics</h3>
        <div class="stats-grid">
          {#if movie.like_count}
            <div class="stat-item">
              <span class="stat-value">{movie.like_count.toLocaleString()}</span>
              <span class="stat-label">Likes</span>
            </div>
          {/if}
          {#if movie.download_count}
            <div class="stat-item">
              <span class="stat-value">{movie.download_count.toLocaleString()}</span>
              <span class="stat-label">Downloads</span>
            </div>
          {/if}
        </div>
      </div>
    {/if}

    <!-- Technical Info -->
    <div class="card tech-card">
      <h3>Technical Info</h3>
      <div class="details-grid">
        <div class="detail-row">
          <span class="detail-label">Database ID</span>
          <span class="detail-value mono">{movie.id}</span>
        </div>
        {#if movie.imdb_code}
          <div class="detail-row">
            <span class="detail-label">IMDB Code</span>
            <span class="detail-value">
              <a href="https://www.imdb.com/title/{movie.imdb_code}" target="_blank" rel="noopener" class="mono link">{movie.imdb_code}</a>
            </span>
          </div>
        {/if}
        {#if movie.slug}
          <div class="detail-row">
            <span class="detail-label">Slug</span>
            <span class="detail-value mono">{movie.slug}</span>
          </div>
        {/if}
        {#if movie.provider}
          <div class="detail-row">
            <span class="detail-label">Provider</span>
            <span class="detail-value">{movie.provider}</span>
          </div>
        {/if}
        {#if movie.content_type}
          <div class="detail-row">
            <span class="detail-label">Content Type</span>
            <span class="detail-value">{movie.content_type}</span>
          </div>
        {/if}
        {#if movie.date_uploaded}
          <div class="detail-row">
            <span class="detail-label">Date Uploaded</span>
            <span class="detail-value">{movie.date_uploaded}</span>
          </div>
        {/if}
        {#if movie.yt_trailer_code}
          <div class="detail-row">
            <span class="detail-label">YouTube Trailer</span>
            <span class="detail-value">
              <a href="https://www.youtube.com/watch?v={movie.yt_trailer_code}" target="_blank" rel="noopener" class="mono link">{movie.yt_trailer_code}</a>
            </span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Images Card -->
    <div class="card images-card">
      <h3>Images ({(movie.all_images?.length || 0) + (movie.medium_cover_image ? 1 : 0) + (movie.background_image ? 1 : 0)})</h3>
      <div class="images-grid">
        {#if movie.medium_cover_image}
          <div class="image-item">
            <span class="image-label">Poster</span>
            <a href={movie.medium_cover_image} target="_blank" rel="noopener">
              <img src={movie.medium_cover_image} alt="Poster" class="preview-image" />
            </a>
          </div>
        {/if}
        {#if movie.background_image}
          <div class="image-item image-wide">
            <span class="image-label">Background</span>
            <a href={movie.background_image} target="_blank" rel="noopener">
              <img src={movie.background_image} alt="Background" class="preview-image preview-wide" />
            </a>
          </div>
        {/if}
        {#if movie.all_images && movie.all_images.length > 0}
          {#each movie.all_images.slice(0, 12) as imgUrl, i}
            <div class="image-item">
              <span class="image-label">Image {i + 1}</span>
              <a href={imgUrl} target="_blank" rel="noopener">
                <img src={imgUrl} alt="Image {i + 1}" class="preview-image" loading="lazy" />
              </a>
            </div>
          {/each}
          {#if movie.all_images.length > 12}
            <div class="image-item more-images">
              <span class="more-count">+{movie.all_images.length - 12} more</span>
            </div>
          {/if}
        {/if}
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
        <div class="torrents-actions">
          <button class="btn btn-sm btn-secondary" on:click={handleSyncSubtitles} disabled={syncingSubtitles || !movie.imdb_code}>
            {syncingSubtitles ? 'SYNCING...' : 'SYNC SUBS'}
          </button>
          <button class="btn btn-sm btn-primary" on:click={() => showTorrentModal = true}>ADD TORRENT</button>
        </div>
      </div>
      {#if movie.torrents && movie.torrents.length > 0}
        {#each movie.torrents as torrent}
          <div class="torrent-section">
            <div class="torrent-row" role="button" tabindex="0" on:click={() => expandedTorrentHash = expandedTorrentHash === torrent.hash ? null : torrent.hash} on:keypress={(e) => e.key === 'Enter' && (expandedTorrentHash = expandedTorrentHash === torrent.hash ? null : torrent.hash)}>
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="chevron" class:expanded={expandedTorrentHash === torrent.hash}>
                <polyline points="9 18 15 12 9 6"/>
              </svg>
              <span class="torrent-quality badge badge-{torrent.quality}">{torrent.quality}</span>
              <span class="torrent-type">{torrent.type}</span>
              <span class="torrent-size">{torrent.size}</span>
              <span class="torrent-hash" title={torrent.hash}>{torrent.hash.substring(0, 8)}...</span>
              {#if subtitles.length > 0}
                <span class="subtitle-count-badge" title="{subtitles.length} subtitle(s)">{subtitles.length} sub{subtitles.length !== 1 ? 's' : ''}</span>
              {/if}
            </div>
            {#if expandedTorrentHash === torrent.hash}
              <div class="torrent-subtitles">
                {#if loadingSubtitles}
                  <p class="text-muted sub-loading">Loading subtitles...</p>
                {:else if subtitles.length > 0}
                  {#each subtitles as sub}
                    <div class="subtitle-row">
                      <span class="subtitle-lang badge">{sub.language_name || sub.language}</span>
                      <span class="subtitle-release">{sub.release_name || 'Unknown'}</span>
                      {#if sub.source}
                        <span class="subtitle-source badge badge-source">{sub.source}</span>
                      {/if}
                      {#if sub.hearing_impaired}
                        <span class="subtitle-hi badge badge-hi">HI</span>
                      {/if}
                      <div class="subtitle-actions">
                        <button class="btn btn-xs btn-secondary" on:click|stopPropagation={() => previewSubtitle(sub.id)}>Preview</button>
                        <button class="btn btn-xs btn-danger" on:click|stopPropagation={() => handleDeleteSubtitle(sub.id)}>Delete</button>
                      </div>
                    </div>
                  {/each}
                {:else}
                  <p class="text-muted sub-empty">No subtitles synced</p>
                {/if}
              </div>
            {/if}
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
        <p class="form-hint">Mark as upcoming release</p>
      </div>
      {#if movieForm.status === 'coming_soon'}
        <div class="form-group">
          <label class="form-label" for="release_date">Release Date</label>
          <input type="date" id="release_date" class="form-input" bind:value={movieForm.release_date} />
        </div>
      {/if}
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

<!-- Find Franchise Modal -->
<Modal bind:open={showFranchiseModal} title="Find Franchise" size="lg" on:close={() => showFranchiseModal = false}>
  <div class="franchise-modal">
    <div class="form-group">
      <label class="form-label" for="franchise_name">Franchise Name</label>
      <div class="search-row">
        <input
          type="text"
          id="franchise_name"
          class="form-input"
          bind:value={franchiseSearch}
          placeholder="e.g., Avengers, Spider-Man, Batman"
          on:input={() => searchFranchiseMovies()}
        />
      </div>
      <p class="text-muted" style="margin-top: 8px; font-size: 13px;">
        Search IMDB for movies to add to this franchise
      </p>
    </div>

    {#if searchingFranchise}
      <div class="loading-small">Searching IMDB...</div>
    {:else if franchiseSearchResults.length > 0}
      <div class="franchise-results">
        <p class="results-label">Movies from IMDB ({franchiseSearchResults.length}):</p>
        <div class="franchise-grid">
          {#each franchiseSearchResults as fm}
            <div class="franchise-result-item">
              {#if fm.primaryImage?.url}
                <img src={fm.primaryImage.url} alt={fm.primaryTitle} />
              {:else}
                <div class="no-poster">
                  <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
                    <rect x="2" y="2" width="20" height="20" rx="2"/>
                    <line x1="7" y1="2" x2="7" y2="22"/>
                    <line x1="17" y1="2" x2="17" y2="22"/>
                  </svg>
                </div>
              {/if}
              <div class="result-info">
                <span class="result-title">{fm.primaryTitle}</span>
                <span class="result-year">{fm.startYear}</span>
                {#if fm.rating?.aggregateRating}
                  <span class="result-rating">IMDB: {fm.rating.aggregateRating}</span>
                {/if}
              </div>
              <button class="btn btn-sm btn-primary add-btn" on:click={() => addIMDBMovieToFranchise(fm)} title="Add to franchise">
                +
              </button>
            </div>
          {/each}
        </div>
      </div>
    {:else if franchiseSearch.trim()}
      <p class="text-muted">No movies found matching "{franchiseSearch}"</p>
    {/if}
  </div>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showFranchiseModal = false}>Cancel</button>
    <button
      class="btn btn-primary"
      on:click={setFranchiseForCurrentMovie}
      disabled={!franchiseSearch.trim()}
    >
      Set "{franchiseSearch}" as Franchise
    </button>
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

<!-- Subtitle Preview Modal -->
<Modal bind:open={showPreviewModal} title="Subtitle Preview" size="lg" on:close={() => showPreviewModal = false}>
  {#if loadingPreview}
    <div class="loading-small">Loading preview...</div>
  {:else if subtitlePreview}
    <div class="preview-info">
      <span class="badge">{subtitlePreview.language}</span>
      <span class="preview-release">{subtitlePreview.release_name}</span>
      <span class="text-muted">({subtitlePreview.total_lines} total lines)</span>
    </div>
    <pre class="vtt-preview">{subtitlePreview.preview}</pre>
  {:else}
    <p class="text-muted">Failed to load preview</p>
  {/if}
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showPreviewModal = false}>Close</button>
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

  .torrents-actions {
    display: flex;
    gap: 8px;
  }

  .torrent-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 4px;
    cursor: pointer;
    border-radius: 6px;
    transition: background 0.15s;
  }

  .torrent-row:hover {
    background: var(--bg-tertiary);
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

  .checkbox-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }

  .checkbox-label input[type="checkbox"] {
    width: 18px;
    height: 18px;
    cursor: pointer;
  }

  .form-hint {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  .empty-state {
    text-align: center;
    padding: 48px;
  }

  .empty-state p {
    margin-bottom: 16px;
    color: var(--text-muted);
  }

  /* Details Card */
  .details-card,
  .ratings-card,
  .stats-card,
  .tech-card,
  .images-card {
    margin-bottom: 24px;
  }

  .details-card h3,
  .ratings-card h3,
  .stats-card h3,
  .tech-card h3,
  .images-card h3 {
    margin-bottom: 16px;
    font-size: 16px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .details-grid {
    display: grid;
    gap: 12px;
  }

  .detail-row {
    display: grid;
    grid-template-columns: 140px 1fr;
    gap: 16px;
    padding: 8px 0;
    border-bottom: 1px solid var(--border-color);
  }

  .detail-row:last-child {
    border-bottom: none;
  }

  .detail-label {
    color: var(--text-muted);
    font-size: 13px;
    font-weight: 500;
  }

  .detail-value {
    color: var(--text-primary);
    font-size: 14px;
  }

  .detail-value.mono,
  .mono {
    font-family: 'SF Mono', Monaco, 'Cascadia Code', monospace;
    font-size: 13px;
  }

  .link {
    color: var(--accent-blue);
    text-decoration: none;
  }

  .link:hover {
    text-decoration: underline;
  }

  /* Ratings Card */
  .ratings-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }

  .rating-box {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;
    background: var(--bg-tertiary);
    border-radius: 12px;
  }

  .rating-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    flex-shrink: 0;
  }

  .rt-icon {
    font-size: 32px;
  }

  .mc-score {
    width: 48px;
    height: 48px;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 20px;
    font-weight: 700;
    color: white;
    border-radius: 8px;
  }

  .mc-score.mc-good {
    background: #66cc33;
  }

  .mc-score.mc-mixed {
    background: #ffcc33;
    color: #333;
  }

  .mc-score.mc-bad {
    background: #ff0000;
  }

  .rating-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .rating-source {
    font-size: 12px;
    color: var(--text-muted);
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .rating-value {
    font-size: 24px;
    font-weight: 700;
    color: var(--text-primary);
  }

  .rating-max {
    font-size: 14px;
    font-weight: 400;
    color: var(--text-muted);
  }

  .rating-votes,
  .rating-label {
    font-size: 12px;
    color: var(--text-muted);
  }

  .rating-rt-fresh .rating-value {
    color: #21d07a;
  }

  .rating-rt-rotten .rating-value {
    color: #fa320a;
  }

  /* Stats Card */
  .stats-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 16px;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 20px;
    background: var(--bg-tertiary);
    border-radius: 12px;
    text-align: center;
  }

  .stat-value {
    font-size: 28px;
    font-weight: 700;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .stat-label {
    font-size: 13px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  /* Images Card */
  .images-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
    gap: 16px;
  }

  .image-item {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .image-item.image-wide {
    grid-column: span 2;
  }

  .image-label {
    font-size: 12px;
    color: var(--text-muted);
    font-weight: 500;
  }

  .preview-image {
    width: 100%;
    max-width: 150px;
    height: auto;
    border-radius: 8px;
    border: 1px solid var(--border-color);
    transition: transform 0.2s, box-shadow 0.2s;
  }

  .preview-image:hover {
    transform: scale(1.05);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  }

  .preview-wide {
    max-width: 300px;
  }

  /* Franchise Section */
  .franchise-card {
    margin-bottom: 24px;
  }

  .franchise-card h3 {
    margin-bottom: 16px;
    font-size: 16px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .franchise-section {
    margin-bottom: 20px;
  }

  .franchise-section:last-child {
    margin-bottom: 0;
  }

  .franchise-subtitle {
    font-size: 13px;
    color: var(--text-muted);
    margin-bottom: 12px;
    font-weight: 500;
  }

  .franchise-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 12px;
  }

  .franchise-movie {
    position: relative;
    display: flex;
    flex-direction: column;
    background: var(--bg-tertiary);
    border-radius: 8px;
    overflow: hidden;
    text-decoration: none;
    color: inherit;
    transition: transform 0.2s, box-shadow 0.2s;
  }

  .franchise-movie:hover {
    transform: translateY(-4px);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.3);
  }

  .franchise-poster {
    width: 100%;
    aspect-ratio: 2/3;
    object-fit: cover;
  }

  .franchise-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-secondary);
    font-size: 32px;
    font-weight: 600;
    color: var(--text-muted);
  }

  .franchise-info {
    padding: 8px 10px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .franchise-title {
    font-size: 12px;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .franchise-year {
    font-size: 11px;
    color: var(--text-muted);
  }

  .franchise-movie.suggested {
    border: 2px dashed var(--border-color);
  }

  .add-franchise-btn {
    position: absolute;
    top: 8px;
    right: 8px;
    width: 28px;
    height: 28px;
    padding: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 50%;
    opacity: 0;
    transition: opacity 0.2s;
  }

  .franchise-movie.suggested:hover .add-franchise-btn {
    opacity: 1;
  }

  /* Franchise Modal */
  .franchise-modal {
    min-height: 200px;
  }

  .search-row {
    display: flex;
    gap: 8px;
  }

  .search-row .form-input {
    flex: 1;
  }

  .loading-small {
    padding: 20px;
    text-align: center;
    color: var(--text-muted);
  }

  .franchise-results {
    margin-top: 16px;
  }

  .results-label {
    font-size: 14px;
    font-weight: 500;
    margin-bottom: 12px;
    color: var(--text-secondary);
  }

  .franchise-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
    gap: 12px;
    max-height: 300px;
    overflow-y: auto;
  }

  .franchise-result-item {
    display: flex;
    gap: 12px;
    padding: 10px;
    background: var(--bg-tertiary);
    border-radius: 8px;
    align-items: center;
  }

  .franchise-result-item img {
    width: 45px;
    height: 67px;
    object-fit: cover;
    border-radius: 4px;
  }

  .franchise-result-item .no-poster {
    width: 45px;
    height: 67px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-secondary);
    border-radius: 4px;
    color: var(--text-muted);
  }

  .result-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
    flex: 1;
    min-width: 0;
  }

  .result-title {
    font-weight: 500;
    font-size: 14px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .result-year {
    font-size: 12px;
    color: var(--text-muted);
  }

  .result-franchise {
    font-size: 11px;
    color: var(--accent-color);
  }

  .result-rating {
    font-size: 11px;
    color: #f5c518;
  }

  .franchise-result-item {
    position: relative;
  }

  .franchise-result-item .add-btn {
    position: absolute;
    right: 8px;
    top: 50%;
    transform: translateY(-50%);
    width: 28px;
    height: 28px;
    padding: 0;
    font-size: 18px;
    line-height: 1;
    border-radius: 50%;
  }

  /* Box Office */
  .box-office-card {
    margin-bottom: 24px;
  }

  .box-office-card h3 {
    margin-bottom: 16px;
    font-size: 16px;
    font-weight: 600;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .box-office-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 16px;
  }

  .box-office-item {
    display: flex;
    flex-direction: column;
    padding: 16px;
    background: var(--bg-tertiary);
    border-radius: 12px;
  }

  .box-office-label {
    font-size: 12px;
    color: var(--text-muted);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 4px;
  }

  .box-office-value {
    font-size: 24px;
    font-weight: 700;
    color: var(--accent-green, #22c55e);
  }

  /* More images indicator */
  .more-images {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-tertiary);
    border-radius: 8px;
    min-height: 100px;
  }

  .more-count {
    font-size: 14px;
    color: var(--text-muted);
    font-weight: 500;
  }

  /* Expandable Torrent Rows */
  .torrent-section {
    border-bottom: 1px solid var(--border-color);
  }

  .torrent-section:last-child {
    border-bottom: none;
  }

  .chevron {
    transition: transform 0.2s;
    color: var(--text-muted);
    flex-shrink: 0;
  }

  .chevron.expanded {
    transform: rotate(90deg);
  }

  .subtitle-count-badge {
    font-size: 11px;
    padding: 2px 8px;
    background: rgba(59, 130, 246, 0.15);
    color: var(--accent-blue);
    border-radius: 10px;
    margin-left: auto;
  }

  .torrent-subtitles {
    padding: 8px 0 12px 30px;
    border-top: 1px solid var(--border-color);
    background: var(--bg-tertiary);
    border-radius: 0 0 6px 6px;
    margin: 0 -4px 4px -4px;
    padding-left: 34px;
    padding-right: 12px;
  }

  .sub-loading, .sub-empty {
    padding: 8px 0;
    font-size: 13px;
  }

  .subtitle-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 0;
    border-bottom: 1px solid var(--border-color);
  }

  .subtitle-row:last-child {
    border-bottom: none;
  }

  .subtitle-lang {
    min-width: 40px;
    text-align: center;
    font-weight: 600;
    font-size: 11px;
    padding: 3px 6px;
    background: var(--accent-blue);
    color: white;
    border-radius: 4px;
  }

  .subtitle-release {
    flex: 1;
    font-size: 12px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .badge-source {
    font-size: 10px;
    padding: 2px 6px;
    background: var(--bg-secondary);
    color: var(--text-muted);
    border-radius: 4px;
  }

  .badge-hi {
    font-size: 10px;
    padding: 2px 5px;
    background: rgba(245, 158, 11, 0.2);
    color: #f59e0b;
    border-radius: 4px;
  }

  .subtitle-actions {
    display: flex;
    gap: 6px;
    flex-shrink: 0;
  }

  .btn-xs {
    padding: 4px 8px;
    font-size: 11px;
    line-height: 1;
  }

  /* Subtitle Preview Modal */
  .preview-info {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 16px;
  }

  .preview-release {
    font-weight: 500;
    font-size: 14px;
  }

  .vtt-preview {
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 16px;
    font-size: 12px;
    line-height: 1.6;
    overflow-x: auto;
    max-height: 400px;
    overflow-y: auto;
    white-space: pre-wrap;
    color: var(--text-secondary);
  }
</style>
