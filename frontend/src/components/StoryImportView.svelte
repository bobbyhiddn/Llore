<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { database } from '@wailsjs/go/models'; // Import namespace
  import StoryImportStatus from './StoryImportStatus.svelte'; // Import the status component
  import type { Writable } from 'svelte/store'; // For type hinting if needed later

  // Type for the status prop of StoryImportStatus (sync with StoryImportStatus.svelte)
  type ImportStatus = 'idle' | 'sending' | 'receiving' | 'parsing' | 'checking_existing' | 'updating' | 'embedding' | 'complete' | 'error';

  // Props
  export let isProcessingStory: boolean = false; // Keep for text area processing? Or merge logic? For now, keep separate.
  // export let isProcessingImport: boolean = false; // Replaced by importStatus
  // export let processStoryErrorMsg: string = ''; // Replaced by importStatusError for file import
  // export let importError: string = ''; // Replaced by importStatusError
  export let vaultIsReady: boolean = false;

  // --- State for StoryImportStatus ---
  let importStatus: ImportStatus = 'idle';
  let importStatusError: string | null = null;
  let importNewEntries: database.CodexEntry[] = [];
  let importUpdatedEntries: database.CodexEntry[] = [];
  // --- End State for StoryImportStatus ---

  // Local State for this view (related to UI interaction)
  let storyText = '';
  let isDraggingFile = false;
  let importedFileName = '';
  let importedContent = '';
  let showImportModal = false; // Modal for file preview before processing
  let showExistingEntriesModal = false; // Modal for confirming overwrite/update
  let existingEntries: database.CodexEntry[] = []; // Entries found during import that already exist (for modal)
  // let processStorySuccessMsg = ''; // Replaced by StoryImportStatus display
  // let processedEntries: database.CodexEntry[] = []; // Replaced by importNewEntries/importUpdatedEntries
  // let createdEntriesCount = 0; // Can derive from importNewEntries.length
  // let showResultFeedback = false; // Handled by StoryImportStatus visibility
  // let lastImportWasSuccess = false; // Handled by importStatus === 'complete' or 'error'

  const dispatch = createEventDispatcher();

  function goBack() {
    dispatch('back');
  }

  // Handle file drop for story import
  function handleFileDrop(event: DragEvent) {
    event.preventDefault();
    isDraggingFile = false;
    // importError = ''; // Reset via status change
    importStatus = 'idle'; // Reset status on new drop
    importStatusError = null;

    const files = event.dataTransfer?.files;
    if (!files || files.length === 0) return;

    const file = files[0];
    if (!file.name.match(/\.(txt|md)$/i)) {
      // importError = 'Please drop a .txt or .md file';
      importStatus = 'error';
      importStatusError = 'Please drop a .txt or .md file';
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
      // importError = 'Please select a .txt or .md file';
      importStatus = 'error';
      importStatusError = 'Please select a .txt or .md file';
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

    showImportModal = false; // Close the modal immediately

    // Reset status for processing
    importStatus = 'sending'; // Start the status tracking
    importStatusError = null;
    importNewEntries = [];
    importUpdatedEntries = [];
    showExistingEntriesModal = false; // Ensure this is closed

    // TODO: Update status progressively via events from App.svelte/backend
    // For now, we just set it to 'sending' and App.svelte will call
    // showExistingConfirmation, showImportSuccess, or setProcessingError

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

    // Reset local states for processing (Keep separate for now)
    isProcessingStory = true; // Use the specific text processing flag
    // processStoryErrorMsg = ''; // Keep separate for text area errors?
    // processStorySuccessMsg = ''; // Keep separate?

    // Set status to 'sending' when text import starts
    importStatus = 'sending'; // <-- Changed from 'idle'
    importStatusError = null;
    importNewEntries = [];
    importUpdatedEntries = [];

    dispatch('importstorytext', { content: storyText });
    // App.svelte will handle the backend call and update props or dispatch success/error events
  }

  // Called by App.svelte via dispatch or prop update when import finds existing entries
  export function showExistingConfirmation(foundExisting: database.CodexEntry[]) {
      existingEntries = foundExisting;
      importStatus = 'checking_existing'; // Update status
      showExistingEntriesModal = true;
      // isProcessingImport = false; // Status handles loading state
  }

  // Called by App.svelte via dispatch or prop update on successful import/processing
  export function showImportSuccess(result: { newEntries: database.CodexEntry[], updatedEntries: database.CodexEntry[] }) {
      importNewEntries = result.newEntries || [];
      importUpdatedEntries = result.updatedEntries || [];
      importStatus = 'complete';
      importStatusError = null;

      // Clear modal state
      showImportModal = false;
      importedContent = '';
      importedFileName = '';

      // Dispatch success event (App.svelte might listen to this)
      dispatch('importsuccess', {
          message: `Import complete: ${importNewEntries.length} new, ${importUpdatedEntries.length} updated.`,
          newEntries: importNewEntries,
          updatedEntries: importUpdatedEntries
      });

      // After successful import/update, automatically switch to codex view - REMOVED FOR USER FEEDBACK
      // setTimeout(() => dispatch('gotocodex'), 1500);
  }

  // Called by App.svelte to update status during processing
  export function updateImportStatus(newStatus: ImportStatus, message?: string) {
      importStatus = newStatus;
      if (message) {
          // Potentially display intermediate messages if needed, though status text might suffice
          console.log(`Import Status: ${newStatus} - ${message}`);
      }
      if (newStatus === 'error' && message) {
          importStatusError = message;
      }
  }


  // Called by App.svelte on error during file import
  export function setImportError(message: string) {
      importStatus = 'error';
      importStatusError = message;
      // Optionally, reset modal states
      showImportModal = false;
      importedContent = '';
      importedFileName = '';
  }

  // TODO: Handle errors specifically from the text area import separately if needed
  // export function setStoryProcessingError(message: string) {
  //     processStoryErrorMsg = message;
  //     isProcessingStory = false;
  // }


  function cancelImport() {
      showImportModal = false;
      importedContent = '';
      importedFileName = '';
      // importError = ''; // Reset via status
      importStatus = 'idle';
      importStatusError = null;
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
        disabled={isProcessingStory || importStatus !== 'idle'}
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
          disabled={isProcessingStory || importStatus !== 'idle'}
        />
        <button class="browse-btn" on:click={() => document.getElementById('file-input')?.click()} disabled={isProcessingStory || importStatus !== 'idle'}>
          Browse Files
        </button>
        <!-- Error message display moved to StoryImportStatus -->
        <!-- {#if importError && !showImportModal}
          <p class="error-message">{importError}</p>
        {/if} -->
      </div>
      <button
        class="import-btn"
        on:click={handleImportStoryText}
        disabled={isProcessingStory || importStatus !== 'idle' || !storyText.trim()}
      >
        {#if isProcessingStory}Processing Text...{:else}Import Text & Add Entries{/if}
      </button>
      <!-- Text area processing error message (keep separate for now?) -->
      <!-- {#if processStoryErrorMsg}
        <p class="error-message">{processStoryErrorMsg}</p>
      {/if} -->
      <!-- Feedback messages moved to StoryImportStatus -->
      <!-- {#if showResultFeedback && !isProcessingStory && !isProcessingImport} ... {/if} -->
    </div> <!-- End of drop-zone-container -->

    <!-- Instance removed from here -->

  </div> <!-- End of drop-zone-section -->

  <!-- Remove the conditional block and the extra instance below -->

  <!-- Add the single instance here, before the end of the section -->
  <StoryImportStatus
      status={importStatus}
      errorMsg={importStatusError}
      newEntries={importNewEntries}
      updatedEntries={importUpdatedEntries}
  />
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
      <!-- Error display moved to status component, but maybe keep one here for modal-specific issues? -->
      <!-- {#if importError}
        <p class="error-message">{importError}</p>
      {/if} -->
      <div class="modal-actions">
        <button on:click={() => processImportedStory(false)} disabled={importStatus !== 'idle' && importStatus !== 'error'}>
          {#if importStatus !== 'idle' && importStatus !== 'error'}Processing...{:else}Process Story{/if}
        </button>
        <button on:click={cancelImport} disabled={importStatus !== 'idle' && importStatus !== 'error'}>
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
      <!-- Error display moved to status component -->
      <!-- {#if importError}
        <p class="error-message">{importError}</p>
      {/if} -->
      <div class="modal-actions">
         <button on:click={cancelExistingOverwrite} disabled={importStatus === 'sending' || importStatus === 'receiving' || importStatus === 'parsing' || importStatus === 'updating'}>Cancel</button>
         <button
           class="primary"
           on:click={() => processImportedStory(true)}
           disabled={importStatus === 'sending' || importStatus === 'receiving' || importStatus === 'parsing' || importStatus === 'updating'}
         >
           {#if importStatus === 'updating'} <!-- Or maybe a generic 'processing' state? -->
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
    overflow-y: auto; /* Allow the main section to scroll if needed */
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
    /* flex: 1; */ /* Remove flex-grow: 1 */
    display: flex;
    flex-direction: column;
    min-height: 150px; /* Ensure minimum height */
    flex-shrink: 0; /* Prevent shrinking */
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

  /* .error-message class is now primarily used within StoryImportStatus */
  /* Keep it here if needed for other potential errors, but remove if unused */
  /* .error-message { ... } */

  /* Success message styling (if needed outside status component) */
  .success-message {
      color: var(--success-color);
      background: rgba(46, 204, 113, 0.1);
      padding: 0.75rem 1rem;
      border-radius: 8px;
      margin-top: 1rem;
      border: 1px solid rgba(46, 204, 113, 0.2);
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