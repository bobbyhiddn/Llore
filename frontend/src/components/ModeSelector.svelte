<script lang="ts">
  import { createEventDispatcher, onMount } from 'svelte';
  import logo from '../assets/images/logo.png';

  const dispatch = createEventDispatcher();

  function setMode(mode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write') {
    dispatch('setmode', mode);
  }

  function backToVaultSelect() {
    dispatch('backtovault');
  }

  let startAnimation = false;
  onMount(() => {
    setTimeout(() => {
      startAnimation = true;
    }, 100); 
  });
</script>

<div class="mode-select" class:animate-scroll={startAnimation}>
  <!-- Back to Vault Select Button -->
  <button class="back-to-vault-btn" on:click={backToVaultSelect} title="Back to Vault Selection">
    ← Back to Vault Selection
  </button>

  <div class="scroll-stave-assembly top-stave">
    <div class="stave-handle left"></div>
    <div class="stave-roller"></div>
    <div class="stave-handle right"></div>
  </div>

  <div class="scroll-container">
    <img src={logo} alt="Llore Logo" class="logo scroll-content-item" />
    <div class="mode-buttons scroll-content-item">
      <!-- ... buttons ... -->
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

  <div class="scroll-stave-assembly bottom-stave">
    <div class="stave-handle left"></div>
    <div class="stave-roller"></div>
    <div class="stave-handle right"></div>
  </div>
</div>

<style>
  /* Mode Selection Screen */
  .mode-select {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    width: 100%;
    padding: 1rem;
    background: var(--bg-primary);
    position: relative;
    opacity: 0; /* Start hidden for overall fade-in */
    transition: opacity 0.5s ease-in-out;
  }

  .mode-select.animate-scroll {
    opacity: 1;
  }

  /* Stave Assembly - Container for roller and handles */
  .scroll-stave-assembly {
    display: flex;
    align-items: center;
    width: clamp(400px, 80vw, 950px);
    position: relative;
    z-index: 10;
    flex-shrink: 0;
    opacity: 0;
    transform: scale(0.95);
  }

  .animate-scroll .scroll-stave-assembly {
    animation: fadeInStaves 0.8s cubic-bezier(0.25, 0.8, 0.25, 1) forwards;
    animation-delay: 0.2s; /* Staves appear first */
  }

  /* Stave Roller - The main horizontal bar */
  .stave-roller {
    height: 35px;
    flex-grow: 1;
    min-width: clamp(200px, 50vw, 700px);
    background: linear-gradient(to right, #8B4513, #A0522D, #8B4513);
    border-radius: 5px;
    box-shadow: 0 3px 7px rgba(0, 0, 0, 0.4),
                inset 0 2px 3px rgba(255,255,255,0.1),
                inset 0 -2px 3px rgba(0,0,0,0.2);
    border: 1.5px solid #5c2e11;
    position: relative;
    z-index: 9;
  }

  /* Stave Handle - The "cap" at the end of the roller */
  .stave-handle {
    width: 25px;
    height: 40px;
    background: linear-gradient(to right, #703816, #8B4513, #703816);
    box-shadow: 0 2px 5px rgba(0,0,0,0.35),
                inset 0 1px 2px rgba(255,255,255,0.08),
                inset 0 -1px 2px rgba(0,0,0,0.12);
    border: 1.5px solid #4a2a0f;
    position: relative;
    flex-shrink: 0;
    z-index: 11;
    display: flex;
    align-items: center;
  }
  .stave-handle.left {
    border-radius: 20px 3px 3px 20px;
    margin-right: -2px;
  }
  .stave-handle.right {
    border-radius: 3px 20px 20px 3px;
    margin-left: -2px;
  }

  .stave-handle::before {
    content: '';
    position: absolute;
    top: 50%;
    transform: translateY(-50%);
    width: 45px;
    height: 20px;
    background: linear-gradient(to bottom, #804015, #98582a, #804015);
    box-shadow: 0 1px 3px rgba(0,0,0,0.4);
    border: 1px solid #502a10;
    z-index: 12;
    border-radius: 3px 10px 10px 3px / 50% 50% 50% 50%;
  }
  .stave-handle.left::before {
    right: calc(100% - 8px);
    border-radius: 10px 3px 3px 10px / 50% 50% 50% 50%;
  }
  .stave-handle.right::before {
    left: calc(100% - 8px);
    border-radius: 3px 10px 10px 3px / 50% 50% 50% 50%;
  }

  .scroll-container {
    background: #fdf6e3;
    max-height: 0;
    opacity: 0;
    padding-top: 0;
    padding-bottom: 0;
    overflow: hidden;
    border-radius: 8px;
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.25),
                inset 0 0 20px rgba(219, 206, 180, 0.5);
    width: clamp(280px, 60vw, 780px);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.5rem;
    border: 1px solid #d3c0a5;
    position: relative;
    z-index: 5;
    margin-top: -20px;
    margin-bottom: -20px;
  }

  .animate-scroll .scroll-container {
    animation: unrollParchment 1s cubic-bezier(0.68, -0.55, 0.27, 1.55) forwards;
    animation-delay: 0.5s;
  }

  /* Initial state for all content items within the scroll */
  .scroll-content-item {
    opacity: 0;
    transform: translateY(10px);
  }

  /* General animation for scroll content items (like the mode-buttons container) */
  .animate-scroll .mode-buttons.scroll-content-item {
    animation: fadeInContent 0.7s ease-out forwards;
    animation-delay: 1.2s; /* Buttons appear after logo and parchment unroll */
  }

  /* Specific combined animation for the logo */
  .animate-scroll .logo.scroll-content-item {
    /* 
      Animation shorthand: name | duration | timing-function | delay | iteration-count | direction | fill-mode | play-state 
      We're applying two animations: fadeInContent first, then float.
    */
    animation: 
      fadeInContent 0.7s ease-out 1.0s 1 forwards, /* name duration timing delay iteration fill-mode */
      float 6s ease-in-out 1.7s infinite;       /* name duration timing delay iteration (fill-mode 'none' is default for infinite) */
  }

  .scroll-container .logo { /* Base styles for logo - not animation related */
    width: clamp(100px, 20vw, 150px);
    height: auto;
    margin-bottom: 1rem;
    filter: drop-shadow(0 3px 4px rgba(0,0,0,0.15));
  }

  @keyframes float {
    0% { transform: translateY(0px); } /* Ensure float animation starts from the correct Y after fadeIn */
    50% { transform: translateY(-8px); }
    100% { transform: translateY(0px); }
  }

  .mode-buttons {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 1rem;
    width: 100%;
    max-width: 550px;
    margin-top: 0.5rem;
  }

  .mode-button {
    padding: 0.8rem 1.2rem;
    font-size: 1rem;
    text-align: left;
    background: #f5eeda;
    border: 1px solid #c8b89c;
    border-radius: 8px;
    color: #584c3a;
    cursor: pointer;
    transition: all 0.25s cubic-bezier(0.25, 0.8, 0.25, 1);
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    position: relative;
    overflow: hidden;
    font-family: 'Georgia', 'Times New Roman', serif;
    box-shadow: 0 2px 4px rgba(0,0,0,0.05);
  }

  .mode-button .title {
    font-weight: 600;
    color: #4a3f30;
    font-size: 1.1em;
    z-index: 1;
  }

  .mode-button .description {
    font-size: 0.88em;
    color: #7a6a54;
    opacity: 0.8;
    transform: translateY(0);
    transition: opacity 0.25s ease, transform 0.25s ease;
    z-index: 1;
    line-height: 1.3;
  }

  .mode-button:hover {
    background: #f0e6c8;
    transform: translateY(-3px) scale(1.02);
    box-shadow: 0 5px 15px rgba(88, 76, 58, 0.15);
    border-color: #8c7b62;
  }
  .mode-button:hover .description { opacity: 1; }
  .mode-button:active {
    transform: translateY(-1px) scale(1.01);
    box-shadow: 0 2px 8px rgba(88, 76, 58, 0.1);
  }

  .scroll-container::-webkit-scrollbar { width: 10px; }
  .scroll-container::-webkit-scrollbar-track {
    background: rgba(160, 140, 110, 0.2);
    border-radius: 5px;
  }
  .scroll-container::-webkit-scrollbar-thumb {
    background: #b4a284;
    border-radius: 5px;
    border: 2px solid #fdf6e3;
  }
  .scroll-container::-webkit-scrollbar-thumb:hover { background: #a0937d; }

  /* Keyframes for animations */
  @keyframes fadeInStaves {
    from {
      opacity: 0;
      transform: scale(0.95);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  @keyframes unrollParchment {
    0% {
      max-height: 0;
      opacity: 0;
      padding-top: 0;
      padding-bottom: 0;
    }
    30% {
      padding-top: 1rem;
      padding-bottom: 1rem;
      opacity: 0.5;
    }
    100% {
      max-height: calc(85vh - 2 * 40px - 2rem);
      opacity: 1;
      padding-top: 2.5rem;
      padding-bottom: 2.5rem;
    }
  }

  @keyframes fadeInContent {
    from {
      opacity: 0;
      transform: translateY(10px);
    }
    to {
      opacity: 1;
      transform: translateY(0px); /* Explicitly to 0px */
    }
  }

  /* Back to Vault Button */
  .back-to-vault-btn {
    position: absolute;
    top: 20px;
    left: 20px;
    padding: 0.5rem 1rem;
    background: var(--bg-secondary);
    color: var(--text-primary);
    border: 1px solid var(--border-color-medium);
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
    font-weight: 500;
    z-index: 20;
    transition: all 0.2s ease;
    opacity: 0;
    transform: translateY(-10px);
  }

  .animate-scroll .back-to-vault-btn {
    animation: fadeInButton 0.5s ease-out forwards;
    animation-delay: 0.8s;
  }

  .back-to-vault-btn:hover {
    background: var(--bg-hover-medium);
    border-color: var(--accent-primary);
    transform: translateY(-2px);
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  }

  .back-to-vault-btn:active {
    transform: translateY(0);
  }

  @keyframes fadeInButton {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }

</style>