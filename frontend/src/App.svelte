<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { database, llm } from '@wailsjs/go/models'; // Import namespaces
  import LibraryFileViewer from './LibraryFileViewer.svelte';
  import logo from './assets/images/logo.png';
  import {
    GetAllEntries,
    CreateEntry,
    UpdateEntry,
    DeleteEntry,
    GenerateOpenRouterContent,
    SelectVaultFolder,
    CreateNewVault,
    SwitchVault,
    GetCurrentVaultPath,
    ListLibraryFiles,
    ImportStoryTextAndFile,
    ReadLibraryFile,
    SaveLibraryFile,
    ProcessStory,
    ProcessAndSaveTextAsEntries,
    ListChatLogs,
    LoadChatLog,
    SaveChatLog,
    FetchOpenRouterModelsWithKey,
    GetSettings,
    SaveSettings,
    SaveAPIKeyOnly
  } from '@wailsjs/go/main/App';

  // --- OpenRouter API Key UI State ---
  let showApiKeyModal = false;
  let openrouterApiKey = '';
  let selectedChatModel = '';
  let showApiKey = false;
  let apiKeyErrorMsg = '';
  let apiKeySaveMsg = '';

  // --- Model Selector State ---
  let modelList: llm.OpenRouterModel[] = []; // Use llm.OpenRouterModel
  let selectedModel: string = '';
  let isModelListLoading = false;
  let modelListError = '';

  async function loadModelList() {
    // Don't try to load if API key isn't set (backend also checks, but good for UX)
    if (!openrouterApiKey) {
      console.log("API key not set, skipping model list load.");
      modelListError = 'Set OpenRouter API Key first to load models.';
      modelList = [];
      selectedModel = ''; // Ensure no model is selected if list can't load
      return; 
    }
    console.log("Attempting to load models using key...")
    isModelListLoading = true;
    modelListError = '';
    try {
      // Call the backend function that accepts the key
      const fetchedModels: llm.OpenRouterModel[] = await FetchOpenRouterModelsWithKey(openrouterApiKey); // Use llm.OpenRouterModel
      modelList = fetchedModels || [];
      selectedModel = modelList.length > 0 ? modelList[0].id : ''; // Default to first model if list loaded
      console.log(`Fetched ${modelList.length} models.`);
    } catch (err) {
      console.error("Error fetching models:", err);
      modelListError = 'Failed to load models: ' + err;
      modelList = [];
      selectedModel = '';
    } finally {
      isModelListLoading = false;
    }
  }

  async function saveApiKey() {
    apiKeyErrorMsg = '';
    // Reverting to the simpler save flow specifically for the chat modal
    isLoading = true; // Indicate loading while saving from modal
    try {
      console.log("Saving API key via SaveAPIKeyOnly...");
      await SaveAPIKeyOnly(openrouterApiKey); // Call the simpler backend function
      apiKeySaveMsg = 'API key saved!';
      console.log("API key saved via SaveAPIKeyOnly.");
      showApiKeyModal = false;
      await loadModelList(); // Refresh model list after saving key
    } catch (err) {
      apiKeyErrorMsg = 'Failed to save API key: ' + err;
      console.error("API key save error (SaveAPIKeyOnly):", err);
    } finally {
      isLoading = false;
    }
  }

  function openApiKeyModal() {
    showApiKeyModal = true;
    apiKeySaveMsg = '';
    apiKeyErrorMsg = '';
    openrouterApiKey = '';
  }

  // State Variables
  let entries: database.CodexEntry[] = []; // Use database.CodexEntry
  // Initialize with default values matching the type, not null
  let currentEntry: database.CodexEntry = { id: 0, name: '', type: '', content: '', createdAt: '', updatedAt: '' };
  let isLoading = false;
  let isEditing = false; // This will control if we show "Edit" or "Create"
  let isGenerating = false;
  let errorMsg = '';
  let initialErrorMsg = ''; 
  let vaultErrorMsg = ''; 

  // Settings State
  let settingsErrorMsg = '';
  let settingsSaveMsg = '';
  let chatModelId: string = ''; 
  let storyProcessingModelId: string = '';

  interface OpenRouterConfig {
    openrouter_api_key: string;
    chat_model_id: string;
    story_processing_model_id: string;
  }

  // Vault State
  let vaultIsReady = false;
  let currentVaultPath: string | null = null;

  // Library viewer state
  let showLibraryViewer = false;
  let viewingFilename = '';
  let viewingFileContent = '';

  // Mode ('codex', 'story', 'library', 'chat', 'settings', or null for choice screen)
  let mode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | null = null; 

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
  let processedEntries: database.CodexEntry[] = []; // Use database.CodexEntry

  // Helper: Refresh Library Files 
  async function refreshLibraryFiles() {
    console.log('refreshLibraryFiles called, vault status:', { vaultIsReady, currentVaultPath });
    if (!vaultIsReady) {
      console.log('No vault ready, returning early');
      return;
    }
    isLibraryLoading = true;
    errorMsg = ''; 
    try {
      console.log('Calling ListLibraryFiles...');
      libraryFiles = (await ListLibraryFiles()) || []; 
      console.log('Library files loaded:', libraryFiles);
    } catch (err) {
      console.error("Error loading library files:", err);
      errorMsg = `Error loading library: ${err}`;
      libraryFiles = []; 
    } finally {
      isLibraryLoading = false;
    }
  }

  async function loadSettings() {
    console.log("Attempting to load settings...");
    settingsErrorMsg = '';
    try {
      const settings = await GetSettings();
      console.log("Raw settings from backend:", settings);
      
      openrouterApiKey = settings.openrouter_api_key || ''; 
      chatModelId = settings.chat_model_id || ''; 
      storyProcessingModelId = settings.story_processing_model_id || '';
      
      // Debug logging
      console.log("Settings after processing:", {
        apiKeySet: !!openrouterApiKey,
        chatModelId,
        storyProcessingModelId,
        modelListLength: modelList.length
      });

      // If we have an API key, always try to load models
      if (openrouterApiKey) {
        console.log("API key present, loading model list...");
        await loadModelList();
      }
    } catch (err) {
      settingsErrorMsg = `Error loading settings: ${err}`;
      console.error("Settings load error:", err);
    }
  }

  async function saveSettings() {
    console.log("Attempting to save settings...");
    settingsErrorMsg = '';
    settingsSaveMsg = '';
    isLoading = true;
    try {
      console.log("Current values before save:", {
        apiKeyLength: openrouterApiKey?.length || 0,
        chatModelId,
        storyProcessingModelId
      });

      const settingsToSave: OpenRouterConfig = {
        openrouter_api_key: openrouterApiKey,
        chat_model_id: chatModelId,
        story_processing_model_id: storyProcessingModelId
      };
      console.log("Saving settings:", settingsToSave);

      await SaveSettings(settingsToSave);
      settingsSaveMsg = 'Settings saved successfully!';
      console.log("Settings saved successfully");

      // Verify the save by reloading settings
      console.log("Reloading settings to verify...");
      await loadSettings();
    } catch (err) {
      settingsErrorMsg = `Error saving settings: ${err}`;
      console.error("Settings save error:", err);
    } finally {
      isLoading = false;
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
        let codexEntries: database.CodexEntry[] = await GetAllEntries(); // Use database.CodexEntry
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
      // Determine the model to use: Chat View Selector > Settings > Fallback
      const modelToUse = selectedModel || chatModelId || 'anthropic/claude-3-haiku-20240307'; 
      console.log(`Using chat model: ${modelToUse}`);

      const aiReply = await GenerateOpenRouterContent(currentPrompt, modelToUse);
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
      const potentialEntries: database.CodexEntry[] = await ProcessStory(text); // Use database.CodexEntry
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
    // Reset to default values matching the type
    currentEntry = { id: 0, name: '', type: '', content: '', createdAt: '', updatedAt: '' };
    await loadModelList();
    await loadSettings();
  });

  async function loadEntries() {
    if (!vaultIsReady) return; 
    isLoading = true;
    errorMsg = '';
    try {
      entries = (await GetAllEntries()) || []; 
    } catch (err) {
      console.error("Error loading entries:", err);
      errorMsg = `Error loading entries: ${err}`;
    } finally {
      isLoading = false;
    }
  }
 
  function handleEntrySelect(entry: database.CodexEntry) { // Use database.CodexEntry
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
        // Ensure all required fields are present for the Go struct
        const updatePayload: database.CodexEntry = { ...currentEntry } as database.CodexEntry; // Use database.CodexEntry
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
      // Reset to default values matching the type
      currentEntry = { id: 0, name: '', type: '', content: '', createdAt: '', updatedAt: '' };
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
      const newEntriesResult: database.CodexEntry[] = await ImportStoryTextAndFile(storyText); // Use database.CodexEntry
      
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

  // Function to open the library file viewer
  async function viewLibraryFileContent(filename: string) {
    console.log(`Viewing library file: ${filename}`);
    if (!vaultIsReady) {
      errorMsg = 'No vault is currently loaded';
      return;
    }

    isLoading = true;
    errorMsg = '';
    try {
      console.log("Calling ReadLibraryFile...");
      const content = await ReadLibraryFile(filename);
      console.log("ReadLibraryFile successful, content length:", content?.length);
      viewingFilename = filename;
      viewingFileContent = content;
      showLibraryViewer = true;
      console.log("Set showLibraryViewer to true");
    } catch (err) {
      console.error(`Error reading library file ${filename}:`, err);
      errorMsg = `Error reading file: ${err}`;
      alert(errorMsg);
    } finally {
      isLoading = false;
      console.log(`viewLibraryFileContent finished. showLibraryViewer: ${showLibraryViewer}`);
    }
  }

  // Function to save library file content
  async function handleSaveLibraryFile(filename: string, content: string) {
    console.log(`Saving library file: ${filename}`);
    if (!vaultIsReady) {
      errorMsg = 'No vault is currently loaded';
      return;
    }
    isLoading = true;
    errorMsg = '';
    try {
      await SaveLibraryFile(filename, content);
      console.log(`Successfully saved ${filename}`);
      showLibraryViewer = false; // Close viewer on success
    } catch (err) {
      console.error('Error saving library file:', err);
      errorMsg = `Failed to save file ${filename}: ${err}`;
      alert(errorMsg);
    } finally {
      isLoading = false;
    }
  }

  // Renamed from fetchCurrentDBPath
  async function fetchCurrentVaultPath() {
    console.log('fetchCurrentVaultPath called');
    try {
      currentVaultPath = await GetCurrentVaultPath();
      console.log('Current vault path:', currentVaultPath);
      vaultIsReady = !!currentVaultPath;
      console.log('Vault ready status:', vaultIsReady);
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
        await fetchCurrentVaultPath();
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
        await fetchCurrentVaultPath();
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


  // Global error handler
  function handleError(message: string | Event, source?: string, lineno?: number, colno?: number, error?: Error) {
    console.error('Global error caught:', message, source, lineno, colno, error);
    vaultErrorMsg = `An application error occurred: ${message}${error ? ' (' + error.message + ')' : ''}. Please check console for details.`;
    return true; 
  }
  window.onerror = handleError;

  // Handler for keyboard navigation in entry list
  function createKeyDownHandler(entry: database.CodexEntry) { // Use database.CodexEntry
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
      availableChatLogs = (await ListChatLogs()) || [];
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
      const loadedMessages = (await LoadChatLog(filename)) || [];
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

  async function setMode(newMode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | null) {
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
        console.log('Vault status:', { vaultIsReady, currentVaultPath });
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
      } else if (newMode === 'settings') {
        console.log('Handling mode: settings');
        await loadSettings();
      } else if (newMode === null) {
        console.log('Handling mode: null (Vault selection)');
      }
    } else {
      console.log(`Mode ${newMode} is already active.`); // Log no change
    }
  }

  // Helper function to format timestamps
  function formatDate(timestamp: string): string {
    if (!timestamp) return 'N/A';
    try {
      // Assuming timestamp is in RFC3339 or ISO 8601 format
      return new Date(timestamp).toLocaleString();
    } catch (e) {
      console.error("Error formatting date:", e);
      return timestamp; // Return original if formatting fails
    }
  }
 
  // Simple function for debugging button click
  function logButtonClick(filename: string) {
    console.log('logButtonClick called for:', filename);
  }
 
</script>
 
{#if !vaultIsReady} <!-- Vault is NOT ready, show initial screen FIRST -->
  <div class="initial-prompt">
    <img src={logo} alt="Llore Logo" class="logo logo-large" />
    <h2>Select or Create a Vault</h2>
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
  <div class="mode-select">
    <div class="scroll-stave-top"></div> <!-- Top Stave -->
    <div class="scroll-container">
      <img src={logo} alt="Llore Logo" class="logo" />
      <h1>Choose a mode</h1>
      <div class="mode-buttons">
        <button 
          on:click={() => setMode('codex')}
          class="mode-button"
        >
          <span class="title">Codex</span>
          <span class="description">Manage your world's knowledge</span>
        </button>
        <button 
          on:click={() => setMode('story')}
          class="mode-button"
        >
          <span class="title">Story Import</span>
          <span class="description">Analyze and extract lore</span>
        </button>
        <button 
          on:click={() => setMode('library')}
          class="mode-button"
        >
          <span class="title">Library</span>
          <span class="description">Organize your story files</span>
        </button>
        <button 
          on:click={() => setMode('chat')}
          class="mode-button"
        >
          <span class="title">Lore Chat</span>
          <span class="description">Explore your world with AI</span>
        </button>
        <button 
          on:click={() => setMode('settings')}
          class="mode-button"
        >
          <span class="title">Settings</span>
          <span class="description">Configure your experience</span>
        </button>
      </div>
    </div>
    <div class="scroll-stave-bottom"></div> <!-- Bottom Stave -->
  </div>
{:else if mode === 'codex'}
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>

  <div class="codex-view">
    <div class="entries-list">
      <button class="new-entry-btn" on:click={prepareNewEntry}>
        + New Entry
      </button>
      {#if entries.length === 0}
        <p class="empty-state">No entries yet. Create your first one!</p>
      {:else}
        {#each entries as entry (entry.id)}
          <button
            class="entry-item"
            class:active={currentEntry && currentEntry.id === entry.id}
            on:click={() => handleEntrySelect(entry)}
          >
            {entry.name} ({entry.type})
          </button>
        {/each}
      {/if}
    </div>

    <div class="codex-entry">
      {#if currentEntry}
        <form on:submit|preventDefault={handleSaveEntry}>
          <h2>Edit Entry: {currentEntry.name}</h2>
          <div class="codex-entry-content">
            <div class="codex-entry-field">
              <label for="name">Name:</label>
              <input
                type="text"
                id="name"
                bind:value={currentEntry.name}
                placeholder="Entry name"
              />
            </div>

            <div class="codex-entry-field">
              <label for="type">Type:</label>
              <input
                type="text"
                id="type"
                bind:value={currentEntry.type}
                placeholder="Entry type"
              />
            </div>

            <div class="codex-entry-field">
              <label for="content">Content:</label>
              <textarea
                id="content"
                bind:value={currentEntry.content}
                placeholder="Entry content"
              />
            </div>
          </div>

          <div class="button-group">
            <button type="submit" class="save-btn">Save Changes</button>
            <button type="button" class="delete-btn" on:click={handleDeleteEntry}>Delete Entry</button>
            <button type="button" class="generate-btn" on:click={handleGenerateContent} disabled={isGenerating}>
              {#if isGenerating}Generating...{:else}Generate Content (AI){/if}
            </button>
          </div>

          {#if errorMsg}
            <p class="error-message">{errorMsg}</p>
          {/if}
        </form>
      {:else}
        <div class="empty-state">
          <p>Select an entry to edit or create a new one</p>
        </div>
      {/if}
    </div>
  </div>
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
    <button on:click={() => refreshLibraryFiles()} disabled={isLibraryLoading}>
      Refresh Library
    </button>
    {#if isLibraryLoading}
      <p>Loading library files...</p>
    {:else if errorMsg}
      <p class="error-message">{errorMsg}</p>
    {:else}
      {#if libraryFiles.length === 0}
        <p>No files in library.</p>
      {:else}
        <ul>
          {#each libraryFiles as filename}
            <li>
              {filename}
              <button on:click={() => viewLibraryFileContent(filename)}>View/Edit</button>
            </li>
          {/each}
        </ul>
      {/if} <!-- End check file list -->
    {/if} <!-- End check error -->
  </section>
{:else if mode === 'chat'}
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>
  <div class="chat-container">
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
            {#each modelList as model (model.id)}
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
  </div>
{:else if mode === 'settings'}
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>
  <section class="settings">
    <h2>Settings</h2>
    <div class="settings-container">
      <div class="form-group">
        <label for="apiKey">OpenRouter API Key:</label>
        <div class="api-key-input">
          {#if showApiKey}
            <input
              type="text"
              id="apiKey"
              bind:value={openrouterApiKey}
              placeholder="Enter your OpenRouter API key"
            />
          {:else}
            <input
              type="password"
              id="apiKey"
              bind:value={openrouterApiKey}
              placeholder="Enter your OpenRouter API key"
            />
          {/if}
          <button 
            class="toggle-visibility" 
            on:click={() => showApiKey = !showApiKey}
            title={showApiKey ? "Hide API Key" : "Show API Key"}
          >
            {#if showApiKey}
              👁️
            {:else}
              👁️‍🗨️
            {/if}
          </button>
        </div>
      </div>
      <div class="form-group">
        <label for="chat-model-select">Chat Model:</label>
        {#if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span style="color:red">{modelListError}</span>
        {:else}
          <select id="chat-model-select" bind:value={chatModelId}>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
      </div>
      <div class="form-group">
        <label for="story-processing-model-select">Story Processing Model:</label>
        {#if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span style="color:red">{modelListError}</span>
        {:else}
          <select id="story-processing-model-select" bind:value={storyProcessingModelId}>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
      </div>
      <button type="submit" disabled={isLoading}>Save Settings</button>
      {#if settingsSaveMsg}
        <p style="color: green;">{settingsSaveMsg}</p>
      {/if}
      {#if settingsErrorMsg}
        <p class="error-message">{settingsErrorMsg}</p>
      {/if}
    </div>
  </section>
{/if} <!-- This NOW closes the entire chain starting with #if !vaultIsReady -->

{#if showLibraryViewer}
  <LibraryFileViewer 
    filename={viewingFilename} 
    initialContent={viewingFileContent} 
    on:close={() => showLibraryViewer = false} 
    on:save={(event) => handleSaveLibraryFile(event.detail.filename, event.detail.content)}
  />
{/if}

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
  /* Reset and Base Styles */
  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
    background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
    color: #e0e0e0;
    font-size: 16px;
    line-height: 1.6;
    height: 100vh;
    overflow: hidden;
  }

  :global(#app) {
    height: 100vh;
    display: flex;
    flex-direction: column;
    overflow: hidden;
  }

  :global(*) {
    box-sizing: border-box;
  }

  /* Variables */
  :root {
    --accent-silver: #c0c0c0;
    --accent-gradient: linear-gradient(135deg, #6d5ed9, #8a7ef9);
    --accent-primary: #6d5ed9;
    --accent-secondary: #8a7ef9;
    --bg-primary: rgba(26, 26, 46, 0.95);
    --bg-secondary: rgba(22, 33, 62, 0.95);
    --text-primary: #e0e0e0;
    --text-secondary: #a0a0a0;
    --error-color: #ff4757;
    --success-color: #2ed573;
  }

  /* Layout Container */
  .layout-container {
    display: flex;
    gap: 1.5rem;
    padding: 1.5rem;
    max-width: 1400px;
    margin: 0 auto;
    height: calc(100vh - 4rem);
  }

  /* Sidebar */
  .entries-list {
    width: 300px;
    background: var(--bg-secondary);
    border-radius: 12px;
    padding: 1rem;
    display: flex;
    flex-direction: column;
    gap: 1rem;
    overflow-y: auto;
    max-height: 100%;
  }

  .entry-item-button {
    width: 100%;
    padding: 0.75rem 1rem;
    background: rgba(255, 255, 255, 0.05);
    border: none;
    border-radius: 8px;
    color: var(--text-primary);
    cursor: pointer;
    transition: all 0.3s ease;
    text-align: left;
  }

  .entry-item-button:hover {
    background: rgba(255, 255, 255, 0.1);
    transform: translateY(-1px);
  }

  /* Main Content */
  .main-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    height: calc(100vh - 4rem);
    overflow: hidden;
    padding: 1rem;
  }

  .codex-view {
    display: flex;
    gap: 2rem;
    flex: 1;
    overflow: hidden;
    background: var(--bg-primary);
    border-radius: 12px;
    padding: 1rem;
    height: calc(100vh - 6rem);
  }

  .entries-list {
    width: 300px;
    padding: 1rem;
    background: rgba(255, 255, 255, 0.03);
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    overflow-y: auto;
  }

  .codex-entry {
    flex: 1;
    padding: 1.5rem;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    overflow-y: auto;
  }

  .codex-entry form {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .codex-entry-content {
    display: flex;
    flex-direction: column;
    gap: 1.25rem;
  }

  .codex-entry-field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .codex-entry-field label {
    color: var(--text-secondary);
    font-size: 0.9rem;
  }

  .codex-entry-field input,
  .codex-entry-field textarea {
    background: rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    padding: 0.75rem;
    color: var(--text-primary);
    font-size: 1rem;
  }

  .codex-entry-field textarea {
    min-height: 200px;
    resize: vertical;
  }

  .button-group {
    display: flex;
    gap: 1rem;
    margin-top: 1rem;
  }

  .save-btn,
  .delete-btn,
  .generate-btn {
    padding: 0.75rem 1.5rem;
    border-radius: 6px;
    font-weight: 500;
    transition: all 0.3s ease;
  }

  .save-btn {
    background: var(--accent-primary);
    color: white;
  }

  .delete-btn {
    background: var(--error-color);
    color: white;
  }

  .generate-btn {
    background: var(--accent-secondary);
    color: white;
  }

  .generate-btn:hover {
    background: var(--accent-secondary-hover);
  }

  .empty-state {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: var(--text-secondary);
    font-size: 1.1rem;
    text-align: center;
    padding: 2rem;
  }

  .new-entry-btn {
    background: var(--accent-primary);
    color: white;
    border: none;
    padding: 0.75rem;
    border-radius: 6px;
    font-weight: 500;
    transition: all 0.3s ease;
    margin-bottom: 1rem;
    width: 100%;
  }

  .new-entry-btn:hover {
    background: var(--accent-primary-hover);
    transform: translateY(-1px);
  }

  .entry-item {
    width: 100%;
    text-align: left;
    padding: 0.75rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text-primary);
    transition: all 0.3s ease;
  }

  .entry-item:hover {
    background: rgba(255, 255, 255, 0.1);
  }

  .entry-item.active {
    background: var(--accent-primary);
    color: white;
  }

  .codex-entry {
    flex: 1;
    padding: 1.5rem;
    background: var(--bg-secondary);
    border-radius: 12px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
    border: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
    gap: 1rem;
    overflow: hidden;
  }

  .codex-entry-content {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    overflow-y: auto;
    padding-right: 0.5rem;
  }

  .codex-entry-field {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .codex-entry-field label {
    font-weight: 500;
    color: var(--text-secondary);
  }

  .codex-entry-field input,
  .codex-entry-field textarea {
    width: 100%;
    padding: 0.75rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text-primary);
  }

  .codex-entry-field textarea {
    flex: 1;
    min-height: 200px;
    resize: none;
  }

  /* Forms */
  .form-group {
    margin-bottom: 1.5rem;
  }

  label {
    display: block;
    color: var(--accent-silver);
    margin-bottom: 0.5rem;
    font-weight: 500;
  }

  input[type="text"],
  .form-group input {
    width: 100%;
    padding: 0.75rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 1rem;
    transition: border-color 0.3s ease;
  }

  .api-key-input {
    position: relative;
    display: flex;
    align-items: center;
  }

  .api-key-input input {
    padding-right: 3rem;
  }

  .toggle-visibility {
    position: absolute;
    right: 0.75rem;
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.25rem;
    font-size: 1.25rem;
    transition: color 0.3s ease;
  }

  .toggle-visibility:hover {
    color: var(--text-primary);
  }

  input[type="text"]:focus,
  textarea:focus,
  select:focus {
    outline: none;
    border-color: var(--accent-primary);
    box-shadow: 0 0 0 3px rgba(109, 94, 217, 0.3);
  }

  textarea {
    resize: vertical;
    min-height: 150px;
  }

  /* Buttons */
  button {
    padding: 0.75rem 1.5rem;
    background: var(--accent-gradient);
    border: none;
    border-radius: 8px;
    color: white;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
    box-shadow: 0 4px 12px rgba(109, 94, 217, 0.2);
  }

  button:hover {
    transform: translateY(-2px);
    box-shadow: 0 6px 16px rgba(109, 94, 217, 0.3);
  }

  button:active {
    transform: translateY(0);
  }

  button:disabled {
    background: rgba(255, 255, 255, 0.1);
    color: var(--text-secondary);
    cursor: not-allowed;
    transform: none;
    box-shadow: none;
  }

  .button-group {
    display: flex;
    gap: 1rem;
    margin-top: 1.5rem;
  }

  .mode-buttons {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    margin-top: 1.5rem;
    width: 100%;
    max-width: 300px;
    margin-left: auto;
    margin-right: auto;
  }

  .mode-buttons button {
    width: 100%;
    font-size: 1.1rem;
    padding: 1rem;
  }

  .logo {
    width: 200px;
    height: auto;
    margin-bottom: 2rem;
  }

  .logo-large {
    width: 300px;
    margin-bottom: 3rem;
  }

  /* Chat Interface */
  .chat-window {
    background: var(--bg-secondary);
    border-radius: 12px;
    padding: 1.5rem;
    height: 60vh;
    overflow-y: auto;
    margin-bottom: 1.5rem;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .message {
    margin-bottom: 1rem;
    padding: 1rem;
    border-radius: 12px;
    max-width: 80%;
  }

  .chat-user {
    background: var(--accent-gradient);
    color: white;
    margin-left: auto;
    border-bottom-right-radius: 4px;
  }

  .chat-ai {
    background: var(--bg-primary);
    margin-right: auto;
    border-bottom-left-radius: 4px;
  }

  /* Modal */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.8);
    backdrop-filter: blur(4px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    overflow-y: auto;
    padding: 2rem;
  }

  .modal-content {
    background: var(--bg-primary);
    color: var(--text-primary);
    border-radius: 12px;
    padding: 2rem;
    width: 100%;
    max-width: 600px;
    margin: auto;
    box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4), 0 0 15px rgba(253, 246, 227, 0.1) inset;
    border: 1px solid rgba(255, 255, 255, 0.1);
    position: relative;
  }

  .modal-close {
    position: absolute;
    top: 1rem;
    right: 1rem;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    padding: 0.5rem;
    cursor: pointer;
    transition: color 0.3s ease;
  }

  .modal-close:hover {
    color: var(--text-primary);
  }

  /* Initial Prompt */
  .settings-container {
    max-width: 800px;
    margin: 0 auto;
    padding: 2rem;
    display: flex;
    flex-direction: column;
    gap: 2rem;
    height: 100%;
    overflow-y: auto;
  }

  .settings-section {
    background: rgba(255, 255, 255, 0.05);
    padding: 2rem;
    border-radius: 12px;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  /* Mode Selection Screen */
  .mode-select {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    min-height: 100vh;
    padding: 3rem 2rem;
    background: var(--bg-primary); /* Keep overall background dark */
    position: relative; /* Needed for absolute positioning of staves */
  }

  .scroll-container {
    background: #fdf6e3; /* Parchment-like color */
    padding: 3rem 2.5rem;
    border-radius: 15px;
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3), 0 0 15px rgba(253, 246, 227, 0.1) inset;
    max-width: 500px;
    width: 90%;
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2rem;
    border: 1px solid rgba(0,0,0,0.1);
    position: relative; /* Keep relative for content */
    z-index: 5; /* Ensure parchment is behind staves */
  }

  .scroll-container .logo {
    width: 200px; /* Doubled from 100px */
    height: auto;
    margin-bottom: 0.5rem;
    animation: float 6s ease-in-out infinite;
    filter: drop-shadow(0 2px 3px rgba(0,0,0,0.2));
  }

  @keyframes float {
    0% {
      transform: translateY(0);
    }
    50% {
      transform: translateY(-10px);
    }
    100% {
      transform: translateY(0);
    }
  }

  .scroll-container h1 {
    color: #584c3a; /* Darker brown text */
    font-family: 'Georgia', serif; /* More classic font */
    font-size: 2.2rem;
    margin: 0;
    text-align: center;
    opacity: 0.9;
  }

  .mode-buttons {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    margin-top: 1.5rem;
    width: 100%;
    max-width: 300px;
    margin-left: auto;
    margin-right: auto;
  }

  .mode-button {
    width: 100%;
    padding: 1rem 1.5rem;
    font-size: 1.1rem;
    text-align: left;
    background: transparent; /* Make background transparent */
    border: 1px solid #a0937d; /* Defined parchment border */
    border-radius: 8px;
    color: #65594a; /* Brownish text */
    cursor: pointer;
    transition: all 0.3s ease;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    position: relative;
    overflow: hidden;
    font-family: 'Georgia', serif; /* Match title font */
  }

  .mode-button .title {
    font-weight: 600; /* Slightly bolder */
    color: #584c3a; /* Darker brown */
    z-index: 1;
  }

  .mode-button .description {
    font-size: 0.85rem;
    color: #8a7a66; /* Lighter brown */
    opacity: 0;
    transform: translateY(10px);
    transition: all 0.3s ease;
    z-index: 1;
  }

  .mode-button:hover {
    background: rgba(88, 76, 58, 0.05); /* Subtle hover background */
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(88, 76, 58, 0.15); /* Brownish shadow */
    border-color: #584c3a; /* Darker border on hover */
  }

  .mode-button:hover .description {
    opacity: 1;
    transform: translateY(0);
  }

  /* Back Button */
  .back-btn {
    position: absolute;
    top: 1rem;
    left: 1rem;
    background: transparent;
    border: none;
    color: var(--text-secondary);
    padding: 0.5rem;
    cursor: pointer;
    transition: color 0.3s ease;
  }

  .back-btn:hover {
    color: var(--text-primary);
  }

  /* Utilities */
  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    border: 1px solid rgba(255, 71, 87, 0.2);
  }

  .success-message {
    color: var(--success-color);
    background: rgba(46, 213, 115, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    border: 1px solid rgba(46, 213, 115, 0.2);
  }

  /* Scrollbar */
  ::-webkit-scrollbar {
    width: 8px;
    height: 8px;
  }

  ::-webkit-scrollbar-track {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 4px;
  }

  ::-webkit-scrollbar-thumb {
    background: var(--accent-primary);
    border-radius: 4px;
  }

  ::-webkit-scrollbar-thumb:hover {
    background: var(--accent-secondary);
  }

  /* Staves */
  .scroll-stave-top,
  .scroll-stave-bottom {
    position: relative; /* Position relative to mode-select center */
    width: 90%; /* Match scroll-container width */
    max-width: 500px; /* Match scroll-container max-width */
    height: 30px; /* Height of the stave */
    background: linear-gradient(to right, #8B4513, #A0522D, #8B4513); /* Wood-like gradient */
    border-radius: 15px; /* Rounded ends */
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
    z-index: 10; /* Ensure staves are above the parchment visually */
    border: 1px solid #5c2e11;
  }

  .scroll-stave-top {
    margin-bottom: -15px; /* Overlap slightly with parchment top */
  }

  .scroll-stave-bottom {
    margin-top: -15px; /* Overlap slightly with parchment bottom */
  }
</style>
