<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  // Props
  export let libraryFiles: string[] = [];
  export let isLibraryLoading: boolean = false;
  export let errorMsg: string = '';

  const dispatch = createEventDispatcher();

  function goBack() {
    dispatch('back');
  }

  function refreshLibrary() {
    dispatch('refresh');
  }

  function viewFile(filename: string) {
    dispatch('viewfile', filename);
  }
</script>

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>
<section class="library-view">
  <h2>Library</h2>
  <button on:click={refreshLibrary} disabled={isLibraryLoading}>
    {#if isLibraryLoading}Refreshing...{:else}Refresh Library{/if}
  </button>
  {#if isLibraryLoading && libraryFiles.length === 0} <!-- Show loading only if list is empty initially -->
    <p>Loading library files...</p>
  {:else if errorMsg}
    <p class="error-message">{errorMsg}</p>
  {:else}
    {#if libraryFiles.length === 0}
      <p class="empty-state">No files in library. Add some via Story Import or Write mode.</p>
    {:else}
      <ul class="file-list">
        {#each libraryFiles as filename (filename)}
          <li class="file-item">
            <span class="filename">{filename}</span>
            <button class="view-btn" on:click={() => viewFile(filename)} title="View/Edit {filename}">View/Edit</button>
          </li>
        {/each}
      </ul>
    {/if} <!-- End check file list -->
  {/if} <!-- End check error/loading -->
</section>

<style>
  .back-btn {
    position: absolute;
    top: 1rem;
    left: 1rem;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    padding: 0.5rem;
    cursor: pointer;
    transition: color 0.3s ease;
    z-index: 10;
  }

  .back-btn:hover {
    color: var(--text-primary);
  }

  .library-view {
    padding: 2rem;
    padding-top: 4rem; /* Space for back button */
    height: calc(100vh - 4rem); /* Adjust if header exists */
    overflow-y: auto;
    max-width: 900px;
    margin: 0 auto;
  }

  h2 {
    margin-bottom: 1rem;
    color: var(--text-primary);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    padding-bottom: 0.5rem;
  }

  button { /* General button style for refresh */
    margin-bottom: 1.5rem;
    padding: 0.6rem 1.2rem;
    background: var(--accent-primary);
    color: white;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition: background 0.3s ease;
  }
  button:hover:not(:disabled) {
    background: var(--accent-secondary);
  }
  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .empty-state {
    color: var(--text-secondary);
    margin-top: 2rem;
    text-align: center;
  }

  .file-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .file-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 1rem;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 6px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    transition: background 0.3s ease;
  }

  .file-item:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .filename {
    color: var(--text-primary);
    word-break: break-all; /* Prevent long names from overflowing */
    margin-right: 1rem;
  }

  .view-btn {
    padding: 0.4rem 0.8rem;
    font-size: 0.9rem;
    background: #0984e3; /* Blue */
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.3s ease;
    flex-shrink: 0; /* Prevent button from shrinking */
  }

  .view-btn:hover {
    background: #74b9ff; /* Lighter blue */
  }

  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    border: 1px solid rgba(255, 71, 87, 0.2);
  }

  /* Scrollbar */
  ::-webkit-scrollbar {
    width: 6px;
  }
  ::-webkit-scrollbar-track {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 3px;
  }
  ::-webkit-scrollbar-thumb {
    background: var(--accent-primary);
    border-radius: 3px;
  }
  ::-webkit-scrollbar-thumb:hover {
    background: var(--accent-secondary);
  }
</style>