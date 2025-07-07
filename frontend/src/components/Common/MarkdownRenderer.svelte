<script lang="ts">
  import { onMount } from 'svelte';
  
  export let content: string = '';
  export let inline: boolean = false;
  
  let renderedContent: string = '';
  
  // Simple markdown renderer that's safer and more controlled
  function renderMarkdown(text: string): string {
    if (!text) return '';
    
    let html = text;
    
    // Escape HTML first to prevent XSS
    html = html
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
    
    // Apply markdown formatting
    if (!inline) {
      // Headers
      html = html.replace(/^### (.*$)/gm, '<h3>$1</h3>');
      html = html.replace(/^## (.*$)/gm, '<h2>$1</h2>');
      html = html.replace(/^# (.*$)/gm, '<h1>$1</h1>');
      
      // Lists
      html = html.replace(/^\* (.+)$/gm, '<li>$1</li>');
      html = html.replace(/^- (.+)$/gm, '<li>$1</li>');
      html = html.replace(/^(\d+)\. (.+)$/gm, '<li>$1. $2</li>');
      
      // Wrap consecutive list items in ul tags
      html = html.replace(/(<li>.*<\/li>)/gs, (match) => {
        const items = match.split('</li>').filter(item => item.trim());
        if (items.length > 0) {
          return '<ul>' + items.map(item => item + '</li>').join('') + '</ul>';
        }
        return match;
      });
      
      // Paragraphs (split by double newlines)
      const paragraphs = html.split(/\n\s*\n/);
      html = paragraphs.map(p => {
        p = p.trim();
        if (!p) return '';
        if (p.startsWith('<h') || p.startsWith('<ul') || p.startsWith('<ol')) {
          return p;
        }
        return `<p>${p.replace(/\n/g, '<br>')}</p>`;
      }).join('');
    }
    
    // Inline formatting (works for both inline and block)
    html = html.replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
    html = html.replace(/\*(.*?)\*/g, '<em>$1</em>');
    html = html.replace(/`(.*?)`/g, '<code>$1</code>');
    
    // Line breaks for inline mode
    if (inline) {
      html = html.replace(/\n/g, '<br>');
    }
    
    return html;
  }
  
  $: renderedContent = renderMarkdown(content);
</script>

<div class="markdown-content" class:inline>
  {@html renderedContent}
</div>

<style>
  .markdown-content {
    max-width: 100%;
    word-wrap: break-word;
    overflow-wrap: break-word;
  }
  
  .markdown-content.inline {
    display: inline;
  }
  
  .markdown-content :global(h1),
  .markdown-content :global(h2),
  .markdown-content :global(h3) {
    font-weight: 600;
    margin: 0.5rem 0 0.25rem 0;
    line-height: 1.3;
  }
  
  .markdown-content :global(h1) { font-size: 0.85rem; }
  .markdown-content :global(h2) { font-size: 0.8rem; }
  .markdown-content :global(h3) { font-size: 0.75rem; }
  
  .markdown-content :global(p) {
    margin: 0 0 0.5rem 0;
    line-height: 1.5;
  }
  
  .markdown-content :global(p:last-child) {
    margin-bottom: 0;
  }
  
  .markdown-content :global(ul),
  .markdown-content :global(ol) {
    margin: 0.5rem 0;
    padding-left: 1.2rem;
  }
  
  .markdown-content :global(li) {
    margin-bottom: 0.25rem;
    line-height: 1.4;
  }
  
  .markdown-content :global(strong) {
    font-weight: 600;
  }
  
  .markdown-content :global(em) {
    font-style: italic;
  }
  
  .markdown-content :global(code) {
    background: rgba(255, 255, 255, 0.1);
    padding: 0.1rem 0.3rem;
    border-radius: 3px;
    font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
    font-size: 0.85em;
  }
  
  .markdown-content :global(br) {
    line-height: 1.5;
  }
</style>