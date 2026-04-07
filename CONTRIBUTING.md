# Contributing to Gurl

## Development Setup
```bash
git clone https://github.com/sreeram/gurl
cd gurl
go mod download
go build -o gurl ./cmd/gurl
```

## Testing
```bash
go test ./...
```

## Code Style
- Run `go fmt ./...` before committing
- Follow deterministic programming patterns (no if-else-if-else chains)
- Use switch statements or early returns

## Submitting Changes
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request
