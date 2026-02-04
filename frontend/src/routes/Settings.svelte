<script lang="ts">
  import { onMount } from 'svelte';

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

  onMount(async () => {
    await loadYTSSettings();
  });

  async function loadYTSSettings() {
    try {
      const res = await fetch('/admin/api/settings/yts');
      if (res.ok) {
        ytsSettings = await res.json();
        selectedMirror = ytsSettings?.current_mirror || '';
        // Check if current mirror is custom
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
</script>

<div class="settings-page">
  <header class="page-header">
    <h1 class="page-title">SETTINGS</h1>
  </header>

  <div class="card">
    <h3>YTS Mirror Configuration</h3>
    <p class="text-muted mb-4">Select the YTS mirror to use for torrent searches. The server auto-detects working mirrors on startup.</p>

    {#if ytsSettings}
      <div class="current-mirror mb-4">
        <span class="label">Current Mirror:</span>
        <span class="value">{getMirrorName(ytsSettings.current_mirror)}</span>
      </div>

      <div class="form-group">
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
        {saving ? 'Saving...' : 'Save YTS Mirror'}
      </button>
    {:else}
      <p class="text-muted">Loading settings...</p>
    {/if}
  </div>

  <div class="card mt-4">
    <h3>API Configuration</h3>
    <div class="form-group">
      <label class="form-label">OMDB API Key</label>
      <input type="text" class="form-input" placeholder="Enter OMDB API key" />
    </div>
    <div class="form-group">
      <label class="form-label">Admin Password</label>
      <input type="password" class="form-input" placeholder="••••••••" />
    </div>
    <button class="btn btn-primary">Save Settings</button>
  </div>

  <div class="card mt-4">
    <h3>Database</h3>
    <p class="text-muted mb-4">Database path: ./torrents.db</p>
    <div class="flex gap-2">
      <button class="btn btn-secondary">Export Database</button>
      <button class="btn btn-danger">Clear All Data</button>
    </div>
  </div>
</div>

<style>
  h3 {
    margin-bottom: 16px;
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

  .btn-sm {
    padding: 4px 8px;
    font-size: 0.8em;
  }
</style>
