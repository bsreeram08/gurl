---
title: "gurl tui"
description: "Launch interactive TUI (not functional yet)"
---

# gurl tui

> **Not functional.** The `tui` command exists but the interface does not work. The description below reflects the intended design, not the current state. Do not rely on this for any actual use.

Launch the interactive terminal user interface for gurl.

## Usage

```bash
gurl tui
```

## Description (aspirational — not functional yet)

The `tui` command is intended to open an interactive three-pane workspace where you can browse saved requests, edit the active request, and inspect the latest response without leaving the terminal. This is not implemented in the current release.

## Workspace (not functional yet)

The intended workspace design:

- **Requests pane** for saved requests and recent history
- **Request pane** for method, URL, headers, body, query, and auth editing
- **Response pane** for preview, headers, cookies, timing, and diff views

## Key controls (not functional yet)

- `Ctrl+1`, `Ctrl+2`, `Ctrl+3` focus the requests, request, and response panes
- `Ctrl+Enter` sends the active request
- `Ctrl+S` saves the active request
- `Ctrl+E` opens the environment switcher
- `Ctrl+K` opens request search

## Flags

None.

## Aliases

- `ui`

## Examples

### Launch TUI (not functional yet)

```bash
gurl tui
```

The command exists but the interface is not functional.

## See also

- [`gurl save`](save) - Save a new request
- [`gurl run`](run) - Execute a request
- [`gurl list`](list) - List saved requests
