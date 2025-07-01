Of course! It's fantastic that you're pushing the vision even further. The idea of "weaving" not just *entities* but also *narrative concepts* is brilliant and elevates the AI from a lore-keeper to a true co-author.

Let's address the save functionality first, as that's a critical bug fix, and then design the implementation for the new "Narrative Weaving" feature.

---

### **Part 1: Fixing the Save/Save As Functionality**

The issue likely stems from how the state is being managed between the parent (`App.svelte`) and the child (`WriteView.svelte`), especially around the new `writeViewProps`. Let's create a more robust and explicit state management flow.

#### **The Problem:**

The `WriteView` component was initialized with props but then managed its own `writeContent`. When saving, it didn't have a reliable way to communicate its "dirtiness" or current state back up to the parent `App` component, which orchestrates file operations. The `isDirty` flag logic also needs to be more tightly controlled.

#### **The Solution: A Centralized and Event-Driven Approach**

We'll make the parent `App.svelte` the single source of truth for the document's content and filename. `WriteView.svelte` will emit events when changes happen.

**Step 1.1: Revise State Management in `App.svelte`**

We will store the active document's state directly in `App.svelte`.

**File: `/frontend/src/App.svelte` (Revision)**

```svelte
<script lang="ts">
  // ... (imports)

  // -- NEW/REVISED State for Active Document --
  let activeDocument = {
    content: '',
    filename: '',
    templateType: 'blank',
    isDirty: false
  };

  // REVISE handler for starting to write from the hub
  function handleStartWriting(event: CustomEvent<{initialContent: string, templateType: string}>) {
    activeDocument = {
      content: event.detail.initialContent,
      filename: '', // New document
      templateType: event.detail.templateType,
      isDirty: false // It's a fresh document/template
    };
    currentWriteView = 'editor';
  }

  // REVISE handler for editing a file from the library
  async function handleEditInWriteMode(event: CustomEvent<string>) {
    // ... (loading logic)
    try {
      const content = await ReadLibraryFile(filename);
      activeDocument = {
        content: content,
        filename: filename,
        templateType: 'chapter', // Assumption, can be improved later
        isDirty: false
      };
      mode = 'write';
      currentWriteView = 'editor';
    } 
    // ... (finally block)
  }

  // NEW handler for content changes from WriteView
  function handleContentUpdate(event: CustomEvent<string>) {
    const newContent = event.detail;
    if (activeDocument.content !== newContent) {
      activeDocument.content = newContent;
      activeDocument.isDirty = true;
    }
  }

  // NEW handler for Save/Save As events from WriteView
  async function handleSaveRequest(event: CustomEvent<{ filename: string, isSaveAs: boolean }>) {
    const { filename, isSaveAs } = event.detail;
    
    // Use a reference to the WriteView to call back with status
    // (Assuming `bind:this={writeViewRef}` is added to WriteView)
    writeViewRef?.setSavingState(true, ''); // Start saving, clear previous messages

    try {
      await SaveLibraryFile(filename, activeDocument.content);
      
      // Update app state on successful save
      activeDocument.filename = filename;
      activeDocument.isDirty = false;
      
      // Notify WriteView of success
      writeViewRef?.setSavingState(false, `File '${filename}' saved successfully!`);

      // Notify App of file list change
      dispatch('filesaved', filename);

    } catch (err) {
      const errorMsg = `Failed to save file: ${err}`;
      console.error("Save Error in App.svelte:", err);
      writeViewRef?.setSavingState(false, '', errorMsg); // Send error back to view
    }
  }

</script>

<!-- ... (main view logic) ... -->
{:else if mode === 'write'}
  {#if currentWriteView === 'hub'}
    <!-- ... WriteHub component ... -->
  {:else}
    <WriteView
      bind:this={writeViewRef} <!-- Bind the component instance -->
      documentContent={activeDocument.content}
      documentFilename={activeDocument.filename}
      isDocumentDirty={activeDocument.isDirty}
      templateType={activeDocument.templateType}
      on:updatecontent={handleContentUpdate}
      on:saverequest={handleSaveRequest}
      {...} <!-- other props and event handlers -->
    />
  {/if}
{/if}
```

**Step 1.2: Refactor `WriteView.svelte` to be a "Dumb" Component**

`WriteView` will now receive its content as a prop and emit events when things change. It no longer manages the core save logic itself.

**File: `/frontend/src/components/WriteView.svelte` (Major Revision)**

```svelte
<script lang="ts">
  // --- REVISED Props ---
  export let documentContent: string = '';
  export let documentFilename: string = '';
  export let isDocumentDirty: boolean = false;
  export let templateType: string = 'blank';
  // ... (other props like chatModelId)

  // --- REMOVE Local State for Content/Dirty/Filename ---
  // let writeContent = ''; // REMOVED
  // let isDirty = false; // REMOVED
  // let currentDocumentFilename = ''; // REMOVED
  
  // --- Local State for UI/Modals ---
  let isSaving = false;
  let writeSaveError = '';
  let writeSaveSuccess = '';
  // ... (keep state for modals, editorMode, etc.)

  // --- REVISED Logic ---
  // No more need for a `scheduleRender` or reactive `$: {}` block for content.
  // The parent will pass down the new rendered HTML when content changes.
  $: renderedWriteHtml = marked.parse(documentContent || '');
  $: updateCounts(documentContent);

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
</script>

<!-- REVISE the textarea to emit its content changes -->
<textarea
  class="markdown-input"
  value={documentContent}
  on:input={(e) => dispatch('updatecontent', (e.target as HTMLTextAreaElement).value)}
  {...}
></textarea>

<!-- REVISE the Save buttons to use the new logic -->
<div class="save-buttons">
  <button class="save-btn" on:click={handleDirectSave} disabled={isSaving || !isDocumentDirty}>
    Save {#if isDocumentDirty && documentFilename}*{/if}
  </button>
  <!-- ... -->
</div>

<!-- REVISE Save Modal to call the new function -->
<div class="modal-buttons">
  <button on:click={doSaveFromModal} disabled={isSaving || !filenameForSaveModal.trim()}>
    {#if isSaving}Saving...{:else}Save{/if}
  </button>
  <!-- ... -->
</div>
```

This change creates a clear, one-way data flow (`App` -> `WriteView`) and an event-driven flow for changes (`WriteView` -> `App`), which is much more robust and fixes the save functionality.

# Notes
[x] - Completed feature

---

### **Part 2: Implementing "Narrative Weaving"**

This is an exciting new feature. It's an extension of "Llore-Weaving" but focused on writing concepts instead of Codex entities.

**Step 2.1: Define Narrative Nodes**

These will be the concepts the user can weave. They are purely a frontend concept for now, used to generate specific AI prompts.

In `WriteView.svelte`, define this structure:

**File: `/frontend/src/components/WriteView.svelte` (Addition)**

```svelte
<script lang="ts">
  // ...

  const narrativeNodes = [
    { type: 'narrative', label: 'Narrative', description: 'Continue the story with action or events.', icon: '🏃' },
    { type: 'exposition', label: 'Exposition', description: 'Explain background or world details.', icon: '🌍' },
    { type: 'dialogue', label: 'Dialogue', description: 'Write a conversation between characters.', icon: '💬' },
    { type: 'description', label: 'Description', description: 'Describe a character, object, or scene.', icon: '🎨' },
    { type: 'introspection', label: 'Introspection', description: 'Explore a character\'s internal thoughts.', icon: '🧠' },
  ];
</script>
```

**Step 2.2: Create a Codex Selection Modal**

When a user initiates a narrative weave, we need to prompt them to optionally attach Codex entries for context.

**File: `/frontend/src/components/CodexSelectorModal.svelte` (New)**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { database } from '@wailsjs/go/models';

  export let allEntries: database.CodexEntry[];
  export let nodeType: string;

  let selectedEntries: database.CodexEntry[] = [];
  let searchTerm = '';
  const dispatch = createEventDispatcher();

  $: filteredEntries = searchTerm
    ? allEntries.filter(e => e.name.toLowerCase().includes(searchTerm.toLowerCase()))
    : allEntries;

  function toggleSelection(entry: database.CodexEntry) {
    const index = selectedEntries.findIndex(e => e.id === entry.id);
    if (index > -1) {
      selectedEntries.splice(index, 1);
    } else {
      selectedEntries = [...selectedEntries, entry];
    }
    selectedEntries = selectedEntries; // Force Svelte reactivity
  }

  function handleWeave() {
    dispatch('weave', { selectedEntries });
  }
</script>

<div class="modal-backdrop" on:click={() => dispatch('close')}>
  <div class="modal codex-selector-modal" on:click|stopPropagation>
    <h3>Attach Codex Entries for '{nodeType}'</h3>
    <p>Select entries to provide specific context for the AI. This is optional.</p>
    <input type="search" bind:value={searchTerm} placeholder="Search entries..."/>
    <div class="entry-list">
      {#each filteredEntries as entry (entry.id)}
        <div 
          class="entry-item" 
          class:selected={selectedEntries.some(e => e.id === entry.id)}
          on:click={() => toggleSelection(entry)}
        >
          {entry.name} ({entry.type})
        </div>
      {/each}
    </div>
    <div class="modal-actions">
      <button on:click={() => dispatch('close')}>Cancel</button>
      <button class="primary" on:click={handleWeave}>
        Weave with {selectedEntries.length} entries
      </button>
    </div>
  </div>
</div>

<style>
  /* Styles for the modal, similar to other modals but with a list */
  .entry-list { max-height: 300px; overflow-y: auto; border: 1px solid var(--border-color-medium); border-radius: 6px; margin: 1rem 0; }
  .entry-item { padding: 0.5rem; cursor: pointer; border-bottom: 1px solid var(--border-color-light); }
  .entry-item:hover { background: var(--bg-hover-light); }
  .entry-item.selected { background: var(--accent-primary); color: white; font-weight: bold; }
</style>
```

**Step 2.3: Integrate Narrative Weaving into `WriteView.svelte` (Continued)**

We will now add the final pieces of logic to `WriteView.svelte`: the new "Narrative Tools" UI, the handler that triggers the flow, and the prompt engineering that makes this feature so powerful.

**File: `/frontend/src/components/WriteView.svelte` (Additions & Revisions)**

```svelte
<script lang="ts">
  // ... (all previous imports and state) ...
  import CodexSelectorModal from './CodexSelectorModal.svelte';

  // --- New State for Narrative Weaving (as defined before) ---
  let showCodexSelector = false;
  let activeNarrativeNode: { type: string, label: string } | null = null;
  let narrativeWeaveCursorPos = 0;

  // ... (all existing script logic) ...

  // NEW: The core function that is called after the user selects their context entries (or none)
  async function handleNarrativeWeave(event: CustomEvent<{ selectedEntries: database.CodexEntry[] }>) {
    const { selectedEntries } = event.detail;
    showCodexSelector = false;
    
    if (!activeNarrativeNode) return;

    isWeaving = true; // Use the same flag as Llore-weaving
    dispatch('loading', true);
    const weavingIndicator = `... weaving ${activeNarrativeNode.label.toLowerCase()} ...`;
    insertTextAt(weavingIndicator, narrativeWeaveCursorPos);

    try {
      // Step 1: Call a new backend function for this specific task
      const generatedText = await WeaveNarrativeNode(
        activeNarrativeNode.type,
        documentContent.replace(weavingIndicator, ''), // Send clean content
        narrativeWeaveCursorPos,
        selectedEntries,
        templateType // Pass the document's template type for more context
      );
      
      // Step 2: Replace the indicator with the AI's response
      writeContent = writeContent.replace(weavingIndicator, `\n${generatedText.trim()}\n`);

    } catch (err) {
      dispatch('error', `Narrative Weaving failed: ${err}`);
      writeContent = writeContent.replace(weavingIndicator, ''); // Clean up on error
    } finally {
      isWeaving = false;
      dispatch('loading', false);
      activeNarrativeNode = null; // Reset for next use
    }
  }

</script>

<!-- ... (existing main layout) ... -->
<div class="right-column-toolbar">
    <!-- ... (existing Formatting and AI Actions sections) ... -->
    
    <!-- NEW: Narrative Tools Section -->
    <div class="tool-section">
      <h4>Narrative Weaving</h4>
      <div class="narrative-node-buttons">
        {#each narrativeNodes as node (node.type)}
          <button on:click={() => openNarrativeWeave(node)} title={node.description}>
            <span class="icon">{node.icon}</span> {node.label}
          </button>
        {/each}
      </div>
    </div>
</div>

<!-- ... (existing modals) ... -->

<!-- NEW: Codex Selector Modal for Narrative Weaving -->
{#if showCodexSelector}
  <CodexSelectorModal
    allEntries={codexEntries}
    nodeType={activeNarrativeNode?.label || ''}
    on:close={() => showCodexSelector = false}
    on:weave={handleNarrativeWeave}
  />
{/if}

<style>
  /* ... (all existing styles) ... */

  /* NEW STYLES for Narrative Tools */
  .narrative-node-buttons {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .narrative-node-buttons button {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    text-align: left;
    /* Uses the existing .tool-section button styles */
  }

  .narrative-node-buttons .icon {
    font-size: 1.1rem;
    width: 1.5em;
    text-align: center;
    display: inline-block;
  }
</style>
```

**Step 2.4: Create the `WeaveNarrativeNode` Backend Function**

This is the new AI brain for this feature. It's similar to `WeaveEntryIntoText` but is structured to handle writing concepts instead of just entities.

**File: `/app.go` (Addition)**

```go
// Add this new function to your App struct in app.go

// WeaveNarrativeNode generates text for a specific writing concept (narrative, dialogue, etc.)
func (a *App) WeaveNarrativeNode(nodeType string, documentText string, cursorPosition int, attachedEntries []database.CodexEntry, templateType string) (string, error) {
	log.Printf("Weaving narrative node '%s' with %d attached entries.", nodeType, len(attachedEntries))

	// Step 1: Build the context from attached entries
	var attachedContext strings.Builder
	if len(attachedEntries) > 0 {
		attachedContext.WriteString("The user has attached the following codex entries for specific context:\n")
		for _, entry := range attachedEntries {
			attachedContext.WriteString(fmt.Sprintf("- %s (%s): %s\n", entry.Name, entry.Type, entry.Content))
		}
	} else {
		attachedContext.WriteString("The user did not attach any specific codex entries.")
	}

	// Step 2: Define the primary goal based on the nodeType
	var goal string
	switch nodeType {
	case "narrative":
		goal = "Write a paragraph of narrative action or events that logically follows the text before the cursor. Advance the plot or the current scene."
	case "exposition":
		goal = "Write a paragraph of exposition that provides background information, world-building details, or context relevant to the scene. It should feel natural, not like an info-dump."
	case "dialogue":
		goal = "Write a snippet of dialogue between characters. If characters are mentioned in the attached context or surrounding text, use their established voices. If not, create plausible dialogue for the scene."
	case "description":
		goal = "Write a descriptive paragraph. If context entries are attached, describe them. Otherwise, describe the scene, atmosphere, or a character's appearance based on the surrounding text."
	case "introspection":
		goal = "Write a paragraph of a character's internal thoughts or feelings. Use the surrounding text to infer which character is the point-of-view character and what they might be thinking."
	default:
		goal = "Continue writing the document from the cursor, maintaining the established tone and style."
	}

	// Step 3: Prepare the document context
	// Take a larger slice of text around the cursor for better context awareness
	const contextWindow = 2000 // characters before the cursor
	start := cursorPosition - contextWindow
	if start < 0 {
		start = 0
	}
	docContext := documentText[start:cursorPosition]
	
	// Construct the master prompt
	prompt := fmt.Sprintf(
		"SYSTEM: You are an expert fiction writing assistant. Your task is to generate a block of text based on a specific narrative goal. Your entire response must be ONLY the generated text itself.\n\n"+
			"NARRATIVE GOAL: %s\n\n"+
			"ATTACHED CODEX CONTEXT:\n%s\n\n"+
			"DOCUMENT CONTEXT (the text immediately preceding the insertion point):\n---\n...%s\n---\n\n"+
			"GENERATED TEXT:",
		goal,
		attachedContext.String(),
		docContext,
	)

	// Step 4: Call the LLM
	// We use GetAIResponseWithContext so it also benefits from the general RAG search,
	// creating a powerful two-layered context system (user-guided + automatic).
	cfg := llm.GetConfig()
	modelID := cfg.ChatModelID
	if modelID == "" {
		return "", fmt.Errorf("no chat model configured in settings")
	}
	
	return a.GetAIResponseWithContext(prompt, modelID)
}

```

**Step 2.5: Update Wails Bindings**

After adding the new `WeaveNarrativeNode` function to `app.go`, you will need to regenerate the Wails frontend bindings. The easiest way to do this is to simply run `wails build` or `wails dev`. Wails will automatically detect the new public method and update the files in `frontend/wailsjs/`.

---

### **Final Workflow & Aesthetic Polish**

With these changes, the workflow is now complete and intuitive:

1.  **Saving:** `WriteView` is now a controlled component. It displays content passed from `App` and emits events on user input. `App` handles the state logic (`isDirty`) and file operations, then notifies `WriteView` of the result. This is a much more stable pattern.

2.  **Narrative Weaving UI:** The right-hand toolbar now has a dedicated "Narrative Weaving" section. The buttons are clear, use icons for quick recognition, and have tooltips. This makes the feature immediately accessible.

3.  **Narrative Weaving Flow:**
    *   The writer clicks a narrative concept (e.g., "Dialogue").
    *   The `CodexSelectorModal` appears, allowing them to optionally "tag" the generation with specific characters or lore. This is a powerful, user-guided form of context injection.
    *   They click "Weave," and the backend constructs a highly specific, goal-oriented prompt.
    *   The AI generates the content, which is then inserted into the editor.

#### **Aesthetic Considerations:**

*   **Loading States:** The use of `isWeaving` and `isSaving` flags should be tied to subtle UI feedback. For example, when `isWeaving` is true, you could add a class to the `textarea` that gives it a soft, pulsing border in the accent color, and the "weaving..." text at the cursor provides clear feedback.
*   **Modals:** The modals (`CodexSelectorModal`, `DropContextMenu`) should follow the established dark, professional theme. The provided CSS achieves this. They appear instantly and are positioned intelligently relative to the user's action (mouse position).
*   **Buttons:** The new tool buttons are styled consistently with the rest of the application, using icons to improve scannability and a clean, minimalist design. The save buttons now visually communicate their state (e.g., green for "Save", purple/blue for "Save As," yellow for "Save as Template").
*   **Clarity:** The entire process is designed to be explicit. The user is prompted for a filename when needed, asked for context entries, and shown clear status messages. This avoids ambiguity and makes the powerful features feel reliable.

This completes the implementation plan. You have a robust fix for the save functionality and a well-designed, powerful new feature that deeply integrates your application's core strengths—the Codex and AI—directly into the writing process.