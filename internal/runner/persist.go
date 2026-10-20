package runner

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sreeram/gurl/internal/env"
	"github.com/sreeram/gurl/internal/resolve"
	"github.com/sreeram/gurl/internal/secrets"
	"github.com/sreeram/gurl/internal/storage"
	"github.com/sreeram/gurl/pkg/types"
)

type PersistSummary struct {
	EnvironmentName       string
	EnvironmentVars       map[string]string
	Environment           *env.Environment
	DefaultCollectionName string
	Collections           []CollectionPersistSummary
}

type CollectionPersistSummary struct {
	Name       string
	Vars       map[string]string
	Collection *types.Collection
}

func PersistDirtyVariables(envStorage env.Store, collectionStore storage.CollectionStore, envName string, collectionName string, vars map[string]string, origins map[string]resolve.Origin) (*PersistSummary, error) {
	return PersistDirtyVariablesWithCollectionTargets(envStorage, collectionStore, envName, collectionName, vars, origins, nil)
}

func PersistDirtyVariablesWithCollectionTargets(envStorage env.Store, collectionStore storage.CollectionStore, envName string, defaultCollectionName string, vars map[string]string, origins map[string]resolve.Origin, collectionTargets map[string]string) (*PersistSummary, error) {
	summary := &PersistSummary{
		EnvironmentName:       envName,
		EnvironmentVars:       make(map[string]string),
		DefaultCollectionName: defaultCollectionName,
	}
	if len(vars) == 0 {
		return summary, nil
	}

	for _, key := range sortedStringKeys(vars) {
		// Persisted values come from responses and scripts. Environment aliases
		// and collection variables expand {{...}} from stored values, so a
		// response value like "{{API_KEY}}" could pull a secret into a later request.
		if strings.Contains(vars[key], "{{") {
			return nil, fmt.Errorf("refusing to persist %q: value contains \"{{\"", key)
		}
	}

	collectionVars := make(map[string]map[string]string)
	for key, value := range vars {
		collectionName := defaultCollectionName
		if collectionTargets != nil && collectionTargets[key] != "" {
			collectionName = collectionTargets[key]
		}
		switch targetForOrigin(origins[key], collectionName, collectionStore != nil) {
		case resolve.OriginCollection:
			if collectionVars[collectionName] == nil {
				collectionVars[collectionName] = make(map[string]string)
			}
			collectionVars[collectionName][key] = value
		default:
			summary.EnvironmentVars[key] = value
		}
	}

	if len(summary.EnvironmentVars) > 0 {
		persisted, err := env.PersistVariables(envStorage, envName, summary.EnvironmentVars)
		if err != nil {
			return nil, err
		}
		summary.EnvironmentVars = persisted
		if envStorage != nil && envName != "" {
			summary.Environment, _ = envStorage.GetEnvByName(envName)
		}
	}

	collectionNames := make([]string, 0, len(collectionVars))
	for collectionName := range collectionVars {
		collectionNames = append(collectionNames, collectionName)
	}
	sort.Strings(collectionNames)
	for _, collectionName := range collectionNames {
		varsForCollection := collectionVars[collectionName]
		if len(varsForCollection) > 0 {
			if collectionName == "" || collectionStore == nil {
				return nil, fmt.Errorf("--persist requires --env, an active environment, or a collection-backed run")
			}
			collection, err := collectionStore.GetCollectionByName(collectionName)
			switch {
			case storage.IsCollectionNotFound(err) || (err == nil && collection == nil):
				collection = types.NewCollection(collectionName)
			case err != nil:
				// A locked collection must not be replaced by a fresh one:
				// that would drop its stored variables and secrets.
				return nil, fmt.Errorf("failed to load collection %q for --persist: %w", collectionName, err)
			default:
				collection = cloneCollection(collection)
			}
			if collection.Variables == nil {
				collection.Variables = make(map[string]string)
			}
			if collection.SecretKeys == nil {
				collection.SecretKeys = make(map[string]bool)
			}
			for key, value := range varsForCollection {
				if collection.IsSecret(key) {
					collection.SetSecretVariable(key, value)
					continue
				}
				collection.SetVariable(key, value)
			}
			if err := collectionStore.SaveCollection(collection); err != nil {
				return nil, fmt.Errorf("failed to persist variables to collection %q: %w", collectionName, err)
			}
			summary.Collections = append(summary.Collections, CollectionPersistSummary{
				Name:       collectionName,
				Vars:       varsForCollection,
				Collection: collection,
			})
		}
	}

	return summary, nil
}

func targetForOrigin(origin resolve.Origin, collectionName string, hasCollectionStore bool) resolve.Origin {
	switch origin {
	case resolve.OriginEnvironment:
		return resolve.OriginEnvironment
	case resolve.OriginCollection:
		if collectionName != "" && hasCollectionStore {
			return resolve.OriginCollection
		}
		return resolve.OriginEnvironment
	case resolve.OriginCLI, resolve.OriginData:
		if collectionName != "" && hasCollectionStore {
			return resolve.OriginCollection
		}
		return resolve.OriginEnvironment
	default:
		if collectionName != "" && hasCollectionStore {
			return resolve.OriginCollection
		}
		return resolve.OriginEnvironment
	}
}

func PrintPersistSummaries(out io.Writer, summary *PersistSummary) {
	if summary == nil {
		return
	}
	if len(summary.EnvironmentVars) > 0 {
		PrintPersistSummary(out, summary.EnvironmentName, summary.EnvironmentVars, summary.Environment)
	}
	for _, collectionSummary := range summary.Collections {
		PrintCollectionPersistSummary(out, collectionSummary.Name, collectionSummary.Vars, collectionSummary.Collection)
	}
	if len(summary.EnvironmentVars) == 0 && len(summary.Collections) == 0 {
		if summary.DefaultCollectionName != "" {
			PrintCollectionPersistSummary(out, summary.DefaultCollectionName, nil, nil)
			return
		}
		PrintPersistSummary(out, summary.EnvironmentName, nil, summary.Environment)
	}
}

func PrintCollectionPersistSummary(out io.Writer, collectionName string, persisted map[string]string, collection *types.Collection) {
	count := len(persisted)
	label := "variables"
	if count == 1 {
		label = "variable"
	}
	fmt.Fprintf(out, "\nPersisted %d %s to collection %q\n", count, label, collectionName)

	keys := make([]string, 0, len(persisted))
	for key := range persisted {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := persisted[key]
		if collection != nil && collection.IsSecret(key) {
			value = secrets.Mask(value)
		}
		fmt.Fprintf(out, "  %s = %s\n", key, value)
	}
}

func cloneCollection(source *types.Collection) *types.Collection {
	if source == nil {
		return nil
	}
	clone := *source
	clone.Variables = make(map[string]string, len(source.Variables))
	for key, value := range source.Variables {
		clone.Variables[key] = value
	}
	clone.SecretKeys = make(map[string]bool, len(source.SecretKeys))
	for key, value := range source.SecretKeys {
		clone.SecretKeys[key] = value
	}
	return &clone
}

func sortedStringKeys(values map[string]string) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
