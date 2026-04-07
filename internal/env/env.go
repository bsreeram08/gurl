package env

import (
	"time"

	"github.com/google/uuid"
)

type Environment struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Variables  map[string]string `json:"variables"`
	SecretKeys map[string]bool   `json:"secret_keys"`
	ParentID   string            `json:"parent_id"`
	CreatedAt  int64             `json:"created_at"`
	UpdatedAt  int64             `json:"updated_at"`
}

func NewEnvironment(name string, parentID string) *Environment {
	now := time.Now().Unix()
	return &Environment{
		ID:         uuid.New().String(),
		Name:       name,
		Variables:  make(map[string]string),
		SecretKeys: make(map[string]bool),
		ParentID:   parentID,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (e *Environment) SetVariable(key, value string) {
	e.Variables[key] = value
	e.UpdatedAt = time.Now().Unix()
}

func (e *Environment) GetVariable(key string) (string, bool) {
	v, ok := e.Variables[key]
	return v, ok
}

func (e *Environment) DeleteVariable(key string) {
	delete(e.Variables, key)
	delete(e.SecretKeys, key)
	e.UpdatedAt = time.Now().Unix()
}

func (e *Environment) SetSecretVariable(key, value string) {
	e.Variables[key] = value
	e.SecretKeys[key] = true
	e.UpdatedAt = time.Now().Unix()
}

func (e *Environment) IsSecret(key string) bool {
	return e.SecretKeys[key]
}
