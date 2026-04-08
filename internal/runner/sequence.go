package runner

import (
	"fmt"
	"sort"

	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

// SetSortOrder sets the execution order for a request
func SetSortOrder(db storage.DB, requestName string, order int) error {
	req, err := db.GetRequestByName(requestName)
	if err != nil {
		return fmt.Errorf("request not found: %s", requestName)
	}

	req.SortOrder = order
	if err := db.UpdateRequest(req); err != nil {
		return fmt.Errorf("failed to update sort order: %w", err)
	}

	return nil
}

// GetSequence returns requests in a collection sorted by SortOrder.
// SortOrder=0 means unordered (alphabetical by name).
// SortOrder > 0 means explicit ordering.
// Ties are broken by name (alphabetical).
func GetSequence(db storage.DB, collection string) ([]*types.SavedRequest, error) {
	reqs, err := db.ListRequests(&storage.ListOptions{
		Collection: collection,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list requests: %w", err)
	}

	if len(reqs) == 0 {
		return reqs, nil
	}

	// Sort by SortOrder (ascending), then by Name (alphabetical) for ties
	sorted := make([]*types.SavedRequest, len(reqs))
	copy(sorted, reqs)

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].SortOrder != sorted[j].SortOrder {
			return sorted[i].SortOrder < sorted[j].SortOrder
		}
		return sorted[i].Name < sorted[j].Name
	})

	return sorted, nil
}

// sortBySequence sorts requests by SortOrder before execution.
// SortOrder=0 requests run alphabetically.
// SortOrder>0 run in explicit order.
func sortBySequence(reqs []*types.SavedRequest) []*types.SavedRequest {
	sorted := make([]*types.SavedRequest, len(reqs))
	copy(sorted, reqs)

	sort.Slice(sorted, func(i, j int) bool {
		// Explicit ordering: SortOrder > 0 comes before SortOrder = 0
		// (but we want ascending, so 0 still sorts after positive for explicit-first)
		// Actually, per spec: SortOrder=0 means unordered (alphabetical)
		// So we sort: first by SortOrder (ascending), then by Name for ties
		if sorted[i].SortOrder != sorted[j].SortOrder {
			return sorted[i].SortOrder < sorted[j].SortOrder
		}
		return sorted[i].Name < sorted[j].Name
	})

	return sorted
}
