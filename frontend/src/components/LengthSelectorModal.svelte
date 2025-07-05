<script lang="ts">
  import { createEventDispatcher } from 'svelte';

  const dispatch = createEventDispatcher<{
    close: void;
    select: { selectedLength: 'small' | 'medium' | 'large' | 'extra-large' };
  }>();

  const lengthOptions = [
    { 
      value: 'small', 
      label: 'Small', 
      description: 'Exactly 1 sentence that flows naturally',
      icon: '📝'
    },
    { 
      value: 'medium', 
      label: 'Medium', 
      description: 'Approximately 1 paragraph (3-5 sentences)',
      icon: '📄'
    },
    { 
      value: 'large', 
      label: 'Large', 
      description: 'Approximately 1 page worth of content (200-400 words)',
      icon: '📃'
    },
    { 
      value: 'extra-large', 
      label: 'Extra Large', 
      description: 'Approximately 2 pages worth of content (400-800 words)',
      icon: '📚'
    }
  ] as const;

  function handleSelect(length: 'small' | 'medium' | 'large' | 'extra-large') {
    dispatch('select', { selectedLength: length });
  }

  function handleKeydown(event: KeyboardEvent) {
    if (event.key === 'Escape') {
      dispatch('close');
    }
  }
</script>

<svelte:window on:keydown={handleKeydown} />

<div class="modal-backdrop" role="button" tabindex="0" on:click={() => dispatch('close')} on:keydown={(e) => { if (e.key === 'Enter' || e.key === ' ') dispatch('close'); }}>
  <div class="modal length-selector-modal" role="dialog" aria-labelledby="modal-title" on:click|stopPropagation on:keydown|stopPropagation>
    <div class="modal-header">
      <h3>Continue Writing - Select Length</h3>
      <button class="close-btn" on:click={() => dispatch('close')} title="Close">×</button>
    </div>
    
    <div class="modal-content">
      <p class="modal-description">How much content would you like the AI to generate?</p>
      
      <div class="length-options">
        {#each lengthOptions as option}
          <button 
            class="length-option-btn"
            on:click={() => handleSelect(option.value)}
            title={option.description}
          >
            <span class="option-icon">{option.icon}</span>
            <div class="option-content">
              <div class="option-label">{option.label}</div>
              <div class="option-description">{option.description}</div>
            </div>
          </button>
        {/each}
      </div>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background-color: rgba(0, 0, 0, 0.5);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 1000;
  }

  .length-selector-modal {
    background: white;
    border-radius: 8px;
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
    max-width: 500px;
    width: 90%;
    max-height: 80vh;
    overflow-y: auto;
  }

  .modal-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px 24px 16px;
    border-bottom: 1px solid #e5e7eb;
  }

  .modal-header h3 {
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: #1f2937;
  }

  .close-btn {
    background: none;
    border: none;
    font-size: 24px;
    cursor: pointer;
    color: #6b7280;
    padding: 0;
    width: 32px;
    height: 32px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 4px;
    transition: all 0.2s;
  }

  .close-btn:hover {
    background-color: #f3f4f6;
    color: #374151;
  }

  .modal-content {
    padding: 20px 24px 24px;
  }

  .modal-description {
    margin: 0 0 20px 0;
    color: #6b7280;
    font-size: 0.95rem;
    line-height: 1.5;
  }

  .length-options {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .length-option-btn {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;
    border: 2px solid #e5e7eb;
    border-radius: 8px;
    background: white;
    cursor: pointer;
    transition: all 0.2s;
    text-align: left;
    width: 100%;
  }

  .length-option-btn:hover {
    border-color: #3b82f6;
    background-color: #f8fafc;
    transform: translateY(-1px);
    box-shadow: 0 2px 8px rgba(59, 130, 246, 0.15);
  }

  .length-option-btn:active {
    transform: translateY(0);
  }

  .option-icon {
    font-size: 1.5rem;
    flex-shrink: 0;
  }

  .option-content {
    flex: 1;
  }

  .option-label {
    font-weight: 600;
    color: #1f2937;
    margin-bottom: 4px;
    font-size: 1rem;
  }

  .option-description {
    color: #6b7280;
    font-size: 0.875rem;
    line-height: 1.4;
  }

  /* Dark mode support */
  @media (prefers-color-scheme: dark) {
    .length-selector-modal {
      background: #1f2937;
      color: #f9fafb;
    }

    .modal-header {
      border-bottom-color: #374151;
    }

    .modal-header h3 {
      color: #f9fafb;
    }

    .close-btn {
      color: #9ca3af;
    }

    .close-btn:hover {
      background-color: #374151;
      color: #d1d5db;
    }

    .modal-description {
      color: #9ca3af;
    }

    .length-option-btn {
      background: #374151;
      border-color: #4b5563;
      color: #f9fafb;
    }

    .length-option-btn:hover {
      border-color: #3b82f6;
      background-color: #1e293b;
    }

    .option-label {
      color: #f9fafb;
    }

    .option-description {
      color: #9ca3af;
    }
  }
</style>
