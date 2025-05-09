<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import logo from '../assets/images/logo.png';

  const dispatch = createEventDispatcher();

  function setMode(mode: 'codex' | 'story' | 'library' | 'chat' | 'settings' | 'write') {
    dispatch('setmode', mode);
  }
</script>

<div class="mode-select">
  <div class="scroll-stave-assembly top-stave">
    <div class="stave-handle left"></div>
    <div class="stave-roller"></div>
    <div class="stave-handle right"></div>
  </div>

  <div class="scroll-container">
    <img src={logo} alt="Llore Logo" class="logo" />
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
  }

  /* Stave Assembly - Container for roller and handles */
  .scroll-stave-assembly {
    display: flex;
    align-items: center;
    /* Width now needs to accommodate roller AND horizontal handles */
    /* Parchment width + 2 * handle_cap_width + 2 * spindle_length */
    /* Let's make the assembly wide, and the roller will be sized based on parchment */
    width: clamp(400px, 80vw, 950px); /* Increased max width */
    position: relative;
    z-index: 10;
    flex-shrink: 0;
  }

  /* Stave Roller - The main horizontal bar */
  .stave-roller {
    height: 35px;
    /* This width should ideally match the parchment container width */
    /* We'll set it to grow and then ensure parchment matches, or set fixed width */
    flex-grow: 1; /* It will fill space between handles */
    min-width: clamp(200px, 50vw, 700px); /* Ensure it has a reasonable min width */
    background: linear-gradient(to right, #8B4513, #A0522D, #8B4513);
    border-radius: 5px; /* Less rounded ends as they meet handles */
    box-shadow: 0 3px 7px rgba(0, 0, 0, 0.4),
                inset 0 2px 3px rgba(255,255,255,0.1),
                inset 0 -2px 3px rgba(0,0,0,0.2);
    border: 1.5px solid #5c2e11;
    position: relative; /* For z-indexing if needed relative to handles */
    z-index: 9;
  }

  /* Stave Handle - The "cap" at the end of the roller */
  .stave-handle {
    width: 25px; /* Width of the cap itself along the roller's axis */
    height: 40px; /* Slightly taller/thicker than the roller for a distinct cap */
    background: linear-gradient(to right, #703816, #8B4513, #703816); /* Darker wood for cap */
    box-shadow: 0 2px 5px rgba(0,0,0,0.35),
                inset 0 1px 2px rgba(255,255,255,0.08),
                inset 0 -1px 2px rgba(0,0,0,0.12);
    border: 1.5px solid #4a2a0f;
    position: relative; /* Context for the spindle pseudo-element */
    flex-shrink: 0;
    z-index: 11; /* Ensure cap is visually distinct */
    display: flex; /* For centering spindle if it were a child */
    align-items: center;
  }
  .stave-handle.left {
    border-radius: 20px 3px 3px 20px; /* Rounded outer, flatter inner to meet roller */
    margin-right: -2px; /* Overlap roller slightly */
  }
  .stave-handle.right {
    border-radius: 3px 20px 20px 3px; /* Rounded outer, flatter inner */
    margin-left: -2px; /* Overlap roller slightly */
  }

  /* Spindle part of the handle using ::before, extending HORIZONTALLY */
  .stave-handle::before {
    content: '';
    position: absolute;
    top: 50%; /* Center vertically on the handle cap */
    transform: translateY(-50%);
    width: 45px;  /* LENGTH of the spindle (how much it sticks out) */
    height: 20px; /* THICKNESS of the spindle */
    background: linear-gradient(to bottom, #804015, #98582a, #804015); /* Turned wood spindle */
    box-shadow: 0 1px 3px rgba(0,0,0,0.4);
    border: 1px solid #502a10;
    z-index: 12; /* Above cap */
    /* Rounded ends for the spindle */
    /* border-radius: horizontal-radius / vertical-radius */
    border-radius: 3px 10px 10px 3px / 50% 50% 50% 50%; /* Pill shape with one end more rounded */
  }

  .stave-handle.left::before {
    right: calc(100% - 8px); /* Position spindle to the left of the cap, slight overlap */
    border-radius: 10px 3px 3px 10px / 50% 50% 50% 50%; /* Rounded end pointing left */
  }
  .stave-handle.right::before {
    left: calc(100% - 8px); /* Position spindle to the right of the cap, slight overlap */
    border-radius: 3px 10px 10px 3px / 50% 50% 50% 50%; /* Rounded end pointing right */
  }


  /* Scroll Container (Parchment) */
  .scroll-container {
    background: #fdf6e3;
    padding: 2.5rem 2rem;
    border-radius: 8px;
    overflow-y: auto;
    max-height: calc(85vh - 2 * 40px - 2rem); /* 40px is handle height, 2rem total vertical padding */
    box-shadow: 0 8px 25px rgba(0, 0, 0, 0.25),
                inset 0 0 20px rgba(219, 206, 180, 0.5);
    /* Width of parchment should be less than roller's min-width to ensure roller is visible around it */
    width: clamp(280px, 60vw, 780px);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.5rem;
    border: 1px solid #d3c0a5;
    position: relative;
    z-index: 5; /* Behind staves */
    margin-top: -20px; /* Pull parchment under top stave assembly slightly more */
    margin-bottom: -20px; /* Pull parchment under bottom stave assembly */
  }

  /* Ensure the roller's visible part (behind parchment) matches parchment width */
  /* This might require adjusting the roller's flex properties or explicit width
     if the assembly isn't perfectly sized by the parchment.
     For simplicity, the current flex-grow on roller makes it fill space
     between handles. The parchment sits on top. */


  .scroll-container .logo {
    width: clamp(100px, 20vw, 150px);
    height: auto;
    margin-bottom: 1rem;
    animation: float 6s ease-in-out infinite;
    filter: drop-shadow(0 3px 4px rgba(0,0,0,0.15));
  }

  @keyframes float {
    0% { transform: translateY(0); }
    50% { transform: translateY(-8px); }
    100% { transform: translateY(0); }
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

</style>