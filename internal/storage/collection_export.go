package storage

import (
	"encoding/json"
	"time"

	"github.com/sreeram/gurl/pkg/types"
)

type CollectionExport struct {
	Version    string                `json:"version"`
	ExportedAt string                `json:"exported_at"`
	Collection *types.Collection     `json:"collection"`
	Requests   []*types.SavedRequest `json:"requests"`
}

func BuildCollectionExport(collection *types.Collection, requests []*types.SavedRequest, passphrase string) (*CollectionExport, error) {
	exportedCollection, err := encryptCollectionForPassphrase(collection, passphrase)
	if err != nil {
		return nil, err
	}
	return &CollectionExport{
		Version:    "1.0",
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Collection: exportedCollection,
		Requests:   cloneRequestsForExport(requests),
	}, nil
}

func MarshalCollectionExport(exportData *CollectionExport) ([]byte, error) {
	return json.MarshalIndent(exportData, "", "  ")
}

func ParseCollectionExport(data []byte, passphrase string) (*types.Collection, []*types.SavedRequest, error) {
	var exportData CollectionExport
	if err := json.Unmarshal(data, &exportData); err != nil {
		return nil, nil, err
	}
	if exportData.Version != "1.0" {
		return nil, nil, ErrUnsupportedCollectionExportVersion(exportData.Version)
	}
	collection, err := decryptCollectionFromPassphrase(exportData.Collection, passphrase)
	if err != nil {
		return nil, nil, err
	}
	requests := cloneRequestsForExport(exportData.Requests)
	if collection != nil {
		for _, req := range requests {
			if req.Collection == "" {
				req.Collection = collection.Name
			}
		}
	}
	return collection, requests, nil
}

type UnsupportedCollectionExportVersionError string

func (e UnsupportedCollectionExportVersionError) Error() string {
	return "unsupported collection export version: " + string(e)
}

func ErrUnsupportedCollectionExportVersion(version string) error {
	return UnsupportedCollectionExportVersionError(version)
}

func cloneRequestsForExport(requests []*types.SavedRequest) []*types.SavedRequest {
	if requests == nil {
		return nil
	}
	cloned := make([]*types.SavedRequest, 0, len(requests))
	for _, req := range requests {
		if req == nil {
			continue
		}
		copy := *req
		copy.Headers = append([]types.Header(nil), req.Headers...)
		copy.Variables = append([]types.Var(nil), req.Variables...)
		copy.PathParams = append([]types.Var(nil), req.PathParams...)
		copy.Tags = append([]string(nil), req.Tags...)
		copy.Assertions = append([]types.Assertion(nil), req.Assertions...)
		copy.Extracts = append([]types.Extract(nil), req.Extracts...)
		if req.AuthConfig != nil {
			authCopy := *req.AuthConfig
			authCopy.Params = cloneStringMap(req.AuthConfig.Params)
			copy.AuthConfig = &authCopy
		}
		cloned = append(cloned, &copy)
	}
	return cloned
}
