<script lang="ts">
  import { onMount } from 'svelte';
  import Modal from '../lib/components/Modal.svelte';

  interface HomeSection {
    id: number;
    section_id: string;
    title: string;
    section_type: string;
    query_type?: string;
    genre?: string;
    curated_list_id?: number;
    sort_by: string;
    order_by: string;
    minimum_rating: number;
    limit_count: number;
    is_active: boolean;
    display_order: number;
  }

  interface CuratedList {
    id: number;
    name: string;
    slug: string;
  }

  let sections: HomeSection[] = $state([]);
  let curatedLists: CuratedList[] = $state([]);
  let loading = $state(true);
  let showAddModal = $state(false);
  let showEditModal = $state(false);
  let selectedSection: HomeSection | null = $state(null);

  const sectionTypes = [
    { value: 'recent', label: 'Recently Added' },
    { value: 'top_rated', label: 'Top Rated' },
    { value: 'genre', label: 'By Genre' },
    { value: 'curated_list', label: 'Curated List' },
    { value: 'query', label: 'Custom Query' },
  ];

  const genres = [
    'Action', 'Adventure', 'Animation', 'Biography', 'Comedy', 'Crime',
    'Documentary', 'Drama', 'Family', 'Fantasy', 'History', 'Horror',
    'Music', 'Mystery', 'Romance', 'Sci-Fi', 'Sport', 'Thriller', 'War', 'Western'
  ];

  let form = $state({
    section_id: '',
    title: '',
    section_type: 'recent',
    query_type: '',
    genre: '',
    curated_list_id: null as number | null,
    sort_by: 'rating',
    order_by: 'desc',
    minimum_rating: 0,
    limit_count: 10,
    is_active: true,
    display_order: 0,
  });

  onMount(async () => {
    await Promise.all([loadSections(), loadCuratedLists()]);
  });

  async function loadSections() {
    loading = true;
    try {
      const res = await fetch('/admin/api/home/sections');
      if (res.ok) {
        sections = await res.json();
      }
    } catch (e) {
      console.error('Failed to load sections:', e);
    } finally {
      loading = false;
    }
  }

  async function loadCuratedLists() {
    try {
      const res = await fetch('/admin/api/curated');
      if (res.ok) {
        curatedLists = await res.json();
      }
    } catch (e) {
      console.error('Failed to load curated lists:', e);
    }
  }

  function openAddModal() {
    form = {
      section_id: '',
      title: '',
      section_type: 'recent',
      query_type: '',
      genre: '',
      curated_list_id: null,
      sort_by: 'rating',
      order_by: 'desc',
      minimum_rating: 0,
      limit_count: 10,
      is_active: true,
      display_order: sections.length,
    };
    showAddModal = true;
  }

  function openEditModal(section: HomeSection) {
    selectedSection = section;
    form = {
      section_id: section.section_id,
      title: section.title,
      section_type: section.section_type,
      query_type: section.query_type || '',
      genre: section.genre || '',
      curated_list_id: section.curated_list_id || null,
      sort_by: section.sort_by,
      order_by: section.order_by,
      minimum_rating: section.minimum_rating,
      limit_count: section.limit_count,
      is_active: section.is_active,
      display_order: section.display_order,
    };
    showEditModal = true;
  }

  async function handleCreate() {
    // Auto-generate section_id if empty
    if (!form.section_id) {
      form.section_id = form.title.toLowerCase().replace(/[^a-z0-9]+/g, '_');
    }

    try {
      const res = await fetch('/admin/api/home/sections', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });
      if (res.ok) {
        showAddModal = false;
        await loadSections();
      }
    } catch (e) {
      console.error('Failed to create section:', e);
    }
  }

  async function handleUpdate() {
    if (!selectedSection) return;
    try {
      const res = await fetch(`/admin/api/home/sections/${selectedSection.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...form, id: selectedSection.id }),
      });
      if (res.ok) {
        showEditModal = false;
        await loadSections();
      }
    } catch (e) {
      console.error('Failed to update section:', e);
    }
  }

  async function handleDelete(section: HomeSection) {
    if (!confirm(`Delete "${section.title}"?`)) return;
    try {
      await fetch(`/admin/api/home/sections/${section.id}`, { method: 'DELETE' });
      await loadSections();
    } catch (e) {
      console.error('Failed to delete section:', e);
    }
  }

  async function toggleActive(section: HomeSection) {
    try {
      await fetch(`/admin/api/home/sections/${section.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ...section, is_active: !section.is_active }),
      });
      await loadSections();
    } catch (e) {
      console.error('Failed to toggle section:', e);
    }
  }

  async function moveSection(section: HomeSection, direction: 'up' | 'down') {
    const idx = sections.findIndex(s => s.id === section.id);
    if (direction === 'up' && idx > 0) {
      const newOrder = [...sections];
      [newOrder[idx - 1], newOrder[idx]] = [newOrder[idx], newOrder[idx - 1]];
      await reorderSections(newOrder);
    } else if (direction === 'down' && idx < sections.length - 1) {
      const newOrder = [...sections];
      [newOrder[idx], newOrder[idx + 1]] = [newOrder[idx + 1], newOrder[idx]];
      await reorderSections(newOrder);
    }
  }

  async function reorderSections(newOrder: HomeSection[]) {
    const ids = newOrder.map(s => s.id);
    try {
      await fetch('/admin/api/home/sections/reorder', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ ids }),
      });
      await loadSections();
    } catch (e) {
      console.error('Failed to reorder sections:', e);
    }
  }

  function getSectionTypeLabel(type: string): string {
    return sectionTypes.find(t => t.value === type)?.label || type;
  }
</script>

<div class="home-page">
  <header class="page-header">
    <h1 class="page-title">HOME SECTIONS</h1>
    <div class="page-actions">
      <a href="/api/v2/home.json" target="_blank" class="btn btn-secondary">Preview API</a>
      <button class="btn btn-primary" onclick={openAddModal}>ADD SECTION</button>
    </div>
  </header>

  <p class="page-description">Configure the sections that appear on the Streamer home page. Drag to reorder.</p>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if sections.length === 0}
    <div class="empty-state">
      <p>No home sections configured yet.</p>
      <p class="text-muted">Default sections (Recently Added, Top Rated, Curated Lists) will be shown.</p>
      <button class="btn btn-primary mt-4" onclick={openAddModal}>Add First Section</button>
    </div>
  {:else}
    <div class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 50px;">Order</th>
            <th>Title</th>
            <th>Type</th>
            <th>Config</th>
            <th>Limit</th>
            <th>Status</th>
            <th style="text-align: right;">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each sections as section, idx}
            <tr class:inactive={!section.is_active}>
              <td>
                <div class="order-controls">
                  <button class="btn-icon" onclick={() => moveSection(section, 'up')} disabled={idx === 0}>▲</button>
                  <button class="btn-icon" onclick={() => moveSection(section, 'down')} disabled={idx === sections.length - 1}>▼</button>
                </div>
              </td>
              <td>
                <strong>{section.title}</strong>
                <div class="section-id">{section.section_id}</div>
              </td>
              <td>
                <span class="badge badge-type">{getSectionTypeLabel(section.section_type)}</span>
              </td>
              <td>
                {#if section.section_type === 'genre'}
                  <span class="config-tag">{section.genre}</span>
                {:else if section.section_type === 'curated_list'}
                  <span class="config-tag">List #{section.curated_list_id}</span>
                {:else if section.minimum_rating > 0}
                  <span class="config-tag">≥ {section.minimum_rating}★</span>
                {:else}
                  <span class="text-muted">—</span>
                {/if}
              </td>
              <td>{section.limit_count}</td>
              <td>
                <button
                  class="status-toggle"
                  class:active={section.is_active}
                  onclick={() => toggleActive(section)}
                >
                  {section.is_active ? 'Active' : 'Inactive'}
                </button>
              </td>
              <td>
                <div class="actions">
                  <button class="btn btn-sm btn-secondary" onclick={() => openEditModal(section)}>EDIT</button>
                  <button class="btn btn-sm btn-danger" onclick={() => handleDelete(section)}>DELETE</button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}
</div>

<!-- Add Section Modal -->
<Modal bind:open={showAddModal} title="Add Home Section" size="md" on:close={() => showAddModal = false}>
  <form onsubmit={(e) => { e.preventDefault(); handleCreate(); }}>
    <div class="form-group">
      <label class="form-label" for="title">Title *</label>
      <input type="text" id="title" class="form-input" bind:value={form.title} placeholder="Recently Added" required />
    </div>

    <div class="form-group">
      <label class="form-label" for="section_id">Section ID</label>
      <input type="text" id="section_id" class="form-input" bind:value={form.section_id} placeholder="auto-generated from title" />
      <small class="text-muted">Leave empty to auto-generate</small>
    </div>

    <div class="form-group">
      <label class="form-label" for="section_type">Section Type</label>
      <select id="section_type" class="form-input" bind:value={form.section_type}>
        {#each sectionTypes as type}
          <option value={type.value}>{type.label}</option>
        {/each}
      </select>
    </div>

    {#if form.section_type === 'genre'}
      <div class="form-group">
        <label class="form-label" for="genre">Genre</label>
        <select id="genre" class="form-input" bind:value={form.genre}>
          <option value="">Select genre...</option>
          {#each genres as genre}
            <option value={genre}>{genre}</option>
          {/each}
        </select>
      </div>
    {/if}

    {#if form.section_type === 'curated_list'}
      <div class="form-group">
        <label class="form-label" for="curated_list">Curated List</label>
        <select id="curated_list" class="form-input" bind:value={form.curated_list_id}>
          <option value={null}>Select list...</option>
          {#each curatedLists as list}
            <option value={list.id}>{list.name}</option>
          {/each}
        </select>
      </div>
    {/if}

    {#if form.section_type !== 'curated_list'}
      <div class="form-row">
        <div class="form-group">
          <label class="form-label" for="sort_by">Sort By</label>
          <select id="sort_by" class="form-input" bind:value={form.sort_by}>
            <option value="rating">Rating</option>
            <option value="date_uploaded">Date Added</option>
            <option value="year">Year</option>
            <option value="title">Title</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label" for="order_by">Order</label>
          <select id="order_by" class="form-input" bind:value={form.order_by}>
            <option value="desc">Descending</option>
            <option value="asc">Ascending</option>
          </select>
        </div>
      </div>

      <div class="form-row">
        <div class="form-group">
          <label class="form-label" for="minimum_rating">Minimum Rating</label>
          <input type="number" id="minimum_rating" class="form-input" bind:value={form.minimum_rating} min="0" max="10" step="0.1" />
        </div>
        <div class="form-group">
          <label class="form-label" for="limit_count">Limit</label>
          <input type="number" id="limit_count" class="form-input" bind:value={form.limit_count} min="1" max="50" />
        </div>
      </div>
    {/if}
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" onclick={() => showAddModal = false}>Cancel</button>
    <button class="btn btn-primary" onclick={handleCreate}>Create Section</button>
  </svelte:fragment>
</Modal>

<!-- Edit Section Modal -->
<Modal bind:open={showEditModal} title="Edit Home Section" size="md" on:close={() => showEditModal = false}>
  <form onsubmit={(e) => { e.preventDefault(); handleUpdate(); }}>
    <div class="form-group">
      <label class="form-label" for="edit_title">Title *</label>
      <input type="text" id="edit_title" class="form-input" bind:value={form.title} required />
    </div>

    <div class="form-group">
      <label class="form-label" for="edit_section_id">Section ID</label>
      <input type="text" id="edit_section_id" class="form-input" bind:value={form.section_id} />
    </div>

    <div class="form-group">
      <label class="form-label" for="edit_section_type">Section Type</label>
      <select id="edit_section_type" class="form-input" bind:value={form.section_type}>
        {#each sectionTypes as type}
          <option value={type.value}>{type.label}</option>
        {/each}
      </select>
    </div>

    {#if form.section_type === 'genre'}
      <div class="form-group">
        <label class="form-label" for="edit_genre">Genre</label>
        <select id="edit_genre" class="form-input" bind:value={form.genre}>
          <option value="">Select genre...</option>
          {#each genres as genre}
            <option value={genre}>{genre}</option>
          {/each}
        </select>
      </div>
    {/if}

    {#if form.section_type === 'curated_list'}
      <div class="form-group">
        <label class="form-label" for="edit_curated_list">Curated List</label>
        <select id="edit_curated_list" class="form-input" bind:value={form.curated_list_id}>
          <option value={null}>Select list...</option>
          {#each curatedLists as list}
            <option value={list.id}>{list.name}</option>
          {/each}
        </select>
      </div>
    {/if}

    {#if form.section_type !== 'curated_list'}
      <div class="form-row">
        <div class="form-group">
          <label class="form-label" for="edit_sort_by">Sort By</label>
          <select id="edit_sort_by" class="form-input" bind:value={form.sort_by}>
            <option value="rating">Rating</option>
            <option value="date_uploaded">Date Added</option>
            <option value="year">Year</option>
            <option value="title">Title</option>
          </select>
        </div>
        <div class="form-group">
          <label class="form-label" for="edit_order_by">Order</label>
          <select id="edit_order_by" class="form-input" bind:value={form.order_by}>
            <option value="desc">Descending</option>
            <option value="asc">Ascending</option>
          </select>
        </div>
      </div>

      <div class="form-row">
        <div class="form-group">
          <label class="form-label" for="edit_minimum_rating">Minimum Rating</label>
          <input type="number" id="edit_minimum_rating" class="form-input" bind:value={form.minimum_rating} min="0" max="10" step="0.1" />
        </div>
        <div class="form-group">
          <label class="form-label" for="edit_limit_count">Limit</label>
          <input type="number" id="edit_limit_count" class="form-input" bind:value={form.limit_count} min="1" max="50" />
        </div>
      </div>
    {/if}

    <div class="form-group">
      <label class="toggle-label">
        <input type="checkbox" bind:checked={form.is_active} />
        Active
      </label>
    </div>
  </form>
  <svelte:fragment slot="footer">
    <button class="btn btn-secondary" onclick={() => showEditModal = false}>Cancel</button>
    <button class="btn btn-primary" onclick={handleUpdate}>Save Changes</button>
  </svelte:fragment>
</Modal>

<style>
  .page-description {
    color: var(--text-muted);
    margin-bottom: 24px;
  }

  .section-id {
    font-size: 12px;
    color: var(--text-muted);
    font-family: monospace;
  }

  .order-controls {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .btn-icon {
    padding: 2px 6px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 4px;
    color: var(--text-muted);
    cursor: pointer;
    font-size: 10px;
  }

  .btn-icon:hover:not(:disabled) {
    background: var(--bg-secondary);
    color: var(--text-primary);
  }

  .btn-icon:disabled {
    opacity: 0.3;
    cursor: not-allowed;
  }

  .badge-type {
    background: var(--accent-blue);
    color: white;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
  }

  .config-tag {
    background: var(--bg-tertiary);
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 12px;
  }

  .status-toggle {
    padding: 4px 12px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 500;
    border: none;
    cursor: pointer;
    transition: all 0.15s;
  }

  .status-toggle.active {
    background: rgba(34, 197, 94, 0.2);
    color: #22c55e;
  }

  .status-toggle:not(.active) {
    background: rgba(156, 163, 175, 0.2);
    color: #9ca3af;
  }

  tr.inactive {
    opacity: 0.5;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-label {
    display: block;
    margin-bottom: 6px;
    font-size: 13px;
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

  .toggle-label {
    display: flex;
    align-items: center;
    gap: 8px;
    cursor: pointer;
  }

  .toggle-label input[type="checkbox"] {
    width: 18px;
    height: 18px;
    accent-color: var(--accent-green, #22c55e);
  }

  .empty-state {
    text-align: center;
    padding: 48px;
  }

  .mt-4 {
    margin-top: 16px;
  }
</style>
