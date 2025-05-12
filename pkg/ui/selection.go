package ui

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/lipgloss"
)

// Selection represents a text selection in the viewport
type Selection struct {
	Active      bool
	StartLine   int
	StartCol    int
	EndLine     int
	EndCol      int
	Content     []string  // Content of the viewport split by lines
	SelectedText string   // The most recently selected text

	// Cursor for visual indication
	CursorVisible bool
	CursorLine    int
	CursorCol     int
}

// NewSelection creates a new empty selection
func NewSelection() *Selection {
	return &Selection{
		Active:        false,
		StartLine:     0,
		StartCol:      0,
		EndLine:       0,
		EndCol:        0,
		Content:       []string{},
		CursorVisible: false,
		CursorLine:    0,
		CursorCol:     0,
	}
}

// Start begins a new selection at the given position
func (s *Selection) Start(line, col int) {
	s.Active = true
	s.StartLine = line
	s.StartCol = col
	s.EndLine = line
	s.EndCol = col

	// Set cursor at the same position
	s.CursorVisible = true
	s.CursorLine = line
	s.CursorCol = col
}

// ShowCursor makes the cursor visible at the specified position
func (s *Selection) ShowCursor(line, col int) {
	s.CursorVisible = true
	s.CursorLine = line
	s.CursorCol = col
}

// HideCursor hides the cursor
func (s *Selection) HideCursor() {
	s.CursorVisible = false
}

// MoveCursor moves the cursor to a new position
func (s *Selection) MoveCursor(lineOffset, colOffset int) {
	if !s.CursorVisible {
		return
	}

	// Update cursor position
	s.CursorLine += lineOffset
	s.CursorCol += colOffset

	// Ensure cursor stays within bounds
	if s.CursorLine < 0 {
		s.CursorLine = 0
	}
	if s.CursorLine >= len(s.Content) {
		s.CursorLine = len(s.Content) - 1
		if s.CursorLine < 0 {
			s.CursorLine = 0
		}
	}

	if s.CursorCol < 0 {
		s.CursorCol = 0
	}
	if s.CursorLine < len(s.Content) && s.CursorCol > len(s.Content[s.CursorLine]) {
		s.CursorCol = len(s.Content[s.CursorLine])
	}
}

// Update updates the end position of the selection
func (s *Selection) Update(line, col int) {
	s.EndLine = line
	s.EndCol = col
}

// End finalizes the selection and extracts the selected text
func (s *Selection) End() string {
	if !s.Active || len(s.Content) == 0 {
		return ""
	}
	
	// Make sure start is before end
	startLine, endLine := s.StartLine, s.EndLine
	startCol, endCol := s.StartCol, s.EndCol
	
	if startLine > endLine || (startLine == endLine && startCol > endCol) {
		startLine, endLine = endLine, startLine
		startCol, endCol = endCol, startCol
	}
	
	// Adjust line indices to be within bounds
	if startLine < 0 {
		startLine = 0
		startCol = 0
	}
	if startLine >= len(s.Content) {
		startLine = len(s.Content) - 1
		if startLine < 0 {
			startLine = 0
		}
	}
	if endLine < 0 {
		endLine = 0
	}
	if endLine >= len(s.Content) {
		endLine = len(s.Content) - 1
	}
	
	// Extract the selected text
	var result strings.Builder
	
	for i := startLine; i <= endLine; i++ {
		line := s.Content[i]
		
		// Adjust column indices
		start := 0
		end := len(line)
		
		if i == startLine {
			start = startCol
			if start > len(line) {
				start = len(line)
			}
		}
		
		if i == endLine {
			end = endCol
			if end > len(line) {
				end = len(line)
			}
		}
		
		// Append the selected portion of this line
		if start <= end && end <= len(line) {
			result.WriteString(line[start:end])
		}
		
		// Add newline if not the last line
		if i < endLine {
			result.WriteString("\n")
		}
	}
	
	s.SelectedText = result.String()
	return s.SelectedText
}

// Clear clears the selection
func (s *Selection) Clear() {
	s.Active = false
	s.CursorVisible = false
}

// CopyToClipboard copies the selected text to the system clipboard
func (s *Selection) CopyToClipboard() {
	if s.SelectedText != "" {
		clipboard.WriteAll(s.SelectedText)
	}
}

// IsActive returns whether there is an active selection
func (s *Selection) IsActive() bool {
	return s.Active
}

// HighlightedContent returns the content with selection highlighted
func (s *Selection) HighlightedContent() string {
	var result strings.Builder
	highlightStyle := lipgloss.NewStyle().Background(lipgloss.Color("#3B7EAA")).Foreground(lipgloss.Color("#FFFFFF"))
	cursorStyle := lipgloss.NewStyle().Background(lipgloss.Color("#FF5F87")).Foreground(lipgloss.Color("#FFFFFF"))

	if len(s.Content) == 0 {
		return ""
	}

	// Handle active selection highlighting
	if s.Active {
		// Make sure start is before end
		startLine, endLine := s.StartLine, s.EndLine
		startCol, endCol := s.StartCol, s.EndCol

		if startLine > endLine || (startLine == endLine && startCol > endCol) {
			startLine, endLine = endLine, startLine
			startCol, endCol = endCol, startCol
		}

		// Adjust indices to be within bounds
		if startLine < 0 {
			startLine = 0
		}
		if startLine >= len(s.Content) {
			startLine = len(s.Content) - 1
		}
		if endLine < 0 {
			endLine = 0
		}
		if endLine >= len(s.Content) {
			endLine = len(s.Content) - 1
		}

		// Process each line
		for i, line := range s.Content {
			if s.CursorVisible && i == s.CursorLine {
				// This line contains the cursor
				cursorCol := s.CursorCol
				if cursorCol < 0 {
					cursorCol = 0
				}
				if cursorCol > len(line) {
					cursorCol = len(line)
				}

				// Handle selection and cursor together
				if i < startLine || i > endLine {
					// Line has cursor but no selection
					if cursorCol > 0 {
						result.WriteString(line[:cursorCol])
					}
					if cursorCol < len(line) {
						result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
						if cursorCol+1 < len(line) {
							result.WriteString(line[cursorCol+1:])
						}
					} else {
						// Cursor at end of line
						result.WriteString(cursorStyle.Render(" "))
					}
				} else if i == startLine && i == endLine {
					// Single line with both selection and cursor
					start := startCol
					end := endCol

					if start > len(line) {
						start = len(line)
					}
					if end > len(line) {
						end = len(line)
					}

					// Add pre-selection text
					if start > 0 {
						result.WriteString(line[:start])
					}

					// Add selection with cursor if needed
					if cursorCol >= start && cursorCol <= end {
						// Cursor inside selection
						if cursorCol > start {
							result.WriteString(highlightStyle.Render(line[start:cursorCol]))
						}
						result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
						if cursorCol+1 < end {
							result.WriteString(highlightStyle.Render(line[cursorCol+1:end]))
						}
					} else {
						// Cursor outside selection
						result.WriteString(highlightStyle.Render(line[start:end]))
					}

					// Add post-selection text
					if end < len(line) {
						if cursorCol > end {
							result.WriteString(line[end:cursorCol])
							result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
							if cursorCol+1 < len(line) {
								result.WriteString(line[cursorCol+1:])
							}
						} else {
							result.WriteString(line[end:])
						}
					}
				} else {
					// Multi-line selection with cursor
					if i == startLine {
						// First line of selection
						start := startCol
						if start > len(line) {
							start = len(line)
						}

						if start > 0 {
							result.WriteString(line[:start])
						}

						if cursorCol >= start {
							if cursorCol > start {
								result.WriteString(highlightStyle.Render(line[start:cursorCol]))
							}
							result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
							if cursorCol+1 < len(line) {
								result.WriteString(highlightStyle.Render(line[cursorCol+1:]))
							}
						} else {
							result.WriteString(highlightStyle.Render(line[start:]))
						}
					} else if i == endLine {
						// Last line of selection
						end := endCol
						if end > len(line) {
							end = len(line)
						}

						if cursorCol < end {
							if cursorCol > 0 {
								result.WriteString(highlightStyle.Render(line[:cursorCol]))
							}
							result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
							if cursorCol+1 < end {
								result.WriteString(highlightStyle.Render(line[cursorCol+1:end]))
							}
						} else {
							result.WriteString(highlightStyle.Render(line[:end]))
						}

						if end < len(line) {
							if cursorCol >= end {
								result.WriteString(line[end:cursorCol])
								result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
								if cursorCol+1 < len(line) {
									result.WriteString(line[cursorCol+1:])
								}
							} else {
								result.WriteString(line[end:])
							}
						}
					} else {
						// Middle line of selection
						if cursorCol < len(line) {
							if cursorCol > 0 {
								result.WriteString(highlightStyle.Render(line[:cursorCol]))
							}
							result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
							if cursorCol+1 < len(line) {
								result.WriteString(highlightStyle.Render(line[cursorCol+1:]))
							}
						} else {
							result.WriteString(highlightStyle.Render(line))
						}
					}
				}
			} else {
				// This line doesn't have cursor, just handle selection
				if i < startLine || i > endLine {
					// Line not in selection
					result.WriteString(line)
				} else if i == startLine && i == endLine {
					// Selection within a single line
					start := startCol
					end := endCol

					if start > len(line) {
						start = len(line)
					}
					if end > len(line) {
						end = len(line)
					}

					if start <= end {
						result.WriteString(line[:start])
						result.WriteString(highlightStyle.Render(line[start:end]))
						if end < len(line) {
							result.WriteString(line[end:])
						}
					} else {
						result.WriteString(line)
					}
				} else if i == startLine {
					// First line of multi-line selection
					start := startCol
					if start > len(line) {
						start = len(line)
					}

					result.WriteString(line[:start])
					result.WriteString(highlightStyle.Render(line[start:]))
				} else if i == endLine {
					// Last line of multi-line selection
					end := endCol
					if end > len(line) {
						end = len(line)
					}

					if end > 0 {
						result.WriteString(highlightStyle.Render(line[:end]))
					}
					if end < len(line) {
						result.WriteString(line[end:])
					}
				} else {
					// Middle line of multi-line selection
					result.WriteString(highlightStyle.Render(line))
				}
			}

			// Add newline if not the last line
			if i < len(s.Content)-1 {
				result.WriteString("\n")
			}
		}
	} else {
		// No active selection, only handle cursor if visible
		for i, line := range s.Content {
			if s.CursorVisible && i == s.CursorLine {
				// This line contains the cursor
				cursorCol := s.CursorCol
				if cursorCol < 0 {
					cursorCol = 0
				}
				if cursorCol > len(line) {
					cursorCol = len(line)
				}

				if cursorCol > 0 {
					result.WriteString(line[:cursorCol])
				}
				if cursorCol < len(line) {
					result.WriteString(cursorStyle.Render(line[cursorCol:cursorCol+1]))
					if cursorCol+1 < len(line) {
						result.WriteString(line[cursorCol+1:])
					}
				} else {
					// Cursor at end of line
					result.WriteString(cursorStyle.Render(" "))
				}
			} else {
				// No cursor on this line
				result.WriteString(line)
			}

			// Add newline if not the last line
			if i < len(s.Content)-1 {
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}

// GetSelectedText returns the currently selected text
func (s *Selection) GetSelectedText() string {
	return s.SelectedText
}

// SetContent updates the content to select from
func (s *Selection) SetContent(content []string) {
	s.Content = content
}