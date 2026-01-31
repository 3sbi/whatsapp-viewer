package main

import (
	"log"
	"sort"
	"sync"
	"time"
)

type SessionStore struct {
	mu       sync.RWMutex
	sessions map[string]*ChatSession
}

const MaxMemoryMB = 500 // Maximum memory usage in MB
const MaxMemoryBytes = int64(MaxMemoryMB * 1024 * 1024)

func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*ChatSession),
	}
}

func (s *SessionStore) Get(id string) (*ChatSession, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, exists := s.sessions[id]
	return session, exists
}

func (s *SessionStore) Set(id string, session *ChatSession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[id] = session
}

func (s *SessionStore) getTotalSize() int64 {
	var total int64
	for _, session := range s.sessions {
		total += session.SizeBytes
	}
	return total
}

func (s *SessionStore) Cleanup() {
	totalMemory := s.getTotalSize()

	if totalMemory > MaxMemoryBytes {
		// Get sessions sorted by creation time (oldest first)
		type sessionInfo struct {
			id        string
			createdAt time.Time
			sizeBytes int64
		}

		var sessionList []sessionInfo
		for id, session := range s.sessions {
			sessionList = append(sessionList, sessionInfo{
				id:        id,
				createdAt: session.CreatedAt,
				sizeBytes: session.SizeBytes,
			})
		}

		// Sort by creation time (oldest first)
		sort.Slice(sessionList, func(i, j int) bool {
			return sessionList[i].createdAt.Before(sessionList[j].createdAt)
		})

		// Remove oldest sessions until total memory usage is acceptable
		var removedCount int
		var removedSize int64
		for _, si := range sessionList {
			delete(s.sessions, si.id)
			removedCount++
			removedSize += si.sizeBytes
			totalMemory -= si.sizeBytes
			if totalMemory <= MaxMemoryBytes {
				break
			}
		}

		log.Printf("Memory cleanup triggered. Removed %d oldest sessions (%.2fMB). Current session memory: %.2fMB",
			removedCount, float64(removedSize)/1024/1024, float64(totalMemory)/1024/1024)
	}
}

func (s *SessionStore) RemoveOldSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.sessions {
		if time.Since(session.CreatedAt) > time.Hour {
			delete(s.sessions, id)
		}
	}
}
