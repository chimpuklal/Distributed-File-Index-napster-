package index

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

// Test 1: Registering a peer
func TestRegisterPeer(t *testing.T) {
	idx := New()
	files := []string{"notes.pdf", "report.txt"}
	regFiles, err := idx.Register("peer-A", files)
	if err != nil {
		t.Fatalf("unexpected error registering peer: %v", err)
	}

	expected := []string{"notes.pdf", "report.txt"}
	if !reflect.DeepEqual(regFiles, expected) {
		t.Errorf("expected files %v, got %v", expected, regFiles)
	}

	peers := idx.ListPeers()
	if len(peers) != 1 {
		t.Fatalf("expected 1 peer, got %d", len(peers))
	}
	if peers[0].PeerID != "peer-A" {
		t.Errorf("expected peer ID 'peer-A', got '%s'", peers[0].PeerID)
	}
}

// Test 2: Searching for an existing file
func TestSearchExistingFile(t *testing.T) {
	idx := New()
	_, err := idx.Register("peer-A", []string{"notes.pdf", "photo.jpg"})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	res, err := idx.Search("notes.pdf")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	expected := []string{"peer-A"}
	if !reflect.DeepEqual(res, expected) {
		t.Errorf("expected %v, got %v", expected, res)
	}
}

// Test 3: Searching for a nonexistent file
func TestSearchNonexistentFile(t *testing.T) {
	idx := New()
	_, err := idx.Register("peer-A", []string{"notes.pdf"})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	res, err := idx.Search("unknown_file.iso")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	if res == nil || len(res) != 0 {
		t.Errorf("expected empty non-nil slice, got %v", res)
	}
}

// Test 4: Multiple peers registering the same filename
func TestMultiplePeersSameFile(t *testing.T) {
	idx := New()
	_, err := idx.Register("peer-B", []string{"shared.pdf", "b_only.txt"})
	if err != nil {
		t.Fatalf("peer-B failed: %v", err)
	}
	_, err = idx.Register("peer-A", []string{"shared.pdf", "a_only.txt"})
	if err != nil {
		t.Fatalf("peer-A failed: %v", err)
	}

	res, err := idx.Search("shared.pdf")
	if err != nil {
		t.Fatalf("search failed: %v", err)
	}

	expected := []string{"peer-A", "peer-B"}
	if !reflect.DeepEqual(res, expected) {
		t.Errorf("expected deterministic sorted peers %v, got %v", expected, res)
	}
}

// Test 5: Updating an existing peer's registration
func TestUpdatePeerRegistration(t *testing.T) {
	idx := New()
	_, err := idx.Register("peer-A", []string{"old_doc.txt", "staying_doc.txt"})
	if err != nil {
		t.Fatalf("initial registration failed: %v", err)
	}

	// Verify initial registration
	res, _ := idx.Search("old_doc.txt")
	if len(res) != 1 || res[0] != "peer-A" {
		t.Fatalf("expected peer-A for old_doc.txt")
	}

	// Re-register with new list
	_, err = idx.Register("peer-A", []string{"staying_doc.txt", "new_doc.txt"})
	if err != nil {
		t.Fatalf("update registration failed: %v", err)
	}

	// Old file should now have no peers
	resOld, err := idx.Search("old_doc.txt")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(resOld) != 0 {
		t.Errorf("expected 0 peers for old_doc.txt, got %v", resOld)
	}

	// Staying file should still have peer-A
	resStaying, err := idx.Search("staying_doc.txt")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(resStaying) != 1 || resStaying[0] != "peer-A" {
		t.Errorf("expected [peer-A] for staying_doc.txt, got %v", resStaying)
	}

	// New file should have peer-A
	resNew, err := idx.Search("new_doc.txt")
	if err != nil {
		t.Fatalf("search error: %v", err)
	}
	if len(resNew) != 1 || resNew[0] != "peer-A" {
		t.Errorf("expected [peer-A] for new_doc.txt, got %v", resNew)
	}

	// Verify stats
	stats := idx.Stats()
	if stats.TotalPeers != 1 {
		t.Errorf("expected 1 peer, got %d", stats.TotalPeers)
	}
	if stats.TotalUniqueFiles != 2 {
		t.Errorf("expected 2 unique files, got %d", stats.TotalUniqueFiles)
	}
	if stats.TotalRegistrations != 2 {
		t.Errorf("expected 2 total registrations, got %d", stats.TotalRegistrations)
	}
}

// Test 6: Duplicate filenames
func TestDuplicateFilenames(t *testing.T) {
	idx := New()
	rawFiles := []string{
		"notes.pdf",
		"NOTES.pdf",
		"notes.pdf",
		"  photo.jpg  ",
		"PHOTO.JPG",
	}

	cleanFiles, err := idx.Register("peer-A", rawFiles)
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	expected := []string{"notes.pdf", "photo.jpg"}
	if !reflect.DeepEqual(cleanFiles, expected) {
		t.Errorf("expected deduplicated files %v, got %v", expected, cleanFiles)
	}

	// Reverse index should also only have 1 entry per file for peer-A
	res, err := idx.Search("notes.pdf")
	if err != nil || len(res) != 1 {
		t.Errorf("expected 1 peer for notes.pdf, got %v", res)
	}
}

// Test 7: Case-insensitive search
func TestCaseInsensitiveSearch(t *testing.T) {
	idx := New()
	_, err := idx.Register("peer-A", []string{"MyDocument.PDF"})
	if err != nil {
		t.Fatalf("registration failed: %v", err)
	}

	queries := []string{
		"mydocument.pdf",
		"MYDOCUMENT.PDF",
		"MyDocument.PDF",
		"mYdOcUmEnT.pDf",
		"  mydocument.pdf  ",
	}

	for _, q := range queries {
		res, err := idx.Search(q)
		if err != nil {
			t.Errorf("search for '%s' returned error: %v", q, err)
		}
		if len(res) != 1 || res[0] != "peer-A" {
			t.Errorf("search for '%s' expected ['peer-A'], got %v", q, res)
		}
	}
}

// Test 8: Concurrent registration and search
func TestConcurrentRegistrationAndSearch(t *testing.T) {
	idx := New()
	var wg sync.WaitGroup
	numGoroutines := 40

	// Concurrent registers
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			peerID := fmt.Sprintf("peer-%02d", id)
			files := []string{
				"common.txt",
				fmt.Sprintf("unique_%d.dat", id),
			}
			_, err := idx.Register(peerID, files)
			if err != nil {
				t.Errorf("concurrent register failed: %v", err)
			}
		}(i)
	}

	// Concurrent searches and reads
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			_, _ = idx.Search("common.txt")
			_, _ = idx.Search(fmt.Sprintf("unique_%d.dat", id))
			_ = idx.ListPeers()
			_ = idx.Stats()
		}(i)
	}

	wg.Wait()

	// Verify consistency after all concurrent operations
	stats := idx.Stats()
	if stats.TotalPeers != numGoroutines {
		t.Errorf("expected %d peers, got %d", numGoroutines, stats.TotalPeers)
	}
	commonPeers, err := idx.Search("common.txt")
	if err != nil {
		t.Fatalf("failed to search common.txt: %v", err)
	}
	if len(commonPeers) != numGoroutines {
		t.Errorf("expected %d peers for common.txt, got %d", numGoroutines, len(commonPeers))
	}
}

// Test 9: Retrieving the registered peer list
func TestListPeers(t *testing.T) {
	idx := New()
	_, _ = idx.Register("peer-Z", []string{"z.txt", "a.txt"})
	_, _ = idx.Register("peer-A", []string{"b.txt", "c.txt"})
	_, _ = idx.Register("peer-M", []string{"m.txt"})

	peers := idx.ListPeers()
	if len(peers) != 3 {
		t.Fatalf("expected 3 peers, got %d", len(peers))
	}

	expectedOrder := []string{"peer-A", "peer-M", "peer-Z"}
	for i, expectedID := range expectedOrder {
		if peers[i].PeerID != expectedID {
			t.Errorf("index %d: expected peer %s, got %s", i, expectedID, peers[i].PeerID)
		}
	}

	// Verify files inside peer-Z are sorted
	expectedZFiles := []string{"a.txt", "z.txt"}
	if !reflect.DeepEqual(peers[2].Files, expectedZFiles) {
		t.Errorf("expected peer-Z files %v, got %v", expectedZFiles, peers[2].Files)
	}
}

// Additional validations
func TestValidations(t *testing.T) {
	idx := New()

	// Empty peer ID
	if _, err := idx.Register("", []string{"a.txt"}); err != ErrEmptyPeerID {
		t.Errorf("expected ErrEmptyPeerID, got %v", err)
	}
	if _, err := idx.Register("   ", []string{"a.txt"}); err != ErrEmptyPeerID {
		t.Errorf("expected ErrEmptyPeerID, got %v", err)
	}

	// Empty file list
	if _, err := idx.Register("peer-A", []string{}); err != ErrNoFiles {
		t.Errorf("expected ErrNoFiles, got %v", err)
	}

	// Empty filename
	if _, err := idx.Register("peer-A", []string{""}); err != ErrEmptyFilename {
		t.Errorf("expected ErrEmptyFilename, got %v", err)
	}
	if _, err := idx.Register("peer-A", []string{"   "}); err != ErrEmptyFilename {
		t.Errorf("expected ErrEmptyFilename, got %v", err)
	}

	// Empty search filename
	if _, err := idx.Search(""); err != ErrEmptyFilename {
		t.Errorf("expected ErrEmptyFilename, got %v", err)
	}
	if _, err := idx.Search("   "); err != ErrEmptyFilename {
		t.Errorf("expected ErrEmptyFilename, got %v", err)
	}
}
