package ui

import (
	"context"
	"fmt"
	"os"
	"strings"
)

// PromptService implements the Service interface
type PromptService struct {
	config  Config
	adapter Adapter
}

// Adapter interface for different UI implementations
type Adapter interface {
	Select(ctx context.Context, prompt string, options []string) (string, error)
	Input(ctx context.Context, prompt, placeholder string) (string, error)
	Password(ctx context.Context, prompt string) (string, error)
	Confirm(ctx context.Context, message string) (bool, error)
	MultiSelect(ctx context.Context, prompt string, options []string) ([]string, error)
	Message(level MessageLevel, text string)
	SupportsInteraction() bool
}

// NewService creates a new prompt service
func NewService(config Config) (Service, error) {
	adapter, err := createAdapter(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create adapter: %w", err)
	}

	return &PromptService{
		config:  config,
		adapter: adapter,
	}, nil
}

// createAdapter creates the appropriate adapter based on configuration
func createAdapter(config Config) (Adapter, error) {
	// Auto-detect adapter if not specified
	if config.AdapterType == "" {
		config.AdapterType = detectBestAdapter()
	}

	switch config.AdapterType {
	case AdapterBubbletea:
		return NewBubbleteaAdapter(config), nil
	case AdapterFallback:
		return NewFallbackAdapter(config), nil
	default:
		return nil, fmt.Errorf("unknown adapter type: %s", config.AdapterType)
	}
}

// detectBestAdapter automatically selects the best adapter for the environment
func detectBestAdapter() AdapterType {
	// Check environment variable override
	if adapter := os.Getenv("MCL_UI_ADAPTER"); adapter != "" {
		switch adapter {
		case "bubbletea":
			return AdapterBubbletea
		case "fallback":
			return AdapterFallback
		}
	}

	// Check if we're in an interactive environment
	if !isInteractiveEnvironment() {
		return AdapterFallback
	}

	// Check if bubbletea features are available
	if supportsBubbletea() {
		return AdapterBubbletea
	}

	// Fallback to basic text mode
	return AdapterFallback
}

// isInteractiveEnvironment checks if we're in an interactive environment
func isInteractiveEnvironment() bool {
	// Check for CI environments
	ciEnvs := []string{"CI", "GITHUB_ACTIONS", "JENKINS_URL", "TRAVIS", "CIRCLECI"}
	for _, env := range ciEnvs {
		if os.Getenv(env) != "" {
			return false
		}
	}

	// Check if stdin/stdout are terminals
	if os.Getenv("TERM") == "dumb" || os.Getenv("TERM") == "" {
		return false
	}

	return true
}

// supportsBubbletea checks if the environment supports bubbletea features
func supportsBubbletea() bool {
	// Check for color support
	colorTerm := os.Getenv("COLORTERM")
	term := os.Getenv("TERM")

	if strings.Contains(colorTerm, "truecolor") ||
		strings.Contains(colorTerm, "24bit") ||
		strings.Contains(term, "256color") {
		return true
	}

	// Basic color support is enough for bubbletea
	return term != "dumb" && term != ""
}

// Service interface implementation

func (s *PromptService) Select(ctx context.Context, prompt string, options []string) (string, error) {
	return s.adapter.Select(ctx, prompt, options)
}

func (s *PromptService) Input(ctx context.Context, prompt, placeholder string) (string, error) {
	return s.adapter.Input(ctx, prompt, placeholder)
}

func (s *PromptService) Password(ctx context.Context, prompt string) (string, error) {
	return s.adapter.Password(ctx, prompt)
}

func (s *PromptService) Confirm(ctx context.Context, message string) (bool, error) {
	return s.adapter.Confirm(ctx, message)
}

func (s *PromptService) MultiSelect(ctx context.Context, prompt string, options []string) ([]string, error) {
	return s.adapter.MultiSelect(ctx, prompt, options)
}

func (s *PromptService) Message(level MessageLevel, text string) {
	s.adapter.Message(level, text)
}

// Helper methods

// SelectWithOptions provides enhanced selection with Option structs
func (s *PromptService) SelectWithOptions(ctx context.Context, prompt string, options []Option) (*Option, error) {
	if len(options) == 0 {
		return nil, fmt.Errorf("no options provided")
	}

	// Convert options to strings
	stringOptions := make([]string, len(options))
	for i, opt := range options {
		if opt.DisplayText != "" {
			stringOptions[i] = opt.DisplayText
		} else {
			stringOptions[i] = opt.Value
		}
	}

	selected, err := s.adapter.Select(ctx, prompt, stringOptions)
	if err != nil {
		return nil, err
	}

	// Find the selected option
	for i, opt := range options {
		if stringOptions[i] == selected {
			return &opt, nil
		}
	}

	return nil, fmt.Errorf("selected option not found")
}

// Default instance
var defaultService Service

// GetDefault returns the default prompt service instance
func GetDefault() Service {
	if defaultService == nil {
		config := Config{
			AdapterType: detectBestAdapter(),
			Debug:       os.Getenv("MCL_DEBUG") == "true",
		}

		if theme := os.Getenv("MCL_THEME"); theme != "" {
			config.Theme = theme
		}

		service, err := NewService(config)
		if err != nil {
			// Fallback to basic service
			service, _ = NewService(Config{AdapterType: AdapterFallback})
		}
		defaultService = service
	}

	return defaultService
}

// SetDefault sets the default prompt service instance
func SetDefault(service Service) {
	defaultService = service
}
