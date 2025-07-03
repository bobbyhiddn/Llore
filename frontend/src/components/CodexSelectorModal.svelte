<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import type { database } from '@wailsjs/go/models';

  export let allEntries: database.CodexEntry[];
  export let nodeType: string;

  let selectedEntries: database.CodexEntry[] = [];
  let searchTerm = '';
  let selectedLength = 'medium'; // Default to medium
  const dispatch = createEventDispatcher();

  const lengthOptions = [
    { value: 'small', label: 'Small', description: '1 sentence' },
    { value: 'medium', label: 'Medium', description: '1 paragraph' },
    { value: 'large', label: 'Large', description: '1 page' },
    { value: 'extra-large', label: 'Extra Large', description: '2 pages' }
  ];

  function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      dispatch('close');
    }
  }

  // Focus the search input on mount
  let searchInput: HTMLInputElement;
  onMount(() => {
    searchInput?.focus();
  });

  $: filteredEntries = searchTerm
    ? allEntries.filter(e => e.name.toLowerCase().includes(searchTerm.toLowerCase()))
    : allEntries;

  function toggleSelection(entry: database.CodexEntry) {
    const index = selectedEntries.findIndex(e => e.id === entry.id);
    if (index > -1) {
      selectedEntries.splice(index, 1);
    } else {
      selectedEntries = [...selectedEntries, entry];
    }
    selectedEntries = selectedEntries; // Force Svelte reactivity
  }

  function handleWeave() {
    dispatch('weave', { selectedEntries, selectedLength });
  }
</script>

<div class="modal-backdrop" role="button" tabindex="-1" on:click={() => dispatch('close')} on:keydown={handleKeyDown}>
  <div class="modal codex-selector-modal" role="dialog" aria-modal="true" on:click|stopPropagation>
    <h3>Attach Codex Entries for '{nodeType}'</h3>
    <p>Select entries to provide specific context for the AI. This is optional.</p>
        <input type="search" bind:this={searchInput} bind:value={searchTerm} placeholder="Search entries..."/>
    
    <!-- Length Selector -->
    <div class="length-selector">
      <label for="length-select">Response Length:</label>
      <select id="length-select" bind:value={selectedLength}>
        {#each lengthOptions as option}
          <option value={option.value}>{option.label} ({option.description})</option>
        {/each}
      </select>
    </div>
    <div class="entry-list" role="listbox">
      {#each filteredEntries as entry (entry.id)}
        <button 
          type="button"
          class="entry-item" 
          role="option"
          aria-selected={selectedEntries.some(e => e.id === entry.id)}
          class:selected={selectedEntries.some(e => e.id === entry.id)}
          on:click={() => toggleSelection(entry)}
        >
          {entry.name} ({entry.type})
        </button>
      {/each}
    </div>
    <div class="modal-actions">
      <button on:click={() => dispatch('close')}>Cancel</button>
      <button class="primary" on:click={handleWeave}>
        Weave with {selectedEntries.length} entries
      </button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: var(--bg-primary);
    border: 1px solid var(--border-color-medium);
    border-radius: 8px;
    padding: 1.5rem;
    min-width: 450px;
    max-width: 550px;
    display: flex;
    flex-direction: column;
  }

  h3 {
    margin: 0 0 0.5rem 0;
    color: var(--text-primary);
  }

  p {
    margin: 0 0 1rem 0;
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  input[type="search"] {
    padding: 0.5rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color-medium);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 1rem;
    margin-bottom: 1rem;
  }

  .length-selector {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 1rem;
  }

  .length-selector label {
    color: var(--text-primary);
    font-weight: 500;
    font-size: 0.9rem;
  }

  .length-selector select {
    padding: 0.5rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color-medium);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 1rem;
    cursor: pointer;
  }

  .length-selector select:focus {
    outline: none;
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 2px rgba(109, 94, 217, 0.2);
  }

  .entry-list {
    flex-grow: 1;
    max-height: 300px;
    overflow-y: auto;
    border: 1px solid var(--border-color-medium);
    border-radius: 6px;
    margin: 0;
  }

  .entry-item {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--border-color-light);
    border-radius: 4px;
    cursor: pointer;
    transition: all 0.2s ease;
    background: var(--bg-primary);
    color: var(--text-primary);
    text-align: left;
    font-family: inherit;
    font-size: inherit;
  }

  .entry-item:hover {
    background: var(--bg-secondary);
    border-color: var(--border-color-medium);
  }

  .entry-item:focus {
    outline: none;
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 2px rgba(109, 94, 217, 0.2);
  }

  .entry-item.selected {
    background: var(--accent-primary);
    color: white;
    border-color: var(--accent-primary);
  }

  .modal-actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
    margin-top: 1.5rem;
  }

  .modal-actions button {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
    transition: background-color 0.2s;
  }

  .modal-actions button.primary {
    background: var(--accent-primary);
    color: white;
  }

  .modal-actions button.primary:hover {
    background: var(--accent-secondary);
  }

  .modal-actions button:not(.primary) {
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color-medium);
  }

  .modal-actions button:not(.primary):hover {
    background: var(--bg-hover-medium);
  }
</style>