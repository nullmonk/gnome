// ... (main function starts) ...
package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"net"
	"net/http"

	"golang.org/x/crypto/ssh"
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

// --- Custom net.Listener Implementation (Feeds the Conn to the Server) ---

type PipeListener struct {
	conn net.Conn
}

// Accept returns the single connection immediately.
// It is the entry point for http.Serve.
func (l *PipeListener) Accept() (net.Conn, error) {
	// The HTTP server calls Accept() repeatedly. Since we only have one session
	// pipe connection, we return it once and then block forever or return EOF.
	if l.conn != nil {
		c := l.conn
		l.conn = nil // Only return the connection once
		return c, nil
	}
	// After the first return, block or signal end to the server.
	// For a single request/response cycle, io.EOF is typical.
	// For multiple, you'd need a more complex signal/wait loop.
	time.Sleep(1 * time.Hour) // Simply block for demonstration
	return nil, io.EOF
}

func (l *PipeListener) Close() error {
	// This should ideally close the underlying SSH pipes, but since they
	// are owned by the session, we'll let the session.Close() handle it.
	return nil
}

func (l *PipeListener) Addr() net.Addr {
	// Required by net.Listener, but not relevant for pipes.
	return nil
}

func main() {
	home, _ := os.UserHomeDir()
	keyBytes, err := os.ReadFile(filepath.Join(home, ".ssh/id_ed25519"))
	if err != nil {
		log.Fatalf("Failed to read private key file: %v", err)
	}

	signer, _ := ssh.ParsePrivateKey(keyBytes)

	config := &ssh.ClientConfig{
		User: "lethicon",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// 2. Dial the SSH Server
	conn, err := ssh.Dial("tcp", "localhost:22", config)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// 3. Create a new Session
	session, err := conn.NewSession()
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}
	defer session.Close()

	// 4. Get I/O Streams
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		log.Fatalf("Unable to setup stdin for session: %v", err)
	}
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		log.Fatalf("Unable to setup stdout for session: %v", err)
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		log.Fatalf("Unable to setup stderr for session: %v", err)
	}

	// The remote command to execute (the HTTP client binary/script)
	cmd := "/home/lethicon/Projects/gnome/client"

	// 5. Start the command
	if err := session.Start(cmd); err != nil {
		log.Fatalf("Failed to start command: %v", err)
	}

	// --- The Core Logic ---
	serveHttpOverSSH(stdinPipe, stdoutPipe, stderrPipe)

	// Wait for the remote command to finish
	if err := session.Wait(); err != nil {
		log.Printf("Remote command finished with error: %v", err)
	} else {
		log.Println("Remote command finished successfully.")
	}
}

// ... (serveHttpOverSSH function) ...
func serveHttpOverSSH(stdin io.Writer, stdout io.Reader, stderr io.Reader) {

	conn := &PipeConn{
		Reader: stdout,
		Writer: stdin,
	}

	// --- 3. Create the Custom net.Listener ---

	// This Listener will immediately return our single PipeConn object.
	listener := &PipeListener{conn: conn}

	// --- 4. Define the HTTP Handler ---
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("🔥 SSH Client (HTTP Server) received request from remote! Method: %s, Path: %s", r.Method, r.URL)

		// Example logic: read data from the client
		bodyBytes, _ := io.ReadAll(r.Body)
		log.Printf("Request Body from remote: %s", string(bodyBytes))

		// Write the response back to the client (which is the remote binary)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "Hello, Remote HTTP Client! Received data: %s", string(bodyBytes))
	})

	// --- 5. Start the HTTP Server on the Listener ---
	log.Println("Serving HTTP requests over the SSH session pipes...")

	// http.Serve immediately calls listener.Accept() to get the conn and start
	// reading the request from the remote binary.
	if err := http.Serve(listener, mux); err != nil && err != io.EOF {
		log.Fatalf("HTTP server on SSH pipes failed: %v", err)
	}

	log.Println("HTTP server processing complete.")
}
