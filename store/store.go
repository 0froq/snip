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
	c, _ := s.AddWithLimit(text, 0)
	return c
}

// AddWithLimit appends a card. When maxCards > 0, oldest cards are removed
// first (FIFO) until len < maxCards, then the new card is added. Returns
// evicted card IDs in eviction order.
func (s *Store) AddWithLimit(text string, maxCards int) (Card, []string) {
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
	c := Card{ID: id, Text: text, CreatedAt: time.Now().UTC()}
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

// Get returns a card by id.
func (s *Store) Get(id string) (Card, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.byID[id]
	return c, ok
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
