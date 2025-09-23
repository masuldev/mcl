package ui

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// FallbackAdapter implements Adapter for non-interactive environments
type FallbackAdapter struct {
	config Config
	reader *bufio.Reader
}

// NewFallbackAdapter creates a new fallback adapter
func NewFallbackAdapter(config Config) Adapter {
	return &FallbackAdapter{
		config: config,
		reader: bufio.NewReader(os.Stdin),
	}
}

// Select implements Adapter.Select
func (a *FallbackAdapter) Select(ctx context.Context, prompt string, options []string) (string, error) {
	if len(options) == 0 {
		return "", fmt.Errorf("no options provided")
	}

	// In non-interactive mode, try environment variable first
	if envVar := os.Getenv("MCL_AUTO_SELECT"); envVar != "" {
		for _, option := range options {
			if strings.Contains(strings.ToLower(option), strings.ToLower(envVar)) {
				a.Message(MessageInfo, fmt.Sprintf("Auto-selected: %s", option))
				return option, nil
			}
		}
	}

	// If only one option, auto-select it
	if len(options) == 1 {
		a.Message(MessageInfo, fmt.Sprintf("Auto-selected (only option): %s", options[0]))
		return options[0], nil
	}

	// Interactive fallback
	fmt.Printf("\n%s\n", prompt)
	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}
	fmt.Print("Select option (1-", len(options), "): ")

	line, err := a.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	line = strings.TrimSpace(line)
	index, err := strconv.Atoi(line)
	if err != nil || index < 1 || index > len(options) {
		return "", fmt.Errorf("invalid selection: %s", line)
	}

	return options[index-1], nil
}

// Input implements Adapter.Input
func (a *FallbackAdapter) Input(ctx context.Context, prompt, placeholder string) (string, error) {
	// Check environment variable first
	if envKey := a.getEnvKey(prompt); envKey != "" {
		if envValue := os.Getenv(envKey); envValue != "" {
			a.Message(MessageInfo, fmt.Sprintf("Using environment variable %s", envKey))
			return envValue, nil
		}
	}

	// Interactive input
	if placeholder != "" {
		fmt.Printf("%s [%s]: ", prompt, placeholder)
	} else {
		fmt.Printf("%s: ", prompt)
	}

	line, err := a.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" && placeholder != "" {
		return placeholder, nil
	}

	return line, nil
}

// Password implements Adapter.Password
func (a *FallbackAdapter) Password(ctx context.Context, prompt string) (string, error) {
	// Check environment variable first
	if envKey := a.getEnvKey(prompt); envKey != "" {
		if envValue := os.Getenv(envKey); envValue != "" {
			a.Message(MessageInfo, fmt.Sprintf("Using environment variable %s", envKey))
			return envValue, nil
		}
	}

	// For non-interactive environments, require environment variable
	if !a.SupportsInteraction() {
		return "", fmt.Errorf("password input not supported in non-interactive mode")
	}

	fmt.Printf("%s: ", prompt)
	line, err := a.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(line), nil
}

// Confirm implements Adapter.Confirm
func (a *FallbackAdapter) Confirm(ctx context.Context, message string) (bool, error) {
	// Check environment variable
	if envVar := os.Getenv("MCL_AUTO_CONFIRM"); envVar != "" {
		return strings.ToLower(envVar) == "true" || envVar == "1", nil
	}

	fmt.Printf("%s (y/N): ", message)
	line, err := a.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read input: %w", err)
	}

	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

// MultiSelect implements Adapter.MultiSelect
func (a *FallbackAdapter) MultiSelect(ctx context.Context, prompt string, options []string) ([]string, error) {
	if len(options) == 0 {
		return []string{}, nil
	}

	fmt.Printf("\n%s (comma-separated numbers)\n", prompt)
	for i, option := range options {
		fmt.Printf("  %d) %s\n", i+1, option)
	}
	fmt.Print("Select options (e.g., 1,3,5): ")

	line, err := a.reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	line = strings.TrimSpace(line)
	if line == "" {
		return []string{}, nil
	}

	var selected []string
	for _, indexStr := range strings.Split(line, ",") {
		indexStr = strings.TrimSpace(indexStr)
		index, err := strconv.Atoi(indexStr)
		if err != nil || index < 1 || index > len(options) {
			continue
		}
		selected = append(selected, options[index-1])
	}

	return selected, nil
}

// Message implements Adapter.Message
func (a *FallbackAdapter) Message(level MessageLevel, text string) {
	var prefix string
	switch level {
	case MessageInfo:
		prefix = "[INFO] "
	case MessageWarning:
		prefix = "[WARN] "
	case MessageError:
		prefix = "[ERROR] "
	case MessageSuccess:
		prefix = "[SUCCESS] "
	}

	fmt.Printf("%s%s\n", prefix, text)
}

// SupportsInteraction implements Adapter.SupportsInteraction
func (a *FallbackAdapter) SupportsInteraction() bool {
	// Check if we're in CI or non-interactive environment
	ciEnvs := []string{"CI", "GITHUB_ACTIONS", "JENKINS_URL", "TRAVIS", "CIRCLECI"}
	for _, env := range ciEnvs {
		if os.Getenv(env) != "" {
			return false
		}
	}

	return os.Getenv("TERM") != "dumb"
}

// getEnvKey generates environment variable key from prompt
func (a *FallbackAdapter) getEnvKey(prompt string) string {
	// Convert prompt to environment variable key
	key := strings.ToUpper(prompt)
	key = strings.ReplaceAll(key, " ", "_")
	key = strings.ReplaceAll(key, ":", "")
	key = strings.ReplaceAll(key, "?", "")
	key = "MCL_" + key

	return key
}
