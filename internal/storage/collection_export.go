package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sreeram/gurl/internal/secrets"
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

func ParseCollectionDirectory(dir string, passphrase string) (*types.Collection, []*types.SavedRequest, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to inspect collection directory: %w", err)
	}
	if !info.IsDir() {
		return nil, nil, fmt.Errorf("collection import path is not a directory: %s", dir)
	}

	collection, err := readCollectionFile(filepath.Join(dir, collectionFileName))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read collection metadata: %w", err)
	}
	collection, err = decryptCollectionDirectory(collection, dir, passphrase)
	if err != nil {
		return nil, nil, err
	}

	records, err := readRequestRecordsInDir(dir)
	if err != nil {
		return nil, nil, err
	}
	requests := make([]*types.SavedRequest, 0, len(records))
	for _, record := range records {
		req := record.request
		if collection != nil {
			req.Collection = collection.Name
		}
		requests = append(requests, req)
	}
	return collection, requests, nil
}

func decryptCollectionDirectory(collection *types.Collection, dir string, passphrase string) (*types.Collection, error) {
	if collection == nil || !collectionHasSecrets(collection) {
		return collection, nil
	}
	if collection.Encryption != nil && collection.Encryption.Mode == CollectionEncryptionModePassphrase {
		return decryptCollectionFromPassphrase(collection, passphrase)
	}

	hasEncrypted := false
	for key, isSecret := range collection.SecretKeys {
		if isSecret && IsCollectionEncryptedValue(collection.Variables[key]) {
			hasEncrypted = true
			break
		}
	}
	if !hasEncrypted {
		return collection, nil
	}

	key, err := os.ReadFile(collectionKeyPath(dir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, &CollectionLockedError{
				Name: collection.Name,
				Hint: "missing collection.key; restore the local key or import a passphrase-protected export",
			}
		}
		return nil, err
	}
	if len(key) != secrets.KeySize {
		return nil, fmt.Errorf("invalid collection key size")
	}
	if err := decryptCollectionSecrets(collection, key); err != nil {
		return nil, fmt.Errorf("failed to decrypt collection secrets: %w", err)
	}
	return collection, nil
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
