# Llore: Dynamic Codex Management for Fiction Writing

[![Go Version](https://img.shields.io/github/go-mod/go-version/YOUR_GITHUB_USERNAME/llore)](https://golang.org)
[![Wails Version](https://img.shields.io/badge/Wails-v2-brightgreen)](https://wails.io)
<!-- Add other badges if you set up CI/CD, license, etc. -->

**Llore** is a modern desktop application built with Wails (Go + Web Technologies) to help fiction writers dynamically manage their world-building codex (characters, locations, lore, etc.) with integrated Large Language Model (LLM) assistance. With its sleek interface featuring gradient backgrounds and silver accents, Llore provides a beautiful and intuitive environment for organizing your creative world.

## Table of Contents

-   [Motivation](#motivation)
-   [Core Features (Prototype)](#core-features-prototype)
-   [Technology Stack](#technology-stack)
-   [Screenshots (Optional)](#screenshots-optional)
-   [Prerequisites](#prerequisites)
-   [Setup & Installation](#setup--installation)
-   [Configuration](#configuration)
-   [Running Llore](#running-llore)
    -   [Development Mode](#development-mode)
    -   [Building for Production](#building-for-production)
-   [Basic Usage](#basic-usage)
-   [Project Status](#project-status)
-   [Future Work / Roadmap](#future-work--roadmap)
-   [Contributing](#contributing)
-   [License](#license)

## Motivation

Fiction writers often juggle vast amounts of information about their worlds, characters, and plots. Traditional notes or static wiki-like tools can become unwieldy. **Llore** aims to provide a more dynamic and intelligent solution by:

1.  **Leveraging LLMs:** Assisting writers in generating initial ideas, descriptions, or expanding on existing entries.
2.  **Structured Data:** Storing codex entries in a robust database for easy querying and management.
3.  **Cross-Platform:** Using Wails to create a native desktop experience for Windows, macOS, and Linux.
4.  **(Future Goal) Version Control:** Implementing a Git-like system for tracking changes to codex entries, allowing writers to experiment without losing previous versions.
5.  **(Future Goal) Context-Aware Generation:** Using Retrieval-Augmented Generation (RAG) to make LLM responses more relevant to the writer's existing codex.

## Core Features

*   **Modern Desktop Experience:**
    *   Beautiful gradient backgrounds and silver accents for a premium feel
    *   Responsive and intuitive interface
    *   Smooth transitions and animations
    *   Cross-platform support via Wails v2

*   **Vault System:**
    *   Create and manage multiple vaults for different projects
    *   Easy vault switching and organization
    *   Secure local storage of your creative content

*   **Codex Entry Management:**
    *   Create and organize entries (Characters, Locations, Items, Lore)
    *   Rich text editing with proper formatting
    *   Quick entry filtering and search
    *   Bulk operations for efficient management

*   **AI Integration:**
    *   OpenRouter API support for access to multiple AI models
    *   Context-aware content generation
    *   AI-assisted entry creation and enhancement
    *   Customizable model selection

*   **Story Import:**
    *   Import stories and automatically extract relevant entries
    *   Smart parsing of characters, locations, and items
    *   Bulk entry creation from imported content

*   **Library Management:**
    *   Organize and view your story files
    *   Easy access to your creative content
    *   File preview and management

*   **Lore Chat:**
    *   Interactive chat interface for exploring your world
    *   Context-aware responses based on your codex
    *   Save chat insights directly to your codex

## Technology Stack

*   **Framework:** [Wails v2](https://wails.io/)
*   **Backend Language:** [Go](https://go.dev/)
*   **Frontend Framework:** [Svelte](https://svelte.dev/) (or React/Vue/Preact/Vanilla based on initialization)
*   **UI:** HTML, CSS, JavaScript/TypeScript
*   **Database:** [SQLite](https://www.sqlite.org/index.html) (via `mattn/go-sqlite3`)
*   **LLM Integration:** [OpenRouter API](https://openrouter.ai/docs) for access to multiple AI models
*   **Configuration:** Environment-based configuration

## Screenshots (Optional)

*(Add screenshots of Llore here if available)*
<!--
![Screenshot 1](path/to/screenshot1.png)
![Screenshot 2](path/to/screenshot2.png)
-->

## Requirements

- **Go** (>=1.20): [Download Go](https://go.dev/dl/)
- **Node.js & npm** (for frontend): [Download Node.js](https://nodejs.org/)
- **Wails CLI**: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`
- **C Compiler (for CGO/go-sqlite3)**:
    - **Windows:** Install [MinGW-w64](https://www.mingw-w64.org/) or [TDM-GCC](https://jmeubank.github.io/tdm-gcc/)
    - **macOS:** Xcode Command Line Tools (`xcode-select --install`)
    - **Linux:** `sudo apt install build-essential`

### Installing MinGW-w64 on Windows

1. Download the latest installer from the [MinGW-w64 website](https://www.mingw-w64.org/downloads/).
2. Run the installer and follow the prompts (default options are fine).
3. Add the `bin` directory (e.g., `C:\Program Files\mingw-w64\...\bin`) to your PATH environment variable.
4. Restart your terminal after installation.

## Prerequisites

Before you begin, ensure you have the following installed:

1.  **Go:** Version 1.18 or higher ([Installation Guide](https://go.dev/doc/install))
2.  **Node.js & npm:** Required for frontend dependencies ([Installation Guide](https://nodejs.org/))
3.  **Wails CLI v2:** ([Installation Guide](https://wails.io/docs/gettingstarted/installation))
4.  **Git:** For cloning the repository ([Installation Guide](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git))
5.  **Python 3.x:** For icon generation and utilities ([Installation Guide](https://www.python.org/downloads/))
6.  **OpenRouter Account:**
    *   Sign up at [OpenRouter](https://openrouter.ai/)
    *   Get your API key for AI model access

## Setup & Installation

1.  **Clone the repository:**
    ```bash
    git clone https://github.com/YOUR_GITHUB_USERNAME/llore.git
    cd llore
    ```

2.  **Install a C compiler (required for go-sqlite3):**
    - **Windows:**
      Download and install MinGW-w64 from [mingw-w64.org](https://www.mingw-w64.org/downloads/), then add its `bin` directory to your PATH.
      ```powershell
      # Example: (update path as needed)
      setx PATH "%PATH%;C:\Program Files\mingw-w64\x86_64-<version>\mingw64\bin"
      ```
    - **macOS:**
      ```sh
      xcode-select --install
      ```
    - **Linux:**
      ```sh
      sudo apt install build-essential
      ```

3.  **Install Go dependencies:**
    ```bash
    go mod tidy
    go install github.com/wailsapp/wails/v2/cmd/wails@latest
    ```

3.  **Install frontend dependencies:**
    ```bash
    cd frontend
    npm install
    cd ..
    ```

## Configuration

Configuration is done through the `~/.llore` folder under `config.json`.

## Running Llore

> **Note:** Llore uses [go-sqlite3](https://github.com/mattn/go-sqlite3), which requires CGO to be enabled. Ensure you have a C compiler installed and run builds with `CGO_ENABLED=1` (default on most systems if a C compiler is present).

### Development Mode

This mode provides hot-reloading for both the Go backend and the frontend.

1.  Navigate to the project root directory (`llore`).
2.  Run the command:
    ```powershell
    set CGO_ENABLED=1
    wails dev
    ```
    (On macOS/Linux, use `export CGO_ENABLED=1` instead of `set`.)
3.  The Llore application window will appear. Changes to Go files or frontend code will trigger automatic rebuilds and reloads.

### Building for Production

This compiles Llore into a native, self-contained executable for your platform.

1.  Navigate to the project root directory (`llore`).
2.  Run the command:
    ```powershell
    set CGO_ENABLED=1
    wails build
    ```
    (On macOS/Linux, use `export CGO_ENABLED=1` instead of `set`.)
3.  The executable will be placed in the `build/bin/` directory (e.g., `Llore.exe` on Windows, `Llore` on Linux/macOS).

---

## Troubleshooting

### Error: CGO_ENABLED=0 / go-sqlite3 requires cgo to work

If you see an error like:
```
failed to initialize codex database: failed to ping database ...: Binary was compiled with 'CGO_ENABLED=0', go-sqlite3 requires cgo to work. This is a stub
```
- Make sure you have a C compiler installed (see Requirements above).
- Always run builds and development with `CGO_ENABLED=1`.
- On Windows, use `set CGO_ENABLED=1` before `wails dev` or `wails build`.
- On macOS/Linux, use `export CGO_ENABLED=1`.

## Basic Usage

1.  **First Launch:**
    * Start Llore via `wails dev` or the built executable
    * Create or select a vault for your project
    * The main interface will show various modes: Codex, Story Import, Library, Lore Chat, and Settings

2.  **Configure OpenRouter:**
    * Get your API key from [OpenRouter](https://openrouter.ai/)
    * Configure it in the Settings page of the application

3.  **Managing Entries:**
    * In Codex mode, use the sidebar to view and manage entries
    * Click "+ New Entry" to create a character, location, item, or lore entry
    * Fill in the name, type, and content
    * Use the AI generation button for creative assistance
    * Save your changes

4.  **Importing Stories:**
    * Use Story Import mode to analyze existing writing
    * Select a story file from your computer
    * Review and approve the automatically extracted entries
    * Import them directly into your codex

5.  **Using Lore Chat:**
    * Enter Lore Chat mode for interactive exploration
    * Ask questions about your world and characters
    * Save interesting responses directly to your codex
    * The AI will consider your existing entries for context

6.  **Managing Files:**
    * Use Library mode to organize your story files
    * Preview and edit file contents
    * Keep your creative work organized

## Project Status

**Llore** is a fully functional application with a modern, polished interface.

**Implemented Features:**

*   **Core Architecture:**
    * Go backend with Wails v2 integration
    * Svelte frontend with modern UI components
    * SQLite database for efficient data storage
    * Multi-vault support for project organization

*   **User Interface:**
    * Modern gradient and silver styling
    * Responsive and intuitive layout
    * Smooth transitions and animations
    * Dark theme optimized for creative work

*   **AI Integration:**
    * OpenRouter API integration
    * Multiple AI model support
    * Context-aware content generation
    * Interactive Lore Chat system

*   **Content Management:**
    * Comprehensive CRUD operations for entries
    * Story import and analysis
    * Library file management
    * Bulk operations support

## Future Work / Roadmap

*   [ ] Enhanced AI Features:
    * [ ] Vector embeddings for improved context awareness
    * [ ] Local model support via Ollama
    * [ ] Custom AI model fine-tuning options

*   [ ] Advanced Content Management:
    * [ ] Version control for entries
    * [ ] Rich text editor with formatting
    * [ ] Entry linking and relationship mapping
    * [ ] Advanced search and filtering

*   [ ] Collaboration Features:
    * [ ] Multi-user support
    * [ ] Real-time collaboration
    * [ ] Change tracking and history

*   [ ] Data Management:
    * [ ] Cloud backup options
    * [ ] Import/export in multiple formats
    * [ ] Data migration tools

*   [ ] UI Enhancements:
    * [ ] Customizable themes
    * [ ] Visualization tools for world-building
    * [ ] Mobile-responsive design
    * [ ] Accessibility improvements

## Contributing

Contributions to Llore are welcome! If you'd like to contribute, please:

1.  Fork the repository (`https://github.com/YOUR_GITHUB_USERNAME/llore.git`).
2.  Create a new branch (`git checkout -b feature/your-feature-name`).
3.  Make your changes.
4.  Commit your changes (`git commit -am 'Add some feature'`).
5.  Push to the branch (`git push origin feature/your-feature-name`).
6.  Open a Pull Request against the main repository.

Please report bugs or suggest features using the GitHub Issues tab on the main repository.

## License

This project is licensed under the [MIT License](LICENSE) - see the LICENSE file for details.