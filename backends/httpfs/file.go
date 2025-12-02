package httpfs

import (
	"fmt"
	"io"
	"io/fs"
	"time"
)

var _ fs.File = (*HTTPFile)(nil)

/*
Implement DirInfo, FileInfo, and File based on a JSON response
*/
type HTTPFile struct {
	name       string
	size       int64
	readCloser io.ReadCloser
	isDir      bool
	entries    []fs.DirEntry
	entriesI   int // current index in the entries struct
}

// Name returns the base name of the file.
func (hf *HTTPFile) Name() string {
	return hf.name
}

// Size returns the size of the file in bytes.
func (hf *HTTPFile) Size() int64 {
	return hf.size
}

// Mode returns the file mode bits.
func (hf *HTTPFile) Mode() fs.FileMode {
	if hf.isDir {
		return fs.ModeDir | 0555 // Example directory mode
	}
	return 0444 // Example file mode
}

// ModTime returns the modification time.
func (hf *HTTPFile) ModTime() time.Time {
	return time.Now()
}

// IsDir reports whether hf is a directory.
func (hf *HTTPFile) IsDir() bool {
	return hf.isDir
}

// Sys returns the underlying data source.
func (hf *HTTPFile) Sys() interface{} {
	return nil
}

// Type returns the type bits for the entry (required for fs.DirEntry).
func (hf *HTTPFile) Type() fs.FileMode {
	return hf.Mode().Type()
}

// Info returns the FileInfo for the entry (required for fs.DirEntry).
func (hf *HTTPFile) Info() (fs.FileInfo, error) {
	return hf, nil
}

// Stat returns the file info (required by fs.File).
func (hf *HTTPFile) Stat() (fs.FileInfo, error) {
	return hf, nil
}

// Read reads data from the underlying response body (required by fs.File).
func (hf *HTTPFile) Read(p []byte) (n int, err error) {
	return hf.readCloser.Read(p)
}

// Close closes the underlying HTTP response body (required by fs.File).
func (hf *HTTPFile) Close() error {
	return hf.readCloser.Close()
}

func (hf *HTTPFile) ReadDir(count int) ([]fs.DirEntry, error) {
	if !hf.isDir {
		return nil, fmt.Errorf("%s is not a directory", hf.name)
	}

	start := hf.entriesI
	end := len(hf.entries)

	// If we are already past the last entry, return EOF immediately
	if start >= end {
		return nil, io.EOF
	}

	// Determine the end index based on 'count'
	if count > 0 {
		end = start + count
		if end > len(hf.entries) {
			end = len(hf.entries)
		}
	}

	// Slice the entries and advance the index
	result := hf.entries[start:end]
	hf.entriesI = end

	// Determine if this is the last chunk
	if hf.entriesI == len(hf.entries) {
		return result, io.EOF
	}

	return result, nil
}
