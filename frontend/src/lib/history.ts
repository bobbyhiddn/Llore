// frontend/src/lib/history.ts

import { writable, get } from 'svelte/store';
import type { Writable } from 'svelte/store';

const MAX_HISTORY_SIZE = 100;

/**
 * Creates a store for managing state history (undo/redo).
 * @param initialValue The initial string value of the content.
 */
export function createHistory(initialValue: string) {
    const present = writable(initialValue);
    const undoStack: Writable<string[]> = writable([initialValue]);
    const redoStack: Writable<string[]> = writable([]);

    let lastRecorded = initialValue;

    /**
     * Records a new state in the history.
     * This clears the redo stack.
     * @param value The new string value to record.
     */
    function record(value: string) {
        if (value === lastRecorded) {
            return; // No change, do not record
        }
        
        // When a new state is recorded, the redo stack is cleared.
        const currentUndoStack = get(undoStack);
        // Add new value and trim history if it's too long
        undoStack.set([...currentUndoStack.slice(-MAX_HISTORY_SIZE + 1), value]);
        redoStack.set([]);
        lastRecorded = value;
    }

    /**
     * Moves one step back in history.
     */
    function undo() {
        const uStack = get(undoStack);
        if (uStack.length <= 1) { // Can't undo the initial state
            return;
        }

        const currentValue = get(present);
        redoStack.update(stack => [currentValue, ...stack]);

        // Get the previous value and update the undo stack
        const previousValue = uStack[uStack.length - 2];
        undoStack.update(stack => stack.slice(0, stack.length - 1));
        
        present.set(previousValue);
        lastRecorded = previousValue;
    }

    /**
     * Moves one step forward in history.
     */
    function redo() {
        const rStack = get(redoStack);
        if (rStack.length === 0) {
            return;
        }
        
        const currentValue = get(present);
        undoStack.update(stack => [...stack, currentValue]);

        const nextValue = rStack[0];
        redoStack.update(stack => stack.slice(1));

        present.set(nextValue);
        lastRecorded = nextValue;
    }

    // A debounced version of `record` for frequent updates like typing.
    let debounceTimer: number;
    function recordDebounced(value: string, delay: number = 500) {
        clearTimeout(debounceTimer);
        debounceTimer = window.setTimeout(() => {
            record(value);
        }, delay);
    }

    // When external content changes, reset the history.
    function reset(newValue: string) {
        present.set(newValue);
        undoStack.set([newValue]);
        redoStack.set([]);
        lastRecorded = newValue;
    }

    return {
        present,
        undo,
        redo,
        record,
        recordDebounced,
        reset,
        // Expose stacks for UI binding (e.g., disabling buttons)
        undoStack,
        redoStack
    };
}
