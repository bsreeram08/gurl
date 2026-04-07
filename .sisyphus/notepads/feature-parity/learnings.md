# Feature Parity Learnings

## Task: ParsedCurl ↔ SavedRequest Conversion Functions

### What Worked
- Created `ParsedCurlToSavedRequest` and `SavedRequestToParsedCurl` in `pkg/types/types.go`
- `ParsedCurl.Headers` is `map[string]string`, `SavedRequest.Headers` is `[]Header` where `Header = {Key, Value string}`
- Used range loops for field mapping (not if-else chains) as required
- Preserved `nil` for both nil maps and nil slices — critical edge case

### Key Edge Cases Handled
- `nil` headers map → `nil` headers slice (and vice versa)
- Empty headers map → empty headers slice (acceptable)
- Empty method/body preserved correctly

### Test Results
- `go test ./pkg/types -v -count=1` → 11 passed
- Round-trip: SavedRequest → ParsedCurl → SavedRequest preserves all data ✓

### Notes
- Pre-existing build error in `internal/cli/commands/run_test.go` (unused imports) — unrelated to this task
