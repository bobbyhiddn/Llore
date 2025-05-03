<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { llm } from '@wailsjs/go/models'; // Import namespace

  // Props
  export let initialApiKey: string = '';
  export let initialChatModelId: string = '';
  export let initialStoryProcessingModelId: string = '';
  export let initialGeminiApiKey: string = ''; // Added for Gemini key
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
  let geminiApiKey: string = ''; // Added for Gemini key
  let showApiKey = false;
  let showGeminiKey = false; // Added for Gemini key visibility

  const dispatch = createEventDispatcher();

  // Initialize local state from props on mount and when props change
  onMount(() => {
    openrouterApiKey = initialApiKey;
    chatModelId = initialChatModelId;
    storyProcessingModelId = initialStoryProcessingModelId;
    geminiApiKey = initialGeminiApiKey; // Initialize Gemini key
    // Request model list load if API key is present but list is empty
    if (openrouterApiKey && modelList.length === 0 && !isModelListLoading) {
        dispatch('loadmodels');
    }
  });

  // $: if (initialApiKey !== openrouterApiKey && !isLoading) openrouterApiKey = initialApiKey;
  // Only update local model IDs from props if the user hasn't interacted
  $: if (initialChatModelId !== chatModelId && !userChangedChatModel) chatModelId = initialChatModelId;
  $: if (initialStoryProcessingModelId !== storyProcessingModelId && !userChangedStoryModel) storyProcessingModelId = initialStoryProcessingModelId;

  let userChangedChatModel = false;
  let userChangedStoryModel = false;

  function handleChatModelChange(event: Event) {
    userChangedChatModel = true;
    chatModelId = (event.target as HTMLSelectElement).value;
  }

  function handleStoryModelChange(event: Event) {
    userChangedStoryModel = true;
    storyProcessingModelId = (event.target as HTMLSelectElement).value;
  }

  function goBack() {
    dispatch('back');
  }

  function saveSettings() {
    // Basic validation
    if (!openrouterApiKey) {
        dispatch('error', 'OpenRouter API Key cannot be empty if you want to use AI features.');
        // Allow saving empty key if user intends to disable AI? Or enforce?
        // For now, let's allow saving an empty key but maybe warn.
    }
     if (openrouterApiKey && (!chatModelId || !storyProcessingModelId)) {
         if (modelList.length > 0) {
             // If models are loaded, prompt user to select models if key is present
             dispatch('error', 'Please select both a Chat Model and a Story Processing Model.');
             return; // Prevent saving if models are available but not selected
         }
         // If models haven't loaded (e.g., bad key), allow saving the key itself.
     }

    dispatch('savesettings', {
      openrouter_api_key: openrouterApiKey,
      chat_model_id: chatModelId,
      story_processing_model_id: storyProcessingModelId,
      gemini_api_key: geminiApiKey // Add Gemini key to payload
    });
  }

  // Function to trigger model list loading, typically after API key is entered/changed
  async function handleApiKeyChange() {
      // Clear previous errors/messages when key changes
      dispatch('clearerrors');
      // If key is present, trigger model load
      if (openrouterApiKey) {
          dispatch('loadmodels');
      } else {
          // If key is cleared, clear model list and errors locally? Or let parent handle?
          // Parent should handle clearing model list based on dispatched event.
          dispatch('clearmodels'); // Ask parent to clear models
      }
  }

</script>

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>
<section class="settings">
  <h2>Settings</h2>
  <div class="settings-container">
    <form on:submit|preventDefault={saveSettings}>
      <div class="form-group">
        <label for="apiKey">OpenRouter API Key:</label>
        <div class="api-key-input">
          {#if showApiKey}
            <input
              type="text"
              id="apiKey"
              bind:value={openrouterApiKey}
              on:change={handleApiKeyChange}
              placeholder="Enter your OpenRouter API key (e.g., sk-...)"
              autofocus
            />
          {:else}
            <input
              type="password"
              id="apiKey"
              bind:value={openrouterApiKey}
              on:change={handleApiKeyChange}
              placeholder="Enter your OpenRouter API key"
              autofocus
            />
          {/if}
          <button
            type="button"
            class="toggle-visibility"
            on:click={() => showApiKey = !showApiKey}
            title={showApiKey ? "Hide API Key" : "Show API Key"}
          >
            {#if showApiKey}👁️{:else}👁️‍🗨️{/if}
          </button>
        </div>
         <p class="help-text">Get your key from <a href="https://openrouter.ai/keys" target="_blank" rel="noopener noreferrer">OpenRouter.ai</a>. Required for AI features.</p>
      </div>

      <!-- Gemini API Key Input -->
      <div class="form-group">
          <label for="geminiApiKey">Gemini API Key (for embeddings):</label>
          <div class="api-key-input">
              {#if showGeminiKey}
                  <input
                      type="text"
                      id="geminiApiKey"
                      bind:value={geminiApiKey}
                      placeholder="Enter your Gemini API key (e.g., AIza...)"
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
                  {#if showGeminiKey}👁️{:else}👁️‍🗨️{/if}
              </button>
          </div>
          <p class="help-text">
              Required for intelligent search & context features. Get your API key from
              <a href="https://makersuite.google.com/app/apikey" target="_blank" rel="noopener noreferrer">Google AI Studio</a>.
          </p>
      </div>
      <!-- End Gemini API Key Input -->

      <div class="form-group">
        <label for="chat-model-select">Default Chat Model:</label>
        {#if !openrouterApiKey}
           <p class="info-text">Set API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
        {:else if modelList.length === 0}
           <span class="info-text">No models found or API key invalid.</span>
           <select
             id="chat-model-select"
             value={chatModelId}
             on:change={handleChatModelChange}
             disabled={isModelListLoading || !openrouterApiKey || modelList.length === 0}
           >
             <option value="" disabled selected>Select a model</option>
             {#each modelList as model}
               <option value={model.id}>{model.name}</option>
             {/each}
           </select>
        {:else}
          <select
            id="chat-model-select"
            value={chatModelId}
            on:change={handleChatModelChange}
            disabled={isModelListLoading || !openrouterApiKey || modelList.length === 0}
          >
            <option value="" disabled selected>Select a model</option>
            {#each modelList as model}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
         <p class="help-text">Model used in the main Chat view and Write mode chat.</p>
      </div>

      <div class="form-group">
        <label for="story-processing-model-select">Story Processing Model:</label>
         {#if !openrouterApiKey}
           <p class="info-text">Set API Key to load models.</p>
        {:else if isModelListLoading}
          <span>Loading models...</span>
        {:else if modelListError}
          <span class="error-inline">{modelListError}</span>
         {:else if modelList.length === 0}
           <span class="info-text">No models found or API key invalid.</span>
           <select
             id="story-processing-model-select"
             value={storyProcessingModelId}
             on:change={handleStoryModelChange}
             disabled={isModelListLoading || !openrouterApiKey || modelList.length === 0}
           >
             <option value="" disabled selected>Select a model</option>
             {#each modelList as model}
               <option value={model.id}>{model.name}</option>
             {/each}
           </select>
        {:else}
          <select
            id="story-processing-model-select"
            value={storyProcessingModelId}
            on:change={handleStoryModelChange}
            disabled={isModelListLoading || !openrouterApiKey || modelList.length === 0}
          >
            <option value="" disabled selected>Select a model</option>
            {#each modelList as model}
              <option value={model.id}>{model.name}</option>
            {/each}
          </select>
        {/if}
         <p class="help-text">Model used for extracting Codex entries from stories.</p>
      </div>

      <button type="submit" disabled={isLoading}>
          {#if isLoading}Saving...{:else}Save Settings{/if}
      </button>

      {#if settingsSaveMsg}
        <p class="success-message">{settingsSaveMsg}</p>
      {/if}
      {#if settingsErrorMsg}
        <p class="error-message">{settingsErrorMsg}</p>
      {/if}
    </form>
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
    margin-bottom: 1rem; /* Space below each group */
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
      background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' fill='%23a0a0a0' viewBox='0 0 16 16'%3E%3Cpath d='M7.247 11.14 2.451 5.658C1.885 5.013 2.345 4 3.204 4h9.592a1 1 0 0 1 .753 1.659l-4.796 5.48a1 1 0 0 1-1.506 0z'/%3E%3C/svg%3E");
      background-repeat: no-repeat;
      background-position: right 0.75rem center;
      background-size: 16px 12px;
      padding-right: 2.5rem; /* Space for arrow */
  }
   select:disabled {
       opacity: 0.5;
   }

  .api-key-input input {
    padding-right: 3.5rem; /* Space for button */
  }

  input:focus,
  select:focus {
    outline: none;
    border-color: var(--accent-primary);
    background-color: rgba(255, 255, 255, 0.08);
  }

  .toggle-visibility {
    position: absolute;
    right: 0.5rem; /* Position inside the input */
    top: 50%;
    transform: translateY(-50%);
    background: none;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: 0.5rem;
    font-size: 1.2rem;
    line-height: 1;
  }
  .toggle-visibility:hover {
    color: var(--text-primary);
  }

  .help-text {
      font-size: 0.85rem;
      color: var(--text-secondary);
      margin-top: 0.5rem;
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
      padding: 0.5rem;
      background: rgba(255, 255, 255, 0.03);
      border-radius: 4px;
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
  }

  button[type="submit"]:hover:not(:disabled) {
    background: var(--accent-secondary);
    transform: translateY(-1px);
  }

  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error-message, .success-message {
    padding: 0.75rem 1rem;
    border-radius: 8px;
    margin-top: 1rem;
    font-size: 0.9rem;
  }

  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    border: 1px solid rgba(255, 71, 87, 0.2);
  }
   .error-inline {
      color: var(--error-color);
      font-size: 0.9rem;
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