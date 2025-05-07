<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate } from 'svelte';
  import { Marked } from 'marked'; // Import Marked class
  import { SaveLibraryFile, GenerateOpenRouterContent, GetAIResponseWithContext } from '@wailsjs/go/main/App';

  // Props
  export let initialContent: string = ''; // If loading existing content
  export let initialFilename: string = ''; // If loading existing content
  export let chatModelId: string = ''; // From global settings
  export let isLoading: boolean = false; // Global loading state

  // Local State
  let writeContent: string = initialContent;
  let renderedWriteHtml = '';
  let markdownTextareaElement: HTMLTextAreaElement;
  let writeChatDisplayElement: HTMLDivElement; // For auto-scrolling

  // Write Mode Chat State
  let writeChatMessages: { sender: 'user' | 'ai', text: string }[] = [];
  let writeChatInput: string = '';
  let isWriteChatLoading: boolean = false;
  let writeChatError: string = '';

  // Save State
  let showWriteSaveModal: boolean = false;
  let writeFilename: string = initialFilename;
  let writeSaveError: string = '';
  let writeSaveSuccess: string = '';

  const dispatch = createEventDispatcher();
  const marked = new Marked(); // Create an instance

  // --- Lifecycle ---
  onMount(() => {
      // Initial render on mount
      renderMarkdown(writeContent);
      // Focus editor on mount?
      // markdownTextareaElement?.focus();
  });

  // Reactive Markdown Rendering
  $: {
    if (writeContent !== undefined) {
      renderMarkdown(writeContent);
    }
  }
  
  // Track editor view mode
  let editorMode: 'split' | 'edit' | 'preview' = 'split';

  // Auto-scroll chat display
  afterUpdate(() => {
    if (writeChatDisplayElement) {
       writeChatDisplayElement.scrollTop = writeChatDisplayElement.scrollHeight;
    }
  });

  // --- Functions ---

  function goBack() {
    dispatch('back');
  }

  function renderMarkdown(text: string) {
    try {
      // Use the marked instance with basic configuration
      // Cast the result to string to fix TypeScript error
      const result = marked.parse(text || '');
      renderedWriteHtml = typeof result === 'string' ? result : String(result);
    } catch (err) {
      console.error("Markdown rendering error:", err);
      // Fallback to plain text on error
      renderedWriteHtml = text.replace(/</g, "&lt;").replace(/>/g, "&gt;");
    }
  }

  // --- Write Mode Chat Function ---
  async function handleSendWriteChat() {
    if (!writeChatInput.trim() || isWriteChatLoading) return;
    if (!chatModelId) {
        writeChatError = 'No chat model selected in settings.';
        dispatch('error', 'No chat model selected in settings. Please configure it in the Settings view.');
        return;
    }

    const userMessage = writeChatInput.trim();
    writeChatMessages = [...writeChatMessages, { sender: 'user', text: userMessage }];
    writeChatInput = '';
    isWriteChatLoading = true;
    writeChatError = '';

    // Construct the prompt with context
    // Limit context and history to avoid overly large prompts
    const draftContext = writeContent.length > 2000 ? writeContent.substring(0, 2000) + "\n...[Draft Truncated]..." : writeContent;
    let prompt = `System: You are an AI assistant helping a user write. Here is their current draft:\n\n<draft>\n${draftContext}\n</draft>\n\nRecent Chat History:\n`;
    const historyLimit = 5; // Limit to last 5 messages (user + AI)
    writeChatMessages.slice(-historyLimit).forEach(msg => {
      prompt += `${msg.sender === 'user' ? 'User' : 'AI'}: ${msg.text}\n`;
    });
    // Ensure the latest user message (which triggered this) is included if not already in slice
    if (!writeChatMessages.slice(-historyLimit).some(m => m.sender === 'user' && m.text === userMessage)){
         prompt += `User: ${userMessage}\n`;
    }
    prompt += "AI:"; // Prompt the AI for its turn

    // Define modelToUse outside the try block so it's available in the catch block for fallback
    const modelToUse = chatModelId; // Use the model from props
    
    try {
      console.log(`Write Chat using model: ${modelToUse} with RAG context`);

      // Use GetAIResponseWithContext instead of GenerateOpenRouterContent to leverage RAG
      // This will automatically find and include relevant entries from the codex
      const aiReply = await GetAIResponseWithContext(prompt, modelToUse);
      writeChatMessages = [...writeChatMessages, { sender: 'ai', text: aiReply }];
      console.log("Received RAG-enhanced response for write chat");

    } catch (err) {
      console.error("Error in RAG-based write chat:", err);
      writeChatError = `AI error: ${err}`;
      // Fall back to direct generation if RAG fails
      try {
        console.log("Falling back to direct generation for write chat");
        const fallbackReply = await GenerateOpenRouterContent(prompt, modelToUse);
        writeChatMessages = [...writeChatMessages, { sender: 'ai', text: fallbackReply }];
        writeChatError = "Note: Used fallback generation method (RAG unavailable)";
      } catch (fallbackErr) {
        // If even the fallback fails, just show the original error
        writeChatError = `AI error: ${err}. Fallback also failed: ${fallbackErr}`;
      }
    } finally {
      isWriteChatLoading = false;
    }
  }

  // --- Write Mode Save Function ---
  function openSaveModal(saveAs = false) {
      writeSaveError = '';
      writeSaveSuccess = '';
      
      if (saveAs) {
        // For Save As, suggest a new filename based on the current one
        if (writeFilename) {
          // Remove extension if present
          const baseName = writeFilename.replace(/\.[^/.]+$/, '');
          // Suggest a new filename with "_copy" appended
          writeFilename = `${baseName}_copy.md`;
        } else if (writeContent.trim()) {
          // If no filename but content exists, suggest from first line
          const firstLine = writeContent.trim().split('\n')[0];
          writeFilename = firstLine.substring(0, 30).replace(/[^a-z0-9\s]/gi, '').replace(/\s+/g, '-') + '.md';
        }
      } else {
        // Regular save - suggest filename if empty and content exists
        if (!writeFilename && writeContent.trim()) {
          const firstLine = writeContent.trim().split('\n')[0];
          // Basic filename suggestion from first line
          writeFilename = firstLine.substring(0, 30).replace(/[^a-z0-9\s]/gi, '').replace(/\s+/g, '-') + '.md';
        }
      }
      
      showWriteSaveModal = true;
  }

  async function handleSaveWriteContent() {
    if (!writeFilename.trim()) {
      writeSaveError = 'Filename cannot be empty.';
      return;
    }
    let filenameToSave = writeFilename.trim();
    // Ensure .md extension
    if (!filenameToSave.toLowerCase().endsWith('.md')) {
      filenameToSave += '.md';
    }

    // Reset messages
    writeSaveError = '';
    writeSaveSuccess = '';
    dispatch('loading', true); // Use dispatch to signal loading start

    try {
      // Use the imported SaveLibraryFile function directly
      await SaveLibraryFile(filenameToSave, writeContent);
      writeSaveSuccess = `File '${filenameToSave}' saved successfully!`;
      showWriteSaveModal = false; // Close modal on success
      // Don't clear filename, user might want to save again
      dispatch('filesaved', filenameToSave); // Notify parent
    } catch (err) {
      writeSaveError = `Failed to save file: ${err}`;
      console.error("Save Write Content Error:", err);
    } finally {
      dispatch('loading', false); // Use dispatch to signal loading end
    }
  }

  // --- Write Mode Formatting Tools Function ---
  function applyMarkdownFormat(formatType: 'bold' | 'italic' | 'h1' | 'h2' | 'h3' | 'code' | 'blockquote') {
    if (!markdownTextareaElement) return;

    const start = markdownTextareaElement.selectionStart;
    const end = markdownTextareaElement.selectionEnd;
    const selectedText = writeContent.substring(start, end);
    let prefix = '';
    let suffix = '';
    let blockPrefix = ''; // For formats applied to the start of the line

    switch (formatType) {
      case 'bold':
        prefix = '**';
        suffix = '**';
        break;
      case 'italic':
        prefix = '*';
        suffix = '*';
        break;
      case 'code':
         if (selectedText.includes('\n')) { // Block code for multi-line selection
             prefix = '```\n';
             suffix = '\n```';
         } else { // Inline code for single line or no selection
             prefix = '`';
             suffix = '`';
         }
        break;
      case 'h1': blockPrefix = '# '; break;
      case 'h2': blockPrefix = '## '; break;
      case 'h3': blockPrefix = '### '; break;
      case 'blockquote': blockPrefix = '> '; break;
    }

    // Use requestAnimationFrame to ensure DOM updates before manipulating selection
    requestAnimationFrame(() => {
        if (!markdownTextareaElement) return;

        if (blockPrefix) {
            // Apply block formats to the start of the current or selected line(s)
            const lineStart = writeContent.lastIndexOf('\n', start - 1) + 1;
            // For block quotes, we might need to apply to multiple lines if selected
            if (formatType === 'blockquote' && selectedText.includes('\n')) {
                 const lines = selectedText.split('\n');
                 const prefixedLines = lines.map(line => blockPrefix + line).join('\n');
                 writeContent = writeContent.substring(0, start) + prefixedLines + writeContent.substring(end);
                 markdownTextareaElement.selectionStart = start;
                 markdownTextareaElement.selectionEnd = start + prefixedLines.length;
            } else {
                 // Apply prefix at the beginning of the line for headings
                 writeContent = writeContent.substring(0, lineStart) + blockPrefix + writeContent.substring(lineStart);
                 // Adjust selection points after adding prefix
                 markdownTextareaElement.selectionStart = start + blockPrefix.length;
                 markdownTextareaElement.selectionEnd = end + blockPrefix.length;
            }
        } else {
            // Apply inline formats (bold, italic, code)
            const newText = writeContent.substring(0, start) + prefix + selectedText + suffix + writeContent.substring(end);
            writeContent = newText;

            if (selectedText) {
              // If text was selected, select the newly formatted text
              markdownTextareaElement.selectionStart = start + prefix.length;
              markdownTextareaElement.selectionEnd = end + prefix.length;
            } else {
              // If no text was selected, place cursor between prefix and suffix
              markdownTextareaElement.selectionStart = start + prefix.length;
              markdownTextareaElement.selectionEnd = start + prefix.length;
            }
        }
        markdownTextareaElement.focus();
    });
  }

</script>

<button class="back-btn" on:click={goBack}>← Back to Mode Choice</button>
<div class="write-view-container">
  <!-- Left Panel (Top: Chat, Bottom: Tools) -->
  <div class="write-left-panel">
    <div class="write-chat-panel">
      <h3>Contextual Chat</h3>
      <div class="chat-messages-area" bind:this={writeChatDisplayElement}>
        {#each writeChatMessages as msg, i (i)} <!-- Simple key for reactivity -->
          <div class="message {msg.sender}">
              <strong class="sender-label">{msg.sender === 'user' ? 'You' : 'AI'}:</strong>
              <span class="message-text">{msg.text}</span>
          </div>
        {/each}
        {#if isWriteChatLoading}<div class="message ai loading">AI Thinking...</div>{/if}
         {#if writeChatMessages.length === 0 && !isWriteChatLoading}
           <div class="empty-chat">Ask the AI for ideas, rewrites, or feedback on your draft.</div>
         {/if}
      </div>
      <form on:submit|preventDefault={handleSendWriteChat} class="write-chat-form">
        <input type="text" bind:value={writeChatInput} placeholder="Ask AI..." disabled={isWriteChatLoading || !chatModelId} />
        <button type="submit" disabled={isWriteChatLoading || !writeChatInput.trim() || !chatModelId}>Send</button>
      </form>
      {#if writeChatError}
        <p class="error-message">{writeChatError}</p>
      {/if}
       {#if !chatModelId}
        <p class="info-text">Chat disabled. Select a chat model in Settings.</p>
      {/if}
    </div>
    <div class="write-tools-panel">
      <h3>Tools</h3>
       <div class="formatting-buttons">
          <button on:click={() => applyMarkdownFormat('bold')} title="Bold">B</button>
          <button on:click={() => applyMarkdownFormat('italic')} title="Italic">I</button>
          <button on:click={() => applyMarkdownFormat('h1')} title="Heading 1">H1</button>
          <button on:click={() => applyMarkdownFormat('h2')} title="Heading 2">H2</button>
          <button on:click={() => applyMarkdownFormat('h3')} title="Heading 3">H3</button>
          <button on:click={() => applyMarkdownFormat('code')} title="Code">{'</>'}</button>
          <button on:click={() => applyMarkdownFormat('blockquote')} title="Blockquote">"</button>
          <!-- Add more buttons later: list, link, image -->
       </div>
       <div class="save-buttons">
          <button class="save-btn" on:click={() => openSaveModal(false)} disabled={isLoading}>Save</button>
          <button class="save-as-btn" on:click={() => openSaveModal(true)} disabled={isLoading}>Save As</button>
       </div>
    </div>
  </div>

  <!-- Right Panel (Editor + Preview) -->
  <div class="write-right-panel">
    <div class="editor-toolbar">
      <div class="view-mode-toggles">
        <button 
          class="view-mode-btn {editorMode === 'edit' ? 'active' : ''}"
          on:click={() => editorMode = 'edit'}
          title="Edit mode"
        >
          <span class="btn-icon">📝</span> Edit
        </button>
        <button 
          class="view-mode-btn {editorMode === 'split' ? 'active' : ''}"
          on:click={() => editorMode = 'split'}
          title="Split mode"
        >
          <span class="btn-icon">⚔️</span> Split
        </button>
        <button 
          class="view-mode-btn {editorMode === 'preview' ? 'active' : ''}"
          on:click={() => editorMode = 'preview'}
          title="Preview mode"
        >
          <span class="btn-icon">👁️</span> Preview
        </button>
      </div>
    </div>
    
    <div class="editor-container {editorMode}">
      <textarea
        class="markdown-input"
        bind:value={writeContent}
        bind:this={markdownTextareaElement}
        placeholder="Start writing your masterpiece (Markdown supported)..."
        style="display: {editorMode === 'preview' ? 'none' : 'block'}"
      ></textarea>
      <div 
        class="markdown-preview-container"
        style="display: {editorMode === 'edit' ? 'none' : 'block'}"
      >
        <div class="markdown-preview">{@html renderedWriteHtml}</div>
      </div>
    </div>
  </div>
</div>

<!-- Save Modal -->
{#if showWriteSaveModal}
  <div class="modal-backdrop">
    <div class="modal save-write-modal">
      <h3>Save Written Content</h3>
      <label for="write-filename">Filename (.md):</label>
      <input id="write-filename" type="text" bind:value={writeFilename} placeholder="e.g., chapter-one.md">
      {#if writeSaveError}
        <p class="error-message">{writeSaveError}</p>
      {/if}
      {#if writeSaveSuccess}
        <p class="success-message">{writeSaveSuccess}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={handleSaveWriteContent} disabled={isLoading || !writeFilename.trim()}>
            {#if isLoading}Saving...{:else}Save{/if}
        </button>
        <button on:click={() => { showWriteSaveModal = false; writeSaveSuccess = ''; writeSaveError = ''; }} disabled={isLoading}>Cancel</button>
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

  .write-view-container {
    display: flex;
    gap: 1rem;
    height: calc(100vh - 4rem); /* Adjust height as needed */
    padding: 1rem;
    padding-top: 4rem; /* Space for back button */
  }

  .write-left-panel {
    display: flex;
    flex-direction: column;
    width: 35%; /* Adjust width */
    min-width: 300px; /* Minimum width */
    gap: 1rem;
    flex-shrink: 0;
  }

  .write-chat-panel,
  .write-tools-panel {
    background: var(--bg-secondary);
    padding: 1rem;
    border-radius: 8px;
    display: flex;
    flex-direction: column;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }

  .write-chat-panel {
    flex-grow: 1; /* Allow chat to take more space */
    min-height: 0; /* Important for flex child scrolling */
  }
   .write-chat-panel h3, .write-tools-panel h3 {
       margin-top: 0;
       margin-bottom: 1rem;
       color: var(--accent-secondary);
       font-size: 1.1rem;
   }

  .chat-messages-area {
      flex: 1; /* Allow message area to grow and shrink */
      overflow-y: auto; /* Enable vertical scrolling */
      padding: 0.5rem; /* Reduced padding */
      margin-bottom: 1rem;
      background: rgba(0,0,0,0.15);
      border-radius: 4px;
      min-height: 100px; /* Ensure it doesn't collapse */
  }
   .empty-chat {
      display: flex;
      align-items: center;
      justify-content: center;
      height: 100%;
      color: var(--text-secondary);
      font-style: italic;
      font-size: 0.9rem;
      text-align: center;
      padding: 1rem;
  }

   .message {
    margin-bottom: 0.75rem;
    padding: 0.6rem 1rem;
    border-radius: 10px;
    max-width: 90%;
    display: flex;
    flex-direction: column;
    line-height: 1.4;
  }
  .message.user {
    background: var(--accent-primary);
    color: white;
    margin-left: auto;
    border-bottom-right-radius: 3px;
    align-items: flex-end;
  }
  .message.ai {
    background: rgba(255, 255, 255, 0.08);
    color: var(--text-primary);
    margin-right: auto;
    border-bottom-left-radius: 3px;
    align-items: flex-start;
  }
   .message.loading {
      font-style: italic;
      color: var(--text-secondary);
      background: transparent;
   }
   .sender-label {
      font-size: 0.75rem;
      color: rgba(255, 255, 255, 0.6);
      margin-bottom: 0.2rem;
  }
   .message.ai .sender-label { color: var(--accent-secondary); }
   .message-text { white-space: pre-wrap; word-wrap: break-word; }


  .write-chat-form {
    display: flex;
    gap: 0.5rem;
    margin-top: auto; /* Push form to bottom */
    flex-shrink: 0;
  }

  .write-chat-form input {
    flex-grow: 1;
    padding: 0.6rem;
    background: rgba(255, 255, 255, 0.08);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 4px;
    color: var(--text-primary);
  }
   .write-chat-form input:focus { outline: none; border-color: var(--accent-primary); }

  .write-chat-form button {
      padding: 0.6rem 1rem;
      background: var(--accent-primary);
      border-radius: 4px;
      font-weight: 500;
      border: none;
      color: white;
      cursor: pointer;
  }
   .write-chat-form button:hover:not(:disabled) { background: var(--accent-secondary); }
   .write-chat-form button:disabled { opacity: 0.5; cursor: not-allowed; }

  .write-tools-panel {
      flex-shrink: 0; /* Prevent shrinking */
  }
  
  .formatting-buttons {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
      margin-bottom: 1rem;
  }
  
  .formatting-buttons button {
      padding: 0.4rem 0.8rem;
      font-size: 0.9rem;
      background: rgba(255, 255, 255, 0.1);
      color: var(--text-secondary);
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 4px;
      min-width: 35px;
      text-align: center;
      cursor: pointer;
      transition: background 0.2s ease;
  }
  
  .formatting-buttons button:hover {
      background: rgba(255, 255, 255, 0.2);
      color: var(--text-primary);
  }
  
  .save-buttons {
      display: flex;
      gap: 0.5rem;
      margin-top: 0.5rem;
  }
  
  .save-btn, .save-as-btn {
      padding: 0.6rem 1rem;
      font-size: 0.9rem;
      border-radius: 4px;
      border: none;
      color: white;
      cursor: pointer;
      flex: 1;
      transition: background 0.2s ease;
      font-weight: 500;
  }
  
  .save-btn {
      background: var(--success-color);
  }
  
  .save-btn:hover:not(:disabled) {
      background: #00b894;
  }
  
  .save-as-btn {
      background: #0984e3; /* Blue color */
  }
  
  .save-as-btn:hover:not(:disabled) {
      background: #74b9ff; /* Lighter blue */
  }
  
  .save-btn:disabled, .save-as-btn:disabled {
      opacity: 0.5;
      cursor: not-allowed;
  }


  .write-right-panel {
    display: flex;
    flex-direction: column;
    flex-grow: 1; /* Take remaining space */
    min-width: 0; /* Important for flex child */
    background: var(--bg-secondary);
    border-radius: 8px;
    border: 1px solid rgba(255, 255, 255, 0.1);
    overflow: hidden; /* Ensure content doesn't overflow */
  }
  
  .editor-toolbar {
    display: flex;
    justify-content: space-between;
    padding: 0.5rem;
    background: rgba(0, 0, 0, 0.2);
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  }
  
  .view-mode-toggles {
    display: flex;
    gap: 0.5rem;
  }
  
  .view-mode-btn {
    padding: 0.4rem 0.8rem;
    background: rgba(255, 255, 255, 0.1);
    border: 1px solid rgba(255, 255, 255, 0.15);
    border-radius: 4px;
    color: var(--text-secondary);
    cursor: pointer;
    font-size: 0.9rem;
    display: flex;
    align-items: center;
    gap: 0.3rem;
    transition: all 0.2s ease;
  }
  
  .view-mode-btn:hover {
    background: rgba(255, 255, 255, 0.15);
    color: var(--text-primary);
  }
  
  .view-mode-btn.active {
    background: var(--accent-primary);
    color: white;
    border-color: var(--accent-primary);
  }
  
  .editor-container {
    display: flex;
    flex: 1;
    overflow: hidden;
  }
  
  .editor-container.split {
    flex-direction: row;
  }
  
  .editor-container.edit, .editor-container.preview {
    flex-direction: column;
  }

  .markdown-input {
    flex: 1;
    min-height: 150px;
    resize: none;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: none;
    padding: 1rem;
    line-height: 1.6;
    font-size: 0.95rem;
    overflow-y: auto;
  }
  
  .editor-container.split .markdown-input {
    flex: 1;
    border-right: 1px solid rgba(255, 255, 255, 0.1);
  }
  
  .markdown-input:focus {
    outline: none;
  }

  .markdown-preview-container {
    flex: 1;
    background: var(--bg-secondary);
    padding: 1rem;
    overflow-y: auto;
    display: flex;
    flex-direction: column;
  }
  
  .markdown-preview {
    flex-grow: 1;
    line-height: 1.6;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
    color: var(--text-primary);
    padding-bottom: 2rem; /* Add some bottom padding for scrolling */
  }
  
  /* Enhanced Markdown Preview Styles */
  .markdown-preview :global(h1),
  .markdown-preview :global(h2),
  .markdown-preview :global(h3),
  .markdown-preview :global(h4),
  .markdown-preview :global(h5),
  .markdown-preview :global(h6) {
    margin-top: 1.5em;
    margin-bottom: 0.5em;
    color: var(--accent-secondary);
    font-weight: 600;
    line-height: 1.3;
  }
  
  .markdown-preview :global(h1) {
    font-size: 1.8em;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    padding-bottom: 0.3em;
  }
  
  .markdown-preview :global(h2) {
    font-size: 1.5em;
    border-bottom: 1px solid rgba(255, 255, 255, 0.1);
    padding-bottom: 0.2em;
  }
  
  .markdown-preview :global(h3) { font-size: 1.3em; }
  .markdown-preview :global(h4) { font-size: 1.1em; }
  
  .markdown-preview :global(p) {
    margin-bottom: 1em;
    line-height: 1.6;
  }
  
  .markdown-preview :global(ul),
  .markdown-preview :global(ol) {
    margin-left: 1.5em;
    margin-bottom: 1em;
    padding-left: 0.5em;
  }
  
  .markdown-preview :global(li) {
    margin-bottom: 0.5em;
  }
  
  .markdown-preview :global(li > ul),
  .markdown-preview :global(li > ol) {
    margin-top: 0.3em;
    margin-bottom: 0.3em;
  }
  
  .markdown-preview :global(code) {
    background: rgba(255, 255, 255, 0.1);
    padding: 0.2em 0.4em;
    border-radius: 3px;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 0.9em;
    color: #e6e6e6;
  }
  
  .markdown-preview :global(pre) {
    background: rgba(0, 0, 0, 0.2);
    padding: 1em;
    border-radius: 4px;
    overflow-x: auto;
    margin-bottom: 1em;
    border: 1px solid rgba(255, 255, 255, 0.1);
  }
  
  .markdown-preview :global(pre code) {
    background: none;
    padding: 0;
    font-size: 0.9em;
    color: #e6e6e6;
    white-space: pre;
  }
  
  .markdown-preview :global(blockquote) {
    border-left: 3px solid var(--accent-primary);
    margin: 1em 0;
    padding: 0.5em 1em;
    background: rgba(255, 255, 255, 0.05);
    color: var(--text-secondary);
    border-radius: 0 4px 4px 0;
  }
  
  .markdown-preview :global(blockquote p:last-child) {
    margin-bottom: 0;
  }
  
  .markdown-preview :global(a) {
    color: var(--accent-secondary);
    text-decoration: none;
    border-bottom: 1px dotted var(--accent-secondary);
    transition: all 0.2s ease;
  }
  
  .markdown-preview :global(a:hover) {
    border-bottom: 1px solid var(--accent-secondary);
  }
  
  .markdown-preview :global(strong) {
    font-weight: 600;
    color: #f0f0f0;
  }
  
  .markdown-preview :global(em) {
    font-style: italic;
  }
  
  .markdown-preview :global(img) {
    max-width: 100%;
    border-radius: 4px;
    margin: 1em 0;
  }
  
  .markdown-preview :global(hr) {
    border: none;
    height: 1px;
    background: rgba(255, 255, 255, 0.1);
    margin: 2em 0;
  }
  
  .markdown-preview :global(table) {
    border-collapse: collapse;
    width: 100%;
    margin: 1em 0;
    overflow-x: auto;
    display: block;
  }
  
  .markdown-preview :global(th),
  .markdown-preview :global(td) {
    border: 1px solid rgba(255, 255, 255, 0.1);
    padding: 0.5em 1em;
    text-align: left;
  }
  
  .markdown-preview :global(th) {
    background: rgba(255, 255, 255, 0.05);
    font-weight: 600;
  }
  
  .markdown-preview :global(tr:nth-child(even)) {
    background: rgba(255, 255, 255, 0.02);
  }


  /* Modals */
  .modal-backdrop { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.8); backdrop-filter: blur(4px); display: flex; align-items: center; justify-content: center; z-index: 1000; padding: 1rem; }
  .modal { background: var(--bg-primary); color: var(--text-primary); border-radius: 12px; padding: 1.5rem 2rem; width: 100%; max-width: 500px; margin: auto; box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4); border: 1px solid rgba(255, 255, 255, 0.1); }
  .modal h3 { margin-top: 0; margin-bottom: 1.5rem; color: var(--accent-primary); }
  .modal label { display: block; margin-bottom: 0.5rem; color: var(--text-secondary); }
  .modal input[type="text"] { width: 100%; padding: 0.75rem; background: rgba(255, 255, 255, 0.08); border: 1px solid rgba(255, 255, 255, 0.15); border-radius: 6px; color: var(--text-primary); font-size: 1rem; margin-bottom: 1rem; }
  .modal-buttons { display: flex; justify-content: flex-end; gap: 1rem; margin-top: 1.5rem; }
  .modal-buttons button { padding: 0.6rem 1.2rem; }

  .error-message, .success-message, .info-text { padding: 0.75rem 1rem; border-radius: 8px; margin-top: 1rem; font-size: 0.9rem; }
  .error-message { color: var(--error-color); background: rgba(255, 71, 87, 0.1); border: 1px solid rgba(255, 71, 87, 0.2); }
  .success-message { color: var(--success-color); background: rgba(46, 213, 115, 0.1); border: 1px solid rgba(46, 213, 115, 0.2); }
  .info-text { color: var(--text-secondary); background: rgba(255, 255, 255, 0.05); border: 1px solid rgba(255, 255, 255, 0.1); }

  /* Scrollbar */
  ::-webkit-scrollbar { width: 6px; }
  ::-webkit-scrollbar-track { background: rgba(255, 255, 255, 0.05); border-radius: 3px; }
  ::-webkit-scrollbar-thumb { background: var(--accent-primary); border-radius: 3px; }
  ::-webkit-scrollbar-thumb:hover { background: var(--accent-secondary); }

</style>
