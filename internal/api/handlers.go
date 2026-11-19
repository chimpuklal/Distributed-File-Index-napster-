package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"distributed-file-index/internal/index"
)

// Handler handles HTTP requests for the file index API.
type Handler struct {
	idx *index.FileIndex
}

// NewHandler creates a new Handler instance.
func NewHandler(idx *index.FileIndex) *Handler {
	return &Handler{idx: idx}
}

// RegisterRequest defines the expected payload for peer registration.
type RegisterRequest struct {
	PeerID string   `json:"peer_id"`
	Files  []string `json:"files"`
}

// RegisterResponse defines the response after successful registration.
type RegisterResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	PeerID  string   `json:"peer_id"`
	Files   []string `json:"files"`
}

// PeerItem represents a peer reference in search results.
type PeerItem struct {
	PeerID string `json:"peer_id"`
}

// SearchResponse defines the response structure for file search.
type SearchResponse struct {
	File  string     `json:"file"`
	Peers []PeerItem `json:"peers"`
}

// PeersResponse defines the response structure for listing peers.
type PeersResponse struct {
	Peers []index.PeerInfo `json:"peers"`
}

// HealthResponse defines the health check response.
type HealthResponse struct {
	Status string `json:"status"`
}

// NewRouter sets up routes and returns the http.Handler.
func NewRouter(idx *index.FileIndex, webDir string) http.Handler {
	h := NewHandler(idx)
	mux := http.NewServeMux()

	// API Endpoints
	mux.HandleFunc("GET /health", h.HealthCheck)
	mux.HandleFunc("POST /api/v1/peers/register", h.RegisterPeer)
	mux.HandleFunc("GET /api/v1/search", h.SearchFile)
	mux.HandleFunc("GET /api/v1/peers", h.GetPeers)
	mux.HandleFunc("GET /api/v1/stats", h.GetStats)

	// Static Web Dashboard files
	if webDir != "" {
		fileServer := http.FileServer(http.Dir(webDir))
		mux.Handle("GET /", fileServer)
	}

	return mux
}

// HealthCheck handles GET /health.
func (h *Handler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

// RegisterPeer handles POST /api/v1/peers/register.
func (h *Handler) RegisterPeer(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	files, err := h.idx.Register(req.PeerID, req.Files)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, RegisterResponse{
		Status:  "success",
		Message: "Peer registered successfully",
		PeerID:  strings.TrimSpace(req.PeerID),
		Files:   files,
	})
}

// SearchFile handles GET /api/v1/search?file=<filename>.
func (h *Handler) SearchFile(w http.ResponseWriter, r *http.Request) {
	fileName := r.URL.Query().Get("file")
	if strings.TrimSpace(fileName) == "" {
		writeError(w, http.StatusBadRequest, "file query parameter is required")
		return
	}

	peerIDs, err := h.idx.Search(fileName)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	peers := make([]PeerItem, 0, len(peerIDs))
	for _, id := range peerIDs {
		peers = append(peers, PeerItem{PeerID: id})
	}

	writeJSON(w, http.StatusOK, SearchResponse{
		File:  strings.TrimSpace(fileName),
		Peers: peers,
	})
}

// GetPeers handles GET /api/v1/peers.
func (h *Handler) GetPeers(w http.ResponseWriter, r *http.Request) {
	peers := h.idx.ListPeers()
	if peers == nil {
		peers = make([]index.PeerInfo, 0)
	}
	writeJSON(w, http.StatusOK, PeersResponse{Peers: peers})
}

// GetStats handles GET /api/v1/stats.
func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
	stats := h.idx.Stats()
	writeJSON(w, http.StatusOK, stats)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}
