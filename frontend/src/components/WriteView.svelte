<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate } from 'svelte';
  import { Marked } from 'marked'; // Import Marked class
  import { SaveLibraryFile, GetAIResponseWithContext, GetAllEntries, WeaveEntryIntoText, SaveTemplate } from '@wailsjs/go/main/App';
  import { database } from '@wailsjs/go/models';
  import DropContextMenu from './DropContextMenu.svelte'; // Import the new component
  import AutocompleteMenu from './AutocompleteMenu.svelte'; // Import the new component
  import CodexSelectorModal from './CodexSelectorModal.svelte';

  // --- REVISED Props ---
  export let documentContent: string = '';
  export let documentFilename: string = '';
  export let isDocumentDirty: boolean = false;
  export let templateType: string = 'blank';
  export let chatModelId: string = ''; // From global settings

  // --- Local State for UI/Modals ---
  let renderedWriteHtml = '';
  let markdownTextareaElement: HTMLTextAreaElement;
  let editorMode: 'split' | 'edit' | 'preview' = 'split';
  let wordCount: number = 0;
  let charCount: number = 0;
  let writeChatDisplayElement: HTMLDivElement; // For auto-scrolling
  let writeChatMessages: { sender: 'user' | 'ai', text: string }[] = [];
  let writeChatInput: string = '';
  let isWriteChatLoading: boolean = false;
  let writeChatError: string = '';
  let showWriteSaveModal: boolean = false;
  let filenameForSaveModal: string = ''; // Filename input in the modal
  let isSaving = false;
  let writeSaveError = '';
  let writeSaveSuccess = '';
  let isSaveAsOperation: boolean = false;

  // --- New State ---
  let codexEntries: database.CodexEntry[] = [];
  let codexSearchTerm: string = '';
  let showCodexPanel: boolean = true; // Or make it a tab

  let showDropMenu = false;
  let dropMenuX = 0;
  let dropMenuY = 0;
  let droppedEntry: database.CodexEntry | null = null;
  let showCodexSelector = false;
  let activeWritingWeave: { type: string, label: string } | null = null;
  let writingWeaveCursorPos = 0;
  let isWeaveDragOver = false;
  let dropIndicatorStyle = '';


  const writingWeaves = [
    { type: 'narrative', label: 'Narrative', description: 'Continue the story with action or events.', icon: '🏃' },
    { type: 'exposition', label: 'Exposition', description: 'Explain background or world details.', icon: '🌍' },
    { type: 'dialogue', label: 'Dialogue', description: 'Write a conversation between characters.', icon: '💬' },
    { type: 'description', label: 'Description', description: 'Describe a character, object, or scene.', icon: '🎨' },
    { type: 'introspection', label: 'Introspection', description: 'Explore a character\'s internal thoughts.', icon: '🧠' },
  ];
  let dropCursorPosition: number = 0;
  let isWeaving = false;

  // --- New State for Autocomplete ---
  let showAutocomplete = false;
  let autocompleteX = 0;
  let autocompleteY = 0;
  let autocompleteItems: database.CodexEntry[] = [];
  let autocompleteQuery = '';
  let autocompleteTriggerPos = 0;
  let autocompleteMenuRef: AutocompleteMenu;

  // --- New State ---
  let showSaveTemplateModal = false;
  let newTemplateName = '';

  const dispatch = createEventDispatcher();
  const marked = new Marked({ gfm: true, breaks: true });

  // Custom renderer for links
  // Custom renderer for links has been temporarily removed to resolve a build issue.
  // TODO: Re-implement the custom renderer with the correct signature for the installed marked version.

  // --- Lifecycle ---
  onMount(async () => {
      filenameForSaveModal = documentFilename || 'untitled.md'; // Default for modal
      
      // Focus editor on mount if in edit or split mode
      if (editorMode === 'edit' || editorMode === 'split') {
          markdownTextareaElement?.focus();
      }

      // Fetch codex entries for the panel
      try {
        codexEntries = await GetAllEntries() || [];
      } catch (err) {
        dispatch('error', 'Failed to load Codex entries for reference panel.');
      }
  });

  // --- REVISED Logic ---
  // The parent will pass down the new rendered HTML when content changes.
  $: if (documentContent !== undefined) {
    (async () => {
      try {
        const result = await marked.parse(documentContent || '');
        renderedWriteHtml = result;
      } catch (e) {
        console.error("Markdown parsing error:", e);
        renderedWriteHtml = "Error parsing markdown.";
      }
    })();
  }
  $: updateCounts(documentContent);

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
    const currentText = documentContent;

    dispatch('updatecontent', currentText.substring(0, start) + textToInsert + currentText.substring(end));
    
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
    if (isDocumentDirty && !confirm("You have unsaved changes. Are you sure you want to leave?")) {
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
        const draftContext = documentContent.length > 2000 ? documentContent.substring(0, 2000) + "\n...[Draft Truncated]..." : documentContent;
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

      if (isSaveAs || !documentFilename) {
          // For "Save As" or if no current filename, suggest based on content or use "untitled"
          if (documentFilename && isSaveAs) {
              const baseName = documentFilename.replace(/\.[^/.]+$/, '');
              filenameForSaveModal = `${baseName}_copy.md`;
          } else if (documentContent.trim()) {
              const firstLine = documentContent.trim().split('\n')[0];
              filenameForSaveModal = firstLine.substring(0, 30).replace(/[^a-z0-9\s._-]/gi, '').replace(/\s+/g, '-') + '.md';
          } else {
              filenameForSaveModal = 'untitled.md';
          }
      } else {
          // For regular "Save", use the current document's filename
          filenameForSaveModal = documentFilename;
      }
      showWriteSaveModal = true;
  }

  // REVISE event handlers
  function handleDirectSave() {
    if (!documentFilename) {
      openSaveModal(false); // Still need to prompt for name if it's a new doc
    } else {
      // Dispatch a request to the parent to save with the current filename
      dispatch('saverequest', { filename: documentFilename, isSaveAs: false });
    }
  }

  function doSaveFromModal() {
    // This function now only dispatches the save request with the new filename
    if (!filenameForSaveModal.trim()) {
      writeSaveError = 'Filename cannot be empty.';
      return;
    }
    let finalFilename = filenameForSaveModal.trim();
    if (!finalFilename.toLowerCase().endsWith('.md')) finalFilename += '.md';
    
    dispatch('saverequest', { filename: finalFilename, isSaveAs: isSaveAsOperation });
  }

  // NEW functions to be called by parent App.svelte
  export function setSavingState(saving: boolean, successMsg: string = '', errorMsg: string = '') {
    isSaving = saving;
    writeSaveSuccess = successMsg;
    writeSaveError = errorMsg;
    
    if (successMsg) {
      setTimeout(() => {
        writeSaveSuccess = '';
        if (showWriteSaveModal) showWriteSaveModal = false;
      }, 2000);
    }
  }

  // --- Write Mode Formatting Tools Function ---
  function applyMarkdownFormat(formatType: 'bold' | 'italic' | 'h1' | 'h2' | 'h3' | 'code' | 'blockquote') {
    if (!markdownTextareaElement) return;

    const start = markdownTextareaElement.selectionStart;
    const end = markdownTextareaElement.selectionEnd;
    const selectedText = documentContent.substring(start, end);
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
            const lineStart = documentContent.lastIndexOf('\n', start - 1) + 1;
            // For block quotes, we might need to apply to multiple lines if selected
            if (formatType === 'blockquote' && selectedText.includes('\n')) {
                 const lines = selectedText.split('\n');
                 const prefixedLines = lines.map(line => blockPrefix + line).join('\n');
                 dispatch('updatecontent', documentContent.substring(0, start) + prefixedLines + documentContent.substring(end));
                 markdownTextareaElement.selectionStart = start;
                 markdownTextareaElement.selectionEnd = start + prefixedLines.length;
            } else {
                 // Apply prefix at the beginning of the line for headings
                 dispatch('updatecontent', documentContent.substring(0, lineStart) + blockPrefix + documentContent.substring(lineStart));
                 // Adjust selection points after adding prefix
                 markdownTextareaElement.selectionStart = start + blockPrefix.length;
                 markdownTextareaElement.selectionEnd = end + blockPrefix.length;
            }
        } else {
            // Apply inline formats (bold, italic, code)
            const newText = documentContent.substring(0, start) + prefix + selectedText + suffix + documentContent.substring(end);
            dispatch('updatecontent', newText);

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

  // --- New Drag and Drop Handlers ---
  function handleDragStart(event: DragEvent, entry: database.CodexEntry) {
    event.dataTransfer?.setData('application/json', JSON.stringify(entry));
  }



    function getCoordsFromPos(pos: number): { x: number, y: number } {
    if (!markdownTextareaElement) return { x: 0, y: 0 };

    const textarea = markdownTextareaElement;
    const style = window.getComputedStyle(textarea);
    const paddingLeft = parseFloat(style.paddingLeft) || 0;
    const paddingTop = parseFloat(style.paddingTop) || 0;
    
    // Use same measurements as getCursorPositionFromMouseEvent
    const fontSize = parseFloat(style.fontSize) || 16;
    let lineHeight = parseFloat(style.lineHeight);
    if (isNaN(lineHeight) || lineHeight < fontSize) {
      lineHeight = fontSize * 1.5;
    }
    const charWidth = fontSize * 0.6;
    
    // Account for visual line wrapping
    const textareaWidth = textarea.clientWidth - paddingLeft - parseFloat(style.paddingRight || '0');
    const charsPerLine = Math.floor(textareaWidth / charWidth);
    
    const textUpToPos = textarea.value.substring(0, pos);
    const lines = textUpToPos.split('\n');
    const lineIndex = lines.length - 1;
    const currentLineText = lines[lineIndex];
    
    // Calculate visual line position accounting for wrapping
    let visualLineIndex = 0;
    for (let i = 0; i < lineIndex; i++) {
      const line = textarea.value.split('\n')[i];
      const wrappedLines = Math.max(1, Math.ceil(line.length / charsPerLine));
      visualLineIndex += wrappedLines;
    }
    
    // Add position within current line
    const charIndexInLine = currentLineText.length;
    const visualLineWithinCurrent = Math.floor(charIndexInLine / charsPerLine);
    visualLineIndex += visualLineWithinCurrent;
    
    const y = (visualLineIndex * lineHeight) + paddingTop - textarea.scrollTop;
    
    // Calculate x position within the wrapped line
    const charWithinWrappedLine = charIndexInLine % charsPerLine;
    const x = (charWithinWrappedLine * charWidth) + paddingLeft - textarea.scrollLeft;

    return { x, y };
  }

    function handleDragEnter(event: DragEvent) {
    if (event.dataTransfer?.types.includes('application/x-llore-writing-weave') || 
        event.dataTransfer?.types.includes('application/json')) {
      isWeaveDragOver = true;
    }
  }

  function handleDragLeave(event: DragEvent) {
    const target = event.currentTarget as HTMLElement;
    const rect = target.getBoundingClientRect();
    if (event.clientX <= rect.left || event.clientX >= rect.right || event.clientY <= rect.top || event.clientY >= rect.bottom) {
      isWeaveDragOver = false;
    }
  }

  function handleDragOver(event: DragEvent) {
    if (event.dataTransfer?.types.includes('application/x-llore-writing-weave') ||
        event.dataTransfer?.types.includes('application/json')) {
      event.preventDefault(); // Allow drop
      const pos = getCursorPositionFromMouseEvent(event);
      const coords = getCoordsFromPos(pos);
      dropIndicatorStyle = `top: ${coords.y}px; left: ${coords.x}px;`;
    }
  }

    

  function handleDrop(event: DragEvent) {
    isWeaveDragOver = false;
    event.preventDefault();
    event.stopPropagation();

    if (!event.dataTransfer || !markdownTextareaElement) return;

    const jsonData = event.dataTransfer.getData('application/json');
    const textData = event.dataTransfer.getData('text/plain');

    if (jsonData) {
      try {
        const dropData = JSON.parse(jsonData);
        if (dropData.type === 'writing-weave') {
          // Handle writing weave drop
          const cursorPos = getCursorPositionFromMouseEvent(event);
          activeWritingWeave = dropData.weave;
          writingWeaveCursorPos = cursorPos;
          showCodexSelector = true;
        } else {
          // Handle codex entry drop (has id, name, type, content properties)
          droppedEntry = dropData;
          dropMenuX = event.clientX;
          dropMenuY = event.clientY;
          const target = event.target as HTMLTextAreaElement;
          dropCursorPosition = target.selectionStart;
          showDropMenu = true;
        }
      } catch (e) {
        console.error('Error parsing JSON drop data:', e);
      }
    } else if (textData) {
      // Handle plain text drop
      const cursorPos = getCursorPositionFromMouseEvent(event);
      insertTextAt(textData, cursorPos);
    }
  }

  function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      showWriteSaveModal = false;
    }
  }

  function handleDropMenuAction(event: CustomEvent<'reference' | 'weave'>) {
    const action = event.detail;
    showDropMenu = false;
    if (!droppedEntry) return;

    if (action === 'reference') {
      const referenceText = `[@${droppedEntry.name}](codex://entry/${droppedEntry.id})`;
      insertTextAt(referenceText, dropCursorPosition);
    } else if (action === 'weave') {
      performLloreWeaving();
    }
  }

  // --- New "Weaving" Function ---
  async function performLloreWeaving() {
    if (!droppedEntry) return;
    isWeaving = true;
    dispatch('loading', true);
    let weavingIndicator = '... weaving ...';
    insertTextAt(weavingIndicator, dropCursorPosition);

    try {
      const generatedText = await WeaveEntryIntoText(
        droppedEntry,
        documentContent.replace(weavingIndicator, ''), // Send content without the indicator
        dropCursorPosition,
        templateType
      );

      // Replace indicator with generated text
      dispatch('updatecontent', documentContent.replace(weavingIndicator, `\n${generatedText.trim()}\n`));
    } catch(err) {
      dispatch('error', `Llore-weaving failed: ${err}`);
      dispatch('updatecontent', documentContent.replace(weavingIndicator, '')); // Remove indicator on error
    } finally {
      isWeaving = false;
      dispatch('loading', false);
    }
  }

  // Helper to insert text at a specific position
  function insertTextAt(text: string, position: number) {
    dispatch('updatecontent', documentContent.slice(0, position) + text + documentContent.slice(position));
  }

  // Computed property for filtered codex entries
  $: filteredCodexEntries = codexSearchTerm 
    ? codexEntries.filter(e => e.name.toLowerCase().includes(codexSearchTerm.toLowerCase()))
    : codexEntries;

  // Function to get cursor coordinates
  function getCursorXY() {
    // This is a simplified approach. A real implementation might use a library
    // or a hidden div to get precise coordinates.
    const ta = markdownTextareaElement;
    const style = window.getComputedStyle(ta);
    const lineHeight = parseFloat(style.lineHeight);
    const textUptoCursor = ta.value.substring(0, ta.selectionStart);
    const lines = textUptoCursor.split('\n');
    const currentLine = lines[lines.length - 1];
    
    // Estimate position
    const rect = ta.getBoundingClientRect();
    autocompleteX = rect.left + (currentLine.length * 8) + 15; // 8 is a rough char width
    autocompleteY = rect.top + (lines.length * lineHeight) + 5;
  }

  function handleWriteViewKeydown(event: KeyboardEvent) {
    if (showAutocomplete) {
      autocompleteMenuRef.handleKeyDown(event);
      return; // Let the menu handle key events
    }
    // ... (keep existing keydown logic for Ctrl+B/I/S and Tab)
  }
  
  function handleInput(event: Event) {
    const target = event.target as HTMLTextAreaElement;
    dispatch('updatecontent', target.value);

    // Also handle autocomplete logic
    const cursorPos = target.selectionStart;
    const textBeforeCursor = target.value.substring(0, cursorPos);
    const atMatch = textBeforeCursor.match(/@(\w*)$/);

    if (atMatch) {
      autocompleteTriggerPos = atMatch.index!;
      autocompleteQuery = atMatch[1].toLowerCase();
      
      autocompleteItems = codexEntries.filter(e =>
        e.name.toLowerCase().startsWith(autocompleteQuery)
      );

      if (autocompleteItems.length > 0) {
        getCursorXY();
        showAutocomplete = true;
      } else {
        showAutocomplete = false;
      }
    } else {
      showAutocomplete = false;
    }
  }

  function handleAutocompleteSelect(event: CustomEvent<database.CodexEntry>) {
    const entry = event.detail;
    const referenceText = `[@${entry.name}](codex://entry/${entry.id}) `;
    
    // Replace from the '@' trigger position
    const textBefore = documentContent.substring(0, autocompleteTriggerPos);
    const textAfter = documentContent.substring(autocompleteTriggerPos + autocompleteQuery.length + 1);
    
    dispatch('updatecontent', textBefore + referenceText + textAfter);
    showAutocomplete = false;
    
    // Move cursor after the inserted text
    requestAnimationFrame(() => {
      if (!markdownTextareaElement) return;
      const newCursorPos = autocompleteTriggerPos + referenceText.length;
      markdownTextareaElement.focus();
      markdownTextareaElement.selectionStart = newCursorPos;
      markdownTextareaElement.selectionEnd = newCursorPos;
    });
  }
  // --- New Function ---
  async function handleSaveAsTemplate() {
    if (!newTemplateName.trim()) {
      // You can show an error in the modal
      return;
    }
    try {
      await SaveTemplate(newTemplateName, documentContent);
      alert(`Template '${newTemplateName}.md' saved successfully!`);
      showSaveTemplateModal = false;
      newTemplateName = '';
    } catch (err) {
      alert(`Failed to save template: ${err}`);
    }
  }

  // NEW: The core function that is called after the user selects their context entries (or none)
    async function handleWritingWeave(event: CustomEvent<{ selectedEntries: database.CodexEntry[], selectedLength: string }>) {
    const { selectedEntries, selectedLength } = event.detail;
    showCodexSelector = false;
    
    if (!activeWritingWeave || writingWeaveCursorPos === null) return;

    isWeaving = true;
    dispatch('loading', true);
    
    try {
      const textBeforeCursor = documentContent.substring(0, writingWeaveCursorPos);
      const textAfterCursor = documentContent.substring(writingWeaveCursorPos);

      const contextEntries = selectedEntries.map(entry => `${entry.name} (${entry.type}): ${entry.content}`).join('\n\n');
      
      // Convert length selection to prompt instruction
      const lengthInstructions = {
        'small': 'Keep your response to exactly 1 sentence that flows naturally.',
        'medium': 'Write approximately 1 paragraph (3-5 sentences) that develops the scene.',
        'large': 'Write approximately 1 page worth of content (multiple paragraphs, around 200-400 words).',
        'extra-large': 'Write approximately 2 pages worth of content (multiple paragraphs, around 400-800 words).'
      };
      
      const lengthInstruction = lengthInstructions[selectedLength] || lengthInstructions['medium'];
      
      const prompt = `You are a subtle and masterful fiction writing assistant. Your task is to weave a '${activeWritingWeave.label}' element into the document at the user's cursor position.\nContinue the story from the text provided before the cursor, and ensure it flows naturally into the text after the cursor.\n\nWhen incorporating the context entries, do so with nuance. Do not simply state the information from the context. Instead, use it to inform the atmosphere, character voice, or narrative direction. The insertion should feel like a natural continuation of the story, enhancing it without being jarring or overly explicit.\n\nLENGTH REQUIREMENT: ${lengthInstruction}\n\nText before cursor:\n---\n${textBeforeCursor.slice(-2000)}\n---\n\nText after cursor:\n---\n${textAfterCursor.substring(0, 2000)}\n---\n\nContext entries for inspiration:\n---\n${contextEntries || 'No specific context provided.'}\n---\n\nBased on the weave type ('${activeWritingWeave.label}') and the provided context, generate only the new text to be inserted between the 'before' and 'after' sections. The generated text should blend seamlessly and match the specified length requirement.`;
      
      const generatedText = await GetAIResponseWithContext(prompt, chatModelId);
      
      insertTextAt(generatedText, writingWeaveCursorPos);

    } catch (err) {
      dispatch('error', `Writing Weaving failed: ${err}`);
    } finally {
      isWeaving = false;
      dispatch('loading', false);
      activeWritingWeave = null;
    }
  }

  function openWritingWeave(event: MouseEvent, node: { type: string, label: string }) {
    event.stopPropagation();
    if (!markdownTextareaElement) return;
    
    activeWritingWeave = node;
    writingWeaveCursorPos = markdownTextareaElement.selectionStart;
    showCodexSelector = true;
  }

  // NEW: Handle drag start for writing weave buttons
  function handleWeaveButtonDragStart(event: DragEvent, weave: { type: string, label: string, description: string, icon: string }) {
    if (!event.dataTransfer) return;
    
    const payload = {
      type: 'writing-weave',
      weave: { type: weave.type, label: weave.label, description: weave.description, icon: weave.icon }
    };
    
    // Set both custom type and JSON payload for the new drag handlers
    event.dataTransfer.setData('application/x-llore-writing-weave', 'true');
    event.dataTransfer.setData('application/json', JSON.stringify(payload));
    event.dataTransfer.effectAllowed = 'move';
  }



  // Helper function to get cursor position from mouse event
  function getCursorPositionFromMouseEvent(event: DragEvent): number {
    if (!markdownTextareaElement) return 0;

    const textarea = markdownTextareaElement;
    const rect = textarea.getBoundingClientRect();
    const style = window.getComputedStyle(textarea);
    
    // Simple approach: just use basic math
    const paddingLeft = parseFloat(style.paddingLeft) || 0;
    const paddingTop = parseFloat(style.paddingTop) || 0;
    
    // Mouse position relative to textarea content
    const x = event.clientX - rect.left - paddingLeft + textarea.scrollLeft;
    const y = event.clientY - rect.top - paddingTop + textarea.scrollTop;
    
    // Get basic measurements
    const fontSize = parseFloat(style.fontSize) || 16;
    
    // Try to get actual line height from computed style first
    let lineHeight = parseFloat(style.lineHeight);
    if (isNaN(lineHeight) || lineHeight < fontSize) {
      lineHeight = fontSize * 1.5; // More generous line height
    }
    
    const charWidth = fontSize * 0.6; // Slightly wider character estimate
    
    // Account for visual line wrapping
    const textareaWidth = textarea.clientWidth - paddingLeft - parseFloat(style.paddingRight || '0');
    const charsPerLine = Math.floor(textareaWidth / charWidth);
    
    // Calculate which visual line we're on
    const visualLineIndex = Math.floor(y / lineHeight);
    
    // Split into lines and find position accounting for wrapping
    const lines = textarea.value.split('\n');
    
    let currentVisualLine = 0;
    let position = 0;
    let targetLine = -1;
    let targetChar = 0;
    
    // Find which actual line contains our target visual line
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      const wrappedLines = Math.max(1, Math.ceil(line.length / charsPerLine));
      
      if (currentVisualLine + wrappedLines > visualLineIndex) {
        // This line contains our target
        targetLine = i;
        const visualLineWithinThisLine = visualLineIndex - currentVisualLine;
        targetChar = Math.min(visualLineWithinThisLine * charsPerLine + Math.floor(x / charWidth), line.length);
        break;
      }
      
      currentVisualLine += wrappedLines;
      position += line.length + 1; // +1 for newline
    }
    
    // If we found a target line, calculate final position
    if (targetLine >= 0) {
      position += targetChar;
    } else {
      // Fallback to end of document
      position = textarea.value.length;
    }
    
    console.log(`visualLine=${visualLineIndex}, targetLine=${targetLine}, targetChar=${targetChar}, finalPosition=${position}`);
    
    return Math.min(position, textarea.value.length);
  }
</script>





<button class="back-btn" on:click={goBack}>← Back to Write Hub</button>

<div class="write-view-main-content">
  <!-- LEFT COLUMN: Chat and Tools -->
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
          <button class="save-btn" on:click={handleDirectSave} disabled={isSaving || !isDocumentDirty}>
            Save {#if isDocumentDirty && documentFilename}*{/if}
          </button>
          <button class="save-as-btn" on:click={() => openSaveModal(true)} disabled={isSaving}>Save As...</button>
          <button class="template-btn" on:click={() => showSaveTemplateModal = true} disabled={isSaving}>Save as Template</button>
        </div>
        <div class="doc-info">
          <span>Chars: {charCount}</span>
          <span>Words: {wordCount}</span>
        </div>
      </div>
    </div>
  </div>

  <!-- CENTER COLUMN: Editor -->
  <div class="center-column">
    <div class="editor-toolbar">
      <div class="view-mode-toggles">
        <button class:active={editorMode === 'edit'} on:click={() => editorMode = 'edit'} title="Edit mode">📝 Edit</button>
        <button class:active={editorMode === 'split'} on:click={() => editorMode = 'split'} title="Split mode">⚔️ Split</button>
        <button class:active={editorMode === 'preview'} on:click={() => editorMode = 'preview'} title="Preview mode">👁️ Preview</button>
      </div>
      <div class="current-file-display">
        {documentFilename || "New Document"}{#if isDocumentDirty && documentFilename}*{/if}
      </div>
    </div>
    <div class="editor-pane" style="position: relative;">
      {#if isWeaveDragOver}
      <div class="drop-indicator" style={dropIndicatorStyle}></div>
    {/if}
      <textarea
        class="markdown-input"
        value={documentContent}
        on:input={handleInput}
        bind:this={markdownTextareaElement}
        placeholder="Start writing your masterpiece (Markdown supported)..."
        style="display: {editorMode === 'preview' ? 'none' : 'block'}"
        on:drop={handleDrop}
        on:dragenter={handleDragEnter}
        on:dragleave={handleDragLeave}
        on:dragover={handleDragOver}
        on:dragover|preventDefault
        on:keydown={handleWriteViewKeydown}
      ></textarea>
      <div 
        class="markdown-preview-container"
        style="display: {editorMode === 'edit' ? 'none' : 'block'}"
      >
        <div class="markdown-preview">{@html renderedWriteHtml}</div>
      </div>
    </div>
  </div>

  <!-- RIGHT COLUMN: Codex Reference & AI Tools -->
  <div class="right-column-toolbar">
    <div class="tool-section codex-reference-panel">
      <h4>Codex Reference</h4>
      <input type="search" placeholder="Search Codex..." bind:value={codexSearchTerm} class="codex-search"/>
      <div class="codex-entry-list">
        {#each filteredCodexEntries as entry (entry.id)}
          <div 
            class="codex-item"
            role="button"
            tabindex="0"
            draggable="true"
            on:dragstart={(e) => handleDragStart(e, entry)}
            on:keydown={(e) => {
              if (e.key === 'Enter') {
                const referenceText = `[@${entry.name}](codex://entry/${entry.id})`;
                insertTextAt(referenceText, markdownTextareaElement.selectionStart);
              }
            }}
          >
            <strong>{entry.name}</strong>
            <span>({entry.type})</span>
          </div>
        {/each}
      </div>
    </div>
    <!-- ... (existing formatting and AI action tools) ... -->
    <div class="tool-section">
      <h4>Formatting</h4>
      <div class="formatting-buttons">
        <button on:click={() => applyMarkdownFormat('bold')} title="Bold (Ctrl+B)"><b>B</b></button>
        <button on:click={() => applyMarkdownFormat('italic')} title="Italic (Ctrl+I)"><i>I</i></button>
        <button on:click={() => applyMarkdownFormat('h1')} title="Heading 1">H1</button>
        <button on:click={() => applyMarkdownFormat('h2')} title="Heading 2">H2</button>
        <button on:click={() => applyMarkdownFormat('h3')} title="Heading 3">H3</button>
        <button on:click={() => applyMarkdownFormat('code')} title="Code (`code`)">{"</>"}</button>
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
    
    <!-- Writing Weaving Section -->
    <div class="tool-section">
      <h4>Writing Weaving</h4>
      <div class="writing-weave-buttons">
        {#each writingWeaves as weave (weave.type)}
          <button 
            on:click={(e) => openWritingWeave(e, weave)} 
            title={weave.description}
            draggable="true"
            on:dragstart={(e) => handleWeaveButtonDragStart(e, weave)}
          >
            <span class="icon">{weave.icon}</span> {weave.label}
          </button>
        {/each}
      </div>
    </div>
  </div>
</div>

<!-- Drop Context Menu (new) -->
{#if showDropMenu}
  <DropContextMenu x={dropMenuX} y={dropMenuY} on:action={handleDropMenuAction} />
  <!-- Click outside to close -->
  <div class="overlay" role="button" tabindex="0" on:click={() => showDropMenu = false} on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') showDropMenu = false; }} />
{/if}

<!-- Save Modal -->
<!-- Add the Autocomplete Menu component -->
{#if showAutocomplete}
  <AutocompleteMenu
    bind:this={autocompleteMenuRef}
    items={autocompleteItems}
    x={autocompleteX}
    y={autocompleteY}
    on:select={handleAutocompleteSelect}
  />
  <!-- Overlay to close autocomplete on click outside -->
  <div class="overlay" role="button" tabindex="0" on:click={() => showAutocomplete = false} on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') showAutocomplete = false; }} />
{/if}

{#if showWriteSaveModal}
  <div class="modal-backdrop">
    <div class="modal save-write-modal">
      <h3>{isSaveAsOperation || !documentFilename ? 'Save As' : 'Save File'}</h3>
      <label for="write-filename">Filename: {isDocumentDirty ? '*' : ''}</label>
      <input id="write-filename" type="text" bind:value={filenameForSaveModal} placeholder="e.g., chapter-one.md">
      {#if writeSaveError}
        <p class="error-message small">{writeSaveError}</p>
      {/if}
      {#if writeSaveSuccess}
        <p class="success-message small">{writeSaveSuccess}</p>
      {/if}
      <div class="modal-buttons">
        <button on:click={doSaveFromModal} disabled={isSaving || !filenameForSaveModal.trim()}>
            {#if isSaving}Saving...{:else}Save{/if}
        </button>
        <button on:click={() => { showWriteSaveModal = false; writeSaveSuccess = ''; writeSaveError = ''; }} disabled={isSaving}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<!-- Add the new modal -->
{#if showSaveTemplateModal}
  <div class="modal-backdrop">
    <div class="modal save-template-modal">
      <h3>Save as Template</h3>
      <p>Save the current document's content as a reusable template.</p>
      <label for="template-name">Template Name:</label>
      <input id="template-name" type="text" bind:value={newTemplateName} placeholder="e.g., Character Deep Dive" />
      <div class="modal-buttons">
        <button on:click={handleSaveAsTemplate} disabled={!newTemplateName.trim()}>Save Template</button>
        <button on:click={() => showSaveTemplateModal = false}>Cancel</button>
      </div>
    </div>
  </div>
{/if}

<!-- Weaving Loading Modal -->
{#if isWeaving}
  <div class="modal-backdrop">
    <div class="modal weaving-modal">
      <div class="weaving-content">
        <div class="weaving-spinner">✨</div>
        <h3>Weaving...</h3>
        <p>Crafting your narrative enhancement</p>
      </div>
    </div>
  </div>
{/if}

<style>
  /* NEW STYLES for rendered @mentions */
  .markdown-preview :global(.codex-mention) {
    background-color: rgba(109, 94, 217, 0.2); /* Use accent color but subtle */
    color: var(--accent-secondary);
    padding: 0.1em 0.4em;
    border-radius: 4px;
    font-weight: 500;
    cursor: help; /* Indicate it's interactive */
    border-bottom: 1px dotted var(--accent-secondary);
  }
  /* ... (keep most existing styles) ... */

  /* NEW STYLES for Codex Reference Panel */
  .codex-reference-panel {
    display: flex;
    flex-direction: column;
    height: 100%; /* Or set a max-height */
  }

  .codex-search {
    width: 100%;
    padding: 0.5rem;
    margin-bottom: 0.75rem;
    background: var(--bg-primary);
    border: 1px solid var(--border-color-medium);
    border-radius: 4px;
    color: var(--text-primary);
  }

  .codex-entry-list {
    flex-grow: 1;
    overflow-y: auto;
  }

  .codex-item {
    padding: 0.5rem;
    border-radius: 4px;
    cursor: grab;
    margin-bottom: 0.25rem;
    border: 1px solid transparent;
  }
  .codex-item:hover {
    background-color: var(--bg-hover-medium);
    border-color: var(--border-color-strong);
  }
  .codex-item span { color: var(--text-secondary); font-size: 0.8rem; margin-left: 0.5rem; }

  /* Overlay for closing the context menu */
  .overlay {
    position: fixed;
    top: 0; left: 0; right: 0; bottom: 0;
    z-index: 1000;
  }
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
  .template-btn {
    /* Style it differently, maybe with a different color */
    background-color: #fdcb6e !important; /* A gold/yellow color */
    color: #2d3436 !important;
  }
  .template-btn:hover:not(:disabled) {
    background-color: #ffeaa7 !important;
  }

  .center-column {
    flex: 1 1 auto;
    display: flex;
    flex-direction: column;
    background-color: var(--bg-primary, rgba(26, 26, 46, 0.95)); /* App.svelte var */
    /* No border needed to separate from main-content if same bg */
  }

  .editor-pane {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0; /* Important for flex child to shrink */
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

  /* Writing Weaving Button Styles */
  .writing-weave-buttons {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
  
  .writing-weave-buttons button {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    text-align: left;
    cursor: grab;
  }
  
  .writing-weave-buttons button:active {
    cursor: grabbing;
  }
  
  .writing-weave-buttons button .icon {
    font-size: 1.2em;
    width: 1.5em;
    text-align: center;
  }

  .drop-indicator {
    position: absolute;
    width: 2px;
    height: 1.5em; /* Should roughly match line-height */
    background-color: var(--accent-primary);
    animation: blink 1s infinite steps(1, start);
    pointer-events: none; /* So it doesn't interfere with other events */
    z-index: 10;
    transform: translateY(-2px); /* Minor adjustment */
  }

  @keyframes blink {
    0% { opacity: 1; }
    50% { opacity: 0; }
    100% { opacity: 1; }
  }

  .status-bar {
    padding: 0.5rem 1rem;
    font-size: 0.8rem;
    color: var(--text-secondary, #a0a0a0);
    background-color: var(--bg-secondary, rgba(22, 33, 62, 0.95));
    border-top: 1px solid var(--border-color-strong, rgba(160, 160, 160, 0.3));
    text-align: right;
  }

  /* Save Modal Styles */
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    background: rgba(0, 0, 0, 0.7);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
  }

  .modal {
    background: var(--bg-primary);
    border: 1px solid var(--border-color-medium);
    border-radius: 8px;
    padding: 1.5rem;
    min-width: 400px;
    max-width: 500px;
  }

  .modal h3 {
    margin: 0 0 1rem 0;
    color: var(--text-primary);
  }

  .modal-content {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .filename-input {
    padding: 0.5rem;
    background: var(--bg-secondary);
    border: 1px solid var(--border-color-medium);
    border-radius: 4px;
    color: var(--text-primary);
    font-size: 1rem;
  }

  .modal-buttons {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  .modal-buttons button {
    padding: 0.5rem 1rem;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-weight: 500;
  }

  /* Weaving Modal Styles */
  .weaving-modal {
    text-align: center;
    min-width: 300px;
    max-width: 400px;
  }

  .weaving-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
  }

  .weaving-spinner {
    font-size: 2rem;
    animation: pulse 1.5s ease-in-out infinite;
  }

  .weaving-modal h3 {
    margin: 0;
    color: var(--text-primary);
    font-size: 1.5rem;
  }

  .weaving-modal p {
    margin: 0;
    color: var(--text-secondary);
    font-style: italic;
  }

  @keyframes pulse {
    0%, 100% {
      opacity: 0.6;
      transform: scale(1);
    }
    50% {
      opacity: 1;
      transform: scale(1.1);
    }
  }

  .cancel-btn {
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color-medium);
  }

  .cancel-btn:hover {
    background: var(--bg-hover-medium);
  }

  .save-btn {
    background: var(--accent-primary);
    color: white;
  }

  .save-btn:hover:not(:disabled) {
    background: var(--accent-secondary);
  }

  .save-btn:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }

  .error-message {
    color: #ff6b6b;
    font-size: 0.9rem;
    margin: 0;
  }

  .success-message {
    color: #51cf66;
    font-size: 0.9rem;
    margin: 0;
  }
</style>

<!-- Save Modal -->
{#if showWriteSaveModal}
  <div class="modal-backdrop" role="button" tabindex="-1" on:click={() => showWriteSaveModal = false} on:keydown={handleKeyDown}>
    <div class="modal save-modal" role="dialog" aria-modal="true">
      <h3>{isSaveAsOperation ? 'Save As' : 'Save'}</h3>
      <div class="modal-content">
        <label for="filename-input">Filename:</label>
        <input 
          id="filename-input"
          type="text" 
          bind:value={filenameForSaveModal} 
          placeholder="Enter filename..."
          class="filename-input"
        />
        <div class="modal-buttons">
          <button class="cancel-btn" on:click={() => showWriteSaveModal = false}>Cancel</button>
          <button class="save-btn" on:click={doSaveFromModal} disabled={isSaving || !filenameForSaveModal.trim()}>
            {isSaving ? 'Saving...' : 'Save'}
          </button>
        </div>
        {#if writeSaveError}
          <p class="error-message">{writeSaveError}</p>
        {/if}
        {#if writeSaveSuccess}
          <p class="success-message">{writeSaveSuccess}</p>
        {/if}
      </div>
    </div>
  </div>
{/if}

<!-- Codex Selector Modal for Writing Weaving -->
{#if showCodexSelector}
  
  <CodexSelectorModal
    allEntries={codexEntries}
    nodeType={activeWritingWeave?.label || ''}
    on:close={() => showCodexSelector = false}
    on:weave={handleWritingWeave}
  />
{/if}
