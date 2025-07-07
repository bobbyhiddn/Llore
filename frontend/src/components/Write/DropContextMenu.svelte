<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import '../../styles/DropContextMenu.css';
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

