//go:build !darwin

package macplace

// NativeCardText is false off macOS. The card draws its text in the UI
// toolkit, which scrolls, because there is no NSScrollView here.
const NativeCardText = false

// CardText is the macOS text host. Off macOS the methods do nothing.
type CardText struct{}

// AttachCardText is a no-op off macOS.
func AttachCardText(uintptr, CardTextOptions) *CardText { return &CardText{} }

// Layout is a no-op off macOS.
func (t *CardText) Layout(int, int, int, int, int, int, int) {}

// SetText is a no-op off macOS.
func (t *CardText) SetText(string) {}

// SetFont is a no-op off macOS.
func (t *CardText) SetFont(string, float64) {}

// SetColor is a no-op off macOS.
func (t *CardText) SetColor(uint8, uint8, uint8) {}

// SetBackground is a no-op off macOS.
func (t *CardText) SetBackground(uint8, uint8, uint8) {}

// SetCanEdit is a no-op off macOS.
func (t *CardText) SetCanEdit(bool) {}

// SetTitles is a no-op off macOS.
func (t *CardText) SetTitles(CardMenuTitles) {}

// BeginEdit is a no-op off macOS.
func (t *CardText) BeginEdit(string) {}

// CancelEdit is a no-op off macOS.
func (t *CardText) CancelEdit() {}

// Editing is always false off macOS.
func (t *CardText) Editing() bool { return false }

// Text is empty off macOS.
func (t *CardText) Text() string { return "" }

// ShowMenu is a no-op off macOS.
func (t *CardText) ShowMenu(int, int, int, int) {}

// Close is a no-op off macOS.
func (t *CardText) Close() {}
