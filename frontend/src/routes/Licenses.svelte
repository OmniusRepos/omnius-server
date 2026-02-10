<script lang="ts">
  import { onMount } from 'svelte';

  interface LicenseStatus {
    mode: string;
    plan: string;
    message: string;
    grace_end?: string;
    valid: boolean;
    demo_mode: boolean;
    license_key: string;
  }

  interface LicenseInfo {
    status: LicenseStatus;
    fingerprint: string;
    hostname: string;
    domain: string;
    server_url: string;
  }

  const API_BASE = '/admin/api';

  let info: LicenseInfo | null = $state(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  let keyInput = $state('');
  let activating = $state(false);

  onMount(() => { loadStatus(); });

  async function loadStatus() {
    loading = true;
    error = null;
    try {
      const res = await fetch(`${API_BASE}/license-status`, {
        headers: { 'Accept': 'application/json' },
      });
      info = await res.json();
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  async function handleActivate() {
    if (!keyInput.trim()) return;
    activating = true;
    error = null;
    try {
      const res = await fetch(`${API_BASE}/license-activate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Accept': 'application/json' },
        body: JSON.stringify({ license_key: keyInput.trim() }),
      });
      const data = await res.json();
      if (!res.ok) {
        error = data.error || 'Activation failed';
        if (data.status) {
          info = { ...info!, status: data.status };
        }
      } else {
        info = data;
        keyInput = '';
      }
    } catch (e) {
      error = String(e);
    } finally {
      activating = false;
    }
  }

  function modeColor(mode: string): string {
    switch (mode) {
      case 'licensed': return '#22c55e';
      case 'grace': return '#f59e0b';
      case 'demo': return '#3b82f6';
      default: return '#ef4444';
    }
  }

  function planColor(plan: string): string {
    switch (plan) {
      case 'personal': return '#3b82f6';
      case 'business': return '#8b5cf6';
      case 'enterprise': return '#f59e0b';
      default: return '#6b7280';
    }
  }
</script>

<div class="license-page">
  <header class="page-header">
    <h1 class="page-title">LICENSE</h1>
  </header>

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  {#if loading}
    <div class="loading-state">Loading license status...</div>
  {:else if info}
    <!-- Current Status -->
    <div class="status-card">
      <div class="status-header">
        <span class="mode-badge" style="background: {modeColor(info.status.mode)}">
          {info.status.mode.toUpperCase()}
        </span>
        {#if info.status.plan}
          <span class="plan-badge" style="background: {planColor(info.status.plan)}">
            {info.status.plan.toUpperCase()}
          </span>
        {/if}
        <span class="valid-indicator" class:valid={info.status.valid} class:invalid={!info.status.valid}>
          {info.status.valid ? 'Valid' : 'Invalid'}
        </span>
      </div>

      <p class="status-message">{info.status.message}</p>

      {#if info.status.license_key}
        <div class="detail-grid">
          <div class="detail-item">
            <span class="detail-label">License Key</span>
            <code class="detail-value key">{info.status.license_key}</code>
          </div>
          {#if info.status.plan}
            <div class="detail-item">
              <span class="detail-label">Plan</span>
              <span class="detail-value">{info.status.plan}</span>
            </div>
          {/if}
        </div>
      {/if}

      {#if info.status.grace_end}
        <div class="grace-warning">
          Grace period expires: {new Date(info.status.grace_end).toLocaleDateString()}
        </div>
      {/if}
    </div>

    <!-- Server Info -->
    <div class="info-card">
      <h3>Deployment Info</h3>
      <div class="detail-grid">
        <div class="detail-item">
          <span class="detail-label">Domain</span>
          <span class="detail-value">{info.domain || window.location.hostname}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Hostname</span>
          <span class="detail-value">{info.hostname}</span>
        </div>
        <div class="detail-item">
          <span class="detail-label">Machine Fingerprint</span>
          <code class="detail-value fingerprint">{info.fingerprint?.slice(0, 16)}...</code>
        </div>
        <div class="detail-item">
          <span class="detail-label">License Authority</span>
          <span class="detail-value">{info.server_url}</span>
        </div>
      </div>
    </div>

    <!-- Activate License -->
    <div class="activate-card">
      <h3>{info.status.license_key ? 'Change License Key' : 'Activate License'}</h3>
      <p class="activate-hint">
        {info.status.demo_mode
          ? 'Enter a license key to activate this server. Purchase one at omnius.stream.'
          : 'Enter a new license key to replace the current one.'}
      </p>
      <div class="activate-form">
        <input
          type="text"
          class="key-input"
          bind:value={keyInput}
          placeholder="OMNI-XXXX-XXXX-XXXX-XXXX"
          spellcheck="false"
          autocomplete="off"
          onkeydown={(e) => { if (e.key === 'Enter') handleActivate(); }}
        />
        <button
          class="btn btn-primary"
          onclick={handleActivate}
          disabled={activating || !keyInput.trim()}
        >
          {activating ? 'Activating...' : 'Activate'}
        </button>
      </div>
    </div>
  {/if}
</div>

<style>
  .page-header {
    margin-bottom: 24px;
  }

  .error-banner {
    padding: 12px;
    background: rgba(239, 68, 68, 0.1);
    color: #ef4444;
    border: 1px solid #ef4444;
    border-radius: 8px;
    margin-bottom: 16px;
  }

  .loading-state {
    text-align: center;
    padding: 40px;
    color: var(--text-muted, #888);
  }

  .status-card, .info-card, .activate-card {
    background: var(--bg-secondary, #0d1117);
    border: 1px solid var(--border-color, #333);
    border-radius: 12px;
    padding: 24px;
    margin-bottom: 16px;
  }

  .status-header {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }

  .mode-badge, .plan-badge {
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 700;
    color: white;
    letter-spacing: 0.5px;
  }

  .valid-indicator {
    margin-left: auto;
    font-size: 13px;
    font-weight: 600;
  }
  .valid-indicator.valid { color: #22c55e; }
  .valid-indicator.invalid { color: #ef4444; }

  .status-message {
    color: var(--text-secondary, #999);
    font-size: 14px;
    margin-bottom: 16px;
  }

  .detail-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .detail-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .detail-label {
    color: var(--text-muted, #888);
    font-size: 12px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    font-weight: 600;
  }

  .detail-value {
    font-size: 14px;
    font-weight: 500;
  }

  .detail-value.key {
    color: var(--accent, #6366f1);
    font-size: 15px;
    letter-spacing: 0.5px;
  }

  .detail-value.fingerprint {
    font-size: 12px;
    color: var(--text-muted, #888);
  }

  .grace-warning {
    margin-top: 12px;
    padding: 10px;
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
    border: 1px solid rgba(245, 158, 11, 0.3);
    border-radius: 6px;
    font-size: 13px;
  }

  .info-card h3, .activate-card h3 {
    margin-bottom: 16px;
    font-size: 14px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    color: var(--text-secondary, #999);
  }

  .activate-hint {
    color: var(--text-muted, #888);
    font-size: 13px;
    margin-bottom: 16px;
  }

  .activate-form {
    display: flex;
    gap: 12px;
  }

  .key-input {
    flex: 1;
    padding: 12px 16px;
    background: var(--bg-tertiary, #1a1a2e);
    border: 1px solid var(--border-color, #333);
    border-radius: 8px;
    color: var(--text-primary, #fff);
    font-family: monospace;
    font-size: 15px;
    letter-spacing: 1px;
    text-transform: uppercase;
  }

  .key-input::placeholder {
    color: var(--text-muted, #555);
    text-transform: uppercase;
  }

  .key-input:focus {
    outline: none;
    border-color: var(--accent, #6366f1);
  }
</style>
