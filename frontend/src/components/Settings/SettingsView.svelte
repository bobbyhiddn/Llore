<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { llm } from '@wailsjs/go/models';
  import { FetchOllamaModels } from '@wailsjs/go/main/App';

  type ProviderMode = 'openrouter' | 'openai' | 'gemini' | 'hybrid' | 'local';

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
  export let embeddingModelList: llm.OpenRouterModel[] = [];
  export let isEmbeddingModelListLoading: boolean = false;
  export let embeddingModelListError: string = '';
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
  
  // Store provider-specific model selections
  type ModelSelections = { chat: string; story: string };
  
  let providerModels: Record<ProviderMode, ModelSelections> = {
    openrouter: { chat: '', story: '' },
    openai: { chat: '', story: '' },
    gemini: { chat: '', story: '' },
    hybrid: { chat: '', story: '' },
    local: { chat: '', story: '' }
  };

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

    // Initialize the current mode's model selections
    const currentMode = activeMode as ProviderMode;
    providerModels[currentMode] = {
      chat: initialChatModelId,
      story: initialStoryProcessingModelId
    };

    // Request model list load if relevant API key is present and mode requires it
    handleModeSpecificModelLoad();
  });

  // Keep local model IDs in sync with props if user hasn't changed them
  let userChangedChatModel = false;
  let userChangedStoryModel = false;

  $: if (initialChatModelId !== chatModelId && !userChangedChatModel) chatModelId = initialChatModelId;
  $: if (initialStoryProcessingModelId !== storyProcessingModelId && !userChangedStoryModel) storyProcessingModelId = initialStoryProcessingModelId;


  function handleChatModelChange(event: Event) {
    const newValue = (event.target as HTMLSelectElement).value;
    if (newValue === '') return;
    
    userChangedChatModel = true;
    chatModelId = newValue;
    const currentMode = activeMode as ProviderMode;
    providerModels[currentMode].chat = chatModelId;
  }

  function handleStoryModelChange(event: Event) {
    const newValue = (event.target as HTMLSelectElement).value;
    if (newValue === '') return;
    
    userChangedStoryModel = true;
    storyProcessingModelId = newValue;
    const currentMode = activeMode as ProviderMode;
    providerModels[currentMode].story = storyProcessingModelId;
  }

  async function onActiveModeChange() {
    // Save current model selections for the previous mode
    if (chatModelId || storyProcessingModelId) {
      const previousMode = activeMode as ProviderMode;
      providerModels[previousMode] = {
        chat: chatModelId,
        story: storyProcessingModelId
      };
    }
    
    // Clear current models and errors
    dispatch('clearmodels');
    dispatch('clearerrors');
    
    // Reset selections before loading new models
    chatModelId = '';
    storyProcessingModelId = '';
    userChangedChatModel = false;
    userChangedStoryModel = false;

    // Load models for the new mode
    await handleModeSpecificModelLoad();

    // After models are loaded, restore saved selections if they exist
    const currentMode = activeMode as ProviderMode;
    const savedModels = providerModels[currentMode];
    if (savedModels && modelList.length > 0) {
      // Only restore if the saved model still exists in the new list
      if (savedModels.chat && modelList.some(m => m.id === savedModels.chat)) {
        chatModelId = savedModels.chat;
      }
      if (savedModels.story && modelList.some(m => m.id === savedModels.story)) {
        storyProcessingModelId = savedModels.story;
      }
    }
  }

  async function handleModeSpecificModelLoad() {
    // Clear any existing models first
    dispatch('clearmodels');
    
    // Load models based on active mode
    if (activeMode === 'openrouter' || activeMode === 'local') {
      if (openrouterApiKey && !isModelListLoading) {
        dispatch('loadmodels', { modeToLoadFor: activeMode });
      }
    } else if (activeMode === 'openai') {
      if (openaiApiKey && !isModelListLoading) {
        dispatch('loadmodels', { modeToLoadFor: activeMode });
      }
    } else if (activeMode === 'gemini') {
      if (geminiApiKey && !isModelListLoading) {
        dispatch('loadmodels', { modeToLoadFor: activeMode });
      }
    }

    // Load embedding models if in local mode
    if (activeMode === 'local') {
      await loadEmbeddingModels();
    }
  }

  async function loadEmbeddingModels() {
    isEmbeddingModelListLoading = true;
    embeddingModelListError = '';
    try {
      embeddingModelList = await FetchOllamaModels() || [];
      if (embeddingModelList.length === 0) {
        embeddingModelListError = 'No local Ollama models found for embeddings. Ensure Ollama is running and models are pulled.';
      }
    } catch (err: any) {
      embeddingModelListError = `Error loading embedding models: ${err.message}`;
      embeddingModelList = [];
    } finally {
      isEmbeddingModelListLoading = false;
    }
  }

  function goBack() {
    dispatch('back');
  }

  async function saveSettings() {
    dispatch('clearerrors');
    let hasError = false;
    let errorMessages: string[] = [];

    if (activeMode === 'openai' && !openaiApiKey) {
        errorMessages.push('OpenAI API Key is required for OpenAI mode.');
        hasError = true;
    }
    if ((activeMode === 'gemini' || (activeMode === 'openrouter' && geminiApiKey)) && !geminiApiKey) {
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

    if ((activeMode === 'openrouter' || activeMode === 'local') && openrouterApiKey) {
        if (modelList.length > 0 && (!chatModelId || !storyProcessingModelId)) {
            errorMessages.push('Please select both a Chat Model and a Story Processing Model.');
            hasError = true;
        }
    }

    if (hasError) {
        dispatch('error', errorMessages.join(' '));
        return;
    }

    // Save settings
    dispatch('savesettings', {
      openrouter_api_key: openrouterApiKey,
      chat_model_id: chatModelId,
      story_processing_model_id: storyProcessingModelId,
      gemini_api_key: geminiApiKey,
      active_mode: activeMode,
      openai_api_key: openaiApiKey,
      local_embedding_model_name: localEmbeddingModelName
    });

    // After saving, refresh the model list
    await handleModeSpecificModelLoad();
  }

  // Functions to trigger model list loading after API keys are entered/changed
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
  
  function handleOpenAIApiKeyChange() {
      dispatch('clearerrors');
      if (activeMode === 'openai') {
          if (openaiApiKey) {
              dispatch('loadmodels'); // This is for OpenAI models
          } else {
              dispatch('clearmodels');
          }
      }
  }
  
  function handleGeminiApiKeyChange() {
      dispatch('clearerrors');
      if (activeMode === 'gemini') {
          if (geminiApiKey) {
              dispatch('loadmodels'); // This is for Gemini models
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
          <option value="openrouter">OpenRouter LLM + Gemini Embeddings</option>
          <option value="openai">OpenAI LLM & OpenAI Embeddings</option>
          <option value="gemini">Gemini LLM & Gemini Embeddings</option>
          <option value="hybrid">OpenRouter LLM + Ollama Embeddings (Hybrid)</option>
          <option value="local">Ollama LLM & Embeddings (Offline Mode)</option>
        </select>
        <p class="help-text">Determines services for LLM and Embeddings. Backend decides embedding source for "OpenRouter" mode.</p>
      </div>
      
      <!-- OpenRouter API Key - Show for openrouter and hybrid modes -->
      {#if activeMode === 'openrouter' || activeMode === 'hybrid'}
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
        {#if activeMode === 'openrouter'}
          <p class="help-text">Used for LLM features in 'OpenRouter' mode. Get from <a href="https://openrouter.ai/keys" target="_blank" rel="noopener noreferrer">OpenRouter.ai</a>.</p>
        {:else if activeMode === 'hybrid'}
          <p class="help-text">Used for LLM features in 'Hybrid' mode (Ollama provides embeddings). Get from <a href="https://openrouter.ai/keys" target="_blank" rel="noopener noreferrer">OpenRouter.ai</a>.</p>
        {/if}
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
                on:input={handleGeminiApiKeyChange}
                placeholder="Enter your Gemini API key"
              />
            {:else}
              <input
                type="password"
                id="geminiApiKey"
                bind:value={geminiApiKey}
                on:input={handleGeminiApiKeyChange}
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
                on:input={handleOpenAIApiKeyChange}
                placeholder="Enter your OpenAI API key"
              />
            {:else}
              <input
                type="password"
                id="openaiApiKey"
                bind:value={openaiApiKey}
                on:input={handleOpenAIApiKeyChange}
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
          <label for="local-embedding-model-name">Ollama Embedding Model:</label>
          {#if isEmbeddingModelListLoading}
            <div class="loading-text">Loading embedding models...</div>
          {:else if embeddingModelListError}
            <div class="error-text">{embeddingModelListError}</div>
            <input
              type="text"
              id="local-embedding-model-name"
              bind:value={localEmbeddingModelName}
              placeholder="e.g. nomic-embed-text"
            />
          {:else}
            <select
              id="local-embedding-model-name"
              bind:value={localEmbeddingModelName}
              disabled={isEmbeddingModelListLoading}
            >
              <option value="">Select an embedding model</option>
              {#each embeddingModelList as model}
                <option value={model.id}>{model.name || model.id}</option>
              {/each}
            </select>
          {/if}
        </div>
      {/if}

      <!-- Model selection fields - show for modes that use OpenRouter or Ollama models -->
      {#if activeMode === 'openrouter' || activeMode === 'hybrid' || activeMode === 'local'}
      <div class="form-group">
        <label for="chat-model-select">
          Default Chat Model 
          {#if activeMode === 'openrouter'}(OpenRouter){/if}
          {#if activeMode === 'hybrid'}(OpenRouter){/if}
          {#if activeMode === 'local'}(Ollama){/if}:
        </label>
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
            value={chatModelId}
            on:change={handleChatModelChange}
            disabled={isModelListLoading}
          >
            <option value="">Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
         <p class="help-text">
          Model used in Chat and Write views 
          {#if activeMode === 'openrouter'}(via OpenRouter){/if}
          {#if activeMode === 'hybrid'}(via OpenRouter){/if}
          {#if activeMode === 'local'}(via local Ollama){/if}.
        </p>
      </div>

      <div class="form-group">
        <label for="story-processing-model-select">
          Story Processing Model 
          {#if activeMode === 'openrouter'}(OpenRouter){/if}
          {#if activeMode === 'hybrid'}(OpenRouter){/if}
          {#if activeMode === 'local'}(Ollama){/if}:
        </label>
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
            value={storyProcessingModelId}
            on:change={handleStoryModelChange}
            disabled={isModelListLoading}
          >
            <option value="">Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
         <p class="help-text">
          Model used for extracting Codex entries 
          {#if activeMode === 'openrouter'}(via OpenRouter){/if}
          {#if activeMode === 'hybrid'}(via OpenRouter){/if}
          {#if activeMode === 'local'}(via local Ollama){/if}.
        </p>
      </div>
      {/if}
      
      <!-- For OpenAI mode, show model selection -->
      {#if activeMode === 'openai'}
      <div class="form-group">
        <label for="openai-chat-model-select">Default Chat Model (OpenAI):</label>
        {#if !openaiApiKey}
          <p class="info-text">Set OpenAI API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
        {:else if modelList.length === 0}
          <span class="info-text">No models found or API key invalid. Refresh or check key.</span>
        {:else}
          <select
            id="openai-chat-model-select"
            value={chatModelId}
            on:change={handleChatModelChange}
            disabled={isModelListLoading}
          >
            <option value="">Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
        <p class="help-text">Model used in Chat and Write views (via OpenAI).</p>
      </div>

      <div class="form-group">
        <label for="openai-story-processing-model-select">Story Processing Model (OpenAI):</label>
        {#if !openaiApiKey}
          <p class="info-text">Set OpenAI API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
        {:else if modelList.length === 0}
          <span class="info-text">No models found or API key invalid. Refresh or check key.</span>
        {:else}
          <select
            id="openai-story-processing-model-select"
            value={storyProcessingModelId}
            on:change={handleStoryModelChange}
            disabled={isModelListLoading}
          >
            <option value="">Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
        <p class="help-text">Model used for extracting Codex entries (via OpenAI).</p>
      </div>
      {/if}
      
      <!-- For Gemini mode, show model selection -->
      {#if activeMode === 'gemini'}
      <div class="form-group">
        <label for="gemini-chat-model-select">Default Chat Model (Gemini):</label>
        {#if !geminiApiKey}
          <p class="info-text">Set Gemini API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
        {:else if modelList.length === 0}
          <span class="info-text">No models found or API key invalid. Refresh or check key.</span>
        {:else}
          <select
            id="gemini-chat-model-select"
            value={chatModelId}
            on:change={handleChatModelChange}
            disabled={isModelListLoading}
          >
            <option value="">Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
        <p class="help-text">Model used in Chat and Write views (via Gemini).</p>
      </div>

      <div class="form-group">
        <label for="gemini-story-processing-model-select">Story Processing Model (Gemini):</label>
        {#if !geminiApiKey}
          <p class="info-text">Set Gemini API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
        {:else if modelList.length === 0}
          <span class="info-text">No models found or API key invalid. Refresh or check key.</span>
        {:else}
          <select
            id="gemini-story-processing-model-select"
            value={storyProcessingModelId}
            on:change={handleStoryModelChange}
            disabled={isModelListLoading}
          >
            <option value="">Select a model</option>
            {#each modelList as model (model.id)}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
        <p class="help-text">Model used for extracting Codex entries (via Gemini).</p>
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