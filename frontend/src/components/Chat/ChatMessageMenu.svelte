<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import '../../styles/ChatMessageMenu.css';

  const dispatch = createEventDispatcher();

  export let style = '';
  export let showWeave = true;
  export let showInsert = true;
  export let showCopy = true;
  export let showIndex = true;
  export let className = '';
</script>

<div class="chat-message-menu {className}" {style}>
  {#if showWeave}
    <button on:click|stopPropagation={() => dispatch('action', 'weave')}>
      <span class="icon">✨</span> Weave
    </button>
  {/if}
  {#if showInsert}
    <button on:click|stopPropagation={() => dispatch('action', 'insert')}>
      <span class="icon">↵</span> Insert
    </button>
  {/if}
  {#if showCopy}
    <button on:click|stopPropagation={() => dispatch('action', 'copy')}>
      <span class="icon">📋</span> Copy
    </button>
  {/if}
  {#if showIndex}
    <button on:click|stopPropagation={() => dispatch('action', 'index')}>
      <span class="icon">📚</span> Index
    </button>
  {/if}
</div>

