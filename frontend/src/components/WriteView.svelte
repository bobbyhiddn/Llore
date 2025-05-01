<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate } from 'svelte';
  import { Marked } from 'marked'; // Import Marked class
  import { SaveLibraryFile, GenerateOpenRouterContent } from '@wailsjs/go/main/App';

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
  $: renderMarkdown(writeContent);

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

  async function renderMarkdown(markdown: string) {
      try {
          // Use async parsing
          renderedWriteHtml = await marked.parse(markdown || '');
      } catch (err) {
          console.error("Markdown rendering error:", err);
          // Fallback to plain text on error
          renderedWriteHtml = markdown.replace(/</g, "<").replace(/>/g, ">");
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

    try {
      const modelToUse = chatModelId; // Use the model from props
      console.log(`Write Chat using model: ${modelToUse}`);

      const aiReply = await GenerateOpenRouterContent(prompt, modelToUse);
      writeChatMessages = [...writeChatMessages, { sender: 'ai', text: aiReply }];

    } catch (err) {
      writeChatError = `AI error: ${err}`;
      console.error("Write Chat Error:", err);
      // Optionally add the error as a system message to the chat?
      // writeChatMessages = [...writeChatMessages, { sender: 'ai', text: `Sorry, I encountered an error: ${err}` }];
    } finally {
      isWriteChatLoading = false;
    }
  }

  // --- Write Mode Save Function ---
  function openSaveModal() {
      writeSaveError = '';
      writeSaveSuccess = '';
      // Suggest filename if empty and content exists
      if (!writeFilename && writeContent.trim()) {
          const firstLine = writeContent.trim().split('\n')[0];
          // Basic filename suggestion from first line
          writeFilename = firstLine.substring(0, 30).replace(/[^a-z0-9\s]/gi, '').replace(/\s+/g, '-') + '.md';
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
      <div class="button-group format-tools">
         <button on:click={() => applyMarkdownFormat('bold')} title="Bold"><b>B</b></button>
         <button on:click={() => applyMarkdownFormat('italic')} title="Italic"><i>I</i></button>
         <button on:click={() => applyMarkdownFormat('h1')} title="Heading 1">H1</button>
         <button on:click={() => applyMarkdownFormat('h2')} title="Heading 2">H2</button>
         <button on:click={() => applyMarkdownFormat('h3')} title="Heading 3">H3</button>
         <button on:click={() => applyMarkdownFormat('code')} title="Code">{'</>'}</button>
         <button on:click={() => applyMarkdownFormat('blockquote')} title="Blockquote">"</button>
         <!-- Add more buttons later: list, link, image -->
      </div>
       <button class="save-btn" on:click={openSaveModal} disabled={isLoading}>Save to Library</button>
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
        <h3 class="preview-header">Preview</h3>
        <div class="markdown-preview">{@html renderedWriteHtml}</div>
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
  .button-group.format-tools {
      display: flex;
      flex-wrap: wrap;
      gap: 0.5rem;
      margin-bottom: 1rem;
  }
  .format-tools button {
      padding: 0.4rem 0.8rem;
      font-size: 0.9rem;
      background: rgba(255, 255, 255, 0.1);
      color: var(--text-secondary);
      border: 1px solid rgba(255, 255, 255, 0.2);
      border-radius: 4px;
      min-width: 35px;
      text-align: center;
  }
   .format-tools button:hover {
       background: rgba(255, 255, 255, 0.2);
       color: var(--text-primary);
   }
   .format-tools b, .format-tools i { font-size: 1rem; }

  .save-btn {
      width: 100%;
      padding: 0.7rem;
      background: var(--success-color);
      color: white;
      border: none;
      border-radius: 4px;
      font-weight: 500;
      cursor: pointer;
      margin-top: auto; /* Push to bottom */
  }
   .save-btn:hover:not(:disabled) { background: #00b894; }
   .save-btn:disabled { opacity: 0.5; cursor: not-allowed; }


  .write-right-panel {
    display: flex;
    flex-direction: column; /* Stack editor and preview */
    flex-grow: 1; /* Take remaining space */
    gap: 1rem;
    min-width: 0; /* Important for flex child */
  }

  .markdown-input {
    flex: 1; /* Take half the space */
    min-height: 150px; /* Minimum height */
    resize: none; /* Disable manual resize */
    font-family: monospace;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid rgba(255, 255, 255, 0.1);
    padding: 1rem;
    border-radius: 4px;
    line-height: 1.5;
    font-size: 0.95rem;
  }
   .markdown-input:focus {
       outline: none;
       border-color: var(--accent-primary);
       background: rgba(0,0,0,0.1);
   }

  .markdown-preview-container {
    flex: 1; /* Take half the space */
    background: var(--bg-secondary);
    padding: 0 1rem 1rem 1rem; /* Padding bottom */
    border-radius: 8px;
    overflow-y: auto;
    border: 1px solid rgba(255, 255, 255, 0.1);
    display: flex;
    flex-direction: column;
    min-height: 150px;
  }
  .preview-header {
      margin: 1rem 0 0.5rem 0;
      color: var(--text-secondary);
      font-size: 1rem;
      font-weight: 500;
      border-bottom: 1px solid rgba(255, 255, 255, 0.1);
      padding-bottom: 0.5rem;
      flex-shrink: 0;
  }
  .markdown-preview {
      flex-grow: 1;
      overflow-y: auto; /* Scroll content within preview */
      line-height: 1.6;
      padding-right: 0.5rem; /* Space for scrollbar */
  }
  /* Basic Markdown Preview Styles */
  .markdown-preview :global(h1),
  .markdown-preview :global(h2),
  .markdown-preview :global(h3) { margin-top: 1.5em; margin-bottom: 0.5em; color: var(--accent-secondary); border-bottom: 1px solid rgba(255,255,255,0.1); padding-bottom: 0.2em;}
  .markdown-preview :global(h1) { font-size: 1.8em; }
  .markdown-preview :global(h2) { font-size: 1.5em; }
  .markdown-preview :global(h3) { font-size: 1.2em; }
  .markdown-preview :global(p) { margin-bottom: 1em; }
  .markdown-preview :global(ul),
  .markdown-preview :global(ol) { margin-left: 1.5em; margin-bottom: 1em; }
  .markdown-preview :global(li) { margin-bottom: 0.3em; }
  .markdown-preview :global(code) { background: rgba(255,255,255,0.1); padding: 0.2em 0.4em; border-radius: 3px; font-family: monospace; font-size: 0.9em;}
  .markdown-preview :global(pre) { background: rgba(0,0,0,0.2); padding: 1em; border-radius: 4px; overflow-x: auto; }
  .markdown-preview :global(pre code) { background: none; padding: 0; }
  .markdown-preview :global(blockquote) { border-left: 3px solid var(--accent-primary); margin-left: 0; padding-left: 1em; color: var(--text-secondary); font-style: italic; }
  .markdown-preview :global(a) { color: var(--accent-secondary); }
  .markdown-preview :global(a:hover) { text-decoration: underline; }
  .markdown-preview :global(strong) { font-weight: bold; }
  .markdown-preview :global(em) { font-style: italic; }


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
