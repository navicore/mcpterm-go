package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"mcpterm-go/pkg/chat"
)

// Message represents a chat message
type Message struct {
	Username string
	Content  string
	IsUser   bool
}

// Model represents the TUI state
type Model struct {
	viewport      viewport.Model
	editor        *ViEditor
	messages      []Message
	chatService   chat.ChatService
	err           error
	showHelp      bool
	windowWidth   int

	// Focus state
	viewportFocused bool

	// Selection state
	viewportSelection *Selection
	viewportLines     []string        // Viewport content split by lines
	viewportPosition  int             // Current line position in viewport
	viewportVisual    bool            // Whether visual mode is active in viewport
}

// Style definitions
var (
	userMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7D56F4")).
			PaddingLeft(2)

	botMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D")).
			PaddingLeft(2)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#626262")).
			PaddingLeft(2)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87"))
			
	// Status indicators
	normalModeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginRight(1).
			Bold(true).
			Render("NORMAL")
			
	insertModeIndicator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#43BF6D")).
			Padding(0, 1).
			MarginRight(1).
			Bold(true).
			Render("INSERT")
)

// NewModel creates a new TUI model
func NewModel() Model {
	vp := viewport.New(0, 0)
	vp.KeyMap.PageDown.SetEnabled(true)
	vp.KeyMap.PageUp.SetEnabled(true)

	editor := NewViEditor()
	editor.SetInsertMode()

	return Model{
		viewport:          vp,
		editor:            editor,
		messages:          []Message{},
		chatService:       chat.NewSimpleChatService(),
		showHelp:          true,
		viewportFocused:   false,
		viewportSelection: NewSelection(),
		viewportLines:     []string{},
		viewportPosition:  0,
		viewportVisual:    false,
	}
}

// AddMessage adds a message to the chat history
func (m *Model) AddMessage(msg Message) {
	m.messages = append(m.messages, msg)
	m.updateViewportContent()
}

// updateViewportContent updates the viewport content
func (m *Model) updateViewportContent() {
	var sb strings.Builder

	// Initialize the glamour renderer for markdown
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(m.windowWidth-4),
	)

	if err != nil {
		m.err = err
		return
	}

	for _, msg := range m.messages {
		// Add the username header
		if msg.IsUser {
			sb.WriteString(userMessageStyle.Render("You:") + "\n")
		} else {
			sb.WriteString(botMessageStyle.Render(msg.Username+":") + "\n")
		}

		// Render the message content as markdown
		mdContent, err := renderer.Render(msg.Content)
		if err != nil {
			// Fallback to plain text if markdown rendering fails
			if msg.IsUser {
				sb.WriteString(userMessageStyle.Render(msg.Content) + "\n\n")
			} else {
				sb.WriteString(botMessageStyle.Render(msg.Content) + "\n\n")
			}
		} else {
			// Add the rendered markdown content
			sb.WriteString(mdContent + "\n")
		}

		// Add extra spacing between messages
		sb.WriteString("\n")
	}

	content := sb.String()

	// Update selection content
	m.viewportLines = strings.Split(content, "\n")
	m.viewportSelection.SetContent(m.viewportLines)

	// Set viewport content - always use highlighted content for cursor visibility
	if m.viewportFocused {
		m.viewport.SetContent(m.viewportSelection.HighlightedContent())
	} else {
		m.viewport.SetContent(content)
	}

	m.viewport.GotoBottom()
}

// renderHelp renders keyboard shortcuts help
func (m Model) renderHelp() string {
	if !m.showHelp {
		return ""
	}

	commonHelp := "Tab: switch focus | Ctrl+C: quit | Ctrl+h: toggle help"

	if m.viewportFocused {
		// Viewport mode help
		if m.viewportVisual {
			// Visual mode help for viewport
			return helpStyle.Render("VISUAL: hjkl: extend selection | 0/$: line start/end | y: yank | Esc: exit visual | " + commonHelp)
		} else {
			// Normal viewport help
			viewportHelp := "hjkl: move cursor | j/k: scroll | 0/$: line start/end | g/G: top/bottom | v: visual mode"
			return helpStyle.Render(viewportHelp + " | " + commonHelp)
		}
	} else {
		// Input mode help
		if m.editor.mode == VisualMode {
			// Visual mode help for input
			return helpStyle.Render("VISUAL: h/l: extend selection | y: yank | d: delete | Esc: exit visual | " + commonHelp)
		} else {
			// Normal/insert mode help
			viHelp := "Normal: 0/$: line start/end | hjkl: navigate | w/b: word movement | i/a: insert | v: visual"
			return helpStyle.Render("Esc: normal mode | ↑/↓: history | " + commonHelp + "\n" + helpStyle.Render(viHelp))
		}
	}
}

// Init implements tea.Model
func (m Model) Init() tea.Cmd {
	return m.editor.Init()
}

// Update implements tea.Model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		editorCmd tea.Cmd
		vpCmd tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		
		case "ctrl+h":
			m.showHelp = !m.showHelp
			return m, nil
		
		case "tab":
			// Toggle focus between viewport and editor
			m.viewportFocused = !m.viewportFocused
			if m.viewportFocused {
				m.editor.Blur()

				// Show cursor in viewport at current position
				currentLine := m.viewport.YOffset
				m.viewportSelection.ShowCursor(currentLine, 0)
				m.updateViewportContent()
			} else {
				// Hide viewport cursor when switching to editor
				m.viewportSelection.HideCursor()
				m.updateViewportContent()
				editorCmd = m.editor.Focus()
			}
			return m, editorCmd
			
		case "enter":
			// Only process Enter if editor is focused and in insert mode
			if !m.viewportFocused {
				userMsg := strings.TrimSpace(m.editor.Value())
				if userMsg != "" {
					// Add to input history
					m.editor.AddToHistory(userMsg)
					
					// Add user message
					m.AddMessage(Message{
						Content: userMsg,
						IsUser:  true,
					})

					// Clear editor
					m.editor.Reset()

					// Send message to chat service
					response, err := m.chatService.SendMessage(userMsg)
					if err != nil {
						m.err = err
						return m, nil
					}

					// Add bot response
					m.AddMessage(Message{
						Username: response.Sender,
						Content:  response.Content,
						IsUser:   false,
					})
				}
				return m, nil
			}
		}
		
		// Handle keyboard shortcuts when viewport is focused
		if m.viewportFocused {
			// First, handle commands for both normal and visual modes
			switch msg.String() {
			case "esc":
				if m.viewportVisual {
					// Exit visual mode but keep cursor
					m.viewportVisual = false
					m.viewportSelection.Active = false
					cursorLine := m.viewportSelection.CursorLine
					cursorCol := m.viewportSelection.CursorCol
					m.viewportSelection.HideCursor()
					m.viewportSelection.ShowCursor(cursorLine, cursorCol)
					m.updateViewportContent()
					return m, nil
				}
			}

			// Visual mode specific commands
			if m.viewportVisual {
				switch msg.String() {
				case "j":
					// Extend selection down
					m.viewport.ScrollDown(1)
					cursorLine := m.viewportSelection.CursorLine + 1
					if cursorLine >= len(m.viewportLines) {
						cursorLine = len(m.viewportLines) - 1
					}
					m.viewportSelection.MoveCursor(1, 0)
					m.viewportSelection.Update(cursorLine, m.viewportSelection.CursorCol)
					_ = m.viewportSelection.End() // Update selected text
					m.updateViewportContent()
					return m, nil

				case "k":
					// Extend selection up
					m.viewport.ScrollUp(1)
					cursorLine := m.viewportSelection.CursorLine - 1
					if cursorLine < 0 {
						cursorLine = 0
					}
					m.viewportSelection.MoveCursor(-1, 0)
					m.viewportSelection.Update(cursorLine, m.viewportSelection.CursorCol)
					_ = m.viewportSelection.End() // Update selected text
					m.updateViewportContent()
					return m, nil

				case "h":
					// Extend selection left
					m.viewportSelection.MoveCursor(0, -1)
					m.viewportSelection.Update(m.viewportSelection.CursorLine, m.viewportSelection.CursorCol)
					_ = m.viewportSelection.End()
					m.updateViewportContent()
					return m, nil

				case "l":
					// Extend selection right
					m.viewportSelection.MoveCursor(0, 1)
					m.viewportSelection.Update(m.viewportSelection.CursorLine, m.viewportSelection.CursorCol)
					_ = m.viewportSelection.End()
					m.updateViewportContent()
					return m, nil

				case "0":
					// Move cursor to start of line in visual mode
					m.viewportSelection.CursorCol = 0
					m.viewportSelection.Update(m.viewportSelection.CursorLine, 0)
					_ = m.viewportSelection.End()
					m.updateViewportContent()
					return m, nil

				case "$":
					// Move cursor to end of line in visual mode
					if m.viewportSelection.CursorLine < len(m.viewportLines) {
						line := m.viewportLines[m.viewportSelection.CursorLine]
						m.viewportSelection.CursorCol = len(line)
						m.viewportSelection.Update(m.viewportSelection.CursorLine, len(line))
						_ = m.viewportSelection.End()
						m.updateViewportContent()
					}
					return m, nil

				case "y":
					// Yank (copy) selection
					text := m.viewportSelection.GetSelectedText()
					if text != "" {
						m.viewportSelection.CopyToClipboard()
						m.viewportVisual = false
						m.viewportSelection.Active = false
						// Keep cursor position
						cursorLine := m.viewportSelection.CursorLine
						cursorCol := m.viewportSelection.CursorCol
						m.viewportSelection.HideCursor()
						m.viewportSelection.ShowCursor(cursorLine, cursorCol)
						m.updateViewportContent()
					}
					return m, nil
				}
			} else {
				// Normal mode commands with cursor movement
				switch msg.String() {
				case "j":
					m.viewport.ScrollDown(1)
					m.viewportPosition = m.viewport.YOffset
					m.viewportSelection.MoveCursor(1, 0)
					m.updateViewportContent()
					return m, nil

				case "k":
					m.viewport.ScrollUp(1)
					m.viewportPosition = m.viewport.YOffset
					m.viewportSelection.MoveCursor(-1, 0)
					m.updateViewportContent()
					return m, nil

				case "h":
					// Move cursor left
					m.viewportSelection.MoveCursor(0, -1)
					m.updateViewportContent()
					return m, nil

				case "l":
					// Move cursor right
					m.viewportSelection.MoveCursor(0, 1)
					m.updateViewportContent()
					return m, nil

				case "g":
					m.viewport.GotoTop()
					m.viewportPosition = 0
					m.viewportSelection.ShowCursor(0, 0)
					m.updateViewportContent()
					return m, nil

				case "G":
					m.viewport.GotoBottom()
					// Set position to last line
					lastLine := 0
					if len(m.viewportLines) > 0 {
						lastLine = len(m.viewportLines) - 1
						m.viewportPosition = lastLine
					}
					m.viewportSelection.ShowCursor(lastLine, 0)
					m.updateViewportContent()
					return m, nil

				case "d":
					m.viewport.HalfPageDown()
					m.viewportPosition = m.viewport.YOffset
					m.viewportSelection.ShowCursor(m.viewport.YOffset, m.viewportSelection.CursorCol)
					m.updateViewportContent()
					return m, nil

				case "u":
					m.viewport.HalfPageUp()
					m.viewportPosition = m.viewport.YOffset
					m.viewportSelection.ShowCursor(m.viewport.YOffset, m.viewportSelection.CursorCol)
					m.updateViewportContent()
					return m, nil

				case "v":
					// Enter visual mode at cursor position
					m.viewportVisual = true
					cursorLine := m.viewportSelection.CursorLine
					cursorCol := m.viewportSelection.CursorCol
					m.viewportSelection.Start(cursorLine, cursorCol)
					m.updateViewportContent()
					return m, nil

				case "0":
					// Move cursor to beginning of line
					m.viewportSelection.CursorCol = 0
					m.updateViewportContent()
					return m, nil

				case "$":
					// Move cursor to end of line
					if m.viewportSelection.CursorLine < len(m.viewportLines) {
						line := m.viewportLines[m.viewportSelection.CursorLine]
						m.viewportSelection.CursorCol = len(line)
						m.updateViewportContent()
					}
					return m, nil
				}
			}
		}
	
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width

		// Adjust viewport height to leave room for input and help text
		headerHeight := 2
		footerHeight := 5

		if m.showHelp {
			footerHeight += 2 // Extra line for detailed vi help
		}

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - headerHeight - footerHeight

		m.editor.SetWidth(msg.Width - 4)

		// Update viewport content after resize
		m.updateViewportContent()

		return m, nil
	}

	// Handle editor events if not in viewport mode
	if !m.viewportFocused {
		// Check for j/k in editor normal mode which should only affect editor, not viewport
		skipViewportUpdate := false
		if msg, ok := msg.(tea.KeyMsg); ok {
			if m.editor.mode == ViMode(0) && (msg.String() == "j" || msg.String() == "k") {
				skipViewportUpdate = true
			}
		}

		m.editor, editorCmd = m.editor.Update(msg)
		cmds = append(cmds, editorCmd)

		// Skip viewport update for j/k in normal mode to avoid scrolling viewport
		if skipViewportUpdate {
			return m, tea.Batch(cmds...)
		}
	}

	// Handle viewport events (except when j/k in editor normal mode)
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// renderModeIndicator renders the current mode indicator
func (m Model) renderModeIndicator() string {
	focusText := "INPUT"
	if m.viewportFocused {
		focusText = "VIEWPORT"
	}

	focusIndicator := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#AAAAAA")).
		Render(focusText)

	modeIndicator := m.editor.CurrentMode()
	if m.viewportFocused && m.viewportVisual {
		// Visual mode in viewport
		visualStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#F7768E")).
			Padding(0, 1).
			MarginRight(1).
			Bold(true)
		modeIndicator = visualStyle.Render("VISUAL")
	}

	return modeIndicator + " " + focusIndicator
}

// View implements tea.Model
func (m Model) View() string {
	if m.err != nil {
		return errorStyle.Render(m.err.Error())
	}

	// Style for status line
	_ = strings.Repeat("─", m.windowWidth) // Not used but kept for reference

	// Build the status line with mode indicator
	statusLine := fmt.Sprintf("%s   %s",
		m.renderModeIndicator(),
		lipgloss.NewStyle().
			Foreground(lipgloss.Color("#AAAAAA")).
			Render("MCPTerm Chat"))
			
	// Highlight the viewport when focused
	var viewportView string
	if m.viewportFocused {
		viewportView = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("69")).
			Render(m.viewport.View())
	} else {
		viewportView = m.viewport.View()
	}
	
	// Combine the viewport, status line, input field, and help text
	helpText := m.renderHelp()

	return fmt.Sprintf(
		"%s\n%s\n\n%s\n\n%s",
		viewportView,
		statusLine,
		m.editor.View(),
		helpText,
	)
}