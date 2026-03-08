<div align="center">

# Airi — AI OS Assistant

**An intelligent desktop assistant that watches your system, understands your workflow, and lets you talk to your computer like it already knows what you've been doing.**

[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat-square&logo=go&logoColor=white)](https://golang.org/)
[![React](https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=black)](https://reactjs.org/)
[![TypeScript](https://img.shields.io/badge/TypeScript-4.6-3178C6?style=flat-square&logo=typescript&logoColor=white)](https://www.typescriptlang.org/)
[![Wails](https://img.shields.io/badge/Wails-v2-FF3E00?style=flat-square)](https://wails.io/)
[![Claude AI](https://img.shields.io/badge/Claude-3_Haiku-blueviolet?style=flat-square)](https://www.anthropic.com/)
[![SQLite](https://img.shields.io/badge/SQLite-003B57?style=flat-square&logo=sqlite&logoColor=white)](https://sqlite.org/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20|%20macOS-lightgrey?style=flat-square)]()
[![License](https://img.shields.io/badge/License-MIT-green?style=flat-square)](LICENSE)

</div>

---

## What is Airi?

Airi is a native desktop AI assistant built with **Go + React** via the [Wails](https://wails.io/) framework. Unlike typical AI chat apps, Airi is deeply integrated with your operating system — it passively monitors your file activity, active applications, and clipboard in the background, then uses that context to give you **genuinely useful, personalized answers**.

Ask it *"What did I work on this afternoon?"* and it knows. Ask it *"Find all the Python files I touched this week"* and it does it. Ask it to read, write, or inspect files on your system and it executes — intelligently.

It combines **passive OS telemetry**, **local SQLite storage**, **Claude AI** for reasoning, and **Google Gemini** for activity summarization into a single lightweight native application.

---

## Features

### Passive Activity Monitoring

Airi runs silently in the background, building a picture of your workday:

| Monitor | What it tracks | Interval |
|---|---|---|
| **File Monitor** | Create, modify, delete, rename events across your key directories | Real-time (fsnotify) |
| **Window Tracker** | Active application and window title, with time-spent per app | Every 5 seconds |
| **Clipboard Monitor** | Content changes (first 200 chars, SHA256 deduplicated) | Every 2 seconds |
| **Idle Detector** | System idle state to pause tracking when you're away | On demand |

### Intelligent Natural Language Querying

Airi uses **intent detection** (via Claude) to route your message to the right handler automatically:

- *"What apps did I use this morning?"* — Queries window events, summarizes with Gemini
- *"Show me files I edited yesterday"* — Queries file events, formats a timeline
- *"What was on my clipboard earlier?"* — Retrieves clipboard history
- *"Help me debug this code"* — Routes directly to Claude as a general query

### AI-Powered File System Operations

Claude can perform structured actions on your file system based on your requests:

- **Read** a file and summarize or explain it
- **Write** content to a file
- **Delete** a file (with confirmation)
- **Search** files by name pattern or extension, recursively
- **Inspect** directories and visualize folder trees
- **List and kill** running processes

### Context-Aware Responses

Every message sent to Claude includes a **current system state snapshot** — active application, window title, and idle time — so responses are relevant to what you're doing right now.

### Modern Chat UI

- Markdown rendering with **syntax-highlighted code blocks**
- Structured card output for file operations and directory listings
- Real-time thinking indicator
- Clean, minimal design with a gradient header
- Error-safe markdown rendering with React error boundaries

---

## Architecture

```
+-------------------------------------------------------------+
|                        Frontend (React)                     |
|  +-------------+   +--------------+   +-------------------+ |
|  |  Chat UI    |   | Markdown     |   |  Action Cards     | |
|  |  (App.tsx)  |   | Renderer     |   |  (Files/Procs)    | |
|  +------+------+   +--------------+   +-------------------+ |
+---------|---------------------------------------------------+
          |  Wails Bridge (Go <-> WebView2)
+---------v---------------------------------------------------+
|                        Backend (Go)                         |
|                                                             |
|  SendUserMessage()                                          |
|       |                                                     |
|       +---> Intent Detector (Claude API)                    |
|                   |                                         |
|            +------v-----------+   +---------------------+  |
|            |  Activity Query  |   |   General Query     |  |
|            |  Handler         |   |   (Direct to Claude)|  |
|            |  + Gemini Summ.  |   +---------------------+  |
|            +------------------+                             |
|                                                             |
|       +---> Current State Snapshot                          |
|       +---> Claude API --> JSON Action --> Execute          |
|                                                             |
|  +------------------------------------------------------+   |
|  |           SQLite Database (local)                    |   |
|  |  file_events | window_events | clipboard_events      |   |
|  +------------------------------------------------------+   |
+-------------------------------------------------------------+
```

---

## Tech Stack

| Layer | Technology | Purpose |
|---|---|---|
| Desktop Framework | [Wails v2](https://wails.io/) | Go backend + WebView2 frontend bridge |
| Backend Language | Go 1.24 | Core application logic, OS integration |
| Frontend Framework | React 18 + TypeScript | Chat UI |
| Build Tool | Vite 3 | Frontend bundling and HMR |
| Database | SQLite (modernc.org/sqlite) | Local activity event storage |
| File Watching | fsnotify | Real-time file system events |
| System Info | gopsutil v4 | Process listing |
| AI — Reasoning | [Claude 3 Haiku](https://www.anthropic.com/) (Anthropic) | Intent detection, conversational AI, action planning |
| AI — Summarization | [Gemini 2.0 Flash](https://deepmind.google/technologies/gemini/) (Google) | Activity summarization |
| Markdown | react-markdown + rehype-highlight | Formatted AI output rendering |
| Icons | lucide-react | UI icons |

---

## Getting Started

### Prerequisites

- [Go](https://golang.org/dl/) 1.24+
- [Node.js](https://nodejs.org/) 18+ and npm
- [Wails CLI](https://wails.io/docs/gettingstarted/installation)
- A [Claude API key](https://console.anthropic.com/) (Anthropic)

Install the Wails CLI:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

### Setup

1. **Clone the repository**

   ```bash
   git clone https://github.com/muhamadnurasyraaf/airi-desktop.git
   cd airi-desktop/airi
   ```

2. **Set your API key**

   Create a `.env` file in the `airi/` directory:

   ```env
   CLAUDE_API_KEY=your_anthropic_api_key_here
   ```

3. **Install frontend dependencies**

   ```bash
   cd frontend && npm install && cd ..
   ```

4. **Run in development mode**

   ```bash
   wails dev
   ```

   This starts a live-reload development server. You can also open `http://localhost:34115` in a browser to access the frontend with DevTools.

### Build

**Windows:**

```bash
wails build -platform windows/amd64
```

**macOS:**

```bash
wails build -platform darwin/universal
```

The output binary will be placed in `build/bin/`.

---

## How It Works

### 1. Startup

On launch, Airi initializes the SQLite database and starts four background goroutines:

- File monitor (watches `Documents`, `Desktop`, `Downloads`, `Projects`, etc.)
- Window tracker (polls the active window every 5 seconds)
- Clipboard monitor (polls clipboard every 2 seconds using SHA256 change detection)
- Cleanup scheduler (purges events older than 30 days, runs daily at 3 AM)

### 2. Message Flow

When you send a message:

1. **Intent Detection** — Claude classifies whether the message is about your activity history or a general request.
2. **Context Building** — If activity-related, Airi queries SQLite for relevant events and builds a formatted timeline. A current system state snapshot (active app, window title, idle time) is always included.
3. **Claude API Call** — The prompt with full context is sent to Claude 3 Haiku.
4. **Action Parsing** — If Claude returns a JSON action block, Airi parses and executes it on the OS level.
5. **Response** — The result is returned to the frontend, rendered as markdown or structured cards.

### 3. App Categorization

Applications are automatically categorized into buckets — Development, Browser, Productivity, Communication, Entertainment, Design — for cleaner activity summaries.

---

## Database Schema

Airi stores all events locally in a SQLite database. No data leaves your machine (except API calls to Claude/Gemini).

```sql
-- File system events
CREATE TABLE file_events (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp DATETIME,
    filepath  TEXT,
    action    TEXT  -- 'created' | 'modified' | 'deleted' | 'renamed'
);

-- Active window tracking
CREATE TABLE window_events (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp        DATETIME,
    app_name         TEXT,
    window_title     TEXT,
    duration_seconds INTEGER,
    app_category     TEXT
);

-- Clipboard history
CREATE TABLE clipboard_events (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp       DATETIME,
    content_preview TEXT,  -- first 200 characters
    content_length  INTEGER
);

-- Idle state events
CREATE TABLE idle_events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    timestamp  DATETIME,
    event_type TEXT
);
```

Data is retained for **30 days** and then automatically purged.

---

## Project Structure

```
airi/
├── main.go                # Entry point — Wails app setup
├── app.go                 # Core: SendUserMessage, action executor
├── wails.json             # Wails project config
├── go.mod                 # Go module and dependencies
│
├── database.go            # SQLite schema, CRUD, migrations
├── file_monitor.go        # fsnotify-based file watcher
├── window_tracker.go      # Active window polling (Windows/macOS)
├── clipboard_monitor.go   # Clipboard change detection
├── idle_detector.go       # System idle time (Windows/macOS)
├── query_handler.go       # Activity querying + Gemini summarization
├── intent_detector.go     # Claude-based intent classification
├── current_state.go       # System state snapshot
├── app_categorizer.go     # App category classification
├── cleanup_scheduler.go   # Scheduled 30-day data purge
├── filesystem.go          # File search, read, tree inspection
│
├── frontend/
│   ├── src/
│   │   ├── App.tsx        # Chat UI, action card rendering
│   │   ├── main.tsx       # React entry point
│   │   ├── App.css        # Application styles
│   │   └── style.css      # Global styles
│   ├── index.html
│   └── package.json
│
└── build/
    ├── appicon.png
    └── windows/
        ├── icon.ico
        └── installer/
```

---

## Privacy & Security

- **All activity data is stored locally** in a SQLite database on your machine.
- **No cloud sync** — your file events, window history, and clipboard content never leave your device, except as context in Claude/Gemini API calls.
- **API calls are opt-in** — only triggered when you send a message.
- Store your API key in a `.env` file and **never commit it to version control**.

---

## Platform Support

| Feature | Windows | macOS |
|---|---|---|
| File monitoring | Yes | Yes |
| Window tracking | Yes (PowerShell) | Yes (osascript) |
| Clipboard monitoring | Yes (PowerShell) | Yes (pbpaste) |
| Idle detection | Yes (Windows API) | Yes (ioreg) |
| Build output | `.exe` | `.app` |

Linux is not currently supported for window tracking or idle detection.

---

## Roadmap

- [ ] Linux support (window tracking via `xdotool`)
- [ ] Tray icon with quick-access chat
- [ ] Screenshot capture and visual context
- [ ] Plugin system for custom monitors
- [ ] Local LLM support (Ollama integration)
- [ ] Export activity reports to PDF/Markdown

---

## Author

**Asyraaf** — [masyraaf14@gmail.com](mailto:masyraaf14@gmail.com)

Built as a personal project to explore the intersection of AI agents, OS-level system integration, and native desktop development with Go.

---

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.

---

<div align="center">

Built with [Wails](https://wails.io/) · Powered by [Claude](https://www.anthropic.com/) · Summarized by [Gemini](https://deepmind.google/technologies/gemini/)

</div>
