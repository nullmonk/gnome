package main

import (
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"time"
)

// FileMetadata is the structure used for the JSON response on HEAD requests.
// Fields must be exported (capitalized) to be marshaled to JSON.
type FileMetadata struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
	IsDir   bool      `json:"is_directory"`
	Mode    string    `json:"mode"`
	Path    string    `json:"path"`
}

// FSFileServer implements the http.Handler interface and uses an fs.FS as its root.
type FSFileServer struct {
	Root fs.FS
}

// ServeHTTP handles incoming HTTP requests and dispatches them based on method.
func (s *FSFileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1. Sanitize and validate the request path
	name := path.Clean(r.URL.Path)
	// fs.FS paths should be relative, so strip the leading slash
	if name == "/" {
		name = "."
	} else if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	// 2. Retrieve file metadata (required for both HEAD and GET)
	info, err := fs.Stat(s.Root, name)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, "404 Not Found: "+name, http.StatusNotFound)
			return
		}
		log.Printf("FS Stat Error for %s: %v", name, err)
		http.Error(w, "500 Internal Server Error", http.StatusInternalServerError)
		return
	}

	// 3. Dispatch based on HTTP Method
	switch r.Method {
	case "HEAD":
		fallthrough
	case "GET":
		s.handleHead(w, r, name, info)
	default:
		// Unsupported method
		w.Header().Set("Allow", "HEAD, GET")
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleHead constructs and sends the JSON file metadata response.
func (s *FSFileServer) handleHead(w http.ResponseWriter, r *http.Request, name string, info fs.FileInfo) {
	w.Header().Set("X-Path", info.Name())
	if info.IsDir() {
		w.Header().Set("X-Directory", "true")
	}
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))

	// Custom Header: X-DirEnt (only for directories)
	if info.IsDir() {
		// Attempt to read directory contents
		entries, err := fs.ReadDir(s.Root, name)
		if err != nil {
			http.Error(w, "404 "+err.Error(), http.StatusNotFound)
		} else {
			// Add X-DirEnt header for each entry (Format: name:type)
			for _, entry := range entries {
				name := entry.Name()
				if entry.IsDir() {
					name += "/"
				}
				w.Header().Add("X-Directory-Entry", name)
			}
		}
	}

	// For HEAD, we stop here after writing headers/metadata.
	if r.Method == "HEAD" {
		return
	}

	if info.IsDir() {
		// As requested, a directory GET returns a 403 Forbidden.
		// Standard file servers usually list contents here, but we're keeping it simple.
		http.Error(w, "403 Forbidden: Cannot GET a directory.", http.StatusForbidden)
		return
	}

	// 1. Open the file
	file, err := s.Root.Open(name)
	if err != nil {
		log.Printf("FS Open Error for %s: %v", name, err)
		http.Error(w, "500 Could not open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	w.Header().Set("Content-Type", "application/octet-stream") // Generic binary type
	w.Header().Set("Last-Modified", info.ModTime().UTC().Format(http.TimeFormat))

	// 3. Stream the file content
	if _, err := io.Copy(w, file); err != nil {
		// Log the error, but the headers have already been written, so we can't send a 500 status code.
		log.Printf("Failed to stream file %s: %v", name, err)
	}
}

func main() {
	// Use os.DirFS as the root fs.FS for demonstration purposes
	rootFS := os.DirFS(os.Args[1])

	server := &FSFileServer{Root: rootFS}
	addr := ":8080"

	if err := http.ListenAndServe(addr, server); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}
}
