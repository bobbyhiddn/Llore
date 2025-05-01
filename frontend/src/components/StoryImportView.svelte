<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { database } from '@wailsjs/go/models'; // Import namespace

  // Props
  export let isProcessingStory: boolean = false;
  export let isProcessingImport: boolean = false;
  export let processStoryErrorMsg: string = '';
  export let importError: string = '';
  export let vaultIsReady: boolean = false;

  // Local State for this view
  let storyText = '';
  let isDraggingFile = false;
  let importedFileName = '';
  let importedContent = '';
  let showImportModal = false; // Modal for file preview before processing
  let showExistingEntriesModal = false; // Modal for confirming overwrite/update
  let existingEntries: database.CodexEntry[] = []; // Entries found during import that already exist
  let processStorySuccessMsg = ''; // Message shown after successful processing (maybe in a toast later)
  let processedEntries: database.CodexEntry[] = []; // Entries created/updated by the last import
  let createdEntriesCount = 0; // Count of *new* entries from last import

  const dispatch = createEventDispatcher();

  function goBack() {
    dispatch('back');
  }

  // Handle file drop for story import
  function handleFileDrop(event: DragEvent) {
    event.preventDefault();
    isDraggingFile = false;
    importError = '';

    const files = event.dataTransfer?.files;
    if (!files || files.length === 0) return;

    const file = files[0];
    if (!file.name.match(/\.(txt|md)$/i)) {
      importError = 'Please drop a .txt or .md file';
      return;
    }

    importedFileName = file.name;
    const reader = new FileReader();
    reader.onload = async (e) => {
      const content = e.target?.result as string;
      if (content) {
        importedContent = content;
        showImportModal = true; // Show preview modal
      }
    };
    reader.readAsText(file);
  }

  // Handle manual file selection
  async function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;

    const file = input.files[0];
    if (!file.name.match(/\.(txt|md)$/i)) {
      importError = 'Please select a .txt or .md file';
      return;
    }

    importedFileName = file.name;
    const reader = new FileReader();
    reader.onload = async (e) => {
      const content = e.target?.result as string;
      if (content) {
        importedContent = content;
        showImportModal = true; // Show preview modal
      }
    };
    reader.readAsText(file);
    // Reset file input value to allow selecting the same file again
    input.value = '';
  }

  // Process the imported story (from modal)
  async function processImportedStory(forceReimport = false) {
    if (!importedContent) return;

    // Reset local states for processing
    isProcessingImport = true; // Use the specific import processing flag
    importError = '';
    processStorySuccessMsg = '';
    showExistingEntriesModal = false; // Ensure this is closed

    dispatch('processimport', {
        content: importedContent,
        filename: importedFileName, // Pass filename for saving to library
        force: forceReimport
    });
    // App.svelte will handle the backend call and update props or dispatch success/error events
  }

  // Handle the import button click (for text area content)
  function handleImportStoryText() {
    if (!storyText.trim()) {
      dispatch('error', 'Please paste the story text into the textarea.');
      return;
    }
    if (!vaultIsReady) {
      dispatch('error', 'No Lore Vault is currently loaded.');
      return;
    }

    // Reset local states for processing
    isProcessingStory = true; // Use the specific text processing flag
    processStoryErrorMsg = '';
    processStorySuccessMsg = '';

    dispatch('importstorytext', { content: storyText });
    // App.svelte will handle the backend call and update props or dispatch success/error events
  }

  // Called by App.svelte via dispatch or prop update when import finds existing entries
  export function showExistingConfirmation(foundExisting: database.CodexEntry[]) {
      existingEntries = foundExisting;
      showExistingEntriesModal = true;
      isProcessingImport = false; // Stop loading indicator
  }

  // Called by App.svelte via dispatch or prop update on successful import/processing
  export function showImportSuccess(result: { entries: database.CodexEntry[], newCount: number, updatedCount: number }) {
      processedEntries = result.entries;
      createdEntriesCount = result.newCount;
      if (result.updatedCount > 0) {
          processStorySuccessMsg = `Story Processed\n${result.updatedCount} entries were updated and ${result.newCount} new entries were created.`;
      } else if (result.newCount > 0) {
          processStorySuccessMsg = `Story Processed\n${result.newCount} new codex entries were created.`;
      } else {
          processStorySuccessMsg = 'No codex entries could be extracted or updated from the story.';
      }
      // Reset state after success
      showImportModal = false;
      showExistingEntriesModal = false;
      importedContent = '';
      importedFileName = '';
      storyText = ''; // Clear text area too
      isProcessingImport = false;
      isProcessingStory = false;
      // Maybe show a success toast/message briefly instead of relying on modal?
      // For now, we can just log it or rely on App.svelte to show feedback
      console.log(processStorySuccessMsg);
      dispatch('importsuccess', { message: processStorySuccessMsg, entries: processedEntries });
  }

  // Called by App.svelte on error
  export function setProcessingError(message: string, type: 'import' | 'story') {
      if (type === 'import') {
          importError = message;
          isProcessingImport = false;
      } else {
          processStoryErrorMsg = message;
          isProcessingStory = false;
      }
  }

  function cancelImport() {
      showImportModal = false;
      importedContent = '';
      importedFileName = '';
      importError = '';
  }

  function cancelExistingOverwrite() {
      showExistingEntriesModal = false;
      // Also clear the import modal state if the user cancels here
      cancelImport();
  }

</script>

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>
<section class="story-processor">
  <h2>Import New Story</h2>
  <p>Paste story text or drag & drop a file below. Files will be saved to the Library and processed for codex entries.</p>

  <!-- Text Input -->
  <div class="text-input-section">
    <div class="text-input-container">
      <textarea
        bind:value={storyText}
        class="story-input"
        placeholder="Paste your story text here..."
        disabled={isProcessingStory || isProcessingImport}
      ></textarea>
    </div>
  </div>

  <!-- Drop Zone and Import Button -->
  <div class="drop-zone-section">
    <div class="drop-zone-container">
      <div class="drop-zone {isDraggingFile ? 'dragging' : ''}"
        role="button"
        tabindex="0"
        aria-label="Drop zone for importing story files"
        on:dragenter|preventDefault={(e) => { isDraggingFile = true; }}
        on:dragover|preventDefault={(e) => { isDraggingFile = true; }}
        on:dragleave={() => isDraggingFile = false}
        on:drop={handleFileDrop}
        on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') document.getElementById('file-input')?.click(); }}
      >
        <span class="icon">📚</span>
        <span class="label">Drop Story File Here</span>
        <span class="description">or</span>
        <input
          type="file"
          accept=".txt,.md"
          on:change={handleFileSelect}
          style="display: none"
          id="file-input"
          disabled={isProcessingStory || isProcessingImport}
        />
        <button class="browse-btn" on:click={() => document.getElementById('file-input')?.click()} disabled={isProcessingStory || isProcessingImport}>
          Browse Files
        </button>
        {#if importError && !showImportModal} <!-- Show drop zone error only if modal isn't open -->
          <p class="error-message">{importError}</p>
        {/if}
      </div>
      <button
        class="import-btn"
        on:click={handleImportStoryText}
        disabled={isProcessingStory || isProcessingImport || !storyText.trim()}
      >
        {#if isProcessingStory}Processing Text...{:else}Import Text & Add Entries{/if}
      </button>
      {#if processStoryErrorMsg}
        <p class="error-message">{processStoryErrorMsg}</p>
      {/if}
    </div>
  </div>
</section>

<!-- Import Preview Modal -->
{#if showImportModal}
  <div class="modal-backdrop">
    <div class="modal import-modal">
      <h3>Import Story File</h3>
      <div class="import-preview">
        <p class="filename">File: {importedFileName}</p>
        <div class="content-preview">
          {importedContent.slice(0, 500)}{importedContent.length > 500 ? '...' : ''}
        </div>
      </div>
      {#if importError}
        <p class="error-message">{importError}</p>
      {/if}
      <div class="modal-actions">
        <button on:click={() => processImportedStory(false)} disabled={isProcessingImport}>
          {#if isProcessingImport}Processing...{:else}Process Story{/if}
        </button>
        <button on:click={cancelImport} disabled={isProcessingImport}>
          Cancel
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Existing Entries Confirmation Modal -->
{#if showExistingEntriesModal}
  <div class="modal-backdrop">
    <div class="modal existing-entries-modal">
      <h3>Existing Entries Found</h3>
      <p>The following entries seem to already exist from this story:</p>
      <div class="existing-entries-list">
        {#each existingEntries as entry}
          <div class="existing-entry">
            <strong>{entry.name}</strong> ({entry.type})
            <p class="entry-preview">{entry.content.substring(0, 100)}{entry.content.length > 100 ? '...' : ''}</p>
          </div>
        {/each}
      </div>
      <p>Would you like to update these entries with the content from the file?</p>
      {#if importError}
        <p class="error-message">{importError}</p>
      {/if}
      <div class="modal-actions">
         <button on:click={cancelExistingOverwrite} disabled={isProcessingImport}>Cancel</button>
         <button
           class="primary"
           on:click={() => processImportedStory(true)}
           disabled={isProcessingImport}
         >
           {#if isProcessingImport}
             Updating...
           {:else}
             Update Entries
           {/if}
         </button>
      </div>
    </div>
  </div>
{/if}


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

  .story-processor {
    max-width: 1000px; /* Slightly wider */
    margin: 0 auto;
    padding: 2rem;
    padding-top: 4rem; /* Space for back button */
    height: calc(100vh - 4rem); /* Adjust if header exists */
    display: flex;
    flex-direction: column;
  }

  h2 {
      margin-bottom: 0.5rem;
      color: var(--text-primary);
  }
  p {
      margin-bottom: 1.5rem;
      color: var(--text-secondary);
  }

  .text-input-section {
    flex: 1; /* Grow to fill space */
    display: flex;
    flex-direction: column;
    min-height: 0; /* Crucial for flex child scrolling */
    margin-bottom: 1rem;
  }

  .text-input-container {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    height: 100%; /* Fill the section */
  }

  .story-input {
    flex: 1; /* Grow within its container */
    min-height: 150px; /* Minimum size */
    resize: none; /* Disable manual resize */
    font-family: monospace;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid rgba(255, 255, 255, 0.1);
    padding: 1rem;
    border-radius: 4px;
    line-height: 1.5;
  }
   .story-input:focus {
    outline: none;
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 3px rgba(109, 94, 217, 0.3);
  }

  .drop-zone-section {
    /* margin-top: 1rem; */
    padding-bottom: 1rem;
    flex-shrink: 0; /* Prevent shrinking */
  }

  .drop-zone-container {
    display: flex;
    gap: 1rem;
    align-items: stretch; /* Make items same height */
  }

  .drop-zone {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    border: 2px dashed var(--accent-primary);
    border-radius: 8px;
    background: var(--bg-secondary);
    transition: all 0.3s ease;
    cursor: pointer;
    text-align: center;
    min-height: 120px; /* Ensure minimum height */
  }

  .drop-zone.dragging {
    border-color: var(--accent-secondary);
    background: rgba(109, 94, 217, 0.1); /* Use accent color */
    transform: scale(1.02);
  }

  .drop-zone .icon {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
  }

  .drop-zone .label {
    font-size: 1.1rem;
    font-weight: bold;
    margin-bottom: 0.25rem;
    color: var(--text-primary);
  }

  .drop-zone .description {
    color: var(--text-secondary);
    margin-bottom: 0.5rem;
    font-size: 0.9rem;
  }

  .browse-btn {
    background: var(--accent-primary);
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.3s ease;
    font-size: 0.9rem;
    margin-top: 0.5rem;
  }

  .browse-btn:hover:not(:disabled) {
    background: var(--accent-secondary);
  }

  .import-btn {
    background: var(--accent-primary);
    color: white;
    border: none;
    padding: 0.75rem 1.5rem;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.3s ease;
    font-size: 1rem;
    white-space: nowrap;
    /* height: 100%; */ /* Removed height 100% */
    min-height: 120px; /* Match drop zone height */
    display: flex;
    align-items: center;
    justify-content: center; /* Center text */
    text-align: center;
  }

  .import-btn:hover:not(:disabled) {
    background: var(--accent-secondary);
  }

  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  /* Modals */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.8);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    overflow-y: auto;
    padding: 2rem;
  }

  .modal {
    background: var(--bg-primary);
    color: var(--text-primary);
    border-radius: 12px;
    padding: 2rem;
    width: 100%;
    max-width: 600px; /* Consistent max width */
    margin: auto;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.1);
    position: relative;
  }

  .modal h3 {
      margin-top: 0;
      margin-bottom: 1.5rem;
      color: var(--accent-primary);
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 1.5rem;
  }

  .modal-actions button {
      padding: 0.6rem 1.2rem;
  }
  .modal-actions button.primary {
      background: var(--accent-primary);
  }
   .modal-actions button.primary:hover:not(:disabled) {
      background: var(--accent-secondary);
  }

  .import-preview {
    margin: 1rem 0;
    padding: 1rem;
    background: rgba(0,0,0,0.2); /* Darker background for preview */
    border-radius: 4px;
  }

  .import-preview .filename {
    font-weight: bold;
    margin-bottom: 0.5rem;
    color: var(--accent-secondary); /* Use secondary accent */
    word-break: break-all;
  }

  .content-preview {
    font-family: monospace;
    white-space: pre-wrap;
    max-height: 250px; /* Limit height */
    overflow-y: auto;
    padding: 1rem;
    background: var(--bg-secondary);
    border-radius: 4px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    font-size: 0.9rem;
  }

  .existing-entries-modal {
      max-width: 700px; /* Wider for list */
  }

  .existing-entries-list {
    max-height: 300px;
    overflow-y: auto;
    margin: 1rem 0;
    padding: 1rem;
    background: var(--bg-secondary);
    border-radius: 4px;
    border: 1px solid rgba(255,255,255,0.1);
  }

  .existing-entry {
    margin-bottom: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }
  .existing-entry:last-child {
    margin-bottom: 0;
    padding-bottom: 0;
    border-bottom: none;
  }
  .existing-entry strong {
      color: var(--accent-primary);
  }

  .entry-preview {
    margin-top: 0.5rem;
    color: var(--text-secondary);
    font-size: 0.9rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    border: 1px solid rgba(255, 71, 87, 0.2);
    font-size: 0.9rem;
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