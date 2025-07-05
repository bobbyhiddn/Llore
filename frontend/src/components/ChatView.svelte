<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate, onDestroy } from 'svelte';
  import { database, llm } from '@wailsjs/go/models'; // Import namespaces
  import {
    GetAIResponseWithContext, // For all LLM interactions
    ListChatLogs,
    LoadChatLog,
    SaveChatLog,
    DeleteChatLog, // For chat deletion
    GetAllEntries, // Needed for context injection
    ProcessStory, // Needed for save to codex
    CreateEntry, // Needed for save to codex
    SaveAPIKeyOnly // Needed for API key modal
  } from '@wailsjs/go/main/App';
  import StoryImportStatus from './StoryImportStatus.svelte'; // Import the status component
  import ChatMessageMenu from './ChatMessageMenu.svelte';

  // --- Props ---
  export let vaultIsReady: boolean = false;
  export let modelList: llm.OpenRouterModel[] = [];
  export let isModelListLoading: boolean = false;
  export let modelListError: string = '';
  export let initialSelectedModel: string = ''; // Model selected in settings/globally
  export let initialApiKey: string = ''; // API key from settings

  // --- Local State ---
  let chatInput = '';
  let isChatLoading = false; // Loading state for sending/receiving messages or loading logs
  let chatError = '';
  let chatDisplayElement: HTMLDivElement;
  let chatMessages: { sender: 'user' | 'ai', text: string }[] = [];
  // Removed: let chatContextInjected = false; // Context is now handled by backend

  // Chat Message Menu State
  let activeMenuMessageIndex: number | null = null;
  let menuStyle = '';
  let chatPanelElement: HTMLDivElement;

  // Codex Save Status (mirrors StoryImportView for consistency)
  type CodexSaveStatus = 'idle' | 'sending' | 'receiving' | 'parsing' | 'checking_existing' | 'updating' | 'embedding' | 'complete' | 'error';
  let codexSaveStatus: CodexSaveStatus = 'idle';
  let codexSaveError: string | null = null;
  let codexSaveNewEntry: database.CodexEntry | null = null; // Assuming only one entry is created at a time from chat
  let codexSaveUpdatedEntries: database.CodexEntry[] = []; // Track all updated entries

  // Chat Log Selection State
  let showChatSelection = true; // Start by showing selection
  let availableChatLogs: string[] = [];
  let currentChatLogFilename: string | null = null;
  let isLoadingChatLogs = false;
  let chatLogError = ''; // Error specific to loading logs

  // Save New Chat State
  let showSaveChatModal = false;
  let newChatFilename = '';
  let saveChatError = '';

  // Delete Chat State
  let showDeleteChatModal = false;
  let chatToDelete = '';
  let deleteChatError = '';

  // API Key Modal State (Simplified for chat view)
  let showApiKeyModal = false;
  let openrouterApiKey = initialApiKey; // Initialize with prop
  let openaiApiKey = ''; // For OpenAI mode
  let geminiApiKey = ''; // For Gemini mode
  let activeMode = 'openrouter'; // Default mode
  let apiKeyErrorMsg = '';
  let apiKeySaveMsg = '';

  // Model Selection State (Local to chat view)
  let selectedChatModel = initialSelectedModel; // Initialize with prop

  const dispatch = createEventDispatcher();

  // --- Lifecycle ---
  onMount(async () => {
    window.addEventListener('click', handleClickOutside, true);

    // When component mounts, immediately try to load chat logs if vault is ready
    if (vaultIsReady) {
      await initiateChatSelection();
    } else {
      chatLogError = 'Cannot load chat logs: No vault loaded.';
      showChatSelection = true; // Stay on selection screen if no vault
    }
    // Ensure local API key state is synced with prop on mount
    openrouterApiKey = initialApiKey;
    selectedChatModel = initialSelectedModel || (modelList.length > 0 ? modelList[0].id : '');
  });

  onDestroy(() => {
    window.removeEventListener('click', handleClickOutside, true);
  });

  // Keep local state synced with props when they change
  $: if (initialApiKey !== openrouterApiKey && !showApiKeyModal) openrouterApiKey = initialApiKey;
  $: if (initialSelectedModel !== selectedChatModel) selectedChatModel = initialSelectedModel;
  $: if (vaultIsReady && showChatSelection && availableChatLogs.length === 0 && !isLoadingChatLogs) {
      // If vault becomes ready later, try loading logs again
      initiateChatSelection();
  }

  // Auto-scroll chat display (but not when menu is open)
  afterUpdate(() => {
    if (chatDisplayElement && activeMenuMessageIndex === null) {
      chatDisplayElement.scrollTop = chatDisplayElement.scrollHeight;
    }
  });

  // --- Functions ---

  // --- Message Menu Functions ---
  function toggleMessageMenu(event: MouseEvent, index: number) {
    if (activeMenuMessageIndex === index) {
      activeMenuMessageIndex = null;
      return;
    }

    const button = event.currentTarget as HTMLElement;
    const buttonRect = button.getBoundingClientRect();
    
    // Position menu using viewport coordinates (fixed positioning)
    const menuWidth = 120;
    let left = buttonRect.left - menuWidth - 2;
    let top = buttonRect.top;
    
    // If no space on left, put it on the right
    if (left < 10) {
      left = buttonRect.right + 2;
    }
    
    // Keep menu on screen vertically
    if (top + 140 > window.innerHeight) {
      top = window.innerHeight - 150;
    }
    
    menuStyle = `position: fixed; top: ${top}px; left: ${left}px; z-index: 99999;`;
    activeMenuMessageIndex = index;
  }

  function handleMenuAction(event: CustomEvent) {
    const { detail: action } = event;
    if (activeMenuMessageIndex === null) return;

    const message = chatMessages[activeMenuMessageIndex];
    if (!message) return;

    switch (action) {
      case 'copy':
        navigator.clipboard.writeText(message.text);
        break;
      case 'index':
        saveChatToCodex(message.text);
        break;
    }
    activeMenuMessageIndex = null; // Close menu after action
  }

  function handleClickOutside(event: MouseEvent) {
    const target = event.target as HTMLElement;
    if (activeMenuMessageIndex !== null && !target.closest('.chat-message-menu') && !target.closest('.message-menu-btn')) {
      activeMenuMessageIndex = null;
    }
  }

  function goBack() {
    dispatch('back');
  }

  // --- Chat Log Selection Logic ---
  async function initiateChatSelection() {
    if (!vaultIsReady) {
      chatLogError = 'Cannot load chat logs: No vault loaded.';
      showChatSelection = true; // Ensure selection is shown
      return;
    }
    isLoadingChatLogs = true;
    showChatSelection = true; // Explicitly show selection UI
    chatLogError = '';
    availableChatLogs = [];
    try {
      availableChatLogs = (await ListChatLogs()) || [];
    } catch (err) {
      console.error('Error loading chat logs:', err);
      chatLogError = `Error loading chat logs: ${err}`;
    } finally {
      isLoadingChatLogs = false;
    }
  }

  function startNewChat() {
    chatMessages = [];
    currentChatLogFilename = null; // Indicate it's a new, unsaved chat
    // Removed: chatContextInjected = false;
    showChatSelection = false; // Hide selection UI
    chatError = '';
    chatLogError = ''; // Clear log errors too
  }

  async function loadSelectedChat(filename: string) {
    if (!filename) return;
    isChatLoading = true; // Use main chat loading indicator
    showChatSelection = false; // Hide selection UI
    chatError = '';
    chatLogError = '';
    try {
      const loadedMessages = (await LoadChatLog(filename)) || [];
      // Ensure the loaded data structure matches { sender: string, text: string }
      chatMessages = loadedMessages.map(msg => ({ sender: msg.sender as ('user' | 'ai'), text: msg.text }));
      currentChatLogFilename = filename;
      // Removed: chatContextInjected = true; // Context handling moved to backend
    } catch (err) {
      chatError = `Error loading chat log '${filename}': ${err}`;
      chatMessages = [];
      currentChatLogFilename = null;
    } finally {
      isChatLoading = false;
    }
  }

  // --- Main Chat Logic ---
  async function sendChat() {
    if (!chatInput.trim()) return;
    if (!openrouterApiKey) {
      openApiKeyModal(); // Prompt for key if missing
      chatError = 'OpenRouter API Key is required to chat.';
      return;
    }
    if (!selectedChatModel && modelList.length === 0) {
      chatError = 'No chat model selected or available. Please set API Key and select a model.';
      // Optionally open API key modal or settings view
      return;
    }
    if (!selectedChatModel && modelList.length > 0) {
      // Auto-select first model if none is chosen but list is available
      selectedChatModel = modelList[0].id;
    }

    chatError = '';
    isChatLoading = true;
    chatMessages = [...chatMessages, { sender: 'user', text: chatInput }];
    let userPrompt = chatInput;
    chatInput = '';

    try {
      // --- Call LLM with Context (RAG) ---
      const modelToUse = selectedChatModel; // Use the model selected in this view
      console.log(`Using chat model: ${modelToUse} with context-aware backend.`);

      // Use the new backend function which handles context building
      const aiReply = await GetAIResponseWithContext(userPrompt, modelToUse);
      const newAiMessage = { sender: 'ai' as const, text: aiReply };
      chatMessages = [...chatMessages, newAiMessage];
      // --- End Call LLM ---

      // --- Auto-Save Logic ---
      if (currentChatLogFilename) {
        // If a log is loaded, save the updated messages back to the same file
        await SaveChatLog(currentChatLogFilename, chatMessages);
      } else {
        // Maybe add a subtle indicator that the chat isn't being saved?
      }
    } catch (err) {
      chatError = `AI error: ${err}`;
      console.error("Chat send error:", err);
    } finally {
      isChatLoading = false;
    }
  }

  function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'Enter' && !event.shiftKey) {
      event.preventDefault(); // Prevent new line
      sendChat();
    }
  }

  // --- Save Chat / Codex ---
  async function saveChatToCodex(text: string) {
    if (!text) return;
    // Reset status before dispatching
    codexSaveStatus = 'sending'; // Indicate processing start
    codexSaveError = null;
    codexSaveNewEntry = null;
    codexSaveUpdatedEntries = [];
    console.log("Dispatching savecodex event for text:", text.substring(0, 50) + "...");
    dispatch('savecodex', text); // Let App.svelte handle the complex logic
  }

  // --- Functions called by parent (App.svelte) to update status ---
  export function setCodexSaveResult(result: { newEntries: database.CodexEntry[], updatedEntries: database.CodexEntry[] }) {
    codexSaveStatus = 'complete';
    codexSaveError = null;
    // Handle new entries (still expect only one)
    codexSaveNewEntry = result.newEntries?.length > 0 ? result.newEntries[0] : null;
    // Handle all updated entries
    codexSaveUpdatedEntries = result.updatedEntries || [];
    console.log("ChatView received codex save result:", result);
    // Optional: Automatically reset status after a delay?
    // setTimeout(() => { codexSaveStatus = 'idle'; }, 5000);
  }

  export function setCodexSaveError(message: string) {
    codexSaveStatus = 'error';
    codexSaveError = message;
    codexSaveNewEntry = null;
    codexSaveUpdatedEntries = []; // Clear the array
    console.error("ChatView received codex save error:", message);
  }

  // Generic status update (could be used for intermediate steps if needed)
  export function updateCodexSaveStatus(newStatus: CodexSaveStatus, message: string | null = null) {
    console.log(`ChatView codex status update: ${newStatus} - ${message}`);
    codexSaveStatus = newStatus;
    if (message) {
      codexSaveError = message;
    }
    // Clear updated entries list when starting a new save
    if (newStatus === 'sending') {
      codexSaveUpdatedEntries = [];
    }
    // Clear error if status is not error
    if (newStatus !== 'error') {
      // codexSaveError = null; // Keep error until explicit success or reset?
    }
    // Clear results if moving away from complete/error?
    if (newStatus !== 'complete' && newStatus !== 'error') {
      codexSaveNewEntry = null;
      codexSaveUpdatedEntries = [];
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
      newChatFilename = ''; // Clear input
      // Refresh the list of logs in the selection view if it's somehow still visible
      // (or just update the underlying list App.svelte might use)
      dispatch('refreshlogs'); // Ask parent to refresh log list
    } catch (err) {
      saveChatError = `Failed to save chat: ${err}`;
      console.error("Save new chat error:", err);
    } finally {
      isChatLoading = false;
    }
  }

  // --- Delete Chat Logic ---
  function promptDeleteChat(filename: string) {
    chatToDelete = filename;
    deleteChatError = '';
    showDeleteChatModal = true;
  }

  async function confirmDeleteChat() {
    if (!chatToDelete) return;
    
    isChatLoading = true;
    try {
      await DeleteChatLog(chatToDelete);
      showDeleteChatModal = false;
      chatToDelete = '';
      deleteChatError = '';
      // Refresh the list to remove the deleted chat
      await initiateChatSelection();
    } catch (err) {
      deleteChatError = `Failed to delete chat: ${err}`;
      console.error("Delete chat error:", err);
    } finally {
      isChatLoading = false;
    }
  }

  // --- API Key Modal Logic ---
  function openApiKeyModal(modeOrEvent: string | Event = 'openrouter') {
    // Reset modal state when opening
    apiKeySaveMsg = '';
    apiKeyErrorMsg = '';
    
    // Handle both direct string mode and event from button click
    if (typeof modeOrEvent === 'string') {
      activeMode = modeOrEvent;
    } else {
      // Default to openrouter when called from a button click
      activeMode = 'openrouter';
    }
    
    // Don't clear the keys, let user see/edit current ones
    showApiKeyModal = true;
  }

  async function saveApiKey() {
    apiKeyErrorMsg = '';
    apiKeySaveMsg = '';
    isChatLoading = true; // Indicate loading while saving from modal
    try {
      // Determine which API key to save based on the active mode
      let apiKeyToSave = '';
      if (activeMode === 'openrouter' || activeMode === 'local') {
        apiKeyToSave = openrouterApiKey;
      } else if (activeMode === 'openai') {
        apiKeyToSave = openaiApiKey;
      } else if (activeMode === 'gemini') {
        apiKeyToSave = geminiApiKey;
      } else {
        throw new Error(`Unknown mode: ${activeMode}`);
      }
      
      console.log(`Saving API key for ${activeMode} mode via SaveAPIKeyOnly from ChatView...`);
      await SaveAPIKeyOnly(apiKeyToSave); // Call the simpler backend function
      apiKeySaveMsg = `${activeMode.toUpperCase()} API key saved!`;
      console.log(`API key saved for ${activeMode} mode via SaveAPIKeyOnly.`);
      showApiKeyModal = false;
      
      // Dispatch event to notify parent (App.svelte) that key was updated
      // Include both the key and the mode to update the correct state variable
      dispatch('apikeysaved', {key: apiKeyToSave, mode: activeMode});
      // Parent should handle reloading model list globally
    } catch (err) {
      apiKeyErrorMsg = 'Failed to save API key: ' + err;
      console.error(`API key save error for ${activeMode} mode (SaveAPIKeyOnly):`, err);
    } finally {
      isChatLoading = false;
    }
  }
</script>

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>
<div class="chat-container">
  {#if showChatSelection}
    <section class="chat-log-selection">
      <h2>Select or Start a Chat</h2>
      <div class="log-actions">
        <button on:click={startNewChat} class="start-new-btn">Start New Chat</button>
        {#if !vaultIsReady}
          <p class="error-message">Load or create a vault first.</p>
        {/if}
      </div>
      {#if isLoadingChatLogs}
        <p>Loading chat logs...</p>
      {:else if chatLogError}
        <p class="error-message">{chatLogError}</p>
      {:else if availableChatLogs.length === 0 && vaultIsReady}
        <p class="empty-state">No saved chat logs found.</p>
      {:else if availableChatLogs.length > 0}
        <h3>Load Existing Chat:</h3>
        <ul class="log-list">
          {#each availableChatLogs as filename (filename)}
            <li class="log-item">
              <button on:click={() => loadSelectedChat(filename)} class="log-item-btn">{filename}</button>
              <button on:click={() => promptDeleteChat(filename)} class="delete-chat-btn" title="Delete this chat">🗑️</button>
            </li>
          {/each}
        </ul>
      {/if}
    </section>
  {:else}
    <section class="lore-chat chat-view-container">
      <div class="chat-header">
        <h2>Lore Chat {currentChatLogFilename ? `(${currentChatLogFilename})` : '(New Chat)'}</h2>
        <button class="select-chat-btn" on:click={initiateChatSelection} title="Load different chat or start new">Change Chat</button>
      </div>
      <div class="chat-settings-row">
        <label for="model-select">Model:</label>
        {#if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
          <button on:click={openApiKeyModal} class="inline-btn">Set API Key</button>
        {:else if modelList.length === 0}
          <span class="error-inline">No models found.</span>
          <button on:click={openApiKeyModal} class="inline-btn">Set API Key</button>
        {:else}
          <select id="model-select" bind:value={selectedChatModel}>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
          <button on:click={openApiKeyModal} class="inline-btn" title="Change API Key">API Key</button>
        {/if}
        {#if !currentChatLogFilename && chatMessages.length > 0}
          <button on:click={promptToSaveChat} class="inline-btn save-as-btn" title="Save current chat session">Save Chat As...</button>
        {/if}
      </div>

      <div class="chat-panel" bind:this={chatPanelElement}>
        <div class="chat-display" bind:this={chatDisplayElement}>
          {#each chatMessages as message, i (i)}
            <div class="message {message.sender}">
              <div class="message-header">
                <span class="sender-label">{message.sender === 'user' ? 'You' : 'AI'}</span>
                <button class="message-menu-btn" on:click|stopPropagation={(e) => toggleMessageMenu(e, i)}>
                  ⋮
                </button>
              </div>
              <div class="message-text">{message.text}</div>
            </div>
          {/each}

          {#if isChatLoading}
            <div class="message ai">
              <div class="message-header"><span class="sender-label">AI</span></div>
              <div class="message-text loading-dots">
                <span>.</span><span>.</span><span>.</span>
              </div>
            </div>
          {/if}
        </div>
        <!-- chat-display ends -->
      </div>
      <!-- chat-panel ends -->

      {#if chatError}
        <p class="error-message">{chatError}</p>
      {/if}

      <div class="codex-status-container">
        {#if codexSaveStatus !== 'idle'}
          <StoryImportStatus
            status={codexSaveStatus}
            errorMsg={codexSaveError}
            newEntries={codexSaveNewEntry ? [codexSaveNewEntry] : []}
            updatedEntries={codexSaveUpdatedEntries}
          />
          {#if codexSaveStatus === 'complete' || codexSaveStatus === 'error'}
            <div class="modal-buttons" style="margin-top: 1rem; justify-content: center;">
              <button on:click={() => { codexSaveStatus = 'idle'; }} class="save-btn">OK</button>
            </div>
          {/if}
        {/if}
      </div>

      <form on:submit|preventDefault={sendChat} class="chat-form">
        <input type="text" bind:value={chatInput} placeholder="Ask about your lore..." disabled={isChatLoading || !vaultIsReady || !openrouterApiKey} on:keydown={handleKeyDown}>
        <button type="submit" disabled={isChatLoading || !chatInput.trim() || !vaultIsReady || !openrouterApiKey}>Send</button>
      </form>
    </section>
  {/if}
</div>

<!-- Portal menu outside all containers -->
{#if activeMenuMessageIndex !== null}
  <div class="menu-portal" style={menuStyle}>
    <ChatMessageMenu
      className="chat-message-menu"
      showWeave={false}
      showInsert={false}
      on:action={handleMenuAction}
    />
  </div>
{/if}

<!-- Save New Chat Modal -->
{#if showSaveChatModal}
  <div class="modal-backdrop">
    <div class="modal save-chat-modal">
      <h3>Save New Chat Log</h3>
      <label for="chat-filename">Filename:</label>
      <input type="text" id="chat-filename" bind:value={newChatFilename} placeholder="Chat 2024-01-01.json">
      {#if saveChatError}
        <p class="error-message">{saveChatError}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={() => showSaveChatModal = false} class="cancel-btn">Cancel</button>
        <button on:click={saveNewChat} class="save-btn" disabled={isChatLoading}>Save</button>
      </div>
    </div>
  </div>
{/if}

<!-- Delete Chat Modal -->
{#if showDeleteChatModal}
  <div class="modal-backdrop">
    <div class="modal delete-chat-modal">
      <h3>Delete Chat Log</h3>
      <p>Are you sure you want to delete <strong>{chatToDelete}</strong>?</p>
      <p class="warning-text">This action cannot be undone.</p>
      {#if deleteChatError}
        <p class="error-message">{deleteChatError}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={() => showDeleteChatModal = false} class="cancel-btn">Cancel</button>
        <button on:click={confirmDeleteChat} class="delete-btn" disabled={isChatLoading}>Delete</button>
      </div>
    </div>
  </div>
{/if}

<!-- API Key Modal -->
{#if showApiKeyModal}
  <div class="modal-backdrop">
    <div class="modal api-key-modal">
      <h3>Set LLM API Key</h3>
      
      <!-- Mode selection tabs -->
      <div class="api-key-mode-tabs">
        <button 
          class={activeMode === 'openrouter' ? 'active' : ''} 
          on:click={() => activeMode = 'openrouter'}
          disabled={isChatLoading}
        >OpenRouter</button>
        <button 
          class={activeMode === 'openai' ? 'active' : ''} 
          on:click={() => activeMode = 'openai'}
          disabled={isChatLoading}
        >OpenAI</button>
        <button 
          class={activeMode === 'gemini' ? 'active' : ''} 
          on:click={() => activeMode = 'gemini'}
          disabled={isChatLoading}
        >Gemini</button>
        <button 
          class={activeMode === 'local' ? 'active' : ''} 
          on:click={() => activeMode = 'local'}
          disabled={isChatLoading}
        >Local</button>
      </div>
      
      <!-- API key input based on active mode -->
      {#if activeMode === 'openrouter' || activeMode === 'local'}
        <label for="openrouter-key">OpenRouter API Key:</label>
        <input 
          id="openrouter-key"
          type="password" 
          bind:value={openrouterApiKey} 
          placeholder="sk-..." 
          style="width: 100%; padding: 0.5em; margin-bottom: 1em;" 
        />
        <small>Required for OpenRouter and Local modes</small>
      {:else if activeMode === 'openai'}
        <label for="openai-key">OpenAI API Key:</label>
        <input 
          id="openai-key"
          type="password" 
          bind:value={openaiApiKey} 
          placeholder="sk-..." 
          style="width: 100%; padding: 0.5em; margin-bottom: 1em;" 
        />
      {:else if activeMode === 'gemini'}
        <label for="gemini-key">Gemini API Key:</label>
        <input 
          id="gemini-key"
          type="password" 
          bind:value={geminiApiKey} 
          placeholder="..." 
          style="width: 100%; padding: 0.5em; margin-bottom: 1em;" 
        />
      {/if}
      
      {#if apiKeySaveMsg}
        <p class="success-message">{apiKeySaveMsg}</p>
      {/if}
      {#if apiKeyErrorMsg}
        <p class="error-message">{apiKeyErrorMsg}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={saveApiKey} disabled={isChatLoading}>Save</button>
        <button on:click={() => { 
          showApiKeyModal = false; 
          apiKeyErrorMsg=''; 
          apiKeySaveMsg=''; 
          // Reset to initial values
          openrouterApiKey=initialApiKey; 
        }} disabled={isChatLoading}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
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
    z-index: 10;
  }

  .back-btn:hover {
    color: var(--text-primary);
  }

  .chat-container {
    padding: 1rem;
    padding-top: 4rem; /* Space for back button */
    height: calc(100vh - 2rem); /* Account for padding */
    display: flex;
    flex-direction: column;
    max-width: 1000px;
    margin: 0 auto;
    overflow: hidden;
  }

  /* Chat Log Selection */
  .chat-log-selection {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 2rem;
    background: var(--bg-secondary);
    border-radius: 8px;
    margin: auto; /* Center selection box */
    width: fit-content;
    max-width: 90%;
  }
  .chat-log-selection h2 {
    margin-bottom: 1.5rem;
    color: var(--text-primary);
  }
  .log-actions {
    margin-bottom: 1.5rem;
  }
  .start-new-btn {
    padding: 0.8rem 1.8rem;
    font-size: 1.1rem;
  }
  .chat-log-selection h3 {
    margin-top: 2rem;
    margin-bottom: 1rem;
    color: var(--text-secondary);
    font-size: 1rem;
    text-align: center;
    width: 100%;
  }
  .log-list {
    list-style: none;
    padding: 0;
    margin: 1rem 0;
  }
  .log-item {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.5rem;
  }
  .log-item-btn {
    flex: 1;
    padding: 0.75rem 1rem;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 6px;
    color: var(--text-primary);
    text-align: left;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  .log-item-btn:hover {
    background: rgba(255, 255, 255, 0.12);
    border-color: var(--accent-primary);
  }
  .delete-chat-btn {
    padding: 0.5rem;
    background: rgba(255, 71, 87, 0.1);
    border: 1px solid rgba(255, 71, 87, 0.3);
    border-radius: 4px;
    color: var(--error-color, #ff4757);
    cursor: pointer;
    transition: all 0.2s ease;
    font-size: 0.9rem;
  }
  .delete-chat-btn:hover {
    background: rgba(255, 71, 87, 0.2);
    border-color: var(--error-color, #ff4757);
  }
  .empty-state {
    color: var(--text-secondary);
    margin-top: 1rem;
  }

  /* Chat View */
  .chat-view-container {
    display: flex;
    flex-direction: column;
    height: 100%;
    background: var(--bg-secondary);
  }

  .chat-form {
    flex-shrink: 0; /* Prevent form from shrinking */
    display: flex;
    gap: 0.5rem;
  }

  .chat-form input {
    flex-grow: 1; /* Allow input to take available space */
    padding: 0.75rem;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 1rem;
  }
  .chat-form input:focus {
    outline: none;
    border-color: var(--accent-primary);
    background: rgba(255, 255, 255, 0.1);
  }

  .chat-form button {
    padding: 0.75rem 1.2rem;
    background: var(--accent-primary);
    border-radius: 6px;
    font-weight: 500;
  }
  .chat-form button:hover:not(:disabled) {
    background: var(--accent-secondary);
  }

  /* Modals */
  .modal-backdrop {
    position: fixed; inset: 0; background: rgba(0, 0, 0, 0.8);
    backdrop-filter: blur(4px); display: flex; align-items: center;
    justify-content: center; z-index: 1000; padding: 1rem;
  }
  .modal {
    background: var(--bg-primary); color: var(--text-primary);
    border-radius: 12px; padding: 1.5rem 2rem; width: 100%;
    max-width: 500px; margin: auto; box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
    border: 1px solid rgba(255, 255, 255, 0.1);
  }
  .modal h3 { margin-top: 0; margin-bottom: 1.5rem; color: var(--accent-primary); }
  .modal label { display: block; margin-bottom: 0.5rem; color: var(--text-secondary); }
  .modal input[type="text"], .modal input[type="password"] {
    width: 100%; padding: 0.75rem; background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.15); border-radius: 6px;
    color: var(--text-primary); font-size: 1rem; margin-bottom: 1rem;
  }
  
  /* API Key Modal Tabs */
  .api-key-mode-tabs {
    display: flex;
    margin-bottom: 1rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.15);
  }
  
  .api-key-mode-tabs button {
    padding: 0.5rem 1rem;
    background: transparent;
    border: none;
    border-bottom: 2px solid transparent;
    cursor: pointer;
    transition: all 0.2s ease;
    color: var(--text-secondary, #999);
  }
  
  .api-key-mode-tabs button.active {
    border-bottom: 2px solid var(--accent-primary, #4a90e2);
    color: var(--text-primary, #fff);
    font-weight: bold;
  }
  
  .api-key-mode-tabs button:hover:not(.active):not(:disabled) {
    background-color: rgba(255, 255, 255, 0.05);
    color: var(--text-primary, #fff);
  }
  
  .modal small {
    display: block;
    color: var(--text-secondary, #999);
    margin-top: -0.5rem;
    margin-bottom: 1rem;
    font-size: 0.8rem;
  }
  .modal-buttons { display: flex; justify-content: flex-end; gap: 1rem; margin-top: 1.5rem; }
  .modal-buttons button { padding: 0.6rem 1.2rem; }
  .delete-btn {
    background: var(--error-color, #ff4757) !important;
    color: white !important;
  }
  .delete-btn:hover:not(:disabled) {
    background: #ff3838 !important;
  }
  .warning-text {
    color: var(--error-color, #ff4757);
    font-size: 0.9rem;
    margin: 0.5rem 0;
  }

  .error-message {
    color: var(--error-color); background: rgba(255, 71, 87, 0.1);
    padding: 0.75rem 1rem; border-radius: 8px; margin-top: 1rem;
    border: 1px solid rgba(255, 71, 87, 0.2); font-size: 0.9rem;
  }
  .success-message {
    color: var(--success-color); background: rgba(46, 213, 115, 0.1);
    padding: 0.75rem 1rem; border-radius: 8px; margin-top: 1rem;
    border: 1px solid rgba(46, 213, 115, 0.2); font-size: 0.9rem;
  }

  /* Scrollbar */
  ::-webkit-scrollbar { width: 6px; }
  ::-webkit-scrollbar-track { background: rgba(255, 255, 255, 0.05); border-radius: 3px; }
  ::-webkit-scrollbar-thumb { background: var(--accent-primary); border-radius: 3px; }
  ::-webkit-scrollbar-thumb:hover { background: var(--accent-secondary); }

  .codex-status-container {
    flex-shrink: 0; /* Prevent shrinking */
    margin-top: 0.5rem; /* Space above status */
    max-height: 200px; /* Limit height */
    overflow-y: auto; /* Enable scrolling */
    border-radius: 8px; /* Match other containers */
  }
  /* Message header with button */
  .message-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 0.5rem;
  }
  
  /* Message Menu Button */
  .message-menu-btn {
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 1.2rem;
    line-height: 1;
    padding: 0.2rem 0.5rem;
    border-radius: 4px;
    margin-left: auto; /* Pushes button to the far right */
    opacity: 0; /* Hidden by default */
    transition: opacity 0.2s ease;
    flex-shrink: 0; /* Prevent button from shrinking */
  }
  
  .menu-portal {
    position: fixed;
    z-index: 99999;
    pointer-events: auto;
  }

  .message:hover .message-menu-btn {
    opacity: 1; /* Show on hover */
  }

  .message-menu-btn:hover {
    background-color: rgba(255, 255, 255, 0.1);
    color: var(--text-primary);
  }

  /* Chat display needs relative positioning for menu positioning context */
  .chat-display {
    position: relative;
    overflow: visible; /* Allow menu to overflow */
  }
  
  .chat-panel {
    overflow: visible; /* Allow menu to overflow */
  }

  /* ChatMessageMenu is positioned absolute relative to chat-display */
  :global(.chat-message-menu) {
    position: absolute;
    z-index: 1000;
  }
</style>