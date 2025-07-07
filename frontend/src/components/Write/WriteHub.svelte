<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import { ListTemplates, SaveTemplate, ReadLibraryFile } from '@wailsjs/go/main/App';
  import '../../styles/WriteHub.css';

  const dispatch = createEventDispatcher();

  let customTemplates: string[] = [];
  let isLoading = true;
  let showCreateTemplateModal = false;
  let templateName = '';
  let templateFrontendText = '';
  let templateBackendInstructions = '';
  let createTemplateError = '';
  let isSavingTemplate = false;

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

  function openCreateTemplateModal() {
    showCreateTemplateModal = true;
    templateName = '';
    templateFrontendText = '';
    templateBackendInstructions = '';
    createTemplateError = '';
  }

  function closeCreateTemplateModal() {
    showCreateTemplateModal = false;
  }

  async function saveCustomTemplate() {
    if (!templateName.trim()) {
      createTemplateError = 'Template name is required';
      return;
    }

    isSavingTemplate = true;
    createTemplateError = '';

    try {
      // Create markdown template with frontmatter
      const frontmatter = [
        '---',
        `name: "${templateName.trim()}"`,
        `type: custom`,
        `created: ${new Date().toISOString()}`,
        templateBackendInstructions.trim() ? `instructions: "${templateBackendInstructions.trim().replace(/"/g, '\\"')}"` : '',
        '---',
        ''
      ].filter(line => line !== '').join('\n');

      const templateContent = frontmatter + templateFrontendText.trim();
      
      // Save as markdown file
      const filename = `${templateName.trim().replace(/[^a-zA-Z0-9\s-]/g, '').replace(/\s+/g, '-').toLowerCase()}.md`;
      await SaveTemplate(filename, templateContent);
      
      // Refresh template list
      customTemplates = await ListTemplates() || [];
      
      closeCreateTemplateModal();
    } catch (error) {
      createTemplateError = `Failed to save template: ${error}`;
    } finally {
      isSavingTemplate = false;
    }
  }

  async function loadCustomTemplate(templateFile: string) {
    try {
      console.log('loadCustomTemplate called with:', templateFile);
      
      // Read the markdown template file
      const templateContent = await ReadLibraryFile(`../Templates/${templateFile}`);
      
      // Parse frontmatter if present
      let content = templateContent;
      let templateName = templateFile.replace('.md', '');
      
      if (templateContent.startsWith('---')) {
        const frontmatterEnd = templateContent.indexOf('---', 3);
        if (frontmatterEnd !== -1) {
          // Extract content after frontmatter
          content = templateContent.substring(frontmatterEnd + 3).trim();
          
          // Extract template name from frontmatter if available
          const frontmatter = templateContent.substring(3, frontmatterEnd);
          const nameMatch = frontmatter.match(/name:\s*"([^"]+)"/);  
          if (nameMatch) {
            templateName = nameMatch[1];
          }
        }
      }
      
      console.log('Starting writing with custom template:', templateName);
      startWriting(content, templateName);
      
    } catch (error) {
      console.error('Failed to load custom template:', error);
    }
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
    
    <!-- Create Template Card -->
    <button class="option-card create-template-card" on:click={openCreateTemplateModal}>
      <div class="icon">✨</div>
      <div class="title">Create Template</div>
      <div class="description">Design a custom writing template with AI instructions.</div>
    </button>
  </div>

  {#if isLoading}
    <p>Loading custom templates...</p>
  {:else if customTemplates.length > 0}
    <h3 class="custom-templates-header">Your Templates</h3>
    <div class="custom-templates-list">
      {#each customTemplates as templateFile}
        <button class="custom-template-item" on:click={() => loadCustomTemplate(templateFile)}>
          {templateFile.replace(/\.(md|json)$/, '')}
        </button>
      {/each}
    </div>
  {/if}
</div>

<!-- Create Template Modal -->
{#if showCreateTemplateModal}
  <div class="modal-backdrop">
    <div class="modal create-template-modal">
      <h3>Create Custom Template</h3>
      <p class="modal-description">Design a template with optional starter text and AI instructions for enhanced writing assistance.</p>
      
      <label for="template-name">Template Name *</label>
      <input 
        id="template-name" 
        type="text" 
        bind:value={templateName} 
        placeholder="e.g., Sci-Fi Chapter, Character Profile"
        maxlength="50"
      >
      
      <label for="template-frontend">Starter Text (Optional)</label>
      <textarea 
        id="template-frontend"
        bind:value={templateFrontendText}
        placeholder="Initial content that will appear in the editor when this template is used..."
        rows="4"
      ></textarea>
      
      <label for="template-backend">AI Instructions (Optional)</label>
      <textarea 
        id="template-backend"
        bind:value={templateBackendInstructions}
        placeholder="Instructions for AI assistance when using this template (e.g., 'Focus on vivid descriptions and sensory details for fantasy scenes')..."
        rows="3"
      ></textarea>
      
      {#if createTemplateError}
        <p class="error-message">{createTemplateError}</p>
      {/if}
      
      <div class="modal-buttons">
        <button on:click={closeCreateTemplateModal} disabled={isSavingTemplate}>Cancel</button>
        <button on:click={saveCustomTemplate} disabled={isSavingTemplate || !templateName.trim()} class="primary">
          {#if isSavingTemplate}Saving...{:else}Create Template{/if}
        </button>
      </div>
    </div>
  </div>
{/if}