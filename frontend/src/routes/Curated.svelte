<script lang="ts">
  import { onMount } from 'svelte';
  import { getCuratedLists, type CuratedList } from '../lib/api/client';

  let lists: CuratedList[] = [];
  let loading = true;

  onMount(async () => {
    try {
      lists = (await getCuratedLists()) || [];
    } catch (err) {
      console.error('Failed to load curated lists:', err);
    } finally {
      loading = false;
    }
  });
</script>

<div class="curated-page">
  <header class="page-header">
    <h1 class="page-title">CURATED LISTS</h1>
    <div class="page-actions">
      <button class="btn btn-primary">CREATE LIST</button>
    </div>
  </header>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if lists.length === 0}
    <div class="empty-state">
      <p>No curated lists yet. Create one to get started.</p>
    </div>
  {:else}
    <div class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Slug</th>
            <th>Sort By</th>
            <th>Filters</th>
            <th>Limit</th>
            <th>Active</th>
            <th style="text-align: right;">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each lists as list}
            <tr>
              <td class="list-name">{list.name}</td>
              <td><code>{list.slug}</code></td>
              <td>{list.sort_by} {list.order_by}</td>
              <td class="filters">
                {#if list.minimum_rating}
                  <span class="filter">Rating ≥ {list.minimum_rating}</span>
                {/if}
                {#if list.genre}
                  <span class="filter">Genre: {list.genre}</span>
                {/if}
                {#if list.minimum_year}
                  <span class="filter">Year ≥ {list.minimum_year}</span>
                {/if}
                {#if list.maximum_year}
                  <span class="filter">Year ≤ {list.maximum_year}</span>
                {/if}
              </td>
              <td>{list.limit}</td>
              <td>
                {#if list.is_active}
                  <span class="status-active">Active</span>
                {:else}
                  <span class="status-inactive">Inactive</span>
                {/if}
              </td>
              <td>
                <div class="actions">
                  <button class="btn btn-sm btn-secondary">MANAGE</button>
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
  .list-name {
    font-weight: 500;
  }

  code {
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 12px;
  }

  .filters {
    display: flex;
    flex-wrap: wrap;
    gap: 4px;
  }

  .filter {
    background: var(--bg-tertiary);
    padding: 2px 6px;
    border-radius: 4px;
    font-size: 11px;
    color: var(--text-secondary);
  }

  .status-active {
    color: var(--accent-green);
  }

  .status-inactive {
    color: var(--text-muted);
  }
</style>
