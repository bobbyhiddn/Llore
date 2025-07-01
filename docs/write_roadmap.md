Of course. I'm excited too! This is where Llore can make a leap from a great tool to an essential one. Let's build a comprehensive, step-by-step implementation plan. We'll go from the backend foundations to the frontend UI, paying close attention to the workflow and aesthetic you've envisioned.

Here is your feature-complete implementation plan for the new, template-driven, Llore-weaving `Write` mode.

---

### **Phase 1: Backend Foundations & Vault Structure**

First, we need to add the necessary backend functions and update the vault structure.

#### 1.1. Update Vault Structure

In `internal/vault/vault.go`, we'll add a `Templates` directory to every vault.

**File: `/internal/vault/vault.go` (Revision)**

```go
// ... (keep existing code)

// CreateNewVault creates a new vault folder with the required structure.
func CreateNewVault(ctx context.Context, vaultName string) (string, error) {
	// ... (keep existing selection logic)

	if vaultName == "" {
		vaultName = "LoreVault"
	}
	vaultPath := filepath.Join(selection, vaultName)
	// ADD "Templates" to the list of subdirectories
	subdirs := []string{
		filepath.Join(vaultPath, "Library"),
		filepath.Join(vaultPath, "Codex"),
		filepath.Join(vaultPath, "Chat"),
		filepath.Join(vaultPath, "Templates"), // New line
	}

	// ... (rest of the function remains the same)
	log.Printf("Created new vault at: %s with Templates directory", vaultPath)
	return vaultPath, nil
}

// ... (keep existing code)
```
*(Also ensure `SwitchVault` checks for this new directory for robustness).*

# Notes
[x] - Completed feature

#### 1.2. Add New Backend Functions

In `/app.go`, we will add three new methods to be exposed to the frontend.

**File: `/app.go` (Additions)**

```go
// Add these new functions to your App struct in app.go

// ListTemplates returns a list of .md files in the vault's Templates folder
func (a *App) ListTemplates() ([]string, error) {
	if a.db == nil {
		return nil, fmt.Errorf("no vault is currently loaded")
	}
	templatesPath := filepath.Join(a.dbPath, "Templates")
	entries, err := os.ReadDir(templatesPath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Templates directory does not exist, creating it: %s", templatesPath)
			if err := os.MkdirAll(templatesPath, 0755); err != nil {
				return nil, fmt.Errorf("failed to create Templates directory: %w", err)
			}
			return []string{}, nil // Return empty list after creating
		}
		return nil, fmt.Errorf("failed to read Templates directory: %w", err)
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}

// SaveTemplate writes content to a specified file in the vault's Templates folder
func (a *App) SaveTemplate(filename string, content string) error {
	if a.db == nil {
		return fmt.Errorf("no vault is currently loaded")
	}
	// Basic validation
	if strings.Contains(filename, "..") || strings.ContainsRune(filename, filepath.Separator) {
		return fmt.Errorf("invalid template filename")
	}
	if !strings.HasSuffix(filename, ".md") {
		filename += ".md"
	}

	filePath := filepath.Join(a.dbPath, "Templates", filename)
	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("failed to write template file %s: %w", filename, err)
	}
	log.Printf("Successfully saved template file: %s", filePath)
	return nil
}


// WeaveEntryIntoText is the core "Llore-weaving" function.
func (a *App) WeaveEntryIntoText(droppedEntry database.CodexEntry, documentText string, cursorPosition int, templateType string) (string, error) {
	log.Printf("Weaving entry '%s' into a '%s' document.", droppedEntry.Name, templateType)

	var goal string
	// Determine the AI's goal based on context
	switch templateType {
	case "character-sheet":
		if droppedEntry.Type == "Character" {
			goal = fmt.Sprintf("The user dropped Character '%s' onto this character sheet. Generate a new 'Relationships' section describing a plausible connection (e.g., friend, family, rival, mentor) between the sheet's character and the dropped character.", droppedEntry.Name)
		} else { // Handle Artifact, Location, etc.
			goal = fmt.Sprintf("The user dropped the %s '%s' onto this character sheet. Generate a new section describing how the character acquired, uses, or is connected to this %s.", droppedEntry.Type, droppedEntry.Name, droppedEntry.Type)
		}
	case "chapter":
		goal = fmt.Sprintf("The user dropped the %s '%s' into this narrative scene. Weave its introduction or a mention of it naturally into the story at the cursor position. It could be a character noticing it, interacting with it, or thinking about it.", droppedEntry.Type, droppedEntry.Name)
	default: // Generic fallback for blank documents or unknown templates
		goal = fmt.Sprintf("The user dropped the %s '%s' into their document. Based on the surrounding text, intelligently integrate this information. This could be a new descriptive sentence, a new paragraph, or an expansion of an existing idea.", droppedEntry.Type, droppedEntry.Name)
	}

	// Prepare the document with a cursor marker
	if cursorPosition > len(documentText) {
		cursorPosition = len(documentText)
	}
	docWithCursor := documentText[:cursorPosition] + "<<CURSOR>>" + documentText[cursorPosition:]

	// Construct the master prompt
	prompt := fmt.Sprintf(
		"SYSTEM: You are an expert fiction writing assistant. Your task is to seamlessly weave a new codex entry into an existing draft. Your response must be ONLY the text to be inserted. Do not include explanations.\n\n"+
			"GOAL: %s\n\n"+
			"DROPPED ENTRY DETAILS:\n- Name: %s\n- Type: %s\n- Content: %s\n\n"+
			"DOCUMENT CONTEXT (with cursor position):\n---\n%s\n---\n\n"+
			"GENERATED TEXT TO INSERT:",
		goal,
		droppedEntry.Name,
		droppedEntry.Type,
		droppedEntry.Content,
		docWithCursor,
	)

	// Use the existing RAG function to get the completion
	cfg := llm.GetConfig()
	modelID := cfg.ChatModelID // Or a more powerful model if desired for this task
	if modelID == "" {
		return "", fmt.Errorf("no chat model configured in settings")
	}
	
	return a.GetAIResponseWithContext(prompt, modelID)
}

```

# Notes
[x] - Completed feature

---

### **Phase 2: Frontend "Writing Hub"**

We'll create a new "hub" to act as the entry point for `Write` mode.

#### 2.1. Create `WriteHub.svelte`

This is a new file.

**File: `/frontend/src/components/WriteHub.svelte` (New)**

```svelte
<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { ListTemplates } from '@wailsjs/go/main/App';

  const dispatch = createEventDispatcher();

  let customTemplates: string[] = [];
  let isLoading = true;

  onMount(async () => {
    try {
      customTemplates = await ListTemplates() || [];
    } catch (err) {
      console.error("Failed to load templates:", err);
      // You could dispatch an error event here
    } finally {
      isLoading = false;
    }
  });
  
  // Hardcoded built-in templates
  const builtInTemplates = [
    { name: 'Chapter', description: 'A standard prose template for writing scenes and chapters.', content: '# Chapter Title\n\n', type: 'chapter' },
    { name: 'Character Sheet', description: 'A detailed profile for a character.', content: '# Character: \n\n## Physical Description\n\n## Personality\n\n## Backstory\n\n## Relationships\n', type: 'character-sheet' },
    { name: 'Arc Outline', description: 'A three-act structure for plotting.', content: '# Arc: \n\n## Act I: The Setup\n\n## Act II: The Confrontation\n\n## Act III: The Resolution\n', type: 'arc-outline' }
  ];

  function startWriting(content: string, templateType: string) {
    dispatch('startwriting', { initialContent: content, templateType });
  }
</script>

<div class="write-hub-container">
  <h2>Start a New Document</h2>
  <div class="options-grid">
    <!-- Blank Document Card -->
    <button class="option-card" on:click={() => startWriting('', 'blank')}>
      <div class="icon">📄</div>
      <div class="title">Blank Document</div>
      <div class="description">Start with a clean slate.</div>
    </button>
    
    <!-- Built-in Template Cards -->
    {#each builtInTemplates as template}
      <button class="option-card" on:click={() => startWriting(template.content, template.type)}>
        <div class="icon">📜</div>
        <div class="title">{template.name}</div>
        <div class="description">{template.description}</div>
      </button>
    {/each}
  </div>

  {#if isLoading}
    <p>Loading custom templates...</p>
  {:else if customTemplates.length > 0}
    <h3 class="custom-templates-header">Your Templates</h3>
    <div class="custom-templates-list">
      {#each customTemplates as templateFile}
        <!-- In a real app, you'd fetch content on click -->
        <button class="custom-template-item" on:click={() => dispatch('loadtemplate', templateFile)}>
          {templateFile.replace('.md', '')}
        </button>
      {/each}
    </div>
  {/if}
</div>

<style>
  .write-hub-container { padding: 2rem; text-align: center; }
  h2 { color: var(--text-primary); margin-bottom: 2rem; }
  .options-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 1.5rem;
    max-width: 800px;
    margin: 0 auto;
  }
  .option-card {
    background: var(--bg-secondary);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 12px;
    padding: 1.5rem;
    text-align: left;
    transition: all 0.3s ease;
    cursor: pointer;
  }
  .option-card:hover {
    transform: translateY(-5px);
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.2);
    border-color: var(--accent-primary);
  }
  .icon { font-size: 2rem; margin-bottom: 1rem; }
  .title { font-size: 1.1rem; font-weight: 600; color: var(--text-primary); }
  .description { font-size: 0.9rem; color: var(--text-secondary); }
  .custom-templates-header { margin-top: 3rem; border-top: 1px solid rgba(255,255,255,0.1); padding-top: 2rem; }
</style>
```

# Notes
[x] - Completed feature

#### 2.2. Update `App.svelte` to use the Hub

**File: `/frontend/src/App.svelte` (Revision)**

```svelte
<script lang="ts">
  // ... imports
  import WriteHub from './components/WriteHub.svelte'; // Import the new hub
  import { ReadLibraryFile } from '@wailsjs/go/main/App'; // Make sure this is imported

  // ... other state variables

  // NEW state to manage the Write mode's view
  let currentWriteView: 'hub' | 'editor' = 'hub';
  let writeViewProps = {
    initialContent: '',
    initialFilename: '',
    templateType: 'blank'
  };

  // REVISE handleEditInWriteMode
  async function handleEditInWriteMode(event: CustomEvent<string>) {
    // ... (existing loading/error logic)
    try {
      const content = await ReadLibraryFile(filename);
      // Set props for the editor view
      writeViewProps = {
        initialContent: content,
        initialFilename: filename,
        templateType: 'chapter' // Assume library files are chapters, or determine from metadata later
      };
      // Set mode to 'write' and view to 'editor'
      mode = 'write';
      currentWriteView = 'editor';
    } 
    // ... (finally block)
  }
  
  // NEW handler for starting to write from the hub
  function handleStartWriting(event: CustomEvent<{initialContent: string, templateType: string}>) {
    writeViewProps = {
      initialContent: event.detail.initialContent,
      initialFilename: '', // It's a new document
      templateType: event.detail.templateType
    };
    currentWriteView = 'editor';
  }

  // NEW handler for loading a custom template
  async function handleLoadCustomTemplate(event: CustomEvent<string>) {
    const filename = event.detail;
    // We need a new backend function to read a template file, let's assume it exists for now
    // For simplicity, we can reuse ReadLibraryFile but point it to the Templates dir
    // Let's create `ReadTemplateFile` in backend. For now, we'll imagine it.
    // --> We will just reuse `ReadLibraryFile` but need to make the backend aware.
    // For now, let's assume `ReadLibraryFile` can handle `Templates/file.md`
    try {
      const content = await ReadLibraryFile(`../Templates/${filename}`); // A bit of a hack, better to have a dedicated func
      writeViewProps = {
        initialContent: content,
        initialFilename: '',
        templateType: filename.replace('.md', '') // Use filename as template type
      };
      currentWriteView = 'editor';
    } catch (err) {
      // dispatch global error
    }
  }

  // REVISE the main display logic
</script>

<!-- ... -->
{:else if mode === 'write'}
  {#if currentWriteView === 'hub'}
    <WriteHub 
      on:startwriting={handleStartWriting}
      on:loadtemplate={handleLoadCustomTemplate}
      on:back={() => { mode = null; currentWriteView = 'hub'; }}
    />
  {:else}
    <WriteView
      initialContent={writeViewProps.initialContent}
      initialFilename={writeViewProps.initialFilename}
      templateType={writeViewProps.templateType}
      chatModelId={chatModelId}
      on:back={() => { currentWriteView = 'hub'; /* Go back to hub, not mode select */ }}
      on:filesaved={handleWriteFileSaved}
      on:loading={handleWriteLoading}
      on:error={handleGenericError}
    />
  {/if}
{/if}
<!-- ... -->
```
*(You will also need to add `ListTemplates`, `SaveTemplate`, and `WeaveEntryIntoText` to your Wails bindings in `frontend/wailsjs/go/main/App.js` and `App.d.ts`. A `wails build` will do this automatically.)*

# Notes
[x] - Completed feature

---

### **Phase 3: The `WriteView` Transformation**

Now we'll overhaul `WriteView.svelte` to include the Codex panel and the drag-and-drop logic.

#### 3.1. Create `DropContextMenu.svelte`

This small component will show our drop options.

**File: `/frontend/src/components/DropContextMenu.svelte` (New)**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  export let x: number;
  export let y: number;

  const dispatch = createEventDispatcher();
</script>

<div class="context-menu" style="left: {x}px; top: {y}px;">
  <button on:click={() => dispatch('action', 'reference')}>
    <span class="icon">@</span> Insert as Reference
  </button>
  <button on:click={() => dispatch('action', 'weave')}>
    <span class="icon">✨</span> Weave into Document
  </button>
</div>

<style>
  .context-menu {
    position: fixed;
    z-index: 1001;
    background: var(--bg-secondary);
    border: 1px solid var(--accent-primary);
    border-radius: 8px;
    box-shadow: 0 5px 15px rgba(0,0,0,0.3);
    padding: 0.5rem;
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
  }
  button {
    background: transparent;
    border: none;
    color: var(--text-primary);
    padding: 0.5rem 1rem;
    text-align: left;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  button:hover { background: var(--bg-hover-medium); }
  .icon { font-weight: bold; color: var(--accent-secondary); }
</style>
```

# Notes
[x] - Completed feature

#### 3.2. Major Revision of `WriteView.svelte`

This is the most significant change. We'll add the Codex panel, drag-and-drop listeners, and the logic to call the new backend function.

**File: `/frontend/src/components/WriteView.svelte` (Major Revision)**

```svelte
<script lang="ts">
  // ... (keep existing imports)
  import { GetAllEntries, WeaveEntryIntoText } from '@wailsjs/go/main/App';
  import { database } from '@wailsjs/go/models';
  import DropContextMenu from './DropContextMenu.svelte'; // Import the new component

  // --- Props ---
  export let templateType: string = 'blank';
  // ... (keep other props)
  
  // --- New State ---
  let codexEntries: database.CodexEntry[] = [];
  let codexSearchTerm: string = '';
  let showCodexPanel: boolean = true; // Or make it a tab

  let showDropMenu = false;
  let dropMenuX = 0;
  let dropMenuY = 0;
  let droppedEntry: database.CodexEntry | null = null;
  let dropCursorPosition: number = 0;
  let isWeaving = false;

  onMount(async () => {
    // ... (keep existing onMount logic)
    
    // Fetch codex entries for the panel
    try {
      codexEntries = await GetAllEntries() || [];
    } catch (err) {
      dispatch('error', 'Failed to load Codex entries for reference panel.');
    }
  });

  // --- New Drag and Drop Handlers ---
  function handleDragStart(event: DragEvent, entry: database.CodexEntry) {
    event.dataTransfer?.setData('application/json', JSON.stringify(entry));
  }

  function handleDrop(event: DragEvent) {
    event.preventDefault();
    const entryData = event.dataTransfer?.getData('application/json');
    if (!entryData) return;

    droppedEntry = JSON.parse(entryData);
    dropMenuX = event.clientX;
    dropMenuY = event.clientY;
    
    // Calculate cursor position in textarea
    const target = event.target as HTMLTextAreaElement;
    dropCursorPosition = target.selectionStart; // Or more complex logic to find char under cursor
    
    showDropMenu = true;
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
        writeContent.replace(weavingIndicator, ''), // Send content without the indicator
        dropCursorPosition,
        templateType
      );

      // Replace indicator with generated text
      writeContent = writeContent.replace(weavingIndicator, `\n${generatedText.trim()}\n`);
    } catch(err) {
      dispatch('error', `Llore-weaving failed: ${err}`);
      writeContent = writeContent.replace(weavingIndicator, ''); // Remove indicator on error
    } finally {
      isWeaving = false;
      dispatch('loading', false);
    }
  }

  // Helper to insert text at a specific position
  function insertTextAt(text: string, position: number) {
    writeContent = writeContent.slice(0, position) + text + writeContent.slice(position);
  }

  // Computed property for filtered codex entries
  $: filteredCodexEntries = codexSearchTerm 
    ? codexEntries.filter(e => e.name.toLowerCase().includes(codexSearchTerm.toLowerCase()))
    : codexEntries;

</script>

<!-- The HTML needs a significant restructure -->

<div class="write-view-main-content">
  <!-- LEFT COLUMN: Chat and Tools -->
  <div class="left-column">
    <!-- ... (existing chat and save tools) ... -->
  </div>

  <!-- CENTER COLUMN: Editor -->
  <div class="center-column">
    <!-- ... (existing editor toolbar) ... -->
    <div class="editor-container">
      <textarea
        class="markdown-input"
        on:drop={handleDrop}
        on:dragover|preventDefault
        {...} <!-- Keep existing bindings and event handlers -->
      ></textarea>
      <!-- ... (existing markdown preview) ... -->
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
            draggable="true"
            on:dragstart={(e) => handleDragStart(e, entry)}
          >
            <strong>{entry.name}</strong>
            <span>({entry.type})</span>
          </div>
        {/each}
      </div>
    </div>
    <!-- ... (existing formatting and AI action tools) ... -->
  </div>
</div>

<!-- Drop Context Menu (new) -->
{#if showDropMenu}
  <DropContextMenu x={dropMenuX} y={dropMenuY} on:action={handleDropMenuAction} />
  <!-- Click outside to close -->
  <div class="overlay" on:click={() => showDropMenu = false}></div>
{/if}

<style>
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

  Excellent. Let's continue with the implementation plan, picking up right where we left off. Phase 3 focused on the big `WriteView` revision. Phase 4 will handle the `@mention` functionality and finishing touches.

# Notes
[x] - Completed feature

---

### **Phase 4: Inline Functionality and Polish**

This phase brings the dynamic, in-the-flow features to life and adds the necessary aesthetic polish.

#### 4.1. Implement Inline `@mention` Autocomplete

We need a way to show a dropdown of Codex entries as the user types `@`.

**Step 4.1.1: Create `AutocompleteMenu.svelte`**

This will be a reusable component for showing the list of entries.

**File: `/frontend/src/components/AutocompleteMenu.svelte` (New)**

```svelte
<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { database } from '@wailsjs/go/models';

  export let items: database.CodexEntry[] = [];
  export let x: number;
  export let y: number;
  
  let selectedIndex = 0;
  const dispatch = createEventDispatcher();

  function selectItem(item: database.CodexEntry) {
    dispatch('select', item);
  }

  // Allow keyboard navigation
  export function handleKeyDown(event: KeyboardEvent) {
    if (event.key === 'ArrowDown') {
      event.preventDefault();
      selectedIndex = (selectedIndex + 1) % items.length;
    } else if (event.key === 'ArrowUp') {
      event.preventDefault();
      selectedIndex = (selectedIndex - 1 + items.length) % items.length;
    } else if (event.key === 'Enter' || event.key === 'Tab') {
      event.preventDefault();
      if (items[selectedIndex]) {
        selectItem(items[selectedIndex]);
      }
    }
  }
</script>

<div class="autocomplete-menu" style="left: {x}px; top: {y}px;">
  {#if items.length > 0}
    <ul>
      {#each items as item, i (item.id)}
        <li class:selected={i === selectedIndex} on:mousedown={() => selectItem(item)}>
          <strong>{item.name}</strong>
          <span>({item.type})</span>
        </li>
      {/each}
    </ul>
  {:else}
    <div class="no-results">No matching entries found.</div>
  {/if}
</div>

<style>
  .autocomplete-menu {
    position: fixed;
    z-index: 1002;
    background: var(--bg-secondary);
    border: 1px solid var(--accent-secondary);
    border-radius: 8px;
    box-shadow: 0 4px 12px rgba(0,0,0,0.25);
    max-height: 250px;
    overflow-y: auto;
    min-width: 200px;
  }
  ul { list-style: none; margin: 0; padding: 0.25rem; }
  li {
    padding: 0.5rem 0.75rem;
    cursor: pointer;
    border-radius: 4px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }
  li:hover, li.selected {
    background: var(--accent-primary);
    color: white;
  }
  li strong { font-weight: 500; }
  li span { font-size: 0.8rem; color: var(--text-secondary); }
  li:hover span, li.selected span { color: rgba(255, 255, 255, 0.7); }
  .no-results { padding: 0.75rem; color: var(--text-secondary); }
</style>
```

# Notes
[x] - Completed feature

**Step 4.1.2: Update `WriteView.svelte` with Autocomplete Logic**

**File: `/frontend/src/components/WriteView.svelte` (Additions & Revisions)**

```svelte
<script lang="ts">
  // ... (keep all previous imports)
  import AutocompleteMenu from './AutocompleteMenu.svelte'; // Import the new component

  // --- New State for Autocomplete ---
  let showAutocomplete = false;
  let autocompleteX = 0;
  let autocompleteY = 0;
  let autocompleteItems: database.CodexEntry[] = [];
  let autocompleteQuery = '';
  let autocompleteTriggerPos = 0;
  let autocompleteMenuRef: AutocompleteMenu;

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
  
  function handleWriteViewInput(event: Event) {
    const ta = event.target as HTMLTextAreaElement;
    const cursorPos = ta.selectionStart;
    const textBeforeCursor = ta.value.substring(0, cursorPos);

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
    const textBefore = writeContent.substring(0, autocompleteTriggerPos);
    const textAfter = writeContent.substring(autocompleteTriggerPos + autocompleteQuery.length + 1);
    
    writeContent = textBefore + referenceText + textAfter;
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
</script>

<!-- ... Inside the main div ... -->
<div class="editor-container">
  <textarea
    class="markdown-input"
    on:keydown={handleWriteViewKeydown}
    on:input={handleWriteViewInput}
    {...} <!-- Keep existing bindings and events -->
  ></textarea>
  <!-- ... markdown preview ... -->
</div>

<!-- ... -->

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
  <div class="overlay" on:click={() => showAutocomplete = false}></div>
{/if}
```

# Notes
[x] - Completed feature

#### 4.2. Custom Markdown Rendering for `@mentions`

We need to make our `@mentions` interactive in the preview pane.

**File: `/frontend/src/components/WriteView.svelte` (Revision)**

```svelte
<script lang="ts">
  // ... (keep imports)
  import { Marked } from 'marked';

  // --- Configure marked.js ---
  const marked = new Marked({ gfm: true, breaks: true });

  // Custom renderer for links
  const renderer = new marked.Renderer();
  const originalLinkRenderer = renderer.link;
  renderer.link = (href, title, text) => {
    if (href?.startsWith('codex://entry/')) {
      const entryId = href.substring('codex://entry/'.length);
      // Render as a span with special styling and a data attribute
      return `<span class="codex-mention" data-entry-id="${entryId}" title="Codex Entry: ${text}">${text}</span>`;
    }
    // Fallback to default renderer for other links
    return originalLinkRenderer.call(renderer, href, title, text);
  };
  marked.use({ renderer });

  // ... (rest of the script tag)
</script>

<!-- ... (rest of the component) ... -->

<style>
  /* ... (keep existing styles) ... */

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
</style>
```
*(Note: A tooltip on hover for these mentions would require more complex JS, potentially a separate `Tooltip.svelte` component that you show/hide based on `mouseover` and `mouseout` events on `.codex-mention` elements. This is a great "next step" feature.)*

# Notes
[x] - Completed feature

#### 4.3. Implement "Save as Template"

This adds the final piece to the template workflow.

**File: `/frontend/src/components/WriteView.svelte` (Additions)**

```svelte
<script lang="ts">
  // ... (imports)
  import { SaveTemplate } from '@wailsjs/go/main/App';

  // --- New State ---
  let showSaveTemplateModal = false;
  let newTemplateName = '';

  // --- New Function ---
  async function handleSaveAsTemplate() {
    if (!newTemplateName.trim()) {
      // You can show an error in the modal
      return;
    }
    try {
      await SaveTemplate(newTemplateName, writeContent);
      alert(`Template '${newTemplateName}.md' saved successfully!`);
      showSaveTemplateModal = false;
      newTemplateName = '';
    } catch (err) {
      alert(`Failed to save template: ${err}`);
    }
  }
</script>

<!-- Add a "Save as Template" button in the .save-tools-module -->
<div class="save-tools-module">
  <div class="tool-section">
    <h4>File</h4>
    <div class="save-buttons">
      <!-- ... existing save buttons ... -->
      <button class="template-btn" on:click={() => showSaveTemplateModal = true} disabled={isSaving}>Save as Template</button>
    </div>
    <!-- ... doc info ... -->
  </div>
</div>

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

<style>
  /* ... */
  .template-btn {
    /* Style it differently, maybe with a different color */
    background-color: #fdcb6e !important; /* A gold/yellow color */
    color: #2d3436 !important;
  }
  .template-btn:hover:not(:disabled) {
    background-color: #ffeaa7 !important;
  }
  /* ... */
</style>
```

# Notes
[x] - Completed feature

---

### **Final Review and Summary of Changes**

With these phases completed, you will have a fully integrated writing workflow.

1.  **Backend:**
    *   `vault.go` updated to create a `Templates` directory.
    *   `app.go` has three new powerful functions: `ListTemplates`, `SaveTemplate`, and the core `WeaveEntryIntoText`.

2.  **Frontend Workflow:**
    *   Clicking "Write" now takes you to a new `WriteHub.svelte` component.
    *   The hub lets you start from a blank slate, a built-in template, or a custom template.
    *   `App.svelte` is updated to manage this new "hub-then-editor" flow.

3.  **`WriteView.svelte` Enhancements:**
    *   **Codex Panel:** A new searchable, draggable list of all Codex entries is available in the right-hand sidebar.
    *   **Llore-Weaving:** Dragging an entry onto the editor presents a choice: simple reference or intelligent weaving. The weaving action calls your new backend function to generate context-aware text.
    *   **`@mentions`:** Typing `@` triggers an `AutocompleteMenu.svelte` component, allowing for quick, in-line referencing of Codex entries.
    *   **Custom Rendering:** The Markdown preview now highlights `@mentions`, making them visually distinct and ready for future interactivity (like tooltips).
    *   **Save as Template:** A new button and modal allow the current document to be saved as a reusable template in the vault.

This plan provides a robust and feature-rich implementation that directly addresses your goals. By building it out in these phases, you can tackle the logic systematically, from the data layer up to the user interface. Good luck, and I'm excited to see Llore come to life with these powerful new capabilities