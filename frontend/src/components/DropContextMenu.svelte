<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let x: number;
  export let y: number;

  const dispatch = createEventDispatcher();
</script>

<div class="context-menu" style="left: {x}px; top: {y}px;">
  <button on:click={() => dispatch('action', 'reference')}>
    <span class="icon">@</span> Insert as Reference
  </button>
  <button on:click={() => dispatch('action', 'weave')}>
    <span class="icon">✨</span> Weave into Document
  </button>
</div>

<style>
  .context-menu {
    position: fixed;
    z-index: 1001;
    background: var(--bg-secondary);
    border: 1px solid var(--accent-primary);
    border-radius: 8px;
    box-shadow: 0 5px 15px rgba(0,0,0,0.3);
    padding: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  button {
    background: transparent;
    border: none;
    color: var(--text-primary);
    padding: 0.5rem 1rem;
    text-align: left;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  button:hover { background: var(--bg-hover-medium); }
  .icon { font-weight: bold; color: var(--accent-secondary); }
</style>