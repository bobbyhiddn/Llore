<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { database } from '@wailsjs/go/models';
  import '../../styles/AutocompleteMenu.css';

  export let items: database.CodexEntry[] = [];
  export let x: number;
  export let y: number;
  
  let selectedIndex = 0;
  const dispatch = createEventDispatcher();

  function selectItem(item: database.CodexEntry) {
    dispatch('select', item);
  }

  // Allow keyboard navigation
  export function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      selectedIndex = (selectedIndex + 1) % items.length;
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      selectedIndex = (selectedIndex - 1 + items.length) % items.length;
    } else if (event.key === 'Enter' || event.key === 'Tab') {
      event.preventDefault();
      if (items[selectedIndex]) {
        selectItem(items[selectedIndex]);
      }
    }
  }
</script>

<div class="autocomplete-menu" style="left: {x}px; top: {y}px;">
  {#if items.length > 0}
    <ul>
      {#each items as item, i (item.id)}
        <li class:selected={i === selectedIndex} on:mousedown={() => selectItem(item)}>
          <strong>{item.name}</strong>
          <span>({item.type})</span>
        </li>
      {/each}
    </ul>
  {:else}
    <div class="no-results">No matching entries found.</div>
  {/if}
</div>

