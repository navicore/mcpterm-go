package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/navicore/mcpterm-go/pkg/chat"
)

// Message represents a chat message
type Message struct {
	Username string
	Content  string
	IsUser   bool
}

// llmResponseMsg represents a response from the LLM
type llmResponseMsg struct {
	response chat.Message
	err      error
}

// Model represents the TUI state
type Model struct {
	viewport    viewport.Model
	editor      *ViEditor
	messages    []Message
	chatService chat.ChatServiceInterface
	err         error
	showHelp    bool
	windowWidth int

	// Focus state
	viewportFocused bool

	// Selection state
	viewportSelection *Selection
	viewportLines     []string // Viewport content split by lines
	viewportPosition  int      // Current line position in viewport
	viewportVisual    bool     // Whether visual mode is active in viewport

	// Processing state
	isProcessing bool // Whether the LLM is currently processing a response
}

// Style definitions
var (
	userMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#7D56F4")).
				PaddingLeft(2)

	botMessageStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#43BF6D")).
			PaddingLeft(2)

	processingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")). // Orange for processing indicator
			PaddingLeft(2).
			Italic(true).
			Bold(true)

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
		showHelp:          true,
		viewportFocused:   false,
		viewportSelection: NewSelection(),
		viewportLines:     []string{},
		viewportPosition:  0,
		viewportVisual:    false,
	}
}

// SetChatService sets the chat service for the model
func (m *Model) SetChatService(service chat.ChatServiceInterface) {
	m.chatService = service
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

	// Add processing indicator if LLM is generating a response
	if m.isProcessing {
		sb.WriteString(botMessageStyle.Render("Assistant:") + "\n")
		sb.WriteString(processingStyle.Render("⏳ Processing...") + "\n\n")
	}

	content := sb.String()

	// Update selection content
	m.viewportLines = strings.Split(content, "\n")
	m.viewportSelection.SetContent(m.viewportLines)

	// Set viewport content - always use highlighted content for cursor visibility
	if m.viewportFocused {
		// Always ensure cursor is visible when viewport is focused
		m.viewportSelection.CursorVisible = true

		// If cursor position isn't valid, set it to current viewport position
		if m.viewportSelection.CursorLine >= len(m.viewportLines) || m.viewportSelection.CursorLine < 0 {
			m.viewportSelection.ShowCursor(m.viewportPosition, 0)
		}

		m.viewport.SetContent(m.viewportSelection.HighlightedContent())

		// Ensure the cursor is visible in the viewport view
		// If cursor is above viewport view
		if m.viewportSelection.CursorLine < m.viewport.YOffset {
			m.viewport.SetYOffset(m.viewportSelection.CursorLine)
		}
		// If cursor is below viewport view
		if m.viewportSelection.CursorLine >= m.viewport.YOffset+m.viewport.Height {
			m.viewport.SetYOffset(m.viewportSelection.CursorLine - m.viewport.Height + 1)
		}
	} else {
		m.viewport.SetContent(content)

		// Default to bottom for new messages when not focused
		m.viewport.GotoBottom()
	}
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
			return helpStyle.Render("VIEWPORT VISUAL: hjkl: extend selection | 0/$: line start/end | y: yank | Esc: exit visual | " + commonHelp)
		} else {
			// Normal viewport help
			viewportHelp := "VIEWPORT: j/k: scroll | g/G: top/bottom | d/u: page down/up | 0/$: line start/end | v: visual"
			return helpStyle.Render(viewportHelp + " | " + commonHelp)
		}
	} else {
		// Input mode help
		if m.editor.mode == VisualMode {
			// Visual mode help for input
			return helpStyle.Render("INPUT VISUAL: h/l: extend selection | y: yank | d: delete | Esc: exit visual | " + commonHelp)
		} else if m.editor.mode == NormalMode {
			// Normal mode help
			viHelp := "INPUT NORMAL: 0/$: line start/end | hjkl: navigate | w/b: word | i/a: insert | v: visual | j/k: history"
			return helpStyle.Render(viHelp + " | " + commonHelp)
		} else {
			// Insert mode help
			viHelp := "INPUT INSERT: Esc: normal mode | ↑/↓: history | Tab to switch to message history"
			return helpStyle.Render(viHelp + " | " + commonHelp)
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
		vpCmd     tea.Cmd
		cmds      []tea.Cmd
	)

	switch msg := msg.(type) {
	case llmResponseMsg:
		// Handle LLM response
		m.isProcessing = false

		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}

		// Add bot response
		m.AddMessage(Message{
			Username: "Assistant",
			Content:  msg.response.Content,
			IsUser:   false,
		})

		return m, nil

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

				// Show cursor in viewport at a position we can see
				visibleLine := m.viewport.YOffset
				if visibleLine >= len(m.viewportLines) {
					visibleLine = len(m.viewportLines) - 1
					if visibleLine < 0 {
						visibleLine = 0
					}
				}

				// Store the current position for j/k navigation
				m.viewportPosition = visibleLine
				m.viewportSelection.ShowCursor(visibleLine, 0)
				// Make sure the cursor is visible and the selection is not active
				m.viewportVisual = false
				m.viewportSelection.Active = false
				m.viewportSelection.CursorVisible = true
				m.updateViewportContent()
			} else {
				// Always keep viewport cursor visible even when switching to editor
				m.viewportSelection.CursorVisible = true
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

					// Add user message immediately
					m.AddMessage(Message{
						Content: userMsg,
						IsUser:  true,
					})

					// Clear editor
					m.editor.Reset()

					// Set processing indicator
					m.isProcessing = true
					m.updateViewportContent()

					// Process message in the background
					return m, func() tea.Msg {
						response, err := m.chatService.SendMessage(userMsg)
						return llmResponseMsg{
							response: response,
							err:      err,
						}
					}
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
					// Make sure cursor remains visible
					m.viewportSelection.CursorVisible = true
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
					if m.viewportSelection.CursorCol > 0 {
						m.viewportSelection.MoveCursor(0, -1)
						m.viewportSelection.Update(m.viewportSelection.CursorLine, m.viewportSelection.CursorCol)
						_ = m.viewportSelection.End()
						m.updateViewportContent()
					}
					return m, nil

				case "l":
					// Extend selection right
					if m.viewportSelection.CursorLine < len(m.viewportLines) {
						line := m.viewportLines[m.viewportSelection.CursorLine]
						if m.viewportSelection.CursorCol < len(line) {
							m.viewportSelection.MoveCursor(0, 1)
							m.viewportSelection.Update(m.viewportSelection.CursorLine, m.viewportSelection.CursorCol)
							_ = m.viewportSelection.End()
							m.updateViewportContent()
						}
					}
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

				case "w":
					// Move to start of next word
					if m.viewportSelection.CursorLine < len(m.viewportLines) {
						line := m.viewportLines[m.viewportSelection.CursorLine]
						curCol := m.viewportSelection.CursorCol

						// Handle cursor at the end of line - move to next line
						if curCol >= len(line) {
							if m.viewportSelection.CursorLine+1 < len(m.viewportLines) {
								// Move to start of next line
								m.viewportSelection.MoveCursor(1, 0)
								m.viewportSelection.CursorCol = 0
								m.viewportSelection.Update(m.viewportSelection.CursorLine, 0)
								_ = m.viewportSelection.End()
							}
						} else {
							// Find next word boundary
							isSpace := func(r byte) bool { return r == ' ' || r == '\t' || r == '\n' }
							foundSpace := false
							i := curCol

							// Skip current word if in the middle of one
							for i < len(line) && !isSpace(line[i]) {
								i++
							}

							// Skip spaces to the next word
							for i < len(line) && isSpace(line[i]) {
								i++
								foundSpace = true
							}

							// If we found a next word, move to it
							if i < len(line) {
								m.viewportSelection.MoveCursor(0, i-curCol)
								m.viewportSelection.Update(m.viewportSelection.CursorLine, i)
								_ = m.viewportSelection.End()
							} else if foundSpace {
								// If we only found spaces at the end, go to end of line
								m.viewportSelection.MoveCursor(0, len(line)-curCol)
								m.viewportSelection.Update(m.viewportSelection.CursorLine, len(line))
								_ = m.viewportSelection.End()
							} else if m.viewportSelection.CursorLine+1 < len(m.viewportLines) {
								// If at end of line and no spaces found, go to next line
								m.viewportSelection.MoveCursor(1, 0)
								m.viewportSelection.CursorCol = 0
								m.viewportSelection.Update(m.viewportSelection.CursorLine, 0)
								_ = m.viewportSelection.End()
							}
						}

						m.updateViewportContent()
					}
					return m, nil

				case "b":
					// Move to start of previous word
					curLine := m.viewportSelection.CursorLine
					curCol := m.viewportSelection.CursorCol

					if curCol == 0 && curLine > 0 {
						// At start of line, move to previous line's end
						prevLine := m.viewportLines[curLine-1]
						m.viewportSelection.MoveCursor(-1, len(prevLine))
						m.viewportSelection.Update(curLine-1, len(prevLine))
						_ = m.viewportSelection.End()
						m.updateViewportContent()
					} else if curLine < len(m.viewportLines) {
						// Move within current line
						line := m.viewportLines[curLine]
						isSpace := func(r byte) bool { return r == ' ' || r == '\t' || r == '\n' }
						i := curCol

						// Move back one character to start if we're at a word boundary
						if i > 0 && i < len(line) && !isSpace(line[i]) && isSpace(line[i-1]) {
							i--
						}

						// Skip any spaces backward
						for i > 0 && isSpace(line[i-1]) {
							i--
						}

						// Skip the current word backward
						for i > 0 && !isSpace(line[i-1]) {
							i--
						}

						m.viewportSelection.MoveCursor(0, i-curCol)
						m.viewportSelection.Update(curLine, i)
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
						// Make sure cursor remains visible after yanking
						m.viewportSelection.CursorVisible = true
						m.viewportSelection.ShowCursor(cursorLine, cursorCol)
						m.updateViewportContent()
					}
					return m, nil
				}
			} else {
				// Normal mode commands with cursor movement
				switch msg.String() {
				case "j":
					// Move down one line and synchronize cursor position
					m.viewport.ScrollDown(1)

					// Update our position tracker to follow viewport
					m.viewportPosition = m.viewport.YOffset

					// Make sure cursor is at the current line
					if m.viewportPosition < len(m.viewportLines) {
						m.viewportSelection.ShowCursor(m.viewportPosition, 0)
					}

					// Update display
					m.updateViewportContent()
					return m, nil

				case "k":
					// Move up one line and synchronize cursor position
					m.viewport.ScrollUp(1)

					// Update our position tracker to follow viewport
					m.viewportPosition = m.viewport.YOffset

					// Make sure cursor is at the current line
					if m.viewportPosition < len(m.viewportLines) {
						m.viewportSelection.ShowCursor(m.viewportPosition, 0)
					}

					// Update display
					m.updateViewportContent()
					return m, nil

				case "h":
					// Move cursor left
					// Keep the same line but adjust column
					oldCol := m.viewportSelection.CursorCol
					if oldCol > 0 {
						m.viewportSelection.ShowCursor(m.viewportPosition, oldCol-1)
					}
					m.updateViewportContent()
					return m, nil

				case "l":
					// Move cursor right
					// Keep the same line but adjust column
					oldCol := m.viewportSelection.CursorCol
					if m.viewportPosition < len(m.viewportLines) {
						line := m.viewportLines[m.viewportPosition]
						if oldCol < len(line) {
							m.viewportSelection.ShowCursor(m.viewportPosition, oldCol+1)
						}
					}
					m.updateViewportContent()
					return m, nil

				case "g":
					// Go to the top of the viewport and set cursor there
					m.viewport.GotoTop()
					m.viewportPosition = 0

					// Make cursor visible at the top and ensure it's set to line 0
					m.viewportSelection.ShowCursor(0, 0)

					// Force YOffset to 0 to ensure we're truly at the top
					m.viewport.SetYOffset(0)

					m.updateViewportContent()
					return m, nil

				case "G":
					// Go to the bottom of the viewport
					m.viewport.GotoBottom()

					// Set position to last line
					lastLine := 0
					if len(m.viewportLines) > 0 {
						lastLine = len(m.viewportLines) - 1
						m.viewportPosition = lastLine

						// Calculate offset to ensure the cursor is visible
						// This sets YOffset so the last line appears at the bottom of the viewport
						if m.viewport.Height < len(m.viewportLines) {
							m.viewport.SetYOffset(lastLine - m.viewport.Height + 1)
							if m.viewport.YOffset < 0 {
								m.viewport.SetYOffset(0)
							}
						}
					}

					// Show cursor on the last line
					m.viewportSelection.ShowCursor(lastLine, 0)

					m.updateViewportContent()
					return m, nil

				case "d":
					// Half page down
					m.viewport.HalfPageDown()

					// Update tracking position to match viewport
					m.viewportPosition = m.viewport.YOffset

					// Show cursor at the new position with same column
					curCol := m.viewportSelection.CursorCol
					m.viewportSelection.ShowCursor(m.viewport.YOffset, curCol)

					// Make sure y-offset is actually updated
					m.viewport.SetYOffset(m.viewportPosition)

					m.updateViewportContent()
					return m, nil

				case "u":
					// Half page up
					m.viewport.HalfPageUp()

					// Update tracking position to match viewport
					m.viewportPosition = m.viewport.YOffset

					// Show cursor at the new position with same column
					curCol := m.viewportSelection.CursorCol
					m.viewportSelection.ShowCursor(m.viewport.YOffset, curCol)

					// Make sure y-offset is actually updated
					m.viewport.SetYOffset(m.viewportPosition)

					m.updateViewportContent()
					return m, nil

				case "v":
					// Enter visual mode at cursor position
					// Make sure cursor is visible and position is valid
					if !m.viewportSelection.CursorVisible {
						m.viewportSelection.CursorVisible = true
						m.viewportSelection.ShowCursor(m.viewportPosition, 0)
					}
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

				case "w":
					// Move to start of next word in normal mode
					if m.viewportSelection.CursorLine < len(m.viewportLines) {
						line := m.viewportLines[m.viewportSelection.CursorLine]
						curCol := m.viewportSelection.CursorCol

						// Handle cursor at the end of line
						if curCol >= len(line) {
							if m.viewportSelection.CursorLine+1 < len(m.viewportLines) {
								// Move to start of next line
								m.viewportPosition = m.viewportSelection.CursorLine + 1
								m.viewportSelection.ShowCursor(m.viewportPosition, 0)

								// Make sure the cursor is visible
								if m.viewportPosition < m.viewport.YOffset ||
									m.viewportPosition >= m.viewport.YOffset+m.viewport.Height {
									m.viewport.SetYOffset(m.viewportPosition)
								}

								m.updateViewportContent()
							}
						} else {
							// Find next word boundary
							isSpace := func(r byte) bool { return r == ' ' || r == '\t' || r == '\n' }
							foundSpace := false
							i := curCol

							// Skip current word if in the middle of one
							for i < len(line) && !isSpace(line[i]) {
								i++
							}

							// Skip spaces to the next word
							for i < len(line) && isSpace(line[i]) {
								i++
								foundSpace = true
							}

							if i < len(line) {
								// Found next word
								m.viewportSelection.ShowCursor(m.viewportSelection.CursorLine, i)
							} else if foundSpace {
								// Found only spaces at the end
								m.viewportSelection.ShowCursor(m.viewportSelection.CursorLine, len(line))
							} else if m.viewportSelection.CursorLine+1 < len(m.viewportLines) {
								// At end of line, go to next line
								m.viewportPosition = m.viewportSelection.CursorLine + 1
								m.viewportSelection.ShowCursor(m.viewportPosition, 0)

								// Make sure the new line is visible
								if m.viewportPosition < m.viewport.YOffset ||
									m.viewportPosition >= m.viewport.YOffset+m.viewport.Height {
									m.viewport.SetYOffset(m.viewportPosition)
								}
							}

							m.updateViewportContent()
						}
					}
					return m, nil

				case "b":
					// Move to start of previous word in normal mode
					curLine := m.viewportSelection.CursorLine
					curCol := m.viewportSelection.CursorCol

					if curCol == 0 && curLine > 0 {
						// At start of line, move to previous line
						m.viewportPosition = curLine - 1
						prevLine := m.viewportLines[m.viewportPosition]
						m.viewportSelection.ShowCursor(m.viewportPosition, len(prevLine))

						// Make sure the cursor is visible
						if m.viewportPosition < m.viewport.YOffset ||
							m.viewportPosition >= m.viewport.YOffset+m.viewport.Height {
							m.viewport.SetYOffset(m.viewportPosition)
						}

						m.updateViewportContent()
					} else if curLine < len(m.viewportLines) {
						// Move within current line
						line := m.viewportLines[curLine]
						isSpace := func(r byte) bool { return r == ' ' || r == '\t' || r == '\n' }
						i := curCol

						// Move back one character if at a word boundary
						if i > 0 && i < len(line) && !isSpace(line[i]) && isSpace(line[i-1]) {
							i--
						}

						// Skip spaces backward
						for i > 0 && isSpace(line[i-1]) {
							i--
						}

						// Skip the word backward
						for i > 0 && !isSpace(line[i-1]) {
							i--
						}

						m.viewportSelection.ShowCursor(curLine, i)
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
		// Don't pass navigation key events to viewport when input is focused
		skipViewportUpdate := false
		if msg, ok := msg.(tea.KeyMsg); ok {
			// Skip viewport update for navigation keys regardless of editor mode when input is focused
			switch msg.String() {
			case "j", "k", "g", "G", "d", "u":
				// These keys would otherwise cause unwanted scrolling
				skipViewportUpdate = true
			}
		}

		m.editor, editorCmd = m.editor.Update(msg)
		cmds = append(cmds, editorCmd)

		// Skip viewport update for navigation keys to avoid scrolling viewport while typing
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
