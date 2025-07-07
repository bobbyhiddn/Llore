<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import '../../styles/LengthSelectorModal.css';

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

