---
title: "gurl tui"
description: "Launch interactive TUI"
---

# gurl tui

Launch the interactive terminal user interface for gurl.

## Usage

```bash
gurl tui
```

## Description

The `tui` command opens an interactive three-pane workspace where you can browse saved requests, edit the active request, and inspect the latest response without leaving the terminal.

## Workspace

- **Requests pane** for saved requests and recent history
- **Request pane** for method, URL, headers, body, query, and auth editing
- **Response pane** for preview, headers, cookies, timing, and diff views

## Key controls

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

### Launch TUI

```bash
gurl tui
```

Opens the interactive terminal interface.

## See also

- [`gurl save`](save) - Save a new request
- [`gurl run`](run) - Execute a request
- [`gurl list`](list) - List saved requests
