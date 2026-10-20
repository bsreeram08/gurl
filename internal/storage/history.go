package storage

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/sreeram/gurl/pkg/types"
)

var (
	// ErrHistoryNotFound means no execution record has the given ID.
	ErrHistoryNotFound = errors.New("execution record not found")
	// ErrAmbiguousHistoryID means an ID prefix matches several records.
	ErrAmbiguousHistoryID = errors.New("ambiguous execution id")
	// ErrHistoryIDRequired means an empty ID was given.
	ErrHistoryIDRequired = errors.New("execution id is required")
)

func normalizeHistoryID(idOrPrefix string) string {
	return strings.ToLower(strings.TrimSpace(idOrPrefix))
}

// MatchHistoryID picks the record whose ID is idOrPrefix, or the only one
// whose ID starts with it. Adapters use it to implement GetHistoryByID.
func MatchHistoryID(records []*types.ExecutionHistory, idOrPrefix string) (*types.ExecutionHistory, error) {
	prefix := normalizeHistoryID(idOrPrefix)
	if prefix == "" {
		return nil, ErrHistoryIDRequired
	}
	var matches []*types.ExecutionHistory
	for _, h := range records {
		id := strings.ToLower(h.ID)
		if id == prefix {
			return h, nil
		}
		if strings.HasPrefix(id, prefix) {
			matches = append(matches, h)
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("%w: %s", ErrHistoryNotFound, idOrPrefix)
	case 1:
		return matches[0], nil
	}
	ids := make([]string, len(matches))
	for i, h := range matches {
		// Short ids can't tell records apart once the prefix is that long.
		ids[i] = h.ShortID()
		if len(prefix) >= len(ids[i]) {
			ids[i] = h.ID
		}
	}
	sort.Strings(ids)
	return nil, fmt.Errorf("%w: %s matches %d records (%s); use more characters", ErrAmbiguousHistoryID, idOrPrefix, len(matches), strings.Join(ids, ", "))
}
