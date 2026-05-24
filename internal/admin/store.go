package admin

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

type Store struct {
	path  string
	mutex sync.RWMutex
	state State
}

func OpenStore(path string) (*Store, error) {
	store := &Store{
		path:  path,
		state: NewState(0),
	}

	content, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return nil, err
		}
		if err := store.saveLocked(); err != nil {
			return nil, err
		}
		return store, nil
	}
	if err != nil {
		return nil, err
	}
	if len(content) > 0 {
		if err := json.Unmarshal(content, &store.state); err != nil {
			return nil, err
		}
	}
	store.normalizeLocked()
	return store, nil
}

func (s *Store) View(fn func(State) error) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return fn(s.state)
}

func (s *Store) Update(fn func(*State) error) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.normalizeLocked()
	if err := fn(&s.state); err != nil {
		return err
	}
	s.normalizeLocked()
	return s.saveLocked()
}

func (s *Store) saveLocked() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, content, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) normalizeLocked() {
	if s.state.Users == nil {
		s.state.Users = map[string]User{}
	}
	if s.state.UsersByName == nil {
		s.state.UsersByName = map[string]string{}
	}
	if s.state.CoinLedger == nil {
		s.state.CoinLedger = []CoinTransaction{}
	}
	if s.state.EmbyLinks == nil {
		s.state.EmbyLinks = map[string]EmbyUserLink{}
	}
	if s.state.SecurityEvents == nil {
		s.state.SecurityEvents = []SecurityEvent{}
	}
	if s.state.RegistrationPolicy.Windows == nil {
		s.state.RegistrationPolicy.Windows = []RegistrationWindow{}
	}
	if s.state.SystemConfig.Features == nil {
		s.state.SystemConfig.Features = DefaultFeaturePolicies()
	}
	for id, user := range s.state.Users {
		s.state.UsersByName[user.Username] = id
	}
}
