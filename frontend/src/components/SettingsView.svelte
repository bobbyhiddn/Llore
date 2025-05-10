<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { llm } from '@wailsjs/go/models'; // Import namespace

  // Props
  export let initialApiKey: string = ''; // This is OpenRouter API Key
  export let initialChatModelId: string = '';
  export let initialStoryProcessingModelId: string = '';
  export let initialGeminiApiKey: string = '';
  export let initialActiveMode: string = 'openrouter';
  export let initialOpenAIAPIKey: string = '';
  export let initialLocalEmbeddingModelName: string = ''; // This is Ollama model tag for local mode
  export let modelList: llm.OpenRouterModel[] = [];
  export let isModelListLoading: boolean = false;
  export let modelListError: string = '';
  export let isLoading: boolean = false; // General loading state from parent
  export let settingsSaveMsg: string = ''; // Passed from parent
  export let settingsErrorMsg: string = ''; // Passed from parent

  // Local State
  let openrouterApiKey: string = '';
  let chatModelId: string = '';
  let storyProcessingModelId: string = '';
  let geminiApiKey: string = '';
  let activeMode: string = '';
  let openaiApiKey: string = '';
  let localEmbeddingModelName: string = ''; // For Ollama model tag
  let showOpenRouterKey = false; // Renamed for clarity
  let showGeminiKey = false;
  let showOpenAIKey = false;

  const dispatch = createEventDispatcher();

  // Initialize local state from props on mount
  onMount(() => {
    openrouterApiKey = initialApiKey;
    chatModelId = initialChatModelId;
    storyProcessingModelId = initialStoryProcessingModelId;
    geminiApiKey = initialGeminiApiKey;
    activeMode = initialActiveMode || 'openrouter'; // Default to openrouter
    openaiApiKey = initialOpenAIAPIKey;
    localEmbeddingModelName = initialLocalEmbeddingModelName;

    // Request model list load if relevant API key is present and mode requires it
    handleModeSpecificModelLoad();
  });

  // Keep local model IDs in sync with props if user hasn't changed them
  let userChangedChatModel = false;
  let userChangedStoryModel = false;

  $: if (initialChatModelId !== chatModelId && !userChangedChatModel) chatModelId = initialChatModelId;
  $: if (initialStoryProcessingModelId !== storyProcessingModelId && !userChangedStoryModel) storyProcessingModelId = initialStoryProcessingModelId;


  function handleChatModelChange(event: Event) {
    userChangedChatModel = true;
    chatModelId = (event.target as HTMLSelectElement).value;
    console.log("Chat model changed to:", chatModelId);
  }

  function handleStoryModelChange(event: Event) {
    userChangedStoryModel = true;
    storyProcessingModelId = (event.target as HTMLSelectElement).value;
  }

  function onActiveModeChange() {
    dispatch('clearmodels'); // Clear any existing model list from other modes
    chatModelId = ''; // Reset selected models
    storyProcessingModelId = '';
    userChangedChatModel = false; // Reset user interaction flags for models
    userChangedStoryModel = false;
    handleModeSpecificModelLoad();
    dispatch('clearerrors'); // Clear save/error messages
  }

  function handleModeSpecificModelLoad() {
    // This function primarily triggers loading of LLM models (e.g., for OpenRouter).
    // Embedding provider selection is handled by the backend based on activeMode.
    if (activeMode === 'openrouter' || activeMode === 'local') { // 'local' uses OpenRouter for LLM
      if (openrouterApiKey && modelList.length === 0 && !isModelListLoading) {
        dispatch('loadmodels'); // This event is for OpenRouter models
      } else if (!openrouterApiKey) {
        dispatch('clearmodels');
      }
    } else if (activeMode === 'openai') {
      // OpenAI uses its own models; no separate list to load via OpenRouter dispatcher
      // If you had a specific event for OpenAI models: dispatch('loadopenaistuff');
      dispatch('clearmodels'); // Clear OpenRouter models if switching away
    } else if (activeMode === 'gemini') {
      // Gemini uses its own models for LLM if it were implemented for LLM;
      // no separate list to load via OpenRouter dispatcher for LLM side.
      dispatch('clearmodels'); // Clear OpenRouter models
    }
  }

  function goBack() {
    dispatch('back');
  }

  function saveSettings() {
    let hasError = false;
    let errorMessages: string[] = [];

    if (activeMode === 'openai' && !openaiApiKey) {
        errorMessages.push('OpenAI API Key is required for OpenAI mode.');
        hasError = true;
    }
    // For "gemini" mode, Gemini API key is crucial (for both LLM and embeddings if Gemini is the LLM)
    // For "openrouter" mode, if backend logic uses Gemini for embeddings, this key is also needed.
    if ((activeMode === 'gemini' || (activeMode === 'openrouter' && geminiApiKey)) && !geminiApiKey) {
      // A bit complex: If activeMode is "gemini", geminiApiKey is essential.
      // If activeMode is "openrouter", geminiApiKey might be used by backend for embeddings.
      // For simplicity, let's check if gemini is chosen or if it's openrouter *and* user provided a gemini key, then it must be valid.
      // The backend handles the actual logic of which embedding provider to use.
      // Here, we primarily validate required fields for the *selected mode*.
      if (activeMode === 'gemini'){
        errorMessages.push('Gemini API Key is required for Gemini mode.');
        hasError = true;
      }
    }
    if ((activeMode === 'openrouter' || activeMode === 'local') && !openrouterApiKey) {
        errorMessages.push('OpenRouter API Key is required for OpenRouter/Local LLM mode.');
        hasError = true;
    }
    if (activeMode === 'local' && !localEmbeddingModelName) {
        errorMessages.push('Ollama Embedding Model Tag is required for Local mode.');
        hasError = true;
    }

    // Model validation for OpenRouter-based LLM modes
    if ((activeMode === 'openrouter' || activeMode === 'local') && openrouterApiKey) {
        if (modelList.length > 0 && (!chatModelId || !storyProcessingModelId)) {
            errorMessages.push('Please select both a Chat Model and a Story Processing Model for OpenRouter.');
            hasError = true;
        }
        // If models haven't loaded (e.g., bad key), allow saving the key itself without model selection.
    }

    if (hasError) {
        dispatch('error', errorMessages.join(' '));
        return;
    }

    dispatch('savesettings', {
      openrouter_api_key: openrouterApiKey,
      chat_model_id: chatModelId,
      story_processing_model_id: storyProcessingModelId,
      gemini_api_key: geminiApiKey,
      active_mode: activeMode,
      openai_api_key: openaiApiKey,
      local_embedding_model_name: localEmbeddingModelName
    });
  }

  // Function to trigger model list loading, typically after OpenRouter API key is entered/changed
  function handleOpenRouterApiKeyChange() {
      dispatch('clearerrors');
      if (activeMode === 'openrouter' || activeMode === 'local') {
          if (openrouterApiKey) {
              dispatch('loadmodels'); // This is for OpenRouter models
          } else {
              dispatch('clearmodels');
          }
      }
  }

</script>

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>
<section class="settings">
  <h2>Settings</h2>
  <div class="settings-container">
    <form on:submit|preventDefault={saveSettings}>

      <!-- Active Processing Mode Selector -->
      <div class="form-group">
        <label for="activeModeSelect">Active Processing Mode:</label>
        <select id="activeModeSelect" bind:value={activeMode} on:change={onActiveModeChange}>
          <option value="openrouter">OpenRouter LLM + Configurable Embedding</option>
          <option value="openai">OpenAI LLM & OpenAI Embeddings</option>
          <option value="gemini">Gemini LLM & Gemini Embeddings</option>
          <option value="local">Local Embeddings (Ollama) + OpenRouter LLM</option>
        </select>
        <p class="help-text">Determines services for LLM and Embeddings. Backend decides embedding source for "OpenRouter" mode.</p>
      </div>
      
      <!-- OpenRouter API Key - Show for openrouter and local (LLM part) modes -->
      {#if activeMode === 'openrouter' || activeMode === 'local'}
      <div class="form-group">
        <label for="openrouterApiKey">OpenRouter API Key:</label>
        <div class="api-key-input">
          {#if showOpenRouterKey}
            <input
              type="text"
              id="openrouterApiKey"
              bind:value={openrouterApiKey}
              on:input={handleOpenRouterApiKeyChange} 
              placeholder="Enter your OpenRouter API key"
            />
          {:else}
            <input
              type="password"
              id="openrouterApiKey"
              bind:value={openrouterApiKey}
              on:input={handleOpenRouterApiKeyChange} 
              placeholder="Enter your OpenRouter API key"
            />
          {/if}
          <button
            type="button"
            class="toggle-visibility"
            on:click={() => (showOpenRouterKey = !showOpenRouterKey)}
            title={showOpenRouterKey ? "Hide API Key" : "Show API Key"}
          >
            {showOpenRouterKey ? "👁️" : "👁️‍🗨️"}
          </button>
        </div>
         <p class="help-text">Used for LLM features in 'OpenRouter' and 'Local' modes. Get from <a href="https://openrouter.ai/keys" target="_blank" rel="noopener noreferrer">OpenRouter.ai</a>.</p>
      </div>
      {/if}

      <!-- Gemini API Key Input - Show for 'gemini' mode (LLM & Embeddings) AND 'openrouter' mode (for potential Gemini embeddings via backend) -->
      {#if activeMode === 'gemini' || activeMode === 'openrouter'}
      <div class="form-group">
          <label for="geminiApiKey">Gemini API Key:</label>
          <div class="api-key-input">
            {#if showGeminiKey}
              <input
                type="text"
                id="geminiApiKey"
                bind:value={geminiApiKey}
                placeholder="Enter your Gemini API key"
              />
            {:else}
              <input
                type="password"
                id="geminiApiKey"
                bind:value={geminiApiKey}
                placeholder="Enter your Gemini API key"
              />
            {/if}
            <button
                type="button"
                class="toggle-visibility"
                on:click={() => showGeminiKey = !showGeminiKey}
                title={showGeminiKey ? "Hide Gemini Key" : "Show Gemini Key"}
            >
                {showGeminiKey ? "👁️" : "👁️‍🗨️"}
            </button>
          </div>
          <p class="help-text">
              {#if activeMode === 'gemini'}Required for Gemini LLM and Embeddings.{:else if activeMode === 'openrouter'}May be used by backend for embeddings in OpenRouter mode.{/if}
              Get from <a href="https://makersuite.google.com/app/apikey" target="_blank" rel="noopener noreferrer">Google AI Studio</a>.
          </p>
      </div>
      {/if}

      <!-- OpenAI API Key Input - Show only for 'openai' mode -->
      {#if activeMode === 'openai'}
      <div class="form-group">
          <label for="openaiApiKey">OpenAI API Key:</label>
          <div class="api-key-input">
            {#if showOpenAIKey}
              <input
                type="text"
                id="openaiApiKey"
                bind:value={openaiApiKey}
                placeholder="Enter your OpenAI API key"
              />
            {:else}
              <input
                type="password"
                id="openaiApiKey"
                bind:value={openaiApiKey}
                placeholder="Enter your OpenAI API key"
              />
            {/if}
            <button
                type="button"
                class="toggle-visibility"
                on:click={() => showOpenAIKey = !showOpenAIKey}
                title={showOpenAIKey ? "Hide API Key" : "Show API Key"}
            >
                {showOpenAIKey ? "👁️" : "👁️‍🗨️"}
            </button>
          </div>
          <p class="help-text">
              Required for OpenAI LLM and Embeddings. Get from
              <a href="https://platform.openai.com/api-keys" target="_blank" rel="noopener noreferrer">OpenAI API Keys</a>.
          </p>
      </div>
      {/if}

      <!-- Local Embedding Model Input (Ollama Tag) - Show only for 'local' mode -->
      {#if activeMode === 'local'}
      <div class="form-group">
          <label for="localEmbeddingModelName">Ollama Embedding Model Tag:</label>
          <input 
              type="text" 
              id="localEmbeddingModelName" 
              bind:value={localEmbeddingModelName} 
              placeholder="e.g., nomic-embed-text"
          />
          <p class="help-text">Exact tag of an Ollama model pulled locally (e.g., 'nomic-embed-text'). Ensure Ollama is running.</p>
      </div>
      {/if}

      <!-- OpenRouter Model selection fields - only show for modes that use OpenRouter for LLM -->
      {#if activeMode === 'openrouter' || activeMode === 'local'}
      <div class="form-group">
        <label for="chat-model-select">Default Chat Model (OpenRouter):</label>
        {#if !openrouterApiKey}
           <p class="info-text">Set OpenRouter API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
        {:else if modelList.length === 0}
           <span class="info-text">No models found or API key invalid. Refresh or check key.</span>
        {:else}
          <select
            id="chat-model-select"
            bind:value={chatModelId}
            on:change={handleChatModelChange}
            disabled={isModelListLoading}
          >
            <option value="" disabled={chatModelId !== ""}>Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
         <p class="help-text">Model used in Chat and Write views (via OpenRouter).</p>
      </div>

      <div class="form-group">
        <label for="story-processing-model-select">Story Processing Model (OpenRouter):</label>
         {#if !openrouterApiKey}
           <p class="info-text">Set OpenRouter API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
         {:else if modelList.length === 0}
           <span class="info-text">No models found or API key invalid. Refresh or check key.</span>
         {:else}
          <select
            id="story-processing-model-select"
            bind:value={storyProcessingModelId}
            on:change={handleStoryModelChange}
            disabled={isModelListLoading}
          >
            <option value="" disabled={storyProcessingModelId !== ""}>Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
         <p class="help-text">Model used for extracting Codex entries (via OpenRouter).</p>
      </div>
      {/if}
      
      <!-- For OpenAI and Gemini modes, LLM model choice is typically implicit or handled by fewer options not exposed here -->
      {#if activeMode === 'openai' || activeMode === 'gemini'}
      <div class="form-group">
        <p class="info-text">LLM model selection for {activeMode === 'openai' ? 'OpenAI' : 'Gemini'} mode is typically handled by the backend or has fewer choices not exposed here.</p>
      </div>
      {/if}

      <button type="submit" disabled={isLoading}>
          {#if isLoading}Saving...{:else}Save Settings{/if}
      </button>
    </form>
    
    {#if settingsSaveMsg}
      <p class="success-message">{settingsSaveMsg}</p>
    {/if}
    {#if settingsErrorMsg}
      <p class="error-message">{settingsErrorMsg}</p>
    {/if}
  </div>
</section>

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

  .settings {
    padding: 2rem;
    padding-top: 4rem; /* Space for back button */
    height: calc(100vh - 4rem); /* Adjust if header exists */
    overflow-y: auto;
  }

  h2 {
    margin-bottom: 1.5rem;
    color: var(--text-primary);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    padding-bottom: 0.5rem;
  }

  .settings-container {
    max-width: 700px;
    margin: 0 auto;
    display: flex;
    flex-direction: column;
    gap: 1rem; /* Gap between form groups */
  }

  .form-group {
    margin-bottom: 1.25rem; /* Increased space below each group */
    padding-bottom: 1rem; /* Space before potential border */
    border-bottom: 1px solid rgba(255, 255, 255, 0.05); /* Subtle separator */
  }
  .form-group:last-of-type {
      border-bottom: none; /* No border for the last group before buttons */
  }


  label {
    display: block;
    color: var(--text-secondary);
    margin-bottom: 0.5rem;
    font-weight: 500;
  }

  .api-key-input {
    position: relative;
    display: flex;
    align-items: center;
  }

  input[type="text"],
  input[type="password"],
  select {
    width: 100%;
    padding: 0.75rem;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    color: var(--text-primary);
    font-size: 1rem;
    transition: border-color 0.3s ease, background-color 0.3s ease;
  }
  select {
      /* Appearance for dropdown */
      appearance: none;
      -webkit-appearance: none; /* Safari and Chrome */
      -moz-appearance: none; /* Firefox */
      background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='%23a0a0a0' viewBox='0 0 16 16'%3E%3Cpath d='M7.247 11.14 2.451 5.658C1.885 5.013 2.345 4 3.204 4h9.592a1 1 0 0 1 .753 1.659l-4.796 5.48a1 1 0 0 1-1.506 0z'/%3E%3C/svg%3E");
      background-repeat: no-repeat;
      background-position: right 0.75rem center;
      background-size: 16px 12px;
      padding-right: 2.5rem; /* Space for arrow */
  }
   select:disabled {
       opacity: 0.5;
       background-color: rgba(255,255,255,0.02);
   }

  .api-key-input input { /* Specific to inputs within .api-key-input */
    padding-right: 3.5rem; /* Space for button */
  }

  input:focus,
  select:focus {
    outline: none;
    border-color: var(--accent-primary);
    background-color: rgba(255, 255, 255, 0.08);
    box-shadow: 0 0 0 2px rgba(109, 94, 217, 0.2); /* Accent focus ring */
  }

  .toggle-visibility {
    position: absolute;
    right: 0.25rem; /* Position inside the input */
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.5rem;
    font-size: 1.2rem; /* Make icon slightly larger */
    line-height: 1;
    border-radius: 4px; /* Add slight rounding */
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .toggle-visibility:hover {
    color: var(--text-primary);
    background-color: rgba(255,255,255,0.1); /* Subtle hover */
  }

  .help-text {
      font-size: 0.85rem;
      color: var(--text-secondary);
      margin-top: 0.5rem;
      line-height: 1.4;
  }
   .help-text a {
       color: var(--accent-secondary);
       text-decoration: none;
   }
    .help-text a:hover {
        text-decoration: underline;
    }

  .info-text {
      font-size: 0.9rem;
      color: var(--text-secondary);
      margin-top: 0.5rem;
      padding: 0.75rem 1rem; /* More padding */
      background: rgba(255, 255, 255, 0.03);
      border-radius: 6px; /* Match inputs */
      border: 1px solid rgba(255,255,255,0.07); /* Subtle border */
      line-height: 1.5;
  }

  button[type="submit"] {
    padding: 0.75rem 1.5rem;
    background: var(--accent-primary);
    border: none;
    border-radius: 8px;
    color: white;
    font-weight: 600;
    cursor: pointer;
    transition: all 0.3s ease;
    margin-top: 1rem; /* Space above save button */
    min-width: 150px; /* Ensure decent width */
  }

  button[type="submit"]:hover:not(:disabled) {
    background: var(--accent-secondary);
    transform: translateY(-1px);
    box-shadow: 0 4px 8px rgba(0,0,0,0.15); /* More pronounced shadow */
  }

  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
    background: var(--bg-secondary) !important; /* More muted disabled bg */
    box-shadow: none !important;
    transform: none !important;
  }

  .error-message, .success-message {
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    font-size: 0.9rem;
    text-align: center; /* Center messages */
  }

  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    border: 1px solid rgba(255, 71, 87, 0.2);
  }
   .error-inline {
      color: var(--error-color);
      font-size: 0.9rem;
      display: block; /* Make it block for better spacing */
      margin-top: 0.5rem;
  }

  .success-message {
    color: var(--success-color);
    background: rgba(46, 213, 115, 0.1);
    border: 1px solid rgba(46, 213, 115, 0.2);
  }

  /* Scrollbar */
  ::-webkit-scrollbar { width: 6px; }
  ::-webkit-scrollbar-track { background: rgba(255, 255, 255, 0.05); border-radius: 3px; }
  ::-webkit-scrollbar-thumb { background: var(--accent-primary); border-radius: 3px; }
  ::-webkit-scrollbar-thumb:hover { background: var(--accent-secondary); }
</style>