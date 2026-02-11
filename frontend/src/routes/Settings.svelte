<script lang="ts">
  import { onMount } from 'svelte';
  import { getServices, updateServices, getLicenseStatus, type ServiceConfig, type LicenseStatus } from '../lib/api/client';

  // Tab state
  let activeTab = $state('general');

  interface YTSSettings {
    current_mirror: string;
    mirrors: string[];
  }

  let ytsSettings: YTSSettings | null = $state(null);
  let selectedMirror = $state('');
  let customMirror = $state('');
  let useCustomMirror = $state(false);
  let testingMirror = $state(false);
  let testResult = $state<{ status: string; message: string } | null>(null);
  let saving = $state(false);

  // Services state
  let services: ServiceConfig[] = $state([]);
  let savingServices = $state(false);
  let servicesMessage = $state<string | null>(null);

  // License state
  let licenseStatus: LicenseStatus | null = $state(null);

  // Sync state
  let refreshingMovies = $state(false);
  let refreshingShows = $state(false);
  let syncMessage = $state<string | null>(null);

  // Update state
  let updating = $state(false);
  let updateMessage = $state<string | null>(null);
  let updateError = $state(false);

  // Feature check
  let hasLiveChannels = $derived(licenseStatus?.status?.features?.includes('live_channels') ?? false);
  let filteredServices = $derived(
    hasLiveChannels ? services : services.filter((s: ServiceConfig) => s.icon !== 'live')
  );

  onMount(async () => {
    await Promise.all([loadYTSSettings(), loadServices(), loadLicenseStatus()]);
  });

  async function loadLicenseStatus() {
    try {
      licenseStatus = await getLicenseStatus();
    } catch (e) {
      console.error('Failed to load license status:', e);
    }
  }

  async function loadServices() {
    try {
      services = await getServices();
    } catch (e) {
      console.error('Failed to load services:', e);
    }
  }

  async function saveServices() {
    savingServices = true;
    servicesMessage = null;
    try {
      await updateServices(services);
      servicesMessage = 'Services saved! Client apps will update on next load.';
    } catch (e) {
      servicesMessage = 'Error: ' + String(e);
    } finally {
      savingServices = false;
    }
  }

  function toggleService(id: string) {
    services = services.map(s => s.id === id ? { ...s, enabled: !s.enabled } : s);
  }

  function updateLabel(id: string, label: string) {
    services = services.map(s => s.id === id ? { ...s, label } : s);
  }

  async function loadYTSSettings() {
    try {
      const res = await fetch('/admin/api/settings/yts');
      if (res.ok) {
        ytsSettings = await res.json();
        selectedMirror = ytsSettings?.current_mirror || '';
        if (ytsSettings && !ytsSettings.mirrors.includes(ytsSettings.current_mirror)) {
          useCustomMirror = true;
          customMirror = ytsSettings.current_mirror;
        }
      }
    } catch (e) {
      console.error('Failed to load YTS settings:', e);
    }
  }

  async function testMirror(mirror: string) {
    testingMirror = true;
    testResult = null;
    try {
      const res = await fetch('/admin/api/settings/yts/test', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mirror }),
      });
      testResult = await res.json();
    } catch (e) {
      testResult = { status: 'error', message: String(e) };
    } finally {
      testingMirror = false;
    }
  }

  async function saveYTSMirror() {
    const mirrorToSave = useCustomMirror ? customMirror : selectedMirror;
    if (!mirrorToSave) return;

    saving = true;
    try {
      const res = await fetch('/admin/api/settings/yts', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mirror: mirrorToSave }),
      });
      if (res.ok) {
        await loadYTSSettings();
        testResult = { status: 'ok', message: 'Settings saved!' };
      } else {
        const err = await res.text();
        testResult = { status: 'error', message: err };
      }
    } catch (e) {
      testResult = { status: 'error', message: String(e) };
    } finally {
      saving = false;
    }
  }

  function getMirrorName(url: string): string {
    try {
      const hostname = new URL(url).hostname;
      return hostname.toUpperCase();
    } catch {
      return url;
    }
  }

  async function refreshAllMovies() {
    refreshingMovies = true;
    syncMessage = null;
    try {
      const res = await fetch('/admin/api/refresh_all_movies', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (res.ok) {
        syncMessage = 'Movie refresh started! Check server logs for progress.';
      } else {
        const data = await res.json();
        syncMessage = 'Error: ' + (data.status_message || 'Failed to start refresh');
      }
    } catch (e) {
      syncMessage = 'Error: ' + String(e);
    } finally {
      refreshingMovies = false;
    }
  }

  async function refreshAllShows() {
    refreshingShows = true;
    syncMessage = null;
    try {
      const res = await fetch('/admin/api/refresh_all_series', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (res.ok) {
        syncMessage = 'TV Shows refresh started! Check server logs for progress.';
      } else {
        const data = await res.json();
        syncMessage = 'Error: ' + (data.status_message || 'Failed to start refresh');
      }
    } catch (e) {
      syncMessage = 'Error: ' + String(e);
    } finally {
      refreshingShows = false;
    }
  }

  async function triggerUpdate() {
    updating = true;
    updateMessage = null;
    updateError = false;
    try {
      const res = await fetch('/admin/api/update', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      const data = await res.json();
      if (res.ok) {
        updateMessage = 'Update downloaded. Server is restarting...';
        // Poll until server responds (any status means it's back)
        setTimeout(async () => {
          for (let i = 0; i < 30; i++) {
            try {
              const check = await fetch('/admin/api/stats', { method: 'GET' });
              if (check.status > 0) {
                updateMessage = 'Update complete! Reloading...';
                updating = false;
                setTimeout(() => window.location.reload(), 1000);
                return;
              }
            } catch {}
            await new Promise(r => setTimeout(r, 2000));
          }
          updateMessage = 'Server is restarting. Please refresh the page.';
          updating = false;
        }, 3000);
      } else {
        updateError = true;
        updateMessage = data.error || 'Update failed';
        updating = false;
      }
    } catch (e) {
      updateError = true;
      updateMessage = 'Error: ' + String(e);
      updating = false;
    }
  }
</script>

<div class="settings-page">
  <header class="page-header">
    <h1 class="page-title">SETTINGS</h1>
  </header>

  <!-- Tabs -->
  <div class="tabs">
    <button class="tab" class:active={activeTab === 'services'} onclick={() => activeTab = 'services'}>
      Services
    </button>
    {#if licenseStatus?.status?.valid}
      <button class="tab" class:active={activeTab === 'general'} onclick={() => activeTab = 'general'}>
        General
      </button>
    {/if}
    <button class="tab" class:active={activeTab === 'sync'} onclick={() => activeTab = 'sync'}>
      Data Sync
    </button>
    <button class="tab" class:active={activeTab === 'api'} onclick={() => activeTab = 'api'}>
      API Keys
    </button>
    <button class="tab" class:active={activeTab === 'database'} onclick={() => activeTab = 'database'}>
      Database
    </button>
    <button class="tab" class:active={activeTab === 'license'} onclick={() => activeTab = 'license'}>
      License
    </button>
    <button class="tab" class:active={activeTab === 'update'} onclick={() => activeTab = 'update'}>
      Update
    </button>
  </div>

  <!-- Services Tab -->
  {#if activeTab === 'services'}
    <div class="card">
      <h3>Client Services</h3>
      <p class="text-muted mb-4">
        Choose which services your server provides. Client apps will show/hide sidebar sections based on this configuration.
        The client fetches <code>/api/v2/config.json</code> on startup.
      </p>

      <div class="services-list">
        {#each filteredServices as service}
          <div class="service-item" class:disabled={!service.enabled}>
            <div class="service-toggle">
              <button
                class="toggle-btn"
                class:active={service.enabled}
                onclick={() => toggleService(service.id)}
              >
                <span class="toggle-knob"></span>
              </button>
            </div>
            <div class="service-icon">{service.icon === 'movie' ? '🎬' : service.icon === 'tv' ? '📺' : service.icon === 'live' ? '📡' : '📦'}</div>
            <div class="service-details">
              <input
                type="text"
                class="service-label-input"
                value={service.label}
                oninput={(e) => updateLabel(service.id, (e.target as HTMLInputElement).value)}
              />
              <span class="service-id text-muted">{service.id}</span>
            </div>
          </div>
        {/each}
      </div>

      {#if servicesMessage}
        <div class="sync-message" class:error={servicesMessage.startsWith('Error')}>
          {servicesMessage}
        </div>
      {/if}

      <button
        class="btn btn-primary mt-4"
        onclick={saveServices}
        disabled={savingServices}
      >
        {savingServices ? 'Saving...' : 'Save Services'}
      </button>
    </div>

    <div class="card mt-4">
      <h3>API Endpoint</h3>
      <p class="text-muted mb-4">Client apps connect to this server using the config endpoint:</p>
      <div class="api-endpoint">
        <code>{window.location.origin}/api/v2/config.json</code>
      </div>
    </div>
  {/if}

  <!-- General Tab (hidden in demo/unlicensed — internal torrent config) -->
  {#if activeTab === 'general' && licenseStatus?.status?.valid}
    <div class="card">
      <h3>YTS Mirror Configuration</h3>
      <p class="text-muted mb-4">Select the YTS mirror to use for torrent searches.</p>

      {#if ytsSettings}
        <div class="current-mirror mb-4">
          <span class="label">Current Mirror:</span>
          <span class="value">{getMirrorName(ytsSettings.current_mirror)}</span>
        </div>

        <div class="form-group">
          <!-- svelte-ignore a11y_label_has_associated_control -->
          <label class="form-label">Select Mirror</label>
          <div class="mirror-options">
            {#each ytsSettings.mirrors as mirror}
              <label class="mirror-option" class:selected={!useCustomMirror && selectedMirror === mirror}>
                <input
                  type="radio"
                  name="mirror"
                  value={mirror}
                  checked={!useCustomMirror && selectedMirror === mirror}
                  onchange={() => { selectedMirror = mirror; useCustomMirror = false; testResult = null; }}
                />
                <span class="mirror-name">{getMirrorName(mirror)}</span>
                <span class="mirror-url">{mirror}</span>
                <button
                  type="button"
                  class="btn btn-sm btn-secondary test-btn"
                  onclick={() => testMirror(mirror)}
                  disabled={testingMirror}
                >
                  Test
                </button>
              </label>
            {/each}

            <label class="mirror-option" class:selected={useCustomMirror}>
              <input
                type="radio"
                name="mirror"
                checked={useCustomMirror}
                onchange={() => { useCustomMirror = true; testResult = null; }}
              />
              <span class="mirror-name">CUSTOM</span>
              <input
                type="text"
                class="form-input custom-input"
                placeholder="https://yts.example.com/api/v2"
                bind:value={customMirror}
                onfocus={() => { useCustomMirror = true; }}
              />
              {#if useCustomMirror && customMirror}
                <button
                  type="button"
                  class="btn btn-sm btn-secondary test-btn"
                  onclick={() => testMirror(customMirror)}
                  disabled={testingMirror}
                >
                  Test
                </button>
              {/if}
            </label>
          </div>
        </div>

        {#if testResult}
          <div class="test-result {testResult.status}">
            {#if testResult.status === 'ok'}
              ✓ {testResult.message}
            {:else}
              ✗ {testResult.message}
            {/if}
          </div>
        {/if}

        <button
          class="btn btn-primary mt-4"
          onclick={saveYTSMirror}
          disabled={saving || (!useCustomMirror && !selectedMirror) || (useCustomMirror && !customMirror)}
        >
          {saving ? 'Saving...' : 'Save Mirror'}
        </button>
      {:else}
        <p class="text-muted">Loading settings...</p>
      {/if}
    </div>
  {/if}

  <!-- Data Sync Tab -->
  {#if activeTab === 'sync'}
    <div class="card">
      <h3>Refresh All Content</h3>
      <p class="text-muted mb-4">
        Fetch latest metadata from IMDB/OMDB for all content. This includes ratings, cast, directors, images, and box office data.
        Runs in background with rate limiting.
      </p>

      <div class="sync-buttons">
        <div class="sync-item">
          <div class="sync-info">
            <span class="sync-title">Movies</span>
            <span class="sync-desc">Refresh all movie metadata from IMDB/OMDB</span>
          </div>
          <button
            class="btn btn-primary"
            onclick={refreshAllMovies}
            disabled={refreshingMovies}
          >
            {refreshingMovies ? 'Starting...' : 'Refresh All Movies'}
          </button>
        </div>

        <div class="sync-item">
          <div class="sync-info">
            <span class="sync-title">TV Shows</span>
            <span class="sync-desc">Refresh all TV show metadata from IMDB + torrents from EZTV</span>
          </div>
          <button
            class="btn btn-primary"
            onclick={refreshAllShows}
            disabled={refreshingShows}
          >
            {refreshingShows ? 'Starting...' : 'Refresh All TV Shows'}
          </button>
        </div>
      </div>

      {#if syncMessage}
        <div class="sync-message" class:error={syncMessage.startsWith('Error')}>
          {syncMessage}
        </div>
      {/if}
    </div>

    {#if licenseStatus?.status?.valid}
      <div class="card mt-4">
        <h3>Torrent Sync</h3>
        <p class="text-muted mb-4">Search and add torrents from YTS, EZTV, and 1337x for all content.</p>

        <div class="sync-buttons">
          <div class="sync-item">
            <div class="sync-info">
              <span class="sync-title">Sync Torrents</span>
              <span class="sync-desc">Find new torrents for all movies</span>
            </div>
            <button class="btn btn-secondary" disabled>Coming Soon</button>
          </div>
        </div>
      </div>
    {/if}
  {/if}

  <!-- API Keys Tab -->
  {#if activeTab === 'api'}
    <div class="card">
      <h3>IMDB API</h3>
      <p class="text-muted mb-4">
        Using <a href="https://imdbapi.dev" target="_blank" class="link">imdbapi.dev</a> - Free, no API key required.
      </p>
      <div class="api-status success">
        <span class="status-dot"></span>
        Active - No configuration needed
      </div>
    </div>

    <div class="card mt-4">
      <h3>OMDB API (Fallback)</h3>
      <p class="text-muted mb-4">
        Used as fallback for Rotten Tomatoes scores. Get a free key at <a href="https://www.omdbapi.com/apikey.aspx" target="_blank" class="link">omdbapi.com</a>
      </p>
      <div class="form-group">
        <label class="form-label" for="omdb_api_key">API Key</label>
        <input type="text" id="omdb_api_key" class="form-input" placeholder="Enter OMDB API key" />
      </div>
      <p class="text-muted text-sm">Set via environment variable: OMDB_API_KEY</p>
    </div>
  {/if}

  <!-- License Tab -->
  {#if activeTab === 'license'}
    <div class="card">
      <h3>License Status</h3>
      {#if licenseStatus}
        <div class="license-info">
          <div class="db-row">
            <span class="db-label">Server Mode</span>
            <span class="db-value">{licenseStatus.server_mode ? 'License Authority' : 'Client'}</span>
          </div>
          {#if licenseStatus.status}
            <div class="db-row">
              <span class="db-label">Mode</span>
              <span class="db-value" style="color: {licenseStatus.status.mode === 'licensed' ? '#22c55e' : licenseStatus.status.mode === 'demo' ? '#f59e0b' : licenseStatus.status.mode === 'grace' ? '#f59e0b' : licenseStatus.status.mode === 'authority' ? '#3b82f6' : '#ef4444'}">
                {licenseStatus.status.mode?.toUpperCase()}
              </span>
            </div>
            <div class="db-row">
              <span class="db-label">Message</span>
              <span class="db-value">{licenseStatus.status.message}</span>
            </div>
            {#if licenseStatus.status.plan}
              <div class="db-row">
                <span class="db-label">Plan</span>
                <span class="db-value">{licenseStatus.status.plan}</span>
              </div>
            {/if}
            {#if licenseStatus.status.license_key}
              <div class="db-row">
                <span class="db-label">License Key</span>
                <span class="db-value"><code>{licenseStatus.status.license_key}</code></span>
              </div>
            {/if}
            {#if licenseStatus.status.grace_end}
              <div class="db-row">
                <span class="db-label">Grace Until</span>
                <span class="db-value" style="color: #f59e0b">{new Date(licenseStatus.status.grace_end).toLocaleDateString()}</span>
              </div>
            {/if}
          {/if}
        </div>
      {:else}
        <p class="text-muted">Loading license status...</p>
      {/if}
    </div>

    <div class="card mt-4">
      <h3>Configuration</h3>
      <p class="text-muted mb-4">License settings are configured via environment variables:</p>
      <div class="db-info">
        <div class="db-row">
          <span class="db-label">LICENSE_KEY</span>
          <span class="db-value">Your license key (empty = demo mode)</span>
        </div>
      </div>
    </div>
  {/if}

  <!-- Database Tab -->
  {#if activeTab === 'database'}
    <div class="card">
      <h3>Database Info</h3>
      <div class="db-info">
        <div class="db-row">
          <span class="db-label">Location</span>
          <span class="db-value">./data/torrents.db</span>
        </div>
        <div class="db-row">
          <span class="db-label">Type</span>
          <span class="db-value">SQLite</span>
        </div>
      </div>
    </div>

    <div class="card mt-4">
      <h3>Maintenance</h3>
      <div class="sync-buttons">
        <div class="sync-item">
          <div class="sync-info">
            <span class="sync-title">Export Database</span>
            <span class="sync-desc">Download a backup of your database</span>
          </div>
          <button class="btn btn-secondary" disabled>Export</button>
        </div>
        <div class="sync-item danger">
          <div class="sync-info">
            <span class="sync-title">Clear All Data</span>
            <span class="sync-desc">Delete all movies, shows, and torrents</span>
          </div>
          <button class="btn btn-danger" disabled>Clear Data</button>
        </div>
      </div>
    </div>
  {/if}

  {#if activeTab === 'update'}
    <div class="card">
      <h3>Server Update</h3>
      <p class="text-muted mb-4">
        Download and install the latest Omnius server binary from GitHub Releases. The server will restart automatically after updating.
      </p>
      <div class="sync-buttons">
        <div class="sync-item">
          <div class="sync-info">
            <span class="sync-title">Update Server</span>
            <span class="sync-desc">Download the latest release and restart</span>
          </div>
          <button class="btn btn-primary" onclick={triggerUpdate} disabled={updating}>
            {updating ? 'Updating...' : 'Check for Updates'}
          </button>
        </div>
      </div>
      {#if updateMessage}
        <div class="sync-message" class:error={updateError}>
          {updateMessage}
        </div>
      {/if}
    </div>
  {/if}
</div>

<style>
  /* Tabs */
  .tabs {
    display: flex;
    gap: 4px;
    margin-bottom: 24px;
    border-bottom: 1px solid var(--border-color, #333);
    padding-bottom: 0;
  }

  .tab {
    padding: 12px 20px;
    background: none;
    border: none;
    color: var(--text-muted, #888);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
    transition: all 0.2s;
  }

  .tab:hover {
    color: var(--text-primary, #fff);
  }

  .tab.active {
    color: var(--accent, #6366f1);
    border-bottom-color: var(--accent, #6366f1);
  }

  h3 {
    margin-bottom: 16px;
  }

  .link {
    color: var(--accent-blue, #3b82f6);
  }

  .current-mirror {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px;
    background: var(--bg-tertiary, #1a1a2e);
    border-radius: 8px;
  }

  .current-mirror .label {
    color: var(--text-muted, #888);
  }

  .current-mirror .value {
    color: var(--accent, #6366f1);
    font-weight: 600;
  }

  .mirror-options {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .mirror-option {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: var(--bg-tertiary, #1a1a2e);
    border-radius: 8px;
    cursor: pointer;
    border: 2px solid transparent;
    transition: border-color 0.2s;
  }

  .mirror-option:hover {
    border-color: var(--border-color, #333);
  }

  .mirror-option.selected {
    border-color: var(--accent, #6366f1);
  }

  .mirror-option input[type="radio"] {
    accent-color: var(--accent, #6366f1);
  }

  .mirror-name {
    font-weight: 600;
    min-width: 120px;
  }

  .mirror-url {
    color: var(--text-muted, #888);
    font-size: 0.85em;
    flex: 1;
  }

  .custom-input {
    flex: 1;
    margin: 0;
  }

  .test-btn {
    padding: 4px 12px;
    font-size: 0.85em;
  }

  .test-result {
    margin-top: 12px;
    padding: 12px;
    border-radius: 8px;
  }

  .test-result.ok {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
    border: 1px solid #22c55e;
  }

  .test-result.error {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid #ef4444;
  }

  /* Sync section */
  .sync-buttons {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .sync-item {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
    background: var(--bg-tertiary, #1a1a2e);
    border-radius: 8px;
  }

  .sync-item.danger {
    border: 1px solid rgba(239, 68, 68, 0.3);
  }

  .sync-info {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .sync-title {
    font-weight: 600;
    font-size: 1rem;
  }

  .sync-desc {
    color: var(--text-muted, #888);
    font-size: 0.85rem;
  }

  .sync-message {
    margin-top: 16px;
    padding: 12px;
    border-radius: 8px;
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
    border: 1px solid #22c55e;
  }

  .sync-message.error {
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid #ef4444;
  }

  /* API Status */
  .api-status {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 12px;
    border-radius: 8px;
    font-weight: 500;
  }

  .api-status.success {
    background: rgba(34, 197, 94, 0.1);
    color: #22c55e;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    border-radius: 50%;
    background: currentColor;
  }

  .text-sm {
    font-size: 0.85rem;
  }

  /* Database */
  .db-info {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .db-row {
    display: flex;
    padding: 12px;
    background: var(--bg-tertiary, #1a1a2e);
    border-radius: 8px;
  }

  .db-label {
    min-width: 100px;
    color: var(--text-muted, #888);
  }

  .db-value {
    font-family: monospace;
  }

  /* Services */
  .services-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .service-item {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;
    background: var(--bg-tertiary, #1a1a2e);
    border-radius: 8px;
    transition: opacity 0.2s;
  }

  .service-item.disabled {
    opacity: 0.5;
  }

  .service-icon {
    font-size: 1.5rem;
    width: 36px;
    text-align: center;
  }

  .service-details {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .service-label-input {
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-primary, #fff);
    font-size: 1rem;
    font-weight: 600;
    padding: 4px 8px;
    border-radius: 4px;
  }

  .service-label-input:focus {
    border-color: var(--accent, #6366f1);
    outline: none;
    background: var(--bg-secondary, #0d1117);
  }

  .service-id {
    font-size: 0.75rem;
    padding-left: 8px;
    font-family: monospace;
  }

  .toggle-btn {
    width: 44px;
    height: 24px;
    border-radius: 12px;
    border: none;
    background: var(--bg-secondary, #0d1117);
    cursor: pointer;
    position: relative;
    transition: background 0.2s;
    padding: 0;
  }

  .toggle-btn.active {
    background: var(--accent-green, #22c55e);
  }

  .toggle-knob {
    position: absolute;
    top: 3px;
    left: 3px;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: white;
    transition: transform 0.2s;
  }

  .toggle-btn.active .toggle-knob {
    transform: translateX(20px);
  }

  .api-endpoint {
    padding: 12px;
    background: var(--bg-tertiary, #1a1a2e);
    border-radius: 8px;
    font-family: monospace;
    font-size: 0.9rem;
    color: var(--accent-blue, #3b82f6);
  }
</style>
