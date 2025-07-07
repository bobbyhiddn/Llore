<!-- frontend/src/components/Editor.svelte -->
<script lang="ts">
    import { createHistory } from '../../lib/history';
    import { onMount, onDestroy } from 'svelte';
    import { get } from 'svelte/store';
    import '../../styles/Editor.css';

    // The initial content passed to the editor
    export let content: string = '';
    // A function to call when the content needs to be saved (e.g., autosave)
    export let onSave: (newContent: string) => void = () => {};
    // Optional placeholder text
    export let placeholder: string = 'Start writing your lore...';
    // Optional class for styling
    export let editorClass: string = '';

    const { present, undo, redo, recordDebounced, reset, undoStack, redoStack } = createHistory(content);

    // When the 'content' prop changes from outside (e.g., loading a new document),
    // we must reset the history store.
    $: if (content !== get(present)) {
        reset(content);
    }
    
    // --- Autosave Logic ---
    let autoSaveTimer: number;
    function triggerAutoSave(value: string) {
        clearTimeout(autoSaveTimer);
        autoSaveTimer = window.setTimeout(() => {
            onSave(value);
        }, 1500); // Autosave 1.5 seconds after user stops typing
    }

    // --- Event Handlers ---
    function handleInput() {
        // `$present` is already updated by the `bind:value` directive.
        // We just need to record the new state (debounced) and trigger autosave.
        recordDebounced(get(present), 500);
        triggerAutoSave(get(present));
    }

    function handleKeyDown(event: KeyboardEvent) {
        // Use `metaKey` for macOS (Command key) and `ctrlKey` for Windows/Linux
        const modifier = event.metaKey || event.ctrlKey;

        // --- Undo Logic (Ctrl+Z) ---
        if (modifier && !event.shiftKey && event.key.toLowerCase() === 'z') {
            event.preventDefault(); // Prevent browser's default undo action
            undo();
            triggerAutoSave(get(present)); // Trigger save after undoing
        }

        // --- Redo Logic (Ctrl+Y or Ctrl+Shift+Z) ---
        const isRedoY = modifier && !event.shiftKey && event.key.toLowerCase() === 'y';
        const isRedoShiftZ = modifier && event.shiftKey && event.key.toLowerCase() === 'z';

        if (isRedoY || isRedoShiftZ) {
            event.preventDefault(); // Prevent browser's default redo action
            redo();
            triggerAutoSave(get(present)); // Trigger save after redoing
        }
    }

    function handleUndoClick() {
        undo();
        triggerAutoSave(get(present));
    }

    function handleRedoClick() {
        redo();
        triggerAutoSave(get(present));
    }

    onDestroy(() => {
        // Clean up timer on component destruction
        clearTimeout(autoSaveTimer);
    });
</script>

<div class="editor-container {editorClass}">
    <div class="editor-toolbar">
        <button 
            class="toolbar-btn"
            on:click={handleUndoClick} 
            disabled={$undoStack.length <= 1}
            title="Undo (Ctrl+Z)"
        >
            ↶ Undo
        </button>
        <button 
            class="toolbar-btn"
            on:click={handleRedoClick} 
            disabled={$redoStack.length === 0}
            title="Redo (Ctrl+Y)"
        >
            ↷ Redo
        </button>
    </div>
    <textarea
        class="editor-textarea"
        bind:value={$present}
        on:input={handleInput}
        on:keydown={handleKeyDown}
        {placeholder}
    ></textarea>
</div>
