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
    GenerateOpenRouterContent, SelectVaultFolder, CreateNewVault, SwitchVault,
    GetCurrentVaultPath, ListLibraryFiles, ImportStoryTextAndFile, ReadLibraryFile,
    SaveLibraryFile, ProcessStory, ListChatLogs, LoadChatLog, SaveChatLog,
    FetchOpenRouterModelsWithKey, GetSettings, SaveSettings, SaveAPIKeyOnly,
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
  let storyImportViewRef: StoryImportView;
  let chatViewRef: ChatView;
  let settingsViewRef: SettingsView;

  // --- Chat State (Managed by ChatView, but some feedback might bubble up) ---
  // let chatError = ''; // Now local to ChatView mostly

  // --- Write State (Managed by WriteView) ---
  // let writeError = ''; // Now local to WriteView

  // --- Interfaces --- (Keep if needed globally, or move to models.ts if applicable)
  interface OpenRouterConfig {
    openrouter_api_key: string;
    chat_model_id: string;
    story_processing_model_id: string;
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
      console.log("Settings loaded:", { keySet: !!openrouterApiKey, chatM: chatModelId, storyM: storyProcessingModelId });
      // If API key is present, trigger model list load
      if (openrouterApiKey) {
        await loadModelList();
      } else {
        // Clear models if key is not set
        modelList = [];
        modelListError = '';
      }
    } catch (err) {
      settingsErrorMsg = `Error loading settings: ${err}`;
      console.error("Settings load error:", err);
    } finally {
      isLoading = false;
    }
  }

  async function handleSaveSettings(event: CustomEvent<OpenRouterConfig>) {
    console.log("Attempting to save settings...");
    isLoading = true;
    settingsErrorMsg = '';
    settingsSaveMsg = '';
    const settingsToSave = event.detail;
    try {
      await SaveSettings(settingsToSave);
      settingsSaveMsg = 'Settings saved successfully!';
      console.log("Settings saved successfully");
      // Update local state after successful save
      openrouterApiKey = settingsToSave.openrouter_api_key;
      chatModelId = settingsToSave.chat_model_id;
      storyProcessingModelId = settingsToSave.story_processing_model_id;
      // Reload models if API key might have changed
      if (openrouterApiKey) {
          await loadModelList();
      } else {
          modelList = []; // Clear models if key removed
          modelListError = '';
      }
    } catch (err) {
      settingsErrorMsg = `Error saving settings: ${err}`;
      console.error("Settings save error:", err);
    } finally {
      isLoading = false;
    }
  }

   // Handle API key save from ChatView modal
  async function handleApiKeySaved(event: CustomEvent<string>) {
      const newApiKey = event.detail;
      console.log("App.svelte received api key saved event");
      openrouterApiKey = newApiKey;
      settingsSaveMsg = 'API Key updated!'; // Provide feedback
      // Reload models
      if (openrouterApiKey) {
          await loadModelList();
      } else {
          modelList = [];
          modelListError = '';
      }
  }


  async function loadModelList() {
    if (!openrouterApiKey) {
      console.log("API key not set, skipping model list load.");
      modelListError = 'Set OpenRouter API Key in Settings first.';
      modelList = [];
      // Don't clear selected models here, let components handle defaults
      return;
    }
    console.log("Attempting to load models using key...");
    isModelListLoading = true;
    modelListError = '';
    try {
      const fetchedModels: llm.OpenRouterModel[] = await FetchOpenRouterModelsWithKey(openrouterApiKey);
      modelList = fetchedModels || [];
      // Set default models if they are not set or invalid
      if (!chatModelId || !modelList.some(m => m.id === chatModelId)) {
          chatModelId = modelList.length > 0 ? modelList[0].id : '';
      }
      if (!storyProcessingModelId || !modelList.some(m => m.id === storyProcessingModelId)) {
          // Maybe choose a different default for story processing? Or same as chat?
          storyProcessingModelId = modelList.length > 0 ? modelList[0].id : '';
      }
      console.log(`Fetched ${modelList.length} models. Defaults set: Chat=${chatModelId}, Story=${storyProcessingModelId}`);
    } catch (err) {
      console.error("Error fetching models:", err);
      modelListError = 'Failed to load models: ' + err;
      modelList = [];
      chatModelId = ''; // Clear selection on error
      storyProcessingModelId = '';
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
    
    // Important: Create local copies of state to avoid reactivity issues
    let tempCurrentEntry = currentEntry;
    let tempIsEditing = isEditing;
    
    // Set loading state
    isLoading = true;
    console.log(`isLoading set to true`);
    codexErrorMsg = '';
    
    try {
      if (wasEditing) {
        console.log("Attempting to update entry:", entryData.id);
        await UpdateEntry(entryData); // Assumes entryData includes the ID
        console.log("UpdateEntry returned successfully for:", entryData.id);
        alert('Entry updated successfully!'); // Simple feedback for now
        
        // Refresh entries list
        await loadEntries(); 
        console.log("loadEntries after update completed.");
        
        // Find the updated entry in the refreshed list
        const updatedEntry = entries.find(e => e.id === entryData.id) || null;
        console.log("Found updated entry:", updatedEntry?.id);
        
        // Update state in a specific order
        tempCurrentEntry = updatedEntry;
        tempIsEditing = !!updatedEntry;
      } else {
        console.log("Attempting to create entry:", entryData.name);
        const newEntry = await CreateEntry(entryData.name, entryData.type, entryData.content);
        console.log("CreateEntry returned successfully. New ID:", newEntry.id);
        alert(`Entry '${newEntry.name}' created successfully!`);
        
        // Refresh entries list
        await loadEntries();
        console.log("loadEntries after create completed.");
        
        // Find the newly created entry in the refreshed list
        const createdEntry = entries.find(e => e.id === newEntry.id) || null;
        console.log("Found created entry:", createdEntry?.id);
        
        // Update state in a specific order
        tempCurrentEntry = createdEntry;
        tempIsEditing = !!createdEntry;
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
      let prompt = `Expand on the following codex entry. Provide more details, background, or connections based on its name, type, and existing content.\n\nName: ${entryData.name}\nType: ${entryData.type}\nContent: ${entryData.content || '(empty)'}`;

      try {
          console.log(`Generating content for entry '${entryData.name}' using model ${model}`);
          generatedContent = await GenerateOpenRouterContent(prompt, model);
          console.log("Content generation successful");

          // Update the current entry with the generated content
          if (isEditing && currentEntry && currentEntry.id === entryData.id) {
              tempCurrentEntry = { ...currentEntry, content: generatedContent };
              console.log("Updated existing entry with generated content");
          } else if (!isEditing && entryData.id === 0) { // Handle generating for a new entry
              tempCurrentEntry = { ...entryData, content: generatedContent };
              console.log("Updated new entry with generated content");
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
        await ImportStoryTextAndFile(content);
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

  async function handleProcessImport(event: CustomEvent<{ content: string, force?: boolean }>) {
    const { content, force } = event.detail;

    try {
      if (storyImportViewRef) {
        storyImportViewRef.updateImportStatus('sending');
      }

      // Call ProcessStory and check for existing entries
      const result = await ProcessStory(content);

      if (!force && result.existingEntries && result.existingEntries.length > 0) {
        // Show confirmation modal for existing entries
        storyImportViewRef?.showExistingConfirmation(result.existingEntries);
      } else {
        // Process directly if no existing entries or force update
        if (storyImportViewRef) {
          storyImportViewRef.updateImportStatus('library', 'Saving story to library...');
        }

        // Save to library
        await ImportStoryTextAndFile(content);

        if (storyImportViewRef) {
          storyImportViewRef.showImportSuccess({
            newEntries: result.newEntries || [],
            updatedEntries: result.updatedEntries || []
          });
        }

        // Refresh codex entries
        await loadEntries();
      }
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
      bind:isLoading
      bind:settingsSaveMsg
      bind:settingsErrorMsg
      initialApiKey={openrouterApiKey}
      initialChatModelId={chatModelId}
      initialStoryProcessingModelId={storyProcessingModelId}
      bind:modelList
      bind:isModelListLoading
      bind:modelListError
      on:back={() => setModeAndUpdate(null)}
      on:savesettings={handleSaveSettings}
      on:loadmodels={loadModelList}
      on:clearmodels={() => { modelList = []; modelListError = ''; }}
      on:error={(e) => settingsErrorMsg = e.detail}
      on:clearerrors={() => { settingsErrorMsg = ''; settingsSaveMsg = ''; }}
    />
  {:else if mode === 'write'}
    <WriteView
       bind:isLoading
       chatModelId={chatModelId}
       on:back={() => setModeAndUpdate(null)}
       on:filesaved={handleWriteFileSaved}
       on:loading={handleWriteLoading}
       on:error={handleGenericError}
       initialContent=""
       initialFilename=""
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
