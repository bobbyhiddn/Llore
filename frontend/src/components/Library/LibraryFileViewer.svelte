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

  // Reference to the Editor component to get current content
  let editorComponent: any;
  
  // Handle content changes (but don't auto-save)
  function handleContentChange(newContent: string) {
    content = newContent;
  }
</script>

<div class="modal-backdrop">
  <div class="modal-content library-viewer">
    <h3>Viewing/Editing: {filename}</h3>
    <div class="editor-container">
      <Editor 
        bind:this={editorComponent}
        content={content}
        onSave={handleContentChange} 
        placeholder="Edit your library file content..."
        editorClass="library-editor"
      />
    </div>
    {#if errorMsg}<p class="error-message">{errorMsg}</p>{/if}
    {#if successMsg}<p class="success-message">{successMsg}</p>{/if}
    <div class="modal-actions">
      <button on:click={handleSave} disabled={isLoading}>Save Changes</button>
      <button on:click={handleReprocess} disabled={isLoading}>Reprocess for Codex</button>
      <button on:click={handleClose} disabled={isLoading}>Close</button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    /* Dark theme modal backdrop */
    position: fixed; top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0, 0, 0, 0.7); display: flex;
    align-items: center; justify-content: center; z-index: 1000;
  }
  .modal-content {
    /* Dark theme modal content */
    background: var(--bg-secondary, #2a2a3e);
    color: var(--text-primary, #e0e0e0);
    border: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
    border-radius: 8px;
    padding: 2rem;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
    min-width: 600px; /* Larger for editor */
    max-width: 80vw;
    max-height: 90vh;
    display: flex;
    flex-direction: column;
  }
  .library-viewer h3 {
    margin-top: 0;
    color: var(--text-primary, #e0e0e0);
    font-size: 1.2rem;
    font-weight: 600;
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
    font-family: var(--font-mono, 'SF Mono', Monaco, 'Cascadia Code', 'Roboto Mono', Consolas, 'Courier New', monospace);
    min-height: 400px;
    background: var(--bg-primary, #1e1e1e);
    color: var(--text-primary, #e0e0e0);
    border: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
    border-radius: 4px;
  }
  
  .modal-actions {
    margin-top: 1rem;
    text-align: right;
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }
  
  .modal-actions button {
    padding: 0.6rem 1rem;
    background: var(--bg-tertiary, #404040);
    color: var(--text-primary, #e0e0e0);
    border: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.2s ease, color 0.2s ease;
    font-size: 0.9rem;
  }
  
  .modal-actions button:hover:not(:disabled) {
    background: var(--bg-hover-medium, rgba(255, 255, 255, 0.1));
    color: var(--text-accent, var(--accent-primary, #6d5ed9));
  }
  
  .modal-actions button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
  
  .error-message {
    color: var(--error-color, #ff6b6b);
    background: var(--error-bg, rgba(255, 107, 107, 0.1));
    padding: 0.5rem;
    border-radius: 4px;
    border-left: 3px solid var(--error-color, #ff6b6b);
    margin-top: 0.5rem;
    font-size: 0.9rem;
  }
  
  .success-message {
    color: var(--success-color, #51cf66);
    background: var(--success-bg, rgba(81, 207, 102, 0.1));
    padding: 0.5rem;
    border-radius: 4px;
    border-left: 3px solid var(--success-color, #51cf66);
    margin-top: 0.5rem;
    font-size: 0.9rem;
  }
</style>