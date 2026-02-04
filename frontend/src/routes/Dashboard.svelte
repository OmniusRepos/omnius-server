<script lang="ts">
  import { onMount } from 'svelte';
  import StatCard from '../lib/components/StatCard.svelte';
  import { getStats } from '../lib/api/client';

  let loading = true;
  let stats = {
    movies: 0,
    series: 0,
    channels: 0,
    activeStreams: 0,
  };

  onMount(async () => {
    try {
      const data = await getStats();
      stats = {
        movies: data.movies || 0,
        series: data.series || 0,
        channels: 0,
        activeStreams: 0,
      };
    } catch (e) {
      console.error('Failed to load stats:', e);
    }
    loading = false;
  });
</script>

<div class="dashboard">
  <h1 class="page-title">Dashboard</h1>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else}
    <section class="stats-section">
      <div class="section-header">
        <h2 class="section-title">
          <span>ONLINE</span><span>STATS</span>
        </h2>
      </div>
      <div class="stats-grid">
        <StatCard label="ACTIVE" value={stats.activeStreams} suffix="STREAMS" />
        <StatCard label="SERVER" value="Online" suffix="" />
      </div>
    </section>

    <section class="stats-section">
      <div class="section-header">
        <h2 class="section-title">
          <span>DATABASE</span><span>STATS</span>
        </h2>
      </div>
      <div class="stats-grid">
        <StatCard label="MOVIES" value={stats.movies} />
        <StatCard label="TV" value={stats.series} suffix="SHOWS" />
        <StatCard label="LIVE" value={stats.channels} suffix="CHANNELS" />
      </div>
    </section>

    <section class="quick-actions">
      <h2 class="section-title">
        <span>QUICK</span><span>ACTIONS</span>
      </h2>
      <div class="actions-grid">
        <a href="#/movies" class="action-card">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/>
            <line x1="7" y1="2" x2="7" y2="22"/>
            <line x1="17" y1="2" x2="17" y2="22"/>
            <line x1="2" y1="12" x2="22" y2="12"/>
          </svg>
          <span>Manage Movies</span>
        </a>
        <a href="#/curated" class="action-card">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <line x1="8" y1="6" x2="21" y2="6"/>
            <line x1="8" y1="12" x2="21" y2="12"/>
            <line x1="8" y1="18" x2="21" y2="18"/>
            <line x1="3" y1="6" x2="3.01" y2="6"/>
            <line x1="3" y1="12" x2="3.01" y2="12"/>
            <line x1="3" y1="18" x2="3.01" y2="18"/>
          </svg>
          <span>Curated Lists</span>
        </a>
        <a href="#/settings" class="action-card">
          <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="12" cy="12" r="3"/>
            <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
          </svg>
          <span>Settings</span>
        </a>
      </div>
    </section>
  {/if}
</div>

<style>
  .dashboard {
    max-width: 1200px;
  }

  .page-title {
    font-size: 28px;
    font-weight: 600;
    margin-bottom: 32px;
  }

  .stats-section {
    margin-bottom: 32px;
  }

  .section-title {
    font-size: 14px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 1px;
    margin-bottom: 16px;
  }

  .section-title span:first-child {
    color: var(--text-muted);
  }

  .section-title span:last-child {
    color: var(--text-primary);
  }

  .quick-actions {
    margin-top: 48px;
  }

  .actions-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 16px;
  }

  .action-card {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 20px;
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    color: var(--text-primary);
    text-decoration: none;
    transition: all var(--transition-fast);
  }

  .action-card:hover {
    background: var(--bg-tertiary);
    border-color: var(--accent-red);
    text-decoration: none;
  }

  .action-card svg {
    color: var(--accent-red);
  }
</style>
