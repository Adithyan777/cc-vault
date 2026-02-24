# cc-vault

A lazygit-style three-panel TUI for browsing, previewing, searching, renaming, deleting, and exporting Claude Code sessions.

## Tech Stack

- **Go 1.26** with **Bubble Tea** (TUI framework), **Lip Gloss** (styling), **Bubbles** (components)
- Build: `go build -o cc-vault .`

## Architecture

```
main.go                          Entry point. Runs TUI, execs `claude --resume` on Enter.
internal/
  claude/                        Data layer - reads Claude Code's filesystem
    config.go                    Reads ~/.claude.json for per-project stats (cost, tokens, duration)
    projects.go                  Discovers projects from ~/.claude/projects/, decodes encoded paths
    sessions.go                  Parses session JSONL files. Quick scan (first 30 lines) for
                                 title/date/branch. Tail scan (last 8KB) for custom-title.
                                 Lazy conversation count for prune.
    messages.go                  Loads all messages for preview/export. Parses JSONL message types.
  storage/
    names.go                     Legacy session-names.json (unused - kept for reference)
  tui/                           UI layer
    app.go                       Main Bubble Tea model. Panel navigation, all keybindings, state.
    styles.go                    Lip Gloss theme. Pink user, green agent, purple primary, cyan secondary.
    projects.go                  Left panel: project list with scroll
    sessions.go                  Middle panel: session list with date, pin indicator, select marker
    preview.go                   Right panel: scrollable message preview with cached rendered lines
    markdown.go                  Lightweight markdown renderer (headers, code blocks, bullets, word wrap)
    dialog.go                    Modal dialogs (delete confirm, bulk delete, prune, rename, help)
    search.go                    Full-text search across session JSONL content. Session file deletion.
```

## Key Design Decisions

### Performance
- **Session scanning**: `scanSessionQuick()` reads only first ~30 lines per file (title, date, branch). Never reads entire files on boot.
- **Custom titles**: `readCustomTitle()` reads last 8KB of file via `ReadAt` (O(1) seek, not sequential scan).
- **Preview caching**: Glamour was removed. Preview lines are rendered once per session change via lightweight `renderMarkdownLines()` and cached on the model. `View()` only slices the cache for scrolling.
- **Conversation count**: Computed lazily only when prune (`P`) is pressed, not on load.

### Claude Code Integration
- **Rename**: Uses Claude's native `custom-title` JSONL entry format: `{"type":"custom-title","customTitle":"name","sessionId":"id"}`. Appended to the session JSONL file. Compatible with `claude --resume "name"` and `/rename` inside Claude.
- **Resume**: On Enter, TUI quits and `syscall.Exec`s `claude --resume {sessionId}` in the project directory.
- **Delete cleanup**: Removes JSONL + agent files + debug + file-history + session-env + todos.

## Data Sources

| Data | Source |
|------|--------|
| Projects | `~/.claude/projects/` directory names (encoded paths) |
| Sessions | `*.jsonl` files in project dirs (excluding `agent-*.jsonl`) |
| Custom names | `custom-title` entries inside session JSONL files |
| Last session / stats | `~/.claude.json` → `projects[path].lastSessionId`, cost, tokens, duration |
| Messages | JSONL lines with `type: "user"/"assistant"`, `isMeta: false`, non-XML content |

## Session Title Priority

1. Custom name (from `custom-title` JSONL entry)
2. Slash command title (e.g. `/feature-dev: implement the plan...`) — parsed from `<command-name>` and `<command-args>` XML tags
3. First non-meta user message text (from first 30 lines, truncated to 60 chars)
4. Session UUID prefix (`3829d730...`) as fallback

## Keybindings

| Key | Action |
|-----|--------|
| `j/k` or `up/down` | Navigate within active panel |
| `h/l` or `left/right` | Switch panels |
| `Tab` | Cycle panels |
| `Enter` | Resume selected session (execs claude) |
| `r` | Rename session (writes native custom-title) |
| `d` | Delete session with confirmation |
| `x` | Export session to `~/Desktop/{name}.md` |
| `Space` | Toggle select for bulk operations |
| `D` | Bulk delete selected |
| `X` | Bulk export selected |
| `P` | Prune all empty sessions (0 conversation messages) |
| `/` | Search sessions by content |
| `?` | Help overlay |
| `q` / `Ctrl+C` | Quit |

## JSONL Message Format

```jsonl
{"type":"user","message":{"role":"user","content":"text or [{type,text}]"},"isMeta":false,"timestamp":"...","gitBranch":"..."}
{"type":"assistant","message":{"role":"assistant","content":"text or [{type,text}]"},"isMeta":false}
{"type":"custom-title","customTitle":"name","sessionId":"uuid"}
{"type":"file-history-snapshot",...}
```

Skip: `isMeta: true`, content starting with `<` (except `<command-*>` slash commands), types other than `user`/`assistant` for preview.

Slash commands use XML tags in content: `<command-name>/cmd</command-name>`, `<command-message>cmd</command-message>`, `<command-args>args</command-args>`. Tag order varies.

## Path Encoding

`~/.claude/projects/` dirs encode paths by replacing `/` with `-`:
`-Users-adithyankrishnan-Desktop-lifie` -> `/Users/adithyankrishnan/Desktop/lifie` -> `~/Desktop/lifie`
