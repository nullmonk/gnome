package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

type PipeConn struct {
	io.Reader
	io.Writer
}

func (pc *PipeConn) Close() error                       { return nil }
func (pc *PipeConn) LocalAddr() net.Addr                { return nil }
func (pc *PipeConn) RemoteAddr() net.Addr               { return nil }
func (pc *PipeConn) SetDeadline(t time.Time) error      { return nil }
func (pc *PipeConn) SetReadDeadline(t time.Time) error  { return nil }
func (pc *PipeConn) SetWriteDeadline(t time.Time) error { return nil }

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// --- 1. Signal Readiness (to the SSH Client's Stderr) ---
	// This lets the SSH client know it can start its HTTP server.
	fmt.Fprintf(os.Stderr, "%s\n", "Here we go")

	conn := &PipeConn{
		Reader: os.Stdin,  // Read the HTTP response from the SSH client (server)
		Writer: os.Stdout, // Write the HTTP request to the SSH client (server)
	}

	client := &http.Client{
		Transport: &http.Transport{
			// Replace the standard Dial function with one that returns our custom connection.
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return conn, nil
			},
			// Disable connection pooling since we only have one persistent pipe.
			DisableKeepAlives: true,
		},
	}
	req, err := http.NewRequest("GET", "http://pipehost/execute?data=test", nil)
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Failed to execute HTTP request over custom connection: %v", err)
	}
	defer resp.Body.Close()

	log.Println("--- Received HTTP Response (via Stdin) ---")

	// Dump all response headers and status
	fmt.Printf("Status: %s\n", resp.Status)
	fmt.Println("Headers:")
	for k, v := range resp.Header {
		fmt.Printf("  %s: %s\n", k, v)
	}

	// Dump the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error reading response body: %v", err)
	}
	fmt.Println("\nBody:")
	fmt.Println(string(bodyBytes))

	log.Println("Processing complete. Remote HTTP client exiting.")
}
