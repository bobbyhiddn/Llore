# Llore: Your Dynamic Codex for Fiction Writing

![Llore Logo](frontend/src/assets/images/logo.png)

**Llore** is a modern, cross-platform desktop application designed to be the ultimate world-building companion for fiction writers. Built with Go and Svelte using the Wails framework, Llore provides a beautiful and intuitive environment to manage your creative universe—from characters and locations to complex lore—all enhanced with powerful, integrated Large Language Model (LLM) assistance.

Llore's core philosophy is to combine structured data management with the creative power of AI, using Retrieval-Augmented Generation (RAG) to ensure the AI understands the unique context of *your* world.

## Table of Contents

- [Core Features](#core-features)
- [How It Works: The RAG Engine](#how-it-works-the-rag-engine)
- [Technology Stack](#technology-stack)
- [Getting Started](#getting-started)
  - [Prerequisites](#prerequisites)
  - [Installation](#installation)
  - [Configuration](#configuration)
- [How to Use Llore](#how-to-use-llore)
  - [1. The Vault System](#1-the-vault-system)
  - [2. The Codex](#2-the-codex)
  - [3. Story Import & Analysis](#3-story-import--analysis)
  - [4. Lore Chat](#4-lore-chat)
  - [5. The Library & Templates](#5-the-library--templates)
  - [6. Weaving](#6-weaving)
- [Project Status](#project-status)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)

## Core Features

Llore is more than just a notebook; it's an intelligent system for creative writing.

*   **✨ Modern Desktop Experience:** A sleek, responsive interface with gradient backgrounds and silver accents, optimized for dark mode to keep you in the creative zone.
*   **🗄️ Vault System:** Organize your work by creating separate, self-contained "vaults" for each of your projects. Each vault holds its own codex, library, and chat history.
*   **📚 Codex Management:** Create, read, update, and delete entries for your **Characters, Locations, Items, and Lore**. A powerful search and filtering system makes finding information effortless.
*   **🧠 Multi-Provider AI Integration:** Llore is not tied to a single AI provider. It supports:
    *   **OpenRouter:** Access a wide variety of models from different providers.
    *   **OpenAI:** Use GPT models directly via your API key.
    *   **Google Gemini:** Leverage Gemini models for generation and high-quality embeddings.
    *   **Ollama (Local):** Run models like Llama 3 or Mistral completely offline on your own machine for ultimate privacy and control.
*   **🤖 Context-Aware AI:**
    *   **RAG-Powered Chat:** Chat with an AI that has read your codex. Ask questions like "What is the relationship between Alice and Bob?" or "Summarize the history of the Sunstone," and get answers based on your lore.
    *   **AI-Assisted Creation:** Generate new entry ideas, expand on existing descriptions, or brainstorm plot points with the click of a button.
*   **📄 Story Import & Analysis:** Paste or upload your manuscript, and Llore's AI will automatically read it, identify key entities, and suggest new codex entries. It can also intelligently merge new information into existing entries.
*   **✍️ Library & Templates:** Store your manuscript files, notes, and other documents directly within your project vault. Create reusable templates for character sheets, chapter outlines, and more.
*   **🕸️ Weaving:** A unique feature that allows you to drag a codex entry onto an open document. Llore's AI will then "weave" that entry's information into the text at your cursor position, creating natural-sounding prose.

## How It Works: The RAG Engine

Llore's "magic" comes from its Retrieval-Augmented Generation (RAG) system, which gives the AI long-term memory of your world.

```mermaid
flowchart TD
    subgraph UserInteraction["User Interaction"]
        A[User asks: "What does Alice think of the Sunstone?"]
    end

    subgraph BackendProcessing["Llore's Backend"]
        B[Query is converted to a vector embedding]
        C{Codex Database}
        C -- "Finds entries for 'Alice' & 'Sunstone' via vector similarity" --> D[Relevant Context is Retrieved]
        D -- "Context + Original Query" --> E[A new prompt is built for the LLM]
        E --> F{LLM (OpenAI, Gemini, Ollama, etc.)}
        F --> G[LLM generates a context-aware answer]
    end

    subgraph UIResponse["Response to User"]
        G --> H["AI Responds: 'Based on your lore, Alice believes the Sunstone is a dangerous artifact... she acquired it from...'"]
    end
```
1.  **Embedding:** When you create or update a codex entry, its content is converted into a numerical representation (a "vector embedding") and stored.
2.  **Retrieval:** When you ask a question, your query is also converted into a vector. Llore performs a semantic search to find the most relevant codex entries by comparing your query's vector to all the entry vectors in the database.
3.  **Augmentation:** The content from the most relevant entries is collected and injected as context into a new, detailed prompt for the LLM.
4.  **Generation:** The LLM receives the augmented prompt and generates a response that is directly informed by your world's specific lore, not just its general knowledge.

## Technology Stack

*   **Core Framework:** [Wails v2](https://wails.io/)
*   **Backend:** [Go](https://go.dev/)
*   **Frontend:** [Svelte](https://svelte.dev/) & TypeScript
*   **Database:** [SQLite](https://www.sqlite.org/index.html) for data and vector storage.
*   **AI Providers:**
    *   [OpenRouter API](https://openrouter.ai/docs)
    *   [OpenAI API](https://platform.openai.com/docs)
    *   [Google Gemini API](https://ai.google.dev/)
    *   [Ollama](https://ollama.com/) (for local models)

## Getting Started

### Prerequisites

Ensure you have the following installed on your system:
1.  **Go** (version 1.20 or higher)
2.  **Node.js** and **npm**
3.  **Wails CLI v2:** Run `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
4.  **Git**
5.  **Python 3.x** (used by a build script for icon generation)
6.  **API Keys (Optional):**
    *   [OpenRouter API Key](https://openrouter.ai/)
    *   [OpenAI API Key](https://platform.openai.com/api-keys)
    *   [Gemini API Key](https://makersuite.google.com/app/apikey)

### Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/bobbyhiddn/llore.git
    cd llore
    ```

2.  **Install backend dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Install frontend dependencies:**
    ```bash
    cd frontend
    npm install
    cd ..
    ```

### Configuration

-   On first launch, Llore creates a configuration file at `~/.llore/config.json`.
-   API keys, model preferences, and the active AI mode (e.g., `local`, `openai`) can be easily managed from the **Settings** page within the application itself.

## How to Use Llore

### Running the App

*   **For Development (with hot-reloading):**
    ```bash
    wails dev
    ```
*   **To Build a Production Executable:**
    ```bash
    wails build
    ```
    The final application will be in the `build/bin/` directory.

### 1. The Vault System
On first launch, you'll be prompted to either **Create a New Vault** or **Select an Existing Vault**. A vault is simply a folder on your computer that will contain all the data for a specific project.

### 2. The Codex
This is the heart of your world.
-   Use the sidebar to navigate between **Characters, Locations, Items,** and **Lore**.
-   Click **"+ New Entry"** to create an entry.
-   Use the **AI Generate** button within the editor to get help writing or expanding content.
-   Your changes are saved automatically. Embeddings for the RAG system are generated in the background whenever you create or update an entry.

### 3. Story Import & Analysis
-   Navigate to the **Story Import** tab.
-   Paste or upload a text file containing your manuscript or notes.
-   The AI will process the text and present you with a list of suggested new entries and updates to existing ones.
-   Review the suggestions and import them into your codex with a single click.

### 4. Lore Chat
-   Go to the **Lore Chat** tab.
-   Ask questions about your world. The AI will use the RAG engine to provide answers based on your codex.
-   If the AI generates a new piece of lore you like, you can easily save the insight directly to a new or existing codex entry.

### 5. The Library & Templates
-   The **Library** tab shows all non-codex files in your vault (e.g., `.txt`, `.md` files). You can use it to store and edit drafts.
-   The **Templates** folder within your vault can hold Markdown files for reusable structures like character sheets.

### 6. Weaving
This is a powerful editing feature.
1.  Open a document in the Library.
2.  Drag a codex entry from the sidebar and drop it onto the text editor.
3.  The AI will generate and insert text that seamlessly weaves the dropped entry's information into your narrative at the cursor's location.

## Project Status

**Llore is fully functional and in active development.**

**Implemented:**
*   ✅ Multi-Vault Project System
*   ✅ Full CRUD for Codex Entries (Characters, Locations, Items, Lore)
*   ✅ Multi-Provider AI Support (OpenRouter, OpenAI, Gemini, Ollama)
*   ✅ RAG-Powered Lore Chat
*   ✅ Automated Story Import and Entity Extraction
*   ✅ Intelligent Content Merging for existing entries
*   ✅ Background Vector Embedding Generation
*   ✅ Library and Template File Management

## Roadmap

We have exciting plans for the future of Llore!

*   **Advanced Content Management:**
    *   [ ] Rich text editor with advanced formatting.
    *   [ ] Direct linking between entries (e.g., `@CharacterName`).
    *   [ ] Graph visualization of relationships between entries.
    *   [ ] Git-like version control for codex entries.
*   **Enhanced AI Features:**
    *   [ ] Proactive suggestions (e.g., "This character lacks a defined goal. Would you like to brainstorm one?").
    *   [ ] AI-powered consistency checking across your lore.
*   **UI/UX Enhancements:**
    *   [ ] Customizable themes.
    *   [ ] Improved accessibility.
*   **Data Management:**
    *   [ ] More robust import/export options (e.g., JSON, CSV).
    *   [ ] Optional cloud backup solutions.

## Contributing

Contributions are welcome! If you'd like to help improve Llore, please follow these steps:

1.  Fork the repository.
2.  Create a new branch (`git checkout -b feature/your-awesome-feature`).
3.  Make your changes.
4.  Commit your changes (`git commit -m 'Add some awesome feature'`).
5.  Push to the branch (`git push origin feature/your-awesome-feature`).
6.  Open a Pull Request.

Please use the [GitHub Issues](https://github.com/bobbyhiddn/llore/issues) tab to report bugs or suggest new features.

## License

This project is licensed under the **MIT License**. See the [LICENSE](LICENSE) file for details.