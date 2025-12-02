package httpfs

import (
	"bytes"
	"io"
	"io/fs"
	"net/http"
)

type RequestModifier func(*http.Request)

// HTTPFS implements the fs.FS interface over an http endpoint
type HTTPFS struct {
	Url      string
	Client   *http.Client
	Modifier RequestModifier
}

// TEST IMPLEMENTATIONS OF INTERFACES
var _ fs.FS = (*HTTPFS)(nil)

func (hfs *HTTPFS) Open(name string) (fs.File, error) {
	// 1. Fetch metadata (using a HEAD-like request or dedicated Stat API)
	info, err := hfs.Stat(name)
	if err != nil {
		return nil, err
	}

	// 2. Handle Directory
	if info.IsDir() {
		// For directories, Open typically loads the entry list if fs.ReadDirFS is not used.
		// Since we implement ReadDirFS, we can rely on that, but the fs.File returned
		// must still implement ReadDirFile (which HTTPFile does).
		entries, err := hfs.ReadDir(name)
		if err != nil && err != io.EOF {
			return nil, err
		}
		// Return a directory file handle pre-populated with entries
		return &HTTPFile{
			name:       name,
			isDir:      true,
			size:       0,
			entries:    entries,
			readCloser: io.NopCloser(bytes.NewReader(nil)), // Directory has no body
			entriesI:   0,
		}, nil
	}

	// 3. Handle Regular File (make a GET request)
	body, size, err := mockGetAPI(hfs.BaseURL, name) // Mock HTTP GET
	if err != nil {
		return nil, err
	}

	return &HTTPFile{
		name:       name,
		isDir:      false,
		size:       size,
		readCloser: body,
		entriesI:   0,
	}, nil
}

// Stat implements the Stat function for the filesystem (a standard utility, often mirrored by the FS itself).
// Note: This is not part of the standard fs.FS interface but is provided here for completeness,
// and to be called by Open. We implement it based on the name path.
func (hfs *HTTPFS) Stat(name string) (fs.FileInfo, error) {
	// Mock HTTP HEAD/Stat call to get remote file metadata
	info, err := mockStatAPI(hfs.BaseURL, name)
	if err != nil {
		return nil, err
	}
	return info, nil
}

// ReadDir implements the fs.ReadDirFS interface.
func (hfs *HTTPFS) ReadDir(name string) ([]fs.DirEntry, error) {
	// Mock HTTP call to get directory listing
	entries, err := mockListAPI(hfs.BaseURL, name)
	if err != nil {
		return nil, err
	}
	return entries, nil
}
