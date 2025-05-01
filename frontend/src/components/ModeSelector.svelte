<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import logo from '../assets/images/logo.png';

  const dispatch = createEventDispatcher();

  function setMode(mode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write') {
    dispatch('setmode', mode);
  }
</script>

<div class="mode-select">
  <div class="scroll-stave-top"></div> <!-- Top Stave -->
  <div class="scroll-container">
    <img src={logo} alt="Llore Logo" class="logo" style="margin-bottom: 1.5rem;" />
    <div class="mode-buttons">
      <button
        on:click={() => setMode('codex')}
        class="mode-button"
      >
        <span class="title">Codex</span>
        <span class="description">Manage your world's knowledge</span>
      </button>
      <button
        on:click={() => setMode('story')}
        class="mode-button"
      >
        <span class="title">Story Import</span>
        <span class="description">Analyze and extract lore</span>
      </button>
      <button
        on:click={() => setMode('library')}
        class="mode-button"
      >
        <span class="title">Library</span>
        <span class="description">Organize your story files</span>
      </button>
      <button
        on:click={() => setMode('chat')}
        class="mode-button"
      >
        <span class="title">Lore Chat</span>
        <span class="description">Explore your world with AI</span>
      </button>
      <button
        on:click={() => setMode('settings')}
        class="mode-button"
      >
        <span class="title">Settings</span>
        <span class="description">Configure your experience</span>
      </button>
      <button
        on:click={() => setMode('write')}
        class="mode-button"
      >
        <span class="title">Write</span>
        <span class="description">Compose stories & articles</span>
      </button>
    </div>
  </div>
  <div class="scroll-stave-bottom"></div> <!-- Bottom Stave -->
</div>

<style>
  /* Mode Selection Screen */
  .mode-select {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center; /* Center the scroll container vertically */
    height: 100%; /* Fill parent #app */
    width: 100%; /* Ensure it respects parent width */
    padding: 1rem 2rem; /* Add some overall padding */
    background: var(--bg-primary); /* Keep overall background dark */
    position: relative; /* Needed for absolute positioning of staves */
  }

  .scroll-container {
    background: #fdf6e3; /* Parchment-like color */
    padding: 2rem 2.5rem;
    border-radius: 15px;
    overflow-y: auto; /* Make the inner container scroll */
    max-height: calc(100% - 80px); /* Limit height (adjust 80px as needed for staves/padding) */
    box-shadow: 0 10px 30px rgba(0, 0, 0, 0.3), 0 0 15px rgba(253, 246, 227, 0.1) inset;
    /* Use clamp for responsive width: min 90%, preferred 60vw, max 800px */
    width: clamp(90%, 60vw, 800px);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 2rem;
    border: 1px solid rgba(0,0,0,0.1);
    position: relative; /* Keep relative for content */
    z-index: 5; /* Ensure parchment is behind staves */
  }

  .scroll-container .logo {
    width: 150px; /* Reduced from 200px */
    height: auto;
    margin-bottom: 1.5rem;
    animation: float 6s ease-in-out infinite;
    filter: drop-shadow(0 2px 3px rgba(0,0,0,0.2));
  }

  @keyframes float {
    0% {
      transform: translateY(0);
    }
    50% {
      transform: translateY(-10px);
    }
    100% {
      transform: translateY(0);
    }
  }

  .mode-buttons {
    /* Use Grid for responsive button layout */
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.75rem;
    width: 100%;
    max-width: 500px;
    margin-top: 0;
    margin-left: auto;
    margin-right: auto;
  }

  .mode-button {
    /* Removed width: 100% - Grid handles sizing */
    padding: 0.4rem 1.2rem;
    font-size: 0.9rem;
    text-align: left;
    background: #f5eeda;
    border: 1px solid #a0937d;
    border-radius: 8px;
    color: #65594a;
    cursor: pointer;
    transition: all 0.3s ease;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    position: relative;
    overflow: hidden;
    font-family: 'Georgia', serif;
  }

  .mode-button .title {
    font-weight: 600;
    color: #584c3a;
    z-index: 1;
    font-size: 1rem;
  }

  .mode-button .description {
    font-size: 0.85rem;
    color: #8a7a66;
    opacity: 0;
    transform: translateY(10px);
    transition: all 0.3s ease;
    z-index: 1;
  }

  .mode-button:hover {
    background: rgba(88, 76, 58, 0.05);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(88, 76, 58, 0.15);
    border-color: #584c3a;
  }

  .mode-button:hover .description {
    opacity: 1;
    transform: translateY(0);
  }

  /* Staves */
  .scroll-stave-top,
  .scroll-stave-bottom {
    position: relative;
    width: 100%;
    max-width: 800px;
    height: 30px;
    background: linear-gradient(to right, #8B4513, #A0522D, #8B4513);
    border-radius: 15px;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.4);
    z-index: 10;
    border: 1px solid #5c2e11;
  }
</style>