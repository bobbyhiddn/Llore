<!-- frontend/src/components/StoryImportStatus.svelte -->
<script lang="ts">
  import type { database } from '@wailsjs/go/models'; // Import namespace for CodexEntry type

  // --- Props ---
  export let status: 'idle' | 'sending' | 'receiving' | 'parsing' | 'checking_existing' | 'updating' | 'embedding' | 'library' | 'complete' | 'error' = 'idle'; // Added 'library'
  export let errorMsg: string | null = null;
  export let newEntries: database.CodexEntry[] = [];
  export let updatedEntries: database.CodexEntry[] = [];

  // --- Reactive Computations ---
  $: totalNew = newEntries.length;
  $: totalUpdated = updatedEntries.length;

  // --- Status Text Mapping ---
  const statusMap = {
    idle: 'Waiting for import...',
    sending: 'Sending story to AI...',
    receiving: 'Receiving response from AI...',
    parsing: 'Parsing AI response into entries...',
    checking_existing: 'Checking for existing entries...',
    updating: 'Creating updates for existing entries...',
    embedding: 'Generating embeddings for new/updated entries...',
    library: 'Saving story to library...',
    complete: 'Import process finished.',
    error: 'An error occurred during import.',
  };

  $: currentStatusText = statusMap[status] || 'Unknown status';

  // --- Time Tracking ---
  let startTime: number | null = null;
  let elapsedTime = '0s';
  let elapsedInterval: number | null = null;

  // Update elapsed time every second while processing
  $: {
    if (status !== 'idle' && status !== 'complete' && status !== 'error') {
      if (!startTime) {
        startTime = Date.now();
        elapsedInterval = window.setInterval(() => {
          const elapsed = Math.floor((Date.now() - startTime!) / 1000);
          const minutes = Math.floor(elapsed / 60);
          const seconds = elapsed % 60;
          elapsedTime = minutes > 0 ? `${minutes}m ${seconds}s` : `${seconds}s`;
        }, 1000);
      }
    } else {
      if (elapsedInterval) {
        clearInterval(elapsedInterval);
        elapsedInterval = null;
      }
      if (status === 'idle') {
        startTime = null;
        elapsedTime = '0s';
      }
    }
  }

  // --- Helper to determine if a step is active or done ---
  const isActiveOrDone = (stepStatus: typeof status) => {
    // Added 'library' and 'embedding' to the order
    const order: (typeof status)[] = ['sending', 'receiving', 'parsing', 'checking_existing', 'updating', 'embedding', 'library', 'complete'];
    const currentIdx = order.indexOf(status);
    const stepIdx = order.indexOf(stepStatus);

    // A step is 'done' if the current status index is greater than the step's index,
    // OR if the status is 'complete', regardless of index (as complete is the final state).
    // It should not be marked done if there was an error before reaching it.
    const isDone = status !== 'error' && (currentIdx > stepIdx || status === 'complete');

    // --- DEBUG LOGGING --- 
    if (status === 'complete') {
        console.log(`[StoryImportStatus] Status='complete'. Checking step '${stepStatus}': isDone=${isDone}`);
    }
    // --- END DEBUG LOGGING ---

    return isDone;
  };
  
  // --- Helper to extract entry data from various formats ---
  function extractEntryData(entry: any): { name: string, type: string, content: string } {
    // If the entry is already a proper object with name and type, use it directly
    if (entry && typeof entry === 'object' && !Array.isArray(entry) && entry.name && entry.type) {
      return {
        name: entry.name,
        type: entry.type,
        content: entry.content || ''
      };
    }
    
    // Check if the entry might be a stringified JSON
    if (typeof entry === 'string') {
      try {
        let parsed = JSON.parse(entry);
        
        // Handle case where it's a JSON array with a single object
        if (Array.isArray(parsed) && parsed.length > 0) {
          parsed = parsed[0]; // Take the first item from the array
        }
        
        if (parsed && typeof parsed === 'object' && parsed.name && parsed.type) {
          return {
            name: parsed.name,
            type: parsed.type,
            content: parsed.content || ''
          };
        }
      } catch (e) {
        console.error('Failed to parse entry as JSON:', e);
      }
    }
    
    // Return a default object if parsing failed
    return {
      name: 'Unknown',
      type: 'Unknown',
      content: String(entry)
    };
  }
</script>

<div class="status-container">
  <h4>Import Progress</h4>

  {#if status !== 'idle'}
    {#key status} <!-- Force re-render on status change -->
      <div class="status-steps">
        <div class="step" class:active={status === 'sending'} class:done={isActiveOrDone('sending')}>
          <span class="dot"></span> Sending to AI
        </div>
        <div class="step" class:active={status === 'receiving'} class:done={isActiveOrDone('receiving')}>
          <span class="dot"></span> Receiving Response
        </div>
        <div class="step" class:active={status === 'parsing'} class:done={isActiveOrDone('parsing')}>
          <span class="dot"></span> Parsing Entries
        </div>
        {#if isActiveOrDone('checking_existing') || status === 'checking_existing'}
          <div class="step optional" class:active={status === 'checking_existing'} class:done={isActiveOrDone('checking_existing')}>
            <span class="dot"></span> Checking Existing
          </div>
        {/if}
        {#if isActiveOrDone('updating') || status === 'updating'}
          <div class="step optional" class:active={status === 'updating'} class:done={isActiveOrDone('updating')}>
            <span class="dot"></span> Updating Entries
          </div>
        {/if}
        {#if isActiveOrDone('embedding') || status === 'embedding'} <!-- Added Embedding Step -->
          <div class="step" class:active={status === 'embedding'} class:done={isActiveOrDone('embedding')}>
            <span class="dot"></span> Generating Embeddings
          </div>
        {/if}
        {#if isActiveOrDone('library') || status === 'library'} <!-- Added Library Step -->
          <div class="step" class:active={status === 'library'} class:done={isActiveOrDone('library')}>
            <span class="dot"></span> Saving to Library
          </div>
        {/if}
      </div>
    {/key} <!-- End key block -->

    <p class="current-status {status}">
      {currentStatusText}
      {#if status !== 'complete' && status !== 'error'}
        <span class="elapsed-time">({elapsedTime})</span>
      {/if}
    </p>

    {#if status === 'complete'}
      <div class="results">
        <h5>Results:</h5>
        {#if totalNew > 0 || totalUpdated > 0}
          {#if totalNew > 0}
            <p><strong>New Entries ({totalNew}):</strong></p>
            <ul>
              {#each newEntries as entry (entry.id)}
                <!-- Extract entry data from various formats -->
                {@const entryData = extractEntryData(entry)}
                <li>{entryData.name} ({entryData.type}): {entryData.content}</li>
              {/each}
            </ul>
          {/if}
          {#if totalUpdated > 0}
            <p><strong>Updated Entries ({totalUpdated}):</strong></p>
            <ul>
              {#each updatedEntries as entry (entry.id)}
                <!-- Extract entry data from various formats -->
                {@const entryData = extractEntryData(entry)}
                <li>{entryData.name} ({entryData.type}): {entryData.content}</li>
              {/each}
            </ul>
          {/if}
        {:else}
          <p>No new or updated entries were created.</p>
        {/if}
      </div>
    {/if}

    {#if status === 'error' && errorMsg}
      <p class="error-message">{errorMsg}</p>
    {/if}

  {/if}
</div>

<style>
  .status-container {
    border: 1px solid var(--bg-tertiary);
    background-color: var(--bg-secondary);
    padding: 1rem 1.5rem;
    border-radius: 8px;
    margin-top: 1.5rem;
    color: var(--text-secondary);
    /* max-height: 300px; */ /* REMOVED height limit */
    /* overflow-y: auto; */ /* REMOVED overflow - parent handles scrolling */
    display: flex; /* Use flexbox for internal layout */
    flex-direction: column; /* Stack children vertically */
  }

  h4 {
    margin-top: 0;
    margin-bottom: 1rem;
    color: var(--text-primary);
    font-weight: 600;
  }

  .status-steps {
    margin-bottom: 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem; /* Spacing between steps */
  }

  .step {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    color: var(--text-secondary);
    opacity: 0.6;
    transition: color 0.4s ease, opacity 0.4s ease, font-weight 0.4s ease; /* Added transition */
    font-size: 0.9rem;
  }

  .step .dot {
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background-color: var(--text-secondary); /* Default dot color */
    transition: background-color 0.4s ease, box-shadow 0.4s ease; /* Added transition */
    flex-shrink: 0;
  }

  .step.active {
    color: var(--accent-primary);
    opacity: 1;
    font-weight: 500;
  }
  .step.active .dot {
    background-color: var(--accent-primary); /* Active dot color */
    /* Optional: Add a subtle pulse or glow */
    box-shadow: 0 0 5px var(--accent-primary);
  }

  .step.done {
    color: var(--text-primary); /* Slightly brighter for done steps */
    opacity: 0.8;
  }
  .step.done .dot {
    background-color: var(--success-color); /* Green for done */
  }

  .step.optional {
    /* Slightly different style for optional steps if needed */
    font-style: italic;
  }

  .current-status {
    margin-top: 1rem;
    margin-bottom: 1rem;
    font-weight: 500;
    font-size: 0.95rem;
    padding: 0.5rem 0.75rem;
    border-radius: 4px;
    background-color: rgba(255, 255, 255, 0.05);
    transition: color 0.4s ease, background-color 0.4s ease; /* Added transition */
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .current-status .elapsed-time {
    color: var(--text-secondary);
    font-size: 0.85rem;
    font-weight: normal;
  }
  .current-status.complete {
      color: var(--success-color);
      background-color: rgba(46, 204, 113, 0.1);
  }
  .current-status.error {
      color: var(--error-color);
      background-color: rgba(255, 71, 87, 0.1);
  }

  .results {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid var(--bg-tertiary);
  }

  .results h5 {
    margin-bottom: 0.75rem;
    color: var(--text-primary);
  }

  .results ul {
    list-style: none;
    padding-left: 1rem;
    margin: 0.5rem 0;
    max-height: 150px; /* Limit height if many entries - Confirmed */
    overflow-y: auto; /* Confirmed */
  }

  .results li {
    margin-bottom: 0.3rem;
    font-size: 0.9rem;
  }

  .error-message {
    color: var(--error-color);
    background: rgba(255, 71, 87, 0.1);
    padding: 0.75rem 1rem;
    border-radius: 4px;
    margin-top: 1rem;
    border: 1px solid rgba(255, 71, 87, 0.2);
    font-size: 0.9rem;
  }

  /* Scrollbar for results list */
  .results ul::-webkit-scrollbar {
    width: 4px;
  }
  .results ul::-webkit-scrollbar-track {
    background: rgba(255, 255, 255, 0.05);
    border-radius: 2px;
  }
  .results ul::-webkit-scrollbar-thumb {
    background: var(--accent-primary);
    border-radius: 2px;
  }
  .results ul::-webkit-scrollbar-thumb:hover {
    background: var(--accent-secondary);
  }
</style>