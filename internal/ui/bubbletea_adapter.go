package ui

import (
	"context"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dracula theme color palette
var dracula = struct {
	Background   lipgloss.Color
	CurrentLine  lipgloss.Color
	Selection    lipgloss.Color
	Foreground   lipgloss.Color
	Comment      lipgloss.Color
	Cyan         lipgloss.Color
	Green        lipgloss.Color
	Orange       lipgloss.Color
	Pink         lipgloss.Color
	Purple       lipgloss.Color
	Red          lipgloss.Color
	Yellow       lipgloss.Color
}{
	Background:   lipgloss.Color("#282a36"),
	CurrentLine:  lipgloss.Color("#44475a"),
	Selection:    lipgloss.Color("#44475a"),
	Foreground:   lipgloss.Color("#f8f8f2"),
	Comment:      lipgloss.Color("#6272a4"),
	Cyan:         lipgloss.Color("#8be9fd"),
	Green:        lipgloss.Color("#50fa7b"),
	Orange:       lipgloss.Color("#ffb86c"),
	Pink:         lipgloss.Color("#ff79c6"),
	Purple:       lipgloss.Color("#bd93f9"),
	Red:          lipgloss.Color("#ff5555"),
	Yellow:       lipgloss.Color("#f1fa8c"),
}

// BubbleteaAdapter implements Adapter using bubbletea
type BubbleteaAdapter struct {
	config Config
}

// NewBubbleteaAdapter creates a new bubbletea adapter
func NewBubbleteaAdapter(config Config) Adapter {
	return &BubbleteaAdapter{config: config}
}

// Select implements Adapter.Select using bubbletea
func (a *BubbleteaAdapter) Select(ctx context.Context, prompt string, options []string) (string, error) {
	model := newSelectModel(prompt, options, a.config)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("bubbletea program failed: %w", err)
	}

	selectModel := finalModel.(selectModel)
	if selectModel.cancelled {
		return "", fmt.Errorf("selection cancelled")
	}

	if selectModel.selected >= 0 && selectModel.selected < len(options) {
		return options[selectModel.selected], nil
	}

	return "", fmt.Errorf("no selection made")
}

// Input implements Adapter.Input using bubbletea
func (a *BubbleteaAdapter) Input(ctx context.Context, prompt, placeholder string) (string, error) {
	model := newInputModel(prompt, placeholder, a.config)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("bubbletea program failed: %w", err)
	}

	inputModel := finalModel.(inputModel)
	if inputModel.cancelled {
		return "", fmt.Errorf("input cancelled")
	}

	return inputModel.value, nil
}

// Password implements Adapter.Password using bubbletea
func (a *BubbleteaAdapter) Password(ctx context.Context, prompt string) (string, error) {
	model := newPasswordModel(prompt, a.config)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return "", fmt.Errorf("bubbletea program failed: %w", err)
	}

	passwordModel := finalModel.(passwordModel)
	if passwordModel.cancelled {
		return "", fmt.Errorf("password cancelled")
	}

	return passwordModel.value, nil
}

// Confirm implements Adapter.Confirm using bubbletea
func (a *BubbleteaAdapter) Confirm(ctx context.Context, message string) (bool, error) {
	model := newConfirmModel(message, a.config)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return false, fmt.Errorf("bubbletea program failed: %w", err)
	}

	confirmModel := finalModel.(confirmModel)
	if confirmModel.cancelled {
		return false, fmt.Errorf("confirmation cancelled")
	}

	return confirmModel.confirmed, nil
}

// MultiSelect implements Adapter.MultiSelect using bubbletea
func (a *BubbleteaAdapter) MultiSelect(ctx context.Context, prompt string, options []string) ([]string, error) {
	model := newMultiSelectModel(prompt, options, a.config)

	program := tea.NewProgram(model)
	finalModel, err := program.Run()
	if err != nil {
		return nil, fmt.Errorf("bubbletea program failed: %w", err)
	}

	multiSelectModel := finalModel.(multiSelectModel)
	if multiSelectModel.cancelled {
		return nil, fmt.Errorf("multi-select cancelled")
	}

	var selected []string
	for i, isSelected := range multiSelectModel.selected {
		if isSelected && i < len(options) {
			selected = append(selected, options[i])
		}
	}

	return selected, nil
}

// Message implements Adapter.Message with Dracula theme
func (a *BubbleteaAdapter) Message(level MessageLevel, text string) {
	var style lipgloss.Style
	var prefix string

	switch level {
	case MessageInfo:
		style = lipgloss.NewStyle().Foreground(dracula.Cyan).Bold(true)
		prefix = "INFO: "
	case MessageWarning:
		style = lipgloss.NewStyle().Foreground(dracula.Yellow).Bold(true)
		prefix = "WARN: "
	case MessageError:
		style = lipgloss.NewStyle().Foreground(dracula.Red).Bold(true)
		prefix = "ERROR: "
	case MessageSuccess:
		style = lipgloss.NewStyle().Foreground(dracula.Green).Bold(true)
		prefix = "SUCCESS: "
	}

	fmt.Println(style.Render(prefix + text))
}

// SupportsInteraction implements Adapter.SupportsInteraction
func (a *BubbleteaAdapter) SupportsInteraction() bool {
	return true
}

// Bubbletea models

// selectModel handles selection UI with paging and instant search
type selectModel struct {
	prompt       string
	options      []string
	filtered     []string
	cursor       int
	selected     int
	cancelled    bool
	config       Config
	searchValue  string
	pageSize     int
	pageOffset   int
}

func newSelectModel(prompt string, options []string, config Config) selectModel {
	pageSize := 10 // Default page size
	if len(options) < pageSize {
		pageSize = len(options)
	}

	return selectModel{
		prompt:      prompt,
		options:     options,
		filtered:    options, // Initially show all options
		cursor:      0,
		selected:    -1,
		config:      config,
		searchValue: "",
		pageSize:    pageSize,
		pageOffset:  0,
	}
}

func (m selectModel) Init() tea.Cmd {
	return nil
}

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			if len(m.searchValue) > 0 {
				// Clear search if there's a search term
				m.searchValue = ""
				m.filtered = m.options
				m.cursor = 0
				m.pageOffset = 0
			} else {
				m.cancelled = true
				return m, tea.Quit
			}
		case "enter":
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.getCurrentPageOptions()) {
				// Find original index in options
				selectedOption := m.getCurrentPageOptions()[m.cursor]
				for i, option := range m.options {
					if option == selectedOption {
						m.selected = i
						break
					}
				}
			}
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			} else if m.pageOffset > 0 {
				m.pageOffset--
				m.cursor = 0
			}
		case "down", "j":
			maxCursor := len(m.getCurrentPageOptions()) - 1
			if m.cursor < maxCursor {
				m.cursor++
			} else if m.pageOffset+m.pageSize < len(m.filtered) {
				m.pageOffset++
				m.cursor = maxCursor
			}
		case "pgup":
			if m.pageOffset > 0 {
				m.pageOffset -= m.pageSize
				if m.pageOffset < 0 {
					m.pageOffset = 0
				}
				m.cursor = 0
			}
		case "pgdown":
			if m.pageOffset+m.pageSize < len(m.filtered) {
				m.pageOffset += m.pageSize
				if m.pageOffset >= len(m.filtered) {
					m.pageOffset = len(m.filtered) - 1
				}
				m.cursor = 0
			}
		case "backspace":
			if len(m.searchValue) > 0 {
				m.searchValue = m.searchValue[:len(m.searchValue)-1]
				m.filterOptions()
			}
		case "ctrl+u": // Clear all search
			m.searchValue = ""
			m.filtered = m.options
			m.cursor = 0
			m.pageOffset = 0
		default:
			// Instant typing for search - any printable character
			if len(msg.String()) == 1 && msg.String() >= " " {
				m.searchValue += msg.String()
				m.filterOptions()
			}
		}
	}
	return m, nil
}

// Helper methods for selectModel
func (m *selectModel) filterOptions() {
	if m.searchValue == "" {
		m.filtered = m.options
	} else {
		m.filtered = []string{}
		searchLower := strings.ToLower(m.searchValue)
		for _, option := range m.options {
			if strings.Contains(strings.ToLower(option), searchLower) {
				m.filtered = append(m.filtered, option)
			}
		}
	}
	m.cursor = 0
	m.pageOffset = 0
}

func (m selectModel) getCurrentPageOptions() []string {
	start := m.pageOffset
	end := start + m.pageSize
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	if start >= len(m.filtered) {
		return []string{}
	}
	return m.filtered[start:end]
}

func (m selectModel) View() string {
	var s strings.Builder

	// Prompt with Dracula Pink
	promptStyle := lipgloss.NewStyle().Foreground(dracula.Pink).Bold(true)
	s.WriteString(promptStyle.Render(m.prompt))
	s.WriteString("\n")

	// Search indicator if there's a search term
	if len(m.searchValue) > 0 {
		searchStyle := lipgloss.NewStyle().Foreground(dracula.Yellow)
		s.WriteString(searchStyle.Render(fmt.Sprintf("Search: %s", m.searchValue)))
		s.WriteString("\n")
	}

	// Help text
	helpStyle := lipgloss.NewStyle().Foreground(dracula.Comment)
	s.WriteString(helpStyle.Render("Type to search, ↑/↓ navigate, Enter select, Esc/q quit"))
	s.WriteString("\n\n")

	currentOptions := m.getCurrentPageOptions()
	if len(currentOptions) == 0 {
		noItemsStyle := lipgloss.NewStyle().Foreground(dracula.Red)
		s.WriteString(noItemsStyle.Render("No items found"))
		s.WriteString("\n")
		return s.String()
	}

	// Display current page options with Dracula styling
	normalStyle := lipgloss.NewStyle().Foreground(dracula.Foreground)
	selectedStyle := lipgloss.NewStyle().
		Foreground(dracula.Foreground).
		Background(dracula.CurrentLine).
		Bold(true)

	for i, option := range currentOptions {
		if m.cursor == i {
			// Highlight search term in selected item
			if len(m.searchValue) > 0 {
				highlighted := m.highlightSearchTerm(option, m.searchValue)
				s.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", highlighted)))
			} else {
				s.WriteString(selectedStyle.Render(fmt.Sprintf("> %s", option)))
			}
		} else {
			// Highlight search term in normal items
			if len(m.searchValue) > 0 {
				highlighted := m.highlightSearchTerm(option, m.searchValue)
				s.WriteString(normalStyle.Render(fmt.Sprintf("  %s", highlighted)))
			} else {
				s.WriteString(normalStyle.Render(fmt.Sprintf("  %s", option)))
			}
		}
		s.WriteString("\n")
	}

	// Pagination info with Dracula Purple
	totalPages := (len(m.filtered) + m.pageSize - 1) / m.pageSize
	currentPage := (m.pageOffset / m.pageSize) + 1

	paginationStyle := lipgloss.NewStyle().Foreground(dracula.Purple)
	paginationText := fmt.Sprintf("\nPage %d/%d | Total: %d items", currentPage, totalPages, len(m.filtered))
	if len(m.filtered) != len(m.options) {
		paginationText += fmt.Sprintf(" (filtered from %d)", len(m.options))
	}
	s.WriteString(paginationStyle.Render(paginationText))

	return s.String()
}

// highlightSearchTerm highlights the search term in yellow
func (m selectModel) highlightSearchTerm(text, searchTerm string) string {
	if searchTerm == "" {
		return text
	}

	highlightStyle := lipgloss.NewStyle().Foreground(dracula.Yellow).Bold(true)
	searchLower := strings.ToLower(searchTerm)
	textLower := strings.ToLower(text)

	if idx := strings.Index(textLower, searchLower); idx != -1 {
		before := text[:idx]
		match := text[idx : idx+len(searchTerm)]
		after := text[idx+len(searchTerm):]
		return before + highlightStyle.Render(match) + after
	}

	return text
}

// inputModel handles text input UI with cursor support
type inputModel struct {
	prompt      string
	placeholder string
	value       string
	cursor      int
	cancelled   bool
	config      Config
}

func newInputModel(prompt, placeholder string, config Config) inputModel {
	return inputModel{
		prompt:      prompt,
		placeholder: placeholder,
		value:       "",
		cursor:      0,
		config:      config,
	}
}

func (m inputModel) Init() tea.Cmd {
	return nil
}

func (m inputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "left", "ctrl+b":
			if m.cursor > 0 {
				m.cursor--
			}
		case "right", "ctrl+f":
			if m.cursor < len(m.value) {
				m.cursor++
			}
		case "home", "ctrl+a":
			m.cursor = 0
		case "end", "ctrl+e":
			m.cursor = len(m.value)
		case "backspace":
			if m.cursor > 0 {
				m.value = m.value[:m.cursor-1] + m.value[m.cursor:]
				m.cursor--
			}
		case "delete", "ctrl+d":
			if m.cursor < len(m.value) {
				m.value = m.value[:m.cursor] + m.value[m.cursor+1:]
			}
		case "ctrl+w": // Delete word backwards
			if m.cursor > 0 {
				wordStart := m.cursor - 1
				for wordStart > 0 && m.value[wordStart-1] != ' ' {
					wordStart--
				}
				m.value = m.value[:wordStart] + m.value[m.cursor:]
				m.cursor = wordStart
			}
		case "ctrl+k": // Delete to end of line
			m.value = m.value[:m.cursor]
		case "ctrl+u": // Delete to beginning of line
			m.value = m.value[m.cursor:]
			m.cursor = 0
		default:
			if len(msg.String()) == 1 && msg.String() >= " " {
				// Insert character at cursor position
				m.value = m.value[:m.cursor] + msg.String() + m.value[m.cursor:]
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m inputModel) View() string {
	var s strings.Builder

	// Prompt with Dracula Pink
	promptStyle := lipgloss.NewStyle().Foreground(dracula.Pink).Bold(true)
	s.WriteString(promptStyle.Render(m.prompt))
	s.WriteString("\n")

	// Placeholder with Dracula Comment color
	if m.placeholder != "" && m.value == "" {
		placeholderStyle := lipgloss.NewStyle().Foreground(dracula.Comment).Italic(true)
		s.WriteString(placeholderStyle.Render(fmt.Sprintf("(%s)", m.placeholder)))
		s.WriteString("\n")
	}

	// Display input with cursor and Dracula styling
	inputStyle := lipgloss.NewStyle().Foreground(dracula.Foreground)
	cursorStyle := lipgloss.NewStyle().Background(dracula.Selection).Foreground(dracula.Foreground)

	displayValue := m.value
	if m.cursor <= len(displayValue) {
		// Insert cursor character at current position
		before := displayValue[:m.cursor]
		after := displayValue[m.cursor:]
		if m.cursor < len(displayValue) {
			// Cursor on character - highlight the character
			cursorChar := string(displayValue[m.cursor])
			displayValue = before + cursorStyle.Render(cursorChar) + after[1:]
		} else {
			// Cursor at end - show cursor block
			displayValue = before + cursorStyle.Render(" ")
		}
	}

	inputPrefix := lipgloss.NewStyle().Foreground(dracula.Green).Render("> ")
	s.WriteString(inputPrefix)
	s.WriteString(inputStyle.Render(displayValue))
	s.WriteString("\n")

	// Help text with Dracula Comment color
	helpStyle := lipgloss.NewStyle().Foreground(dracula.Comment)
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Navigation: ←/→ Home/End | Editing: Ctrl+W/K/U | Submit: Enter | Cancel: Esc"))
	s.WriteString("\n")

	return s.String()
}

// passwordModel handles password input UI
type passwordModel struct {
	prompt    string
	value     string
	cancelled bool
	config    Config
}

func newPasswordModel(prompt string, config Config) passwordModel {
	return passwordModel{
		prompt: prompt,
		value:  "",
		config: config,
	}
}

func (m passwordModel) Init() tea.Cmd {
	return nil
}

func (m passwordModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case "backspace":
			if len(m.value) > 0 {
				m.value = m.value[:len(m.value)-1]
			}
		default:
			if len(msg.String()) == 1 {
				m.value += msg.String()
			}
		}
	}
	return m, nil
}

func (m passwordModel) View() string {
	var s strings.Builder

	// Prompt with Dracula Pink
	promptStyle := lipgloss.NewStyle().Foreground(dracula.Pink).Bold(true)
	s.WriteString(promptStyle.Render(m.prompt))
	s.WriteString("\n")

	// Password input with masked characters
	inputPrefix := lipgloss.NewStyle().Foreground(dracula.Green).Render("> ")
	maskedStyle := lipgloss.NewStyle().Foreground(dracula.Foreground)
	s.WriteString(inputPrefix)
	s.WriteString(maskedStyle.Render(strings.Repeat("*", len(m.value))))
	s.WriteString("\n")

	// Help text with Dracula Comment color
	helpStyle := lipgloss.NewStyle().Foreground(dracula.Comment)
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Submit: Enter | Cancel: Esc"))
	s.WriteString("\n")

	return s.String()
}

// confirmModel handles confirmation UI
type confirmModel struct {
	message   string
	confirmed bool
	cancelled bool
	config    Config
}

func newConfirmModel(message string, config Config) confirmModel {
	return confirmModel{
		message: message,
		config:  config,
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "y", "Y":
			m.confirmed = true
			return m, tea.Quit
		case "n", "N", "enter":
			m.confirmed = false
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	var s strings.Builder

	// Message with Dracula Pink
	messageStyle := lipgloss.NewStyle().Foreground(dracula.Pink).Bold(true)
	s.WriteString(messageStyle.Render(m.message))

	// Options with different colors
	yesStyle := lipgloss.NewStyle().Foreground(dracula.Green).Bold(true)
	noStyle := lipgloss.NewStyle().Foreground(dracula.Red).Bold(true)
	s.WriteString(" (")
	s.WriteString(yesStyle.Render("y"))
	s.WriteString("/")
	s.WriteString(noStyle.Render("N"))
	s.WriteString(")")
	s.WriteString("\n")

	// Help text with Dracula Comment color
	helpStyle := lipgloss.NewStyle().Foreground(dracula.Comment)
	s.WriteString("\n")
	s.WriteString(helpStyle.Render("Yes: y | No: n | Cancel: Esc"))
	s.WriteString("\n")

	return s.String()
}

// multiSelectModel handles multi-selection UI
type multiSelectModel struct {
	prompt    string
	options   []string
	cursor    int
	selected  []bool
	cancelled bool
	config    Config
}

func newMultiSelectModel(prompt string, options []string, config Config) multiSelectModel {
	return multiSelectModel{
		prompt:   prompt,
		options:  options,
		cursor:   0,
		selected: make([]bool, len(options)),
		config:   config,
	}
}

func (m multiSelectModel) Init() tea.Cmd {
	return nil
}

func (m multiSelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case " ":
			if m.cursor >= 0 && m.cursor < len(m.selected) {
				m.selected[m.cursor] = !m.selected[m.cursor]
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
}

func (m multiSelectModel) View() string {
	s := fmt.Sprintf("%s\n\n", m.prompt)

	for i, option := range m.options {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		checked := " "
		if m.selected[i] {
			checked = "x"
		}

		s += fmt.Sprintf("%s [%s] %s\n", cursor, checked, option)
	}

	s += "\nPress space to select, enter to confirm, q to quit.\n"
	return s
}
