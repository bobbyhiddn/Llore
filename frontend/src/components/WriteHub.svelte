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

  function goBackToModeSelection() {
    dispatch('backtomodeselection');
  }
</script>

<div class="write-hub-container">
  <button class="back-btn" on:click={goBackToModeSelection}>← Back to Mode Selection</button>
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
  .back-btn {
    position: absolute;
    top: 1rem;
    left: 1rem;
    background: var(--bg-secondary);
    color: var(--text-secondary);
    border: 1px solid rgba(255, 255, 255, 0.1);
    border-radius: 6px;
    padding: 0.5rem 1rem;
    font-size: 0.9rem;
    cursor: pointer;
    transition: all 0.2s ease;
  }
  
  .back-btn:hover {
    background: var(--bg-hover-medium);
    color: var(--text-primary);
    border-color: var(--accent-primary);
  }
  
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