<script lang="ts">
  import { link } from 'svelte-spa-router';
  import { location } from 'svelte-spa-router';
  import Logo from './Logo.svelte';

  const version = '1.0.0';
  const commit = 'abc212';

  interface NavItem {
    path: string;
    label: string;
  }

  const mainNav: NavItem[] = [
    { path: '/', label: 'Dashboard' },
    { path: '/analytics', label: 'Analytics' },
  ];

  const contentNav: NavItem[] = [
    { path: '/home', label: 'Home Sections' },
    { path: '/movies', label: 'Movies' },
    { path: '/tvshows', label: 'TV Shows' },
    { path: '/channels', label: 'Live Channels' },
    { path: '/curated', label: 'Curated Lists' },
  ];

  const bottomNav: NavItem[] = [
    { path: '/licenses', label: 'Licenses' },
    { path: '/api-docs', label: 'API Docs' },
    { path: '/settings', label: 'Settings' },
  ];

  function isActive(path: string, current: string): boolean {
    if (path === '/') return current === '/';
    return current.startsWith(path);
  }

  function getIcon(path: string): string {
    const icons: Record<string, string> = {
      '/': 'dashboard',
      '/analytics': 'analytics',
      '/movies': 'movie',
      '/tvshows': 'tv',
      '/channels': 'live',
      '/curated': 'list',
      '/api-docs': 'docs',
      '/settings': 'settings',
    };
    return icons[path] || 'default';
  }
</script>

<aside class="sidebar">
  <div class="sidebar-header">
    <Logo width={91} height={18} />
  </div>

  <nav class="sidebar-nav">
    <div class="nav-section">
      {#each mainNav as item}
        <a
          href={item.path}
          use:link
          class="nav-item"
          class:active={isActive(item.path, $location)}
        >
          {#if item.path === '/'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="3" width="7" height="9" rx="1"/>
              <rect x="14" y="3" width="7" height="5" rx="1"/>
              <rect x="14" y="12" width="7" height="9" rx="1"/>
              <rect x="3" y="16" width="7" height="5" rx="1"/>
            </svg>
          {:else if item.path === '/analytics'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="18" y1="20" x2="18" y2="10"/>
              <line x1="12" y1="20" x2="12" y2="4"/>
              <line x1="6" y1="20" x2="6" y2="14"/>
            </svg>
          {/if}
          {item.label}
        </a>
      {/each}
    </div>

    <div class="nav-section">
      {#each contentNav as item}
        <a
          href={item.path}
          use:link
          class="nav-item"
          class:active={isActive(item.path, $location)}
        >
          {#if item.path === '/home'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>
              <polyline points="9 22 9 12 15 12 15 22"/>
            </svg>
          {:else if item.path === '/movies'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="2" width="20" height="20" rx="2.18" ry="2.18"/>
              <line x1="7" y1="2" x2="7" y2="22"/>
              <line x1="17" y1="2" x2="17" y2="22"/>
              <line x1="2" y1="12" x2="22" y2="12"/>
              <line x1="2" y1="7" x2="7" y2="7"/>
              <line x1="2" y1="17" x2="7" y2="17"/>
              <line x1="17" y1="17" x2="22" y2="17"/>
              <line x1="17" y1="7" x2="22" y2="7"/>
            </svg>
          {:else if item.path === '/tvshows'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="2" y="7" width="20" height="15" rx="2" ry="2"/>
              <polyline points="17 2 12 7 7 2"/>
            </svg>
          {:else if item.path === '/channels'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="2"/>
              <path d="M16.24 7.76a6 6 0 0 1 0 8.49m-8.48-.01a6 6 0 0 1 0-8.49m11.31-2.82a10 10 0 0 1 0 14.14m-14.14 0a10 10 0 0 1 0-14.14"/>
            </svg>
          {:else if item.path === '/curated'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <line x1="8" y1="6" x2="21" y2="6"/>
              <line x1="8" y1="12" x2="21" y2="12"/>
              <line x1="8" y1="18" x2="21" y2="18"/>
              <line x1="3" y1="6" x2="3.01" y2="6"/>
              <line x1="3" y1="12" x2="3.01" y2="12"/>
              <line x1="3" y1="18" x2="3.01" y2="18"/>
            </svg>
          {/if}
          {item.label}
        </a>
      {/each}
    </div>
  </nav>

  <div class="sidebar-footer">
    <div class="nav-section">
      {#each bottomNav as item}
        <a
          href={item.path}
          use:link
          class="nav-item"
          class:active={isActive(item.path, $location)}
        >
          {#if item.path === '/licenses'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <rect x="3" y="11" width="18" height="11" rx="2" ry="2"/>
              <path d="M7 11V7a5 5 0 0 1 10 0v4"/>
            </svg>
          {:else if item.path === '/api-docs'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
              <polyline points="14 2 14 8 20 8"/>
              <line x1="16" y1="13" x2="8" y2="13"/>
              <line x1="16" y1="17" x2="8" y2="17"/>
              <polyline points="10 9 9 9 8 9"/>
            </svg>
          {:else if item.path === '/settings'}
            <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="3"/>
              <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"/>
            </svg>
          {/if}
          {item.label}
        </a>
      {/each}
      <a href="/admin/logout" class="nav-item logout">
        <svg class="nav-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4"/>
          <polyline points="16 17 21 12 16 7"/>
          <line x1="21" y1="12" x2="9" y2="12"/>
        </svg>
        Logout
      </a>
    </div>

    <div class="version">
      version: {version} | commit: {commit}
    </div>
  </div>
</aside>

<style>
  .sidebar {
    position: fixed;
    left: 0;
    top: 0;
    bottom: 0;
    width: var(--sidebar-width);
    background: #0a0a0a;
    display: flex;
    flex-direction: column;
    border-right: 1px solid var(--border-color);
  }

  .sidebar-header {
    padding: 20px;
  }

  .sidebar-nav {
    flex: 1;
    padding: 0 12px;
    overflow-y: auto;
  }

  .nav-section {
    margin-bottom: 24px;
  }

  .nav-item {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 10px 12px;
    color: var(--text-secondary);
    text-decoration: none;
    border-radius: 6px;
    font-size: 14px;
    transition: all var(--transition-fast);
  }

  .nav-item:hover {
    color: var(--text-primary);
    background: var(--bg-tertiary);
    text-decoration: none;
  }

  .nav-item.active {
    color: var(--text-primary);
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
  }

  .nav-icon {
    flex-shrink: 0;
    opacity: 0.7;
  }

  .nav-item:hover .nav-icon,
  .nav-item.active .nav-icon {
    opacity: 1;
  }

  .sidebar-footer {
    padding: 12px;
    border-top: 1px solid var(--border-color);
  }

  .logout {
    color: var(--accent-red) !important;
  }

  .logout .nav-icon {
    stroke: var(--accent-red);
  }

  .version {
    padding: 12px;
    font-size: 11px;
    color: var(--text-muted);
  }
</style>
