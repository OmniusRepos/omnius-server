<script lang="ts">
  import { onMount } from 'svelte';

  interface TopItem {
    id?: number;
    name: string;
    count: number;
    change: number;
    image?: string;
  }

  interface QualityItem {
    quality: string;
    percentage: number;
  }

  let loading = $state(true);
  let selectedPeriod = $state('week');

  let streamStats = $state({
    totalStreams: 0,
    activeStreams: 0,
    peakToday: 0,
    peakTime: '',
    avgDuration: '0 min',
    totalChange: 0,
  });

  let bandwidthStats = $state({
    totalToday: '0 B',
    avgPerStream: '0 B',
    peakRate: 'N/A',
    totalMonth: '0 B',
    todayChange: 0,
  });

  let userStats = $state({
    uniqueToday: 0,
    uniqueWeek: 0,
    uniqueMonth: 0,
    returningRate: '0%',
    todayChange: 0,
  });

  let topMovies = $state<TopItem[]>([]);
  let topGenres = $state<TopItem[]>([]);
  let qualityDistribution = $state<QualityItem[]>([]);
  let hourlyActivity = $state<number[]>(new Array(24).fill(0));

  async function loadAnalytics() {
    loading = true;
    try {
      const response = await fetch(`/admin/api/analytics?period=${selectedPeriod}`, {
        credentials: 'include'
      });
      if (!response.ok) throw new Error('Failed to load analytics');

      const data = await response.json();

      streamStats = {
        totalStreams: data.stream_stats?.total_streams || 0,
        activeStreams: data.stream_stats?.active_streams || 0,
        peakToday: data.stream_stats?.peak_today || 0,
        peakTime: data.stream_stats?.peak_time || '',
        avgDuration: data.stream_stats?.avg_duration || '0 min',
        totalChange: data.stream_stats?.total_change || 0,
      };

      bandwidthStats = {
        totalToday: data.bandwidth_stats?.total_today || '0 B',
        avgPerStream: data.bandwidth_stats?.avg_per_stream || '0 B',
        peakRate: data.bandwidth_stats?.peak_rate || 'N/A',
        totalMonth: data.bandwidth_stats?.total_month || '0 B',
        todayChange: data.bandwidth_stats?.today_change || 0,
      };

      userStats = {
        uniqueToday: data.user_stats?.unique_today || 0,
        uniqueWeek: data.user_stats?.unique_week || 0,
        uniqueMonth: data.user_stats?.unique_month || 0,
        returningRate: data.user_stats?.returning_rate || '0%',
        todayChange: data.user_stats?.today_change || 0,
      };

      topMovies = data.top_movies || [];
      topGenres = data.top_genres || [];
      qualityDistribution = data.quality_distribution || [];
      hourlyActivity = data.hourly_activity || new Array(24).fill(0);
    } catch (err) {
      console.error('Failed to load analytics:', err);
    } finally {
      loading = false;
    }
  }

  function handlePeriodChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    selectedPeriod = select.value;
    loadAnalytics();
  }

  onMount(() => {
    loadAnalytics();
    // Refresh every 30 seconds
    const interval = setInterval(loadAnalytics, 30000);
    return () => { clearInterval(interval); };
  });
</script>

<div class="page-header">
  <h1>Analytics</h1>
  <div class="header-actions">
    <select class="time-select" bind:value={selectedPeriod} onchange={handlePeriodChange}>
      <option value="today">Today</option>
      <option value="week">This Week</option>
      <option value="month">This Month</option>
      <option value="year">This Year</option>
    </select>
    <button class="btn btn-secondary">
      <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
        <polyline points="7 10 12 15 17 10"/>
        <line x1="12" y1="15" x2="12" y2="3"/>
      </svg>
      Export
    </button>
  </div>
</div>

{#if loading}
<div class="loading-overlay">
  <div class="spinner"></div>
  <p>Loading analytics...</p>
</div>
{/if}

<div class="analytics-grid">
  <!-- Streaming Stats -->
  <section class="stat-section">
    <h2 class="section-title">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <polygon points="23 7 16 12 23 17 23 7"/>
        <rect x="1" y="5" width="15" height="14" rx="2" ry="2"/>
      </svg>
      Streaming
    </h2>
    <div class="stat-cards">
      <div class="stat-card">
        <span class="stat-label">Total Streams</span>
        <span class="stat-value">{streamStats.totalStreams.toLocaleString()}</span>
        {#if streamStats.totalChange !== 0}
        <span class="stat-change" class:positive={streamStats.totalChange > 0} class:negative={streamStats.totalChange < 0}>
          {streamStats.totalChange > 0 ? '+' : ''}{streamStats.totalChange.toFixed(1)}%
        </span>
        {/if}
      </div>
      <div class="stat-card">
        <span class="stat-label">Active Now</span>
        <span class="stat-value highlight">{streamStats.activeStreams}</span>
        <span class="stat-sublabel">live</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Peak Today</span>
        <span class="stat-value">{streamStats.peakToday}</span>
        {#if streamStats.peakTime}
        <span class="stat-sublabel">at {streamStats.peakTime}</span>
        {/if}
      </div>
      <div class="stat-card">
        <span class="stat-label">Avg Duration</span>
        <span class="stat-value">{streamStats.avgDuration}</span>
      </div>
    </div>
  </section>

  <!-- Bandwidth Stats -->
  <section class="stat-section">
    <h2 class="section-title">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M22 12h-4l-3 9L9 3l-3 9H2"/>
      </svg>
      Bandwidth
    </h2>
    <div class="stat-cards">
      <div class="stat-card">
        <span class="stat-label">Today</span>
        <span class="stat-value">{bandwidthStats.totalToday}</span>
        <span class="stat-change positive">+8.2%</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Avg / Stream</span>
        <span class="stat-value">{bandwidthStats.avgPerStream}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Peak Rate</span>
        <span class="stat-value">{bandwidthStats.peakRate}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">This Month</span>
        <span class="stat-value">{bandwidthStats.totalMonth}</span>
      </div>
    </div>
  </section>

  <!-- Users Stats -->
  <section class="stat-section">
    <h2 class="section-title">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/>
        <circle cx="9" cy="7" r="4"/>
        <path d="M23 21v-2a4 4 0 0 0-3-3.87"/>
        <path d="M16 3.13a4 4 0 0 1 0 7.75"/>
      </svg>
      Users
    </h2>
    <div class="stat-cards">
      <div class="stat-card">
        <span class="stat-label">Today</span>
        <span class="stat-value">{userStats.uniqueToday.toLocaleString()}</span>
        <span class="stat-change positive">+5.3%</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">This Week</span>
        <span class="stat-value">{userStats.uniqueWeek.toLocaleString()}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">This Month</span>
        <span class="stat-value">{userStats.uniqueMonth.toLocaleString()}</span>
      </div>
      <div class="stat-card">
        <span class="stat-label">Returning</span>
        <span class="stat-value">{userStats.returningRate}</span>
      </div>
    </div>
  </section>

  <!-- Hourly Activity Chart -->
  <section class="chart-section wide">
    <h2 class="section-title">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <line x1="18" y1="20" x2="18" y2="10"/>
        <line x1="12" y1="20" x2="12" y2="4"/>
        <line x1="6" y1="20" x2="6" y2="14"/>
      </svg>
      Hourly Activity
    </h2>
    <div class="chart-container">
      <div class="bar-chart">
        {#each hourlyActivity as value, i}
          <div class="bar-wrapper">
            <div
              class="bar"
              style="height: {(value / 200) * 100}%"
              class:peak={value === Math.max(...hourlyActivity)}
            ></div>
            <span class="bar-label">{i}</span>
          </div>
        {/each}
      </div>
    </div>
  </section>

  <!-- Quality Distribution -->
  <section class="chart-section">
    <h2 class="section-title">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="2" y="3" width="20" height="14" rx="2" ry="2"/>
        <line x1="8" y1="21" x2="16" y2="21"/>
        <line x1="12" y1="17" x2="12" y2="21"/>
      </svg>
      Quality Distribution
    </h2>
    <div class="quality-bars">
      {#each qualityDistribution as item}
        <div class="quality-row">
          <span class="quality-label">{item.quality}</span>
          <div class="quality-bar-container">
            <div class="quality-bar" style="width: {item.percentage}%"></div>
          </div>
          <span class="quality-value">{item.percentage}%</span>
        </div>
      {/each}
    </div>
  </section>

  <!-- Top Movies -->
  <section class="list-section">
    <h2 class="section-title">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/>
        <line x1="7" y1="2" x2="7" y2="22"/>
        <line x1="17" y1="2" x2="17" y2="22"/>
        <line x1="2" y1="12" x2="22" y2="12"/>
        <line x1="2" y1="7" x2="7" y2="7"/>
        <line x1="2" y1="17" x2="7" y2="17"/>
        <line x1="17" y1="17" x2="22" y2="17"/>
        <line x1="17" y1="7" x2="22" y2="7"/>
      </svg>
      Top Movies
    </h2>
    <div class="top-list">
      {#each topMovies as item, i}
        <div class="top-item">
          <span class="rank">#{i + 1}</span>
          <span class="name">{item.name}</span>
          <span class="count">{item.count.toLocaleString()}</span>
          <span class="change" class:positive={item.change > 0} class:negative={item.change < 0}>
            {item.change > 0 ? '+' : ''}{item.change}%
          </span>
        </div>
      {/each}
    </div>
  </section>

  <!-- Top Genres -->
  <section class="list-section">
    <h2 class="section-title">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/>
        <path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>
      </svg>
      Top Genres
    </h2>
    <div class="top-list">
      {#each topGenres as item, i}
        <div class="top-item">
          <span class="rank">#{i + 1}</span>
          <span class="name">{item.name}</span>
          <span class="count">{item.count.toLocaleString()}</span>
          <span class="change" class:positive={item.change > 0} class:negative={item.change < 0}>
            {item.change > 0 ? '+' : ''}{item.change}%
          </span>
        </div>
      {/each}
    </div>
  </section>
</div>

<style>
  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 32px;
  }

  .page-header h1 {
    font-size: 24px;
    font-weight: 600;
  }

  .header-actions {
    display: flex;
    gap: 12px;
    align-items: center;
  }

  .time-select {
    padding: 8px 12px;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 14px;
  }

  .analytics-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 24px;
  }

  .stat-section, .chart-section, .list-section {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    padding: 20px;
  }

  .chart-section.wide {
    grid-column: span 3;
  }

  .section-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-secondary);
    margin-bottom: 16px;
    text-transform: uppercase;
    letter-spacing: 0.5px;
  }

  .section-title svg {
    color: var(--accent-red);
  }

  .stat-cards {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;
  }

  .stat-card {
    background: var(--bg-tertiary);
    border-radius: 8px;
    padding: 16px;
    display: flex;
    flex-direction: column;
  }

  .stat-label {
    font-size: 12px;
    color: var(--text-muted);
    margin-bottom: 4px;
  }

  .stat-value {
    font-size: 24px;
    font-weight: 600;
    color: var(--text-primary);
  }

  .stat-value.highlight {
    color: var(--accent-green);
  }

  .stat-change {
    font-size: 12px;
    margin-top: 4px;
  }

  .stat-change.positive {
    color: var(--accent-green);
  }

  .stat-change.negative {
    color: var(--accent-red);
  }

  .stat-sublabel {
    font-size: 12px;
    color: var(--text-muted);
    margin-top: 2px;
  }

  /* Bar Chart */
  .chart-container {
    height: 200px;
    padding-top: 20px;
  }

  .bar-chart {
    display: flex;
    align-items: flex-end;
    justify-content: space-between;
    height: 100%;
    gap: 4px;
  }

  .bar-wrapper {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    height: 100%;
  }

  .bar {
    width: 100%;
    background: var(--accent-red);
    border-radius: 2px 2px 0 0;
    opacity: 0.7;
    transition: opacity 0.2s;
  }

  .bar:hover {
    opacity: 1;
  }

  .bar.peak {
    background: var(--accent-green);
    opacity: 1;
  }

  .bar-label {
    font-size: 10px;
    color: var(--text-muted);
    margin-top: 4px;
  }

  /* Quality Distribution */
  .quality-bars {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .quality-row {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .quality-label {
    width: 50px;
    font-size: 14px;
    font-weight: 500;
  }

  .quality-bar-container {
    flex: 1;
    height: 24px;
    background: var(--bg-tertiary);
    border-radius: 4px;
    overflow: hidden;
  }

  .quality-bar {
    height: 100%;
    background: var(--accent-red);
    border-radius: 4px;
  }

  .quality-value {
    width: 40px;
    text-align: right;
    font-size: 14px;
    color: var(--text-secondary);
  }

  /* Top Lists */
  .top-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .top-item {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px;
    background: var(--bg-tertiary);
    border-radius: 8px;
  }

  .rank {
    width: 30px;
    font-size: 14px;
    font-weight: 600;
    color: var(--text-muted);
  }

  .name {
    flex: 1;
    font-size: 14px;
  }

  .count {
    font-size: 14px;
    color: var(--text-secondary);
  }

  .change {
    width: 50px;
    text-align: right;
    font-size: 12px;
  }

  .positive {
    color: var(--accent-green);
  }

  .negative {
    color: var(--accent-red);
  }

  @media (max-width: 1200px) {
    .analytics-grid {
      grid-template-columns: repeat(2, 1fr);
    }

    .chart-section.wide {
      grid-column: span 2;
    }
  }

  @media (max-width: 768px) {
    .analytics-grid {
      grid-template-columns: 1fr;
    }

    .chart-section.wide {
      grid-column: span 1;
    }
  }
</style>
