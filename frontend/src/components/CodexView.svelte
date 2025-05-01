<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { database } from '@wailsjs/go/models'; // Import namespace

  // Props from parent (App.svelte)
  export let entries: database.CodexEntry[] = [];
  export let currentEntry: database.CodexEntry | null = null;
  export let isLoading: boolean = false;
  export let isEditing: boolean = false;
  export let isGenerating: boolean = false;
  export let errorMsg: string = '';
  export let selectedModel: string = ''; // Needed for generate content

  // Local state for the form binding if needed, or bind directly to prop
  let localCurrentEntry: database.CodexEntry = { id: 0, name: '', type: '', content: '', createdAt: '', updatedAt: '' };

  $: if (currentEntry) {
      // Deep copy to avoid modifying the prop directly until save
      localCurrentEntry = JSON.parse(JSON.stringify(currentEntry));
  } else {
      // Reset local state if currentEntry becomes null
      localCurrentEntry = { id: 0, name: '', type: '', content: '', createdAt: '', updatedAt: '' };
  }

  const dispatch = createEventDispatcher();

  function handleEntrySelect(entry: database.CodexEntry) {
    dispatch('selectentry', entry);
  }

  function prepareNewEntry() {
    dispatch('newentry');
    // Optionally focus the name input after dispatch
    // setTimeout(() => document.getElementById('codex-name-input')?.focus(), 0);
  }

  function handleSaveEntry() {
    // Validate local state before dispatching
    if (!localCurrentEntry.name) {
        dispatch('error', 'Entry must have a name.');
        return;
    }
    if (isEditing && (typeof localCurrentEntry.id !== 'number' || localCurrentEntry.id <= 0)) {
        dispatch('error', 'Cannot update: Invalid entry ID.');
        return;
    }
    dispatch('saveentry', { entryData: localCurrentEntry, isEditing });
  }

  function handleDeleteEntry() {
    if (!localCurrentEntry || typeof localCurrentEntry.id !== 'number' || localCurrentEntry.id <= 0) {
      dispatch('error', 'No valid entry selected for deletion.');
      return;
    }
    if (confirm(`Are you sure you want to delete '${localCurrentEntry.name}'?`)) {
      dispatch('deleteentry', localCurrentEntry.id);
    }
  }

  async function handleGenerateContent() {
    if (!localCurrentEntry || !localCurrentEntry.name) {
        dispatch('error', 'Please provide a name before generating content.');
        return;
    }
    if (!selectedModel) {
        dispatch('error', 'Please select an AI model from the settings first.');
        return;
    }
    // Dispatch event with necessary info for App.svelte to call backend
    dispatch('generatecontent', { entryData: localCurrentEntry, model: selectedModel });
  }

  function goBack() {
    dispatch('back');
  }

</script>

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>

<div class="codex-view">
  <div class="entries-list">
    <button class="new-entry-btn" on:click={prepareNewEntry}>
      + New Entry
    </button>
    {#if entries.length === 0 && !isLoading}
      <p class="empty-state">No entries yet. Create your first one!</p>
    {:else if isLoading}
       <p>Loading entries...</p>
    {:else}
      {#each entries as entry (entry.id)}
        <button
          class="entry-item"
          class:active={currentEntry && currentEntry.id === entry.id}
          on:click={() => handleEntrySelect(entry)}
        >
          {entry.name} ({entry.type})
        </button>
      {/each}
    {/if}
  </div>

  <div class="codex-entry">
    {#if isEditing || !currentEntry} <!-- Show form for new or editing -->
      <form on:submit|preventDefault={handleSaveEntry}>
        <h2>{isEditing ? `Edit Entry: ${localCurrentEntry.name}` : 'Create New Entry'}</h2>
        <div class="codex-entry-content">
          <div class="codex-entry-field">
            <label for="name">Name:</label>
            <input
              type="text"
              id="name"
              bind:value={localCurrentEntry.name}
              placeholder="Entry name"
              required
            />
          </div>

          <div class="codex-entry-field">
            <label for="type">Type:</label>
            <input
              type="text"
              id="type"
              bind:value={localCurrentEntry.type}
              placeholder="Entry type (e.g., Character, Location)"
            />
          </div>

          <div class="codex-entry-field">
            <label for="content">Content:</label>
            <textarea
              id="content"
              bind:value={localCurrentEntry.content}
              placeholder="Describe the entry..."
            />
          </div>
        </div>

        <div class="button-group">
          <button type="submit" class="save-btn" disabled={isLoading || isGenerating}>
            {#if isLoading}Saving...{:else}{isEditing ? 'Save Changes' : 'Create Entry'}{/if}
          </button>

          <button
            type="button"
            class="generate-btn"
            disabled={isLoading || isGenerating}
            on:click={handleGenerateContent}
          >
            {#if isGenerating}Generating...{:else}Generate Content (AI){/if}
          </button>

          {#if isEditing}
            <button
              type="button"
              class="delete-btn"
              disabled={isLoading || isGenerating}
              on:click={handleDeleteEntry}
            >
              Delete Entry
            </button>
          {/if}
        </div>

        {#if errorMsg}
          <p class="error-message">{errorMsg}</p>
        {/if}
      </form>
    {:else}
      <div class="empty-state">
        <p>Select an entry to edit or create a new one</p>
      </div>
    {/if}
  </div>
</div>

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
    z-index: 10; /* Ensure it's above other content */
  }

  .back-btn:hover {
    color: var(--text-primary);
  }

  .codex-view {
    display: flex;
    gap: 1rem; /* Reduced gap */
    flex: 1;
    overflow: hidden;
    background: var(--bg-primary);
    border-radius: 12px;
    padding: 1rem;
    height: calc(100vh - 6rem); /* Adjust based on header/back button */
    margin-top: 3rem; /* Space for back button */
  }

  .entries-list {
    width: 250px; /* Slightly narrower */
    padding: 1rem;
    background: rgba(255, 255, 255, 0.03);
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
    gap: 0.5rem; /* Reduced gap */
    overflow-y: auto;
    flex-shrink: 0;
  }

  .codex-entry {
    flex: 1;
    padding: 1.5rem;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    overflow-y: auto; /* Allow form content to scroll if needed */
    display: flex; /* Use flex for form layout */
    flex-direction: column;
  }

  .codex-entry form {
    display: flex;
    flex-direction: column;
    gap: 1rem; /* Reduced gap */
    flex: 1; /* Allow form to take up space */
    min-height: 0; /* Needed for flex child scrolling */
  }

  .codex-entry-content {
    display: flex;
    flex-direction: column;
    gap: 1rem; /* Reduced gap */
    overflow-y: auto; /* Allow fields to scroll if form is too tall */
    flex-grow: 1; /* Allow content area to expand */
    padding-right: 0.5rem; /* Space for scrollbar */
  }

  .codex-entry-field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .codex-entry-field label {
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .codex-entry-field input,
  .codex-entry-field textarea {
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    padding: 0.75rem;
    color: var(--text-primary);
    font-size: 1rem;
  }

  .codex-entry-field textarea {
    min-height: 150px; /* Reduced min-height */
    resize: vertical;
    flex-grow: 1; /* Allow textarea to grow */
  }

  .button-group {
    display: flex;
    flex-wrap: wrap; /* Allow buttons to wrap on smaller screens */
    gap: 0.75rem; /* Reduced gap */
    margin-top: 1rem; /* Space above buttons */
    flex-shrink: 0; /* Prevent button group from shrinking */
  }

  /* Use specific classes for buttons */
  .save-btn,
  .delete-btn,
  .generate-btn,
  .new-entry-btn {
    padding: 0.6rem 1.2rem; /* Slightly smaller padding */
    border-radius: 6px;
    font-weight: 500;
    transition: all 0.3s ease;
    border: none;
    cursor: pointer;
  }

  .save-btn {
    background: var(--accent-primary);
    color: white;
  }
  .save-btn:hover:not(:disabled) { background: var(--accent-secondary); }

  .delete-btn {
    background: var(--error-color);
    color: white;
  }
  .delete-btn:hover:not(:disabled) { background: #d63031; } /* Darker red */

  .generate-btn {
    background: #0984e3; /* Blue for generate */
    color: white;
  }
   .generate-btn:hover:not(:disabled) { background: #74b9ff; } /* Lighter blue */

  .new-entry-btn {
    background: var(--success-color); /* Green for new */
    color: white;
    width: 100%;
    margin-bottom: 0.5rem;
  }
  .new-entry-btn:hover:not(:disabled) { background: #00b894; } /* Darker green */

  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    font-size: 1.1rem;
    text-align: center;
    padding: 2rem;
  }

  .entry-item {
    width: 100%;
    text-align: left;
    padding: 0.6rem 1rem; /* Slightly smaller padding */
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text-primary);
    transition: all 0.3s ease;
    cursor: pointer;
    border: none; /* Remove default button border */
  }

  .entry-item:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .entry-item.active {
    background: var(--accent-primary);
    color: white;
    font-weight: bold;
  }

  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    border: 1px solid rgba(255, 71, 87, 0.2);
    flex-shrink: 0; /* Prevent error message from shrinking */
  }

  /* Scrollbar */
  ::-webkit-scrollbar {
    width: 6px;
    height: 6px;
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