<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { main } from '@wailsjs/go/models';

  export let item: main.LibraryItem;
  export let selectedItem: string | null;
  export let selectedItems: Set<string>;
  export let expandedFolders: Set<string>;
  export let dragOverItem: string | null;
  export let level: number = 0;
  export let cutItem: string | null = null;

  const dispatch = createEventDispatcher();

  function handleItemClick(event: MouseEvent) {
    dispatch('itemclick', { item, event });
  }

  function handleItemDoubleClick() {
    dispatch('itemdoubleclick', { item });
  }

  function handleItemRightClick(event: MouseEvent) {
    dispatch('itemrightclick', { item, event });
  }

  function handleDragStart(event: DragEvent) {
    dispatch('dragstart', { item, event });
  }

  function handleDragOver(event: DragEvent) {
    dispatch('dragover', { item, event });
  }

  function handleDragLeave() {
    dispatch('dragleave');
  }

  function handleDrop(event: DragEvent) {
    dispatch('drop', { item, event });
  }

  function handleDragEnd() {
    dispatch('dragend');
  }

  // Format file size
  function formatFileSize(bytes: number): string {
    if (bytes === 0) return '';
    const sizes = ['B', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(1024));
    return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
  }

  // Format modification time
  function formatModTime(modTime: any): string {
    if (!modTime) return '';
    const date = new Date(modTime);
    return date.toLocaleDateString();
  }
</script>

<div class="tree-item"
     class:selected={selectedItems.has(item.path)}
     class:primary-selected={selectedItem === item.path}
     class:drag-over={dragOverItem === item.path}
     class:cut={cutItem === item.path}
     style="margin-left: {level * 1.5}rem;">
  <div class="item-content"
       draggable="true"
       on:dragstart={handleDragStart}
       on:dragover={handleDragOver}
       on:dragleave={handleDragLeave}
       on:drop={handleDrop}
       on:dragend={handleDragEnd}
       on:click={handleItemClick}
       on:dblclick={handleItemDoubleClick}
       on:contextmenu={handleItemRightClick}>
    
    <div class="item-icon">
      {#if item.isDir}
        <span class="folder-icon">
          {expandedFolders.has(item.path) ? '📂' : '📁'}
        </span>
      {:else}
        <span class="file-icon">📄</span>
      {/if}
    </div>
    
    <div class="item-details">
      <span class="item-name">{item.name}</span>
      <div class="item-meta">
        {#if !item.isDir}
          <span class="file-size">{formatFileSize(item.size)}</span>
        {/if}
        <span class="mod-time">{formatModTime(item.modTime)}</span>
      </div>
    </div>
  </div>
</div>

<!-- Recursively render children if folder is expanded -->
{#if item.isDir && expandedFolders.has(item.path) && item.children && item.children.length > 0}
  {#each item.children as childItem (childItem.path)}
    <svelte:self 
      item={childItem}
      {selectedItem}
      {selectedItems}
      {expandedFolders}
      {dragOverItem}
      {cutItem}
      level={level + 1}
      on:itemclick
      on:itemdoubleclick
      on:itemrightclick
      on:dragstart
      on:dragover
      on:dragleave
      on:drop
      on:dragend
    />
  {/each}
{/if}

<style>
  .tree-item {
    margin-bottom: 0.25rem;
  }

  .item-content {
    display: flex;
    align-items: center;
    padding: 0.5rem;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s ease;
    user-select: none;
  }

  .item-content:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .tree-item.selected .item-content {
    background: rgba(var(--accent-primary-rgb, 0, 123, 255), 0.6);
    color: white;
  }

  .tree-item.primary-selected .item-content {
    background: var(--accent-primary);
    color: white;
  }

  .tree-item.drag-over .item-content {
    background: rgba(0, 123, 255, 0.3);
    border: 2px dashed #007bff;
  }

  .tree-item.cut .item-content {
    opacity: 0.5;
  }

  .tree-item.cut .item-name {
    text-decoration: line-through;
  }

  .item-icon {
    margin-right: 0.5rem;
    font-size: 1.1rem;
  }

  .item-details {
    flex: 1;
    min-width: 0;
  }

  .item-name {
    display: block;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .item-meta {
    display: flex;
    gap: 1rem;
    font-size: 0.8rem;
    color: var(--text-secondary);
    margin-top: 0.2rem;
  }

  .tree-item.selected .item-meta {
    color: rgba(255, 255, 255, 0.8);
  }
</style>