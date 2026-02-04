<script lang="ts">
  export let label: string;
  export let value: number | string;
  export let suffix: string = '';
</script>

<div class="stat-card">
  <div class="stat-label">
    <span class="label-main">{label}</span>
    {#if suffix}
      <span class="label-suffix">{suffix}</span>
    {/if}
  </div>
  <div class="stat-value">
    {typeof value === 'number' ? value.toLocaleString() : value}
  </div>
</div>

<style>
  .stat-card {
    background: var(--bg-card);
    border: 1px solid var(--border-color);
    border-radius: 8px;
    padding: 16px 20px;
    min-width: 160px;
  }

  .stat-label {
    font-size: 12px;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.5px;
    margin-bottom: 8px;
  }

  .label-main {
    color: var(--text-muted);
  }

  .label-suffix {
    color: var(--text-primary);
  }

  .stat-value {
    font-size: 28px;
    font-weight: 300;
    color: var(--text-primary);
    font-variant-numeric: tabular-nums;
  }
</style>
