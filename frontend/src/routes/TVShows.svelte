<script lang="ts">
  import { onMount } from 'svelte';
  import { getSeries, type Series } from '../lib/api/client';

  let series: Series[] = [];
  let loading = true;

  onMount(async () => {
    try {
      const result = await getSeries({ limit: 20 });
      series = result.series;
    } catch (err) {
      console.error('Failed to load series:', err);
    } finally {
      loading = false;
    }
  });
</script>

<div class="tvshows-page">
  <header class="page-header">
    <h1 class="page-title">TV SHOWS</h1>
    <div class="page-actions">
      <button class="btn btn-secondary">SEARCH</button>
      <button class="btn btn-primary">ADD</button>
    </div>
  </header>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if series.length === 0}
    <div class="empty-state">
      <p>No TV shows found</p>
    </div>
  {:else}
    <div class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th>Poster</th>
            <th>Title</th>
            <th>Year</th>
            <th>Seasons</th>
            <th>Status</th>
            <th style="text-align: right;">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each series as show}
            <tr>
              <td>
                {#if show.poster_image}
                  <img src={show.poster_image} alt={show.title} class="poster" />
                {:else}
                  <div class="poster"></div>
                {/if}
              </td>
              <td>{show.title}</td>
              <td>{show.year}</td>
              <td>{show.total_seasons}</td>
              <td>
                <span class="status status-{show.status?.toLowerCase()}">{show.status}</span>
              </td>
              <td>
                <div class="actions">
                  <button class="btn btn-sm btn-warning">EDIT</button>
                  <button class="btn btn-sm btn-danger">DELETE</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<style>
  .status {
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 12px;
    text-transform: capitalize;
  }

  .status-continuing,
  .status-ongoing {
    background: rgba(46, 160, 67, 0.2);
    color: var(--accent-green);
  }

  .status-ended {
    background: rgba(139, 148, 158, 0.2);
    color: var(--text-muted);
  }
</style>
