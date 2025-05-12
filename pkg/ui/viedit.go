package ui

import (
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ViMode represents the current editor mode
type ViMode int

const (
	NormalMode ViMode = iota
	InsertMode
	VisualMode
)

// ViEditor extends textinput.Model with vim-like editing capabilities
type ViEditor struct {
	textinput textinput.Model
	mode      ViMode
	
	// Clipboard for yank/paste operations
	clipboard string
	
	// Visual mode selection
	visualStart int
	visualEnd   int
	
	// Command buffer for multi-key commands
	commandBuffer []rune
	
	// History navigation
	history     []string
	historyPos  int
	tempContent string
	
	// Style
	normalModeStyle lipgloss.Style
	insertModeStyle lipgloss.Style
	visualModeStyle lipgloss.Style
}

// NewViEditor creates a new vi-like editor
func NewViEditor() *ViEditor {
	ti := textinput.New()
	ti.Placeholder = "Type a message (Esc for normal mode)..."
	ti.Focus()
	
	return &ViEditor{
		textinput: ti,
		mode:      InsertMode,
		history:   make([]string, 0),
		historyPos: -1,
		
		normalModeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1),
			
		insertModeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#43BF6D")).
			Padding(0, 1),
			
		visualModeStyle: lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#F7768E")).
			Padding(0, 1),
	}
}

// Init initializes the editor
func (e *ViEditor) Init() tea.Cmd {
	return e.textinput.Focus()
}

// Focus focuses the editor
func (e *ViEditor) Focus() tea.Cmd {
	return e.textinput.Focus()
}

// Blur blurs the editor
func (e *ViEditor) Blur() {
	e.textinput.Blur()
}

// IsFocused returns whether the editor is focused
func (e *ViEditor) IsFocused() bool {
	return e.textinput.Focused()
}

// Value returns the current text value
func (e *ViEditor) Value() string {
	return e.textinput.Value()
}

// SetValue sets the text value
func (e *ViEditor) SetValue(s string) {
	e.textinput.SetValue(s)
}

// CursorStart moves cursor to the start of the line
func (e *ViEditor) CursorStart() {
	e.textinput.CursorStart()
}

// CursorEnd moves cursor to the end of the line
func (e *ViEditor) CursorEnd() {
	e.textinput.CursorEnd()
}

// Reset clears the input
func (e *ViEditor) Reset() {
	e.textinput.Reset()
	e.commandBuffer = []rune{}
}

// SetWidth sets the width of the editor
func (e *ViEditor) SetWidth(w int) {
	e.textinput.Width = w
}

// AddToHistory adds the current input to history
func (e *ViEditor) AddToHistory(s string) {
	if s == "" {
		return
	}
	
	// Don't add duplicate entries consecutively
	if len(e.history) > 0 && e.history[len(e.history)-1] == s {
		return
	}
	
	e.history = append(e.history, s)
	e.historyPos = len(e.history)
}

// PreviousHistory navigates to the previous item in history
func (e *ViEditor) PreviousHistory() {
	if len(e.history) == 0 {
		return
	}
	
	// Save current input when starting history navigation
	if e.historyPos == len(e.history) {
		e.tempContent = e.textinput.Value()
	}
	
	if e.historyPos > 0 {
		e.historyPos--
		e.textinput.SetValue(e.history[e.historyPos])
	}
}

// NextHistory navigates to the next item in history
func (e *ViEditor) NextHistory() {
	if len(e.history) == 0 {
		return
	}
	
	if e.historyPos < len(e.history) {
		e.historyPos++
		
		if e.historyPos == len(e.history) {
			e.textinput.SetValue(e.tempContent)
		} else {
			e.textinput.SetValue(e.history[e.historyPos])
		}
	}
}

// CurrentMode returns a styled string indicating the current mode
func (e *ViEditor) CurrentMode() string {
	switch e.mode {
	case NormalMode:
		return e.normalModeStyle.Render("NORMAL")
	case InsertMode:
		return e.insertModeStyle.Render("INSERT")
	case VisualMode:
		return e.visualModeStyle.Render("VISUAL")
	default:
		return e.insertModeStyle.Render("INSERT")
	}
}

// SetNormalMode switches to normal mode
func (e *ViEditor) SetNormalMode() {
	e.mode = NormalMode
	e.commandBuffer = []rune{}
}

// SetInsertMode switches to insert mode
func (e *ViEditor) SetInsertMode() {
	e.mode = InsertMode
	e.commandBuffer = []rune{}
}

// SetVisualMode switches to visual mode
func (e *ViEditor) SetVisualMode() {
	e.mode = VisualMode
	e.visualStart = e.textinput.Position()
	e.visualEnd = e.visualStart
	e.commandBuffer = []rune{}
}

// Update handles editor updates
func (e *ViEditor) Update(msg tea.Msg) (*ViEditor, tea.Cmd) {
	var cmd tea.Cmd
	
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle mode-specific keys
		switch e.mode {
		case NormalMode:
			switch msg.String() {
			case "i":
				// Switch to insert mode
				e.SetInsertMode()
				return e, nil
				
			case "a":
				// Append after cursor
				pos := e.textinput.Position()
				if pos < len(e.textinput.Value()) {
					e.textinput.SetCursor(pos + 1)
				}
				e.SetInsertMode()
				return e, nil
				
			case "A":
				// Append at end of line
				e.textinput.CursorEnd()
				e.SetInsertMode()
				return e, nil
				
			case "0":
				// Go to beginning of line
				e.textinput.CursorStart()
				return e, nil
				
			case "$":
				// Go to end of line
				e.textinput.CursorEnd()
				return e, nil
				
			case "h":
				// Move left
				pos := e.textinput.Position()
				if pos > 0 {
					e.textinput.SetCursor(pos - 1)
				}
				return e, nil

			case "l":
				// Move right
				pos := e.textinput.Position()
				if pos < len(e.textinput.Value()) {
					e.textinput.SetCursor(pos + 1)
				}
				return e, nil
				
			case "w":
				// Move to next word
				curPos := e.textinput.Position()
				text := e.textinput.Value()
				if curPos >= len(text) {
					return e, nil
				}

				// Skip current word
				i := curPos
				for i < len(text) && text[i] != ' ' {
					i++
				}

				// Skip spaces
				for i < len(text) && text[i] == ' ' {
					i++
				}

				if i < len(text) {
					e.textinput.SetCursor(i)
				}
				return e, nil
				
			case "b":
				// Move to previous word
				curPos := e.textinput.Position()
				text := e.textinput.Value()
				if curPos <= 0 {
					return e, nil
				}
				
				// Skip spaces backwards
				i := curPos - 1
				for i > 0 && text[i] == ' ' {
					i--
				}
				
				// Skip current word backwards
				for i > 0 && text[i] != ' ' {
					i--
				}
				
				// If we hit a space, move forward one
				if i > 0 && text[i] == ' ' {
					i++
				}
				
				e.textinput.SetCursor(i)
				return e, nil
				
			case "x":
				// Delete character under cursor
				curPos := e.textinput.Position()
				text := e.textinput.Value()
				if len(text) > 0 && curPos < len(text) {
					newText := text[:curPos] + text[curPos+1:]
					e.textinput.SetValue(newText)
				}
				return e, nil
				
			case "D":
				// Delete to end of line
				curPos := e.textinput.Position()
				text := e.textinput.Value()
				if len(text) > 0 && curPos < len(text) {
					e.clipboard = text[curPos:]
					e.textinput.SetValue(text[:curPos])
				}
				return e, nil
				
			case "d":
				// Command buffer for delete operations
				e.commandBuffer = append(e.commandBuffer, 'd')
				if len(e.commandBuffer) > 1 {
					if e.commandBuffer[0] == 'd' && e.commandBuffer[1] == 'd' {
						// 'dd' - Delete whole line
						e.clipboard = e.textinput.Value()
						e.textinput.Reset()
						e.commandBuffer = []rune{}
					}
				}
				return e, nil
				
			case "y":
				// Command buffer for yank operations
				e.commandBuffer = append(e.commandBuffer, 'y')
				if len(e.commandBuffer) > 1 {
					if e.commandBuffer[0] == 'y' && e.commandBuffer[1] == 'y' {
						// 'yy' - Yank whole line
						e.clipboard = e.textinput.Value()
						e.commandBuffer = []rune{}
					}
				}
				return e, nil
				
			case "p":
				// Paste after cursor
				curPos := e.textinput.Position()
				text := e.textinput.Value()
				if e.clipboard != "" {
					if len(text) == 0 || curPos >= len(text) {
						e.textinput.SetValue(text + e.clipboard)
					} else {
						newText := text[:curPos+1] + e.clipboard + text[curPos+1:]
						e.textinput.SetValue(newText)
						e.textinput.SetCursor(curPos + len(e.clipboard))
					}
				}
				return e, nil
				
			case "P":
				// Paste before cursor
				curPos := e.textinput.Position()
				text := e.textinput.Value()
				if e.clipboard != "" {
					newText := text[:curPos] + e.clipboard + text[curPos:]
					e.textinput.SetValue(newText)
					e.textinput.SetCursor(curPos + len(e.clipboard) - 1)
				}
				return e, nil
				
			case "v":
				// Enter visual mode
				e.SetVisualMode()
				return e, nil
				
			case "j":
				// Next in history in normal mode
				e.NextHistory()
				return e, nil
				
			case "k":
				// Previous in history in normal mode
				e.PreviousHistory()
				return e, nil
				
			case "Y":
				// Yank whole line to system clipboard
				text := e.textinput.Value()
				clipboard.WriteAll(text)
				return e, nil
				
			case "ctrl+p":
				// Paste from system clipboard
				if text, err := clipboard.ReadAll(); err == nil {
					curPos := e.textinput.Position()
					currentText := e.textinput.Value()
					newText := currentText[:curPos] + text + currentText[curPos:]
					e.textinput.SetValue(newText)
					e.textinput.SetCursor(curPos + len(text))
				}
				return e, nil
			}
			
		case VisualMode:
			switch msg.String() {
			case "esc":
				// Exit visual mode
				e.SetNormalMode()
				return e, nil
				
			case "h":
				// Move selection left
				if e.visualEnd > 0 {
					e.visualEnd--
					e.textinput.SetCursor(e.visualEnd)
				}
				return e, nil
				
			case "l":
				// Move selection right
				text := e.textinput.Value()
				if e.visualEnd < len(text) {
					e.visualEnd++
					e.textinput.SetCursor(e.visualEnd)
				}
				return e, nil
				
			case "y":
				// Yank selection
				text := e.textinput.Value()
				start, end := e.visualStart, e.visualEnd
				if start > end {
					start, end = end, start
				}
				
				if end < len(text) {
					end++ // Include character under cursor
				}
				
				if start < len(text) && end <= len(text) {
					e.clipboard = text[start:end]
					clipboard.WriteAll(e.clipboard)
				}
				
				e.SetNormalMode()
				return e, nil
				
			case "d":
				// Delete selection
				text := e.textinput.Value()
				start, end := e.visualStart, e.visualEnd
				if start > end {
					start, end = end, start
				}
				
				if end < len(text) {
					end++ // Include character under cursor
				}
				
				if start < len(text) && end <= len(text) {
					e.clipboard = text[start:end]
					newText := text[:start] + text[end:]
					e.textinput.SetValue(newText)
					e.textinput.SetCursor(start)
				}
				
				e.SetNormalMode()
				return e, nil
			}
		}
		
		// Global key handlers
		switch msg.String() {
		case "esc":
			// Switch to normal mode
			if e.mode != NormalMode {
				e.SetNormalMode()
				return e, nil
			}
			
		case "ctrl+c":
			// Copy text to system clipboard in any mode
			if e.mode == VisualMode {
				text := e.textinput.Value()
				start, end := e.visualStart, e.visualEnd
				if start > end {
					start, end = end, start
				}
				
				if end < len(text) {
					end++ // Include character under cursor
				}
				
				if start < len(text) && end <= len(text) {
					clipboard.WriteAll(text[start:end])
				}
			} else {
				clipboard.WriteAll(e.textinput.Value())
			}
			return e, nil
			
		case "ctrl+v":
			// Paste from system clipboard in any mode
			if text, err := clipboard.ReadAll(); err == nil {
				if e.mode == InsertMode {
					curPos := e.textinput.Position()
					currentText := e.textinput.Value()
					newText := currentText[:curPos] + text + currentText[curPos:]
					e.textinput.SetValue(newText)
					e.textinput.SetCursor(curPos + len(text))
				}
			}
			return e, nil
			
		case "up":
			// Previous history with up arrow
			if e.mode == InsertMode {
				e.PreviousHistory()
				return e, nil
			}
			
		case "down":
			// Next history with down arrow
			if e.mode == InsertMode {
				e.NextHistory()
				return e, nil
			}
		}
	}
	
	// Pass to underlying text input if in insert mode
	if e.mode == InsertMode {
		e.textinput, cmd = e.textinput.Update(msg)
		return e, cmd
	}
	
	return e, nil
}

// View renders the editor
func (e *ViEditor) View() string {
	return e.textinput.View()
}