<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate, onDestroy } from 'svelte';
  import { Marked } from 'marked';
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
  import StoryImportStatus from '../Story/StoryImportStatus.svelte'; // Import the status component
  import ChatMessageMenu from './ChatMessageMenu.svelte';
  import '../../styles/ChatView.css';

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
  let chatMessages: { sender: 'user' | 'ai', text: string, html?: string }[] = [];
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
  const marked = new Marked({ 
    gfm: true, 
    breaks: true,
    pedantic: false,
    sanitize: false
  });

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
      chatMessages = loadedMessages.map(msg => ({ 
        sender: msg.sender as ('user' | 'ai'), 
        text: msg.text,
        html: msg.sender === 'ai' ? String(marked.parse(msg.text || '')) : undefined
      }));
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
      const newAiMessage = { sender: 'ai' as const, text: aiReply, html: String(marked.parse(aiReply)) };
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
              <div class="message-text">
                {#if message.sender === 'ai' && message.html}
                  {@html message.html}
                {:else}
                  {message.text}
                {/if}
              </div>
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

