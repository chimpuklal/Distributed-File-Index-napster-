package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"distributed-file-index/internal/index"
)

func setupTestServer() (http.Handler, *index.FileIndex) {
	idx := index.New()
	router := NewRouter(idx, "")
	return router, idx
}

func TestHealthCheck(t *testing.T) {
	router, _ := setupTestServer()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.Status != "ok" {
		t.Errorf("expected status 'ok', got '%s'", res.Status)
	}
}

func TestRegisterPeerAPI(t *testing.T) {
	router, _ := setupTestServer()

	// 1. Success case
	payload := `{"peer_id": "peer-A", "files": ["notes.pdf", "photo.jpg"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/peers/register", bytes.NewBufferString(payload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res RegisterResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.PeerID != "peer-A" || len(res.Files) != 2 {
		t.Errorf("unexpected response content: %+v", res)
	}

	// 2. Empty Peer ID
	payloadErr := `{"peer_id": "", "files": ["notes.pdf"]}`
	reqErr := httptest.NewRequest(http.MethodPost, "/api/v1/peers/register", bytes.NewBufferString(payloadErr))
	recErr := httptest.NewRecorder()
	router.ServeHTTP(recErr, reqErr)
	if recErr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for empty peer ID, got %d", recErr.Code)
	}

	// 3. Malformed JSON
	reqBadJSON := httptest.NewRequest(http.MethodPost, "/api/v1/peers/register", bytes.NewBufferString("{invalid-json}"))
	recBadJSON := httptest.NewRecorder()
	router.ServeHTTP(recBadJSON, reqBadJSON)
	if recBadJSON.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for bad JSON, got %d", recBadJSON.Code)
	}
}

func TestSearchFileAPI(t *testing.T) {
	router, idx := setupTestServer()

	// Pre-populate index
	_, _ = idx.Register("peer-A", []string{"photo.jpg", "notes.pdf"})
	_, _ = idx.Register("peer-B", []string{"photo.jpg", "music.mp3"})

	// 1. Existing file search
	req := httptest.NewRequest(http.MethodGet, "/api/v1/search?file=photo.jpg", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if res.File != "photo.jpg" {
		t.Errorf("expected file 'photo.jpg', got '%s'", res.File)
	}
	if len(res.Peers) != 2 {
		t.Fatalf("expected 2 peers, got %d", len(res.Peers))
	}
	if res.Peers[0].PeerID != "peer-A" || res.Peers[1].PeerID != "peer-B" {
		t.Errorf("unexpected peer list: %+v", res.Peers)
	}

	// 2. Case-insensitive search
	reqUpper := httptest.NewRequest(http.MethodGet, "/api/v1/search?file=PHOTO.JPG", nil)
	recUpper := httptest.NewRecorder()
	router.ServeHTTP(recUpper, reqUpper)
	if recUpper.Code != http.StatusOK {
		t.Fatalf("expected status 200 for uppercase search, got %d", recUpper.Code)
	}
	var resUpper SearchResponse
	_ = json.Unmarshal(recUpper.Body.Bytes(), &resUpper)
	if len(resUpper.Peers) != 2 {
		t.Errorf("expected 2 peers for case-insensitive search, got %d", len(resUpper.Peers))
	}

	// 3. Nonexistent file search
	reqNone := httptest.NewRequest(http.MethodGet, "/api/v1/search?file=nonexistent.doc", nil)
	recNone := httptest.NewRecorder()
	router.ServeHTTP(recNone, reqNone)
	if recNone.Code != http.StatusOK {
		t.Fatalf("expected 200 for nonexistent file, got %d", recNone.Code)
	}
	var resNone SearchResponse
	_ = json.Unmarshal(recNone.Body.Bytes(), &resNone)
	if resNone.Peers == nil || len(resNone.Peers) != 0 {
		t.Errorf("expected empty non-nil peers slice, got %v", resNone.Peers)
	}

	// 4. Missing file parameter
	reqMissing := httptest.NewRequest(http.MethodGet, "/api/v1/search", nil)
	recMissing := httptest.NewRecorder()
	router.ServeHTTP(recMissing, reqMissing)
	if recMissing.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing file query, got %d", recMissing.Code)
	}
}

func TestGetPeersAPI(t *testing.T) {
	router, idx := setupTestServer()

	// Initially empty
	req := httptest.NewRequest(http.MethodGet, "/api/v1/peers", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var resEmpty PeersResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resEmpty); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resEmpty.Peers == nil || len(resEmpty.Peers) != 0 {
		t.Errorf("expected empty slice, got %v", resEmpty.Peers)
	}

	// Populate
	_, _ = idx.Register("peer-1", []string{"file1.txt"})
	recPop := httptest.NewRecorder()
	router.ServeHTTP(recPop, req)
	var resPop PeersResponse
	_ = json.Unmarshal(recPop.Body.Bytes(), &resPop)
	if len(resPop.Peers) != 1 || resPop.Peers[0].PeerID != "peer-1" {
		t.Errorf("unexpected peers response: %+v", resPop)
	}
}

func TestGetStatsAPI(t *testing.T) {
	router, idx := setupTestServer()

	_, _ = idx.Register("peer-1", []string{"fileA.txt", "fileB.txt"})
	_, _ = idx.Register("peer-2", []string{"fileB.txt", "fileC.txt"})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var stats index.Stats
	if err := json.Unmarshal(rec.Body.Bytes(), &stats); err != nil {
		t.Fatalf("failed to decode stats: %v", err)
	}
	if stats.TotalPeers != 2 {
		t.Errorf("expected 2 peers, got %d", stats.TotalPeers)
	}
	if stats.TotalUniqueFiles != 3 {
		t.Errorf("expected 3 unique files, got %d", stats.TotalUniqueFiles)
	}
	if stats.TotalRegistrations != 4 {
		t.Errorf("expected 4 total registrations, got %d", stats.TotalRegistrations)
	}
}
