<script lang="ts">
  import Router from 'svelte-spa-router';
  import Sidebar from './lib/components/Sidebar.svelte';
  import Login from './routes/Login.svelte';

  // Routes
  import Dashboard from './routes/Dashboard.svelte';
  import Analytics from './routes/Analytics.svelte';
  import Home from './routes/Home.svelte';
  import Movies from './routes/Movies.svelte';
  import MovieDetail from './routes/MovieDetail.svelte';
  import TVShows from './routes/TVShows.svelte';
  import TVShowDetail from './routes/TVShowDetail.svelte';
  import Channels from './routes/Channels.svelte';
  import Curated from './routes/Curated.svelte';
  import ApiDocs from './routes/ApiDocs.svelte';
  import Settings from './routes/Settings.svelte';
  import Licenses from './routes/Licenses.svelte';

  let isAuthenticated = $state(false);
  let checkingAuth = $state(true);
  let licenseFeatures: string[] = $state([]);

  // Check if user is already authenticated
  async function checkAuth() {
    try {
      const response = await fetch('/admin/api/auth/check');
      isAuthenticated = response.ok;
      if (isAuthenticated) {
        loadLicenseFeatures();
      }
    } catch (e) {
      console.error('Auth check failed - backend may not be running:', e);
      isAuthenticated = false;
    }
    checkingAuth = false;
  }

  async function loadLicenseFeatures() {
    try {
      const res = await fetch('/admin/api/license-status');
      if (res.ok) {
        const data = await res.json();
        licenseFeatures = data.status?.features || [];
      }
    } catch (e) {
      console.error('Failed to load license features:', e);
    }
  }

  // Check auth on mount
  checkAuth();

  function handleLogin() {
    isAuthenticated = true;
    loadLicenseFeatures();
  }

  const routes = {
    '/': Dashboard,
    '/analytics': Analytics,
    '/home': Home,
    '/movies': Movies,
    '/movies/:id': MovieDetail,
    '/tvshows': TVShows,
    '/tvshows/:id': TVShowDetail,
    '/channels': Channels,
    '/curated': Curated,
    '/licenses': Licenses,
    '/api-docs': ApiDocs,
    '/settings': Settings,
  };
</script>

{#if checkingAuth}
  <div class="loading">
    <div class="spinner"></div>
  </div>
{:else if !isAuthenticated}
  <Login on:login={handleLogin} />
{:else}
  <div class="app-layout">
    <Sidebar features={licenseFeatures} />
    <main class="main-content">
      <Router {routes} />
    </main>
  </div>
{/if}

<style>
  .loading {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    background: var(--bg-primary);
  }

  .spinner {
    width: 40px;
    height: 40px;
    border: 3px solid var(--border-color);
    border-top-color: var(--accent-red);
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  @keyframes spin {
    to {
      transform: rotate(360deg);
    }
  }
</style>
