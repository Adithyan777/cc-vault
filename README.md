# cc-vault

A lazygit-style TUI for browsing and managing your [Claude Code](https://claude.ai/claude-code) sessions.

![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-yellow)

![cc-vault screenshot](assets/screenshot.jpg)

## Features

- **Three-panel layout** — Projects, Sessions, and Preview side-by-side
- **Session preview** — Scrollable markdown-rendered conversation view
- **Resume sessions** — Press Enter to jump straight into a session with `claude --resume`
- **Rename** — Native integration with Claude Code's rename system (`/rename` compatible)
- **Search** — Full-text search across all session content
- **Bulk operations** — Select multiple sessions for delete or export
- **Prune** — Clean up empty sessions with zero messages
- **Export** — Save sessions as readable markdown files

## Install

```bash
# Clone and build
git clone https://github.com/Adithyan777/cc-vault.git
cd cc-vault
go build -o cc-vault .

# Move to PATH
sudo cp cc-vault /usr/local/bin/
```

Requires Go 1.25+ and [Claude Code](https://docs.anthropic.com/en/docs/claude-code) installed.

## Usage

```bash
cc-vault
```

Navigate with familiar vim-style keybindings:

| Key | Action |
|-----|--------|
| `j/k` `↑/↓` | Navigate within panel |
| `h/l` `←/→` | Switch panels |
| `Enter` | Resume selected session |
| `r` | Rename session |
| `d` | Delete session |
| `x` | Export to markdown |
| `Space` | Toggle select |
| `D` | Bulk delete selected |
| `X` | Bulk export selected |
| `P` | Prune empty sessions |
| `/` | Search sessions |
| `?` | Help |
| `q` | Quit |

## How it works

cc-vault reads Claude Code's local data directly from `~/.claude/`:

- **Projects** discovered from `~/.claude/projects/` directory structure
- **Sessions** parsed from JSONL files with fast partial reads (first 30 lines for metadata, last 8KB for custom titles)
- **Renames** written as native `custom-title` JSONL entries — fully compatible with `claude --resume "name"` and `/rename`

## Built with

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Terminal styling

## License

MIT
