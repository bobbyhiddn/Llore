<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import logo from '../../assets/images/logo.png';

  export let isLoading: boolean = false;
  export let initialErrorMsg: string = '';

  const dispatch = createEventDispatcher();

  function handleLoadLore() {
    dispatch('loadlore');
  }

  function handleNewLore() {
    dispatch('newlore');
  }

  function handleRecentVaults() {
    dispatch('recentvaults');
  }
</script>

<div class="initial-prompt">
  <img src={logo} alt="Llore Logo" class="logo logo-large" />
  <h2>Select or Create a Vault</h2>
  <p>Load an existing Lore Vault or create a new one.</p>
  {#if initialErrorMsg}
    <p class="error-message">{initialErrorMsg}</p>
  {/if}
  <button on:click={handleLoadLore} disabled={isLoading}>
      {#if isLoading}Loading...{:else}Load Lore Vault{/if}
  </button>
  <button on:click={handleRecentVaults} disabled={isLoading} class="recent-vaults-btn">
      Recent Vaults
  </button>
  <button on:click={handleNewLore} disabled={isLoading}>
      {#if isLoading}Creating...{:else}Create New Vault{/if}
  </button>
</div>

<style>
  .initial-prompt {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    padding: 2rem;
    text-align: center;
  }

  .logo-large {
    width: 300px;
    margin-bottom: 3rem;
  }

  h2 {
    margin-bottom: 0.5rem;
    color: var(--text-primary);
  }

  p {
    margin-bottom: 2rem;
    color: var(--text-secondary);
  }

  button {
    margin: 0.5rem;
    min-width: 150px;
  }

  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    border: 1px solid rgba(255, 71, 87, 0.2);
  }

  /* Import relevant button styles if needed, or rely on global */
  button {
    padding: 0.75rem 1.5rem;
    background: var(--accent-gradient);
    border: none;
    border-radius: 8px;
    color: white;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
    box-shadow: 0 4px 12px rgba(109, 94, 217, 0.2);
  }

  button:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(109, 94, 217, 0.3);
  }

  button:active {
    transform: translateY(0);
  }

  button:disabled {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-secondary);
    cursor: not-allowed;
    transform: none;
    box-shadow: none;
  }

  .recent-vaults-btn {
    background: var(--accent-gradient);
    color: white;
    border: none;
    box-shadow: 0 4px 12px rgba(109, 94, 217, 0.2);
  }

  .recent-vaults-btn:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(109, 94, 217, 0.3);
  }
</style>