package ui

import (
	"os"
	"strings"
	"syscall"
	"time"
	"unicode"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

const maxInputChars = 3000

type editorModel struct {
	textarea     textarea.Model
	prompt       string
	result       string
	ctrlCPressed bool
}

func initialEditorModel(prompt string) editorModel {
	ta := textarea.New()
	ta.CharLimit = maxInputChars
	ta.ShowLineNumbers = false
	ta.Prompt = ""
	ta.Placeholder = ""
	ta.KeyMap.InsertNewline.SetEnabled(false)

	w := TerminalWidth()
	if w < 20 {
		w = 80
	}
	ta.SetWidth(w - 4)
	ta.SetHeight(6)
	ta.Focus()

	focusedStyle, _ := textarea.DefaultStyles()
	ta.FocusedStyle = focusedStyle

	return editorModel{
		textarea: ta,
		prompt:   prompt,
	}
}

func (m editorModel) Init() tea.Cmd {
	return textarea.Blink
}

func (m editorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			m.ctrlCPressed = true
			return m, tea.Quit
		case tea.KeyEnter:
			if strings.HasSuffix(m.textarea.Value(), "\n") {
				m.result = m.textarea.Value()
				return m, tea.Quit
			}
			m.textarea.InsertRune('\n')
			return m, nil
		case tea.KeyCtrlW:
			deleteWord(&m)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m editorModel) View() string {
	return Colorize(m.prompt, ColorCyan+ColorBold) + "\n\n" + m.textarea.View()
}

func deleteWord(m *editorModel) {
	val := m.textarea.Value()
	if val == "" {
		return
	}

	runes := []rune(val)
	end := len(runes)

	if end == 0 {
		return
	}

	// Trim trailing whitespace
	start := end
	for start > 0 && unicode.IsSpace(runes[start-1]) {
		start--
	}
	// Trim word
	for start > 0 && !unicode.IsSpace(runes[start-1]) {
		start--
	}

	m.textarea.SetValue(string(runes[:start]) + string(runes[end:]))
}

// runEditor runs a Bubble Tea textarea editor and returns the submitted text.
func runEditor(prompt string) (string, error) {
	m := initialEditorModel(prompt)
	p := tea.NewProgram(m)

	finalModel, err := p.Run()
	if err != nil {
		return "", err
	}

	m = finalModel.(editorModel)

	if m.ctrlCPressed {
		syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}

	return strings.TrimSpace(m.result), nil
}
