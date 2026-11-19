package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type RegisterRequest struct {
	PeerID string   `json:"peer_id"`
	Files  []string `json:"files"`
}

type RegisterResponse struct {
	Status  string   `json:"status"`
	Message string   `json:"message"`
	PeerID  string   `json:"peer_id"`
	Files   []string `json:"files"`
}

type PeerItem struct {
	PeerID string `json:"peer_id"`
}

type SearchResponse struct {
	File  string     `json:"file"`
	Peers []PeerItem `json:"peers"`
}

type PeerInfo struct {
	PeerID string   `json:"peer_id"`
	Files  []string `json:"files"`
}

type PeersResponse struct {
	Peers []PeerInfo `json:"peers"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  go run ./cmd/peer <subcommand> [flags]")
	fmt.Println()
	fmt.Println("Subcommands:")
	fmt.Println("  register   Register peer ID and available files with the central server")
	fmt.Println("  search     Search the central server for peers possessing a file")
	fmt.Println("  list       List all registered peers and their files")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  go run ./cmd/peer register --server http://localhost:8080 --id peer-A --files \"notes.pdf,photo.jpg,report.txt\"")
	fmt.Println("  go run ./cmd/peer search --server http://localhost:8080 --file photo.jpg")
	fmt.Println("  go run ./cmd/peer list --server http://localhost:8080")
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	subcommand := os.Args[1]
	client := &http.Client{Timeout: 5 * time.Second}

	switch subcommand {
	case "register":
		runRegister(client, os.Args[2:])
	case "search":
		runSearch(client, os.Args[2:])
	case "list", "peers":
		runList(client, os.Args[2:])
	case "--help", "-h", "help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown subcommand: %s\n\n", subcommand)
		printUsage()
		os.Exit(1)
	}
}

func runRegister(client *http.Client, args []string) {
	fs := flag.NewFlagSet("register", flag.ExitOnError)
	serverURL := fs.String("server", "http://localhost:8080", "Central server URL")
	peerID := fs.String("id", "", "Peer identifier (required)")
	filesStr := fs.String("files", "", "Comma-separated list of filenames (required)")

	_ = fs.Parse(args)

	cleanID := strings.TrimSpace(*peerID)
	if cleanID == "" {
		fmt.Fprintln(os.Stderr, "Error: --id is required and cannot be empty")
		fs.Usage()
		os.Exit(1)
	}

	cleanFilesStr := strings.TrimSpace(*filesStr)
	if cleanFilesStr == "" {
		fmt.Fprintln(os.Stderr, "Error: --files is required and cannot be empty")
		fs.Usage()
		os.Exit(1)
	}

	parts := strings.Split(cleanFilesStr, ",")
	var files []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			files = append(files, trimmed)
		}
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "Error: at least one valid filename must be provided")
		os.Exit(1)
	}

	reqBody := RegisterRequest{
		PeerID: cleanID,
		Files:  files,
	}

	data, err := json.Marshal(reqBody)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling request: %v\n", err)
		os.Exit(1)
	}

	targetURL := strings.TrimRight(*serverURL, "/") + "/api/v1/peers/register"
	resp, err := client.Post(targetURL, "application/json", bytes.NewReader(data))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server at %s: %v\n", targetURL, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Error != "" {
			fmt.Fprintf(os.Stderr, "Registration failed (HTTP %d): %s\n", resp.StatusCode, errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "Registration failed (HTTP %d): %s\n", resp.StatusCode, string(bodyBytes))
		}
		os.Exit(1)
	}

	var regResp RegisterResponse
	if err := json.Unmarshal(bodyBytes, &regResp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("[SUCCESS] %s\n", regResp.Message)
	fmt.Printf("Peer ID: %s\n", regResp.PeerID)
	fmt.Printf("Registered Files (%d):\n", len(regResp.Files))
	for _, f := range regResp.Files {
		fmt.Printf("  • %s\n", f)
	}
}

func runSearch(client *http.Client, args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	serverURL := fs.String("server", "http://localhost:8080", "Central server URL")
	fileName := fs.String("file", "", "Filename to search for (required)")

	_ = fs.Parse(args)

	cleanFile := strings.TrimSpace(*fileName)
	if cleanFile == "" {
		fmt.Fprintln(os.Stderr, "Error: --file is required and cannot be empty")
		fs.Usage()
		os.Exit(1)
	}

	targetURL := fmt.Sprintf("%s/api/v1/search?file=%s",
		strings.TrimRight(*serverURL, "/"),
		url.QueryEscape(cleanFile),
	)

	resp, err := client.Get(targetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server at %s: %v\n", targetURL, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		var errResp ErrorResponse
		if err := json.Unmarshal(bodyBytes, &errResp); err == nil && errResp.Error != "" {
			fmt.Fprintf(os.Stderr, "Search failed (HTTP %d): %s\n", resp.StatusCode, errResp.Error)
		} else {
			fmt.Fprintf(os.Stderr, "Search failed (HTTP %d): %s\n", resp.StatusCode, string(bodyBytes))
		}
		os.Exit(1)
	}

	var searchResp SearchResponse
	if err := json.Unmarshal(bodyBytes, &searchResp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode response: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Search Results for: '%s'\n", searchResp.File)
	if len(searchResp.Peers) == 0 {
		fmt.Println("Result: No peers found possessing this file.")
		return
	}

	fmt.Printf("Matching Peers (%d):\n", len(searchResp.Peers))
	for _, p := range searchResp.Peers {
		fmt.Printf("  • %s\n", p.PeerID)
	}
}

func runList(client *http.Client, args []string) {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	serverURL := fs.String("server", "http://localhost:8080", "Central server URL")

	_ = fs.Parse(args)

	targetURL := strings.TrimRight(*serverURL, "/") + "/api/v1/peers"

	resp, err := client.Get(targetURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to server at %s: %v\n", targetURL, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Failed to fetch peers (HTTP %d): %s\n", resp.StatusCode, string(bodyBytes))
		os.Exit(1)
	}

	var peersResp PeersResponse
	if err := json.Unmarshal(bodyBytes, &peersResp); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode response: %v\n", err)
		os.Exit(1)
	}

	if len(peersResp.Peers) == 0 {
		fmt.Println("No peers currently registered with the central index.")
		return
	}

	fmt.Printf("Registered Peers (%d):\n", len(peersResp.Peers))
	for _, p := range peersResp.Peers {
		fmt.Printf("Peer: %s (%d files)\n", p.PeerID, len(p.Files))
		for _, f := range p.Files {
			fmt.Printf("  - %s\n", f)
		}
	}
}
