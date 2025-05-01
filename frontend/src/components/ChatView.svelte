<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate } from 'svelte';
  import { database, llm } from '@wailsjs/go/models'; // Import namespaces
  import {
    GenerateOpenRouterContent,
    ListChatLogs,
    LoadChatLog,
    SaveChatLog,
    GetAllEntries, // Needed for context injection
    ProcessStory, // Needed for save to codex
    CreateEntry, // Needed for save to codex
    SaveAPIKeyOnly // Needed for API key modal
  } from '@wailsjs/go/main/App';

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
  let chatContextInjected = false; // Track if codex context has been added

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

  // API Key Modal State (Simplified for chat view)
  let showApiKeyModal = false;
  let openrouterApiKey = initialApiKey; // Initialize with prop
  let apiKeyErrorMsg = '';
  let apiKeySaveMsg = '';

  // Model Selection State (Local to chat view)
  let selectedChatModel = initialSelectedModel; // Initialize with prop

  const dispatch = createEventDispatcher();

  // --- Lifecycle ---
  onMount(async () => {
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

  // Keep local state synced with props when they change
  $: if (initialApiKey !== openrouterApiKey && !showApiKeyModal) openrouterApiKey = initialApiKey;
  $: if (initialSelectedModel !== selectedChatModel) selectedChatModel = initialSelectedModel;
  $: if (vaultIsReady && showChatSelection && availableChatLogs.length === 0 && !isLoadingChatLogs) {
      // If vault becomes ready later, try loading logs again
      initiateChatSelection();
  }


  // Auto-scroll chat display
  afterUpdate(() => {
    if (chatDisplayElement) {
      chatDisplayElement.scrollTop = chatDisplayElement.scrollHeight;
    }
  });

  // --- Functions ---

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
    chatContextInjected = false;
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
      chatContextInjected = true; // Assume context was injected when log was saved
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
      // --- Context Injection Logic ---
      let promptToSend = userPrompt;
      if (!chatContextInjected) {
        let codexEntries: database.CodexEntry[] = [];
        try {
            codexEntries = await GetAllEntries();
        } catch (err) {
            console.warn("Could not load codex entries for context:", err);
            // Proceed without context if loading fails
        }

        let contextString = '';
        if (codexEntries && codexEntries.length > 0) {
          contextString = codexEntries.map(e => `Name: ${e.name}\nType: ${e.type}\nContent: ${e.content}`)
            .join('\n---\n');
          // Simple truncation, consider smarter context management later
          if (contextString.length > 4000) contextString = contextString.slice(0, 4000) + '\n...[Context Truncated]...';
        }
        if (contextString) {
          promptToSend = `Use the following context about my world to answer my questions:\n\n<context>\n${contextString}\n</context>\n\nUser Question: ${userPrompt}`;
        }
        chatContextInjected = true; // Mark context as injected for this session
      }

      // --- Call LLM ---
      const modelToUse = selectedChatModel; // Use the model selected in this view
      console.log(`Using chat model: ${modelToUse}`);

      const aiReply = await GenerateOpenRouterContent(promptToSend, modelToUse);
      const newAiMessage = { sender: 'ai' as const, text: aiReply };
      chatMessages = [...chatMessages, newAiMessage];

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

  // --- Save Chat / Codex ---
  async function saveChatToCodex(text: string) {
    dispatch('savecodex', text); // Let App.svelte handle the complex logic
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

  // --- API Key Modal Logic ---
  function openApiKeyModal() {
    // Reset modal state when opening
    apiKeySaveMsg = '';
    apiKeyErrorMsg = '';
    // Don't clear the key, let user see/edit current one
    // openrouterApiKey = '';
    showApiKeyModal = true;
  }

  async function saveApiKey() {
    apiKeyErrorMsg = '';
    apiKeySaveMsg = '';
    isChatLoading = true; // Indicate loading while saving from modal
    try {
      console.log("Saving API key via SaveAPIKeyOnly from ChatView...");
      await SaveAPIKeyOnly(openrouterApiKey); // Call the simpler backend function
      apiKeySaveMsg = 'API key saved!';
      console.log("API key saved via SaveAPIKeyOnly.");
      showApiKeyModal = false;
      // Dispatch event to notify parent (App.svelte) that key was updated
      dispatch('apikeysaved', openrouterApiKey);
      // Parent should handle reloading model list globally
    } catch (err) {
      apiKeyErrorMsg = 'Failed to save API key: ' + err;
      console.error("API key save error (SaveAPIKeyOnly):", err);
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
            <li>
              <button on:click={() => loadSelectedChat(filename)} class="log-item-btn">{filename}</button>
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
      <div class="chat-display chat-messages-area" bind:this={chatDisplayElement}>
        {#each chatMessages as message, i (i)}
          <div class="message {message.sender}">
            <strong class="sender-label">{message.sender === 'user' ? 'You' : 'AI'}:</strong>
            <span class="message-text">{message.text}</span>
            {#if message.sender === 'ai'}
              <button class="codex-btn" on:click={() => saveChatToCodex(message.text)} title="Attempt to save AI response to Codex">Save to Codex</button>
            {/if}
          </div>
        {/each}
        {#if isChatLoading && chatMessages.length > 0} <!-- Show thinking only after first message -->
          <div class="message ai thinking"><em>AI is thinking...</em></div>
        {/if}
         {#if chatMessages.length === 0 && !isChatLoading}
           <div class="empty-chat">Ask a question about your lore to get started!</div>
         {/if}
      </div>
      <form on:submit|preventDefault={sendChat} class="chat-form">
        <input type="text" bind:value={chatInput} placeholder="Ask about your lore..." disabled={isChatLoading || !vaultIsReady || !openrouterApiKey}>
        <button type="submit" disabled={isChatLoading || !chatInput.trim() || !vaultIsReady || !openrouterApiKey}>Send</button>
      </form>
      {#if chatError}
        <p class="error-message">{chatError}</p>
      {/if}
    </section>
  {/if}
</div>

<!-- Save New Chat Modal -->
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

<!-- API Key Modal -->
{#if showApiKeyModal}
  <div class="modal-backdrop">
    <div class="modal api-key-modal">
      <h3>Set OpenRouter API Key</h3>
      <input type="password" bind:value={openrouterApiKey} placeholder="sk-..." style="width: 100%; padding: 0.5em; margin-bottom: 1em;" />
      {#if apiKeySaveMsg}
        <p class="success-message">{apiKeySaveMsg}</p>
      {/if}
      {#if apiKeyErrorMsg}
        <p class="error-message">{apiKeyErrorMsg}</p>
      {/if}
      <div class="modal-buttons">
          <button on:click={saveApiKey} disabled={isChatLoading}>Save</button>
          <button on:click={() => { showApiKeyModal = false; apiKeyErrorMsg=''; apiKeySaveMsg=''; openrouterApiKey=initialApiKey; }} disabled={isChatLoading}>Cancel</button>
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
    height: calc(100vh - 4rem); /* Adjust if header exists */
    display: flex;
    flex-direction: column;
    max-width: 1000px;
    margin: 0 auto;
    overflow: hidden; /* Prevent container scroll */
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
      margin: 0;
      max-height: 40vh;
      overflow-y: auto;
      width: 100%;
      max-width: 400px;
      display: flex;
      flex-direction: column;
      gap: 0.5rem;
  }
  .log-item-btn {
      width: 100%;
      padding: 0.6rem 1rem;
      background: rgba(255, 255, 255, 0.08);
      border: 1px solid rgba(255, 255, 255, 0.1);
      color: var(--text-primary);
      text-align: left;
  }
   .log-item-btn:hover {
       background: rgba(255, 255, 255, 0.15);
   }
   .empty-state {
       color: var(--text-secondary);
       margin-top: 1rem;
   }


  /* Chat View */
  .chat-view-container {
    display: flex;
    flex-direction: column;
    height: 100%; /* Fill chat-container */
    overflow: hidden; /* Prevent whole section from scrolling */
  }

  .chat-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.5rem; /* Reduced margin */
      flex-shrink: 0;
  }
  .chat-header h2 {
      margin: 0;
      color: var(--text-primary);
      font-size: 1.4rem;
  }
  .select-chat-btn {
      padding: 0.4rem 0.8rem;
      font-size: 0.9rem;
      background: rgba(255, 255, 255, 0.1);
      color: var(--text-secondary);
  }
  .select-chat-btn:hover {
      background: rgba(255, 255, 255, 0.2);
      color: var(--text-primary);
  }


  .chat-settings-row {
    flex-shrink: 0; /* Prevent settings row from shrinking */
    margin-bottom: 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem; /* Spacing between items */
    flex-wrap: wrap; /* Allow wrapping */
    padding-bottom: 0.5rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }
  .chat-settings-row label {
      color: var(--text-secondary);
      margin-bottom: 0; /* Remove bottom margin */
  }
  .chat-settings-row select {
      padding: 0.4rem 0.8rem;
      background: var(--bg-secondary);
      color: var(--text-primary);
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 4px;
      max-width: 250px; /* Limit width */
  }
  .inline-btn {
      padding: 0.4rem 0.8rem;
      font-size: 0.9rem;
      background: rgba(255, 255, 255, 0.1);
      color: var(--text-secondary);
      margin-left: 0.5rem; /* Add some left margin */
  }
   .inline-btn:hover {
      background: rgba(255, 255, 255, 0.2);
      color: var(--text-primary);
  }
  .save-as-btn {
      background: #0984e3; /* Blue */
      color: white;
  }
  .save-as-btn:hover {
      background: #74b9ff;
  }
  .error-inline {
      color: var(--error-color);
      font-size: 0.9rem;
      margin-right: 0.5rem;
  }


  .chat-messages-area {
    flex: 1; /* Allow message area to grow and shrink */
    overflow-y: auto; /* Enable vertical scrolling */
    padding: 1rem;
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 8px;
    margin-bottom: 1rem;
    background: rgba(0,0,0,0.15); /* Slightly darker background */
  }
  .empty-chat {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: var(--text-secondary);
      font-style: italic;
  }

  .message {
    margin-bottom: 1rem;
    padding: 0.8rem 1.2rem; /* Slightly adjusted padding */
    border-radius: 12px;
    max-width: 85%; /* Slightly wider max */
    display: flex; /* Use flex for better alignment */
    flex-direction: column; /* Stack sender and text */
    position: relative; /* For button positioning */
  }

  .message.user {
    background: var(--accent-primary); /* Use primary accent */
    color: white;
    margin-left: auto;
    border-bottom-right-radius: 4px;
    align-items: flex-end; /* Align user text right */
  }

  .message.ai {
    background: var(--bg-secondary); /* Use secondary background */
    color: var(--text-primary);
    margin-right: auto;
    border-bottom-left-radius: 4px;
    align-items: flex-start; /* Align AI text left */
  }
  .message.thinking {
      background: transparent;
      color: var(--text-secondary);
      font-style: italic;
      padding: 0.5rem 0;
  }

  .sender-label {
      font-size: 0.8rem;
      color: rgba(255, 255, 255, 0.7);
      margin-bottom: 0.3rem;
  }
  .message.ai .sender-label {
      color: var(--accent-secondary); /* Different color for AI label */
  }

  .message-text {
      white-space: pre-wrap; /* Preserve line breaks */
      word-wrap: break-word; /* Break long words */
      line-height: 1.5;
  }

  .codex-btn {
      position: absolute;
      bottom: 5px;
      right: 8px;
      padding: 2px 6px;
      font-size: 0.75rem;
      background: rgba(255, 255, 255, 0.15);
      color: var(--text-secondary);
      border-radius: 3px;
      opacity: 0; /* Hidden by default */
      transition: opacity 0.2s ease;
  }
  .message.ai:hover .codex-btn {
      opacity: 1; /* Show on hover */
  }
  .codex-btn:hover {
      background: rgba(255, 255, 255, 0.3);
      color: var(--text-primary);
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
  .modal-buttons { display: flex; justify-content: flex-end; gap: 1rem; margin-top: 1.5rem; }
  .modal-buttons button { padding: 0.6rem 1.2rem; }

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

</style>