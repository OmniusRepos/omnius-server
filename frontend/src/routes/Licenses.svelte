<script lang="ts">
  import { onMount } from 'svelte';
  import {
    getLicenses, createLicense, updateLicense, deleteLicense,
    getLicenseDeployments, deactivateDeployment,
    type License, type LicenseDeployment,
  } from '../lib/api/client';

  let licenses: License[] = $state([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Create form
  let showCreate = $state(false);
  let createForm = $state({ plan: 'personal', owner_email: '', owner_name: '', max_deployments: 1, notes: '' });
  let creating = $state(false);

  // Detail view
  let selectedLicense: License | null = $state(null);
  let deployments: LicenseDeployment[] = $state([]);
  let loadingDeps = $state(false);

  // Copied key feedback
  let copiedId: number | null = $state(null);

  onMount(() => { loadLicenses(); });

  async function loadLicenses() {
    loading = true;
    error = null;
    try {
      licenses = await getLicenses();
    } catch (e) {
      error = String(e);
    } finally {
      loading = false;
    }
  }

  async function handleCreate() {
    creating = true;
    try {
      const maxDep = createForm.plan === 'personal' ? 1 : createForm.plan === 'business' ? 5 : createForm.max_deployments;
      await createLicense({ ...createForm, max_deployments: maxDep });
      showCreate = false;
      createForm = { plan: 'personal', owner_email: '', owner_name: '', max_deployments: 1, notes: '' };
      await loadLicenses();
    } catch (e) {
      error = String(e);
    } finally {
      creating = false;
    }
  }

  async function toggleActive(l: License) {
    try {
      await updateLicense(l.id, { is_active: !l.is_active });
      await loadLicenses();
    } catch (e) {
      error = String(e);
    }
  }

  async function handleDelete(l: License) {
    if (!confirm(`Delete license for ${l.owner_name || l.owner_email}? This cannot be undone.`)) return;
    try {
      await deleteLicense(l.id);
      if (selectedLicense?.id === l.id) selectedLicense = null;
      await loadLicenses();
    } catch (e) {
      error = String(e);
    }
  }

  async function viewDeployments(l: License) {
    selectedLicense = l;
    loadingDeps = true;
    try {
      deployments = await getLicenseDeployments(l.id);
    } catch (e) {
      error = String(e);
    } finally {
      loadingDeps = false;
    }
  }

  async function handleDeactivateDep(dep: LicenseDeployment) {
    if (!selectedLicense) return;
    try {
      await deactivateDeployment(selectedLicense.id, dep.id);
      deployments = await getLicenseDeployments(selectedLicense.id);
      await loadLicenses();
    } catch (e) {
      error = String(e);
    }
  }

  function copyKey(key: string, id: number) {
    navigator.clipboard.writeText(key);
    copiedId = id;
    setTimeout(() => { copiedId = null; }, 2000);
  }

  function planColor(plan: string): string {
    switch (plan) {
      case 'personal': return '#3b82f6';
      case 'business': return '#8b5cf6';
      case 'enterprise': return '#f59e0b';
      default: return '#6b7280';
    }
  }

  function timeAgo(dateStr: string): string {
    const d = new Date(dateStr);
    const now = new Date();
    const diff = now.getTime() - d.getTime();
    const mins = Math.floor(diff / 60000);
    if (mins < 60) return `${mins}m ago`;
    const hrs = Math.floor(mins / 60);
    if (hrs < 24) return `${hrs}h ago`;
    const days = Math.floor(hrs / 24);
    return `${days}d ago`;
  }
</script>

<div class="licenses-page">
  <header class="page-header">
    <h1 class="page-title">LICENSES</h1>
    <button class="btn btn-primary" onclick={() => showCreate = !showCreate}>
      {showCreate ? 'Cancel' : '+ New License'}
    </button>
  </header>

  {#if error}
    <div class="error-banner">{error}</div>
  {/if}

  <!-- Create Form -->
  {#if showCreate}
    <div class="card mb-4">
      <h3>Create License</h3>
      <div class="form-grid">
        <div class="form-group">
          <label class="form-label" for="plan">Plan</label>
          <select id="plan" class="form-input" bind:value={createForm.plan}>
            <option value="personal">Personal (1 deployment)</option>
            <option value="business">Business (5 deployments)</option>
            <option value="enterprise">Enterprise (custom)</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label" for="owner_name">Owner Name</label>
          <input id="owner_name" class="form-input" bind:value={createForm.owner_name} placeholder="John Doe" />
        </div>
        <div class="form-group">
          <label class="form-label" for="owner_email">Owner Email</label>
          <input id="owner_email" type="email" class="form-input" bind:value={createForm.owner_email} placeholder="john@example.com" />
        </div>
        {#if createForm.plan === 'enterprise'}
          <div class="form-group">
            <label class="form-label" for="max_dep">Max Deployments</label>
            <input id="max_dep" type="number" class="form-input" bind:value={createForm.max_deployments} min="1" />
          </div>
        {/if}
        <div class="form-group full-width">
          <label class="form-label" for="notes">Notes</label>
          <input id="notes" class="form-input" bind:value={createForm.notes} placeholder="Optional notes..." />
        </div>
      </div>
      <button class="btn btn-primary mt-4" onclick={handleCreate} disabled={creating || !createForm.owner_email}>
        {creating ? 'Creating...' : 'Generate License Key'}
      </button>
    </div>
  {/if}

  <!-- License List -->
  {#if loading}
    <div class="loading-state">Loading licenses...</div>
  {:else if licenses.length === 0}
    <div class="empty-state">
      <p>No licenses yet. Create one to get started.</p>
    </div>
  {:else}
    <div class="licenses-grid">
      {#each licenses as l}
        <div class="license-card" class:revoked={!l.is_active}>
          <div class="license-header">
            <span class="plan-badge" style="background: {planColor(l.plan)}">{l.plan.toUpperCase()}</span>
            <div class="license-actions">
              <button class="btn-icon" title={l.is_active ? 'Revoke' : 'Reactivate'} onclick={() => toggleActive(l)}>
                {l.is_active ? '🔒' : '🔓'}
              </button>
              <button class="btn-icon danger" title="Delete" onclick={() => handleDelete(l)}>
                🗑
              </button>
            </div>
          </div>

          <div class="license-key-row">
            <code class="license-key">{l.license_key}</code>
            <button class="btn-copy" onclick={() => copyKey(l.license_key, l.id)}>
              {copiedId === l.id ? 'Copied!' : 'Copy'}
            </button>
          </div>

          <div class="license-details">
            <div class="detail-row">
              <span class="detail-label">Owner</span>
              <span class="detail-value">{l.owner_name || 'N/A'}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Email</span>
              <span class="detail-value">{l.owner_email || 'N/A'}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Deployments</span>
              <span class="detail-value">{l.active_deployments || 0} / {l.max_deployments}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Status</span>
              <span class="detail-value" class:text-green={l.is_active} class:text-red={!l.is_active}>
                {l.is_active ? 'Active' : 'Revoked'}
              </span>
            </div>
            {#if l.expires_at}
              <div class="detail-row">
                <span class="detail-label">Expires</span>
                <span class="detail-value">{new Date(l.expires_at).toLocaleDateString()}</span>
              </div>
            {/if}
          </div>

          <button class="btn btn-secondary btn-sm mt-3" onclick={() => viewDeployments(l)}>
            View Deployments
          </button>
        </div>
      {/each}
    </div>
  {/if}

  <!-- Deployments Panel -->
  {#if selectedLicense}
    <div class="deployments-panel">
      <div class="panel-header">
        <h3>Deployments for {selectedLicense.license_key}</h3>
        <button class="btn-close" onclick={() => selectedLicense = null}>Close</button>
      </div>

      {#if loadingDeps}
        <p class="text-muted">Loading deployments...</p>
      {:else if deployments.length === 0}
        <p class="text-muted">No deployments registered.</p>
      {:else}
        <table class="dep-table">
          <thead>
            <tr>
              <th>Machine</th>
              <th>IP</th>
              <th>Version</th>
              <th>Last Heartbeat</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each deployments as dep}
              <tr class:inactive={!dep.is_active}>
                <td>
                  <div class="machine-info">
                    <span class="machine-label">{dep.machine_label || 'Unknown'}</span>
                    <span class="fingerprint">{dep.machine_fingerprint.slice(0, 12)}...</span>
                  </div>
                </td>
                <td><code>{dep.ip_address}</code></td>
                <td>{dep.server_version}</td>
                <td>{timeAgo(dep.last_heartbeat)}</td>
                <td>
                  <span class="status-dot" class:active={dep.is_active} class:stale={!dep.is_active}></span>
                  {dep.is_active ? 'Active' : 'Inactive'}
                </td>
                <td>
                  {#if dep.is_active}
                    <button class="btn btn-danger btn-sm" onclick={() => handleDeactivateDep(dep)}>
                      Deactivate
                    </button>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      {/if}
    </div>
  {/if}
</div>

<style>
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
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

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .full-width {
    grid-column: 1 / -1;
  }

  .licenses-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(380px, 1fr));
    gap: 16px;
  }

  .license-card {
    background: var(--bg-secondary, #0d1117);
    border: 1px solid var(--border-color, #333);
    border-radius: 12px;
    padding: 20px;
    transition: opacity 0.2s;
  }

  .license-card.revoked {
    opacity: 0.6;
  }

  .license-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 12px;
  }

  .plan-badge {
    padding: 4px 10px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 700;
    color: white;
    letter-spacing: 0.5px;
  }

  .license-actions {
    display: flex;
    gap: 4px;
  }

  .btn-icon {
    background: none;
    border: none;
    cursor: pointer;
    font-size: 16px;
    padding: 4px 8px;
    border-radius: 4px;
    opacity: 0.6;
  }

  .btn-icon:hover {
    opacity: 1;
    background: var(--bg-tertiary, #1a1a2e);
  }

  .btn-icon.danger:hover {
    background: rgba(239, 68, 68, 0.2);
  }

  .license-key-row {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 16px;
    padding: 10px;
    background: var(--bg-tertiary, #1a1a2e);
    border-radius: 6px;
  }

  .license-key {
    flex: 1;
    font-size: 13px;
    color: var(--accent, #6366f1);
    font-weight: 600;
    letter-spacing: 0.5px;
  }

  .btn-copy {
    background: var(--bg-secondary, #0d1117);
    border: 1px solid var(--border-color, #333);
    color: var(--text-secondary, #999);
    padding: 4px 10px;
    border-radius: 4px;
    cursor: pointer;
    font-size: 12px;
  }

  .btn-copy:hover {
    color: var(--text-primary, #fff);
    border-color: var(--accent, #6366f1);
  }

  .license-details {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .detail-row {
    display: flex;
    justify-content: space-between;
    font-size: 13px;
  }

  .detail-label {
    color: var(--text-muted, #888);
  }

  .detail-value {
    font-weight: 500;
  }

  .text-green { color: #22c55e; }
  .text-red { color: #ef4444; }

  /* Deployments panel */
  .deployments-panel {
    margin-top: 24px;
    background: var(--bg-secondary, #0d1117);
    border: 1px solid var(--border-color, #333);
    border-radius: 12px;
    padding: 20px;
  }

  .panel-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 16px;
  }

  .btn-close {
    background: none;
    border: 1px solid var(--border-color, #333);
    color: var(--text-secondary, #999);
    padding: 6px 14px;
    border-radius: 6px;
    cursor: pointer;
  }

  .dep-table {
    width: 100%;
    border-collapse: collapse;
  }

  .dep-table th {
    text-align: left;
    padding: 10px 12px;
    color: var(--text-muted, #888);
    font-size: 12px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    border-bottom: 1px solid var(--border-color, #333);
  }

  .dep-table td {
    padding: 10px 12px;
    font-size: 13px;
    border-bottom: 1px solid rgba(255,255,255,0.05);
  }

  .dep-table tr.inactive {
    opacity: 0.5;
  }

  .machine-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .machine-label {
    font-weight: 600;
  }

  .fingerprint {
    font-size: 11px;
    color: var(--text-muted, #888);
    font-family: monospace;
  }

  .status-dot {
    display: inline-block;
    width: 8px;
    height: 8px;
    border-radius: 50%;
    margin-right: 6px;
  }

  .status-dot.active { background: #22c55e; }
  .status-dot.stale { background: #6b7280; }

  .loading-state, .empty-state {
    text-align: center;
    padding: 40px;
    color: var(--text-muted, #888);
  }

  .mt-3 { margin-top: 12px; }
  .mb-4 { margin-bottom: 16px; }
  .mt-4 { margin-top: 16px; }

  .btn-sm {
    padding: 6px 14px;
    font-size: 13px;
  }
</style>
