<script lang="ts">
  import { onMount } from 'svelte';
  import { link, push } from 'svelte-spa-router';
  import { getSeriesDetails, deleteSeries, updateSeries, getSubtitles, getSubtitlePreview, deleteSubtitle, syncSubtitles, type SeriesDetails, type Episode, type StoredSubtitle, type SubtitlePreview } from '../lib/api/client';
  import Modal from '../lib/components/Modal.svelte';

  export let params: { id: string };

  let series: SeriesDetails | null = null;
  let loading = true;
  let fetchingImdb = false;
  let refreshing = false;

  // Grouped episodes by season
  let episodesBySeason: Map<number, Episode[]> = new Map();
  let expandedSeasons: Set<number> = new Set();

  // Subtitles
  let subtitles: StoredSubtitle[] = [];
  let loadingSubtitles = false;
  let showPreviewModal = false;
  let subtitlePreview: SubtitlePreview | null = null;
  let loadingPreview = false;
  let syncingSubtitles = false;

  // Modal states
  let showEditModal = false;
  let showDeleteModal = false;
  let showTorrentModal = false;
  let showSeasonTorrentModal = false;
  let selectedEpisode: Episode | null = null;
  let selectedSeason: number | null = null;

  // Torrent form
  let torrentForm = {
    hash: '',
    quality: '1080p',
    size: '',
  };

  // Season torrent form
  let seasonTorrentForm = {
    magnet: '',
    quality: '1080p',
    totalSize: '',
  };

  // Form
  let seriesForm = {
    imdb_code: '',
    title: '',
    year: 0,
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
      series = await getSeriesDetails(parseInt(params.id));
      if (series) {
        groupEpisodesBySeason();
        // Expand first season by default
        if (episodesBySeason.size > 0) {
          expandedSeasons.add(Math.min(...episodesBySeason.keys()));
        }
        loadSubtitles();
      }
    } catch (err) {
      console.error('Failed to load series:', err);
    } finally {
      loading = false;
    }
  }

  function groupEpisodesBySeason() {
    episodesBySeason = new Map();
    if (series?.episodes) {
      for (const ep of series.episodes) {
        const season = ep.season_number;
        if (!episodesBySeason.has(season)) {
          episodesBySeason.set(season, []);
        }
        episodesBySeason.get(season)!.push(ep);
      }
      // Sort episodes within each season
      for (const [season, episodes] of episodesBySeason) {
        episodes.sort((a, b) => a.episode_number - b.episode_number);
      }
    }
  }

  function toggleSeason(season: number) {
    if (expandedSeasons.has(season)) {
      expandedSeasons.delete(season);
    } else {
      expandedSeasons.add(season);
    }
    expandedSeasons = expandedSeasons; // Trigger reactivity
  }

  async function refreshSeriesData() {
    if (!series) return;
    refreshing = true;
    try {
      const res = await fetch('/api/v2/refresh_series', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ series_id: series.id }),
      });
      if (res.ok) {
        await loadSeries();
      } else {
        const data = await res.json();
        alert('Failed to refresh: ' + (data.status_message || 'Unknown error'));
      }
    } catch (err) {
      console.error('Failed to refresh series:', err);
    } finally {
      refreshing = false;
    }
  }

  async function loadSubtitles() {
    if (!series?.imdb_code) return;
    loadingSubtitles = true;
    try {
      const res = await getSubtitles(series.imdb_code);
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
    if (!series?.imdb_code) return;
    syncingSubtitles = true;
    try {
      await syncSubtitles(series.imdb_code);
      await loadSubtitles();
    } catch (err) {
      console.error('Failed to sync subtitles:', err);
    } finally {
      syncingSubtitles = false;
    }
  }

  function openEditModal() {
    if (!series) return;
    seriesForm = {
      imdb_code: series.imdb_code || '',
      title: series.title,
      year: series.year,
      rating: series.rating || 0,
      runtime: series.runtime || 0,
      genres: series.genres?.join(', ') || '',
      summary: series.summary || '',
      poster_image: series.poster_image || '',
      background_image: series.background_image || '',
      total_seasons: series.total_seasons || 1,
      status: series.status || 'Continuing',
      network: series.network || '',
    };
    showEditModal = true;
  }

  async function handleEditSeries() {
    if (!series) return;
    try {
      await updateSeries(series.id, {
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
      await loadSeries();
    } catch (err) {
      console.error('Failed to update series:', err);
    }
  }

  async function handleDeleteSeries() {
    if (!series) return;
    try {
      await deleteSeries(series.id);
      showDeleteModal = false;
      push('/tvshows');
    } catch (err) {
      console.error('Failed to delete series:', err);
    }
  }

  async function fetchFromImdb() {
    if (!seriesForm.imdb_code) return;
    fetchingImdb = true;
    try {
      const res = await fetch(`/admin/api/imdb/title/${seriesForm.imdb_code}`);
      if (res.ok) {
        const data = await res.json();
        seriesForm.title = data.primaryTitle || seriesForm.title;
        seriesForm.year = data.startYear || seriesForm.year;
        seriesForm.rating = data.rating?.aggregateRating || seriesForm.rating;
        seriesForm.runtime = data.runtimeSeconds ? Math.round(data.runtimeSeconds / 60) : seriesForm.runtime;
        seriesForm.genres = data.genres?.join(', ') || seriesForm.genres;
        seriesForm.summary = data.plot || seriesForm.summary;
        seriesForm.poster_image = data.primaryImage?.url || seriesForm.poster_image;
        if (data.totalSeasons) seriesForm.total_seasons = data.totalSeasons;
        seriesForm.status = data.endYear ? 'Ended' : 'Continuing';
      }
    } catch (err) {
      console.error('Failed to fetch from IMDB:', err);
    } finally {
      fetchingImdb = false;
    }
  }

  function getSeasonPack(season: number) {
    return series?.season_packs?.find(p => p.season === season);
  }

  function openTorrentModal(episode: Episode) {
    selectedEpisode = episode;
    torrentForm = { hash: '', quality: '1080p', size: '' };
    showTorrentModal = true;
  }

  async function handleAddTorrent() {
    if (!selectedEpisode || !series) return;
    try {
      const res = await fetch(`/admin/episodes/${selectedEpisode.id}/torrent`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
          'Accept': 'application/json'
        },
        body: new URLSearchParams({
          hash: torrentForm.hash,
          quality: torrentForm.quality,
          size: torrentForm.size,
          series_id: String(series.id),
          season_number: String(selectedEpisode.season_number),
          episode_number: String(selectedEpisode.episode_number),
        }),
      });
      if (res.ok) {
        showTorrentModal = false;
        selectedEpisode = null;
        torrentForm = { hash: '', quality: '1080p', size: '' };
        await loadSeries();
      } else {
        const data = await res.json();
        alert('Failed to add torrent: ' + (data.error || 'Unknown error'));
      }
    } catch (err) {
      console.error('Failed to add torrent:', err);
    }
  }

  async function expandSeasonPack(pack: { id?: number; season: number; quality: string }) {
    if (!pack.id) {
      // Find the pack ID from our data
      const foundPack = series?.season_packs?.find(p => p.season === pack.season && p.quality === pack.quality);
      if (!foundPack) {
        alert('Season pack not found');
        return;
      }
      // We need to query for the ID - for now use a workaround
    }

    try {
      const res = await fetch(`/admin/api/season-packs/${pack.id || 0}/expand`, {
        method: 'POST',
        headers: { 'Accept': 'application/json' },
      });
      const data = await res.json();
      if (res.ok && data.success) {
        alert(`Created ${data.created} episode torrents from season pack`);
        await loadSeries();
      } else {
        alert('Failed to expand season pack: ' + (data.error || 'Unknown error'));
      }
    } catch (err) {
      console.error('Failed to expand season pack:', err);
    }
  }

  function openSeasonTorrentModal(season: number) {
    selectedSeason = season;
    seasonTorrentForm = { magnet: '', quality: '1080p', totalSize: '' };
    showSeasonTorrentModal = true;
  }

  function extractHashFromMagnet(magnet: string): string {
    // Extract hash from magnet link: magnet:?xt=urn:btih:HASH&...
    const match = magnet.match(/btih:([a-fA-F0-9]{40})/i);
    if (match) return match[1].toUpperCase();
    // Also try base32 encoded hash
    const base32Match = magnet.match(/btih:([A-Za-z2-7]{32})/i);
    if (base32Match) return base32Match[1].toUpperCase();
    return magnet.trim().toUpperCase(); // Assume it's just the hash
  }

  async function handleAddSeasonTorrent() {
    if (!series || selectedSeason === null) return;

    const hash = extractHashFromMagnet(seasonTorrentForm.magnet);
    if (!hash) {
      alert('Please enter a valid magnet link or hash');
      return;
    }

    // Get episodes for this season
    const seasonEpisodes = episodesBySeason.get(selectedSeason) || [];
    if (seasonEpisodes.length === 0) {
      alert('No episodes found for this season');
      return;
    }

    // Calculate per-episode size if total size provided
    let perEpisodeSize = '';
    if (seasonTorrentForm.totalSize) {
      const match = seasonTorrentForm.totalSize.match(/([\d.]+)\s*(GB|MB|TB)/i);
      if (match) {
        const value = parseFloat(match[1]);
        const unit = match[2].toUpperCase();
        const perEpValue = value / seasonEpisodes.length;
        perEpisodeSize = `${perEpValue.toFixed(1)} ${unit}`;
      }
    }

    let created = 0;
    let failed = 0;

    for (let i = 0; i < seasonEpisodes.length; i++) {
      const ep = seasonEpisodes[i];
      try {
        const res = await fetch(`/admin/episodes/${ep.id}/torrent`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
            'Accept': 'application/json'
          },
          body: new URLSearchParams({
            hash: hash,
            quality: seasonTorrentForm.quality,
            size: perEpisodeSize,
            series_id: String(series.id),
            season_number: String(selectedSeason),
            episode_number: String(ep.episode_number),
            file_index: String(i),
          }),
        });
        if (res.ok) {
          created++;
        } else {
          failed++;
        }
      } catch {
        failed++;
      }
    }

    showSeasonTorrentModal = false;
    selectedSeason = null;

    if (created > 0) {
      alert(`Added torrent to ${created} episodes` + (failed > 0 ? ` (${failed} failed)` : ''));
      await loadSeries();
    } else {
      alert('Failed to add torrents');
    }
  }
</script>

<div class="detail-page">
  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if !series}
    <div class="error-state">
      <h2>Series not found</h2>
      <a href="#/tvshows" class="btn btn-primary">Back to TV Shows</a>
    </div>
  {:else}
    <!-- Header with background -->
    <div class="detail-header" style="background-image: url('{series.background_image || series.poster_image}')">
      <div class="header-overlay">
        <div class="header-content">
          <a href="#/tvshows" class="back-link">
            <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M19 12H5M12 19l-7-7 7-7"/>
            </svg>
            Back to TV Shows
          </a>

          <div class="header-main">
            {#if series.poster_image}
              <img src={series.poster_image} alt={series.title} class="detail-poster" />
            {:else}
              <div class="detail-poster poster-placeholder">
                <span>{series.title.charAt(0)}</span>
              </div>
            {/if}

            <div class="header-info">
              <h1 class="detail-title">{series.title}</h1>

              <div class="detail-meta">
                <span class="meta-item">{series.year}</span>
                {#if series.runtime}
                  <span class="meta-item">{series.runtime} min/ep</span>
                {/if}
                <span class="meta-item">{series.total_seasons} Season{series.total_seasons !== 1 ? 's' : ''}</span>
                <span class="status-badge status-{series.status?.toLowerCase()}">{series.status}</span>
              </div>

              {#if series.genres && series.genres.length > 0}
                <div class="genre-tags">
                  {#each series.genres as genre}
                    <span class="genre-tag">{genre}</span>
                  {/each}
                </div>
              {/if}

              {#if series.rating}
                <div class="header-rating">
                  <svg width="20" height="20" viewBox="0 0 24 24" fill="#f5c518">
                    <path d="M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z"/>
                  </svg>
                  <span class="rating-value">{series.rating.toFixed(1)}</span>
                  <span class="rating-max">/10</span>
                </div>
              {/if}

              {#if series.summary}
                <p class="detail-summary">{series.summary}</p>
              {/if}

              <div class="header-actions">
                <button class="btn btn-primary" on:click={refreshSeriesData} disabled={refreshing || !series.imdb_code}>
                  {#if refreshing}
                    <span class="spinner-sm"></span> Refreshing...
                  {:else}
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <path d="M23 4v6h-6M1 20v-6h6"/>
                      <path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/>
                    </svg>
                    Refresh from IMDB
                  {/if}
                </button>
                <button class="btn btn-secondary" on:click={openEditModal}>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/>
                    <path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/>
                  </svg>
                  Edit
                </button>
                <button class="btn btn-danger" on:click={() => showDeleteModal = true}>
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M3 6h18M19 6v14a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2V6m3 0V4a2 2 0 0 1 2-2h4a2 2 0 0 1 2 2v2"/>
                  </svg>
                  Delete
                </button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Details Grid -->
    <div class="card details-card compact">
      <h3>Details</h3>
      <div class="details-grid-compact">
        <div class="detail-item">
          <span class="detail-label">Year</span>
          <span class="detail-value">{series.year}{#if series.end_year && series.end_year !== series.year}-{series.end_year}{/if}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Status</span>
          <span class="detail-value">{series.status}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Seasons</span>
          <span class="detail-value">{series.total_seasons}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Episodes</span>
          <span class="detail-value">{series.total_episodes || series.episodes?.length || 0}</span>
        </div>
        {#if series.runtime}
          <div class="detail-item">
            <span class="detail-label">Runtime</span>
            <span class="detail-value">{series.runtime} min</span>
          </div>
        {/if}
        {#if series.network}
          <div class="detail-item">
            <span class="detail-label">Network</span>
            <span class="detail-value">{series.network}</span>
          </div>
        {/if}
        {#if series.genres && series.genres.length > 0}
          <div class="detail-item full-width">
            <span class="detail-label">Genres</span>
            <span class="detail-value">{series.genres.join(', ')}</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Episodes Section -->
    <div class="card episodes-card">
      <h3>Episodes ({series.episodes?.length || 0})</h3>

      {#if episodesBySeason.size === 0}
        <p class="text-muted">No episodes available</p>
      {:else}
        <div class="seasons-list">
          {#each [...episodesBySeason.entries()].sort((a, b) => a[0] - b[0]) as [season, episodes]}
            {@const seasonPack = getSeasonPack(season)}
            <div class="season-section">
              <div class="season-header" role="button" tabindex="0" on:click={() => toggleSeason(season)} on:keypress={(e) => e.key === 'Enter' && toggleSeason(season)}>
                <div class="season-title">
                  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="chevron" class:expanded={expandedSeasons.has(season)}>
                    <polyline points="9 18 15 12 9 6"/>
                  </svg>
                  <span>Season {season}</span>
                  <span class="episode-count">{episodes.length} episodes</span>
                </div>
                <div class="season-pack-actions">
                  {#if seasonPack}
                    <span class="season-pack-badge" title="Season pack available">
                      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
                        <polyline points="7 10 12 15 17 10"/>
                        <line x1="12" y1="15" x2="12" y2="3"/>
                      </svg>
                      {seasonPack.quality} • {seasonPack.size}
                    </span>
                    <button
                      class="btn btn-xs btn-expand"
                      on:click|stopPropagation={() => expandSeasonPack(seasonPack)}
                      title="Create episode torrents from season pack"
                    >
                      Expand
                    </button>
                  {/if}
                  <button
                    class="btn btn-xs btn-add-season"
                    on:click|stopPropagation={() => openSeasonTorrentModal(season)}
                    title="Add magnet link for entire season"
                  >
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                      <line x1="12" y1="5" x2="12" y2="19"/>
                      <line x1="5" y1="12" x2="19" y2="12"/>
                    </svg>
                    Add Season
                  </button>
                </div>
              </div>

              {#if expandedSeasons.has(season)}
                <div class="episodes-list">
                  {#each episodes as episode}
                    <div class="episode-item">
                      <div class="episode-number">E{episode.episode_number}</div>
                      <div class="episode-info">
                        <span class="episode-title">{episode.title || `Episode ${episode.episode_number}`}</span>
                        {#if episode.air_date}
                          <span class="episode-date">{episode.air_date}</span>
                        {/if}
                        {#if episode.summary}
                          <p class="episode-summary">{episode.summary}</p>
                        {/if}
                      </div>
                      <div class="episode-torrents">
                        {#if episode.torrents && episode.torrents.length > 0}
                          {#each episode.torrents as torrent}
                            <span class="torrent-badge badge-{torrent.quality}">{torrent.quality}</span>
                          {/each}
                        {/if}
                        <button class="btn btn-xs btn-add-torrent" on:click={() => openTorrentModal(episode)} title="Add torrent">
                          <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                            <line x1="12" y1="5" x2="12" y2="19"/>
                            <line x1="5" y1="12" x2="19" y2="12"/>
                          </svg>
                        </button>
                      </div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- Technical Info -->
    <div class="card tech-card">
      <h3>Technical Info</h3>
      <div class="details-grid">
        <div class="detail-row">
          <span class="detail-label">Database ID</span>
          <span class="detail-value mono">{series.id}</span>
        </div>
        {#if series.imdb_code}
          <div class="detail-row">
            <span class="detail-label">IMDB Code</span>
            <span class="detail-value">
              <a href="https://www.imdb.com/title/{series.imdb_code}" target="_blank" rel="noopener" class="mono link">{series.imdb_code}</a>
            </span>
          </div>
        {/if}
        {#if series.title_slug}
          <div class="detail-row">
            <span class="detail-label">Slug</span>
            <span class="detail-value mono">{series.title_slug}</span>
          </div>
        {/if}
        {#if series.date_added}
          <div class="detail-row">
            <span class="detail-label">Date Added</span>
            <span class="detail-value">{series.date_added}</span>
          </div>
        {/if}
      </div>
    </div>

    <!-- Images Card -->
    <div class="card images-card">
      <h3>Images</h3>
      <div class="images-grid">
        {#if series.poster_image}
          <div class="image-item">
            <span class="image-label">Poster</span>
            <a href={series.poster_image} target="_blank" rel="noopener">
              <img src={series.poster_image} alt="Poster" class="preview-image" />
            </a>
          </div>
        {/if}
        {#if series.background_image}
          <div class="image-item image-wide">
            <span class="image-label">Background</span>
            <a href={series.background_image} target="_blank" rel="noopener">
              <img src={series.background_image} alt="Background" class="preview-image preview-wide" />
            </a>
          </div>
        {/if}
      </div>
    </div>

    <!-- Subtitles Card -->
    <div class="card subtitles-card">
      <div class="subtitles-header">
        <h3>Subtitles ({subtitles.length})</h3>
        <button class="btn btn-sm btn-secondary" on:click={handleSyncSubtitles} disabled={syncingSubtitles || !series.imdb_code}>
          {syncingSubtitles ? 'SYNCING...' : 'SYNC SUBS'}
        </button>
      </div>
      {#if subtitles.length > 0}
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
              <button class="btn btn-xs btn-secondary" on:click={() => previewSubtitle(sub.id)}>Preview</button>
              <button class="btn btn-xs btn-danger" on:click={() => handleDeleteSubtitle(sub.id)}>Delete</button>
            </div>
          </div>
        {/each}
      {:else}
        <p class="text-muted">No subtitles synced</p>
      {/if}
    </div>
  {/if}
</div>

<!-- Edit Modal -->
<Modal bind:open={showEditModal} title="Edit Series" size="lg" on:close={() => showEditModal = false}>
  <form on:submit|preventDefault={handleEditSeries}>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="edit_imdb_code">IMDB Code</label>
        <div class="input-with-button">
          <input type="text" id="edit_imdb_code" class="form-input" bind:value={seriesForm.imdb_code} placeholder="tt1234567" />
          <button type="button" class="btn btn-sm btn-secondary" on:click={fetchFromImdb} disabled={fetchingImdb || !seriesForm.imdb_code}>
            {fetchingImdb ? 'Updating...' : 'Fetch'}
          </button>
        </div>
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

<!-- Delete Modal -->
<Modal bind:open={showDeleteModal} title="Delete Series" size="sm" on:close={() => showDeleteModal = false}>
  <p class="delete-warning">
    Are you sure you want to delete <strong>{series?.title}</strong>?
  </p>
  <p class="text-muted">This action cannot be undone. All episodes and torrents will also be removed.</p>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showDeleteModal = false}>Cancel</button>
    <button class="btn btn-danger" on:click={handleDeleteSeries}>Delete</button>
  </svelte:fragment>
</Modal>

<!-- Add Torrent Modal -->
<Modal bind:open={showTorrentModal} title="Add Episode Torrent" size="md" on:close={() => showTorrentModal = false}>
  {#if selectedEpisode}
    <p class="torrent-episode-info">
      <strong>S{String(selectedEpisode.season_number).padStart(2, '0')}E{String(selectedEpisode.episode_number).padStart(2, '0')}</strong> - {selectedEpisode.title || `Episode ${selectedEpisode.episode_number}`}
    </p>
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
          <option value="480p">480p</option>
          <option value="720p">720p</option>
          <option value="1080p">1080p</option>
          <option value="2160p">2160p (4K)</option>
        </select>
      </div>
      <div class="form-group">
        <label class="form-label" for="torrent_size">Size</label>
        <input type="text" id="torrent_size" class="form-input" bind:value={torrentForm.size} placeholder="500 MB" />
      </div>
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showTorrentModal = false}>Cancel</button>
    <button class="btn btn-primary" on:click={handleAddTorrent}>Add Torrent</button>
  </svelte:fragment>
</Modal>

<!-- Add Season Torrent Modal -->
<Modal bind:open={showSeasonTorrentModal} title="Add Season Torrent" size="md" on:close={() => showSeasonTorrentModal = false}>
  {#if selectedSeason !== null}
    <p class="torrent-season-info">
      <strong>Season {selectedSeason}</strong> - {episodesBySeason.get(selectedSeason)?.length || 0} episodes
    </p>
    <p class="text-muted text-sm">
      Paste a magnet link for the entire season. A torrent entry will be created for each episode with the correct file index.
    </p>
  {/if}
  <form on:submit|preventDefault={handleAddSeasonTorrent}>
    <div class="form-group">
      <label class="form-label" for="season_magnet">Magnet Link / Hash *</label>
      <textarea
        id="season_magnet"
        class="form-input form-textarea"
        bind:value={seasonTorrentForm.magnet}
        placeholder="magnet:?xt=urn:btih:... or just the hash"
        rows="3"
        required
      ></textarea>
    </div>
    <div class="form-row">
      <div class="form-group">
        <label class="form-label" for="season_quality">Quality</label>
        <select id="season_quality" class="form-input" bind:value={seasonTorrentForm.quality}>
          <option value="480p">480p</option>
          <option value="720p">720p</option>
          <option value="1080p">1080p</option>
          <option value="2160p">2160p (4K)</option>
        </select>
      </div>
      <div class="form-group">
        <label class="form-label" for="season_size">Total Size (optional)</label>
        <input type="text" id="season_size" class="form-input" bind:value={seasonTorrentForm.totalSize} placeholder="16.1 GB" />
        <span class="form-hint">Will be divided among episodes</span>
      </div>
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" on:click={() => showSeasonTorrentModal = false}>Cancel</button>
    <button class="btn btn-primary" on:click={handleAddSeasonTorrent}>Add to All Episodes</button>
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
  .detail-page {
    min-height: 100vh;
  }

  .detail-header {
    position: relative;
    background-size: cover;
    background-position: center top;
    margin: -24px -24px 24px -24px;
  }

  .header-overlay {
    background: linear-gradient(to bottom,
      rgba(17, 17, 17, 0.7) 0%,
      rgba(17, 17, 17, 0.9) 50%,
      rgba(17, 17, 17, 1) 100%
    );
    padding: 24px;
  }

  .header-content {
    max-width: 1200px;
    margin: 0 auto;
  }

  .back-link {
    display: inline-flex;
    align-items: center;
    gap: 8px;
    color: var(--text-secondary);
    text-decoration: none;
    margin-bottom: 24px;
    font-size: 14px;
  }

  .back-link:hover {
    color: var(--text-primary);
  }

  .header-main {
    display: flex;
    gap: 32px;
  }

  .detail-poster {
    width: 200px;
    height: 300px;
    object-fit: cover;
    border-radius: 8px;
    flex-shrink: 0;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.5);
  }

  .poster-placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-tertiary);
    font-size: 64px;
    font-weight: 700;
    color: var(--text-muted);
  }

  .header-info {
    flex: 1;
    padding-top: 16px;
  }

  .detail-title {
    font-size: 32px;
    font-weight: 700;
    margin: 0 0 12px 0;
    color: var(--text-primary);
  }

  .detail-meta {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 16px;
  }

  .meta-item {
    color: var(--text-secondary);
    font-size: 14px;
  }

  .status-badge {
    padding: 4px 12px;
    border-radius: 4px;
    font-size: 12px;
    font-weight: 600;
  }

  .status-continuing {
    background: rgba(46, 160, 67, 0.2);
    color: var(--accent-green);
  }

  .status-ended {
    background: rgba(139, 148, 158, 0.2);
    color: var(--text-muted);
  }

  .genre-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 16px;
  }

  .genre-tag {
    padding: 4px 12px;
    background: var(--bg-tertiary);
    border-radius: 16px;
    font-size: 12px;
    color: var(--text-secondary);
  }

  .header-rating {
    display: flex;
    align-items: center;
    gap: 6px;
    margin-bottom: 16px;
  }

  .rating-value {
    font-size: 24px;
    font-weight: 700;
    color: #f5c518;
  }

  .rating-max {
    font-size: 14px;
    color: var(--text-muted);
  }

  .detail-summary {
    color: var(--text-secondary);
    line-height: 1.6;
    margin-bottom: 24px;
    max-width: 700px;
  }

  .header-actions {
    display: flex;
    gap: 12px;
    flex-wrap: wrap;
  }

  .header-actions .btn {
    display: inline-flex;
    align-items: center;
    gap: 8px;
  }

  /* Cards */
  .card {
    background: var(--bg-secondary);
    border-radius: 8px;
    padding: 24px;
    margin-bottom: 24px;
  }

  .card h3 {
    margin: 0 0 16px 0;
    font-size: 18px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .card.compact {
    padding: 16px;
  }

  .card.compact h3 {
    margin: 0 0 12px 0;
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-muted);
  }

  .details-grid-compact {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
    gap: 12px 24px;
  }

  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .detail-item.full-width {
    grid-column: 1 / -1;
  }

  .detail-item .detail-label {
    font-size: 11px;
    text-transform: uppercase;
    letter-spacing: 0.3px;
  }

  .detail-item .detail-value {
    font-size: 14px;
    font-weight: 500;
    text-align: left;
  }

  .details-grid {
    display: grid;
    gap: 12px;
  }

  .detail-row {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    border-bottom: 1px solid var(--border-color);
  }

  .detail-row:last-child {
    border-bottom: none;
  }

  .detail-label {
    color: var(--text-muted);
    font-size: 14px;
  }

  .detail-value {
    color: var(--text-primary);
    font-size: 14px;
    text-align: right;
  }

  .mono {
    font-family: monospace;
  }

  .link {
    color: var(--accent-blue);
    text-decoration: none;
  }

  .link:hover {
    text-decoration: underline;
  }

  /* Episodes */
  .seasons-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .season-section {
    border: 1px solid var(--border-color);
    border-radius: 8px;
    overflow: hidden;
  }

  .season-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    padding: 16px;
    background: var(--bg-tertiary);
    border: none;
    color: var(--text-primary);
    cursor: pointer;
    text-align: left;
  }

  .season-header:hover {
    background: var(--bg-primary);
  }

  .season-title {
    display: flex;
    align-items: center;
    gap: 12px;
    font-weight: 600;
  }

  .chevron {
    transition: transform 0.2s;
  }

  .chevron.expanded {
    transform: rotate(90deg);
  }

  .episode-count {
    font-weight: 400;
    color: var(--text-muted);
    font-size: 14px;
  }

  .season-pack-actions {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .season-pack-badge {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 10px;
    background: rgba(59, 130, 246, 0.2);
    color: var(--accent-blue);
    border-radius: 4px;
    font-size: 12px;
  }

  .btn-expand {
    padding: 4px 8px;
    font-size: 11px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    color: var(--text-secondary);
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-expand:hover {
    background: var(--accent-blue);
    border-color: var(--accent-blue);
    color: white;
  }

  .btn-add-season {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    font-size: 11px;
    background: transparent;
    border: 1px dashed var(--border-color);
    color: var(--text-muted);
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s;
  }

  .btn-add-season:hover {
    background: rgba(139, 92, 246, 0.1);
    border-color: var(--accent-purple);
    border-style: solid;
    color: var(--accent-purple);
  }

  .episodes-list {
    padding: 8px;
  }

  .episode-item {
    display: flex;
    align-items: flex-start;
    gap: 16px;
    padding: 12px;
    border-radius: 6px;
  }

  .episode-item:hover {
    background: var(--bg-tertiary);
  }

  .episode-number {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-primary);
    border-radius: 8px;
    font-weight: 700;
    font-size: 14px;
    color: var(--text-secondary);
    flex-shrink: 0;
  }

  .episode-info {
    flex: 1;
    min-width: 0;
  }

  .episode-title {
    display: block;
    font-weight: 500;
    color: var(--text-primary);
    margin-bottom: 4px;
  }

  .episode-date {
    display: block;
    font-size: 12px;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  .episode-summary {
    font-size: 13px;
    color: var(--text-secondary);
    margin: 0;
    line-height: 1.4;
  }

  .episode-torrents {
    display: flex;
    gap: 4px;
    flex-shrink: 0;
  }

  .torrent-badge {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
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

  .badge-480p {
    background: #6b7280;
    color: white;
  }

  .btn-xs {
    padding: 4px 6px;
    font-size: 10px;
    line-height: 1;
  }

  .btn-add-torrent {
    background: transparent;
    border: 1px dashed var(--border-color);
    color: var(--text-muted);
    border-radius: 4px;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    opacity: 0.5;
    transition: all 0.2s;
  }

  .btn-add-torrent:hover {
    opacity: 1;
    border-color: var(--accent-blue);
    color: var(--accent-blue);
    background: rgba(59, 130, 246, 0.1);
  }

  .torrent-episode-info {
    margin-bottom: 16px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 6px;
  }

  .torrent-season-info {
    margin-bottom: 8px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 6px;
  }

  .text-sm {
    font-size: 13px;
  }

  .form-hint {
    display: block;
    margin-top: 4px;
    font-size: 11px;
    color: var(--text-muted);
  }

  /* Images */
  .images-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
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
  }

  .preview-image {
    width: 100%;
    border-radius: 6px;
    object-fit: cover;
  }

  .preview-wide {
    aspect-ratio: 16/9;
  }

  /* Forms */
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

  .input-with-button {
    display: flex;
    gap: 8px;
  }

  .input-with-button .form-input {
    flex: 1;
  }

  .delete-warning {
    margin-bottom: 12px;
  }

  .spinner-sm {
    width: 14px;
    height: 14px;
    border: 2px solid var(--border-color);
    border-top-color: white;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .loading, .error-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 400px;
    gap: 16px;
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border-color);
    border-top-color: var(--accent-red);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  /* Subtitles Card */
  .subtitles-card {
    margin-bottom: 24px;
  }

  .subtitles-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .subtitles-header h3 {
    margin: 0 !important;
  }

  .subtitle-row {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 10px 0;
    border-bottom: 1px solid var(--border-color);
  }

  .subtitle-row:last-child {
    border-bottom: none;
  }

  .subtitle-lang {
    min-width: 40px;
    text-align: center;
    font-weight: 600;
    font-size: 12px;
    padding: 4px 8px;
    background: var(--accent-blue);
    color: white;
    border-radius: 4px;
  }

  .subtitle-release {
    flex: 1;
    font-size: 13px;
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .badge-source {
    font-size: 11px;
    padding: 2px 6px;
    background: var(--bg-tertiary);
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

  .subtitle-date {
    font-size: 12px;
    color: var(--text-muted);
  }

  .subtitle-actions {
    display: flex;
    gap: 6px;
    margin-left: auto;
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

  .loading-small {
    padding: 20px;
    text-align: center;
    color: var(--text-muted);
  }
</style>
