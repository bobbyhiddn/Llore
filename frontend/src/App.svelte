<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { database, llm } from '@wailsjs/go/models'; // Import namespaces
  import LibraryFileViewer from './LibraryFileViewer.svelte';
  import logo from './assets/images/logo.png';
  import { Marked } from 'marked'; // Import Marked class
  const marked = new Marked(); // Create synchronous instance
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

  // Mode ('codex', 'story', 'library', 'chat', 'settings', 'write', or null for choice screen)
  let mode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write' | null = null; 

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

  // Story import state
  let showImportModal = false;
  let createdEntriesCount = 0;
  let processedEntries: database.CodexEntry[] = []; // Use database.CodexEntry
  let isDraggingFile = false;
  let importError = '';
  let importedFileName = '';
  let importedContent = '';
  let isProcessingImport = false;
  let existingEntries: database.CodexEntry[] = [];
  let showExistingEntriesModal = false;
  let processStorySuccessMsg = '';


  // --- Write Mode State ---
  let writeContent: string = ''; // Content of the writing area
  let renderedWriteHtml = ''; // Initialize as empty string
  let renderedWriteHtmlPromise: Promise<string> | null = null; // For handling async markdown // Rendered HTML from writeContent
  // Placeholder states for chat and tools within write mode
  let writeChatMessages: { sender: 'user' | 'ai', text: string }[] = [];
  let writeChatInput: string = '';
  let isWriteChatLoading: boolean = false;
  let writeChatError: string = ''; // Add error state for write chat
  let writeChatDisplayElement: HTMLDivElement; // For auto-scrolling
  let markdownTextareaElement: HTMLTextAreaElement; // Reference to the editor textarea

  // Save state for Write mode
  let showWriteSaveModal: boolean = false;
  let writeFilename: string = '';
  let writeSaveError: string = '';
  let writeSaveSuccess: string = '';

  $: if (mode === 'write' && writeContent !== undefined) {
    try {
      // Handle markdown parsing asynchronously
      renderedWriteHtmlPromise = Promise.resolve(marked.parse(writeContent))
        .then(html => {
          renderedWriteHtml = html;
          return html;
        })
        .catch(err => {
          console.error("Markdown rendering error:", err);
          renderedWriteHtml = writeContent;
          return writeContent;
        });
    } catch (err) {
      console.error("Immediate markdown error:", err);
      renderedWriteHtml = writeContent;
    }
  }

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
  // Handle file drop for story import
  function handleFileDrop(event: DragEvent) {
    event.preventDefault();
    isDraggingFile = false;
    importError = '';
    
    const files = event.dataTransfer?.files;
    if (!files || files.length === 0) return;
    
    const file = files[0];
    if (!file.name.match(/\.(txt|md)$/i)) {
      importError = 'Please drop a .txt or .md file';
      return;
    }
    
    importedFileName = file.name;
    const reader = new FileReader();
    reader.onload = async (e) => {
      const content = e.target?.result as string;
      if (content) {
        importedContent = content;
        showImportModal = true;
      }
    };
    reader.readAsText(file);
  }

  // Handle manual file selection
  async function handleFileSelect(event: Event) {
    const input = event.target as HTMLInputElement;
    if (!input.files || input.files.length === 0) return;
    
    const file = input.files[0];
    if (!file.name.match(/\.(txt|md)$/i)) {
      importError = 'Please select a .txt or .md file';
      return;
    }
    
    importedFileName = file.name;
    const reader = new FileReader();
    reader.onload = async (e) => {
      const content = e.target?.result as string;
      if (content) {
        importedContent = content;
        showImportModal = true;
      }
    };
    reader.readAsText(file);
  }

  // Process the imported story
  async function processImportedStory(forceReimport = false) {
    if (!importedContent) return;
    
    isProcessingImport = true;
    importError = '';
    processStorySuccessMsg = '';
    
    try {
      // Process the story for codex entries and save to library
      const entries = await ImportStoryTextAndFile(importedContent);
      
      // Check if any entries already exist
      const existing = entries.filter(e => e.id !== 0);
      const newEntries = entries.filter(e => e.id === 0);
      
      if (existing.length > 0 && !forceReimport) {
        // Show confirmation dialog
        existingEntries = existing;
        showExistingEntriesModal = true;
        isProcessingImport = false;
        return;
      }
      
      processedEntries = entries;
      createdEntriesCount = newEntries.length;
      showImportModal = false;
      importedContent = '';
      importedFileName = '';
      showExistingEntriesModal = false;
      
      // Show feedback
      if (entries.length > 0) {
        if (existing.length > 0) {
          processStorySuccessMsg = `Story Processed\n${existing.length} entries were updated and ${newEntries.length} new entries were created:`;
        } else {
          processStorySuccessMsg = `Story Processed\n${entries.length} new codex entries were created:`;
        }
        processedEntries = entries;
      } else {
        processStorySuccessMsg = 'No codex entries could be extracted from the story.';
      }
    } catch (err) {
      importError = `Failed to process story: ${err}`;
      console.error('Story import error:', err);
    } finally {
      isProcessingImport = false;
    }
  }

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

  async function setMode(newMode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write' | null) {
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
      } else if (newMode === 'write') { // Add handler for write mode
        console.log('Handling mode: write');
        // Add any specific setup for write mode here later
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
 
  // Auto-scroll chat display
  afterUpdate(() => {
    if (chatDisplayElement) {
      chatDisplayElement.scrollTop = chatDisplayElement.scrollHeight;
    }
    // Add similar logic for writeChatDisplayElement if needed
    if (writeChatDisplayElement) {
       writeChatDisplayElement.scrollTop = writeChatDisplayElement.scrollHeight; 
    }
  });

  // --- Write Mode Chat Function ---
  async function handleSendWriteChat() {
    if (!writeChatInput.trim() || isWriteChatLoading) return;

    const userMessage = writeChatInput.trim();
    writeChatMessages = [...writeChatMessages, { sender: 'user', text: userMessage }];
    writeChatInput = '';
    isWriteChatLoading = true;
    writeChatError = '';

    // Construct the prompt with context
    let prompt = `System: You are an AI assistant helping a user write. Here is their current draft:\n\n---\n${writeContent}\n---\n\nChat History:\n`;
    writeChatMessages.slice(-5).forEach(msg => { // Limit history slightly
      prompt += `${msg.sender === 'user' ? 'User' : 'AI'}: ${msg.text}\n`;
    });
    // Ensure the latest user message (which triggered this) is included if not already in slice
    if (!writeChatMessages.slice(-5).some(m => m.sender === 'user' && m.text === userMessage)){
         prompt += `User: ${userMessage}\n`;
    }
    prompt += "AI:"; // Prompt the AI for its turn

    try {
      // Use the globally selected chat model from settings
      const modelToUse = chatModelId || 'anthropic/claude-3-haiku-20240307'; // Use state variable or fallback
      console.log(`Write Chat using model: ${modelToUse}`);

      const aiReply = await GenerateOpenRouterContent(prompt, modelToUse);
      writeChatMessages = [...writeChatMessages, { sender: 'ai', text: aiReply }];

    } catch (err) {
      writeChatError = `AI error: ${err}`;
      // Optionally add the error as a system message to the chat?
      // writeChatMessages = [...writeChatMessages, { sender: 'ai', text: `Sorry, I encountered an error: ${err}` }];
      console.error("Write Chat Error:", err);
    } finally {
      isWriteChatLoading = false;
    }
  }

  // --- Write Mode Save Function ---
  async function handleSaveWriteContent() {
    if (!writeFilename.trim()) {
      writeSaveError = 'Filename cannot be empty.';
      return;
    }
    let filenameToSave = writeFilename.trim();
    if (!filenameToSave.toLowerCase().endsWith('.md')) {
      filenameToSave += '.md';
    }

    // Reset messages
    writeSaveError = '';
    writeSaveSuccess = '';
    isLoading = true; // Use existing loading state

    try {
      await SaveLibraryFile(filenameToSave, writeContent);
      writeSaveSuccess = `File '${filenameToSave}' saved successfully!`;
      showWriteSaveModal = false; // Close modal on success
      writeFilename = ''; // Clear filename input for next save
    } catch (err) {
      writeSaveError = `Failed to save file: ${err}`;
      console.error("Save Write Content Error:", err);
    } finally {
      isLoading = false;
    }
  }

  // --- Write Mode Formatting Tools Function ---
  function applyMarkdownFormat(formatType: 'bold' | 'italic' | 'h1' | 'h2') {
    if (!markdownTextareaElement) return;

    const start = markdownTextareaElement.selectionStart;
    const end = markdownTextareaElement.selectionEnd;
    const selectedText = writeContent.substring(start, end);
    let prefix = '';
    let suffix = '';

    switch (formatType) {
      case 'bold':
        prefix = '**';
        suffix = '**';
        break;
      case 'italic':
        prefix = '*';
        suffix = '*';
        break;
      case 'h1':
      case 'h2':
        // Apply headings to the start of the line
        const lineStart = writeContent.lastIndexOf('\n', start - 1) + 1;
        prefix = (formatType === 'h1' ? '# ' : '## ');
        // Insert prefix at the beginning of the line
        writeContent = writeContent.substring(0, lineStart) + prefix + writeContent.substring(lineStart);
        // Adjust selection points after adding prefix
        markdownTextareaElement.selectionStart = start + prefix.length;
        markdownTextareaElement.selectionEnd = end + prefix.length;
        // Focus after update
        markdownTextareaElement.focus();
        return; // Early return for headings as they modify outside selection
    }

    // Apply prefix/suffix for bold/italic
    const newText = writeContent.substring(0, start) + prefix + selectedText + suffix + writeContent.substring(end);
    writeContent = newText;

    // Restore selection/cursor position after update (might need adjustment)
    // Use timeout to ensure DOM updates before setting selection
    setTimeout(() => {
        if (!markdownTextareaElement) return;
        if (selectedText) {
          // If text was selected, select the newly formatted text
          markdownTextareaElement.selectionStart = start + prefix.length;
          markdownTextareaElement.selectionEnd = end + prefix.length;
        } else {
          // If no text was selected, place cursor between prefix and suffix
          markdownTextareaElement.selectionStart = start + prefix.length;
          markdownTextareaElement.selectionEnd = start + prefix.length;
        }
         markdownTextareaElement.focus();
    }, 0);

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
      <img src={logo} alt="Llore Logo" class="logo" style="margin-bottom: 1.5rem;" />
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
        <button 
          on:click={() => setMode('write')}
          class="mode-button"
        >
          <span class="title">Write</span>
          <span class="description">Compose stories & articles</span>
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
    <p>Paste story text or drag & drop a file below. Files will be saved to the Library and processed for codex entries.</p>
    
    <!-- Text Input -->
    <div class="text-input-section">
      <div class="text-input-container">
        <textarea 
          bind:value={storyText} 
          class="story-input"
          placeholder="Paste your story text here..."
          disabled={isProcessingStory || isProcessingImport}
        ></textarea>
      </div>
    </div>

    <!-- Drop Zone and Import Button -->
    <div class="drop-zone-section">
      <div class="drop-zone-container">
        <div class="drop-zone {isDraggingFile ? 'dragging' : ''}"
          role="button"
          tabindex="0"
          aria-label="Drop zone for importing story files"
          on:dragenter={(e) => { e.preventDefault(); isDraggingFile = true; }}
          on:dragover={(e) => { e.preventDefault(); isDraggingFile = true; }}
          on:dragleave={() => isDraggingFile = false}
          on:drop={handleFileDrop}
          on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') document.getElementById('file-input')?.click(); }}
        >
          <span class="icon">📚</span>
          <span class="label">Drop Story File Here</span>
          <span class="description">or</span>
          <input
            type="file"
            accept=".txt,.md"
            on:change={handleFileSelect}
            style="display: none"
            id="file-input"
          />
          <button class="browse-btn" on:click={() => document.getElementById('file-input')?.click()}>
            Browse Files
          </button>
          {#if importError}
            <p class="error-message">{importError}</p>
          {/if}
        </div>
        <button 
          class="import-btn"
          on:click={handleImportStory} 
          disabled={isProcessingStory || isProcessingImport || !storyText.trim()}
        >
          {#if isProcessingStory}Processing...{:else}Import Story & Add Entries{/if}
        </button>
        {#if processStoryErrorMsg}
          <p class="error-message">{processStoryErrorMsg}</p>
        {/if}
      </div>
    </div>
  </section>

  <!-- Import Preview Modal -->
  {#if showImportModal}
    <div class="modal-backdrop">
      <div class="modal import-modal">
        <h3>Import Story</h3>
        <div class="import-preview">
          <p class="filename">File: {importedFileName}</p>
          <div class="content-preview">
            {importedContent.slice(0, 500)}{importedContent.length > 500 ? '...' : ''}
          </div>
        </div>
        {#if importError}
          <p class="error-message">{importError}</p>
        {/if}
        <div class="modal-actions">
          <button on:click={processImportedStory} disabled={isProcessingImport}>
            {isProcessingImport ? 'Processing...' : 'Process Story'}
          </button>
          <button on:click={() => { showImportModal = false; importError = ''; }} disabled={isProcessingImport}>
            Cancel
          </button>
        </div>
      </div>
    </div>
  {/if}
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
      <section class="lore-chat chat-view-container">
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
        <div class="chat-display chat-messages-area" bind:this={chatDisplayElement}>
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
{:else if mode === 'write'}
  <button class="back-btn" on:click={() => setMode(null)}>← Back to Mode Choice</button>
  <div class="write-view-container">
    <!-- Left Panel (Top: Chat, Bottom: Tools) -->
    <div class="write-left-panel">
      <div class="write-chat-panel">
        <h3>Contextual Chat</h3>
        <div class="chat-messages-area" bind:this={writeChatDisplayElement} style="height: 200px; overflow-y: auto; border: 1px solid #ccc; margin-bottom: 10px;">
          <!-- Chat messages will go here -->
          {#each writeChatMessages as msg (msg.sender + msg.text + Math.random())} <!-- Basic key for reactivity -->
            <div class="message {msg.sender}">{msg.sender === 'user' ? 'You' : 'AI'}: {msg.text}</div>
          {/each}
          {#if isWriteChatLoading}<div class="message loading">AI Thinking...</div>{/if}
        </div>
        <form on:submit|preventDefault={handleSendWriteChat} class="write-chat-form">
          <input type="text" bind:value={writeChatInput} placeholder="Ask AI..." disabled={isWriteChatLoading} />
          <button type="submit" disabled={isWriteChatLoading || !writeChatInput.trim()}>Send</button>
        </form>
        {#if writeChatError}
          <p class="error-message">{writeChatError}</p>
        {/if}
      </div>
      <div class="write-tools-panel">
        <h3>Formatting Tools</h3>
        <div class="button-group">
           <button on:click={() => applyMarkdownFormat('bold')}>B</button> 
           <button on:click={() => applyMarkdownFormat('italic')}>I</button> 
           <button on:click={() => applyMarkdownFormat('h1')}>H1</button> 
           <button on:click={() => applyMarkdownFormat('h2')}>H2</button> 
           <!-- Add more buttons later -->
        </div>
         <button class="save-btn" style="margin-top: 1rem;" on:click={() => { showWriteSaveModal = true; writeSaveError = ''; writeSaveSuccess = ''; }}>Save to Library</button> 
      </div>
    </div>

    <!-- Right Panel (Editor + Preview) -->
    <div class="write-right-panel">
       <textarea 
         class="markdown-input"
         bind:value={writeContent}
         bind:this={markdownTextareaElement} 
         placeholder="Start writing your masterpiece (Markdown supported)..."
       ></textarea>
       <div class="markdown-preview-container">
          <h3>Preview</h3>
          <div class="markdown-preview">{@html renderedWriteHtml}</div>
       </div>
    </div>
  </div>
{/if} <!-- This NOW closes the entire chain starting with #if !vaultIsReady -->

{#if showWriteSaveModal}
  <div class="modal-backdrop">
    <div class="modal save-write-modal">
      <h3>Save Written Content</h3>
      <label for="write-filename">Filename:</label>
      <input id="write-filename" type="text" bind:value={writeFilename} placeholder="e.g., chapter-one.md">
      {#if writeSaveError}
        <p class="error-message">{writeSaveError}</p>
      {/if}
      {#if writeSaveSuccess}
        <p style="color: green;">{writeSaveSuccess}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={handleSaveWriteContent} disabled={isLoading || !writeFilename.trim()}>Save</button>
        <button on:click={() => { showWriteSaveModal = false; writeSaveSuccess = ''; }} disabled={isLoading}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

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
      {#if existingEntries && existingEntries.length > 0}
        <p class="notice-message">Note: This story has already been processed before.</p>
      {/if}
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
        <button on:click={() => { showImportModal = false; }}>OK</button>
        <button on:click={() => closeImportModal(true)}>Go to Codex</button>
      </div>
    </div>
  </div>
{/if}

{#if showExistingEntriesModal}
  <div class="modal-backdrop">
    <div class="modal-content">
      <h3>Existing Entries Found</h3>
      <p>The following entries already exist from this story:</p>
      <div class="existing-entries-list">
        {#each existingEntries as entry}
          <div class="existing-entry">
            <strong>{entry.name}</strong> ({entry.type})
            <p class="entry-preview">{entry.content.substring(0, 100)}{entry.content.length > 100 ? '...' : ''}</p>
          </div>
        {/each}
      </div>
      <p>Would you like to update these entries?</p>
      <div class="modal-actions">
        <button on:click={() => { showExistingEntriesModal = false; importedContent = ''; importedFileName = ''; }}>Cancel</button>
        <button 
          class="primary"
          on:click={() => processImportedStory(true)}
          disabled={isProcessingImport}
        >
          {#if isProcessingImport}
            Processing...
          {:else}
            Update Entries
          {/if}
        </button>
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
    overflow: hidden; /* Re-add to prevent body scroll */
  }

  :global(#app) {
    height: 100%; /* Use percentage height instead of viewport height */
    display: flex;
    flex-direction: column;
    /* overflow: hidden; */ /* Removed to allow child scrolling */
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
    flex: 1;
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
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
    width: 100%;
    max-width: 500px;
    margin-top: 0;
    margin-left: auto;
    margin-right: auto;
  }

  .mode-buttons button {
    width: 100%;
    font-size: 1.1rem;
    padding: 1rem;
  }

  .logo {
    width: 150px;
    height: auto;
    margin-bottom: 1.5rem;
  }

  .logo-large {
    width: 300px;
    margin-bottom: 3rem;
  }

/* Chat View Layout */
  .chat-view-container {
    display: flex;
    flex-direction: column;
    height: calc(100vh - 4rem); /* Adjust based on header/footer height if any */
    overflow: hidden; /* Prevent whole section from scrolling */
    padding: 1rem; /* Add some padding */
  }

  .chat-settings-row {
    flex-shrink: 0; /* Prevent settings row from shrinking */
    margin-bottom: 1rem;
  }

  .chat-messages-area {
    flex: 1; /* Allow message area to grow and shrink */
    overflow-y: auto; /* Enable vertical scrolling */
    padding: 1rem;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    margin-bottom: 1rem;
    background: rgba(0,0,0,0.1);
  }

  .chat-form {
    flex-shrink: 0; /* Prevent form from shrinking */
    display: flex;
    gap: 0.5rem;
  }

  .chat-form input {
    flex-grow: 1; /* Allow input to take available space */
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
    justify-content: center; /* Center the scroll container vertically */
    height: 100%; /* Fill parent #app */
    width: 100%; /* Ensure it respects parent width */
    padding: 1rem 2rem; /* Add some overall padding */
    background: var(--bg-primary); /* Keep overall background dark */
    position: relative; /* Needed for absolute positioning of staves */
    /* Removed flex: 1, min-height: 0, overflow-y: auto */
  }

  .scroll-container {
    background: #fdf6e3; /* Parchment-like color */
    padding: 2rem 2.5rem;
    border-radius: 15px;
    overflow-y: auto; /* Make the inner container scroll */
    max-height: calc(100% - 80px); /* Limit height (adjust 80px as needed for staves/padding) */
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3), 0 0 15px rgba(253, 246, 227, 0.1) inset;
    /* Use clamp for responsive width: min 90%, preferred 60vw, max 800px */
    width: clamp(90%, 60vw, 800px);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2rem;
    border: 1px solid rgba(0,0,0,0.1);
    position: relative; /* Keep relative for content */
    z-index: 5; /* Ensure parchment is behind staves */
  }

  .scroll-container .logo {
    width: 150px; /* Reduced from 200px */
    height: auto;
    margin-bottom: 1.5rem;
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

  .mode-buttons {
    /* Use Grid for responsive button layout */
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
    width: 100%;
    max-width: 500px;
    margin-top: 0;
    margin-left: auto;
    margin-right: auto;
  }

  .mode-button {
    /* Removed width: 100% - Grid handles sizing */
    padding: 0.4rem 1.2rem;
    font-size: 0.9rem;
    text-align: left;
    background: #f5eeda;
    border: 1px solid #a0937d;
    border-radius: 8px;
    color: #65594a;
    cursor: pointer;
    transition: all 0.3s ease;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    position: relative;
    overflow: hidden;
    font-family: 'Georgia', serif;
  }

  .mode-button .title {
    font-weight: 600;
    color: #584c3a;
    z-index: 1;
    font-size: 1rem;
  }

  .mode-button .description {
    font-size: 0.85rem;
    color: #8a7a66;
    opacity: 0;
    transform: translateY(10px);
    transition: all 0.3s ease;
    z-index: 1;
  }

  .mode-button:hover {
    background: rgba(88, 76, 58, 0.05);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(88, 76, 58, 0.15);
    border-color: #584c3a;
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

  /* Story Import Styles */
  .story-processor {
    max-width: 1200px;
    margin: 0 auto;
    padding: 2rem;
    height: calc(100vh - 6rem);
    display: flex;
    flex-direction: column;
  }

  .text-input-section {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
  }

  .text-input-container {
    display: flex;
    flex-direction: column;
    gap: 1rem;
    height: 100%;
  }

  .story-input {
    flex: 1;
    min-height: 200px;
    resize: none;
    font-family: monospace;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid rgba(255, 255, 255, 0.1);
    padding: 1rem;
    border-radius: 4px;
    line-height: 1.5;
  }

  .drop-zone-section {
    margin-top: 1rem;
    padding-bottom: 1rem;
  }

  .drop-zone-container {
    display: flex;
    gap: 1rem;
    align-items: flex-start;
  }

  .drop-zone {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 1rem;
    border: 2px dashed var(--accent-primary);
    border-radius: 8px;
    background: var(--bg-secondary);
    transition: all 0.3s ease;
    cursor: pointer;
    text-align: center;
    min-height: 100px;
  }

  .drop-zone.dragging {
    border-color: var(--accent-secondary);
    background: rgba(var(--accent-primary-rgb), 0.1);
    transform: scale(1.02);
  }

  .drop-zone .icon {
    font-size: 1.5rem;
    margin-bottom: 0.5rem;
  }

  .drop-zone .label {
    font-size: 1.1rem;
    font-weight: bold;
    margin-bottom: 0.25rem;
  }

  .drop-zone .description {
    color: var(--text-secondary);
    margin-bottom: 0.5rem;
    font-size: 0.9rem;
  }

  .browse-btn {
    background: var(--accent-primary);
    color: white;
    border: none;
    padding: 0.5rem 1rem;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.3s ease;
    font-size: 0.9rem;
  }

  .browse-btn:hover {
    background: var(--accent-secondary);
  }



  .section-label {
    font-size: 1.1rem;
    color: var(--text-secondary);
    margin-bottom: 1rem;
  }

  .import-btn {
    background: var(--accent-primary);
    color: white;
    border: none;
    padding: 0.75rem 1.5rem;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.3s ease;
    font-size: 1rem;
    white-space: nowrap;
    height: 100%;
    min-height: 100px;
    display: flex;
    align-items: center;
  }

  .import-btn:hover:not(:disabled) {
    background: var(--accent-secondary);
  }

  .import-btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .import-modal {
    max-width: 600px;
  }

  .existing-entries-list {
    max-height: 300px;
    overflow-y: auto;
    margin: 1rem 0;
    padding: 1rem;
    background: var(--bg-secondary);
    border-radius: 4px;
  }

  .existing-entry {
    margin-bottom: 1rem;
    padding-bottom: 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }

  .existing-entry:last-child {
    margin-bottom: 0;
    padding-bottom: 0;
    border-bottom: none;
  }

  .entry-preview {
    margin-top: 0.5rem;
    color: var(--text-secondary);
    font-size: 0.9rem;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .import-preview {
    margin: 1rem 0;
    padding: 1rem;
    background: var(--bg-primary);
    border-radius: 4px;
  }

  .import-preview .filename {
    font-weight: bold;
    margin-bottom: 0.5rem;
    color: var(--accent-primary);
  }

  .content-preview {
    font-family: monospace;
    white-space: pre-wrap;
    max-height: 300px;
    overflow-y: auto;
    padding: 1rem;
    background: var(--bg-secondary);
    border-radius: 4px;
    border: 1px solid rgba(255, 255, 255, 0.1);
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
  
  .notice-message {
    color: #f39c12;
    background: rgba(243, 156, 18, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-bottom: 1rem;
    border: 1px solid rgba(243, 156, 18, 0.2);
    font-weight: 500;
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
    position: relative;
    width: 100%;
    max-width: 800px;
    height: 30px;
    background: linear-gradient(to right, #8B4513, #A0522D, #8B4513);
    border-radius: 15px;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
    z-index: 10;
    border: 1px solid #5c2e11;
  }

  .scroll-stave-top {
  }

  .scroll-stave-bottom {
  }
  /* Add styles for Write Mode Layout */
  .write-view-container {
    display: flex;
    gap: 1rem;
    height: calc(100vh - 6rem); /* Adjust height as needed */
    padding: 1rem;
  }

  .write-left-panel {
    display: flex;
    flex-direction: column;
    width: 35%; /* Adjust width */
    gap: 1rem;
  }

  .write-chat-panel,
  .write-tools-panel {
    background: var(--bg-secondary);
    padding: 1rem;
    border-radius: 8px;
    display: flex;
    flex-direction: column;
  }

  .write-chat-panel {
    flex-grow: 1; /* Allow chat to take more space */
  }

  .write-right-panel {
    display: flex;
    flex-direction: column; /* Stack editor and preview */
    width: 65%; /* Adjust width */
    gap: 1rem;
  }

  .markdown-input {
    flex-grow: 1; /* Take available space */
    min-height: 200px; /* Minimum height */
    resize: vertical;
    font-family: monospace;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid rgba(255, 255, 255, 0.1);
    padding: 0.5rem;
    border-radius: 4px;
  }

  .markdown-preview-container {
    flex-grow: 1; /* Take available space */
    background: var(--bg-secondary);
    padding: 1rem;
    border-radius: 8px;
    overflow-y: auto;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .markdown-preview h1,
  .markdown-preview h2,
  .markdown-preview h3 {
    margin-top: 1em;
    margin-bottom: 0.5em;
    color: var(--text-primary);
    border-bottom: 1px solid rgba(255, 255, 255, 0.2);
    padding-bottom: 0.3em;
  }

  .markdown-preview p {
    margin-bottom: 1em;
    line-height: 1.6;
    color: var(--text-secondary);
  }

  .markdown-preview strong {
    font-weight: bold;
  }

  .markdown-preview em {
    font-style: italic;
  }

  .markdown-preview ul,
  .markdown-preview ol {
    margin-left: 2em;
    margin-bottom: 1em;
  }

  .markdown-preview code {
    background-color: rgba(255, 255, 255, 0.1);
    padding: 0.2em 0.4em;
    border-radius: 3px;
    font-family: monospace;
  }

  .markdown-preview pre {
    background-color: rgba(0, 0, 0, 0.2);
    padding: 1em;
    border-radius: 4px;
    overflow-x: auto;
  }

  .markdown-preview pre code {
    background-color: transparent;
    padding: 0;
  }
  /* Style for loading message */
  .message.loading {
    font-style: italic;
    color: var(--text-secondary);
  }

  .write-chat-form {
    display: flex;
    gap: 0.5rem;
  }

  .write-chat-form input {
    flex-grow: 1;
  }
</style>
