# Llore: Dynamic Codex Management for Fiction Writing

[![Go Version](https://img.shields.io/github/go-mod/go-version/YOUR_GITHUB_USERNAME/llore)](https://golang.org)
[![Wails Version](https://img.shields.io/badge/Wails-v2-brightgreen)](https://wails.io)
<!-- Add other badges if you set up CI/CD, license, etc. -->

**Llore** is a desktop application built with Wails (Go + Web Technologies) to help fiction writers dynamically manage their world-building codex (characters, locations, lore, etc.) with integrated Large Language Model (LLM) assistance and concepts inspired by Git-like version control.

**Note:** This is currently a functional **prototype** demonstrating core features.

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

## Core Features (Prototype)

*   **Cross-Platform Desktop App:** Built with Wails v2.
*   **Codex Entry Management:**
    *   Create new entries (e.g., Character, Location, Item, Lore).
    *   View a list of all entries.
    *   View and edit the details (Name, Type, Content) of an entry.
    *   Save changes to entries.
    *   Delete entries.
*   **LLM Integration (AWS Bedrock):**
    *   Generate initial content for an entry based on a user prompt via the AWS Bedrock API.
    *   Configurable AWS Region and Bedrock Model ID via `.env` file.
*   **Persistence:**
    *   Stores codex entries locally using an embedded **SQLite** database.
*   **Basic UI:** Simple list/detail view implemented with **Svelte** (or your chosen frontend framework).

## Technology Stack

*   **Framework:** [Wails v2](https://wails.io/)
*   **Backend Language:** [Go](https://go.dev/)
*   **Frontend Framework:** [Svelte](https://svelte.dev/) (or React/Vue/Preact/Vanilla based on initialization)
*   **UI:** HTML, CSS, JavaScript/TypeScript
*   **Database:** [SQLite](https://www.sqlite.org/index.html) (via `mattn/go-sqlite3`)
*   **LLM Integration:** [AWS SDK for Go v2](https://aws.github.io/aws-sdk-go-v2/docs/) (targeting Bedrock Runtime)
*   **Configuration:** `.env` files ([joho/godotenv](https://github.com/joho/godotenv))

## Screenshots (Optional)

*(Add screenshots of Llore here if available)*
<!--
![Screenshot 1](path/to/screenshot1.png)
![Screenshot 2](path/to/screenshot2.png)
-->

## Prerequisites

Before you begin, ensure you have the following installed:

1.  **Go:** Version 1.18 or higher. ([Installation Guide](https://go.dev/doc/install))
2.  **Node.js & npm:** Required for frontend dependencies. ([Installation Guide](https://nodejs.org/))
3.  **Wails CLI v2:** ([Installation Guide](https://wails.io/docs/gettingstarted/installation))
4.  **Git:** For cloning the repository. ([Installation Guide](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git))
5.  **AWS Account & Credentials:**
    *   An active AWS account.
    *   AWS credentials (Access Key ID and Secret Access Key) configured for programmatic access.
    *   Permissions to use AWS Bedrock Runtime and the specific models you intend to use.
    *   **Crucially:** Configure your AWS credentials securely. The recommended methods are setting environment variables (`AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` if needed) or using the shared credentials file (`~/.aws/credentials`). **Avoid committing credentials directly.**

## Setup & Installation

1.  **Clone the repository:**
    ```bash
    # Replace YOUR_GITHUB_USERNAME with your actual username
    git clone https://github.com/YOUR_GITHUB_USERNAME/llore.git
    cd llore
    ```

2.  **Install Go dependencies:**
    ```bash
    go mod tidy
    ```

3.  **Install frontend dependencies:**
    ```bash
    cd frontend
    npm install
    cd ..
    ```

4.  **Configure Environment:**
    *   Copy the example environment file:
        ```bash
        cp .env.example .env
        ```
        *(If you don't have a `.env.example`, copy the `.env` created by the setup script, ensuring sensitive AWS keys are NOT included if you commit it)*
    *   Edit the `.env` file:
        *   Set `DB_PATH` to your desired SQLite database file location (e.g., `./llore_data.db`).
        *   Set `AWS_REGION` to the AWS region where you have Bedrock access (e.g., `us-east-1`).
        *   Set `BEDROCK_MODEL_ID` to the specific Bedrock model you want to use (e.g., `anthropic.claude-instant-v1`, `amazon.titan-text-express-v1`). Find model IDs [here](https://docs.aws.amazon.com/bedrock/latest/userguide/model-ids.html).

5.  **Ensure AWS Credentials are configured** in your environment or AWS credentials file (as mentioned in Prerequisites).

## Configuration

Configuration is managed via the `.env` file in the project root:

*   `DB_PATH`: Filesystem path for the SQLite database file.
*   `AWS_REGION`: The AWS region for Bedrock API calls.
*   `BEDROCK_MODEL_ID`: The specific identifier for the Bedrock model to use for generation.
*   `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`: **Best practice is to set these via environment variables or the `~/.aws/credentials` file, not directly in `.env` if the repo is public.**

## Running Llore

### Development Mode

This mode provides hot-reloading for both the Go backend and the frontend.

1.  Navigate to the project root directory (`llore`).
2.  Run the command:
    ```bash
    wails dev
    ```
3.  The Llore application window will appear. Changes to Go files or frontend code will trigger automatic rebuilds and reloads.

### Building for Production

This compiles Llore into a native, self-contained executable for your platform.

1.  Navigate to the project root directory (`llore`).
2.  Run the command:
    ```bash
    wails build
    ```
3.  The executable will be placed in the `build/bin/` directory (e.g., `Llore.exe` on Windows, `Llore` on Linux/macOS).

## Basic Usage

1.  **Launch** Llore (either via `wails dev` or by running the built executable).
2.  The main window displays a **list of codex entries** on the left (initially empty).
3.  Click **"New Entry"** (or similar button) to create a new codex item.
4.  The **detail view/form** appears on the right. Enter a `Name` and `Type` (e.g., "Character", "Location", "Lore").
5.  **Generate Content:**
    *   Optionally, enter a prompt in the designated input field (e.g., "Describe a cynical space smuggler named Jax").
    *   Click the **"Generate"** (or similar) button.
    *   Llore will call AWS Bedrock, and the response will populate the `Content` field.
6.  **Edit Content:** Manually type or modify text in the `Content` area.
7.  **Save:** Click the **"Save"** button to persist the new entry or changes to an existing one. The entry list should update.
8.  **Select Entry:** Click an entry in the list to view/edit its details.
9.  **(Optional) Delete:** Select an entry and click a "Delete" button if implemented.

## Project Status

**Llore** is currently a **functional prototype (MVP)**.

**Implemented:**

*   Core Wails application structure (Go backend, Svelte frontend).
*   SQLite database integration for persistence.
*   CRUD operations for codex entries.
*   AWS Bedrock integration for text generation (using configured credentials and model).
*   Basic List/Detail UI.

**Not Yet Implemented / Future Goals:**

*   Full Git-like version control for entries (diffing, commit history, branching).
*   Retrieval-Augmented Generation (RAG) for context-aware LLM responses.
*   Support for multiple LLM providers (OpenAI, Google Gemini, local Ollama).
*   Bundled database option (e.g., PostgreSQL).
*   Advanced UI features (rich text editing, inter-entry linking, visualization).
*   Robust error handling and user feedback.
*   Testing (Unit, Integration, E2E).

## Future Work / Roadmap

*   [ ] Implement true version history storage for entries.
*   [ ] Add diff view to compare entry versions.
*   [ ] Integrate vector embeddings for RAG context retrieval.
*   [ ] Build settings UI for configuring multiple LLM APIs (Bedrock, OpenAI, etc.).
*   [ ] Implement Ollama integration for local LLM support.
*   [ ] Explore bundling PostgreSQL or offering it as an alternative DB.
*   [ ] Introduce a rich text editor for the content field.
*   [ ] Develop functionality for linking related codex entries.
*   [ ] Add comprehensive search and filtering capabilities.

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