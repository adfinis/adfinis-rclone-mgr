package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path"

	"github.com/adrg/xdg"
	"github.com/spf13/cobra"
)

var socketDirPath = path.Join(xdg.RuntimeDir, "adfinis-rclone-mgr")

func init() {
	if err := os.MkdirAll(socketDirPath, 0o700); err != nil {
		panic(err)
	}
}

func daemon(cmd *cobra.Command, args []string) {
	driveName := args[0]
	ctx := cmd.Context()

	go journaldReader(ctx, driveName)
	go ipcServer(driveName)

	<-ctx.Done()
}

func cleanupOldSocket(driveName string) {
	socketPath := path.Join(socketDirPath, fmt.Sprintf("%s.sock", driveName))
	if _, err := os.Stat(socketPath); err == nil {
		if err := os.Remove(socketPath); err != nil {
			log.Printf("Failed to remove old socket file %s: %v", socketPath, err)
		} else {
			log.Printf("Removed old socket file: %s", socketPath)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("Error checking for old socket file %s: %v", socketPath, err)
	}
}

func ipcServer(driveName string) {
	cleanupOldSocket(driveName)

	srv := http.Server{
		Handler: newHTTPHandler(),
	}

	unixListener, err := net.Listen("unix", path.Join(socketDirPath, fmt.Sprintf("%s.sock", driveName)))
	if err != nil {
		log.Fatal("Failed to create Unix socket listener:", err)
	}

	defer func() {
		if err := unixListener.Close(); err != nil {
			log.Printf("Failed to close Unix socket listener: %v", err)
		}
	}()

	if err := srv.Serve(unixListener); err != nil {
		if err != http.ErrServerClosed {
			log.Fatal("Failed to start IPC server:", err)
		} else {
			log.Println("IPC server closed gracefully")
		}
	}
}

type gdriveOPRequest struct {
	Sources []string `json:"sources"`
}

func handleGDriveOp(op string) http.HandlerFunc {
	switch op {
	case "copy", "move":
		return handleGDriveOpWithFileSelect(op)
	case "duplicate":
		return handleGDriveOPDuplicate()
	case "open":
		return handleGDriveOpen()
	case "link":
		return handleGDriveLink()
	default:
		return func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid operation")) // nolint:errcheck
		}
	}
}

func parseOpRequestBody(op string, w http.ResponseWriter, r *http.Request) *gdriveOPRequest {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed")) // nolint:errcheck
		return nil
	}
	var req gdriveOPRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || len(req.Sources) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Invalid request: must provide sources")) // nolint:errcheck
		return nil
	}

	log.Printf("Received %s request for sources: %v", op, req.Sources)
	return &req
}

func handleGDriveOpWithFileSelect(op string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := parseOpRequestBody(op, w, r)
		if req == nil {
			return // Error response already sent in parseOpRequestBody
		}

		// run files in background
		go selectDestAndRunOP(req.Sources, op)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) // nolint:errcheck
	}
}

func handleGDriveOPDuplicate() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := parseOpRequestBody("duplicate", w, r)
		if req == nil {
			return // Error response already sent in parseOpRequestBody
		}

		// run in background
		go duplicateFiles(req.Sources)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) // nolint:errcheck
	}
}

func handleGDriveOpen() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := parseOpRequestBody("open", w, r)
		if req == nil {
			return
		}

		// run in background
		go openInBrowser(req.Sources)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) // nolint:errcheck
	}
}

func handleGDriveLink() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req := parseOpRequestBody("link", w, r)
		if req == nil {
			return
		}

		// run in background
		go copyLinkToClipboard(req.Sources)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) // nolint:errcheck
	}
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/gdrive/copy", handleGDriveOp("copy"))
	mux.HandleFunc("/gdrive/move", handleGDriveOp("move"))
	mux.HandleFunc("/gdrive/duplicate", handleGDriveOp("duplicate"))
	mux.HandleFunc("/gdrive/open", handleGDriveOp("open"))
	mux.HandleFunc("/gdrive/link", handleGDriveOp("link"))
	return mux
}
