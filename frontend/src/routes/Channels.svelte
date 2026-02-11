<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    getChannels,
    getChannelCountries,
    getChannelCategories,
    getChannelStats,
    getChannelSettings,
    updateChannelSettings,
    syncIPTVChannels,
    getIPTVSyncStatus,
    deleteChannel,
    startHealthCheck,
    getHealthCheckStatus,
    clearBlocklist,
    getLicenseStatus,
    type Channel,
    type ChannelCountry,
    type ChannelCategory,
  } from '../lib/api/client';

  let notFound = false;
  let channels: Channel[] = [];
  let countries: ChannelCountry[] = [];
  let categories: ChannelCategory[] = [];
  let total = 0;
  let loading = true;
  let search = '';
  let page = 1;
  let limit = 50;
  let selectedCountry = '';
  let selectedCategory = '';

  // Stats
  let stats = { channels: 0, countries: 0, categories: 0, with_streams: 0, blocklisted: 0 };

  // Sync state
  let syncing = false;
  let syncPhase = '';
  let syncProgress = 0;
  let syncTotal = 0;
  let lastSync = '';
  let syncError = '';
  let syncPollInterval: ReturnType<typeof setInterval> | null = null;

  // Health check state
  let healthChecking = false;
  let healthPhase = '';
  let healthTotal = 0;
  let healthChecked = 0;
  let healthRemoved = 0;
  let healthError = '';
  let healthPollInterval: ReturnType<typeof setInterval> | null = null;
  let showClearBlocklistConfirm = false;

  // M3U settings
  let m3uUrl = '';
  let m3uSaved = false;

  // View mode
  let viewMode: 'list' | 'countries' | 'categories' = 'list';

  // Delete
  let showDeleteConfirm = false;
  let channelToDelete: Channel | null = null;

  $: totalPages = Math.ceil(total / limit);

  onMount(async () => {
    try {
      const data = await getLicenseStatus();
      const features: string[] = data.features || [];
      if (!features.includes('live_channels')) {
        notFound = true;
        loading = false;
        return;
      }
    } catch {
      notFound = true;
      loading = false;
      return;
    }
    await Promise.all([loadChannels(), loadFilters(), loadStats(), checkSyncStatus(), loadSettings(), checkHealthStatus()]);
  });

  onDestroy(() => {
    if (syncPollInterval) clearInterval(syncPollInterval);
    if (healthPollInterval) clearInterval(healthPollInterval);
  });

  async function loadChannels() {
    loading = true;
    try {
      const result = await getChannels({
        page,
        limit,
        country: selectedCountry || undefined,
        category: selectedCategory || undefined,
        query_term: search || undefined,
      });
      channels = result.channels;
      total = result.total;
    } catch (err) {
      console.error('Failed to load channels:', err);
    } finally {
      loading = false;
    }
  }

  async function loadFilters() {
    try {
      const [c, cat] = await Promise.all([getChannelCountries(), getChannelCategories()]);
      countries = c;
      categories = cat;
    } catch (err) {
      console.error('Failed to load filters:', err);
    }
  }

  async function loadStats() {
    try {
      stats = await getChannelStats();
    } catch (err) {
      console.error('Failed to load stats:', err);
    }
  }

  async function checkSyncStatus() {
    try {
      const status = await getIPTVSyncStatus();
      syncing = status.running;
      syncPhase = status.phase;
      syncProgress = status.progress;
      syncTotal = status.total;
      lastSync = status.last_sync || '';
      syncError = status.last_error || '';

      if (syncing) {
        startSyncPolling();
      }
    } catch {
      // ignore
    }
  }

  async function loadSettings() {
    try {
      const settings = await getChannelSettings();
      m3uUrl = settings.m3u_url;
    } catch {
      // ignore
    }
  }

  async function saveM3UUrl() {
    try {
      await updateChannelSettings(m3uUrl);
      m3uSaved = true;
      setTimeout(() => m3uSaved = false, 2000);
    } catch (err: any) {
      syncError = err.message || 'Failed to save';
    }
  }

  async function handleSync() {
    try {
      await syncIPTVChannels(m3uUrl);
      syncing = true;
      syncPhase = 'starting';
      syncError = '';
      startSyncPolling();
    } catch (err: any) {
      syncError = err.message || 'Sync failed';
    }
  }

  function startSyncPolling() {
    if (syncPollInterval) clearInterval(syncPollInterval);
    syncPollInterval = setInterval(async () => {
      try {
        const status = await getIPTVSyncStatus();
        syncing = status.running;
        syncPhase = status.phase;
        syncProgress = status.progress;
        syncTotal = status.total;
        lastSync = status.last_sync || '';
        syncError = status.last_error || '';

        if (!status.running) {
          clearInterval(syncPollInterval!);
          syncPollInterval = null;
          // Refresh data after sync completes
          await Promise.all([loadChannels(), loadFilters(), loadStats()]);
        }
      } catch {
        // ignore
      }
    }, 2000);
  }

  async function checkHealthStatus() {
    try {
      const status = await getHealthCheckStatus();
      healthChecking = status.running;
      healthPhase = status.phase;
      healthTotal = status.total;
      healthChecked = status.checked;
      healthRemoved = status.removed;
      healthError = status.last_error || '';

      if (healthChecking) {
        startHealthPolling();
      }
    } catch {
      // ignore
    }
  }

  async function handleHealthCheck() {
    try {
      await startHealthCheck();
      healthChecking = true;
      healthPhase = 'starting';
      healthChecked = 0;
      healthTotal = 0;
      healthRemoved = 0;
      healthError = '';
      startHealthPolling();
    } catch (err: any) {
      healthError = err.message || 'Health check failed';
    }
  }

  function startHealthPolling() {
    if (healthPollInterval) clearInterval(healthPollInterval);
    healthPollInterval = setInterval(async () => {
      try {
        const status = await getHealthCheckStatus();
        healthChecking = status.running;
        healthPhase = status.phase;
        healthTotal = status.total;
        healthChecked = status.checked;
        healthRemoved = status.removed;
        healthError = status.last_error || '';

        if (!status.running) {
          clearInterval(healthPollInterval!);
          healthPollInterval = null;
          await Promise.all([loadChannels(), loadStats()]);
        }
      } catch {
        // ignore
      }
    }, 2000);
  }

  async function handleClearBlocklist() {
    try {
      await clearBlocklist();
      showClearBlocklistConfirm = false;
      await loadStats();
    } catch (err: any) {
      healthError = err.message || 'Failed to clear blocklist';
    }
  }

  function handleSearch() {
    page = 1;
    loadChannels();
  }

  function handleCountryFilter(code: string) {
    selectedCountry = code;
    selectedCategory = '';
    page = 1;
    viewMode = 'list';
    loadChannels();
  }

  function handleCategoryFilter(id: string) {
    selectedCategory = id;
    selectedCountry = '';
    page = 1;
    viewMode = 'list';
    loadChannels();
  }

  function clearFilters() {
    selectedCountry = '';
    selectedCategory = '';
    search = '';
    page = 1;
    loadChannels();
  }

  function goToPage(p: number) {
    if (p >= 1 && p <= totalPages) {
      page = p;
      loadChannels();
    }
  }

  function confirmDelete(ch: Channel) {
    channelToDelete = ch;
    showDeleteConfirm = true;
  }

  async function handleDelete() {
    if (!channelToDelete) return;
    try {
      await deleteChannel(channelToDelete.id);
      showDeleteConfirm = false;
      channelToDelete = null;
      await loadChannels();
      await loadStats();
    } catch (err) {
      console.error('Failed to delete channel:', err);
    }
  }

  function formatLastSync(ts: string): string {
    if (!ts) return 'Never';
    const d = new Date(ts);
    return d.toLocaleDateString() + ' ' + d.toLocaleTimeString();
  }
</script>

{#if notFound}
<div class="not-found">
  <h1>404</h1>
  <p>Page not found</p>
</div>
{:else}
<div class="channels-page">
  <header class="page-header">
    <h1 class="page-title">LIVE CHANNELS</h1>
    <div class="page-actions">
      <div class="search-box">
        <input
          type="text"
          class="form-input search-input"
          placeholder="Search channels..."
          bind:value={search}
          on:keydown={(e) => e.key === 'Enter' && handleSearch()}
        />
      </div>
      <button class="btn btn-primary" on:click={handleSync} disabled={syncing}>
        {syncing ? 'SYNCING...' : 'SYNC IPTV'}
      </button>
    </div>
  </header>

  <!-- Sync Status Banner -->
  {#if syncing}
    <div class="sync-banner">
      <div class="sync-info">
        <span class="sync-spinner"></span>
        <span class="sync-text">{syncPhase}</span>
        {#if syncTotal > 0}
          <span class="sync-progress">({syncProgress}/{syncTotal})</span>
        {/if}
      </div>
      <div class="sync-bar">
        <div class="sync-bar-fill" style="width: {syncTotal > 0 ? (syncProgress / syncTotal * 100) : 0}%"></div>
      </div>
    </div>
  {/if}

  {#if syncError}
    <div class="sync-error">Sync error: {syncError}</div>
  {/if}

  <!-- Stats Cards -->
  <div class="stats-row">
    <div class="stat-card">
      <div class="stat-value">{stats.channels.toLocaleString()}</div>
      <div class="stat-label">Channels</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">{stats.countries}</div>
      <div class="stat-label">Countries</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">{stats.categories}</div>
      <div class="stat-label">Categories</div>
    </div>
    <div class="stat-card">
      <div class="stat-value">{stats.with_streams.toLocaleString()}</div>
      <div class="stat-label">With Streams</div>
    </div>
    <div class="stat-card last-sync">
      <div class="stat-value text-sm">{formatLastSync(lastSync)}</div>
      <div class="stat-label">Last Sync</div>
    </div>
  </div>

  <!-- M3U Source Setting -->
  <div class="m3u-setting">
    <label class="m3u-label">M3U Source URL</label>
    <div class="m3u-input-row">
      <input
        type="text"
        class="form-input m3u-input"
        placeholder="https://iptv-org.github.io/iptv/index.m3u"
        bind:value={m3uUrl}
      />
      <button class="btn btn-secondary btn-sm" on:click={saveM3UUrl}>
        {m3uSaved ? 'SAVED' : 'SAVE'}
      </button>
      <button class="btn btn-primary btn-sm" on:click={handleSync} disabled={syncing}>
        {syncing ? 'SYNCING...' : 'SYNC'}
      </button>
    </div>
  </div>

  <!-- Stream Health Section -->
  <div class="health-section">
    <div class="health-header">
      <div>
        <label class="m3u-label">Stream Health Check</label>
        <span class="health-subtitle">Check all streams and blocklist dead channels</span>
      </div>
      <div class="health-actions">
        {#if stats.blocklisted > 0}
          <span class="blocklist-count">{stats.blocklisted} blocklisted</span>
          <button class="btn btn-secondary btn-sm" on:click={() => showClearBlocklistConfirm = true}>Clear Blocklist</button>
        {/if}
        <button class="btn btn-primary btn-sm" on:click={handleHealthCheck} disabled={healthChecking}>
          {healthChecking ? 'CHECKING...' : 'RUN HEALTH CHECK'}
        </button>
      </div>
    </div>

    {#if healthChecking}
      <div class="sync-banner health-banner">
        <div class="sync-info">
          <span class="sync-spinner"></span>
          <span class="sync-text">{healthPhase}</span>
          {#if healthTotal > 0}
            <span class="sync-progress">({healthChecked}/{healthTotal})</span>
          {/if}
          {#if healthRemoved > 0}
            <span class="health-removed">{healthRemoved} removed</span>
          {/if}
        </div>
        <div class="sync-bar">
          <div class="sync-bar-fill" style="width: {healthTotal > 0 ? (healthChecked / healthTotal * 100) : 0}%"></div>
        </div>
      </div>
    {/if}

    {#if !healthChecking && healthPhase === 'completed'}
      <div class="health-completed">
        Completed: {healthChecked} checked, {healthRemoved} removed
      </div>
    {/if}

    {#if healthError}
      <div class="sync-error">Health check error: {healthError}</div>
    {/if}
  </div>

  <!-- View Mode Tabs -->
  <div class="view-tabs">
    <button class="tab" class:active={viewMode === 'list'} on:click={() => { viewMode = 'list'; clearFilters(); }}>All Channels</button>
    <button class="tab" class:active={viewMode === 'countries'} on:click={() => viewMode = 'countries'}>By Country</button>
    <button class="tab" class:active={viewMode === 'categories'} on:click={() => viewMode = 'categories'}>By Category</button>
  </div>

  <!-- Active Filter -->
  {#if selectedCountry || selectedCategory}
    <div class="active-filter">
      <span>
        Filtered by:
        {#if selectedCountry}
          {countries.find(c => c.code === selectedCountry)?.flag || ''}
          {countries.find(c => c.code === selectedCountry)?.name || selectedCountry}
        {/if}
        {#if selectedCategory}
          {categories.find(c => c.id === selectedCategory)?.name || selectedCategory}
        {/if}
      </span>
      <button class="btn btn-sm btn-secondary" on:click={clearFilters}>Clear</button>
    </div>
  {/if}

  <!-- Countries Grid -->
  {#if viewMode === 'countries'}
    <div class="grid-container">
      {#each countries as country}
        <button class="grid-card" on:click={() => handleCountryFilter(country.code)}>
          <span class="grid-flag">{country.flag || ''}</span>
          <span class="grid-name">{country.name}</span>
          <span class="grid-count">{country.channel_count || 0}</span>
        </button>
      {/each}
    </div>
  {/if}

  <!-- Categories Grid -->
  {#if viewMode === 'categories'}
    <div class="grid-container">
      {#each categories as cat}
        <button class="grid-card" on:click={() => handleCategoryFilter(cat.id)}>
          <span class="grid-name">{cat.name}</span>
          <span class="grid-count">{cat.channel_count || 0}</span>
        </button>
      {/each}
    </div>
  {/if}

  <!-- Channel List -->
  {#if viewMode === 'list'}
    {#if loading}
      <div class="loading-state">Loading channels...</div>
    {:else if channels.length === 0}
      <div class="empty-state">
        <p>No channels found</p>
        {#if stats.channels === 0}
          <p class="text-muted">Click "SYNC IPTV" to fetch channels from iptv-org database</p>
        {/if}
      </div>
    {:else}
      <div class="channel-table-wrapper">
        <table class="data-table">
          <thead>
            <tr>
              <th>Channel</th>
              <th>Country</th>
              <th>Categories</th>
              <th>Stream</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {#each channels as ch}
              <tr>
                <td class="channel-cell">
                  {#if ch.logo}
                    <img src={ch.logo} alt={ch.name} class="channel-logo" />
                  {:else}
                    <div class="channel-logo-placeholder"></div>
                  {/if}
                  <div class="channel-info">
                    <span class="channel-name">{ch.name}</span>
                    <span class="channel-id text-muted">{ch.id}</span>
                  </div>
                </td>
                <td>
                  {#if ch.country}
                    <button class="country-tag" on:click={() => handleCountryFilter(ch.country || '')}>
                      {countries.find(c => c.code === ch.country)?.flag || ''}
                      {ch.country}
                    </button>
                  {:else}
                    <span class="text-muted">-</span>
                  {/if}
                </td>
                <td>
                  {#if ch.categories && ch.categories.length > 0}
                    <div class="category-tags">
                      {#each ch.categories as cat}
                        <button class="category-tag" on:click={() => handleCategoryFilter(cat)}>{cat}</button>
                      {/each}
                    </div>
                  {:else}
                    <span class="text-muted">-</span>
                  {/if}
                </td>
                <td>
                  {#if ch.stream_url}
                    <span class="stream-badge online">LIVE</span>
                  {:else}
                    <span class="stream-badge offline">NO STREAM</span>
                  {/if}
                </td>
                <td>
                  <button class="btn btn-sm btn-danger" on:click={() => confirmDelete(ch)}>DEL</button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      {#if totalPages > 1}
        <div class="pagination">
          <span class="pagination-info text-muted">
            Showing {(page - 1) * limit + 1}-{Math.min(page * limit, total)} of {total.toLocaleString()}
          </span>
          <div class="pagination-controls">
            <button class="btn btn-sm btn-secondary" disabled={page <= 1} on:click={() => goToPage(page - 1)}>Prev</button>
            {#each Array.from({length: Math.min(5, totalPages)}, (_, i) => {
              if (totalPages <= 5) return i + 1;
              if (page <= 3) return i + 1;
              if (page >= totalPages - 2) return totalPages - 4 + i;
              return page - 2 + i;
            }) as p}
              <button class="btn btn-sm" class:btn-primary={p === page} class:btn-secondary={p !== page} on:click={() => goToPage(p)}>{p}</button>
            {/each}
            <button class="btn btn-sm btn-secondary" disabled={page >= totalPages} on:click={() => goToPage(page + 1)}>Next</button>
          </div>
        </div>
      {/if}
    {/if}
  {/if}
</div>

<!-- Delete Confirmation -->
{#if showDeleteConfirm && channelToDelete}
  <div class="modal-backdrop" on:click={() => showDeleteConfirm = false} on:keydown={(e) => e.key === 'Escape' && (showDeleteConfirm = false)}>
    <div class="modal-content" on:click|stopPropagation on:keydown|stopPropagation>
      <h3>Delete Channel</h3>
      <p>Are you sure you want to delete <strong>{channelToDelete.name}</strong>?</p>
      <div class="modal-actions">
        <button class="btn btn-secondary" on:click={() => showDeleteConfirm = false}>Cancel</button>
        <button class="btn btn-danger" on:click={handleDelete}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<!-- Clear Blocklist Confirmation -->
{#if showClearBlocklistConfirm}
  <div class="modal-backdrop" on:click={() => showClearBlocklistConfirm = false} on:keydown={(e) => e.key === 'Escape' && (showClearBlocklistConfirm = false)}>
    <div class="modal-content" on:click|stopPropagation on:keydown|stopPropagation>
      <h3>Clear Blocklist</h3>
      <p>This will remove all {stats.blocklisted} entries from the blocklist. Previously dead channels may reappear on the next sync.</p>
      <div class="modal-actions">
        <button class="btn btn-secondary" on:click={() => showClearBlocklistConfirm = false}>Cancel</button>
        <button class="btn btn-danger" on:click={handleClearBlocklist}>Clear All</button>
      </div>
    </div>
  </div>
{/if}
{/if}

<style>
  .not-found {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 60vh;
    color: var(--text-muted);
  }

  .not-found h1 {
    font-size: 72px;
    font-weight: 700;
    color: var(--text-primary);
    margin: 0;
  }

  .not-found p {
    font-size: 18px;
    margin-top: 8px;
  }

  .channels-page {
    padding: 0;
  }

  .m3u-setting {
    background: var(--bg-card, #1c2128);
    border: 1px solid var(--bg-tertiary, #21262d);
    border-radius: 8px;
    padding: 14px 18px;
    margin-bottom: 16px;
  }

  .m3u-label {
    display: block;
    font-size: 0.75rem;
    color: var(--text-secondary, #8b949e);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
  }

  .m3u-input-row {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .m3u-input {
    flex: 1;
    padding: 8px 12px;
    background: var(--bg-tertiary, #21262d);
    border: 1px solid var(--bg-tertiary, #21262d);
    border-radius: 6px;
    color: var(--text-primary, #e6edf3);
    font-size: 0.85rem;
    font-family: monospace;
  }

  .m3u-input:focus {
    border-color: var(--accent-blue, #58a6ff);
    outline: none;
  }

  .stats-row {
    display: flex;
    gap: 12px;
    margin-bottom: 20px;
    flex-wrap: wrap;
  }

  .stat-card {
    background: var(--bg-card, #1c2128);
    border: 1px solid var(--bg-tertiary, #21262d);
    border-radius: 8px;
    padding: 16px 20px;
    flex: 1;
    min-width: 120px;
  }

  .stat-value {
    font-size: 1.5rem;
    font-weight: 700;
    color: var(--text-primary, #e6edf3);
  }

  .stat-value.text-sm {
    font-size: 0.85rem;
    font-weight: 500;
  }

  .stat-label {
    font-size: 0.75rem;
    color: var(--text-secondary, #8b949e);
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-top: 4px;
  }

  .sync-banner {
    background: var(--bg-card, #1c2128);
    border: 1px solid var(--accent-blue, #58a6ff);
    border-radius: 8px;
    padding: 12px 16px;
    margin-bottom: 16px;
  }

  .sync-info {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 8px;
  }

  .sync-spinner {
    width: 14px;
    height: 14px;
    border: 2px solid var(--accent-blue, #58a6ff);
    border-top-color: transparent;
    border-radius: 50%;
    animation: spin 0.8s linear infinite;
  }

  @keyframes spin {
    to { transform: rotate(360deg); }
  }

  .sync-text {
    color: var(--accent-blue, #58a6ff);
    font-size: 0.9rem;
  }

  .sync-progress {
    color: var(--text-secondary, #8b949e);
    font-size: 0.85rem;
  }

  .sync-bar {
    height: 4px;
    background: var(--bg-tertiary, #21262d);
    border-radius: 2px;
    overflow: hidden;
  }

  .sync-bar-fill {
    height: 100%;
    background: var(--accent-blue, #58a6ff);
    border-radius: 2px;
    transition: width 0.3s ease;
  }

  .sync-error {
    background: rgba(233, 69, 96, 0.1);
    border: 1px solid var(--accent-red, #e94560);
    color: var(--accent-red, #e94560);
    padding: 10px 16px;
    border-radius: 8px;
    margin-bottom: 16px;
    font-size: 0.9rem;
  }

  .health-section {
    background: var(--bg-card, #1c2128);
    border: 1px solid var(--bg-tertiary, #21262d);
    border-radius: 8px;
    padding: 14px 18px;
    margin-bottom: 16px;
  }

  .health-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: 10px;
  }

  .health-subtitle {
    font-size: 0.8rem;
    color: var(--text-secondary, #8b949e);
  }

  .health-actions {
    display: flex;
    gap: 8px;
    align-items: center;
  }

  .blocklist-count {
    font-size: 0.8rem;
    color: var(--accent-red, #e94560);
    background: rgba(233, 69, 96, 0.1);
    padding: 4px 10px;
    border-radius: 4px;
    font-weight: 500;
  }

  .health-banner {
    margin-top: 12px;
    margin-bottom: 0;
  }

  .health-removed {
    color: var(--accent-red, #e94560);
    font-size: 0.85rem;
    font-weight: 500;
  }

  .health-completed {
    margin-top: 10px;
    padding: 8px 12px;
    background: rgba(46, 160, 67, 0.1);
    border: 1px solid rgba(46, 160, 67, 0.3);
    border-radius: 6px;
    color: var(--accent-green, #2ea043);
    font-size: 0.85rem;
  }

  .view-tabs {
    display: flex;
    gap: 0;
    margin-bottom: 16px;
    border-bottom: 1px solid var(--bg-tertiary, #21262d);
  }

  .tab {
    background: none;
    border: none;
    color: var(--text-secondary, #8b949e);
    padding: 10px 20px;
    cursor: pointer;
    font-size: 0.9rem;
    border-bottom: 2px solid transparent;
    transition: all var(--transition-fast, 0.15s ease);
  }

  .tab:hover {
    color: var(--text-primary, #e6edf3);
  }

  .tab.active {
    color: var(--accent-blue, #58a6ff);
    border-bottom-color: var(--accent-blue, #58a6ff);
  }

  .active-filter {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 8px 14px;
    background: var(--bg-card, #1c2128);
    border-radius: 6px;
    margin-bottom: 16px;
    font-size: 0.9rem;
    color: var(--text-primary, #e6edf3);
  }

  .grid-container {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
    gap: 10px;
    margin-bottom: 20px;
  }

  .grid-card {
    background: var(--bg-card, #1c2128);
    border: 1px solid var(--bg-tertiary, #21262d);
    border-radius: 8px;
    padding: 14px;
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 6px;
    transition: all var(--transition-fast, 0.15s ease);
    text-align: center;
  }

  .grid-card:hover {
    border-color: var(--accent-blue, #58a6ff);
    background: var(--bg-tertiary, #21262d);
  }

  .grid-flag {
    font-size: 1.8rem;
  }

  .grid-name {
    font-size: 0.85rem;
    color: var(--text-primary, #e6edf3);
    font-weight: 500;
  }

  .grid-count {
    font-size: 0.75rem;
    color: var(--text-secondary, #8b949e);
  }

  .channel-table-wrapper {
    overflow-x: auto;
  }

  .channel-cell {
    display: flex;
    align-items: center;
    gap: 10px;
  }

  .channel-logo {
    width: 32px;
    height: 32px;
    object-fit: contain;
    border-radius: 4px;
    background: var(--bg-tertiary, #21262d);
    flex-shrink: 0;
  }

  .channel-logo-placeholder {
    width: 32px;
    height: 32px;
    border-radius: 4px;
    background: var(--bg-tertiary, #21262d);
    flex-shrink: 0;
  }

  .channel-info {
    display: flex;
    flex-direction: column;
  }

  .channel-name {
    font-weight: 500;
    color: var(--text-primary, #e6edf3);
  }

  .channel-id {
    font-size: 0.75rem;
  }

  .country-tag {
    background: var(--bg-tertiary, #21262d);
    border: none;
    border-radius: 4px;
    padding: 2px 8px;
    color: var(--text-primary, #e6edf3);
    font-size: 0.8rem;
    cursor: pointer;
  }

  .country-tag:hover {
    background: var(--accent-blue, #58a6ff);
    color: #fff;
  }

  .category-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .category-tag {
    background: var(--bg-tertiary, #21262d);
    border: none;
    border-radius: 4px;
    padding: 2px 8px;
    color: var(--text-secondary, #8b949e);
    font-size: 0.75rem;
    cursor: pointer;
  }

  .category-tag:hover {
    background: var(--accent-blue, #58a6ff);
    color: #fff;
  }

  .stream-badge {
    display: inline-block;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.7rem;
    font-weight: 600;
    letter-spacing: 0.5px;
  }

  .stream-badge.online {
    background: rgba(46, 160, 67, 0.15);
    color: var(--accent-green, #2ea043);
  }

  .stream-badge.offline {
    background: rgba(139, 148, 158, 0.1);
    color: var(--text-muted, #484f58);
  }

  .pagination {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: 16px;
    padding: 12px 0;
  }

  .pagination-controls {
    display: flex;
    gap: 4px;
  }

  .loading-state, .empty-state {
    text-align: center;
    padding: 60px 20px;
    color: var(--text-secondary, #8b949e);
  }

  .search-box {
    position: relative;
  }

  .search-input {
    width: 250px;
    padding: 8px 12px;
    background: var(--bg-tertiary, #21262d);
    border: 1px solid var(--bg-tertiary, #21262d);
    border-radius: 6px;
    color: var(--text-primary, #e6edf3);
    font-size: 0.9rem;
  }

  .search-input:focus {
    border-color: var(--accent-blue, #58a6ff);
    outline: none;
  }

  /* Modal */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal-content {
    background: var(--bg-secondary, #161b22);
    border: 1px solid var(--bg-tertiary, #21262d);
    border-radius: 12px;
    padding: 24px;
    min-width: 400px;
    max-width: 500px;
  }

  .modal-content h3 {
    margin: 0 0 12px;
    color: var(--text-primary, #e6edf3);
  }

  .modal-content p {
    color: var(--text-secondary, #8b949e);
    margin-bottom: 20px;
  }

  .modal-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }
</style>
