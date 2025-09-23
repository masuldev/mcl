package ui

import "context"

// Service provides UI interaction capabilities
type Service interface {
	Select(ctx context.Context, prompt string, options []string) (string, error)
	Input(ctx context.Context, prompt, placeholder string) (string, error)
	Password(ctx context.Context, prompt string) (string, error)
	Confirm(ctx context.Context, message string) (bool, error)
	MultiSelect(ctx context.Context, prompt string, options []string) ([]string, error)
	Message(level MessageLevel, text string)
}

// MessageLevel represents the level of a message
type MessageLevel int

const (
	MessageInfo MessageLevel = iota
	MessageWarning
	MessageError
	MessageSuccess
)

// AdapterType represents different UI adapter types
type AdapterType string

const (
	AdapterBubbletea AdapterType = "bubbletea"
	AdapterFallback  AdapterType = "fallback"
)

// Config holds prompt service configuration
type Config struct {
	AdapterType AdapterType
	Theme       string
	Debug       bool
}

// Option represents a selection option
type Option struct {
	Value       string
	DisplayText string
	Description string
	Disabled    bool
}

// PromptConfig holds configuration for individual prompts
type PromptConfig struct {
	Title       string
	Placeholder string
	Default     string
	Required    bool
	Validator   func(string) error
}
