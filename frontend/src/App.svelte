<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { 
    GetAllEntries, 
    CreateEntry, 
    UpdateEntry, 
    DeleteEntry, 
    GenerateOpenRouterContent, 
    SaveOpenRouterApiKey, 
    SelectVaultFolder, 
    CreateNewVault, 
    SwitchVault, 
    GetCurrentVaultPath, 
    ListLibraryFiles, 
    ImportStoryTextAndFile, 
    ReadLibraryFile, 
    ProcessStory, 
    ProcessAndSaveTextAsEntries,
    ListChatLogs, 
    LoadChatLog, 
    SaveChatLog 
  } from '@wailsjs/go/main/App';
  import type { main } from '@wailsjs/go/models';

  // --- OpenRouter API Key UI State ---
  let showApiKeyModal = false;
  let openrouterApiKey = '';
  let apiKeySaveMsg = '';
  let apiKeyErrorMsg = '';

  // --- Model Selector State ---
  import { FetchOpenRouterModels } from '@wailsjs/go/main/App';
  let modelList: { id: string, name: string }[] = [];
  let selectedModel: string = '';
  let isModelListLoading = false;
  let modelListError = '';

  async function loadModelList() {
    isModelListLoading = true;
    modelListError = '';
    try {
      modelList = await FetchOpenRouterModels();
      selectedModel = modelList.length > 0 ? modelList[0].id : '';
    } catch (err) {
      modelListError = 'Failed to load models: ' + err;
      modelList = [];
      selectedModel = '';
    } finally {
      isModelListLoading = false;
    }
  }

  async function saveApiKey() {
    apiKeySaveMsg = '';
    apiKeyErrorMsg = '';
    try {
      // Save to config.json via backend Go function
      await SaveOpenRouterApiKey(openrouterApiKey);
      apiKeySaveMsg = 'API key saved!';
      showApiKeyModal = false;
      await loadModelList(); // Refresh model list after saving key
    } catch (err) {
      apiKeyErrorMsg = 'Failed to save API key: ' + err;
    }
  }

  function openApiKeyModal() {
    showApiKeyModal = true;
    apiKeySaveMsg = '';
    apiKeyErrorMsg = '';
    openrouterApiKey = '';
  }


  // State Variables
  let entries: main.Entry[] = [];
  let currentEntry: Partial<main.Entry> | null = null;
  let isLoading = false;
  let isEditing = false;
  let isGenerating = false;
  let errorMsg = '';
  let initialErrorMsg = ''; // Declare the missing variable
  let vaultErrorMsg = ''; // Keep this one too, might be used elsewhere

  // Vault State
  let vaultIsReady = false;
  let currentVaultPath: string | null = null;

  // Mode ('codex', 'story', 'library', 'chat', or null for choice screen)
  let mode: 'codex' | 'story' | 'library' | 'chat' | null = null; 

  // Library State (Files)
  let libraryFiles: string[] = []; 
  let isLibraryLoading = false;

  // Story Processing State
  let storyText = '';
  let isProcessingStory = false;
  let processStoryErrorMsg = '';

  // Chat State
  let chatInput = '';
  let isChatLoading = false;
  let chatError = '';
  let chatDisplayElement: HTMLDivElement;

  // --- Chat Log Selection State ---
  let chatMessages: { sender: 'user' | 'ai', text: string }[] = []; // Re-add declaration
  let showChatSelection = false; // Controls the visibility of the selection UI
  let availableChatLogs: string[] = []; // List of filenames like "chat_2024-04-27.json"
  let currentChatLogFilename: string | null = null; // Track the loaded/saved log file
  let isLoadingChatLogs = false;
  let chatLogError = '';

  // --- Save New Chat State ---
  let showSaveChatModal = false;
  let newChatFilename = '';
  let saveChatError = '';

  // Story import feedback
  let showImportModal = false;
  let createdEntriesCount = 0;
  let processedEntries: main.Entry[] = []; 

  // Helper: Refresh Library Files 
  async function refreshLibraryFiles() {
    if (!vaultIsReady) return;
    isLibraryLoading = true;
    errorMsg = ''; 
    try {
      libraryFiles = await ListLibraryFiles() || []; 
    } catch (err) {
      console.error("Error loading library files:", err);
      errorMsg = `Error loading library: ${err}`;
      libraryFiles = []; 
    } finally {
      isLibraryLoading = false;
    }
  }

  // Track if chat context has been injected
  let chatContextInjected = false;

  // Helper: Lore Chat send
  async function sendChat() {
    if (!chatInput.trim()) return;
    chatError = '';
    isChatLoading = true;
    chatMessages = [...chatMessages, { sender: 'user', text: chatInput }];
    let prompt = chatInput;
    chatInput = '';
    try {
      // --- Context Injection Logic (Modified) ---
      let currentPrompt = prompt; // Use a local var for the potentially modified prompt
      if (!chatContextInjected) {
        let codexEntries = await GetAllEntries();
        let contextString = '';
        if (codexEntries && codexEntries.length > 0) {
          contextString = codexEntries.map(e => `Name: ${e.name}\nType: ${e.type}\nContent: ${e.content}`)
            .join('\n---\n');
          if (contextString.length > 4000) contextString = contextString.slice(0, 4000) + '\n...';
        }
        if (contextString) {
          currentPrompt = `Context:\n${contextString}\n\nUser: ${prompt}`;
        }
        chatContextInjected = true; // Mark context as injected for this session
      }

      // --- Call LLM ---
      const aiReply = await GenerateOpenRouterContent(currentPrompt, selectedModel);
      const newAiMessage = { sender: 'ai' as const, text: aiReply };
      chatMessages = [...chatMessages, newAiMessage];

      // --- Auto-Save Logic ---
      if (currentChatLogFilename) {
        // If a log is loaded, save the updated messages back to the same file
        await SaveChatLog(currentChatLogFilename, chatMessages);
      } else {
        // TODO: Maybe add a prompt to save the chat if currentChatLogFilename is null?
      }
    } catch (err) {
      chatError = `AI error: ${err}`;
    } finally {
      isChatLoading = false;
    }
  }

  // Helper: Save AI chat turn to codex
  async function saveChatToCodex(text: string) { 
    try {
      const potentialEntries = await ProcessStory(text);
      console.log(`ProcessStory returned ${potentialEntries ? potentialEntries.length : 0} potential entries`);
      
      if (!potentialEntries || potentialEntries.length === 0) {
        alert('AI processing did not extract any structured entries from the chat response.');
        return;
      }
      
      // Process each entry returned by ProcessStory
      let processedCount = 0;
      let errorMessages = [];
      
      for (const entry of potentialEntries) {
        try {
          // Basic validation
          if (!entry.name || !entry.type) {
            console.warn("Skipping entry with missing name or type:", entry);
            continue;
          }
          
          console.log(`Creating entry: ${entry.name} (${entry.type})`);
          await CreateEntry(entry.name, entry.type, entry.content);
          processedCount++;
        } catch (entryError) {
          console.error(`Error saving entry "${entry.name}":`, entryError);
          errorMessages.push(`${entry.name}: ${entryError}`);
        }
      }
      
      // Report results
      if (processedCount > 0) {
        alert(`Processed ${processedCount} entries from the chat response.`);
        await loadEntries(); // Refresh the entries list
      } else {
        alert('No entries could be saved to the Codex.');
      }
      
      if (errorMessages.length > 0) {
        console.error("Errors during codex entry saving:", errorMessages);
      }
    } catch (err) {
      console.error("Error processing chat for codex:", err);
      alert(`Error processing chat: ${err}`);
    }
  }

  onMount(async () => {
    await fetchCurrentVaultPath(); 
    if (currentVaultPath) {
      vaultIsReady = true;
      // Don't load codex/library/chat immediately, wait for user mode selection
      // await loadEntries(); 
      // refreshLibraryFiles(); 
    } else {
      mode = null; // Show vault selection if no path
    }
    resetForm();
    isEditing = false;
    currentEntry = { id: null, name: '', type: '', content: '', createdAt: null, updatedAt: null };
    await loadModelList();
  });

  async function loadEntries() {
    if (!vaultIsReady) return; 
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

  function handleEntrySelect(entry: main.Entry) {
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
        const updatePayload = { ...currentEntry, id: currentEntry.id!, createdAt: currentEntry.createdAt!, updatedAt: currentEntry.updatedAt! };
        await UpdateEntry(updatePayload); 
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
        await loadEntries(); 
        const newEntryInList = entries.find(e => e.id === newEntry.id);
        if (newEntryInList) {
          handleEntrySelect(newEntryInList);
        } else {
          resetForm(); 
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
    if (!selectedModel) {
        errorMsg = 'Please select an AI model from the settings first.';
        return;
    }
    isGenerating = true;
    errorMsg = '';

    // Construct a prompt for OpenRouter
    // Example: Ask it to expand or elaborate on the existing entry
    // TODO: Make this prompt more sophisticated or user-configurable
    let prompt = `Expand on the following codex entry. Provide more details, background, or connections based on its name, type, and existing content.\n\nName: ${currentEntry.name}\nType: ${currentEntry.type}\nContent: ${currentEntry.content || '(empty)'}`;

    try {
      console.log(`Generating content for entry '${currentEntry.name}' using model ${selectedModel}`);
      const generated = await GenerateOpenRouterContent(prompt, selectedModel);
      // Update the content of the current entry
      currentEntry = { ...currentEntry, content: generated };
      console.log(`Generated content received: ${generated.substring(0, 100)}...`);
    } catch (err) {
      console.error("Error generating content:", err);
      errorMsg = `Error generating content: ${err}`;
    } finally {
      isGenerating = false;
    }
  }

  // Renamed from handleProcessStory -> handleImportStory
  async function handleImportStory() { 
    if (!storyText.trim()) {
      processStoryErrorMsg = 'Please paste the story text into the textarea.';
      return;
    }
    if (!vaultIsReady) {
      processStoryErrorMsg = 'No Lore Vault is currently loaded.';
      return;
    }

    isProcessingStory = true;
    processStoryErrorMsg = '';
    processedEntries = []; 
    try {
      const newEntriesResult = await ImportStoryTextAndFile(storyText);
      
      processedEntries = newEntriesResult || []; 
      createdEntriesCount = processedEntries.length; 
      
      await loadEntries(); 
      refreshLibraryFiles(); 
      showImportModal = true; 
      // Story text is cleared when modal closes
    } catch (err) {
      console.error("Error importing story:", err);
      processStoryErrorMsg = `Failed to import story: ${err}`;
    } finally {
      isProcessingStory = false;
    }
  }

  // Function to potentially view library file content
  async function viewLibraryFileContent(filename: string) {
    if (!vaultIsReady) return;
    alert(`Viewing file (Not Implemented): ${filename}\nNeed ReadLibraryFile Go function.`);
    // try {
    //   const content = await ReadLibraryFile(filename);
    //   // Display content in a modal or dedicated view
    //   console.log(`Content of ${filename}:\n`, content);
    //   alert(`Content of ${filename}:\n${content.substring(0, 200)}...`);
    // } catch (err) {
    //   errorMsg = `Failed to read library file ${filename}: ${err}`;
    // }
  }

  // Renamed from fetchCurrentDBPath
  async function fetchCurrentVaultPath() {
    try {
      currentVaultPath = await GetCurrentVaultPath();
      vaultIsReady = !!currentVaultPath; 
    } catch (err) {
      console.warn("Could not get current vault path:", err); 
      currentVaultPath = null;
      vaultIsReady = false;
    }
  }

  // Renamed from handleCreateNew
  async function handleNewLore() {
    try {
      let vaultName = prompt('Enter a name for your new Lore Vault:', 'LoreVault');
      if (!vaultName) {
        vaultErrorMsg = 'Vault creation cancelled.';
        return;
      }
      const newVaultPath = await CreateNewVault(vaultName);
      if (newVaultPath) {
        await SwitchVault(newVaultPath);
        vaultIsReady = true;
        await updateCurrentVaultPath();
        await loadEntries();
        refreshLibraryFiles();
        vaultErrorMsg = '';
      } else {
        vaultErrorMsg = 'Vault creation was cancelled or failed.';
      }
    } catch (err) {
      vaultErrorMsg = `Error creating new vault: ${err}`;
      vaultIsReady = false;
    }
  }

  // Renamed from handleLoadExisting
  async function handleLoadLore() {
    try {
      const selectedPath = await SelectVaultFolder(); 
      if (selectedPath) {
        await SwitchVault(selectedPath); 
        vaultIsReady = true;
        await updateCurrentVaultPath();
        await loadEntries();
        refreshLibraryFiles();
        vaultErrorMsg = '';
      } else {
        vaultErrorMsg = ''; 
      }
    } catch (err) {
      vaultErrorMsg = `Error loading vault: ${err}`;
      vaultIsReady = false;
    }
  }

  // Renamed from updateCurrentDBPath
  async function updateCurrentVaultPath() {
    try {
      currentVaultPath = await GetCurrentVaultPath();
    } catch (err) {
      currentVaultPath = "Error loading path";
    }
  }

  // Global error handler
  function handleError(message: string | Event, source?: string, lineno?: number, colno?: number, error?: Error) {
    console.error('Global error caught:', message, source, lineno, colno, error);
    vaultErrorMsg = `An application error occurred: ${message}${error ? ' (' + error.message + ')' : ''}. Please check console for details.`;
    return true; 
  }
  window.onerror = handleError;

  // Handler for keyboard navigation in entry list
  function createKeyDownHandler(entry: main.Entry) {
    return (event: KeyboardEvent) => {
      if (event.key === 'Enter' || event.key === ' ') {
        event.preventDefault(); // Prevent scrolling on Space
        handleEntrySelect(entry);
      }
      // Add ArrowUp/ArrowDown logic here if needed for list navigation
    };
  }

  // Close the import success modal
  function closeImportModal(switchToCodex: boolean) {
    showImportModal = false;
    if (switchToCodex) {
      mode = 'codex'; // Optionally switch view after closing
    }
  }

  // --- Chat Log Selection Logic ---
  async function initiateChatSelection() {
    console.log('initiateChatSelection called'); // Log entry
    if (!vaultIsReady) {
      chatLogError = 'Cannot start chat: No vault loaded.';
      console.log('initiateChatSelection aborted: Vault not ready');
      return;
    }
    isLoadingChatLogs = true;
    showChatSelection = true; // Show the selection UI/modal
    console.log(`initiateChatSelection set showChatSelection: ${showChatSelection}`); // Log set true
    chatLogError = '';
    availableChatLogs = [];
    try {
      availableChatLogs = await ListChatLogs() || [];
      console.log('Chat logs fetched:', availableChatLogs); // Log fetched logs
    } catch (err) {
      console.error('Error loading chat logs:', err); // Log error
      chatLogError = `Error loading chat logs: ${err}`;
    } finally {
      isLoadingChatLogs = false;
      console.log(`initiateChatSelection finished. showChatSelection: ${showChatSelection}`); // Log finish
    }
  }

  function startNewChat() {
    chatMessages = [];
    currentChatLogFilename = null; // Indicate it's a new, unsaved chat
    chatContextInjected = false;
    showChatSelection = false; // Hide selection UI
    chatError = '';
  }

  async function loadSelectedChat(filename: string) {
    if (!filename) return;
    isChatLoading = true;
    showChatSelection = false; // Hide selection UI
    chatError = '';
    try {
      const loadedMessages = await LoadChatLog(filename) || [];
      // Need to ensure the loaded data structure matches { sender: string, text: string }
      chatMessages = loadedMessages.map(msg => ({ sender: msg.sender as ('user' | 'ai'), text: msg.text }));
      currentChatLogFilename = filename;
      chatContextInjected = true; // Assume context was injected when log was saved
    } catch (err) {
      chatError = `Error loading chat log '${filename}': ${err}`;
      chatMessages = [];
      currentChatLogFilename = null;
    } finally {
      isChatLoading = false;
    }
  }

  // --- Save New Chat Logic ---
  function promptToSaveChat() {
    if (chatMessages.length === 0) {
      alert("Nothing to save.");
      return;
    }
    // Suggest a default filename based on date
    const today = new Date();
    const dateStr = today.toISOString().split('T')[0]; // YYYY-MM-DD
    newChatFilename = `Chat ${dateStr}.json`; 
    saveChatError = '';
    showSaveChatModal = true;
  }

  async function saveNewChat() {
    if (!newChatFilename.trim()) {
      saveChatError = "Filename cannot be empty.";
      return;
    }
    let filenameToSave = newChatFilename.trim();
    if (!filenameToSave.toLowerCase().endsWith('.json')) {
        filenameToSave += '.json';
    }

    saveChatError = '';
    isChatLoading = true; // Reuse loading indicator
    try {
      await SaveChatLog(filenameToSave, chatMessages);
      currentChatLogFilename = filenameToSave; // Start auto-saving to this file now
      showSaveChatModal = false; // Close modal on success
      // Optionally refresh the list of logs if the selection screen is still accessible
      if (showChatSelection) {
        await initiateChatSelection(); 
      }
    } catch (err) {
      saveChatError = `Failed to save chat: ${err}`;
    } finally {
      isChatLoading = false;
    }
  }

  async function setMode(newMode: 'codex' | 'story' | 'library' | 'chat' | null) {
    console.log(`setMode called with: ${newMode}, current mode: ${mode}`); // Log entry
    if (newMode !== mode) { // Only run if mode changes
      console.log(`Mode changing from ${mode} to ${newMode}`); // Log change
      mode = newMode;
      errorMsg = ''; // Clear general errors on mode change
 
      // Handle actions specific to the new mode
      if (newMode === 'codex') {
        console.log('Handling mode: codex');
        await loadEntries();
      } else if (newMode === 'library') {
        console.log('Handling mode: library');
        await refreshLibraryFiles();
      } else if (newMode === 'chat') {
        console.log('Resetting chat state for selection...'); // Log chat branch
        chatMessages = [];
        currentChatLogFilename = null;
        chatContextInjected = false;
        chatError = '';
        await initiateChatSelection(); // This should set showChatSelection = true
        console.log(`setMode finished for chat. showChatSelection: ${showChatSelection}`); // Log after initiate
      } else if (newMode === 'story') {
        console.log('Handling mode: story');
        storyText = '';
        processStoryErrorMsg = '';
      } else if (newMode === null) {
        console.log('Handling mode: null (Vault selection)');
      }
    } else {
      console.log(`Mode ${newMode} is already active.`); // Log no change
    }
  }

</script>

{#if !vaultIsReady} <!-- Vault is NOT ready, show initial screen FIRST -->
  <div class="initial-prompt">
    <h1>Welcome to Llore</h1>
    <p>Load an existing Lore Vault or create a new one.</p>
    {#if initialErrorMsg}
      <p class="error-message">{initialErrorMsg}</p>
    {/if}
    <button on:click={handleLoadLore} disabled={isLoading}> 
        {#if isLoading && !vaultIsReady}Loading...{:else}Load Lore Vault{/if}
    </button>
    <button on:click={handleNewLore} disabled={isLoading}>
        {#if isLoading && !vaultIsReady}Creating...{:else}Create New Vault{/if} 
    </button>
  </div>
{:else if mode === null} <!-- Vault IS ready, but no mode selected -->
  <!-- Mode Choice Screen -->
  <div class="mode-choice">
    <h2>Choose a mode</h2>
    <button on:click={() => setMode('codex')}>Codex</button>
    <button on:click={() => setMode('story')}>Story Import</button>
    <button on:click={() => setMode('library')}>Library</button>
    <button on:click={() => setMode('chat')}>Lore Chat</button>
  </div>
{:else if mode === 'codex'}
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>

  <main>
    <h1>Llore Codex</h1>

    <div class="db-path-display">
      Current Vault: {currentVaultPath || 'None loaded'}
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
            <li class:selected={currentEntry?.id === entry.id}>
              <div 
                class="entry-item-button"
                role="button"
                tabindex="0"
                on:click={() => handleEntrySelect(entry)} 
                on:keydown={createKeyDownHandler(entry)}
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
              <small>Created: {formatDate(currentEntry.createdAt)} | Updated: {formatDate(currentEntry.updatedAt)}</small>
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

    </div> 

  </main>
{:else if mode === 'story'} 
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>
  <section class="story-processor">
    <h2>Import New Story</h2>
    <p>Paste story text below. It will be saved as a new file in the vault's Library and processed for codex entries.</p>
    <textarea 
      bind:value={storyText} 
      rows="15" 
      placeholder="Paste your story text here..."
      disabled={isProcessingStory}
    ></textarea>
    <button on:click={handleImportStory} disabled={isProcessingStory || !storyText.trim()}>
      {#if isProcessingStory}Processing...{:else}Import Story & Add Entries{/if}
    </button>
    {#if processStoryErrorMsg}
      <p class="error-message">{processStoryErrorMsg}</p>
    {/if}
  </section>
{:else if mode === 'library'}
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>
  <section>
    <h2>Library</h2>
    <button on:click={refreshLibraryFiles} disabled={isLibraryLoading}>
      {#if isLibraryLoading}Loading...{:else}Refresh Library{/if}
    </button>

    {#if isLibraryLoading}
      <p>Loading library files...</p>
    {:else}
      <!-- Not loading -->
      {#if errorMsg}
        <p class="error-message">{errorMsg}</p>
      {:else}
        <!-- Not loading and no error -->
        {#if libraryFiles.length === 0}
          <p>No files found in the vault's Library folder.</p>
        {:else}
          <!-- Not loading, no error, and files exist -->
          <ul>
            {#each libraryFiles as filename (filename)}
              <li>
                {filename}
                <button on:click={() => viewLibraryFileContent(filename)} style="margin-left: 10px; font-size: 0.8em;">View</button>
              </li>
            {/each}
          </ul>
        {/if} <!-- End check file list -->
      {/if} <!-- End check error -->
    {/if} <!-- End check loading -->

  </section>
{:else if mode === 'chat'}
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>
  {#if showChatSelection}
    <section class="chat-log-selection">
      <h2>Select a Chat Log</h2>
      <button on:click={startNewChat}>Start New Chat</button>
      {#if isLoadingChatLogs}
        <p>Loading chat logs...</p>
      {:else if chatLogError}
        <p class="error-message">{chatLogError}</p>
      {:else}
        <ul>
          {#each availableChatLogs as filename (filename)}
            <li>
              <button on:click={() => loadSelectedChat(filename)}>{filename}</button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {:else}
    <section class="lore-chat">
      <h2>Lore Chat</h2>
      <div class="chat-settings-row">
        <label for="model-select">Model:</label>
        {#if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span style="color:red">{modelListError}</span>
        {:else}
          <select id="model-select" bind:value={selectedModel}>
            {#each modelList as model}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
        <button on:click={openApiKeyModal} style="margin-left: 1em;">Set API Key</button>
        {#if !currentChatLogFilename && chatMessages.length > 0}
          <button on:click={promptToSaveChat} style="margin-left: 1em;">Save Chat As...</button>
        {/if}
      </div>
      <div class="chat-display" bind:this={chatDisplayElement}>
        {#each chatMessages as message}
          <div class="message {message.sender}">
            <strong>{message.sender === 'user' ? 'You' : 'AI'}:</strong> {message.text}
            {#if message.sender === 'ai'}
              <button on:click={() => saveChatToCodex(message.text)}>Save to Codex</button>
            {/if}
          </div>
        {/each}
        {#if isChatLoading}
          <div class="chat-ai"><em>AI is thinking...</em></div>
        {/if}
      </div>
      <form on:submit|preventDefault={sendChat} class="chat-form">
        <input type="text" bind:value={chatInput} placeholder="Ask about your lore..." disabled={isChatLoading}>
        <button type="submit" disabled={isChatLoading || !chatInput.trim()}>Send</button>
      </form>
      {#if chatError}
        <p class="error-message">{chatError}</p>
      {/if}
    </section>
  {/if}
{/if} <!-- This NOW closes the entire chain starting with #if !vaultIsReady -->

{#if showSaveChatModal}
  <div class="modal-backdrop">
    <div class="modal save-chat-modal">
      <h3>Save New Chat Log</h3>
      <label for="chat-filename">Filename:</label>
      <input id="chat-filename" type="text" bind:value={newChatFilename} placeholder="e.g., Chat with Claude.json">
      {#if saveChatError}
        <p class="error-message">{saveChatError}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={saveNewChat} disabled={isChatLoading || !newChatFilename.trim()}>Save</button>
        <button on:click={() => showSaveChatModal = false} disabled={isChatLoading}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

{#if showApiKeyModal}
  <div class="modal-backdrop">
    <div class="modal api-key-modal">
      <h3>Set OpenRouter API Key</h3>
      <input type="text" bind:value={openrouterApiKey} placeholder="sk-..." style="width: 100%; padding: 0.5em; margin-bottom: 1em;" />
      <button on:click={saveApiKey} style="margin-right: 1em;">Save</button>
      <button on:click={() => showApiKeyModal = false}>Cancel</button>
      {#if apiKeySaveMsg}
        <p style="color: green;">{apiKeySaveMsg}</p>
      {/if}
      {#if apiKeyErrorMsg}
        <p class="error-message">{apiKeyErrorMsg}</p>
      {/if}
    </div>
  </div>
{/if}

{#if showImportModal}
  <div class="modal-backdrop">
    <div class="modal-content">
      <h3>Story Processed</h3>
      {#if createdEntriesCount > 0}
        <p>{createdEntriesCount} new codex entr{createdEntriesCount === 1 ? 'y was' : 'ies were'} created:</p>
        <ul>
          {#each processedEntries as entry}
            <li>{entry.name} ({entry.type})</li>
          {/each}
        </ul>
      {:else}
        <p>No new entries were created from the story.</p>
      {/if}
      <div class="modal-actions">
        <button on:click={() => closeImportModal(false)}>OK</button>
        <button on:click={() => closeImportModal(true)}>Go to Codex</button>
      </div>
    </div>
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
    padding: 0; 
    cursor: default; 
    border-bottom: 1px solid #eee;
  }
  .sidebar li.selected .entry-item-button {
    background-color: #e0e0ff; 
    font-weight: bold;
  }
  .entry-item-button {
    display: block; 
    padding: 0.5rem; 
    cursor: pointer;
    transition: background-color 0.2s;
    outline: none; 
  }
  .entry-item-button:hover {
    background-color: #f0f0f0;
  }
  .entry-item-button:focus {
    outline: 2px solid blue; 
    outline-offset: -2px; 
    background-color: #e8e8ff; 
  }
  .main-content {
    flex: 1; 
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
  .lore-chat {
    max-width: 600px;
    margin: 0 auto;
  }
  .chat-window {
    border: 1px solid #ccc;
    background: #fafaff;
    padding: 1rem;
    min-height: 200px;
    max-height: 300px;
    overflow-y: auto;
    margin-bottom: 1rem;
  }
  .chat-user {
    text-align: right;
    margin-bottom: 0.5rem;
  }
  .chat-ai {
    text-align: left;
    margin-bottom: 0.5rem;
  }
  .chat-settings-row {
    display: flex;
    align-items: center;
    gap: 1em;
    margin-bottom: 1em;
  }
  .chat-form {
    display: flex;
    gap: 0.5rem;
  }

  /* Modal styles */
  .modal-backdrop {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    background: rgba(0,0,0,0.4);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }
  .modal-content {
    background: white;
    color: #222;
    border-radius: 8px;
    padding: 2rem;
    min-width: 300px;
    text-align: center;
    box-shadow: 0 2px 16px rgba(0,0,0,0.2);
  }

  /* Vault Selection Screen */
  .initial-prompt {
    max-width: 500px;
    margin: 5rem auto;
    padding: 2rem;
    text-align: center;
    background: #f9f9f9;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0,0,0,0.1);
  }

  .initial-prompt h1 {
    margin-bottom: 1rem;
  }

  .initial-prompt p {
    margin-bottom: 1.5rem;
    color: #555;
  }

  /* Adjusted styles for clarity */
  .db-path-display {
    margin-bottom: 1rem;
    padding: 0.5rem;
    background-color: #eee;
    border-radius: 4px;
    font-size: 0.9em;
    color: #333;
  }

  /* Make sure library list items are distinct */
  .library-section ul li {
    padding: 0.5rem;
    border-bottom: 1px solid #eee;
    display: flex; 
    justify-content: space-between; 
    align-items: center;
  }

  .library-section ul li:last-child {
    border-bottom: none;
  }
</style>
