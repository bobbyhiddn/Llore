<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  let title = '';
  let instructions = '';

  function handleGenerate() {
    dispatch('generate', { title, instructions });
  }

  function handleCancel() {
    dispatch('close');
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      handleCancel();
    }
  }
</script>

<svelte:window on:keydown={handleKeydown}/>

<button class="modal-backdrop" on:click={handleCancel} aria-label="Close modal">
  <div class="modal-content" on:click|stopPropagation role="document">
    <h2>Generate Next Document</h2>
    <p>Provide a title and instructions for the next section of your story.</p>
    
    <div class="form-group">
      <label for="title">New Document Title</label>
      <input type="text" id="title" bind:value={title} placeholder="e.g., Chapter 2: The Plot Thickens">
    </div>

    <div class="form-group">
      <label for="instructions">Instructions</label>
      <textarea id="instructions" bind:value={instructions} placeholder="e.g., Continue from the cliffhanger, focusing on the detective's discovery..."></textarea>
    </div>

    <div class="modal-actions">
      <button class="secondary" on:click={handleCancel}>Cancel</button>
      <button class="primary" on:click={handleGenerate}>Generate</button>
    </div>
  </div>
</button>

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.6);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
    border: none;
    padding: 0;
    cursor: default;
  }

  .modal-content {
    background-color: var(--color-background-secondary);
    padding: 2rem;
    border-radius: 8px;
    width: 90%;
    max-width: 500px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
  }

  h2 {
    margin-top: 0;
    color: var(--color-text-primary);
  }

  p {
    color: var(--color-text-secondary);
    margin-bottom: 1.5rem;
  }

  .form-group {
    margin-bottom: 1rem;
  }

  label {
    display: block;
    margin-bottom: 0.5rem;
    color: var(--color-text-primary);
    font-weight: 500;
  }

  input,
  textarea {
    width: 100%;
    padding: 0.75rem;
    border: 1px solid var(--color-border);
    border-radius: 4px;
    background-color: var(--color-background-tertiary);
    color: var(--color-text-primary);
    font-size: 1rem;
  }

  textarea {
    min-height: 120px;
    resize: vertical;
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: 1rem;
    margin-top: 2rem;
  }

  button {
    padding: 0.75rem 1.5rem;
    border-radius: 4px;
    border: none;
    cursor: pointer;
    font-weight: 600;
  }

  .primary {
    background-color: var(--color-accent);
    color: white;
  }

  .secondary {
    background-color: var(--color-background-tertiary);
    color: var(--color-text-primary);
    border: 1px solid var(--color-border);
  }
</style>
