<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import Logo from '../lib/components/Logo.svelte';

  const dispatch = createEventDispatcher();

  let password = '';
  let error = '';
  let loading = false;

  async function handleLogin() {
    if (!password) {
      error = 'Password is required';
      return;
    }

    loading = true;
    error = '';

    try {
      const response = await fetch('/admin/api/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ password }),
      });

      if (response.ok) {
        dispatch('login');
      } else {
        error = 'Invalid password';
      }
    } catch (e) {
      error = 'Connection error - make sure the backend is running on port 8080';
      console.error('Login error:', e);
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-container">
  <div class="login-card">
    <div class="login-header">
      <Logo width={120} height={24} />
      <p class="login-subtitle">Admin Panel</p>
    </div>

    <form on:submit|preventDefault={handleLogin}>
      {#if error}
        <div class="error-message">{error}</div>
      {/if}

      <div class="form-group">
        <label for="password">Password</label>
        <input
          type="password"
          id="password"
          bind:value={password}
          placeholder="Enter admin password"
          disabled={loading}
        />
      </div>

      <button type="submit" class="btn btn-primary login-btn" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign In'}
      </button>
    </form>
  </div>
</div>

<style>
  .login-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    background: var(--bg-primary);
  }

  .login-card {
    background: var(--bg-secondary);
    border: 1px solid var(--border-color);
    border-radius: 12px;
    padding: 40px;
    width: 100%;
    max-width: 400px;
  }

  .login-header {
    text-align: center;
    margin-bottom: 32px;
  }

  .login-subtitle {
    color: var(--text-secondary);
    margin-top: 8px;
    font-size: 14px;
  }

  .form-group {
    margin-bottom: 20px;
  }

  .form-group label {
    display: block;
    margin-bottom: 8px;
    font-size: 14px;
    color: var(--text-secondary);
  }

  .form-group input {
    width: 100%;
    padding: 12px 16px;
    background: var(--bg-tertiary);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    color: var(--text-primary);
    font-size: 14px;
  }

  .form-group input:focus {
    outline: none;
    border-color: var(--accent-red);
  }

  .form-group input::placeholder {
    color: var(--text-muted);
  }

  .error-message {
    background: rgba(229, 9, 20, 0.1);
    border: 1px solid var(--accent-red);
    color: var(--accent-red);
    padding: 12px;
    border-radius: 8px;
    margin-bottom: 20px;
    font-size: 14px;
  }

  .login-btn {
    width: 100%;
    padding: 12px;
    font-size: 16px;
  }

  .login-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
