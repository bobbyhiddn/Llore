<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate } from 'svelte';
  import { Marked } from 'marked'; // Import Marked class
  import { SaveLibraryFile, GetAIResponseWithContext } from '@wailsjs/go/main/App';

  // Props
  export let initialContent: string = ''; // If loading existing content
  export let initialFilename: string = ''; // If loading existing content
  export let chatModelId: string = ''; // From global settings
  // export let isLoading: boolean = false; // Global loading state from parent - Handled by internal isSaving flag now
  
  // Local State for Editor and Content
  let writeContent: string = ''; // Initialized in onMount
  let renderedWriteHtml = '';
  let markdownTextareaElement: HTMLTextAreaElement;
  let editorMode: 'split' | 'edit' | 'preview' = 'split';
  
  // State for Dirty Checking, Word/Char Count
  let isDirty: boolean = false;
  let wordCount: number = 0;
  let charCount: number = 0;
  let baselineContentForDirtyCheck: string = ''; // Stores content as of last load/save

  // Write Mode Chat State
  let writeChatDisplayElement: HTMLDivElement; // For auto-scrolling
  let writeChatMessages: { sender: 'user' | 'ai', text: string }[] = [];
  let writeChatInput: string = '';
  let isWriteChatLoading: boolean = false;
  let writeChatError: string = '';

  // Save State
  let showWriteSaveModal: boolean = false;
  let filenameForSaveModal: string = ''; // Filename input in the modal
  let currentDocumentFilename: string = ''; // The actual filename of the current document (after load/save)
  let isSaving: boolean = false;
  let writeSaveError: string = '';
  let writeSaveSuccess: string = '';
  let isSaveAsOperation: boolean = false;


  const dispatch = createEventDispatcher();
  const marked = new Marked({ gfm: true, breaks: true }); // Enable GFM and line breaks

  // --- Debounced Markdown Rendering ---
  let renderTimeout: number;
  function scheduleRender(text: string) {
      clearTimeout(renderTimeout);
      renderTimeout = window.setTimeout(() => {
          renderMarkdown(text);
      }, 200); // 200ms debounce
  }

  function renderMarkdown(text: string) {
    try {
      const result = marked.parse(text || '');
      renderedWriteHtml = typeof result === 'string' ? result : String(result);
    } catch (err) {
      console.error("Markdown rendering error:", err);
      renderedWriteHtml = `<p style="color:red;">Error rendering Markdown. Check console.</p><pre>${text.replace(/</g, "&lt;").replace(/>/g, "&gt;")}</pre>`;
    }
  }

  // --- Lifecycle ---
  onMount(() => {
      writeContent = initialContent;
      baselineContentForDirtyCheck = initialContent;
      currentDocumentFilename = initialFilename;
      filenameForSaveModal = initialFilename || 'untitled.md'; // Default for modal
      renderMarkdown(writeContent); // Initial render
      updateCounts(writeContent);
      
      // Focus editor on mount if in edit or split mode
      if (editorMode === 'edit' || editorMode === 'split') {
          markdownTextareaElement?.focus();
      }
  });

  // Reactive Updates for Content Changes
  $: {
    if (writeContent !== undefined) { // Check ensures it runs after onMount initialization
      scheduleRender(writeContent);
      isDirty = writeContent !== baselineContentForDirtyCheck;
      updateCounts(writeContent);
    }
  }

  function updateCounts(text: string) {
      charCount = text.length;
      wordCount = text.trim() ? text.trim().split(/\s+/).length : 0;
  }
  
  // Auto-scroll chat display
  afterUpdate(() => {
    if (writeChatDisplayElement) {
       writeChatDisplayElement.scrollTop = writeChatDisplayElement.scrollHeight;
    }
  });

  // --- Helper Functions ---
  function getSelectedText(): string {
      if (!markdownTextareaElement) return '';
      return markdownTextareaElement.value.substring(markdownTextareaElement.selectionStart, markdownTextareaElement.selectionEnd);
  }

  function getTextBeforeCursor(): string {
      if (!markdownTextareaElement) return '';
      return markdownTextareaElement.value.substring(0, markdownTextareaElement.selectionStart);
  }

  function insertTextIntoDraft(textToInsert: string) {
    if (!markdownTextareaElement) return;
    const start = markdownTextareaElement.selectionStart;
    const end = markdownTextareaElement.selectionEnd;
    const currentText = writeContent;

    writeContent = currentText.substring(0, start) + textToInsert + currentText.substring(end);
    
    // Use requestAnimationFrame to ensure DOM updates (textarea value) before setting selection
    requestAnimationFrame(() => {
        if (!markdownTextareaElement) return;
        markdownTextareaElement.focus();
        markdownTextareaElement.selectionStart = start + textToInsert.length;
        markdownTextareaElement.selectionEnd = start + textToInsert.length;
    });
  }


  // --- General Functions ---
  function goBack() {
    // TODO: Add confirmation if isDirty
    if (isDirty && !confirm("You have unsaved changes. Are you sure you want to leave?")) {
        return;
    }
    dispatch('back');
  }

  // --- Write Mode Chat Function ---
  async function handleSendWriteChat(overridePrompt?: string, userMessageOverride?: string) {
    const userMessageToSend = userMessageOverride || writeChatInput.trim();
    if (!userMessageToSend && !overridePrompt) return;
    
    if (!chatModelId) {
        writeChatError = 'No chat model selected in settings.';
        dispatch('error', writeChatError); // Dispatch global error
        return;
    }

    writeChatMessages = [...writeChatMessages, { sender: 'user', text: userMessageToSend }];
    if (!overridePrompt) writeChatInput = ''; // Clear input only if not an override
    isWriteChatLoading = true;
    writeChatError = '';
    dispatch('loading', true);


    let finalPrompt = overridePrompt;

    if (!finalPrompt) {
        // Handle Slash Commands
        if (userMessageToSend.startsWith('/summarize_selection')) {
            const selection = getSelectedText();
            if (!selection) { 
                writeChatError = "Please select text to summarize."; 
                isWriteChatLoading = false; 
                dispatch('loading', false);
                writeChatMessages = writeChatMessages.slice(0, -1); // Remove the user message
                return; 
            }
            finalPrompt = `System: Summarize the following selected text from the user's draft concisely.\n<selected_text>\n${selection}\n</selected_text>\nUser: Summarize the selected text.`;
        } else if (userMessageToSend.startsWith('/rephrase_selection')) {
            const selection = getSelectedText();
            if (!selection) { 
                writeChatError = "Please select text to rephrase."; 
                isWriteChatLoading = false; 
                dispatch('loading', false);
                writeChatMessages = writeChatMessages.slice(0, -1); 
                return; 
            }
            finalPrompt = `System: Rephrase the following selected text from the user's draft. Aim for clarity and improved style.\n<selected_text>\n${selection}\n</selected_text>\nUser: Rephrase the selected text.`;
        } else if (userMessageToSend.startsWith('/continue_writing')) {
            const textBefore = getTextBeforeCursor();
            finalPrompt = `System: Continue writing from the current cursor position in the user's draft. Maintain the existing tone and style.\n<draft_context_before_cursor>\n${textBefore.slice(-1000)}\n</draft_context_before_cursor>\nUser: Continue writing.`;
        }
    }

    if (!finalPrompt) { // Regular chat message, build prompt with context
        // Build prompt with draft context
        const draftContext = writeContent.length > 2000 ? writeContent.substring(0, 2000) + "\n...[Draft Truncated]..." : writeContent;
        finalPrompt = `System: You are an AI writing assistant. The user is working on the following draft:\n<draft>\n${draftContext}\n</draft>\n\n`;

        finalPrompt += `Recent Chat History (user and AI):\n`;
        const historyLimit = 3; // User-AI pairs
        writeChatMessages.slice(-(historyLimit * 2 + 1), -1).forEach(msg => { // Get history before current user message
            finalPrompt += `${msg.sender === 'user' ? 'User' : 'AI'}: ${msg.text}\n`;
        });
        finalPrompt += `User: ${userMessageToSend}\nAI:`;
    }

    try {
      const aiReply = await GetAIResponseWithContext(finalPrompt, chatModelId);
      writeChatMessages = [...writeChatMessages, { sender: 'ai', text: aiReply }];
    } catch (err) {
      console.error("Error in write chat:", err);
      writeChatError = `AI error: ${err}`;
      dispatch('error', writeChatError);
    } finally {
      isWriteChatLoading = false;
      dispatch('loading', false);
    }
  }

  // --- Tool Button Handlers ---
  function handleToolAction(action: 'summarize' | 'rephrase' | 'continue') {
      let prompt = '';
      let userMessageForChat = '';
      
      if (action === 'summarize' || action === 'rephrase') {
          const selection = getSelectedText();
          if (!selection) {
              alert(`Please select text to ${action}.`);
              return;
          }
          userMessageForChat = `User asked to ${action} selection.`;
          prompt = `System: ${action === 'summarize' ? 'Summarize' : 'Rephrase'} the following selected text from the user's draft.\n<selected_text>\n${selection}\n</selected_text>\nUser: ${action} the selected text.`;
      } else if (action === 'continue') {
          const textBefore = getTextBeforeCursor();
          userMessageForChat = `User asked to continue writing.`;
          prompt = `System: Continue writing from the current cursor position in the user's draft.\n<draft_context_before_cursor>\n${textBefore.slice(-1000)}\n</draft_context_before_cursor>\nUser: Continue writing.`;
      }
      handleSendWriteChat(prompt, userMessageForChat);
  }

  // --- Write Mode Save Function ---
  function openSaveModal(isSaveAs: boolean) {
      writeSaveError = '';
      writeSaveSuccess = '';
      isSaveAsOperation = isSaveAs;

      if (isSaveAs || !currentDocumentFilename) {
          // For "Save As" or if no current filename, suggest based on content or use "untitled"
          if (currentDocumentFilename && isSaveAs) {
              const baseName = currentDocumentFilename.replace(/\.[^/.]+$/, '');
              filenameForSaveModal = `${baseName}_copy.md`;
          } else if (writeContent.trim()) {
              const firstLine = writeContent.trim().split('\n')[0];
              filenameForSaveModal = firstLine.substring(0, 30).replace(/[^a-z0-9\s._-]/gi, '').replace(/\s+/g, '-') + '.md';
          } else {
              filenameForSaveModal = 'untitled.md';
          }
      } else {
          // For regular "Save", use the current document's filename
          filenameForSaveModal = currentDocumentFilename;
      }
      showWriteSaveModal = true;
  }

  // Direct Save (used by "Save" button in tools if filename exists)
  async function handleDirectSave() {
      if (!currentDocumentFilename) {
          openSaveModal(false); // If no filename, open modal as if it's the first save
          return;
      }
      filenameForSaveModal = currentDocumentFilename; // Ensure modal uses current filename
      await doSave();
  }

  // Actual save logic, called by modal or direct save
  async function doSave() {
    if (!filenameForSaveModal.trim()) {
      writeSaveError = 'Filename cannot be empty.';
      return;
    }
    let finalFilenameToSave = filenameForSaveModal.trim();
    if (!finalFilenameToSave.toLowerCase().endsWith('.md')) {
      finalFilenameToSave += '.md';
    }

    writeSaveError = '';
    writeSaveSuccess = '';
    isSaving = true;
    dispatch('loading', true);

    try {
      await SaveLibraryFile(finalFilenameToSave, writeContent);
      
      currentDocumentFilename = finalFilenameToSave; // Update the current document's filename
      baselineContentForDirtyCheck = writeContent;   // Update baseline content
      isDirty = false;                               // Reset dirty state

      writeSaveSuccess = `File '${finalFilenameToSave}' saved successfully!`;
      if (!isSaveAsOperation && showWriteSaveModal) { // Only close modal if it wasn't "Save As" that just completed
          // Or always close? User might want to keep it open for "Save As" to see success.
          // Let's close it for now.
          setTimeout(() => { showWriteSaveModal = false; }, 1500); // Close after a delay
      } else if (isSaveAsOperation) {
          showWriteSaveModal = false; // Close immediately for "Save As"
      }

      dispatch('filesaved', finalFilenameToSave);
    } catch (err) {
      writeSaveError = `Failed to save file: ${err}`;
      console.error("Save Write Content Error:", err);
      dispatch('error', writeSaveError);
    } finally {
      isSaving = false;
      dispatch('loading', false);
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

<div class="write-view-main-content">
  <div class="left-column">
    <div class="write-chat-panel">
      <h3>Contextual Chat</h3>
      <div class="chat-messages-area" bind:this={writeChatDisplayElement}>
        {#each writeChatMessages as msg, i (i)} <!-- Simple key for reactivity -->
          <div class="message {msg.sender}">
              <strong class="sender-label">{msg.sender === 'user' ? 'You' : 'AI'}:</strong>
              <span class="message-text">{msg.text}</span>
              {#if msg.sender === 'ai'}
                <button class="insert-btn" on:click={() => insertTextIntoDraft(msg.text)} title="Insert AI response into draft">↵ Insert</button>
              {/if}
            </div>
        {/each}
        {#if isWriteChatLoading}<div class="message ai loading">AI Thinking...</div>{/if}
         {#if writeChatMessages.length === 0 && !isWriteChatLoading}
           <div class="empty-chat">Ask the AI for ideas, rewrites, or feedback on your draft.</div>
         {/if}
      </div>
      <form on:submit|preventDefault={() => handleSendWriteChat()} class="write-chat-form">
        <input type="text" bind:value={writeChatInput} placeholder="Ask AI..." disabled={isWriteChatLoading || !chatModelId} style="flex-grow: 1; padding: 0.6rem; background: rgba(255, 255, 255, 0.08); border: 1px solid rgba(255, 255, 255, 0.15); border-radius: 4px; color: var(--text-primary);" />
        <button type="submit" disabled={isWriteChatLoading || !writeChatInput.trim() || !chatModelId} style="padding: 0.6rem 1rem; background: var(--accent-primary); border-radius: 4px; font-weight: 500; border: none; color: white; cursor: pointer;">Send</button>
      </form>
      {#if writeChatError}
        <p class="error-message">{writeChatError}</p>
      {/if}
       {#if !chatModelId}
        <p class="info-text">Chat disabled. Select a chat model in Settings.</p>
      {/if}
    </div>
    <div class="save-tools-module">
      <div class="tool-section">
        <h4>File</h4>
        <div class="save-buttons">
          <button class="save-btn" on:click={handleDirectSave} disabled={isSaving || !isDirty}>
            Save {#if isDirty && currentDocumentFilename}*{/if}
          </button>
          <button class="save-as-btn" on:click={() => openSaveModal(true)} disabled={isSaving}>Save As...</button>
        </div>
        <div class="doc-info">
          <span>Chars: {charCount}</span>
          <span>Words: {wordCount}</span>
        </div>
      </div>
    </div>
  </div>

  <div class="center-column">
    <div class="editor-toolbar">
      <div class="view-mode-toggles">
        <button class:active={editorMode === 'edit'} on:click={() => editorMode = 'edit'} title="Edit mode">📝 Edit</button>
        <button class:active={editorMode === 'split'} on:click={() => editorMode = 'split'} title="Split mode">⚔️ Split</button>
        <button class:active={editorMode === 'preview'} on:click={() => editorMode = 'preview'} title="Preview mode">👁️ Preview</button>
      </div>
      <div class="current-file-display">
        {currentDocumentFilename || "New Document"}{#if isDirty && currentDocumentFilename}*{/if}
      </div>
    </div>
    <div class="editor-container">
      <textarea
        class="markdown-input"
        bind:value={writeContent}
        bind:this={markdownTextareaElement}
        placeholder="Start writing your masterpiece (Markdown supported)..."
        style="display: {editorMode === 'preview' ? 'none' : 'block'}"
        on:keydown={(e) => {
            // Basic Ctrl+B and Ctrl+I for bold/italic
            if (e.ctrlKey || e.metaKey) {
                if (e.key === 'b') { e.preventDefault(); applyMarkdownFormat('bold'); }
                if (e.key === 'i') { e.preventDefault(); applyMarkdownFormat('italic'); }
                if (e.key === 's') { e.preventDefault(); handleDirectSave(); }
            }
            // Handle Tab for indentation (basic version)
            if (e.key === 'Tab') {
                e.preventDefault();
                const start = markdownTextareaElement.selectionStart;
                const end = markdownTextareaElement.selectionEnd;
                const text = markdownTextareaElement.value;
                // Insert tab character
                writeContent = text.substring(0, start) + '\t' + text.substring(end);
                // Move cursor after tab
                requestAnimationFrame(() => {
                    markdownTextareaElement.selectionStart = markdownTextareaElement.selectionEnd = start + 1;
                });
            }
        }}
      ></textarea>
      <div 
        class="markdown-preview-container"
        style="display: {editorMode === 'edit' ? 'none' : 'block'}"
      >
        <div class="markdown-preview">{@html renderedWriteHtml}</div>
      </div>
    </div>
  </div>

  <div class="right-column-toolbar">
    <div class="tool-section">
      <h4>Formatting</h4>
      <div class="formatting-buttons">
        <button on:click={() => applyMarkdownFormat('bold')} title="Bold (Ctrl+B)"><b>B</b></button>
        <button on:click={() => applyMarkdownFormat('italic')} title="Italic (Ctrl+I)"><i>I</i></button>
        <button on:click={() => applyMarkdownFormat('h1')} title="Heading 1">H1</button>
        <button on:click={() => applyMarkdownFormat('h2')} title="Heading 2">H2</button>
        <button on:click={() => applyMarkdownFormat('h3')} title="Heading 3">H3</button>
        <button on:click={() => applyMarkdownFormat('code')} title="Code (`code`)">{"</>"}}</button>
        <button on:click={() => applyMarkdownFormat('blockquote')} title="Blockquote (> text)">" "</button>
      </div>
    </div>
    <div class="tool-section">
      <h4>AI Actions (Selection-based)</h4>
      <div class="ai-action-buttons">
        <button on:click={() => handleToolAction('summarize')} title="Summarize selected text via chat">Summarize</button>
        <button on:click={() => handleToolAction('rephrase')} title="Rephrase selected text via chat">Rephrase</button>
        <button on:click={() => handleToolAction('continue')} title="Ask AI to continue writing from cursor via chat">Continue</button>
      </div>
    </div>
  </div>
</div>

<!-- Save Modal -->
{#if showWriteSaveModal}
  <div class="modal-backdrop">
    <div class="modal save-write-modal">
      <h3>{isSaveAsOperation || !currentDocumentFilename ? 'Save As' : 'Save File'}</h3>
      <label for="write-filename">Filename: {isDirty ? '*' : ''}</label>
      <input id="write-filename" type="text" bind:value={filenameForSaveModal} placeholder="e.g., chapter-one.md">
      {#if writeSaveError}
        <p class="error-message small">{writeSaveError}</p>
      {/if}
      {#if writeSaveSuccess}
        <p class="success-message small">{writeSaveSuccess}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={doSave} disabled={isSaving || !filenameForSaveModal.trim()}>
            {#if isSaving}Saving...{:else}Save{/if}
        </button>
        <button on:click={() => { showWriteSaveModal = false; writeSaveSuccess = ''; writeSaveError = ''; }} disabled={isSaving}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<style>
  :global(body) {
    font-family: var(--font-family, -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Oxygen, Ubuntu, Cantarell, "Open Sans", "Helvetica Neue", sans-serif);
    color: var(--text-primary, #e0e0e0); /* Light text for dark theme */
    background-color: var(--bg-primary, #1e1e1e); /* Dark background for body */
  }

  .back-btn {
    display: block;
    margin: 1rem 1rem 1.5rem 1rem;
    padding: 0.7rem 1.2rem;
    background: var(--bg-secondary, rgba(22, 33, 62, 0.95)); /* App.svelte var */
    color: var(--text-accent, var(--accent-primary, #6d5ed9)); /* App.svelte var */
    border: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
    border-radius: 6px;
    cursor: pointer;
    text-decoration: none;
    font-size: 0.9rem;
    font-weight: 500;
    transition: background-color 0.2s ease, color 0.2s ease, box-shadow 0.2s ease;
  }
  .back-btn:hover {
    background: var(--bg-hover-medium, rgba(255, 255, 255, 0.1));
    color: var(--text-accent-hover, var(--accent-secondary, #8a7ef9)); /* App.svelte var */
    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  }

  .write-view-main-content {
    display: flex;
    height: calc(100vh - 60px); 
    width: 100%;
    box-sizing: border-box;
    background-color: var(--bg-primary, rgba(26, 26, 46, 0.95)); /* App.svelte var */
    padding-bottom: 1.5rem; /* Added padding to the bottom */
  }

  .left-column, .center-column, .right-column-toolbar {
    padding: 1.25rem;
    box-sizing: border-box;
    overflow-y: auto;
  }

  .left-column {
    flex: 0 0 300px;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-secondary, rgba(22, 33, 62, 0.90)); /* App.svelte var */
    border-right: 1px solid var(--border-color-strong, rgba(160, 160, 160, 0.3));
    gap: 1.5rem;
  }

  .left-column .write-chat-panel {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    background: transparent;
    border: none;
    padding: 0;
  }

  .left-column .write-chat-panel h3 {
    font-size: 1.1em;
    color: var(--text-title, var(--accent-primary, #6d5ed9)); /* App.svelte var */
    margin-bottom: 1rem;
    padding-bottom: 0.5rem;
    border-bottom: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
  }

  .left-column .write-chat-panel .chat-messages-area {
    flex-grow: 1;
    overflow-y: auto;
    margin-bottom: 1rem;
    padding: 0.75rem;
    background: var(--bg-primary, rgba(26, 26, 46, 0.85)); /* App.svelte var */
    border-radius: 6px;
    border: 1px solid var(--border-color-light, rgba(160, 160, 160, 0.1));
  }

  .message {
    margin-bottom: 0.8rem;
    padding: 0.6rem 1rem;
    border-radius: 8px;
    line-height: 1.5;
    word-wrap: break-word;
    position: relative;
  }
  .message.user {
    background-color: var(--user-message-bg, var(--accent-primary, #6d5ed9)); /* Use accent for user */
    color: var(--text-primary, #e0e0e0);
    margin-left: auto;
    max-width: 85%;
    border-bottom-right-radius: 2px;
  }
  .message.ai {
    background-color: var(--ai-message-bg, var(--bg-secondary, rgba(22, 33, 62, 0.9))); /* App.svelte var */
    color: var(--text-primary, #e0e0e0);
    margin-right: auto;
    max-width: 85%;
    border-bottom-left-radius: 2px;
  }
   .message.ai .insert-btn {
    position: absolute;
    bottom: 5px;
    right: 8px;
    padding: 3px 7px;
    font-size: 0.7rem;
    background: var(--bg-hover-medium, rgba(255,255,255,0.1));
    color: var(--text-secondary, #a0a0a0);
    border: none;
    border-radius: 4px;
    cursor: pointer;
    opacity: 0.6;
    transition: opacity 0.2s ease, background-color 0.2s ease;
  }
  .message.ai:hover .insert-btn { opacity: 1; background: var(--bg-hover-strong, rgba(255,255,255,0.2)); }

  .left-column .write-chat-panel .write-chat-form input[type="text"] {
    flex-grow: 1;
    padding: 0.75rem 1rem;
    border: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
    border-radius: 6px;
    font-size: 0.9rem;
    background-color: var(--bg-input, var(--bg-primary, rgba(26, 26, 46, 0.7))); /* Slightly transparent for depth */
    color: var(--text-primary, #e0e0e0);
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
  }
  .left-column .write-chat-panel .write-chat-form input[type="text"]:focus {
    border-color: var(--accent-primary, #6d5ed9);
    box-shadow: 0 0 0 0.2rem rgba(109, 94, 217, .25); /* Derived from accent-primary */
    outline: none;
  }
  .left-column .write-chat-panel .write-chat-form button {
    padding: 0.75rem 1.2rem;
    background-color: var(--accent-primary, #6d5ed9);
    color: var(--text-primary, #e0e0e0); /* Text on accent bg should be light */
    border: none;
    border-radius: 6px;
    font-size: 0.9rem;
    font-weight: 500;
    cursor: pointer;
    transition: background-color 0.2s ease, box-shadow 0.2s ease;
  }
  .left-column .write-chat-panel .write-chat-form button:hover {
    background-color: var(--accent-secondary, #8a7ef9);
    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  }

  .left-column .save-tools-module {
    flex-shrink: 0;
    background: transparent; /* Or var(--bg-secondary) if more separation needed */
    padding: 1rem;
    margin: 0 -1.25rem -1.25rem -1.25rem;
    border-top: 1px solid var(--border-color-strong, rgba(160, 160, 160, 0.3));
  }
  .left-column .save-tools-module .tool-section h4 {
    font-size: 1em;
    color: var(--text-title, var(--accent-primary, #6d5ed9));
    margin-bottom: 0.75rem;
  }

  .save-btn, .save-as-btn {
    padding: 0.7rem 1.1rem !important;
    font-size: 0.9rem !important;
    border-radius: 6px !important;
    border: none !important;
    color: var(--text-primary, #e0e0e0) !important; /* Text on colored button */
    cursor: pointer;
    flex: 1;
    transition: background-color 0.2s ease, box-shadow 0.2s ease;
    font-weight: 500;
    text-align: center;
  }
  .save-btn {
    background-color: var(--success-color, #2ed573) !important; /* App.svelte var */
  }
  .save-btn:hover:not(:disabled) {
    filter: brightness(110%);
    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  }
  .save-as-btn {
    background-color: var(--accent-primary, #6d5ed9) !important; /* App.svelte var for general actions */
  }
  .save-as-btn:hover:not(:disabled) {
    background-color: var(--accent-secondary, #8a7ef9) !important;
    box-shadow: 0 2px 4px rgba(0,0,0,0.2);
  }
  .save-btn:disabled, .save-as-btn:disabled {
    opacity: 0.5;
  }

  .center-column {
    flex: 1 1 auto;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-primary, rgba(26, 26, 46, 0.95)); /* App.svelte var */
    /* No border needed to separate from main-content if same bg */
  }

  .center-column .editor-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 0;
    margin-bottom: 1rem;
    border-bottom: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
  }
  .view-mode-toggles button {
    padding: 0.5rem 1rem;
    background: transparent;
    border: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
    border-radius: 6px;
    color: var(--text-secondary, #a0a0a0); /* App.svelte var */
    cursor: pointer;
    font-size: 0.85rem;
    margin-left: 0.5rem;
    transition: background-color 0.2s, color 0.2s, border-color 0.2s;
  }
  .view-mode-toggles button.active {
    background-color: var(--accent-primary, #6d5ed9);
    color: var(--text-primary, #e0e0e0);
    border-color: var(--accent-primary, #6d5ed9);
  }
  .view-mode-toggles button:hover:not(.active) {
    background-color: var(--bg-hover-light, rgba(255, 255, 255, 0.05));
    border-color: var(--border-color-strong, rgba(160, 160, 160, 0.3));
  }

  .center-column .editor-container {
    flex-grow: 1;
    display: flex;
    flex-direction: column;
    border: 1px solid var(--border-color-strong, rgba(160, 160, 160, 0.3));
    border-radius: 6px;
    overflow: hidden;
    background-color: var(--bg-secondary, rgba(22, 33, 62, 0.9)); /* Slightly different for editor area */
  }

  .center-column .markdown-input {
    flex-grow: 1;
    width: 100%;
    border: none;
    padding: 1rem;
    font-family: var(--font-mono, 'Consolas', 'Monaco', 'Courier New', monospace);
    font-size: 1rem;
    line-height: 1.6;
    resize: none;
    box-sizing: border-box;
    background-color: transparent;
    color: var(--text-primary, #e0e0e0);
  }
  .center-column .markdown-input:focus {
      outline: none;
  }

  .markdown-preview-container {
    flex: 1;
    overflow-y: auto;
    padding: 1rem;
    background: var(--bg-secondary, rgba(22, 33, 62, 0.85)); /* Consistent with editor or slightly different */
    color: var(--text-primary, #e0e0e0);
  }

  .markdown-preview :global(h1),
  .markdown-preview :global(h2),
  .markdown-preview :global(h3),
  .markdown-preview :global(h4),
  .markdown-preview :global(h5),
  .markdown-preview :global(h6) {
    color: var(--text-title, var(--accent-secondary, #8a7ef9)); /* Use lighter accent for headers */
    border-bottom-color: var(--border-color-medium, rgba(160, 160, 160, 0.2));
  }
  .markdown-preview :global(a) {
    color: var(--text-accent, var(--accent-primary, #6d5ed9));
  }
  .markdown-preview :global(code) {
    background: var(--bg-primary, rgba(26, 26, 46, 0.7));
    color: var(--text-secondary, #a0a0a0);
    padding: 0.2em 0.4em;
    border-radius: 3px;
  }
  .markdown-preview :global(pre) {
    background: var(--bg-primary, rgba(26, 26, 46, 0.8));
    border: 1px solid var(--border-color-light, rgba(160, 160, 160, 0.1));
    padding: 1em;
    border-radius: 4px;
    overflow-x: auto;
  }
  .markdown-preview :global(pre code) {
    background: transparent;
    color: var(--text-primary, #e0e0e0);
    padding: 0;
  }
  .markdown-preview :global(blockquote) {
    border-left: 4px solid var(--accent-primary, #6d5ed9);
    background: var(--bg-secondary, rgba(22, 33, 62, 0.5));
    color: var(--text-secondary, #a0a0a0);
    padding: 0.5em 1em;
    margin-left: 0;
  }

  .right-column-toolbar {
    flex: 0 0 220px;
    background-color: var(--bg-secondary, rgba(22, 33, 62, 0.90)); /* Consistent with left column */
    border-left: 1px solid var(--border-color-strong, rgba(160, 160, 160, 0.3));
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .right-column-toolbar .tool-section {
    padding-bottom: 1rem;
    border-bottom: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
  }
  .right-column-toolbar .tool-section:last-child {
      border-bottom: none;
  }

  .right-column-toolbar .tool-section h4 {
    margin-top: 0;
    margin-bottom: 0.8rem;
    font-size: 0.95em;
    color: var(--text-title-secondary, var(--accent-secondary, #8a7ef9));
    border-bottom: none;
    padding-bottom: 0;
  }

  .right-column-toolbar .tool-section button {
    display: block;
    width: 100%;
    margin-bottom: 0.6rem;
    text-align: left;
    padding: 0.6rem 0.9rem;
    font-size: 0.85rem;
    background-color: var(--bg-button-tool, var(--bg-primary, rgba(26, 26, 46, 0.8)));
    color: var(--text-secondary, #a0a0a0);
    border: 1px solid var(--border-color-medium, rgba(160, 160, 160, 0.2));
    border-radius: 5px;
    cursor: pointer;
    transition: background-color 0.2s, color 0.2s, border-color 0.2s, box-shadow 0.2s;
  }
  .right-column-toolbar .tool-section button:hover {
    background-color: var(--bg-hover-medium, rgba(255, 255, 255, 0.1));
    border-color: var(--border-color-strong, rgba(160, 160, 160, 0.3));
    color: var(--text-primary, #e0e0e0);
    box-shadow: 0 1px 3px rgba(0,0,0,0.2);
  }
  .right-column-toolbar .tool-section button b,
  .right-column-toolbar .tool-section button i {
      margin-right: 0.5em;
      font-size: 1.1em;
      display: inline-block;
      width: 1.2em;
      text-align: center;
      color: var(--text-secondary, #a0a0a0); /* Icon color */
  }
  .right-column-toolbar .tool-section button:hover b,
  .right-column-toolbar .tool-section button:hover i {
      color: var(--text-primary, #e0e0e0); /* Icon color on hover */
  }

  .status-bar {
    padding: 0.5rem 1rem;
    font-size: 0.8rem;
    color: var(--text-secondary, #a0a0a0);
    background-color: var(--bg-secondary, rgba(22, 33, 62, 0.95));
    border-top: 1px solid var(--border-color-strong, rgba(160, 160, 160, 0.3));
    text-align: right;
  }
</style>
