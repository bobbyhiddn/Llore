<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher();

  export let style = '';
</script>

<div class="chat-message-menu" {style}>
  <button on:click|stopPropagation={() => dispatch('action', 'weave')}>
    <span class="icon">✨</span> Weave
  </button>
  <button on:click|stopPropagation={() => dispatch('action', 'insert')}>
    <span class="icon">↵</span> Insert
  </button>
  <button on:click|stopPropagation={() => dispatch('action', 'copy')}>
    <span class="icon">📋</span> Copy
  </button>
  <button on:click|stopPropagation={() => dispatch('action', 'index')}>
    <span class="icon">📚</span> Index
  </button>
</div>

<style>
  .chat-message-menu {
    position: absolute;
    z-index: 10;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color-medium);
    border-radius: 6px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.2);
    padding: 0.4rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  button {
    background: transparent;
    border: none;
    color: var(--text-primary);
    padding: 0.5rem 0.8rem;
    text-align: left;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 0.6rem;
    font-size: 0.8rem;
  }
  button:hover { 
    background: var(--bg-hover-strong); 
    color: var(--text-accent-hover);
  }
  .icon { 
    font-weight: bold; 
    color: var(--accent-secondary); 
    width: 16px;
    text-align: center;
  }
</style>