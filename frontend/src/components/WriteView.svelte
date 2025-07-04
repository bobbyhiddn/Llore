<script lang="ts">
  import { createEventDispatcher, onMount, afterUpdate } from 'svelte';
  import { Marked } from 'marked'; // Import Marked class
  import { SaveLibraryFile, GetAIResponseWithContext, GetAllEntries, WeaveEntryIntoText, SaveTemplate, ProcessStory } from '@wailsjs/go/main/App';
  import { database, llm } from '@wailsjs/go/models';
  import DropContextMenu from './DropContextMenu.svelte'; // Import the new component
  import AutocompleteMenu from './AutocompleteMenu.svelte'; // Import the new component
  import CodexSelectorModal from './CodexSelectorModal.svelte';
  import ChatMessageMenu from './ChatMessageMenu.svelte';
  import StoryImportStatus from './StoryImportStatus.svelte';
  import '../styles/WriteView.css';

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
  let writeChatMessages: { sender: 'user' | 'ai', text: string, html?: string }[] = [];
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
  let writingWeaveSelectionEnd = 0; // To preserve selection across modals
  let isWeaveDragOver = false;
  let dropIndicatorStyle = '';
  let activeMenuMessageIndex: number | null = null;


  const writingWeaves = [
    { type: 'narrative', label: 'Narrative', description: 'Continue the story with action or events.', icon: '🏃' },
    { type: 'exposition', label: 'Exposition', description: 'Explain background or world details.', icon: '🌍' },
    { type: 'dialogue', label: 'Dialogue', description: 'Write a conversation between characters.', icon: '💬' },
    { type: 'description', label: 'Description', description: 'Describe a character, object, or scene.', icon: '🎨' },
    { type: 'introspection', label: 'Introspection', description: 'Explore a character\'s internal thoughts.', icon: '🧠' },
  ];
  let dropCursorPosition: number = 0;
  let isWeaving = false;
  type IndexingStatus = 'idle' | 'indexing' | 'complete' | 'error';
  let indexingStatus: IndexingStatus = 'idle';
  let indexingError: string | null = null;
  let newIndexedEntries: database.CodexEntry[] = [];
  let updatedIndexedEntries: database.CodexEntry[] = [];

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

  // --- Error Modal State ---
  let showErrorModal = false;
  let errorModalTitle = '';
  let errorModalMessage = '';

  // --- Undo/Redo State ---
  let undoStack: string[] = [];
  let redoStack: string[] = [];
  let lastHistoryPush = 0;
  const historyDebounce = 1000; // ms

  const dispatch = createEventDispatcher();
  const marked = new Marked({ gfm: true, breaks: true });

  // Custom renderer for links
  // Custom renderer for links has been temporarily removed to resolve a build issue.
  // TODO: Re-implement the custom renderer with the correct signature for the installed marked version.

  // --- Lifecycle ---
  onMount(async () => {
      // Initialize undo stack with the starting content
      undoStack = [documentContent || ''];

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

  // --- Undo/Redo Functions ---
  function pushToHistory(content: string, force = false) {
    const now = Date.now();
    // Debounce pushes unless forced
    if (!force && now - lastHistoryPush < historyDebounce) {
        return;
    }
    // Don't push if content is identical to the last state
    const lastState = undoStack[undoStack.length - 1];
    if (lastState !== undefined && lastState === content) {
        return;
    }
    
    undoStack.push(content);
    redoStack = []; // Clear redo stack on a new action
    lastHistoryPush = now;

    // Limit stack size
    if (undoStack.length > 100) {
        undoStack.shift();
    }
  }

  function handleUndo() {
    if (undoStack.length > 1) { // Can't undo the initial state
        const currentState = undoStack.pop();
        if (currentState !== undefined) {
            redoStack.push(currentState);
        }
        const prevState = undoStack[undoStack.length - 1];
        dispatch('updatecontent', prevState);
    }
  }

  function handleRedo() {
    if (redoStack.length > 0) {
        const nextState = redoStack.pop();
        if (nextState !== undefined) {
            undoStack.push(nextState);
            dispatch('updatecontent', nextState);
        }
    }
  }

  // --- Event Handlers for Textarea ---




  // --- Autocomplete Functions ---
  function updateAutocompletePosition() {
    if (!markdownTextareaElement || autocompleteTriggerPos === null) return;

    const text = markdownTextareaElement.value;
    const pre = text.substring(0, autocompleteTriggerPos);
    
    // Create a temporary div to measure text position
    const div = document.createElement('div');
    div.style.position = 'absolute';
    div.style.visibility = 'hidden';
    div.style.whiteSpace = 'pre-wrap';
    div.style.wordWrap = 'break-word';
    
    // Copy relevant textarea styles
    const style = window.getComputedStyle(markdownTextareaElement);
    const styleProps: (keyof CSSStyleDeclaration)[] = ['font', 'lineHeight', 'padding', 'width', 'borderWidth', 'borderColor', 'borderStyle'];
    for (const prop of styleProps) {
        div.style[prop as any] = style[prop as any];
    }

    // Use a span to get the coordinates of the trigger position
    div.innerHTML = pre.replace(/\n/g, '<br>') + '<span id="caret"></span>';
    document.body.appendChild(div);
    
    const span = div.querySelector('#caret') as HTMLSpanElement;
    const rect = markdownTextareaElement.getBoundingClientRect();
    
    // Position relative to the textarea, accounting for scroll
    autocompleteX = rect.left + span.offsetLeft - markdownTextareaElement.scrollLeft;
    autocompleteY = rect.top + span.offsetTop - markdownTextareaElement.scrollTop + (parseFloat(style.lineHeight) || 20);

    document.body.removeChild(div);
  }

  function filterAutocompleteItems() {
      if (!autocompleteQuery) {
          autocompleteItems = codexEntries.slice(0, 10); // Show some initial entries
          return;
      }
      autocompleteItems = codexEntries.filter(e => 
          e.name.toLowerCase().includes(autocompleteQuery.toLowerCase())
      ).slice(0, 10);
  }

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
    pushToHistory(documentContent, true);
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
        dispatch('error', 'No chat model is selected. Please select a model in the settings.');
        return;
    }
    
    // Add user message to display immediately, unless it's a command that shouldn't be shown
    if (!overridePrompt && !userMessageToSend.startsWith('/')) {
        writeChatMessages = [...writeChatMessages, { sender: 'user', text: userMessageToSend }];
    } else if (userMessageOverride) {
        // For things like rephrase, show the "Rephrase selection" message
        writeChatMessages = [...writeChatMessages, { sender: 'user', text: userMessageOverride }];
    }

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
                writeChatError = "You must select text to use /summarize_selection.";
                isWriteChatLoading = false;
                dispatch('loading', false);
                return; 
            }
            finalPrompt = `System: Summarize the following selected text from the user's draft.\n<selected_text>\n${selection}\n</selected_text>\nUser: Summarize the selected text.`;
        } else if (userMessageToSend.startsWith('/rephrase_selection')) {
            const selection = getSelectedText();
            if (!selection) { 
                writeChatError = "You must select text to use /rephrase_selection.";
                isWriteChatLoading = false;
                dispatch('loading', false);
                return; 
            }
            finalPrompt = `System: Rephrase the following selected text from the user's draft. Aim for clarity and improved style.\n<selected_text>\n${selection}\n</selected_text>\nUser: Rephrase the selected text.`;
        } else if (userMessageToSend.startsWith('/continue_writing')) {
            const textBefore = getTextBeforeCursor();
            finalPrompt = `System: Continue writing from the current cursor position in the user's draft. Maintain the existing tone and style.\n<draft_context_before_cursor>\n${textBefore.slice(-1000)}\n</draft_context_before_cursor>\nUser: Continue writing.`;
        }
    }

    // The query MUST be a string. Use the slash-command prompt if it exists, otherwise use the user's input.
    const query = finalPrompt || userMessageToSend;

    try {
        const aiReply = await GetAIResponseWithContext(query, chatModelId);
        
        const markedResponse = await marked.parse(aiReply || '');

        writeChatMessages = [
            ...writeChatMessages, 
            { sender: 'ai', text: aiReply, html: markedResponse }
        ];

    } catch (err) {
        writeChatError = `AI error: ${err}`;
        console.error("Write chat send error:", err);
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
    pushToHistory(documentContent, true); // Force push for a discrete formatting action
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
  function handleDragStart(e: DragEvent, entry: database.CodexEntry) {
    e.dataTransfer?.setData('application/json', JSON.stringify({ type: 'codex-entry', entry }));
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
          writingWeaveSelectionEnd = cursorPos; // Set end to start for a drop (no selection)
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
    pushToHistory(documentContent, true);
    dispatch('updatecontent', documentContent.slice(0, position) + text + documentContent.slice(position));
  }

  // Helper to replace text within a specific range
  function replaceTextRange(newText: string, startPos: number, endPos: number) {
    pushToHistory(documentContent, true);
    dispatch('updatecontent', documentContent.slice(0, startPos) + newText + documentContent.slice(endPos));
  }

  // Computed property for filtered codex entries
  $: filteredCodexEntries = codexSearchTerm 
    ? codexEntries.filter(e => e.name.toLowerCase().includes(codexSearchTerm.toLowerCase()))
    : codexEntries;

  function getCursorPositionFromMouseEvent(event: MouseEvent): number {
    const textarea = event.target as HTMLTextAreaElement;
    const rect = textarea.getBoundingClientRect();
    const style = window.getComputedStyle(textarea);

    // --- 1. Get accurate coordinates and styles ---
    const x = event.clientX - rect.left - parseFloat(style.paddingLeft);
    const y = event.clientY - rect.top - parseFloat(style.paddingTop);
    const scrollTop = textarea.scrollTop;
    const text = textarea.value;

    // --- 2. Create a hidden div to mirror textarea styles to calculate position accurately ---
    const mirrorDiv = document.createElement('div');
    document.body.appendChild(mirrorDiv);

    // Copy styles that affect layout
    mirrorDiv.style.position = 'absolute';
    mirrorDiv.style.visibility = 'hidden';
    mirrorDiv.style.whiteSpace = 'pre-wrap';
    mirrorDiv.style.wordWrap = 'break-word';
    mirrorDiv.style.width = textarea.clientWidth + 'px';
    mirrorDiv.style.font = style.font;
    mirrorDiv.style.letterSpacing = style.letterSpacing;
    mirrorDiv.style.padding = style.padding;
    mirrorDiv.style.border = style.border;

    // --- 3. Find the character at the click position --- 
    let position = -1;
    // Use a sentinel character to ensure we can always find a range
    mirrorDiv.textContent = text + '|'; 
    const range = document.createRange();
    const textNode = mirrorDiv.childNodes[0];
    if (!textNode || !textNode.textContent) {
        document.body.removeChild(mirrorDiv);
        return text.length; // Fallback if textNode is not found
    }

    // Iterate through characters to find the one at the click coordinates
    for (let i = 0; i < textNode.textContent.length; i++) {
      range.setStart(textNode, i);
      range.setEnd(textNode, i + 1);
      const rangeRect = range.getBoundingClientRect();

      // Check if the click is within the vertical bounds of the current character's line
      // We compare against the mirrorDiv's rect, not the textarea's
      if (event.clientY >= rangeRect.top && event.clientY <= rangeRect.bottom) {
        // Check if the click is to the left of the character's midpoint
        if (event.clientX < rangeRect.left + rangeRect.width / 2) {
          position = i;
          break;
        }
      } else if (event.clientY < rangeRect.top) {
        // Click is on a previous line, so we've gone too far
        position = i > 0 ? i : 0; // Use current `i` as it's the start of the line
        break;
      }
    }
    
    if (position === -1) {
      position = text.length; // Clicked past the last character
    }

    document.body.removeChild(mirrorDiv);
    return position > text.length ? text.length : position;
  }

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
    // 1. Handle autocomplete key events first
    if (showAutocomplete) {
      autocompleteMenuRef.handleKeyDown(event);
      // Prevent further keydown processing if the menu is handling it
      if (['ArrowUp', 'ArrowDown', 'Enter', 'Tab', 'Escape'].includes(event.key)) {
        event.preventDefault();
        return; 
      }
    }

    // 2. Handle Undo/Redo and formatting
    if (event.ctrlKey || event.metaKey) {
        switch (event.key.toLowerCase()) {
            case 'z':
                event.preventDefault();
                if (event.shiftKey) {
                    handleRedo();
                } else {
                    handleUndo();
                }
                break;
            case 'y':
                event.preventDefault();
                handleRedo();
                break;
            case 'b':
                event.preventDefault();
                applyMarkdownFormat('bold');
                break;
            case 'i':
                event.preventDefault();
                applyMarkdownFormat('italic');
                break;
        }
    }
  }

  function handleInput(event: Event) {
    const target = event.target as HTMLTextAreaElement;
    pushToHistory(documentContent);
    dispatch('updatecontent', target.value);

    // Also handle autocomplete logic
    const text = target.value;
    const cursorPos = target.selectionStart;
    const lastAt = text.lastIndexOf('@', cursorPos - 1);

    if (lastAt !== -1) {
      const query = text.substring(lastAt + 1, cursorPos);
      // Basic check to avoid triggering on email-like patterns
      if (!query.includes(' ')) {
        showAutocomplete = true;
        autocompleteQuery = query;
        autocompleteTriggerPos = lastAt;
        updateAutocompletePosition();
        filterAutocompleteItems();
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
    const currentContent = documentContent;
    const cursorPos = markdownTextareaElement.selectionStart;

    // Replace the text from the '@' trigger to the current cursor position
    const textBefore = currentContent.substring(0, autocompleteTriggerPos);
    const textAfter = currentContent.substring(cursorPos);

    const newContent = textBefore + referenceText + textAfter;
    
    dispatch('updatecontent', newContent);
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

  async function handleWritingWeave(event: CustomEvent<{ selectedEntries: database.CodexEntry[], selectedLength: 'small' | 'medium' | 'large' | 'extra-large' }>) {
    const { selectedEntries, selectedLength } = event.detail;
    showCodexSelector = false;
    
    if (!activeWritingWeave || writingWeaveCursorPos === null || !markdownTextareaElement) return;

    isWeaving = true;
    dispatch('loading', true);
    
    try {
      // Use the saved selection end, as the live one is lost when the modal opens.
      const selectedText = documentContent.substring(writingWeaveCursorPos, writingWeaveSelectionEnd);
      const hasSelection = selectedText.length > 0;
      const textBeforeSelection = documentContent.substring(0, writingWeaveCursorPos);
      const textAfterSelection = documentContent.substring(writingWeaveSelectionEnd);

      const contextEntries = selectedEntries.map(entry => `${entry.name} (${entry.type}): ${entry.content}`).join('\n\n');
      
      // Convert length selection to prompt instruction
      const lengthInstructions: Record<string, string> = {
        'small': 'Keep your response to exactly 1 sentence that flows naturally.',
        'medium': 'Write approximately 1 paragraph (3-5 sentences) that develops the scene.',
        'large': 'Write approximately 1 page worth of content (multiple paragraphs, around 200-400 words).',
        'extra-large': 'Write approximately 2 pages worth of content (multiple paragraphs, around 400-800 words).'
      };
      
      const lengthInstruction = lengthInstructions[selectedLength] || lengthInstructions['medium'];
      
      // --- Conditional Prompting ---
      const taskDescription = hasSelection 
        ? `Your task is to enhance and weave a '${activeWritingWeave.label}' element into the selected text.`
        : `Your task is to generate and insert a '${activeWritingWeave.label}' element at the user's cursor position.`;

      const criticalInstruction = hasSelection
        ? `CRITICAL: Your response must be ONLY the enhanced text. Do not include any conversational pleasantries, introductions, or the original text. Your output will be directly inserted into a document.\n\nIMPORTANT: You must INCORPORATE the selected text into your response, not replace it. The selected text should be woven into and enhanced by your generated content, creating a richer, more detailed version that maintains the original meaning while adding the requested weave type.`
        : `CRITICAL: Your response must be ONLY the generated text to insert. Do not include any conversational pleasantries or introductions. Your output will be directly inserted into the document.`;

      const selectedTextContext = hasSelection 
        ? `\n\nSELECTED TEXT TO INCORPORATE:\n---\n${selectedText}\n---\n` 
        : '';
      
      const finalInstruction = hasSelection
        ? `Based on the weave type ('${activeWritingWeave.label}') and the provided context, generate enhanced text that incorporates and builds upon the selected text. The result should flow naturally from the before text, through your enhanced version of the selection, and into the after text. Match the specified length requirement.`
        : `Based on the weave type ('${activeWritingWeave.label}') and the provided context, generate new text to be inserted. The result should flow naturally from the text before the cursor and into the text after it. Match the specified length requirement.`;

      const prompt = `You are a subtle and masterful fiction writing assistant. ${taskDescription}\n\n${criticalInstruction}\n\nWhen incorporating the context entries, do so with nuance. Use them to inform the atmosphere, character voice, or narrative direction. The result should feel like a natural evolution of the original text.\n\nLENGTH REQUIREMENT: ${lengthInstruction}\n\nText before selection:\n---\n${textBeforeSelection.slice(-1500)}\n---${selectedTextContext}\n\nText after selection:\n---\n${textAfterSelection.substring(0, 1500)}\n---\n\nContext entries for inspiration:\n---\n${contextEntries || 'No specific context provided.'}\n---\n\n${finalInstruction}`;
      
      const generatedText = await GetAIResponseWithContext(prompt, chatModelId);
      
      // Replace the entire selection with the enhanced version (or insert if no selection)
      replaceTextRange(generatedText, writingWeaveCursorPos, writingWeaveSelectionEnd);

    } catch (err) {
      dispatch('error', `Writing Weaving failed: ${err}`);
    } finally {
      isWeaving = false;
      dispatch('loading', false);
      activeWritingWeave = null;
    }
  }

  function handleModalKeydown(event: KeyboardEvent) {
    if (showErrorModal && event.key === 'Escape') {
      showErrorModal = false;
    }
  }

  function openWritingWeave(event: MouseEvent, node: { type: string, label: string }) {
    event.stopPropagation();
    if (!markdownTextareaElement) return;
    
    const selectionStart = markdownTextareaElement.selectionStart;
    const selectionEnd = markdownTextareaElement.selectionEnd;
    
    // Check if there's a text selection
    if (selectionStart === selectionEnd) {
      // No selection - show the local error modal
      errorModalTitle = 'Text Selection Required';
      errorModalMessage = 'Please select some text in your document before clicking a writing weave. You can also drag the weave directly into your document at the desired location.';
      showErrorModal = true;
      return;
    }
    
    activeWritingWeave = node;
    writingWeaveCursorPos = selectionStart;
    writingWeaveSelectionEnd = selectionEnd; // Will be same as start if no selection
    showCodexSelector = true;
  }

  function handleCodexEntryClick(event: MouseEvent | KeyboardEvent, entry: database.CodexEntry) {
    event.stopPropagation();
    if (!markdownTextareaElement) return;
    
    const selectionStart = markdownTextareaElement.selectionStart;
    const selectionEnd = markdownTextareaElement.selectionEnd;

    // Check if there's a text selection
    if (selectionStart === selectionEnd) {
      // No selection - show the local error modal
      errorModalTitle = 'Text Selection Required';
      errorModalMessage = 'Please select some text in your document before clicking a codex entry. You can also drag the entry directly into your document to insert a reference.';
      showErrorModal = true;
      return;
    }
    
    // If text is selected, trigger weaving flow with this codex entry
    activeWritingWeave = { type: 'codex', label: entry.name };
    writingWeaveCursorPos = markdownTextareaElement.selectionStart;
    writingWeaveSelectionEnd = markdownTextareaElement.selectionEnd;
    showCodexSelector = true;
  }

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

  function handleMenuAction(event: CustomEvent, messageText: string) {
    const action = event.detail;
    activeMenuMessageIndex = null; // Close menu after action

    switch (action) {
      case 'insert':
        insertTextIntoDraft(messageText);
        break;
      case 'copy':
        navigator.clipboard.writeText(messageText).catch(err => {
          dispatch('error', `Failed to copy text: ${err}`);
        });
        break;
      case 'weave':
        handleWeaveFromChat(messageText);
        break;
      case 'index':
        saveChatToCodex(messageText);
        break;
    }
  }

  function toggleMenu(index: number) {
    if (activeMenuMessageIndex === index) {
      activeMenuMessageIndex = null;
    } else {
      activeMenuMessageIndex = index;
    }
  }

  async function handleWeaveFromChat(messageText: string) {
    const selection = getSelectedText();
    if (!selection) {
        errorModalTitle = 'Text Selection Required';
        errorModalMessage = 'Please select some text in your document to weave the AI response into.';
        showErrorModal = true;
        return;
    }

    isWeaving = true;
    dispatch('loading', true);

    const textBeforeSelection = documentContent.substring(0, markdownTextareaElement.selectionStart);
    const textAfterSelection = documentContent.substring(markdownTextareaElement.selectionEnd);

    const prompt = `You are a subtle and masterful fiction writing assistant. Your task is to weave the provided AI response into the user's selected text, enhancing it while maintaining the original tone and style.

CRITICAL: Your response must be ONLY the enhanced text. Do not include any conversational pleasantries, introductions, or the original text. Your output will be directly inserted into a document.

AI RESPONSE TO WEAVE:
---
${messageText}
---

SELECTED TEXT TO INCORPORATE:
---
${selection}
---

Text before selection:
---
${textBeforeSelection.slice(-1500)}
---

Text after selection:
---
${textAfterSelection.substring(0, 1500)}
---

Based on the AI response and the surrounding context, generate enhanced text that incorporates and builds upon the selected text. The result should flow naturally.`;

    try {
        const generatedText = await GetAIResponseWithContext(prompt, chatModelId);
        replaceTextRange(generatedText, markdownTextareaElement.selectionStart, markdownTextareaElement.selectionEnd);
    } catch (err) {
        dispatch('error', `Weaving from chat failed: ${err}`);
    } finally {
        isWeaving = false;
        dispatch('loading', false);
    }
  }
  async function saveChatToCodex(text: string) {
    if (!text) return;
    indexingStatus = 'indexing';
    indexingError = null;
    newIndexedEntries = [];
    updatedIndexedEntries = [];
    dispatch('savecodex', text);
  }

  export function setCodexSaveResult(result: { newEntries: database.CodexEntry[], updatedEntries: database.CodexEntry[] }) {
    indexingStatus = 'complete';
    indexingError = null;
    newIndexedEntries = result.newEntries || [];
    updatedIndexedEntries = result.updatedEntries || [];
  }

  export function setCodexSaveError(message: string) {
    indexingStatus = 'error';
    indexingError = message;
    newIndexedEntries = [];
    updatedIndexedEntries = [];
  }
</script>

<svelte:window on:keydown={handleModalKeydown} />

<!-- ... (rest of the HTML remains the same) -->

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
              <div class="message-text">
                {#if msg.sender === 'ai' && msg.html}
                  {@html msg.html}
                {:else}
                  {msg.text}
                {/if}
              </div>
              <button class="message-menu-btn" on:click|stopPropagation={() => toggleMenu(i)}>⋮</button>
              {#if activeMenuMessageIndex === i}
                <ChatMessageMenu on:action={(e) => handleMenuAction(e, msg.text)} />
              {/if}
            </div>
        {/each}
        {#if isWriteChatLoading}<div class="message ai loading">AI Thinking...</div>{/if}
         {#if writeChatMessages.length === 0 && !isWriteChatLoading}
           <div class="empty-chat">Ask the AI for ideas, rewrites, or feedback on your draft.</div>
         {/if}
      </div>
      <form on:submit|preventDefault={() => handleSendWriteChat()} class="write-chat-form">
        <input type="text" class="write-chat-input" bind:value={writeChatInput} placeholder="Ask AI..." disabled={isWriteChatLoading || !chatModelId} />
        <button type="submit" class="write-chat-send-btn" disabled={isWriteChatLoading || !writeChatInput.trim() || !chatModelId}>Send</button>
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
        <div class="status-bar">
          <span>{wordCount} words</span> | <span>{charCount} characters</span>
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
    <div class="editor-pane">
      {#if isWeaveDragOver}
      <div class="drop-indicator" style={dropIndicatorStyle}></div>
    {/if}
      <textarea
        class="markdown-input"
        value={documentContent}
        on:input={handleInput}
        bind:this={markdownTextareaElement}
        placeholder="Start writing your masterpiece (Markdown supported)..."
        class:hidden={editorMode === 'preview'}
        on:drop={handleDrop}
        on:dragenter={handleDragEnter}
        on:dragleave={handleDragLeave}
        on:dragover={handleDragOver}
        on:dragover|preventDefault
        on:keydown={handleWriteViewKeydown}
      ></textarea>
      <div 
        class="markdown-preview-container"
        class:hidden={editorMode === 'edit'}
      >
        <div class="markdown-preview">{@html renderedWriteHtml}</div>
      </div>
    </div>
    <div class="bottom-formatting-bar">
        <button class="tool-btn" on:click={() => applyMarkdownFormat('h1')} title="Header 1">H1</button>
        <button class="tool-btn" on:click={() => applyMarkdownFormat('h2')} title="Header 2">H2</button>
        <button class="tool-btn" on:click={() => applyMarkdownFormat('h3')} title="Header 3">H3</button>
        <button class="tool-btn" on:click={() => applyMarkdownFormat('bold')} title="Bold">B</button>
        <button class="tool-btn" on:click={() => applyMarkdownFormat('italic')} title="Italic">I</button>
        <button class="tool-btn" on:click={() => applyMarkdownFormat('code')} title="Code Block">{`<>`}</button>
    </div>
  </div>

  <!-- RIGHT COLUMN: Codex Reference & AI Tools -->
  <div class="right-column-toolbar">
    <!-- Codex Entries -->
    <div class="tool-section codex-reference-panel">
      <h4>Codex Reference</h4>
      <input 
        type="text" 
        placeholder="Search Codex..." 
        bind:value={codexSearchTerm} 
        class="codex-search-input"
      />
      <div class="codex-list">
        {#each filteredCodexEntries as entry (entry.id)}
          <div
            class="codex-item"
            role="button" 
            tabindex="0"
            draggable="true"
            on:dragstart={(e) => handleDragStart(e, entry)}
            on:click={(e) => handleCodexEntryClick(e, entry)}
            on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') handleCodexEntryClick(e, entry); }}
          >
            {entry.name} <span>({entry.type})</span>
          </div>
        {/each}
      </div>
    </div>

    <!-- Writing Weaving -->
    <div class="tool-section">
      <h4>Writing Weaving</h4>
      <div class="writing-weave-buttons">
        {#each writingWeaves as weave}
          <button 
            class="tool-btn writing-weave-btn" 
            title={weave.description}
            draggable="true"
            on:dragstart={(e) => handleWeaveButtonDragStart(e, weave)}
            on:click={(e) => openWritingWeave(e, weave)}
          >
            <span class="icon">{weave.icon}</span> {weave.label}
          </button>
        {/each}
      </div>
    </div>

    <!-- AI Actions -->
    <div class="tool-section">
      <h4>AI Actions</h4>
      <div class="tool-buttons-grid">
        <button class="tool-btn" on:click={() => handleToolAction('summarize')} title="Summarize selected text">
          <span class="icon">📄</span> Summarize
        </button>
        <button class="tool-btn" on:click={() => handleToolAction('rephrase')} title="Rephrase selected text">
          <span class="icon">✍️</span> Rephrase
        </button>
        <button class="tool-btn" on:click={() => handleToolAction('continue')} title="Continue writing from cursor">
          <span class="icon">➡️</span> Continue
        </button>
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


<!-- Add the new modal -->
{#if showSaveTemplateModal}
  <div class="modal-backdrop" role="button" tabindex="0" on:click={() => showSaveTemplateModal = false} on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') showSaveTemplateModal = false; }}>
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

    <!-- Error Modal -->
    {#if showErrorModal}
      <div class="modal-backdrop">
        <div class="modal error-modal" role="dialog" aria-modal="true">
          <h3>{errorModalTitle}</h3>
          <p>{errorModalMessage}</p>
          <div class="modal-buttons">
            <button on:click={() => showErrorModal = false} class="save-btn">OK</button>
          </div>
        </div>
      </div>
    {/if}
    
    <!-- Indexing Modal -->
    {#if indexingStatus !== 'idle'}
      <div class="modal-backdrop">
        <div class="modal indexing-modal">
            <StoryImportStatus
                status={indexingStatus === 'indexing' ? 'sending' : indexingStatus}
                errorMsg={indexingError}
                newEntries={newIndexedEntries}
                updatedEntries={updatedIndexedEntries}
            />
            {#if indexingStatus === 'complete' || indexingStatus === 'error'}
                <div class="modal-buttons" style="margin-top: 1rem; justify-content: center;">
                    <button on:click={() => { indexingStatus = 'idle'; }} class="save-btn">OK</button>
                </div>
            {/if}
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

<!-- Save Modal -->
{#if showWriteSaveModal}
  <div class="modal-backdrop" role="button" tabindex="0" on:click={() => showWriteSaveModal = false} on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') showWriteSaveModal = false; }}>
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
