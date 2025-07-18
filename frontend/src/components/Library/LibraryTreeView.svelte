<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import { ListLibraryHierarchy, CreateLibraryFolder, DeleteLibraryItem, MoveLibraryItem, CopyLibraryItem, SaveLibraryFileWithPath } from '@wailsjs/go/main/App';
  import { main } from '@wailsjs/go/models';
  import LibraryTreeItem from './LibraryTreeItem.svelte';
  import '../../styles/LibraryTreeView.css';

  // Props
  export let isLibraryLoading: boolean = false;
  export let errorMsg: string = '';

  const dispatch = createEventDispatcher();

  // State
  let libraryItems: main.LibraryItem[] = [];
  let expandedFolders: Set<string> = new Set();
  let selectedItem: string | null = null;
  let selectedItems: Set<string> = new Set();
  let lastSelectedItem: string | null = null;
  let draggedItem: string | null = null;
  let dragOverItem: string | null = null;
  let isDragOverRoot: boolean = false;
  let isProcessingDrop: boolean = false;
  let showContextMenu: boolean = false;
  let contextMenuX: number = 0;
  let contextMenuY: number = 0;
  let contextMenuItem: string | null = null;
  let showNewFolderDialog: boolean = false;
  let newFolderName: string = '';
  let newFolderParentPath: string = '';
  let showRenameDialog: boolean = false;
  let renameItemPath: string = '';
  let renameItemNewName: string = '';
  let showNewFileDialog: boolean = false;
  let newFileName: string = '';
  let newFileParentPath: string = '';
  
  // Clipboard state for copy/cut/paste operations
  let clipboardItem: string | null = null;
  let clipboardOperation: 'copy' | 'cut' | null = null;

  // Load library hierarchy
  async function loadLibrary() {
    isLibraryLoading = true;
    errorMsg = '';
    try {
      libraryItems = await ListLibraryHierarchy();
    } catch (err) {
      errorMsg = `Failed to load library: ${err}`;
      console.error('Error loading library:', err);
    } finally {
      isLibraryLoading = false;
    }
  }

  // Initialize on mount
  loadLibrary();

  function goBack() {
    dispatch('back');
  }

  function refreshLibrary() {
    loadLibrary();
  }

  function toggleFolder(folderPath: string) {
    if (expandedFolders.has(folderPath)) {
      expandedFolders.delete(folderPath);
    } else {
      expandedFolders.add(folderPath);
    }
    expandedFolders = expandedFolders; // Trigger reactivity
  }

  function handleItemClick(event: CustomEvent) {
    const { item, event: mouseEvent } = event.detail;
    mouseEvent.stopPropagation();
    
    const isCtrlClick = mouseEvent.ctrlKey || mouseEvent.metaKey;
    const isShiftClick = mouseEvent.shiftKey;
    
    if (isShiftClick && lastSelectedItem) {
      // Shift-click: select range
      selectRange(lastSelectedItem, item.path);
    } else if (isCtrlClick) {
      // Ctrl-click: toggle selection
      toggleItemSelection(item.path);
    } else {
      // Normal click: single selection and show context menu for files
      if (item.isDir) {
        toggleFolder(item.path);
        selectSingleItem(item.path);
      } else {
        selectSingleItem(item.path);
        dispatch('fileselected', item);
        
        // Show context menu for files on single click
        contextMenuItem = item.path;
        positionContextMenu(mouseEvent);
        showContextMenu = true;
      }
    }
  }

  function selectSingleItem(itemPath: string) {
    selectedItems.clear();
    selectedItems.add(itemPath);
    selectedItems = selectedItems; // Trigger reactivity
    selectedItem = itemPath;
    lastSelectedItem = itemPath;
  }

  function toggleItemSelection(itemPath: string) {
    if (selectedItems.has(itemPath)) {
      selectedItems.delete(itemPath);
      if (selectedItem === itemPath) {
        selectedItem = selectedItems.size > 0 ? Array.from(selectedItems)[0] : null;
      }
    } else {
      selectedItems.add(itemPath);
      selectedItem = itemPath;
    }
    selectedItems = selectedItems; // Trigger reactivity
    lastSelectedItem = itemPath;
  }

  function selectRange(startPath: string, endPath: string) {
    // Get all items in a flat list with their paths
    const allItems = getAllItemPaths(libraryItems);
    const startIndex = allItems.indexOf(startPath);
    const endIndex = allItems.indexOf(endPath);
    
    if (startIndex !== -1 && endIndex !== -1) {
      const minIndex = Math.min(startIndex, endIndex);
      const maxIndex = Math.max(startIndex, endIndex);
      
      // Clear current selection and add range
      selectedItems.clear();
      for (let i = minIndex; i <= maxIndex; i++) {
        selectedItems.add(allItems[i]);
      }
      selectedItems = selectedItems; // Trigger reactivity
      selectedItem = endPath;
      lastSelectedItem = endPath;
    }
  }

  function getAllItemPaths(items: main.LibraryItem[]): string[] {
    const paths: string[] = [];
    for (const item of items) {
      paths.push(item.path);
      if (item.isDir && expandedFolders.has(item.path) && item.children) {
        paths.push(...getAllItemPaths(item.children));
      }
    }
    return paths;
  }

  function handleItemDoubleClick(event: CustomEvent) {
    const { item } = event.detail;
    if (!item.isDir) {
      // Close any open context menus before opening preview
      showContextMenu = false;
      dispatch('viewfile', item.path);
    }
  }

  function handleItemRightClick(event: CustomEvent) {
    const { item, event: mouseEvent } = event.detail;
    mouseEvent.preventDefault();
    mouseEvent.stopPropagation();
    
    contextMenuItem = item.path;
    positionContextMenu(mouseEvent);
    showContextMenu = true;
  }

  function handleBackgroundRightClick(event: MouseEvent) {
    event.preventDefault();
    contextMenuItem = null; // Root level
    positionContextMenu(event);
    showContextMenu = true;
  }

  function handleBackgroundClick(event: MouseEvent) {
    // Clear selection when clicking on empty space
    if (event.target === event.currentTarget) {
      selectedItems.clear();
      selectedItems = selectedItems;
      selectedItem = null;
      lastSelectedItem = null;
    }
  }

  function closeContextMenu() {
    showContextMenu = false;
    contextMenuItem = null;
  }

  // Smart positioning for context menu - opens above cursor if in bottom half of screen
  function positionContextMenu(mouseEvent: MouseEvent) {
    const menuHeight = 200; // Approximate height of context menu
    const menuWidth = 180; // Approximate width of context menu
    const screenHeight = window.innerHeight;
    const screenWidth = window.innerWidth;
    const mouseX = mouseEvent.clientX;
    const mouseY = mouseEvent.clientY;
    const padding = 10; // Minimum distance from screen edges
    const offsetX = 15; // Offset to avoid interfering with double-clicks
    const offsetY = 5; // Small vertical offset
    
    // Handle horizontal positioning
    if (mouseX + offsetX + menuWidth > screenWidth - padding) {
      // Position to the left of cursor if too close to right edge
      contextMenuX = mouseX - menuWidth - offsetX;
      if (contextMenuX < padding) {
        contextMenuX = padding;
      }
    } else {
      contextMenuX = mouseX + offsetX;
    }
    
    // Handle vertical positioning
    if (mouseY > screenHeight / 2) {
      // Position above cursor if in bottom half of screen
      contextMenuY = mouseY - menuHeight - offsetY;
      if (contextMenuY < padding) {
        contextMenuY = padding;
      }
    } else {
      // Default position below cursor with offset
      contextMenuY = mouseY + offsetY;
      // Ensure it doesn't go below screen
      if (contextMenuY + menuHeight > screenHeight - padding) {
        contextMenuY = screenHeight - menuHeight - padding;
      }
    }
  }

  // Context menu actions
  function createNewFolder() {
    newFolderParentPath = contextMenuItem || '';
    newFolderName = '';
    showNewFolderDialog = true;
    closeContextMenu();
  }

  function createNewFile() {
    newFileParentPath = contextMenuItem || '';
    newFileName = '';
    showNewFileDialog = true;
    closeContextMenu();
  }

  async function confirmCreateFolder() {
    if (!newFolderName.trim()) return;
    
    const folderPath = newFolderParentPath 
      ? `${newFolderParentPath}/${newFolderName.trim()}`
      : newFolderName.trim();
    
    try {
      await CreateLibraryFolder(folderPath);
      await loadLibrary();
      // Expand parent folder if it exists
      if (newFolderParentPath) {
        expandedFolders.add(newFolderParentPath);
        expandedFolders = expandedFolders;
      }
    } catch (err) {
      errorMsg = `Failed to create folder: ${err}`;
    }
    
    showNewFolderDialog = false;
  }

  async function confirmCreateFile() {
    if (!newFileName.trim()) return;
    
    // Ensure the filename has an extension (default to .md)
    let filename = newFileName.trim();
    if (!filename.includes('.')) {
      filename += '.md';
    }
    
    const filePath = newFileParentPath 
      ? `${newFileParentPath}/${filename}`
      : filename;
    
    try {
      await SaveLibraryFileWithPath(filePath, '# New Document\n\nStart writing here...');
      await loadLibrary();
      
      // Expand parent folder if it exists
      if (newFileParentPath) {
        expandedFolders.add(newFileParentPath);
        expandedFolders = expandedFolders;
      }
      
      // Emit event to open the new file
      dispatch('viewfile', filePath);
    } catch (err) {
      errorMsg = `Failed to create file: ${err}`;
    }
    
    showNewFileDialog = false;
  }

  function renameItem() {
    if (!contextMenuItem) return;
    
    renameItemPath = contextMenuItem;
    const item = findItemByPath(libraryItems, contextMenuItem);
    renameItemNewName = item?.name || '';
    showRenameDialog = true;
    closeContextMenu();
  }

  async function confirmRename() {
    if (!renameItemNewName.trim() || !renameItemPath) return;
    
    const pathParts = renameItemPath.split('/');
    pathParts[pathParts.length - 1] = renameItemNewName.trim();
    const newPath = pathParts.join('/');
    
    if (newPath === renameItemPath) {
      showRenameDialog = false;
      return;
    }
    
    try {
      await MoveLibraryItem(renameItemPath, newPath);
      await loadLibrary();
    } catch (err) {
      errorMsg = `Failed to rename: ${err}`;
    }
    
    showRenameDialog = false;
  }

  async function deleteItem() {
    const itemsToDelete = selectedItems.size > 1 ? Array.from(selectedItems) : [contextMenuItem].filter(Boolean);
    if (itemsToDelete.length === 0) return;
    
    const itemWord = itemsToDelete.length === 1 ? 'item' : 'items';
    
    if (confirm(`Are you sure you want to delete ${itemsToDelete.length} ${itemWord}?`)) {
      try {
        for (const itemPath of itemsToDelete) {
          await DeleteLibraryItem(itemPath);
        }
        selectedItems.clear();
        selectedItems = selectedItems;
        selectedItem = null;
        await loadLibrary();
      } catch (err) {
        errorMsg = `Failed to delete ${itemWord}: ${err}`;
      }
    }
    
    closeContextMenu();
  }

  function editInWriteMode() {
    if (!contextMenuItem) return;
    dispatch('editinwrite', contextMenuItem);
    closeContextMenu();
  }

  function viewFile() {
    if (!contextMenuItem) return;
    dispatch('viewfile', contextMenuItem);
    closeContextMenu();
  }

  // Copy/Cut/Paste operations
  function copyItem() {
    if (selectedItems.size > 1) {
      // Multi-selection copy
      clipboardItem = Array.from(selectedItems).join('|'); // Use pipe separator for multiple items
      clipboardOperation = 'copy';
    } else if (contextMenuItem) {
      clipboardItem = contextMenuItem;
      clipboardOperation = 'copy';
    }
    closeContextMenu();
  }

  function cutItem() {
    if (selectedItems.size > 1) {
      // Multi-selection cut
      clipboardItem = Array.from(selectedItems).join('|'); // Use pipe separator for multiple items
      clipboardOperation = 'cut';
    } else if (contextMenuItem) {
      clipboardItem = contextMenuItem;
      clipboardOperation = 'cut';
    }
    closeContextMenu();
  }

  async function pasteItem() {
    if (!clipboardItem || !clipboardOperation) return;
    
    const destinationPath = contextMenuItem || ''; // Paste to current context or root
    const itemsToPaste = clipboardItem.includes('|') ? clipboardItem.split('|') : [clipboardItem];
    
    try {
      for (const itemPath of itemsToPaste) {
        // Extract item name from clipboard item path
        const itemName = itemPath.split(/[/\\]/).pop();
        if (!itemName) {
          errorMsg = `Failed to extract item name from path: ${itemPath}`;
          continue;
        }
        
        // Build destination path
        const newPath = destinationPath 
          ? `${destinationPath}/${itemName}`
          : itemName;
        
        // Check if we're trying to paste into the same location
        if (itemPath === newPath) {
          continue;
        }
        
        if (clipboardOperation === 'copy') {
          await CopyLibraryItem(itemPath, newPath);
        } else if (clipboardOperation === 'cut') {
          await MoveLibraryItem(itemPath, newPath);
        }
      }
      
      // Clear clipboard after cut operation
      if (clipboardOperation === 'cut') {
        clipboardItem = null;
        clipboardOperation = null;
        selectedItems.clear();
        selectedItems = selectedItems;
      }
      
      await loadLibrary();
      
      // Expand destination folder if it exists
      if (destinationPath) {
        expandedFolders.add(destinationPath);
        expandedFolders = expandedFolders;
      }
      
    } catch (err) {
      errorMsg = `Failed to ${clipboardOperation}: ${err}`;
    }
    
    closeContextMenu();
  }

  // Check if paste is available
  function canPaste(): boolean {
    return clipboardItem !== null && clipboardOperation !== null;
  }

  // Keyboard shortcuts
  function handleKeyDown(event: KeyboardEvent) {
    if (event.ctrlKey || event.metaKey) {
      switch (event.key) {
        case 'c':
          if (selectedItems.size > 0) {
            clipboardItem = Array.from(selectedItems).join('|');
            clipboardOperation = 'copy';
            event.preventDefault();
          }
          break;
        case 'x':
          if (selectedItems.size > 0) {
            clipboardItem = Array.from(selectedItems).join('|');
            clipboardOperation = 'cut';
            event.preventDefault();
          }
          break;
        case 'v':
          if (canPaste()) {
            // Use selected item's parent or root as destination
            contextMenuItem = selectedItem ? findParentPath(selectedItem) : null;
            pasteItem();
            event.preventDefault();
          }
          break;
        case 'a':
          // Select all items
          selectAllItems();
          event.preventDefault();
          break;
      }
    }
  }

  function selectAllItems() {
    const allItems = getAllItemPaths(libraryItems);
    selectedItems.clear();
    for (const itemPath of allItems) {
      selectedItems.add(itemPath);
    }
    selectedItems = selectedItems; // Trigger reactivity
    if (allItems.length > 0) {
      selectedItem = allItems[0];
      lastSelectedItem = allItems[allItems.length - 1];
    }
  }

  // Helper function to find parent path
  function findParentPath(itemPath: string): string | null {
    const pathParts = itemPath.split(/[/\\]/);
    if (pathParts.length <= 1) return null; // Already at root
    pathParts.pop(); // Remove the item name
    return pathParts.join('/');
  }

  // Drag and drop handlers
  function handleDragStart(event: CustomEvent) {
    const { item, event: dragEvent } = event.detail;
    console.log('Drag start:', { item, itemPath: item.path, itemName: item.name, isDir: item.isDir });
    if (dragEvent.dataTransfer) {
      draggedItem = item.path;
      dragEvent.dataTransfer.effectAllowed = 'move';
      dragEvent.dataTransfer.setData('text/plain', item.path);
    }
  }

  function handleDragOver(event: CustomEvent) {
    const { item, event: dragEvent } = event.detail;
    if (item.isDir && draggedItem && draggedItem !== item.path) {
      // Also check if trying to move a folder into itself or its descendants
      // Handle both path separators for Windows compatibility
      const pathSep = item.path.includes('\\') ? '\\' : '/';
      if (!draggedItem.startsWith(item.path + pathSep) && draggedItem !== item.path) {
        dragEvent.preventDefault();
        dragOverItem = item.path;
      }
    }
  }

  function handleDragLeave() {
    dragOverItem = null;
  }

  async function handleDrop(event: CustomEvent) {
    const { item, event: dragEvent } = event.detail;
    dragEvent.preventDefault();
    dragEvent.stopPropagation(); // Stop propagation to prevent root drop
    
    if (isProcessingDrop) {
      console.log('Drop ignored: already processing another drop');
      return;
    }
    
    console.log('Drop target:', { item, itemPath: item.path, itemName: item.name, isDir: item.isDir });
    
    if (!draggedItem || !item.isDir || draggedItem === item.path) {
      console.log('Drop rejected:', { draggedItem, itemIsDir: item.isDir, sameItem: draggedItem === item.path });
      dragOverItem = null;
      draggedItem = null;
      return;
    }
    
    isProcessingDrop = true;
    
    // Check if trying to move a folder into itself or its descendants
    // Handle both path separators for Windows compatibility
    const pathSep = item.path.includes('\\') ? '\\' : '/';
    if (draggedItem.startsWith(item.path + pathSep) || draggedItem === item.path) {
      errorMsg = 'Cannot move a folder into itself or its descendants';
      dragOverItem = null;
      draggedItem = null;
      return;
    }
    
    // Handle both Windows (\) and Unix (/) path separators
    const draggedItemName = draggedItem.split(/[/\\]/).pop();
    const newPath = `${item.path}/${draggedItemName}`;
    
    console.log('Moving item:', {
      draggedItem,
      draggedItemName,
      destinationFolder: item.path,
      newPath
    });
    
    try {
      await MoveLibraryItem(draggedItem, newPath);
      await loadLibrary();
      // Expand the destination folder
      expandedFolders.add(item.path);
      expandedFolders = expandedFolders;
    } catch (err) {
      errorMsg = `Failed to move item: ${err}`;
    }
    
    dragOverItem = null;
    draggedItem = null;
    isProcessingDrop = false;
  }

  function handleDragEnd() {
    console.log('Drag end - cleaning up state');
    draggedItem = null;
    dragOverItem = null;
    isDragOverRoot = false;
  }

  // Root level drag and drop handlers
  function handleRootDragOver(event: DragEvent) {
    if (draggedItem && (draggedItem.includes('/') || draggedItem.includes('\\'))) { // Only allow if item is in a subfolder
      event.preventDefault();
      event.stopPropagation();
      isDragOverRoot = true;
    }
  }

  function handleRootDragLeave(event: DragEvent) {
    // Only clear if we're actually leaving the tree container
    const target = event.currentTarget as HTMLElement;
    const rect = target.getBoundingClientRect();
    if (event.clientX <= rect.left || event.clientX >= rect.right || 
        event.clientY <= rect.top || event.clientY >= rect.bottom) {
      isDragOverRoot = false;
    }
  }

  async function handleRootDrop(event: DragEvent) {
    event.preventDefault();
    event.stopPropagation();
    
    console.log('Root drop triggered:', { draggedItem, isDragOverRoot, isProcessingDrop });
    
    if (isProcessingDrop) {
      console.log('Root drop ignored: folder drop in progress');
      return;
    }
    
    if (!draggedItem || !(draggedItem.includes('/') || draggedItem.includes('\\'))) {
      console.log('Root drop rejected: item not in subfolder');
      isDragOverRoot = false;
      draggedItem = null;
      return;
    }
    
    const draggedItemName = draggedItem.split(/[/\\]/).pop();
    if (!draggedItemName) {
      console.log('Root drop rejected: could not extract filename');
      isDragOverRoot = false;
      draggedItem = null;
      return;
    }
    
    console.log('Root drop executing:', { draggedItem, draggedItemName });
    
    try {
      await MoveLibraryItem(draggedItem, draggedItemName);
      await loadLibrary();
    } catch (err) {
      console.error('Root drop failed:', err);
      errorMsg = `Failed to move item to root: ${err}`;
    }
    
    isDragOverRoot = false;
    draggedItem = null;
  }

  // Helper function to find item by path
  function findItemByPath(items: main.LibraryItem[], path: string): main.LibraryItem | null {
    for (const item of items) {
      if (item.path === path) return item;
      if (item.isDir && item.children) {
        const found = findItemByPath(item.children, path);
        if (found) return found;
      }
    }
    return null;
  }


  // Close dialogs and context menu when clicking outside
  function handleDocumentClick() {
    closeContextMenu();
  }
</script>

<svelte:window on:click={handleDocumentClick} on:keydown={handleKeyDown} />

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>

<section class="library-view" on:contextmenu={handleBackgroundRightClick}>
  <div class="library-header">
    <h2>Library</h2>
    <div class="header-controls">
      <button class="new-file-btn" on:click={createNewFile} title="Create New File">
        📄+ New File
      </button>
      <button class="new-folder-btn" on:click={createNewFolder} title="Create New Folder">
        📁+ New Folder
      </button>
      <button class="refresh-btn" on:click={refreshLibrary} disabled={isLibraryLoading}>
        {#if isLibraryLoading}🔄 Refreshing...{:else}🔄 Refresh{/if}
      </button>
    </div>
  </div>

  {#if isLibraryLoading && libraryItems.length === 0}
    <p class="loading-message">Loading library...</p>
  {:else if errorMsg}
    <p class="error-message">{errorMsg}</p>
  {:else if libraryItems.length === 0}
    <div class="empty-state">
      <p>No files in library.</p>
      <p>Add some via Story Import or Write mode, or create folders to organize your content.</p>
    </div>
  {:else}
    <div class="tree-container" 
         class:drag-over-root={isDragOverRoot}
         on:dragover={handleRootDragOver}
         on:dragleave={handleRootDragLeave}
         on:drop={handleRootDrop}
         on:click={handleBackgroundClick}>
      {#each libraryItems as item (item.path)}
        <LibraryTreeItem 
          {item}
          {selectedItem}
          {selectedItems}
          {expandedFolders}
          {dragOverItem}
          cutItem={clipboardOperation === 'cut' && clipboardItem && (clipboardItem.includes('|') ? clipboardItem.split('|').includes(item.path) : clipboardItem === item.path) ? item.path : null}
          level={0}
          on:itemclick={handleItemClick}
          on:itemdoubleclick={handleItemDoubleClick}
          on:itemrightclick={handleItemRightClick}
          on:dragstart={handleDragStart}
          on:dragover={handleDragOver}
          on:dragleave={handleDragLeave}
          on:drop={handleDrop}
          on:dragend={handleDragEnd}
        />
      {/each}
    </div>
  {/if}
</section>

<!-- Context Menu -->
{#if showContextMenu}
  <div class="context-menu" style="left: {contextMenuX}px; top: {contextMenuY}px;">
    {#if contextMenuItem}
      {@const item = findItemByPath(libraryItems, contextMenuItem)}
      {@const selectionCount = selectedItems.size}
      {#if item}
        {#if selectionCount > 1}
          <div class="menu-header">{selectionCount} items selected</div>
          <div class="menu-separator"></div>
        {/if}
        {#if !item.isDir && selectionCount <= 1}
          <button on:click={viewFile}>👁️ View</button>
          <button on:click={editInWriteMode}>✏️ Edit in Write Mode</button>
          <div class="menu-separator"></div>
        {/if}
        <button on:click={copyItem}>📋 Copy{selectionCount > 1 ? ` (${selectionCount})` : ''}</button>
        <button on:click={cutItem}>✂️ Cut{selectionCount > 1 ? ` (${selectionCount})` : ''}</button>
        {#if canPaste()}
          <button on:click={pasteItem}>📄 Paste</button>
        {/if}
        <div class="menu-separator"></div>
        {#if selectionCount <= 1}
          <button on:click={renameItem}>✏️ Rename</button>
        {/if}
        <button on:click={deleteItem} class="danger">🗑️ Delete{selectionCount > 1 ? ` (${selectionCount})` : ''}</button>
        {#if item.isDir && selectionCount <= 1}
          <div class="menu-separator"></div>
          <button on:click={createNewFile}>📄+ New File</button>
          <button on:click={createNewFolder}>📁+ New Folder</button>
        {/if}
      {/if}
    {:else}
      <button on:click={createNewFile}>📄+ New File</button>
      <button on:click={createNewFolder}>📁+ New Folder</button>
      {#if canPaste()}
        <button on:click={pasteItem}>📄 Paste</button>
      {/if}
    {/if}
  </div>
{/if}

<!-- New File Dialog -->
{#if showNewFileDialog}
  <div class="modal-backdrop">
    <div class="modal">
      <h3>Create New File</h3>
      <p>Creating file in: {newFileParentPath || 'Library root'}</p>
      <input type="text" bind:value={newFileName} placeholder="filename.md" />
      <div class="modal-buttons">
        <button on:click={() => showNewFileDialog = false}>Cancel</button>
        <button on:click={confirmCreateFile} disabled={!newFileName.trim()}>Create</button>
      </div>
    </div>
  </div>
{/if}

<!-- New Folder Dialog -->
{#if showNewFolderDialog}
  <div class="modal-backdrop">
    <div class="modal">
      <h3>Create New Folder</h3>
      <p>Creating folder in: {newFolderParentPath || 'Library root'}</p>
      <input type="text" bind:value={newFolderName} placeholder="Folder name" />
      <div class="modal-buttons">
        <button on:click={() => showNewFolderDialog = false}>Cancel</button>
        <button on:click={confirmCreateFolder} disabled={!newFolderName.trim()}>Create</button>
      </div>
    </div>
  </div>
{/if}

<!-- Rename Dialog -->
{#if showRenameDialog}
  <div class="modal-backdrop">
    <div class="modal">
      <h3>Rename Item</h3>
      <input type="text" bind:value={renameItemNewName} placeholder="New name" />
      <div class="modal-buttons">
        <button on:click={() => showRenameDialog = false}>Cancel</button>
        <button on:click={confirmRename} disabled={!renameItemNewName.trim()}>Rename</button>
      </div>
    </div>
  </div>
{/if}