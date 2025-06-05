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

type copyRequest struct {
	Sources []string `json:"sources"`
}

func newHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/gdrive/copy", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			w.Write([]byte("Method not allowed")) // nolint:errcheck
			return
		}
		var req copyRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil || len(req.Sources) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("Invalid request: must provide sources")) // nolint:errcheck
			return
		}

		log.Println("Received copy request for sources:", req.Sources)

		// copy files in background
		go selectDestAndCopy(req.Sources)

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK")) // nolint:errcheck
	})

	return mux
}
