package env

import (
	"time"

	"github.com/google/uuid"
)

type Environment struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Variables map[string]string `json:"variables"`
	ParentID  string            `json:"parent_id"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}

func NewEnvironment(name string, parentID string) *Environment {
	now := time.Now().Unix()
	return &Environment{
		ID:        uuid.New().String(),
		Name:      name,
		Variables: make(map[string]string),
		ParentID:  parentID,
		CreatedAt: now,
		UpdatedAt: now,
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
	e.UpdatedAt = time.Now().Unix()
}
