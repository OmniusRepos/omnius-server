<script lang="ts">
  import { onMount } from 'svelte';
  import Modal from '../lib/components/Modal.svelte';

  interface HomeSection {
    id: number;
    section_id: string;
    title: string;
    section_type: string;
    display_type: string;
    content_type?: string;
    content_id?: number;
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

  interface Movie {
    id: number;
    title: string;
    year: number;
    medium_cover_image: string;
  }

  interface Series {
    id: number;
    title: string;
    year: number;
    poster_image: string;
  }

  let sections: HomeSection[] = $state([]);
  let curatedLists: CuratedList[] = $state([]);
  let loading = $state(true);
  let showAddModal = $state(false);
  let showEditModal = $state(false);
  let selectedSection: HomeSection | null = $state(null);

  // Content search
  let contentSearchQuery = $state('');
  let contentSearchResults: (Movie | Series)[] = $state([]);
  let searchingContent = $state(false);
  let selectedContent: { type: string; id: number; title: string; image: string } | null = $state(null);

  // Display types determine how section renders
  const displayTypes = [
    { value: 'carousel', label: 'Carousel', description: 'Horizontal scrolling row of posters', needsQuery: true },
    { value: 'top10', label: 'Top 10', description: 'Netflix-style ranked list with numbers', needsQuery: true },
    { value: 'grid', label: 'Grid', description: 'Grid layout of posters', needsQuery: true },
    { value: 'featured', label: 'Featured', description: 'Large featured cards with details', needsQuery: true },
    { value: 'hero', label: 'Hero', description: 'Full-width hero banner with single item', needsQuery: false },
    { value: 'banner', label: 'Banner', description: 'Promotional banner with single item', needsQuery: false },
  ];

  // Section types for query-based sections
  const sectionTypes = [
    { value: 'top_viewed', label: 'Top Viewed (Analytics)' },
    { value: 'recent', label: 'Recently Added' },
    { value: 'top_rated', label: 'Top Rated' },
    { value: 'genre', label: 'By Genre' },
    { value: 'curated_list', label: 'Curated List' },
    { value: 'query', label: 'Custom Query' },
  ];

  // Content types for hero/banner
  const contentTypes = [
    { value: 'movie', label: 'Movie' },
    { value: 'series', label: 'TV Series' },
    { value: 'channel', label: 'Live Channel' },
  ];

  const genres = [
    'Action', 'Adventure', 'Animation', 'Biography', 'Comedy', 'Crime',
    'Documentary', 'Drama', 'Family', 'Fantasy', 'History', 'Horror',
    'Music', 'Mystery', 'Romance', 'Sci-Fi', 'Sport', 'Thriller', 'War', 'Western'
  ];

  let form = $state({
    section_id: '',
    title: '',
    display_type: 'carousel',
    section_type: 'recent',
    content_type: 'movie',
    content_id: null as number | null,
    genre: '',
    curated_list_id: null as number | null,
    sort_by: 'rating',
    order_by: 'desc',
    minimum_rating: 0,
    limit_count: 10,
    is_active: true,
    display_order: 0,
  });

  // Computed: does current display type need a query or single content?
  function needsQuery(displayType: string): boolean {
    const dt = displayTypes.find(t => t.value === displayType);
    return dt?.needsQuery ?? true;
  }

  onMount(async () => {
    await Promise.all([loadSections(), loadCuratedLists()]);
  });

  async function loadSections() {
    loading = true;
    try {
      const res = await fetch('/admin/api/home/sections');
      if (res.ok) {
        sections = await res.json();
      } else {
        console.error('Failed to load sections:', await res.text());
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

  async function searchContent() {
    if (!contentSearchQuery.trim()) {
      contentSearchResults = [];
      return;
    }
    searchingContent = true;
    try {
      if (form.content_type === 'movie') {
        const res = await fetch(`/api/v2/list_movies.json?query_term=${encodeURIComponent(contentSearchQuery)}&limit=10`);
        if (res.ok) {
          const data = await res.json();
          contentSearchResults = data.data?.movies || [];
        }
      } else if (form.content_type === 'series') {
        const res = await fetch(`/api/v2/list_series.json?query_term=${encodeURIComponent(contentSearchQuery)}&limit=10`);
        if (res.ok) {
          const data = await res.json();
          contentSearchResults = data.data?.series || [];
        }
      }
    } catch (e) {
      console.error('Failed to search content:', e);
    } finally {
      searchingContent = false;
    }
  }

  function selectContent(item: Movie | Series) {
    const isMovie = 'medium_cover_image' in item;
    selectedContent = {
      type: form.content_type,
      id: item.id,
      title: item.title,
      image: isMovie ? (item as Movie).medium_cover_image : (item as Series).poster_image
    };
    form.content_id = item.id;
    contentSearchQuery = '';
    contentSearchResults = [];
  }

  function clearSelectedContent() {
    selectedContent = null;
    form.content_id = null;
  }

  function openAddModal() {
    form = {
      section_id: '',
      title: '',
      display_type: 'carousel',
      section_type: 'recent',
      content_type: 'movie',
      content_id: null,
      genre: '',
      curated_list_id: null,
      sort_by: 'rating',
      order_by: 'desc',
      minimum_rating: 0,
      limit_count: 10,
      is_active: true,
      display_order: sections.length,
    };
    selectedContent = null;
    contentSearchQuery = '';
    contentSearchResults = [];
    showAddModal = true;
  }

  function openEditModal(section: HomeSection) {
    selectedSection = section;
    form = {
      section_id: section.section_id,
      title: section.title,
      display_type: section.display_type || 'carousel',
      section_type: section.section_type || 'recent',
      content_type: section.content_type || 'movie',
      content_id: section.content_id || null,
      genre: section.genre || '',
      curated_list_id: section.curated_list_id || null,
      sort_by: section.sort_by || 'rating',
      order_by: section.order_by || 'desc',
      minimum_rating: section.minimum_rating || 0,
      limit_count: section.limit_count || 10,
      is_active: section.is_active,
      display_order: section.display_order,
    };
    // If it's a hero/banner with content, we'd need to load the content info
    if (section.content_id && !needsQuery(section.display_type)) {
      selectedContent = {
        type: section.content_type || 'movie',
        id: section.content_id,
        title: 'Content #' + section.content_id,
        image: ''
      };
    } else {
      selectedContent = null;
    }
    contentSearchQuery = '';
    contentSearchResults = [];
    showEditModal = true;
  }

  async function handleCreate() {
    // Auto-generate section_id if empty
    if (!form.section_id) {
      form.section_id = form.title.toLowerCase().replace(/[^a-z0-9]+/g, '_');
    }

    const payload = { ...form };
    // Clear irrelevant fields based on display type
    if (!needsQuery(form.display_type)) {
      // Hero/Banner - only need content_type and content_id
      payload.section_type = '';
      payload.genre = '';
      payload.curated_list_id = null;
    } else {
      // Query-based - clear content fields
      payload.content_type = '';
      payload.content_id = null;
    }

    try {
      const res = await fetch('/admin/api/home/sections', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        showAddModal = false;
        await loadSections();
      } else {
        const err = await res.text();
        alert('Failed to create section: ' + err);
      }
    } catch (e) {
      console.error('Failed to create section:', e);
    }
  }

  async function handleUpdate() {
    if (!selectedSection) return;

    const payload = { ...form, id: selectedSection.id };
    // Clear irrelevant fields based on display type
    if (!needsQuery(form.display_type)) {
      payload.section_type = '';
      payload.genre = '';
      payload.curated_list_id = null;
    } else {
      payload.content_type = '';
      payload.content_id = null;
    }

    try {
      const res = await fetch(`/admin/api/home/sections/${selectedSection.id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload),
      });
      if (res.ok) {
        showEditModal = false;
        await loadSections();
      } else {
        const err = await res.text();
        alert('Failed to update section: ' + err);
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

  function getDisplayTypeLabel(type: string): string {
    return displayTypes.find(t => t.value === type)?.label || type;
  }

  function getContentTypeLabel(type: string): string {
    return contentTypes.find(t => t.value === type)?.label || type;
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

  <p class="page-description">Configure the sections that appear on the Streamer home page.</p>

  {#if loading}
    <div class="loading">
      <div class="spinner"></div>
    </div>
  {:else if sections.length === 0}
    <div class="empty-state">
      <p>No home sections configured yet.</p>
      <p class="text-muted">Default sections will be shown when no custom sections exist.</p>
      <button class="btn btn-primary mt-4" onclick={openAddModal}>Add First Section</button>
    </div>
  {:else}
    <div class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 50px;">Order</th>
            <th>Title</th>
            <th>Display</th>
            <th>Content</th>
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
                <span class="badge badge-display">{getDisplayTypeLabel(section.display_type)}</span>
              </td>
              <td>
                {#if !needsQuery(section.display_type)}
                  <span class="badge badge-content">{getContentTypeLabel(section.content_type || 'movie')}</span>
                  {#if section.content_id}
                    <span class="config-tag">#{section.content_id}</span>
                  {/if}
                {:else if section.section_type === 'genre'}
                  <span class="badge badge-type">{getSectionTypeLabel(section.section_type)}</span>
                  <span class="config-tag">{section.genre}</span>
                {:else if section.section_type === 'curated_list'}
                  <span class="badge badge-type">{getSectionTypeLabel(section.section_type)}</span>
                  {#if section.curated_list_id}
                    {@const list = curatedLists.find(l => l.id === section.curated_list_id)}
                    <span class="config-tag">{list?.name || '#' + section.curated_list_id}</span>
                  {/if}
                {:else}
                  <span class="badge badge-type">{getSectionTypeLabel(section.section_type)}</span>
                  {#if section.minimum_rating > 0}
                    <span class="config-tag">≥{section.minimum_rating}★</span>
                  {/if}
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
      <label class="form-label" for="title">Section Title *</label>
      <input type="text" id="title" class="form-input" bind:value={form.title} placeholder="e.g., Featured Movie, Top 10, Oscars 2025" required />
    </div>

    <div class="form-group">
      <label class="form-label" for="section_id">Section ID</label>
      <input type="text" id="section_id" class="form-input" bind:value={form.section_id} placeholder="auto-generated from title" />
      <small class="text-muted">Leave empty to auto-generate</small>
    </div>

    <div class="form-group">
      <!-- svelte-ignore a11y_label_has_associated_control -->
      <label class="form-label">Display Type *</label>
      <div class="display-type-grid">
        {#each displayTypes as dt}
          <button
            type="button"
            class="display-type-option"
            class:selected={form.display_type === dt.value}
            onclick={() => form.display_type = dt.value}
          >
            <span class="dt-label">{dt.label}</span>
            <span class="dt-desc">{dt.description}</span>
          </button>
        {/each}
      </div>
    </div>

    {#if !needsQuery(form.display_type)}
      <!-- Hero/Banner: Pick specific content -->
      <div class="content-picker">
        <div class="form-group">
          <!-- svelte-ignore a11y_label_has_associated_control -->
          <label class="form-label">Content Type</label>
          <div class="content-type-row">
            {#each contentTypes as ct}
              <button
                type="button"
                class="content-type-btn"
                class:selected={form.content_type === ct.value}
                onclick={() => { form.content_type = ct.value; clearSelectedContent(); }}
              >
                {ct.label}
              </button>
            {/each}
          </div>
        </div>

        {#if selectedContent}
          <div class="selected-content">
            <div class="selected-content-info">
              {#if selectedContent.image}
                <img src={selectedContent.image} alt="" class="content-thumb" />
              {/if}
              <div>
                <strong>{selectedContent.title}</strong>
                <div class="text-muted">{getContentTypeLabel(selectedContent.type)} #{selectedContent.id}</div>
              </div>
            </div>
            <button type="button" class="btn btn-sm btn-secondary" onclick={clearSelectedContent}>Change</button>
          </div>
        {:else}
          <div class="form-group">
            <!-- svelte-ignore a11y_label_has_associated_control -->
            <label class="form-label">Search {getContentTypeLabel(form.content_type)}</label>
            <div class="search-row">
              <input
                type="text"
                class="form-input"
                bind:value={contentSearchQuery}
                placeholder="Search by title..."
                oninput={() => searchContent()}
              />
              {#if searchingContent}
                <span class="searching">Searching...</span>
              {/if}
            </div>
            {#if contentSearchResults.length > 0}
              <div class="search-results">
                {#each contentSearchResults as item}
                  <button type="button" class="search-result-item" onclick={() => selectContent(item)}>
                    {#if 'medium_cover_image' in item && item.medium_cover_image}
                      <img src={item.medium_cover_image} alt="" class="result-thumb" />
                    {:else if 'poster_image' in item && item.poster_image}
                      <img src={item.poster_image} alt="" class="result-thumb" />
                    {:else}
                      <div class="result-thumb placeholder"></div>
                    {/if}
                    <div class="result-info">
                      <strong>{item.title}</strong>
                      <span class="text-muted">{item.year}</span>
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {:else}
      <!-- Carousel/Grid/Featured: Configure query -->
      <div class="query-config">
        <div class="form-group">
          <label class="form-label" for="section_type">Data Source</label>
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
            <small class="text-muted">
              <a href="/admin/curated" class="link">Manage curated lists</a> (e.g., Oscars 2025, Best of 2024)
            </small>
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
                <option value="download_count">Downloads</option>
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
      <label class="form-label" for="edit_title">Section Title *</label>
      <input type="text" id="edit_title" class="form-input" bind:value={form.title} required />
    </div>

    <div class="form-group">
      <label class="form-label" for="edit_section_id">Section ID</label>
      <input type="text" id="edit_section_id" class="form-input" bind:value={form.section_id} />
    </div>

    <div class="form-group">
      <!-- svelte-ignore a11y_label_has_associated_control -->
      <label class="form-label">Display Type</label>
      <div class="display-type-grid">
        {#each displayTypes as dt}
          <button
            type="button"
            class="display-type-option"
            class:selected={form.display_type === dt.value}
            onclick={() => form.display_type = dt.value}
          >
            <span class="dt-label">{dt.label}</span>
            <span class="dt-desc">{dt.description}</span>
          </button>
        {/each}
      </div>
    </div>

    {#if !needsQuery(form.display_type)}
      <!-- Hero/Banner: Pick specific content -->
      <div class="content-picker">
        <div class="form-group">
          <!-- svelte-ignore a11y_label_has_associated_control -->
          <label class="form-label">Content Type</label>
          <div class="content-type-row">
            {#each contentTypes as ct}
              <button
                type="button"
                class="content-type-btn"
                class:selected={form.content_type === ct.value}
                onclick={() => { form.content_type = ct.value; clearSelectedContent(); }}
              >
                {ct.label}
              </button>
            {/each}
          </div>
        </div>

        {#if selectedContent}
          <div class="selected-content">
            <div class="selected-content-info">
              {#if selectedContent.image}
                <img src={selectedContent.image} alt="" class="content-thumb" />
              {/if}
              <div>
                <strong>{selectedContent.title}</strong>
                <div class="text-muted">{getContentTypeLabel(selectedContent.type)} #{selectedContent.id}</div>
              </div>
            </div>
            <button type="button" class="btn btn-sm btn-secondary" onclick={clearSelectedContent}>Change</button>
          </div>
        {:else}
          <div class="form-group">
            <!-- svelte-ignore a11y_label_has_associated_control -->
            <label class="form-label">Search {getContentTypeLabel(form.content_type)}</label>
            <div class="search-row">
              <input
                type="text"
                class="form-input"
                bind:value={contentSearchQuery}
                placeholder="Search by title..."
                oninput={() => searchContent()}
              />
              {#if searchingContent}
                <span class="searching">Searching...</span>
              {/if}
            </div>
            {#if contentSearchResults.length > 0}
              <div class="search-results">
                {#each contentSearchResults as item}
                  <button type="button" class="search-result-item" onclick={() => selectContent(item)}>
                    {#if 'medium_cover_image' in item && item.medium_cover_image}
                      <img src={item.medium_cover_image} alt="" class="result-thumb" />
                    {:else if 'poster_image' in item && item.poster_image}
                      <img src={item.poster_image} alt="" class="result-thumb" />
                    {:else}
                      <div class="result-thumb placeholder"></div>
                    {/if}
                    <div class="result-info">
                      <strong>{item.title}</strong>
                      <span class="text-muted">{item.year}</span>
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {:else}
      <!-- Carousel/Grid/Featured: Configure query -->
      <div class="query-config">
        <div class="form-group">
          <label class="form-label" for="edit_section_type">Data Source</label>
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
            <small class="text-muted">
              <a href="/admin/curated" class="link">Manage curated lists</a>
            </small>
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
                <option value="download_count">Downloads</option>
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

  .badge-display {
    background: #8b5cf6;
    color: white;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 11px;
    font-weight: 600;
  }

  .badge-content {
    background: #f59e0b;
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
    margin-left: 4px;
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

  /* Display Type Grid */
  .display-type-grid {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 8px;
  }

  .display-type-option {
    padding: 12px;
    background: var(--bg-tertiary);
    border: 2px solid var(--border-color);
    border-radius: 8px;
    cursor: pointer;
    text-align: left;
    transition: all 0.15s;
  }

  .display-type-option:hover {
    border-color: var(--accent-blue);
  }

  .display-type-option.selected {
    border-color: var(--accent-blue);
    background: rgba(59, 130, 246, 0.1);
  }

  .dt-label {
    display: block;
    font-weight: 600;
    font-size: 13px;
    color: var(--text-primary);
  }

  .dt-desc {
    display: block;
    font-size: 11px;
    color: var(--text-muted);
    margin-top: 2px;
  }

  /* Content Type Row */
  .content-type-row {
    display: flex;
    gap: 8px;
  }

  .content-type-btn {
    flex: 1;
    padding: 10px;
    background: var(--bg-tertiary);
    border: 2px solid var(--border-color);
    border-radius: 6px;
    cursor: pointer;
    font-weight: 500;
    color: var(--text-secondary);
    transition: all 0.15s;
  }

  .content-type-btn:hover {
    border-color: var(--accent-blue);
  }

  .content-type-btn.selected {
    border-color: #f59e0b;
    background: rgba(245, 158, 11, 0.1);
    color: #f59e0b;
  }

  /* Content Picker */
  .content-picker {
    background: var(--bg-secondary);
    padding: 16px;
    border-radius: 8px;
    margin-bottom: 16px;
  }

  .selected-content {
    display: flex;
    align-items: center;
    justify-content: space-between;
    background: var(--bg-tertiary);
    padding: 12px;
    border-radius: 8px;
  }

  .selected-content-info {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .content-thumb {
    width: 48px;
    height: 72px;
    object-fit: cover;
    border-radius: 4px;
  }

  /* Search */
  .search-row {
    display: flex;
    align-items: center;
    gap: 8px;
  }

  .searching {
    font-size: 12px;
    color: var(--text-muted);
  }

  .search-results {
    margin-top: 8px;
    background: var(--bg-primary);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    max-height: 200px;
    overflow-y: auto;
  }

  .search-result-item {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    padding: 8px 12px;
    background: none;
    border: none;
    border-bottom: 1px solid var(--border-color);
    cursor: pointer;
    text-align: left;
  }

  .search-result-item:last-child {
    border-bottom: none;
  }

  .search-result-item:hover {
    background: var(--bg-secondary);
  }

  .result-thumb {
    width: 32px;
    height: 48px;
    object-fit: cover;
    border-radius: 4px;
    background: var(--bg-tertiary);
  }

  .result-thumb.placeholder {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .result-info {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .result-info strong {
    font-size: 13px;
    color: var(--text-primary);
  }

  /* Query Config */
  .query-config {
    background: var(--bg-secondary);
    padding: 16px;
    border-radius: 8px;
    margin-bottom: 16px;
  }

  .form-row {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 16px;
  }

  .form-group {
    margin-bottom: 16px;
  }

  .form-group:last-child {
    margin-bottom: 0;
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

  .link {
    color: var(--accent-blue);
    text-decoration: underline;
  }
</style>
