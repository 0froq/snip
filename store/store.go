package store

import (
	"sync"
	"time"
)

// Card is one captured clipboard snippet.
type Card struct {
	ID        string
	Text      string
	CreatedAt time.Time
	// Concealed is true when the copy carried org.nspasteboard.ConcealedType.
	Concealed bool
	// Original is the clipboard text from before the first edit. Empty
	// means Text is still that text.
	Original string
}

// Store holds cards in insertion order (oldest first).
type Store struct {
	mu    sync.RWMutex
	order []string
	byID  map[string]Card
	seq   int
}

// New creates an empty store.
func New() *Store {
	return &Store{byID: make(map[string]Card)}
}

// Add appends a card and returns it.
func (s *Store) Add(text string) Card {
	c, _ := s.AddWithLimit(text, 0, false)
	return c
}

// AddWithLimit appends a card. When maxCards > 0, oldest cards are removed
// first (FIFO) until len < maxCards, then the new card is added. Returns
// evicted card IDs in eviction order.
func (s *Store) AddWithLimit(text string, maxCards int, concealed bool) (Card, []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var evicted []string
	if maxCards > 0 {
		for len(s.order) >= maxCards {
			id := s.order[0]
			s.deleteID(id)
			evicted = append(evicted, id)
		}
	}
	s.seq++
	id := formatID(s.seq)
	c := Card{ID: id, Text: text, CreatedAt: time.Now().UTC(), Concealed: concealed}
	s.byID[id] = c
	s.order = append(s.order, id)
	return c, evicted
}

// All returns cards from oldest to newest.
func (s *Store) All() []Card {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Card, 0, len(s.order))
	for _, id := range s.order {
		out = append(out, s.byID[id])
	}
	return out
}

// FindText returns the newest card whose text matches exactly.
func (s *Store) FindText(text string) (Card, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := len(s.order) - 1; i >= 0; i-- {
		c := s.byID[s.order[i]]
		if c.Text == text {
			return c, true
		}
	}
	return Card{}, false
}

// Get returns a card by id.
func (s *Store) Get(id string) (Card, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[id]
	return c, ok
}

// UpdateText replaces a card's text. The first edit remembers the previous
// text as Original so RestoreOriginal can put it back.
func (s *Store) UpdateText(id, text string) (Card, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[id]
	if !ok {
		return Card{}, false
	}
	if c.Original == "" && c.Text != text {
		c.Original = c.Text
	}
	c.Text = text
	s.byID[id] = c
	return c, true
}

// RestoreOriginal puts the pre-edit clipboard text back. A card that was
// never edited is unchanged.
func (s *Store) RestoreOriginal(id string) (Card, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.byID[id]
	if !ok {
		return Card{}, false
	}
	if c.Original != "" {
		c.Text = c.Original
		c.Original = ""
		s.byID[id] = c
	}
	return c, true
}

// Delete removes a card.
func (s *Store) Delete(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteID(id)
}

func (s *Store) deleteID(id string) bool {
	if _, ok := s.byID[id]; !ok {
		return false
	}
	delete(s.byID, id)
	for i, oid := range s.order {
		if oid == id {
			s.order = append(s.order[:i], s.order[i+1:]...)
			break
		}
	}
	return true
}

// TrimTo drops the oldest cards until at most max remain.
// max <= 0 leaves the store unchanged. Evicted IDs are oldest first.
func (s *Store) TrimTo(max int) []string {
	if max <= 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var evicted []string
	for len(s.order) > max {
		id := s.order[0]
		s.deleteID(id)
		evicted = append(evicted, id)
	}
	return evicted
}

// Len returns the number of cards.
func (s *Store) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.order)
}

func formatID(n int) string {
	return "c" + itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
