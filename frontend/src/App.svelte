<script>
  import { onMount } from 'svelte';
  import { 
    GetAllEntries, 
    UpdateEntry, 
    CreateEntry, 
    DeleteEntry, 
    GenerateContent, 
    ProcessStory,
    GetCurrentDatabasePath,
    SelectDatabaseFile,
    SaveDatabaseFile,
    SwitchDatabase,
    CopyDatabase,
    IsDatabaseLoaded
  } from '../wailsjs/go/main/App.js';

  let entries = [];
  let isLoading = false;
  let isGenerating = false;
  let isProcessingStory = false;
  let errorMsg = '';
  let processStoryErrorMsg = '';
  let storyText = '';
  let currentDBPath = 'No database loaded';
  let databaseIsReady = false;
  let initialErrorMsg = '';
  let currentEntry = { id: null, name: '', type: '', content: '', createdAt: null, updatedAt: null }; 
  let isEditing = false;

  onMount(async () => {
    await fetchCurrentDBPath(); 
    await loadEntries();
    resetForm();
    isEditing = false;
    currentEntry = { id: null, name: '', type: '', content: '', createdAt: null, updatedAt: null };
  });

  async function loadEntries() {
    isLoading = true;
    errorMsg = '';
    try {
      entries = await GetAllEntries() || []; 
    } catch (err) {
      console.error("Error loading entries:", err);
      errorMsg = `Error loading entries: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  function handleEntrySelect(entry) {
    if (!entry) return;
    currentEntry = JSON.parse(JSON.stringify(entry)); 
    isEditing = true; 
    errorMsg = ''; 
  }

  async function handleSaveEntry() {
    if (!currentEntry) { 
      errorMsg = 'No entry data to save.';
      return;
    }
    
    if (isEditing) {
      if (typeof currentEntry.id !== 'number' || currentEntry.id <= 0) {
        errorMsg = 'Cannot update: Invalid entry ID.'; 
        console.warn("Update aborted, invalid ID:", currentEntry.id);
        return; 
      }
      if (!currentEntry.name) {
        errorMsg = 'Entry must have a name to update.';
        return;
      }

      isLoading = true;
      errorMsg = '';
      try {
        console.log("Attempting to update entry:", currentEntry);
        await UpdateEntry(currentEntry); 
        alert('Entry updated successfully!'); 
        const updatedId = currentEntry.id; 
        await loadEntries(); 
        const updatedEntryInList = entries.find(e => e.id === updatedId);
        if (updatedEntryInList) {
          handleEntrySelect(updatedEntryInList); 
        } else {
          resetForm(); 
        }
      } catch (err) {
        console.error("Error updating entry:", err);
        errorMsg = `Failed to update entry: ${err}`;
      } finally {
        isLoading = false;
      }
    } else {
      if (!currentEntry.name) {
        errorMsg = 'Entry must have a name to create.';
        return;
      }

      isLoading = true;
      errorMsg = '';
      try {
        console.log("Attempting to create entry:", currentEntry);
        const newEntry = await CreateEntry(currentEntry.name, currentEntry.type, currentEntry.content);
        alert(`Entry '${newEntry.name}' created successfully!`);
        await loadEntries(); // Reload list to show the new entry
        // Optionally select the newly created entry
        const newEntryInList = entries.find(e => e.id === newEntry.id);
        if (newEntryInList) {
          handleEntrySelect(newEntryInList);
        } else {
          resetForm(); // Or just reset if not found (shouldn't happen)
        }
      } catch (err) {
        console.error("Error creating entry:", err);
        errorMsg = `Failed to create entry: ${err}`;
      } finally {
        isLoading = false;
      }
    }
  }

  function prepareNewEntry() {
    resetForm(); 
    // document.getElementById('entry-name')?.focus(); 
  }

  async function handleDeleteEntry() {
    if (!currentEntry || typeof currentEntry.id !== 'number' || currentEntry.id <= 0) {
      errorMsg = 'No valid entry selected for deletion.';
      return;
    }

    if (!confirm(`Are you sure you want to delete '${currentEntry.name}'?`)) {
      return;
    }

    isLoading = true;
    errorMsg = '';
    try {
      await DeleteEntry(currentEntry.id);
      alert('Entry deleted successfully!');
      await loadEntries(); 
      resetForm(); 
    } catch (err) {
      console.error("Error deleting entry:", err);
      errorMsg = `Failed to delete entry: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  function resetForm() {
      currentEntry = { id: null, name: '', type: '', content: '', createdAt: null, updatedAt: null };
      isEditing = false; 
      errorMsg = '';
  }

  async function handleGenerateContent() {
    if (!currentEntry || !currentEntry.name) {
        errorMsg = 'Please select an entry or provide a name for a new one before generating content.';
        return;
    }
    isGenerating = true;
    errorMsg = '';
    const prompt = `Generate a descriptive paragraph for a codex entry.\nName: ${currentEntry.name}\nType: ${currentEntry.type}\nExisting Content (if any): ${currentEntry.content || 'None'}`;
    try {
      const generated = await GenerateContent(prompt); 
      currentEntry.content = generated; 
      currentEntry = { ...currentEntry }; 
    } catch (err) {
      console.error("Error generating content:", err);
      errorMsg = `Error generating content: ${err}`;
    } finally {
      isGenerating = false;
    }
  }

  async function handleProcessStory() {
    if (!storyText.trim()) {
      processStoryErrorMsg = 'Please paste the story text into the textarea.';
      return;
    }
    isProcessingStory = true;
    processStoryErrorMsg = '';
    errorMsg = ''; 
    try {
      const createdEntries = await ProcessStory(storyText);
      console.log('Successfully processed story, created entries:', createdEntries);
      storyText = ''; 
      await loadEntries(); 
    } catch (err) {
      console.error("Error processing story:", err);
      processStoryErrorMsg = `Error processing story: ${err}`;
    } finally {
      isProcessingStory = false;
    }
  }

  async function fetchCurrentDBPath() {
    try {
      currentDBPath = await GetCurrentDatabasePath();
    } catch (err) {
      errorMsg = `Error fetching current DB path: ${err}`;
      currentDBPath = 'Error loading path';
    }
  }

  async function handleCopyDB() {
    try {
      const newPath = await SaveDatabaseFile(); 
      if (newPath) {
        console.log(`Attempting to copy database to: ${newPath}`);
        await CopyDatabase(newPath); 
        console.log(`Database successfully copied to ${newPath}`);
        alert(`Database saved as ${newPath}`); 
      } else {
        console.log("Save As dialog cancelled.");
      }
    } catch (error) {
      errorMsg = `Error during Save As: ${error}`;
      console.error('Error during Save As:', error);
      alert(errorMsg); 
    }
  }

  async function handleCreateNew() {
    try {
      const newPath = await SaveDatabaseFile();
      if (newPath) {
        await SwitchDatabase(newPath);
        databaseIsReady = true;
        await updateCurrentDBPath();
        await loadEntries();
      }
    } catch (err) {
      initialErrorMsg = `Error creating database: ${err}`;
      databaseIsReady = false;
    }
  }

  async function handleLoadExisting() {
    try {
      const existingPath = await SelectDatabaseFile();
      if (existingPath) {
        await SwitchDatabase(existingPath);
        databaseIsReady = true;
        await updateCurrentDBPath();
        await loadEntries();
      }
    } catch (err) {
      initialErrorMsg = `Error loading database: ${err}`;
      databaseIsReady = false;
    }
  }

  async function updateCurrentDBPath() {
    try {
      currentDBPath = await GetCurrentDatabasePath();
    } catch (err) {
      currentDBPath = "Error loading path";
    }
  }

  // Handle keydown for accessibility on list items
  function handleLiKeyDown(event, entry) {
    // Trigger selection on Enter or Space key press
    if (event.key === 'Enter' || event.key === ' ') {
      event.preventDefault(); // Prevent default space bar scroll
      handleEntrySelect(entry);
    }
  }

</script>

{#if databaseIsReady}
<main>
  <h1>Llore Codex</h1>

  <div class="db-path-display">
      Current DB: {currentDBPath || 'None loaded'}
      <button on:click={handleCopyDB} disabled={isLoading}>Copy DB</button>
  </div>

  <div class="layout-container">
    <aside class="sidebar">
      <h2>Entries</h2>
      <button on:click={prepareNewEntry} disabled={isLoading}>+ New Entry</button> 
      {#if isLoading && entries.length === 0}
        <p>Loading entries...</p>
      {:else if entries.length === 0}
        <p>No entries found.</p>
      {/if}
      <ul>
        {#each entries as entry (entry.id)}
          <!-- Keep li for list structure, move interaction to inner div -->
          <li class:selected={currentEntry?.id === entry.id}>
            <div 
              class="entry-item-button"
              role="button"
              tabindex="0"
              on:click={() => handleEntrySelect(entry)} 
              on:keydown={(e) => handleLiKeyDown(e, entry)}
            >
              {entry.name || '(Unnamed)'} ({entry.type || 'Untyped'})
            </div>
          </li>
        {/each}
      </ul>
    </aside>

    <section class="main-content">
      {#if isEditing}
        <h2>Edit Entry: {currentEntry.name}</h2>
      {:else}
         <h2>Create New Entry</h2> 
      {/if}

      <form on:submit|preventDefault={handleSaveEntry}>
        <div class="form-group">
          <label for="entry-name">Name:</label>
          <input id="entry-name" type="text" bind:value={currentEntry.name} required disabled={isLoading}>
        </div>
        <div class="form-group">
          <label for="entry-type">Type:</label>
          <input id="entry-type" type="text" bind:value={currentEntry.type} disabled={isLoading}>
        </div>
        <div class="form-group">
          <label for="entry-content">Content:</label>
          <textarea id="entry-content" rows="10" bind:value={currentEntry.content} disabled={isLoading || isGenerating}></textarea>
        </div>
        
        {#if currentEntry.id} 
          <div class="timestamps">
            <small>Created: {currentEntry.createdAt || 'N/A'} | Updated: {currentEntry.updatedAt || 'N/A'}</small>
          </div>
        {/if}

        <div class="button-group">
          <button type="submit" disabled={isLoading}>{isEditing ? 'Update Entry' : 'Create Entry'}</button> 
          
          {#if isEditing} 
            <button type="button" on:click={handleDeleteEntry} disabled={isLoading || !currentEntry.id} class="danger">Delete Entry</button>
          {/if}

          <button type="button" on:click={handleGenerateContent} disabled={isLoading || isGenerating || !currentEntry.name}>
            {#if isGenerating}Generating...{:else}Generate Content (AI){/if}
          </button>
        </div>
      </form>

      {#if errorMsg}
        <p class="error-message">{errorMsg}</p>
      {/if}
    </section>

    <section class="story-processor">
      <h2>Process Story (AI)</h2>
      <textarea 
        bind:value={storyText} 
        rows="15" 
        placeholder="Paste your story text here..."
        disabled={isProcessingStory}
      ></textarea>
      <button on:click={handleProcessStory} disabled={isProcessingStory || !storyText.trim()}>
        {#if isProcessingStory}Processing...{:else}Process Story & Add Entries{/if}
      </button>
      {#if processStoryErrorMsg}
        <p class="error-message">{processStoryErrorMsg}</p>
      {/if}
    </section>

  </div> 

</main>
{:else}
  <div class="initial-prompt">
    <h1>Welcome to Llore</h1>
    <p>Create a new database or load an existing one to continue.</p>
    {#if initialErrorMsg}
      <p style="color: red">{initialErrorMsg}</p>
    {/if}
    <button on:click={handleCreateNew} disabled={isLoading}>
        {#if isLoading && !databaseIsReady}Creating...{:else}Create New Database{/if}
    </button>
    <button on:click={handleLoadExisting} disabled={isLoading}>
        {#if isLoading && !databaseIsReady}Loading...{:else}Load Existing Database{/if}
    </button>
  </div>
{/if}

<style>
  /* ... existing styles ... */
  .layout-container {
    display: flex;
    gap: 1rem; 
  }
  .sidebar {
    flex: 0 0 250px; 
    border-right: 1px solid #ccc;
    padding-right: 1rem;
    height: calc(100vh - 150px); 
    overflow-y: auto;
  }
  .sidebar ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .sidebar li {
    padding: 0; /* Remove padding from li itself */
    cursor: default; /* Li is no longer directly clickable */
    border-bottom: 1px solid #eee;
    /* Remove transition and hover effects from li */
  }
  .sidebar li.selected .entry-item-button {
    /* Apply selected styles to the inner div now */
    background-color: #e0e0ff; 
    font-weight: bold;
  }
  /* Style the inner div to be interactive */
  .entry-item-button {
    display: block; /* Make it take full width of li */
    padding: 0.5rem; /* Apply padding here */
    cursor: pointer;
    transition: background-color 0.2s;
    outline: none; /* Remove default outline, rely on :focus style */
  }
  .entry-item-button:hover {
    background-color: #f0f0f0;
  }
  /* Add focus style for keyboard navigation to the inner div */
  .entry-item-button:focus {
    outline: 2px solid blue; /* Or your preferred focus style */
    outline-offset: -2px; /* Adjust offset as needed */
    background-color: #e8e8ff; /* Slightly different background on focus */
  }
  .main-content {
    flex: 1; /* Take remaining space */
  }
  .story-processor {
    flex: 0 0 300px; 
    display: flex;
    flex-direction: column;
  }
  .story-processor textarea {
    flex-grow: 1;
    margin-bottom: 0.5rem;
  }
  .form-group {
    margin-bottom: 1rem;
  }
  label {
    display: block;
    margin-bottom: 0.25rem;
  }
  input[type="text"],
  textarea {
    width: 100%;
    padding: 0.5rem;
    box-sizing: border-box;
  }
  textarea {
    resize: vertical;
  }
  .button-group button {
    margin-right: 0.5rem;
  }
  .button-group button.danger {
    background-color: #dc3545;
    color: white;
  }
  .timestamps {
      font-size: 0.8em;
      color: #666;
      margin-top: 0.5rem;
  }
</style>
