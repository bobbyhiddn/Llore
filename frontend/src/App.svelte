<script>
  import { onMount } from 'svelte';
  import {
    GetAllEntries, 
    // CreateEntry,  
    // UpdateEntry,  
    DeleteEntry,
    GenerateContent, 
    ProcessStory,
    GetCurrentDatabasePath,
    SelectDatabaseFile,
    SaveDatabaseFile,
    SwitchDatabase,
    CopyDatabase,      
  } from '../wailsjs/go/main/App.js';

  let entries = [];
  let isLoading = false;        // Loading state for general list/manual ops
  let isGenerating = false;     // Loading state for AI content generation
  let isProcessingStory = false; // Loading state for story processing
  let errorMsg = '';
  let processStoryErrorMsg = ''; // Separate error message for story processing
  let storyText = ''; // State variable for the story textarea
  let selectedEntry = null; // Variable to hold the selected entry details
  let currentDBPath = 'Loading...'; // Added for DB path display

  // Ensure currentEntry includes all fields expected from Go's CodexEntry struct
  let currentEntry = { id: null, name: '', type: '', content: '', createdAt: null, updatedAt: null }; 
  let isEditing = false;

  // Load entries when component mounts
  onMount(async () => {
    await fetchCurrentDBPath(); // Fetch current path on load
    await loadEntries();
    resetForm();
    isEditing = false;
    currentEntry = { id: null, name: '', type: '', content: '', createdAt: null, updatedAt: null };
  });

  async function loadEntries() {
    isLoading = true;
    errorMsg = '';
    try {
      const result = await GetAllEntries();
      entries = result || []; 
    } catch (err) {
      console.error("Error loading entries:", err);
      errorMsg = `Error loading entries: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  async function saveEntry() {
    isLoading = true;
    errorMsg = '';
    try {
      if (isEditing) {
        // await UpdateEntry(currentEntry); // Keep commented
        console.log("UpdateEntry call commented out");
      } else {
        // const { id, ...newEntryData } = currentEntry;
        // await CreateEntry(newEntryData); // Keep commented
        console.log("CreateEntry call commented out");
      }
      resetForm();
      await loadEntries(); 
    } catch (err) {
      console.error("Error saving entry (logic commented out):", err);
      errorMsg = `Error saving entry: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  async function generateContent() {
    if (!currentEntry.name || !currentEntry.type) {
      alert('Please provide both Name and Type for context before generating content.');
      return;
    }
    isGenerating = true;
    errorMsg = '';
    const prompt = `Generate a descriptive paragraph for a codex entry.\nName: ${currentEntry.name}\nType: ${currentEntry.type}\nExisting Content (if any): ${currentEntry.content || 'None'}`;
    try {
      const generated = await GenerateContent(prompt); 
      currentEntry.content = generated; 
    } catch (err) {
      console.error("Error generating content:", err);
      errorMsg = `Error generating content: ${err}`;
    } finally {
      isGenerating = false;
    }
  }

  function editEntry(entry) {
    isEditing = true;
    currentEntry = JSON.parse(JSON.stringify(entry)); // Use deep copy
  }

  async function deleteEntry(id) {
    isLoading = true;
    errorMsg = '';
    try {
      await DeleteEntry(id);
      await loadEntries(); 
      resetForm(); 
    } catch (err) {
      console.error("Error deleting entry:", err);
      errorMsg = `Error deleting entry: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  // Ensure resetForm assigns an object with the same structure
  function resetForm() {
      isEditing = false;
      currentEntry = { id: null, name: '', type: '', content: '', createdAt: null, updatedAt: null };
  }

  async function processStoryText() {
    if (!storyText.trim()) {
      processStoryErrorMsg = 'Please paste the story text into the textarea.';
      return;
    }
    isProcessingStory = true;
    processStoryErrorMsg = '';
    errorMsg = ''; // Clear other errors

    try {
      const createdEntries = await ProcessStory(storyText);
      console.log('Successfully processed story, created entries:', createdEntries);
      storyText = ''; // Clear the textarea on success
      await loadEntries(); // Refresh the codex list below
      // Optionally show a success message
    } catch (err) {
      console.error("Error processing story:", err);
      processStoryErrorMsg = `Error processing story: ${err}`;
    } finally {
      isProcessingStory = false;
    }
  }

  function selectEntry(entry) {
    selectedEntry = entry;
  }

  // Fetch the current database path from backend
  async function fetchCurrentDBPath() {
    try {
      currentDBPath = await GetCurrentDatabasePath();
    } catch (err) {
      errorMsg = `Error fetching current DB path: ${err}`;
      currentDBPath = 'Error loading path';
    }
  }

  // Function to handle selecting and loading a database file
  async function loadDatabase() {
    errorMsg = ''; // Clear previous errors
    try {
      const selectedPath = await SelectDatabaseFile();
      if (selectedPath) {
        console.log(`Selected DB file: ${selectedPath}`);
        await switchDatabase(selectedPath);
      } else {
        console.log('Database selection cancelled.');
      }
    } catch (err) {
      errorMsg = `Error selecting database file: ${err}`;
      console.error("Error selecting database:", err);
    }
  }

  // Function to handle saving the current database to a new file
  async function saveDatabaseAs() {
    errorMsg = ''; // Clear previous errors
    try {
      const newPath = await SaveDatabaseFile(); // Get the desired save path
      if (newPath) {
        console.log(`Attempting to copy database to: ${newPath}`);
        // Use the new backend function to copy the DB content
        await CopyDatabase(newPath); 
        console.log(`Database successfully copied to ${newPath}`);
        alert(`Database saved as ${newPath}`); // Feedback to user
      } else {
        console.log("Save As dialog cancelled.");
      }
    } catch (error) {
      errorMsg = `Error during Save As: ${error}`;
      console.error('Error during Save As:', error);
      alert(errorMsg); // Show error to user
    }
  }

  // Common function to switch database and refresh UI
  async function switchDatabase(newPath) {
    errorMsg = '';
    try {
      await SwitchDatabase(newPath); // Call backend function
      console.log(`Backend switched database to: ${newPath}`); 
      // Refresh UI after successful switch
      await fetchCurrentDBPath(); 
      await loadEntries();      
    } catch (err) {
      errorMsg = `Error switching database: ${err}`;
      console.error("Error switching database:", err);
      // Optionally revert path display if needed
      // await fetchCurrentDBPath(); 
    } 
  }

</script>

<main>
  <h1>Llore Codex</h1>

  <!-- Database Management Section -->
  <div class="db-manage">
    <button on:click={loadDatabase}>Load DB</button>
    <button on:click={saveDatabaseAs}>Save DB As...</button>
    <span class="db-path">Current DB: {currentDBPath}</span>
  </div>

  <!-- Story Processing Section -->
  <div class="story-processor">
    <h2>Process Story Text</h2>
    <p>Paste your story below. The AI will attempt to extract Characters, Locations, Items, and Lore and add them to the codex.</p>
    <textarea 
      rows="15" 
      placeholder="Paste your story here..."
      bind:value={storyText}
      disabled={isProcessingStory}
    ></textarea>
    <div class="form-actions">
      <button on:click={processStoryText} disabled={isProcessingStory || !storyText.trim()}>
        {#if isProcessingStory}Processing...{:else}Analyze Story & Create Entries{/if}
      </button>
    </div>
    {#if isProcessingStory}
      <p>Analyzing story, please wait...</p>
    {/if}
    {#if processStoryErrorMsg}
      <p class="error">{processStoryErrorMsg}</p>
    {/if}
  </div>

  <!-- Manual Entry Management Section -->
  <div class="form-container">
    <h2>Manual Codex Management</h2> 
    <form on:submit|preventDefault={saveEntry}>
      <input type="hidden" bind:value={currentEntry.id} />
      <div class="form-group">
        <label for="name">Name:</label>
        <input id="name" type="text" bind:value={currentEntry.name} required />
      </div>
      <div class="form-group">
        <label for="type">Type:</label>
        <input id="type" type="text" bind:value={currentEntry.type} placeholder="Character, Location, Item, Lore..." />
      </div>
      <div class="form-group">
        <label for="content">Content:</label>
        <button class="generate-btn" type="button" on:click={generateContent} disabled={isLoading || isGenerating}>
          {#if isGenerating}Generating...{:else}✨ Generate with AI{/if}
        </button>
        <textarea id="content" rows="5" bind:value={currentEntry.content}></textarea>
      </div>
      <div class="form-actions">
        <button type="submit" disabled={isLoading}>{isEditing ? 'Update' : 'Create'}</button>
        {#if isEditing}
          <button type="button" on:click={resetForm} disabled={isLoading}>Cancel Edit</button>
        {/if}
      </div>
    </form>
  </div>

  <!-- Display Loading/Error Messages -->
  {#if isLoading}
    <p>Loading...</p>
  {/if}
  {#if errorMsg}
    <p class="error">{errorMsg}</p>
  {/if}

  <!-- List of Entries -->
  <div class="entries-list">
    <h2>Codex Entries</h2>
    {#if entries.length === 0 && !isLoading}
      <p>No entries found. Create one above!</p>
    {/if}
    <ul class="entry-list">
      {#each entries as entry (entry.id)}
        <li class:selected={selectedEntry && selectedEntry.id === entry.id}>
          <button type="button" class="entry-select-button" on:click={() => selectEntry(entry)}>
            <strong>{entry.name}</strong> ({entry.type})
          </button>
          <div class="entry-actions">
            <button on:click={() => editEntry(entry)} disabled={isLoading}>Edit</button>
            <button on:click={() => deleteEntry(entry.id)} disabled={isLoading}>Delete</button>
          </div>
        </li>
      {/each}
    </ul>
  </div>

  <!-- Selected Entry Details -->
  {#if selectedEntry}
    <div class="entry-details">
      <h3>{selectedEntry.name}</h3>
      <p><strong>Type:</strong> {selectedEntry.type}</p>
      <p><strong>Description:</strong></p>
      <pre>{selectedEntry.content}</pre>
    </div>
  {:else}
    <p>Select an entry from the list to view its details.</p>
  {/if}

</main>

<style>
  main {
    max-width: 800px;
    margin: 2rem auto;
    padding: 1rem;
    font-family: sans-serif;
  }

  h1, h2 {
    color: #333;
    text-align: center;
    margin-bottom: 1.5rem;
  }

  .form-container,
  .entries-list,
  .story-processor {
    background-color: #f9f9f9;
    padding: 1.5rem;
    border-radius: 5px;
    margin-bottom: 2rem;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
  }

  .form-group {
    margin-bottom: 1rem;
  }

  .form-group label {
    display: block;
    margin-bottom: 0.3rem;
    font-weight: bold;
  }

  .form-group input[type="text"],
  .form-group textarea,
  .story-processor textarea {
    width: 100%;
    padding: 0.5rem;
    border: 1px solid #ccc;
    border-radius: 3px;
    box-sizing: border-box; 
  }

  textarea {
    resize: vertical;
  }

  .form-actions {
    margin-top: 1.5rem;
    text-align: right;
  }

  button {
    padding: 0.6rem 1.2rem;
    background-color: #007bff;
    color: white;
    border: none;
    border-radius: 3px;
    cursor: pointer;
    margin-left: 0.5rem;
    transition: background-color 0.2s ease;
  }

  button:hover:not(:disabled) {
    background-color: #0056b3;
  }

  button:disabled {
    background-color: #ccc;
    cursor: not-allowed;
  }

  button[type="button"] {
    background-color: #6c757d;
  }

  button[type="button"]:hover:not(:disabled) {
    background-color: #5a6268;
  }

  ul {
    list-style: none;
    padding: 0;
  }

  li {
    background-color: #fff;
    padding: 1rem;
    border: 1px solid #eee;
    border-radius: 3px;
    margin-bottom: 0.8rem;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  li strong {
    margin-right: 0.5rem;
  }

  .entry-actions button {
    padding: 0.3rem 0.6rem;
    font-size: 0.9em;
  }

  .error {
    color: red;
    font-weight: bold;
    margin-top: 1rem;
    text-align: center;
  }

  .form-group label + .generate-btn {
      display: inline-block;
      vertical-align: middle;
      margin-left: 10px;
      margin-bottom: 0.3rem; 
      padding: 0.2rem 0.5rem;
      font-size: 0.8em;
      background-color: #ffc107; 
      color: #333;
      border: none;
      border-radius: 3px;
      cursor: pointer;
  }
  .form-group label + .generate-btn:hover:not(:disabled) {
      background-color: #e0a800;
  }
  .form-group label + .generate-btn:disabled {
      background-color: #ccc;
      cursor: not-allowed;
  }

  .entry-details {
    margin-top: 1rem;
    padding: 1rem;
    background-color: #fff;
    border: 1px solid #ddd;
    border-radius: 4px;
  }

  .entry-details pre {
    white-space: pre-wrap; /* Wrap long lines in description */
    word-wrap: break-word;
    background-color: #f8f8f8;
    padding: 0.5rem;
    border-radius: 4px;
    max-height: 300px; /* Scroll long descriptions */
    overflow-y: auto;
  }

  .entry-list li.selected {
    background-color: #e0e0ff; 
    font-weight: bold;
  }

  .entry-select-button {
    background: none;
    border: none;
    padding: 0;
    margin: 0; /* Remove default margins */
    font: inherit; /* Inherit font styles from li */
    color: inherit; /* Inherit text color */
    cursor: pointer;
    text-align: left; /* Align text to the left */
    width: auto; /* Adjust width as needed or keep auto */
    display: inline; /* Or block/inline-block as needed */
  }

  .entry-select-button:hover,
  .entry-select-button:focus {
    text-decoration: underline; /* Indicate interactivity */
    outline: none; /* Or custom focus style */
  }

  .db-manage {
    margin-bottom: 20px;
    padding-bottom: 10px;
    border-bottom: 1px solid #ccc;
  }

  .db-manage button {
    margin-right: 10px;
  }

  .db-path {
    font-style: italic;
    color: #555;
    margin-left: 15px;
  }
</style>
