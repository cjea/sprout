package mvp

import (
	"sync"
)

type PassageRepairStore interface {
	SavePassageRepairSnapshot(PassageRepairSnapshot) (PassageRepairSnapshot, error)
	LoadPassageRepairSnapshot(sessionID string, revision int) (PassageRepairSnapshot, error)
	LoadLatestPassageRepairSnapshot(sessionID string) (PassageRepairSnapshot, error)
}

type MemoryPassageRepairStore struct {
	mu        sync.RWMutex
	snapshots map[string]map[int]PassageRepairSnapshot
}

func NewMemoryPassageRepairStore() *MemoryPassageRepairStore {
	return &MemoryPassageRepairStore{
		snapshots: map[string]map[int]PassageRepairSnapshot{},
	}
}

func (s *MemoryPassageRepairStore) SavePassageRepairSnapshot(snapshot PassageRepairSnapshot) (PassageRepairSnapshot, error) {
	if err := snapshot.Validate(); err != nil {
		return PassageRepairSnapshot{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[snapshot.SessionID]; !ok {
		s.snapshots[snapshot.SessionID] = map[int]PassageRepairSnapshot{}
	}
	s.snapshots[snapshot.SessionID][snapshot.Revision] = snapshot
	return snapshot, nil
}

func (s *MemoryPassageRepairStore) LoadPassageRepairSnapshot(sessionID string, revision int) (PassageRepairSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.snapshots[sessionID]
	if !ok {
		return PassageRepairSnapshot{}, ErrNotFound
	}
	snapshot, ok := versions[revision]
	if !ok {
		return PassageRepairSnapshot{}, ErrNotFound
	}
	return snapshot, nil
}

func (s *MemoryPassageRepairStore) LoadLatestPassageRepairSnapshot(sessionID string) (PassageRepairSnapshot, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	versions, ok := s.snapshots[sessionID]
	if !ok || len(versions) == 0 {
		return PassageRepairSnapshot{}, ErrNotFound
	}
	latestRevision := -1
	for revision := range versions {
		if revision > latestRevision {
			latestRevision = revision
		}
	}
	return versions[latestRevision], nil
}
