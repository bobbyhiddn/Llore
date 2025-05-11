<script lang="ts">
  import { onMount, afterUpdate } from 'svelte';
  import { database, llm } from '@wailsjs/go/models'; // Import namespaces
  import LibraryFileViewer from './LibraryFileViewer.svelte'; // Keep this separate modal
  // Import the new components
  import VaultSelector from './components/VaultSelector.svelte';
  import ModeSelector from './components/ModeSelector.svelte';
  import CodexView from './components/CodexView.svelte';
  import StoryImportView from './components/StoryImportView.svelte';
  import LibraryView from './components/LibraryView.svelte';
  import ChatView from './components/ChatView.svelte';
  import SettingsView from './components/SettingsView.svelte';
  import WriteView from './components/WriteView.svelte'; // Assuming WriteView is in components

  import {
    // Keep all backend functions needed by App or passed down
    GetAllEntries, CreateEntry, UpdateEntry, DeleteEntry,
    SelectVaultFolder, CreateNewVault, SwitchVault,
    GetCurrentVaultPath, ListLibraryFiles, ImportStoryTextAndFile, ReadLibraryFile,
    SaveLibraryFile, ProcessStory, ListChatLogs, LoadChatLog, SaveChatLog,
    FetchOpenRouterModelsWithKey, FetchOllamaModels, GetSettings, SaveSettings, SaveAPIKeyOnly, // Added FetchOllamaModels
    GetAIResponseWithContext, FetchOpenAIModels, FetchGeminiModels, // Added new model fetchers
  } from '@wailsjs/go/main/App';

  // --- Core App State ---
  let mode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write' | null = null;
  let vaultIsReady = false;
  let currentVaultPath: string | null = null;
  let isLoading = false; // General loading state
  let errorMsg = ''; // General error message
  let vaultErrorMsg = ''; // Specific vault errors for VaultSelector
  let initialErrorMsg = ''; // Error for initial vault screen

  // --- Shared State (used across multiple components) ---
  let openrouterApiKey = '';
  let chatModelId: string = ''; // Default chat model
  let storyProcessingModelId: string = ''; // Default story processing model
  let geminiApiKey: string = ''; // Gemini API key
  let activeMode: string = 'openrouter'; // Default processing mode
  let openaiApiKey: string = ''; // OpenAI API key
  let localEmbeddingModelName: string = ''; // Local embedding model name
  let modelList: llm.OpenRouterModel[] = [];
  let isModelListLoading = false;
  let modelListError = '';
  let settingsSaveMsg = ''; // Feedback for settings save
  let settingsErrorMsg = ''; // Error for settings save/load

  // --- Codex State ---
  let entries: database.CodexEntry[] = [];
  let currentEntry: database.CodexEntry | null = null; // Use null when no entry is selected
  let isEditing = false;
  let isGenerating = false; // AI content generation state
  let codexErrorMsg = ''; // Specific errors for CodexView
  
  // Add reactivity to track state changes
  $: if (isLoading !== undefined) {
    console.log('App.svelte - isLoading changed:', isLoading);
  }
  
  $: if (isEditing !== undefined) {
    console.log('App.svelte - isEditing changed:', isEditing);
  }
  
  $: if (currentEntry !== undefined) {
    console.log('App.svelte - currentEntry changed:', currentEntry?.id);
  }

  // --- Library State ---
  let libraryFiles: string[] = [];
  let isLibraryLoading = false;
  let libraryErrorMsg = ''; // Specific errors for LibraryView
  let showLibraryViewer = false;
  let viewingFilename = '';
  let viewingFileContent = '';

  // --- Story Import State ---
  let isProcessingStory = false; // Processing text area content
  let isProcessingImport = false; // Processing file content
  let storyImportErrorMsg = ''; // Specific errors for StoryImportView
  let storyImportSuccessMsg = ''; // Feedback for story import
  
  // Refs to child components to call methods
  let codexViewRef: CodexView | null = null;
  let chatViewRef: ChatView | null = null;
  let storyImportViewRef: StoryImportView | null = null;
  let settingsViewRef: SettingsView | null = null;
  
  // Variables for WriteView initial content when opening from library
  let writeViewInitialContent: string = '';
  let writeViewInitialFilename: string = '';

  // --- Interfaces --- (Keep if needed globally, or move to models.ts if applicable)
  interface OpenRouterConfig {
    openrouter_api_key: string;
    chat_model_id: string;
    story_processing_model_id: string;
    gemini_api_key: string;
    active_mode: string;
    openai_api_key: string;
    local_embedding_model_name: string;
  }

  // --- Initialization ---
  onMount(async () => {
    await fetchCurrentVaultPath();
    if (currentVaultPath) {
      vaultIsReady = true;
      // Load settings regardless of vault, API key might be global
      await loadSettings();
      // Don't auto-load data, wait for mode selection
    } else {
      vaultIsReady = false;
      mode = null; // Show vault selection
    }
  });

  // --- Vault Management ---
  async function fetchCurrentVaultPath() {
    console.log('fetchCurrentVaultPath called');
    isLoading = true;
    vaultErrorMsg = '';
    initialErrorMsg = '';
    try {
      currentVaultPath = await GetCurrentVaultPath();
      console.log('Current vault path:', currentVaultPath);
      vaultIsReady = !!currentVaultPath;
      if (!vaultIsReady) mode = null; // Reset mode if vault path is lost
    } catch (err) {
      console.warn("Could not get current vault path:", err);
      currentVaultPath = null;
      vaultIsReady = false;
      mode = null;
      initialErrorMsg = `Error checking vault status: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  async function handleNewLore() {
    isLoading = true;
    vaultErrorMsg = '';
    try {
      // Prompting logic might move to VaultSelector or stay here
      let vaultName = prompt('Enter a name for your new Lore Vault:', 'MyLoreVault');
      if (!vaultName) {
        vaultErrorMsg = 'Vault creation cancelled.';
        isLoading = false;
        return;
      }
      const newVaultPath = await CreateNewVault(vaultName);
      if (newVaultPath) {
        await SwitchVault(newVaultPath);
        await fetchCurrentVaultPath(); // Updates vaultIsReady
        await loadSettings(); // Reload settings in case vault-specific settings exist later
        // Reset data states
        entries = [];
        libraryFiles = [];
        mode = null; // Go to mode selection
      } else {
        vaultErrorMsg = 'Vault creation was cancelled or failed.';
      }
    } catch (err) {
      vaultErrorMsg = `Error creating new vault: ${err}`;
      vaultIsReady = false;
      mode = null;
    } finally {
      isLoading = false;
    }
  }

  async function handleLoadLore() {
    isLoading = true;
    vaultErrorMsg = '';
    try {
      const selectedPath = await SelectVaultFolder();
      if (selectedPath) {
        await SwitchVault(selectedPath);
        await fetchCurrentVaultPath(); // Updates vaultIsReady
        await loadSettings(); // Reload settings
        // Reset data states
        entries = [];
        libraryFiles = [];
        mode = null; // Go to mode selection
      } else {
        // User cancelled selection, do nothing
        vaultErrorMsg = '';
      }
    } catch (err) {
      vaultErrorMsg = `Error loading vault: ${err}`;
      vaultIsReady = false;
      mode = null;
    } finally {
      isLoading = false;
    }
  }

  // --- Mode Switching ---
  // Central function to handle all mode changes and associated actions
  async function setModeAndUpdate(newMode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write' | null) {
    console.log(`Setting mode via setModeAndUpdate: ${newMode}, current mode: ${mode}`);
    if (newMode !== mode) {
      console.log(`Mode changing from ${mode} to ${newMode}`);
      // Assign mode FIRST to trigger UI update
      mode = newMode;
      // Clear general/stale errors on mode change
      errorMsg = '';
      codexErrorMsg = '';
      libraryErrorMsg = '';
      storyImportErrorMsg = '';
      settingsErrorMsg = ''; // Keep settings save msg? Maybe clear too.
      settingsSaveMsg = '';

      // Reset specific states and load data AFTER mode variable is set
      if (newMode === 'codex') { // Check against newMode here
        currentEntry = null; // Deselect entry when entering codex view
        isEditing = false;
        try {
            await loadEntries(); // Load fresh entries
        } catch (err) {
            console.error("Error caught during loadEntries in setModeAndUpdate:", err);
            codexErrorMsg = `Failed to load Codex entries: ${err}`; // Show specific error
        }
      } else if (newMode === 'library') {
        try {
            await refreshLibraryFiles();
        } catch (err) {
             console.error("Error caught during refreshLibraryFiles in setModeAndUpdate:", err);
             libraryErrorMsg = `Failed to load Library files: ${err}`;
        }
      } else if (newMode === 'story') {
        // Reset story import specific state if necessary (handled mostly in component now)
      } else if (newMode === 'chat') {
        // Chat view handles its own loading/state reset on mount/activation
      } else if (newMode === 'settings') {
         try {
            await loadSettings(); // Ensure settings are fresh
         } catch (err) {
              console.error("Error caught during loadSettings in setModeAndUpdate:", err);
              settingsErrorMsg = `Failed to load Settings: ${err}`;
         }
      } else if (newMode === 'write') {
        // Reset write state if necessary (handled in component)
      }
    } else {
      console.log(`Mode ${newMode} is already active.`);
    }
  }

  // Specific handler for the 'setmode' event from ModeSelector
  function handleModeSelectEvent(event: CustomEvent<'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write'>) {
      setModeAndUpdate(event.detail);
  }

  // --- Settings & Models ---
  async function loadSettings() {
    console.log("Attempting to load settings...");
    isLoading = true;
    settingsErrorMsg = '';
    try {
      const settings = await GetSettings();
      console.log("Raw settings from backend:", settings);
      openrouterApiKey = settings.openrouter_api_key || '';
      chatModelId = settings.chat_model_id || '';
      storyProcessingModelId = settings.story_processing_model_id || '';
      geminiApiKey = settings.gemini_api_key || '';
      activeMode = settings.active_mode || 'openrouter'; // Default to openrouter if not set
      openaiApiKey = settings.openai_api_key || '';
      localEmbeddingModelName = settings.local_embedding_model_name || '';
      
      console.log("Settings loaded:", { 
        keySet: !!openrouterApiKey, 
        chatM: chatModelId, 
        storyM: storyProcessingModelId, 
        geminiKeySet: !!geminiApiKey,
        activeMode,
        openaiKeySet: !!openaiApiKey,
        localModelName: localEmbeddingModelName
      });
      
      // Load model list based on active mode, but preserve our loaded settings
      const currentChatModel = chatModelId;
      const currentStoryModel = storyProcessingModelId;
      
      await loadModelListForMode();
      
      // Ensure our loaded settings are preserved after model list load
      if (currentChatModel) chatModelId = currentChatModel;
      if (currentStoryModel) storyProcessingModelId = currentStoryModel;
      
      console.log('Final settings after model list load:', {
        chatModelId,
        storyProcessingModelId,
        activeMode
      });
    } catch (err) {
      settingsErrorMsg = `Error loading settings: ${err}`;
      console.error("Settings load error:", err);
    } finally {
      isLoading = false;
    }
  }
  
  // This function is defined later in the file
  
  // Load model list based on active mode
  async function loadModelListForMode(modeToLoadForParam?: string) {
    console.log("Loading model list for mode:", activeMode);
    const modeToUse = modeToLoadForParam || activeMode;
    console.log("App.svelte: Loading model list for effective mode:", modeToUse);
    
    isModelListLoading = true;
    modelListError = '';
    let newModelList: llm.OpenRouterModel[] = []; // Fetch into a temporary list

    try {
      if (modeToUse === 'openrouter' || modeToUse === 'hybrid') { 
        // Both openrouter and hybrid modes use OpenRouter for LLM
        if (openrouterApiKey) {
          console.log("Fetching OpenRouter models for " + modeToUse + " mode...");
          newModelList = await FetchOpenRouterModelsWithKey(openrouterApiKey) || [];
        } else {
          modelListError = 'Set OpenRouter API Key in Settings for ' + modeToUse + ' LLM mode.';
        }
      } else if (modeToUse === 'local') {
        console.log("Fetching Ollama models for local mode...");
        newModelList = await FetchOllamaModels() || [];
        if (newModelList.length === 0) {
          modelListError = 'No local Ollama models found. Ensure Ollama is running and models are pulled.';
        }
      } else if (modeToUse === 'openai') {
        if (openaiApiKey) {
          newModelList = await FetchOpenAIModels() || [];
        } else {
          modelListError = 'Set OpenAI API Key in Settings for OpenAI mode.';
        }
      } else if (modeToUse === 'gemini') {
        if (geminiApiKey) {
          newModelList = await FetchGeminiModels() || [];
        } else {
          modelListError = 'Set Gemini API Key in Settings for Gemini mode.';
        }
      }
      modelList = newModelList; // Assign the fetched list in one go

      // Default model selection in App.svelte's state.
      // This runs on initial load, after save, or when SettingsView requests a model list update.
      // It should only set defaults if the current selection is invalid or unset for the *App's activeMode*.
      if (modeToUse === activeMode) { // Only default if we are loading models for the App's actual activeMode
        let currentAppChatModel = chatModelId;
        let currentAppStoryModel = storyProcessingModelId;

        if (modelList.length > 0) {
            const chatModelIsValid = modelList.some(m => m.id === currentAppChatModel);
            const storyModelIsValid = modelList.some(m => m.id === currentAppStoryModel);

            if (!currentAppChatModel || !chatModelIsValid) {
                chatModelId = modelList.find(m=>m.id) ? modelList[0].id : '';
            }
            if (!currentAppStoryModel || !storyModelIsValid) {
                storyProcessingModelId = modelList.find(m=>m.id) ? modelList[0].id : '';
            }
        } else {
            chatModelId = '';
            storyProcessingModelId = '';
        }
      } else {
        // If loading models for a mode different from App's activeMode (e.g., SettingsView browsing),
        // App.svelte doesn't change its own chatModelId/storyProcessingModelId.
        // The new modelList is just passed to SettingsView.
      }
      console.log(`App.svelte: Model list updated for mode '${modeToUse}'. Models count: ${modelList.length}. App Chat: ${chatModelId}, App Story: ${storyProcessingModelId}`);

    } catch (err) {
      console.error(`Error fetching models for mode ${modeToUse}:`, err);
      modelListError = `Failed to load models for ${modeToUse} mode: ${err}`;
      modelList = []; // Clear list on error
      // If loading for App's activeMode, clear model selections
      if (modeToUse === activeMode) {
        chatModelId = '';
        storyProcessingModelId = '';
      }
    } finally {
      isModelListLoading = false;
    }
  }
  
  // Handler functions for SettingsView events
  async function handleLoadModels(event?: CustomEvent<{ modeToLoadFor?: string }>) {
    const modeForList = event?.detail?.modeToLoadFor || activeMode;
    await loadModelListForMode(modeForList);
  }
  
  function handleClearModels() {
    modelList = [];
    modelListError = '';
  }

  function handleClearErrors() {
    settingsErrorMsg = '';
    settingsSaveMsg = '';
  }
  
  function handleSettingsError(event: CustomEvent<string>) {
    settingsErrorMsg = event.detail;
  }

  async function handleSaveSettings(event: CustomEvent<OpenRouterConfig>) {
    console.log("Saving settings...", event.detail);
    isLoading = true;
    settingsSaveMsg = '';
    settingsErrorMsg = '';
    try {
      // Update local state
      openrouterApiKey = event.detail.openrouter_api_key;
      chatModelId = event.detail.chat_model_id;
      storyProcessingModelId = event.detail.story_processing_model_id;
      geminiApiKey = event.detail.gemini_api_key;
      activeMode = event.detail.active_mode;
      openaiApiKey = event.detail.openai_api_key;
      localEmbeddingModelName = event.detail.local_embedding_model_name;

      // Save to backend
      await SaveSettings({
        openrouter_api_key: openrouterApiKey,
        chat_model_id: chatModelId,
        story_processing_model_id: storyProcessingModelId,
        gemini_api_key: geminiApiKey,
        active_mode: activeMode,
        openai_api_key: openaiApiKey,
        local_embedding_model_name: localEmbeddingModelName
      });
      
      settingsSaveMsg = 'Settings saved successfully!';
      console.log("Settings saved successfully");
      
      // Load model list based on active mode
      await loadModelListForMode();
    } catch (err) {
      settingsErrorMsg = `Error saving settings: ${err}`;
      console.error("Settings save error:", err);
    } finally {
      isLoading = false;
      // Clear success message after 3 seconds
      if (settingsSaveMsg) {
        setTimeout(() => { settingsSaveMsg = ''; }, 3000);
      }
    }
  }

   // Handle API key save from ChatView modal
  async function handleApiKeySaved(event: CustomEvent<{key: string, mode: string}>) {
      const {key: newApiKey, mode: keyMode} = event.detail;
      console.log(`App.svelte received api key saved event for mode: ${keyMode}`);
      
      // Update the appropriate API key based on the mode
      if (keyMode === 'openrouter' || keyMode === 'local') {
          openrouterApiKey = newApiKey;
      } else if (keyMode === 'openai') {
          openaiApiKey = newApiKey;
      } else if (keyMode === 'gemini') {
          geminiApiKey = newApiKey;
      } else {
          console.warn(`Unknown mode for API key: ${keyMode}`);
          return;
      }
      
      // Save the API key to backend
      try {
          // Save only the API key that was updated
          await SaveAPIKeyOnly(newApiKey);
          
          settingsSaveMsg = `${keyMode.toUpperCase()} API Key updated!`; // Provide feedback
          
          // Reload models based on active mode
          await loadModelListForMode();
      } catch (err) {
          settingsErrorMsg = `Error saving API key: ${err}`;
          console.error("API key save error:", err);
      }
  }


  async function loadModelList() {
    if (!openrouterApiKey) {
      console.log("API key not set, skipping model list load.");
      modelListError = 'Set OpenRouter API Key in Settings first.';
      modelList = [];
      // Don't clear selected models if API key is missing, they might have been valid with a previous key.
      return;
    }
    console.log("Attempting to load models using key...");
    isModelListLoading = true;
    modelListError = '';
    try {
      const fetchedModels: llm.OpenRouterModel[] = await FetchOpenRouterModelsWithKey(openrouterApiKey);
      console.log("Raw fetched models:", fetchedModels);
      modelList = fetchedModels || [];
      
      // Preserve what was loaded from settings or previously saved by the user
      const currentConfiguredChatModel = chatModelId; 
      const currentConfiguredStoryModel = storyProcessingModelId;

      const chatModelExistsInFetchedList = modelList.some(m => m.id === currentConfiguredChatModel);
      const storyModelExistsInFetchedList = modelList.some(m => m.id === currentConfiguredStoryModel);
      
      console.log("Configured Chat Model:", currentConfiguredChatModel, "Exists in fetched list:", chatModelExistsInFetchedList);
      console.log("Configured Story Model:", currentConfiguredStoryModel, "Exists in fetched list:", storyModelExistsInFetchedList);
      
      // Handle Chat Model
      if (currentConfiguredChatModel && !chatModelExistsInFetchedList) {
        // If a chat model was configured but is not in the fetched list, keep it and warn.
        console.warn(`Configured chat model '${currentConfiguredChatModel}' not found in the latest fetched list. It will be kept, but may not be available or valid. Please verify in Settings.`);
        // chatModelId remains currentConfiguredChatModel
      } else if (!currentConfiguredChatModel && modelList.length > 0) {
        // Only set a default if no chat model was configured at all.
        chatModelId = modelList[0].id;
        console.log("No chat model was configured. Setting default chat model to:", chatModelId);
      }
      // If currentConfiguredChatModel exists in the list, chatModelId is already correctly set.

      // Handle Story Processing Model
      if (currentConfiguredStoryModel && !storyModelExistsInFetchedList) {
        // If a story model was configured but is not in the fetched list, keep it and warn.
        console.warn(`Configured story processing model '${currentConfiguredStoryModel}' not found in the latest fetched list. It will be kept, but may not be available or valid. Please verify in Settings.`);
        // storyProcessingModelId remains currentConfiguredStoryModel
      } else if (!currentConfiguredStoryModel && modelList.length > 0) {
        // Only set a default if no story model was configured at all.
        storyProcessingModelId = modelList[0].id;
        console.log("No story processing model was configured. Setting default story model to:", storyProcessingModelId);
      }
      // If currentConfiguredStoryModel exists in the list, storyProcessingModelId is already correctly set.
      
      console.log(`Fetched ${modelList.length} models. Final active selections: Chat=${chatModelId}, Story=${storyProcessingModelId}`);
    } catch (err) {
      console.error("Error fetching models:", err);
      modelListError = 'Failed to load models: ' + err;
      modelList = [];
      // On error fetching models, do not clear previously configured model IDs.
      // They might still be valid if the API error is temporary.
      console.warn("Failed to fetch model list. Previously configured models will be kept but might not be valid or available.");
    } finally {
      isModelListLoading = false;
    }
  }

  // --- Codex Actions ---
  async function loadEntries() {
    if (!vaultIsReady) return;
    isLoading = true;
    codexErrorMsg = '';
    try {
      entries = (await GetAllEntries()) || [];
    } catch (err) {
      console.error("Error loading entries:", err);
      codexErrorMsg = `Error loading entries: ${err}`;
    } finally {
      isLoading = false;
    }
  }

  function handleSelectEntry(event: CustomEvent<database.CodexEntry>) {
    currentEntry = event.detail;
    isEditing = true;
    codexErrorMsg = '';
  }

  function handleNewEntry() {
    // Resetting state for creating a new entry
    currentEntry = { id: 0, name: '', type: '', content: '', createdAt: '', updatedAt: '' }; // Provide a default structure
    isEditing = false;
    codexErrorMsg = '';
    // Focus handled within CodexView if needed
  }

  async function handleSaveEntry(event: CustomEvent<{ entryData: database.CodexEntry, isEditing: boolean }>) {
    const { entryData, isEditing: wasEditing } = event.detail;
    console.log(`handleSaveEntry called. WasEditing: ${wasEditing}`);
    console.log(`Entry data:`, entryData);
    
    // Important: Create local copies of state to avoid reactivity issues
    let tempCurrentEntry = currentEntry;
    let tempIsEditing = isEditing;
    
    // Set loading state
    isLoading = true;
    console.log(`isLoading set to true`);
    codexErrorMsg = '';
    
    try {
      // For existing entries (with ID > 0), use UpdateEntry
      if (wasEditing && entryData.id > 0) {
        console.log("Attempting to update entry:", entryData.id);
        await UpdateEntry(entryData);
        console.log("UpdateEntry returned successfully for:", entryData.id);
        alert('Entry updated successfully!');
        
        // Refresh entries list
        await loadEntries(); 
        console.log("loadEntries after update completed.");
        
        // Find the updated entry in the refreshed list
        const updatedEntry = entries.find(e => e.id === entryData.id) || null;
        console.log("Found updated entry:", updatedEntry?.id);
        
        // Update state in a specific order
        tempCurrentEntry = updatedEntry;
        tempIsEditing = !!updatedEntry;
      } 
      // For new entries (ID = 0 or wasEditing = false), use CreateEntry
      else {
        console.log("Attempting to create entry:", entryData.name);
        const newEntry = await CreateEntry(entryData.name, entryData.type, entryData.content);
        console.log("CreateEntry returned successfully. New ID:", newEntry.id);
        
        // Refresh entries list before trying to find the new entry
        await loadEntries();
        console.log("loadEntries after create completed.");
        
        // Find the newly created entry in the refreshed list
        const createdEntry = entries.find(e => e.id === newEntry.id);
        if (!createdEntry) {
          throw new Error("Failed to find newly created entry after refresh");
        }
        console.log("Found created entry:", createdEntry.id);
        
        // Update state in a specific order
        tempCurrentEntry = createdEntry;
        tempIsEditing = true; // Always set to true after creation to allow immediate editing
        
        alert(`Entry '${newEntry.name}' created successfully!`);
      }
    } catch (err) {
      console.error(`Error in handleSaveEntry (${wasEditing ? 'update' : 'create'})`, err);
      codexErrorMsg = `Failed to ${wasEditing ? 'update' : 'create'} entry: ${err}`;
      // Keep currentEntry and isEditing as they were before the failed attempt
    } finally {
      // Apply state changes after all operations are complete
      console.log("Setting final state values");
      currentEntry = tempCurrentEntry;
      isEditing = tempIsEditing;
      
      // IMPORTANT: Reset loading state last
      isLoading = false;
      console.log(`Final state: isLoading=${isLoading}, isEditing=${isEditing}, currentEntry=${currentEntry?.id}`);
    }
  }

  async function handleDeleteEntry(event: CustomEvent<number>) {
    const entryId = event.detail;
    console.log(`handleDeleteEntry called for ID: ${entryId}`);
    
    // Important: Create local copies of state to avoid reactivity issues
    let tempCurrentEntry = currentEntry;
    let tempIsEditing = isEditing;
    
    // Set loading state
    isLoading = true;
    console.log(`isLoading set to true in handleDeleteEntry`);
    codexErrorMsg = '';
    
    try {
      await DeleteEntry(entryId);
      console.log("DeleteEntry returned successfully for:", entryId);
      alert('Entry deleted successfully!');
      
      // Reset entry selection state
      tempCurrentEntry = null;
      tempIsEditing = false;
      
      // Refresh entries list
      await loadEntries();
      console.log("loadEntries after delete completed.");
    } catch (err) {
      console.error("Error deleting entry:", err);
      codexErrorMsg = `Failed to delete entry: ${err}`;
      // Keep currentEntry and isEditing as they were before the failed attempt
    } finally {
      // Apply state changes after all operations are complete
      console.log("Setting final state values in handleDeleteEntry");
      currentEntry = tempCurrentEntry;
      isEditing = tempIsEditing;
      
      // IMPORTANT: Reset loading state last
      isLoading = false;
      console.log(`Final state after delete: isLoading=${isLoading}, isEditing=${isEditing}, currentEntry=${currentEntry?.id}`);
    }
  }

  async function handleGenerateCodexContent(event: CustomEvent<{ entryData: database.CodexEntry, model: string }>) {
      const { entryData, model } = event.detail;
      console.log(`handleGenerateCodexContent called for entry: ${entryData.name}`);
      
      // Important: Create local copies of state to avoid reactivity issues
      let tempCurrentEntry = currentEntry;
      
      // Set generating state
      isGenerating = true;
      console.log(`isGenerating set to true`);
      codexErrorMsg = '';
      let generatedContent = '';

      // Construct prompt
      let prompt = `Expand ONLY on the following codex entry. Focus exclusively on this entry and do not create entries for other characters or locations. You may mention relationships to other entities, but only as they relate directly to this entry.\n\nName: ${entryData.name}\nType: ${entryData.type}\nContent: ${entryData.content || '(empty)'}\n\nProvide a single, detailed expansion of this entry. Do not format your response as JSON or include multiple entries.`;

      try {
          console.log(`Generating content for entry '${entryData.name}' using model ${model} with RAG context`);
          // Use GetAIResponseWithContext instead of GenerateOpenRouterContent to leverage RAG
          generatedContent = await GetAIResponseWithContext(prompt, model);
          console.log("Content generation with RAG successful");

          // Attempt to parse the JSON response
          let extractedText = generatedContent; // Default to the raw response
          try {
              const parsedResponse = JSON.parse(generatedContent);
              console.log("Successfully parsed generated content as JSON");
              
              if (typeof parsedResponse === 'string') {
                  // Case 1: The JSON parsed into a direct string
                  extractedText = parsedResponse;
                  console.log("Parsed response is a string");
              } else if (Array.isArray(parsedResponse)) {
                  // Case 2: The JSON parsed into an array
                  console.log("Parsed response is an array. Converting to formatted text...");
                  
                  // Convert array to formatted text
                  let formattedText = '';
                  parsedResponse.forEach((item, index) => {
                      if (typeof item === 'string') {
                          formattedText += item + (index < parsedResponse.length - 1 ? '\n\n' : '');
                      } else if (item && typeof item === 'object') {
                          // Handle objects in the array (like the example with name, type, content)
                          if (item.name && item.content) {
                              formattedText += `## ${item.name}${item.type ? ' (' + item.type + ')' : ''}\n\n${item.content}\n\n`;
                          } else {
                              // Generic object formatting
                              formattedText += JSON.stringify(item, null, 2) + (index < parsedResponse.length - 1 ? '\n\n' : '');
                          }
                      } else {
                          formattedText += String(item) + (index < parsedResponse.length - 1 ? '\n\n' : '');
                      }
                  });
                  
                  if (formattedText.trim() !== '') {
                      extractedText = formattedText;
                      console.log("Converted array to formatted text");
                  } else {
                      console.warn("Array conversion resulted in empty text. Using raw JSON.");
                      codexErrorMsg = 'AI response array could not be formatted. Displaying raw data. Check console.';
                  }
              } else if (parsedResponse && typeof parsedResponse === 'object') {
                  // Case 3: The JSON parsed into an object. Check common keys.
                  console.log("Parsed response is an object. Attempting to extract text...");
                  const textField = parsedResponse.response || 
                                  parsedResponse.text ||     
                                  parsedResponse.content ||  
                                  parsedResponse.completion || 
                                  (parsedResponse.choices && Array.isArray(parsedResponse.choices) && 
                                   parsedResponse.choices.length > 0 && parsedResponse.choices[0].text) || 
                                  (parsedResponse.choices && Array.isArray(parsedResponse.choices) && 
                                   parsedResponse.choices.length > 0 && parsedResponse.choices[0].message && 
                                   parsedResponse.choices[0].message.content);

                  if (typeof textField === 'string' && textField.trim() !== '') {
                      extractedText = textField;
                      console.log("Extracted text from object field");
                  } else {
                      console.warn("Parsed JSON object did not contain a recognized text field. Using raw JSON.");
                      // extractedText remains the original generatedContent
                      codexErrorMsg = 'AI response format not standard. Displaying raw data. Check console.';
                  }
              } else {
                  console.warn("Parsed JSON response was not a string, array, or object. Using raw content.");
                  // extractedText remains the original generatedContent
                  codexErrorMsg = 'Unexpected AI response type. Displaying raw data. Check console.';
              }
          } catch (e) {
              console.warn("Failed to parse generated content as JSON. Using raw content:", e);
              // extractedText remains the original generatedContent
          }

          // Update the current entry with the extracted or raw content
          if (isEditing && currentEntry && currentEntry.id === entryData.id) {
              tempCurrentEntry = { ...currentEntry, content: extractedText };
              console.log("Updated existing entry with processed content");
          } else if (!isEditing && entryData.id === 0) { // Handle generating for a new entry
              tempCurrentEntry = { ...entryData, content: extractedText };
              console.log("Updated new entry with processed content");
          }
      } catch (err) {
          console.error("Error generating content:", err);
          codexErrorMsg = `Error generating content: ${err}`;
      } finally {
          // Apply state changes after all operations are complete
          console.log("Setting final state values in handleGenerateCodexContent");
          currentEntry = tempCurrentEntry;
          
          // IMPORTANT: Reset generating state last
          isGenerating = false;
          console.log(`Final state after generate: isGenerating=${isGenerating}, currentEntry=${currentEntry?.id}`);
      }
  }


  // --- Library Actions ---
  async function refreshLibraryFiles() {
    if (!vaultIsReady) return;
    isLibraryLoading = true;
    libraryErrorMsg = '';
    try {
      libraryFiles = (await ListLibraryFiles()) || [];
    } catch (err) {
      console.error("Error loading library files:", err);
      libraryErrorMsg = `Error loading library: ${err}`;
      libraryFiles = [];
    } finally {
      isLibraryLoading = false;
    }
  }

  async function viewLibraryFileContent(event: CustomEvent<string>) {
    const filename = event.detail;
    if (!vaultIsReady) {
      libraryErrorMsg = 'No vault is currently loaded';
      return;
    }
    isLoading = true; // Use global loading indicator
    libraryErrorMsg = '';
    try {
      const content = await ReadLibraryFile(filename);
      viewingFilename = filename;
      viewingFileContent = content;
      showLibraryViewer = true; // Show the modal
    } catch (err) {
      console.error(`Error reading library file ${filename}:`, err);
      libraryErrorMsg = `Error reading file: ${err}`;
      alert(libraryErrorMsg); // Simple feedback
    } finally {
      isLoading = false;
    }
  }

  async function handleSaveLibraryFile(event: CustomEvent<{ filename: string, content: string }>) {
    const { filename, content } = event.detail;
    if (!vaultIsReady) {
      // Error handled within LibraryFileViewer? Or show here?
      alert('No vault is currently loaded');
      return;
    }
    isLoading = true; // Use global loading indicator
    // Clear specific errors?
    try {
      await SaveLibraryFile(filename, content);
      console.log(`Successfully saved ${filename}`);
      showLibraryViewer = false; // Close viewer on success
      // Optionally refresh library list if the saved file was new or renamed?
      // For simplicity, maybe just rely on manual refresh for now.
      // await refreshLibraryFiles();
    } catch (err) {
      console.error('Error saving library file:', err);
      // Show error within the modal?
      alert(`Failed to save file ${filename}: ${err}`);
    } finally {
      isLoading = false;
    }
  }

  // --- Story Import Actions ---
  async function handleImportStoryText(event: CustomEvent<{ content: string }>) {
    const { content } = event.detail;
    isProcessingStory = true;

    try {
      storyImportViewRef?.updateImportStatus('sending');
      
      // Call backend to process story text
      const result = await ProcessStory(content);
      
      if (result.existingEntries && result.existingEntries.length > 0) {
        // Show confirmation modal for existing entries
        storyImportViewRef?.showExistingConfirmation(result.existingEntries);
      } else {
        // Process directly if no existing entries
        storyImportViewRef?.updateImportStatus('library', 'Saving story to library...');
        await ImportStoryTextAndFile(content, ''); // Pass empty string for filename to use auto-generation
        storyImportViewRef?.showImportSuccess({
          newEntries: result.newEntries || [],
          updatedEntries: result.updatedEntries || []
        });
      }
    } catch (error: any) {
      console.error('Error importing story:', error);
      storyImportViewRef?.updateImportStatus('error', error.message || 'Failed to import story');
    } finally {
      isProcessingStory = false; // Keep separate flag for text area button state
    }
  }

  async function handleProcessImport(event: CustomEvent<{ content: string, filename: string, force?: boolean }>) {
    const { content, filename, force } = event.detail;

    try {
      if (storyImportViewRef) {
        storyImportViewRef.updateImportStatus('sending');
      }

      // First, save the file to the library with the provided filename
      if (storyImportViewRef) {
        storyImportViewRef.updateImportStatus('library', 'Saving story to library...');
      }
      
      // Save to library with the provided filename
      // This function both saves the file and processes it for codex entries
      const result = await ImportStoryTextAndFile(content, filename);
      
      // Refresh library files to ensure the new file appears in the list
      await refreshLibraryFiles();
      
      // Show success message
      if (storyImportViewRef) {
        storyImportViewRef.showImportSuccess({
          newEntries: result.newEntries || [],
          updatedEntries: result.updatedEntries || []
        });
      }

      // Refresh codex entries
      await loadEntries();
    } catch (error: any) {
      console.error('Error processing story import:', error);
      if (storyImportViewRef) {
        storyImportViewRef.setImportError(error.message || 'Failed to process story import');
      }
    }
  }

  // --- Event Handlers for Components ---
  function handleCodexError(event: CustomEvent<string>) {
      codexErrorMsg = event.detail;
  }
  function handleImportSuccess(event: CustomEvent<{ message: string }>) {
      storyImportSuccessMsg = event.detail.message;
  }
  function handleGenericError(event: CustomEvent<string>) {
      errorMsg = event.detail; // Used for Chat, Write errors
  }
   function handleWriteLoading(event: CustomEvent<boolean>) {
      isLoading = event.detail;
  }


  // --- Chat Actions ---
  async function handleSaveCodexFromChat(event: CustomEvent<string>) {
    const textToSave = event.detail;
    console.log("App.svelte received savecodex event with text:", textToSave.substring(0, 50) + "...");

    if (!textToSave) {
        chatViewRef?.setCodexSaveError("Cannot save empty text to Codex.");
        return;
    }

    // Use chatViewRef to update status immediately if possible
    // Note: ProcessStory doesn't have granular progress, so we mainly signal start/end.
    chatViewRef?.updateCodexSaveStatus('receiving', 'Processing text...'); // Or 'parsing'? Let's use receiving for now.

    try {
        // Call ProcessStory - assume it handles creating entries from arbitrary text.
        // The backend likely uses the default configured model and handles existing entries (force=false equivalent).
        console.log(`Calling ProcessStory for chat text...`);
        chatViewRef?.updateCodexSaveStatus('parsing', 'Finding codex entries...');
        const result = await ProcessStory(textToSave);
        console.log("ProcessStory result from chat text:", result);

        // Update ChatView with the result
        chatViewRef?.setCodexSaveResult(result);

        // Additionally, refresh the main codex list if the codex view might be visible
        // or just keep it simple and let the user refresh manually if needed.
        // Consider adding a small delay before potentially refreshing main list
        // if (mode === 'codex') {
        //     setTimeout(refreshCodexEntries, 500); // Refresh codex list if in codex mode
        // }

    } catch (err: any) { // Type the error
        const error = `Error processing chat text for Codex: ${err.message || String(err)}`;
        console.error(error);
        chatViewRef?.setCodexSaveError(error);
    } finally {
        // Ensure status isn't left hanging if ProcessStory fails without setting state
        // This might be redundant if setCodexSaveResult/Error always fire, but safer.
        // setTimeout(() => { if (chatViewRef?.codexSaveStatus !== 'complete' && chatViewRef?.codexSaveStatus !== 'error') chatViewRef?.updateCodexSaveStatus('idle'); }, 100);
    }
  }


  // --- Write Actions ---
  function handleWriteFileSaved(event: CustomEvent<string>) {
      const filename = event.detail;
      // Refresh library if the file might be new/relevant
      refreshLibraryFiles();
      // Show feedback? Maybe a temporary message bar?
      console.log(`Write file saved: ${filename}`);
  }
  
  // Handle opening a library file in Write mode
  async function handleEditInWriteMode(event: CustomEvent<string>) {
    const filename = event.detail;
    if (!vaultIsReady) {
      libraryErrorMsg = 'No vault is currently loaded';
      return;
    }
    isLoading = true;
    libraryErrorMsg = '';
    try {
      const content = await ReadLibraryFile(filename);
      // Set the initial content and filename for WriteView
      writeViewInitialContent = content;
      writeViewInitialFilename = filename;
      // Switch to write mode
      await setModeAndUpdate('write');
    } catch (err) {
      console.error(`Error reading library file ${filename} for write mode:`, err);
      libraryErrorMsg = `Error reading file: ${err}`;
      alert(libraryErrorMsg);
    } finally {
      isLoading = false;
    }
  }

  // Function to manually reset UI state when the debug button is clicked
  function handleResetState() {
    console.log("App.svelte: handleResetState called - manually resetting UI state");
    
    // Force reset all state variables that might be causing the UI to lock
    isLoading = false;
    isGenerating = false;
    isEditing = false;
    
    // Re-fetch entries to ensure we have the latest data
    loadEntries();
    
    // Log the state after reset
    console.log("State after reset:", { 
      isLoading, 
      isGenerating, 
      isEditing, 
      currentEntryId: currentEntry?.id 
    });
    
    // Show feedback to the user
    alert("UI state has been reset. Please try creating or editing an entry now.");
  }
  
  // --- Global Error Handling ---
  // Keep simple global handler or rely on specific error states?
  function handleError(message: string | Event, source?: string, lineno?: number, colno?: number, error?: Error) {
    console.error('Global error caught:', message, source, lineno, colno, error);
    // Display a generic error message, specific errors are handled by components/state vars
    errorMsg = `An application error occurred: ${message}${error ? ' (' + error.message + ')' : ''}. Check console.`;
    return true;
  }
  // window.onerror = handleError; // Uncomment if needed

</script>

<!-- Main App Layout -->
<div id="app-container">

  {#if !vaultIsReady}
    <VaultSelector
      bind:isLoading
      bind:initialErrorMsg={vaultErrorMsg}
      on:loadlore={handleLoadLore}
      on:newlore={handleNewLore}
    />
  {:else if mode === null}
    <ModeSelector on:setmode={handleModeSelectEvent} />
  {:else if mode === 'codex'}
    <CodexView
      bind:entries
      bind:currentEntry
      bind:isLoading
      bind:isEditing
      bind:isGenerating
      bind:errorMsg={codexErrorMsg}
      selectedModel={storyProcessingModelId}
      on:back={() => setModeAndUpdate(null)}
      on:selectentry={e => {
        currentEntry = e.detail;
        isEditing = true;
      }}
      on:newentry={() => {
        currentEntry = null;
        isEditing = true;
      }}
      on:saveentry={handleSaveEntry}
      on:deleteentry={handleDeleteEntry}
      on:generatecontent={handleGenerateCodexContent}
      on:resetstate={handleResetState}
      on:error={e => codexErrorMsg = e.detail}
    />
  {:else if mode === 'story'}
    <StoryImportView
      bind:this={storyImportViewRef}
      bind:isProcessingStory
      {vaultIsReady}
      on:back={() => setModeAndUpdate(null)}
      on:importstorytext={handleImportStoryText}
      on:processimport={handleProcessImport}
      on:error={(e) => storyImportViewRef?.setImportError(e.detail)}
      on:importsuccess={handleImportSuccess}
      on:gotocodex={() => setModeAndUpdate('codex')}
    />
  {:else if mode === 'library'}
    <LibraryView
      bind:libraryFiles
      bind:isLibraryLoading
      bind:errorMsg={libraryErrorMsg}
      on:back={() => setModeAndUpdate(null)}
      on:refresh={refreshLibraryFiles}
      on:viewfile={viewLibraryFileContent}
      on:editinwrite={handleEditInWriteMode}
    />
  {:else if mode === 'chat'}
    <ChatView
       bind:this={chatViewRef}
       {vaultIsReady}
       bind:modelList
       bind:isModelListLoading
       bind:modelListError
       initialSelectedModel={chatModelId}
       initialApiKey={openrouterApiKey}
       on:back={() => setModeAndUpdate(null)}
       on:savecodex={handleSaveCodexFromChat}
       on:refreshlogs={ListChatLogs}
       on:apikeysaved={handleApiKeySaved}
       on:error={handleGenericError}
    />
  {:else if mode === 'settings'}
    <SettingsView
      bind:this={settingsViewRef}
      initialApiKey={openrouterApiKey}
      initialChatModelId={chatModelId}
      initialStoryProcessingModelId={storyProcessingModelId}
      initialGeminiApiKey={geminiApiKey}
      initialActiveMode={activeMode}
      initialOpenAIAPIKey={openaiApiKey}
      initialLocalEmbeddingModelName={localEmbeddingModelName}
      {modelList}
      isModelListLoading={isModelListLoading}
      modelListError={modelListError}
      isLoading={isLoading}
      settingsSaveMsg={settingsSaveMsg}
      settingsErrorMsg={settingsErrorMsg}
      on:loadmodels={handleLoadModels}
      on:clearmodels={handleClearModels}
      on:savesettings={handleSaveSettings}
      on:clearerrors={handleClearErrors}
      on:error={handleSettingsError}
      on:back={() => setModeAndUpdate(null)}
    />
  {:else if mode === 'write'}
    <WriteView
      chatModelId={chatModelId}
      on:back={() => setModeAndUpdate(null)}
      on:filesaved={handleWriteFileSaved}
      on:loading={handleWriteLoading}
      on:error={handleGenericError}
      initialContent={writeViewInitialContent}
      initialFilename={writeViewInitialFilename}
    />
  {/if}

  <!-- Global Modals -->
  {#if showLibraryViewer}
    <LibraryFileViewer
      filename={viewingFilename}
      initialContent={viewingFileContent}
      on:close={() => showLibraryViewer = false}
      on:save={handleSaveLibraryFile}
    />
  {/if}

   <!-- Global Loading Indicator? -->
   {#if isLoading}
     <!-- <div class="global-loading">Loading...</div> -->
   {/if}
   <!-- Global Error Display? -->
   {#if errorMsg}
      <!-- <div class="global-error">{errorMsg}</div> -->
   {/if}


</div>

<style>
  /* Reset and Base Styles (Keep) */
  :global(body) {
    margin: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
    background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
    color: #e0e0e0;
    font-size: 16px;
    line-height: 1.6;
    height: 100vh;
    overflow: hidden; /* Prevent body scroll */
  }

  :global(#app) {
    height: 100%;
    display: flex; /* Ensure #app itself is flex if needed */
  }

   #app-container {
       height: 100%;
       display: flex;
       flex-direction: column;
       overflow: hidden; /* Prevent container scroll, children handle their own */
       position: relative; /* For absolute positioned children like back buttons */
       width: 100%; /* Ensure container takes full width */
    }
 
   :global(*) {
    box-sizing: border-box;
  }

  /* Variables (Keep) */
  :root {
    --accent-silver: #c0c0c0;
    --accent-gradient: linear-gradient(135deg, #6d5ed9, #8a7ef9);
    --accent-primary: #6d5ed9;
    --accent-secondary: #8a7ef9;
    --bg-primary: rgba(26, 26, 46, 0.95); /* Slightly less opaque */
    --bg-secondary: rgba(22, 33, 62, 0.95); /* Slightly less opaque */
    --text-primary: #e0e0e0;
    --text-secondary: #a0a0a0;
    --error-color: #ff4757;
    --success-color: #2ed573;
  }


  /* Keep scrollbar styles */
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

</style>
