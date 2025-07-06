<!-- LibraryFileViewer.svelte -->
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { ProcessAndSaveTextAsEntries, SaveLibraryFileWithPath } from '@wailsjs/go/main/App';
  import Editor from '../Write/Editor.svelte';

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
      await SaveLibraryFileWithPath(filename, content); // Call Go function to save
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

  // Autosave function for the Editor component
  async function handleAutoSave(newContent: string) {
    // Only autosave if content has actually changed
    if (newContent === content) {
      return; // No change, don't save
    }

    console.log(`Autosaving library file: ${filename}`);
    try {
      await SaveLibraryFile(filename, newContent);
      
      // Update the content so future comparisons work correctly
      content = newContent;
      
      console.log("Library file autosaved successfully.");
      // Clear any previous error messages on successful save
      errorMsg = '';
    } catch (error) {
      console.error("Failed to autosave library file:", error);
      // Show error to user
      errorMsg = `Autosave failed: ${error}`;
    }
  }
</script>

<div class="modal-backdrop">
  <div class="modal-content library-viewer">
    <h3>Viewing/Editing: {filename}</h3>
    <div class="editor-container">
      <Editor 
        content={content}
        onSave={handleAutoSave}
        placeholder="Edit your library file content..."
        editorClass="library-editor"
      />
    </div>
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
  .editor-container {
    flex-grow: 1;
    margin-bottom: 1rem;
    display: flex;
    flex-direction: column;
  }

  .library-viewer :global(.library-editor) {
    height: 100%;
    flex-grow: 1;
  }

  .library-viewer :global(.library-editor .editor-textarea) {
    font-family: monospace; /* Good for code/text */
    min-height: 400px;
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