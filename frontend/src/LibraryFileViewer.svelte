<!-- LibraryFileViewer.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { ProcessAndSaveTextAsEntries, SaveLibraryFile } from '@wailsjs/go/main/App'; // Import SaveLibraryFile (will add Go func later)

  export let filename: string;
  export let initialContent: string;

  let content: string = initialContent;
  let isLoading = false;
  let errorMsg = '';
  let successMsg = '';

  const dispatch = createEventDispatcher();

  async function handleSave() {
    isLoading = true;
    errorMsg = '';
    successMsg = '';
    try {
      await SaveLibraryFile(filename, content); // Call Go function to save
      successMsg = 'File saved successfully!';
    } catch (err) {
      console.error("Error saving file:", err);
      errorMsg = `Failed to save file: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  async function handleReprocess() {
     isLoading = true;
     errorMsg = '';
     successMsg = '';
     try {
       const count = await ProcessAndSaveTextAsEntries(content);
       successMsg = `Successfully processed and saved ${count} codex entries.`;
       // Optionally close the viewer after reprocessing?
       // dispatch('close'); 
     } catch (err) {
       console.error("Error reprocessing file:", err);
       errorMsg = `Failed to reprocess file: ${err}`;
     } finally {
       isLoading = false;
     }
  }

  function handleClose() {
    dispatch('close');
  }
</script>

<div class="modal-backdrop">
  <div class="modal-content library-viewer">
    <h3>Viewing/Editing: {filename}</h3>
    <textarea bind:value={content} rows="20" disabled={isLoading}></textarea>
    {#if errorMsg}<p class="error-message">{errorMsg}</p>{/if}
    {#if successMsg}<p style="color: green;">{successMsg}</p>{/if}
    <div class="modal-actions">
      <button on:click={handleSave} disabled={isLoading}>Save Changes</button>
      <button on:click={handleReprocess} disabled={isLoading}>Reprocess for Codex</button>
      <button on:click={handleClose} disabled={isLoading}>Close</button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    /* Styles from App.svelte */
    position: fixed; top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0,0,0,0.4); display: flex;
    align-items: center; justify-content: center; z-index: 1000;
  }
  .modal-content {
    /* Styles from App.svelte */
    background: white; color: #222; border-radius: 8px;
    padding: 2rem; box-shadow: 0 2px 16px rgba(0,0,0,0.2);
    min-width: 600px; /* Larger for editor */
    max-width: 80vw;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
  }
  .library-viewer h3 {
    margin-top: 0;
  }
  .library-viewer textarea {
    width: 100%;
    flex-grow: 1; /* Make textarea fill space */
    margin-bottom: 1rem;
    font-family: monospace; /* Good for code/text */
    resize: none; /* Disable manual resize */
  }
  .modal-actions {
    margin-top: 1rem;
    text-align: right;
  }
  .modal-actions button {
    margin-left: 0.5rem;
  }
  .error-message {
    color: red;
    margin-top: 0.5rem;
  }
</style>