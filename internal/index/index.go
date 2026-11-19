package index

import (
	"errors"
	"sort"
	"strings"
	"sync"
)

var (
	ErrEmptyPeerID   = errors.New("peer_id cannot be empty")
	ErrNoFiles       = errors.New("at least one filename must be provided")
	ErrEmptyFilename = errors.New("filename cannot be empty")
)

// PeerInfo represents a registered peer and its list of filenames.
type PeerInfo struct {
	PeerID string   `json:"peer_id"`
	Files  []string `json:"files"`
}

// Stats represents overall indexing metrics.
type Stats struct {
	TotalPeers         int `json:"total_peers"`
	TotalUniqueFiles   int `json:"total_unique_files"`
	TotalRegistrations int `json:"total_registrations"`
}

// FileIndex manages peer registrations and reverse file indexing in memory.
type FileIndex struct {
	mu           sync.RWMutex
	peers        map[string][]string            // peerID -> list of unique filenames (trimmed)
	reverseIndex map[string]map[string]struct{} // lowerFilename -> set of peerIDs
}

// New creates and initializes a new FileIndex.
func New() *FileIndex {
	return &FileIndex{
		peers:        make(map[string][]string),
		reverseIndex: make(map[string]map[string]struct{}),
	}
}

// Register registers or updates the file list for a given peer.
// Re-registering replaces the peer's previous file list and updates the reverse index.
// Filenames are trimmed, deduplicated case-insensitively, and sorted.
func (idx *FileIndex) Register(peerID string, rawFiles []string) ([]string, error) {
	cleanID := strings.TrimSpace(peerID)
	if cleanID == "" {
		return nil, ErrEmptyPeerID
	}

	if len(rawFiles) == 0 {
		return nil, ErrNoFiles
	}

	// Clean, validate, and deduplicate files case-insensitively
	seen := make(map[string]struct{})
	var cleanFiles []string

	for _, f := range rawFiles {
		trimmed := strings.TrimSpace(f)
		if trimmed == "" {
			return nil, ErrEmptyFilename
		}
		lower := strings.ToLower(trimmed)
		if _, exists := seen[lower]; !exists {
			seen[lower] = struct{}{}
			cleanFiles = append(cleanFiles, trimmed)
		}
	}

	if len(cleanFiles) == 0 {
		return nil, ErrNoFiles
	}

	sort.Strings(cleanFiles)

	idx.mu.Lock()
	defer idx.mu.Unlock()

	// If peer already exists, clean up its previous registrations from reverseIndex
	if oldFiles, exists := idx.peers[cleanID]; exists {
		for _, oldFile := range oldFiles {
			oldLower := strings.ToLower(oldFile)
			if peerSet, ok := idx.reverseIndex[oldLower]; ok {
				delete(peerSet, cleanID)
				if len(peerSet) == 0 {
					delete(idx.reverseIndex, oldLower)
				}
			}
		}
	}

	// Update peer's file list
	idx.peers[cleanID] = cleanFiles

	// Update reverse index
	for _, file := range cleanFiles {
		lower := strings.ToLower(file)
		if _, ok := idx.reverseIndex[lower]; !ok {
			idx.reverseIndex[lower] = make(map[string]struct{})
		}
		idx.reverseIndex[lower][cleanID] = struct{}{}
	}

	// Return a copy of registered files
	res := make([]string, len(cleanFiles))
	copy(res, cleanFiles)
	return res, nil
}

// Search searches for peers containing the specified file (case-insensitive).
// Returns a sorted list of peer IDs. Returns an empty slice if no peers match.
func (idx *FileIndex) Search(filename string) ([]string, error) {
	trimmed := strings.TrimSpace(filename)
	if trimmed == "" {
		return nil, ErrEmptyFilename
	}

	lower := strings.ToLower(trimmed)

	idx.mu.RLock()
	defer idx.mu.RUnlock()

	peerSet, exists := idx.reverseIndex[lower]
	if !exists || len(peerSet) == 0 {
		return []string{}, nil
	}

	result := make([]string, 0, len(peerSet))
	for pID := range peerSet {
		result = append(result, pID)
	}

	sort.Strings(result)
	return result, nil
}

// ListPeers returns all registered peers with their files, sorted deterministically.
func (idx *FileIndex) ListPeers() []PeerInfo {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	result := make([]PeerInfo, 0, len(idx.peers))
	for peerID, files := range idx.peers {
		filesCopy := make([]string, len(files))
		copy(filesCopy, files)
		result = append(result, PeerInfo{
			PeerID: peerID,
			Files:  filesCopy,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].PeerID < result[j].PeerID
	})

	return result
}

// Stats returns the total registered peers, total unique files, and total registrations.
func (idx *FileIndex) Stats() Stats {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	totalRegistrations := 0
	for _, files := range idx.peers {
		totalRegistrations += len(files)
	}

	return Stats{
		TotalPeers:         len(idx.peers),
		TotalUniqueFiles:   len(idx.reverseIndex),
		TotalRegistrations: totalRegistrations,
	}
}
