package env

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// MemoryStore is an in-memory Store for tests. The zero value is ready to
// use. Environments are copied on the way in and out, so callers cannot
// mutate stored values through returned pointers.
type MemoryStore struct {
	mu     sync.Mutex
	envs   map[string]*Environment // by ID
	active string
}

// NewMemoryStore returns an empty MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// SaveEnv creates or replaces env, assigning an ID when it has none.
func (s *MemoryStore) SaveEnv(env *Environment) error {
	if env == nil {
		return fmt.Errorf("environment cannot be nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.envs == nil {
		s.envs = make(map[string]*Environment)
	}
	if env.ID == "" {
		env.ID = uuid.New().String()
	}
	for id, existing := range s.envs {
		if existing.Name == env.Name && id != env.ID {
			delete(s.envs, id)
		}
	}
	s.envs[env.ID] = CloneEnvironment(env)
	return nil
}

func (s *MemoryStore) GetEnv(id string) (*Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	env, ok := s.envs[id]
	if !ok {
		return nil, fmt.Errorf("environment not found: %s", id)
	}
	return CloneEnvironment(env), nil
}

func (s *MemoryStore) GetEnvByName(name string) (*Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, env := range s.envs {
		if env.Name == name {
			return CloneEnvironment(env), nil
		}
	}
	return nil, fmt.Errorf("environment not found: %s", name)
}

// DeleteEnv removes an environment. Like EnvStorage, it leaves the active
// environment name alone.
func (s *MemoryStore) DeleteEnv(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.envs[id]; !ok {
		return fmt.Errorf("environment not found: %s", id)
	}
	delete(s.envs, id)
	return nil
}

// ListEnvs returns every environment sorted by name.
func (s *MemoryStore) ListEnvs() ([]*Environment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	envs := make([]*Environment, 0, len(s.envs))
	for _, env := range s.envs {
		envs = append(envs, CloneEnvironment(env))
	}
	sortEnvs(envs)
	return envs, nil
}

func (s *MemoryStore) GetActiveEnv() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.active, nil
}

// SetActiveEnv records name as active; "" clears it.
func (s *MemoryStore) SetActiveEnv(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.active = name
	return nil
}

var _ Store = (*MemoryStore)(nil)
