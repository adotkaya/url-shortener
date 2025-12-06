package main

import "sync"

type URLStore struct {
	urls map[string]string
	mu   sync.Mutex
}

func NewURLStore() *URLStore {
	return &URLStore{
		urls: make(map[string]string),
	}
}

func (s *URLStore) Get(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	val, ok := s.urls[id]
	return val, ok
}

func (s *URLStore) Put(id, originalURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.urls[id] = originalURL
}
