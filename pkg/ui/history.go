package ui

// InputHistory maintains a history of input text with navigation
type InputHistory struct {
	items       []string
	currentPos  int
	currentText string
	saved       bool
}

// NewInputHistory creates a new input history
func NewInputHistory() *InputHistory {
	return &InputHistory{
		items:      []string{},
		currentPos: -1,
		saved:      true,
	}
}

// Add adds an item to the history
func (h *InputHistory) Add(text string) {
	if text == "" {
		return
	}
	
	// Avoid duplicate consecutive entries
	if len(h.items) > 0 && h.items[len(h.items)-1] == text {
		return
	}
	
	h.items = append(h.items, text)
	h.currentPos = len(h.items)
	h.saved = true
}

// Previous navigates to the previous item in history
func (h *InputHistory) Previous(currentText string) string {
	// Save current text if needed
	if h.currentPos == len(h.items) && !h.saved {
		h.currentText = currentText
		h.saved = true
	}
	
	// Handle empty history
	if len(h.items) == 0 {
		return currentText
	}
	
	// Already at the beginning
	if h.currentPos <= 0 {
		return h.items[0]
	}
	
	h.currentPos--
	return h.items[h.currentPos]
}

// Next navigates to the next item in history
func (h *InputHistory) Next() string {
	// Handle empty history
	if len(h.items) == 0 {
		return ""
	}
	
	// At or beyond the end of history
	if h.currentPos >= len(h.items) {
		return h.currentText
	}
	
	h.currentPos++
	
	// If at the end of history, return saved text
	if h.currentPos == len(h.items) {
		return h.currentText
	}
	
	return h.items[h.currentPos]
}

// Reset resets the position to the end of history
func (h *InputHistory) Reset() {
	h.currentPos = len(h.items)
	h.saved = true
}

// Save saves the current text
func (h *InputHistory) Save(text string) {
	h.currentText = text
	h.saved = true
}

// IsSaved returns whether the current text is saved
func (h *InputHistory) IsSaved() bool {
	return h.saved
}

// SetSaved sets the saved flag
func (h *InputHistory) SetSaved(saved bool) {
	h.saved = saved
}

// GetHistory returns the history items
func (h *InputHistory) GetHistory() []string {
	return h.items
}