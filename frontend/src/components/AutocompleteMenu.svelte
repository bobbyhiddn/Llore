<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { database } from '@wailsjs/go/models';

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

<style>
  .autocomplete-menu {
    position: fixed;
    z-index: 1002;
    background: var(--bg-secondary);
    border: 1px solid var(--accent-secondary);
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.25);
    max-height: 250px;
    overflow-y: auto;
    min-width: 200px;
  }
  ul { list-style: none; margin: 0; padding: 0.25rem; }
  li {
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  li:hover, li.selected {
    background: var(--accent-primary);
    color: white;
  }
  li strong { font-weight: 500; }
  li span { font-size: 0.8rem; color: var(--text-secondary); }
  li:hover span, li.selected span { color: rgba(255, 255, 255, 0.7); }
  .no-results { padding: 0.75rem; color: var(--text-secondary); }
</style>