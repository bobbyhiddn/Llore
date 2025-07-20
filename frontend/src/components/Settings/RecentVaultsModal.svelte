<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { main } from '@wailsjs/go/models';

  export let recentVaults: main.RecentVault[] = [];

  const dispatch = createEventDispatcher();

  function formatPath(path: string): string {
    const parts = path.split(/[\/\\]/);
    if (parts.length > 2) {
      return `.../${parts[parts.length - 2]}/${parts[parts.length - 1]}`;
    }
    return path;
  }

  function formatTime(time: any): string {
    if (!time) return '';
    const date = new Date(time);
    return date.toLocaleString();
  }
</script>

<div class="modal-backdrop" on:click={() => dispatch('close')}>
  <div class="modal" on:click|stopPropagation>
    <h3>Recent Vaults</h3>

    {#if recentVaults.length === 0}
      <p class="empty-state">No recent vaults found.</p>
    {:else}
      <ul class="vault-list">
        {#each recentVaults as vault (vault.Path)}
          <li class="vault-item">
            <button class="vault-select-btn" on:click={() => dispatch('select', vault.Path)} title={vault.Path}>
              <span class="vault-name">{vault.Path.split(/[\/\\]/).pop()}</span>
              <span class="vault-path">{formatPath(vault.Path)}</span>
              <span class="vault-time">Last Opened: {formatTime(vault.LastAccessed)}</span>
            </button>
            <button class="remove-btn" on:click|stopPropagation={() => dispatch('remove', vault.Path)} title="Remove from list">
              &times;
            </button>
          </li>
        {/each}
      </ul>
    {/if}

    <div class="modal-actions">
      <button class="browse-btn" on:click={() => dispatch('browse')}>
        Browse for Vault...
      </button>
      <button on:click={() => dispatch('close')}>Cancel</button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed; inset: 0; background: rgba(0, 0, 0, 0.7);
    display: flex; align-items: center; justify-content: center; z-index: 1000;
  }
  .modal {
    background: var(--bg-primary); border: 1px solid var(--border-color-medium);
    border-radius: 12px; padding: 1.5rem; width: 90%; max-width: 600px;
    max-height: 80vh; display: flex; flex-direction: column;
  }
  h3 { margin: 0 0 1rem 0; color: var(--text-primary); }
  .vault-list { list-style: none; padding: 0; margin: 0; flex-grow: 1; overflow-y: auto; }
  .vault-item {
    display: flex; align-items: center; gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .vault-select-btn {
    flex-grow: 1; display: flex; flex-direction: column; text-align: left;
    padding: 0.75rem 1rem; background: var(--bg-secondary);
    border: 1px solid var(--border-color-medium); border-radius: 6px;
    color: var(--text-primary); cursor: pointer; transition: all 0.2s ease;
  }
  .vault-select-btn:hover { background: var(--bg-hover-medium); border-color: var(--accent-primary); }
  .vault-name { font-weight: 600; font-size: 1rem; }
  .vault-path { font-size: 0.85rem; color: var(--text-secondary); margin: 0.25rem 0; }
  .vault-time { font-size: 0.75rem; color: var(--text-secondary); opacity: 0.7; }
  .remove-btn {
    background: var(--bg-secondary); border: 1px solid var(--border-color-medium);
    color: var(--text-secondary); border-radius: 50%; width: 32px; height: 32px;
    font-size: 1.2rem; cursor: pointer; transition: all 0.2s ease;
    display: flex; align-items: center; justify-content: center;
  }
  .remove-btn:hover { background: var(--error-color); color: white; border-color: var(--error-color); }
  .empty-state { color: var(--text-secondary); text-align: center; padding: 2rem 0; }
  .modal-actions { display: flex; justify-content: flex-end; gap: 1rem; margin-top: 1.5rem; }
  .modal-actions button {
    padding: 0.6rem 1.2rem; border-radius: 6px; cursor: pointer;
    background: var(--bg-secondary); color: var(--text-primary);
    border: 1px solid var(--border-color-medium); transition: all 0.2s ease;
  }
  .modal-actions .browse-btn {
    background: var(--accent-primary); color: white; border-color: var(--accent-primary);
  }
  .modal-actions .browse-btn:hover { background: var(--accent-secondary); }
  .modal-actions button:not(.browse-btn):hover { background: var(--bg-hover-medium); }
</style>
