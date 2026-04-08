---
title: "CLI Reference"
description: "Complete reference for all gurl commands"
---

# Gurl CLI Reference

Gurl is a command-line HTTP client for saving, organizing, and executing curl requests.

## Commands

### Core

| Command | Description |
|---------|-------------|
| [`save`](save) | Save a curl request with a name |
| [`run`](run) | Execute a saved request |
| [`list`](list) | List saved requests |
| [`edit`](edit) | Edit a saved request |
| [`delete`](delete) | Remove a saved request |
| [`rename`](rename) | Rename a saved request |

### Data

| Command | Description |
|---------|-------------|
| [`history`](history) | Show execution history for a request |
| [`timeline`](timeline) | Show global execution timeline |
| [`diff`](diff) | Compare responses for a request |
| [`detect`](detect) | Parse curl from stdin or file |

### Organization

| Command | Description |
|---------|-------------|
| [`collection`](collection) | Manage request collections |
| [`env`](env) | Manage environment variables |
| [`sequence`](sequence) | Manage request sequences |

### Import/Export

| Command | Description |
|---------|-------------|
| [`import`](import) | Import from external formats |
| [`export`](export) | Export requests to file |
| [`paste`](paste) | Copy request as curl to clipboard |
| [`codegen`](codegen) | Generate code from a request |

### Interface

| Command | Description |
|---------|-------------|
| [`tui`](tui) | Launch interactive TUI |
| [`graphql`](graphql) | Execute a GraphQL query |
| [`update`](update) | Update gurl to latest version |

## Quick Start

```bash
# Save your first request
gurl save "my-request" https://api.example.com/users

# Run it
gurl run "my-request"

# List all saved requests
gurl list

# Edit in your default editor
gurl edit "my-request"
```
