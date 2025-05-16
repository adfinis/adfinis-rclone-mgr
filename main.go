package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/adfinis/adfinis-rclone-mount/models"
	"github.com/adfinis/adfinis-rclone-mount/templates"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	drive "google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
)

var (
	// Version is the current version of gmotd
	Version = "devel"
	// Commit is the git commit hash of the current version.
	Commit = "none"
	// Date is the build date of the current version.
	Date = "unknown"
	// BuiltBy is the user who built the current version.
	BuiltBy = "unknown"
)

const (
	listenPort = 53682
)

var (
	state = uuid.NewString()
)

var rootCmd = &cobra.Command{
	Use:   "adfinis-rclone-mount",
	Short: "Manage Google Drive mounts using Rclone",
	Run:   root,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	Version: Version,
}

func init() {
	rootCmd.AddCommand(
		mountCmd,
		umountCmd,
		versionCmd,
	)
}

func root(cmd *cobra.Command, _ []string) {
	ctx, cancel := context.WithCancel(cmd.Context())

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", listenPort),
		Handler: newHttpHandler(ctx, cancel),
	}
	go func() {
		log.Printf("Visit http://localhost:%d to start login", listenPort)
		openBrowser(fmt.Sprintf("http://localhost:%d/", listenPort))
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	select {
	case <-ctx.Done():
		log.Println("Server stopped")
	case <-time.After(time.Hour):
		log.Println("Server timed out")
	}

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Server shutdown error: %v", err)
	}
	log.Println("Server shutdown gracefully")
}

func Execute() error {
	return rootCmd.Execute()
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		log.Println("Unsupported platform, please open the URL manually:", url)
	}
	if err != nil {
		log.Printf("Failed to open browser: %v", err)
	}
}

func newHttpHandler(ctx context.Context, cancel context.CancelFunc) *http.ServeMux {
	router := http.NewServeMux()

	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// try to read the oidc credentials from the local keyring
		clientID, clientSecret, err := getCredentials()
		if err != nil {
			log.Printf("Failed to get credentials from keyring: %v", err)
		}
		if err := templates.ComponentInputForm(clientID, clientSecret).Render(ctx, w); err != nil {
			log.Printf("Failed to render template: %v", err)
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	})

	router.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}
		clientID := r.FormValue("client_id")
		clientSecret := r.FormValue("client_secret")
		if clientID == "" || clientSecret == "" {
			http.Error(w, "Missing client_id or client_secret", http.StatusBadRequest)
			return
		}

		// save the credentials to the local keyring
		if err := setCredentials(clientID, clientSecret); err != nil {
			log.Printf("Failed to set credentials in keyring: %v", err)
		}

		oauthConfig := &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  fmt.Sprintf("http://localhost:%d/auth", listenPort),
			Scopes:       []string{drive.DriveScope},
			Endpoint:     google.Endpoint,
		}

		sessionRaw := fmt.Sprintf("%s|%s", clientID, clientSecret)
		sessionEncoded := base64.StdEncoding.EncodeToString([]byte(sessionRaw))
		http.SetCookie(w, &http.Cookie{
			Name:    "oidc",
			Value:   sessionEncoded,
			Expires: time.Now().Add(time.Hour)},
		)

		url := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline)
		http.Redirect(w, r, url, http.StatusFound)
	})

	router.HandleFunc("/auth", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != state {
			http.Error(w, "Invalid state parameter", http.StatusBadRequest)
			return
		}

		code := r.URL.Query().Get("code")
		cookie, err := r.Cookie("oidc")
		if err != nil {
			http.Error(w, "missing oidc cookie", http.StatusBadRequest)
			return
		}

		cookieValue, err := base64.StdEncoding.DecodeString(cookie.Value)
		if err != nil {
			http.Error(w, "Failed to decode oidc cookie: "+err.Error(), http.StatusBadRequest)
			return
		}
		parts := strings.SplitN(string(cookieValue), "|", 2)
		if len(parts) != 2 {
			http.Error(w, "Invalid oidc cookie", http.StatusBadRequest)
			log.Println(parts)
			return
		}

		clientID, clientSecret := parts[0], parts[1]

		oauthConfig := &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  fmt.Sprintf("http://localhost:%d/auth", listenPort),
			Scopes:       []string{drive.DriveScope},
			Endpoint:     google.Endpoint,
		}

		token, err := oauthConfig.Exchange(ctx, code)
		if err != nil {
			http.Error(w, "Failed to exchange token: "+err.Error(), http.StatusInternalServerError)
			return
		}

		tokenString, err := json.Marshal(token)
		if err != nil {
			http.Error(w, "Failed to serialize token: "+err.Error(), http.StatusInternalServerError)
			return
		}
		tokenStringEncoded := base64.StdEncoding.EncodeToString(tokenString)
		http.SetCookie(w, &http.Cookie{
			Name:    "token",
			Value:   tokenStringEncoded,
			Expires: time.Now().Add(time.Hour),
		})

		w.WriteHeader(http.StatusOK)

		mySharedDrives, err := checkAvailableDrives(ctx, oauthConfig, token)
		if err != nil {
			http.Error(w, "Failed to check available drives: "+err.Error(), http.StatusInternalServerError)
		}

		if err := templates.ComponentDriveSelection(mySharedDrives).Render(ctx, w); err != nil {
			log.Printf("Failed to render template: %v", err)
			http.Error(w, "Failed to render template", http.StatusInternalServerError)
			return
		}
	})

	router.HandleFunc("/generate", func(w http.ResponseWriter, r *http.Request) {
		defer cancel()

		cookie, err := r.Cookie("token")
		if err != nil {
			http.Error(w, "missing token cookie", http.StatusBadRequest)
			return
		}
		// decode the token
		tokenValue, err := base64.StdEncoding.DecodeString(cookie.Value)
		if err != nil {
			http.Error(w, "Failed to decode token: "+err.Error(), http.StatusBadRequest)
			return
		}

		cookie, err = r.Cookie("oidc")
		if err != nil {
			http.Error(w, "missing oidc cookie", http.StatusBadRequest)
			return
		}
		// decode the oidc cookie
		cookieValue, err := base64.StdEncoding.DecodeString(cookie.Value)
		if err != nil {
			http.Error(w, "Failed to decode oidc cookie: "+err.Error(), http.StatusBadRequest)
			return
		}
		parts := strings.SplitN(string(cookieValue), "|", 2)
		clientID, clientSecret := parts[0], parts[1]

		if err := r.ParseForm(); err != nil {
			http.Error(w, "Failed to parse form: "+err.Error(), http.StatusBadRequest)
			return
		}

		enabled := map[string]bool{}
		for _, id := range r.Form["drive"] {
			enabled[id] = true
		}
		automount := map[string]bool{}
		for _, id := range r.Form["automount"] {
			automount[id] = true
		}
		var result []models.Drive
		for _, idName := range r.Form["drive_name"] {
			idNameParts := strings.SplitN(idName, ":", 2)
			id := idNameParts[0]
			name := idNameParts[1]
			result = append(result, models.Drive{
				Name:      name,
				ID:        id,
				Enabled:   enabled[id],
				AutoMount: automount[id],
			})
		}

		fmt.Println(result)

		if err := handleRcloneConfig(ctx, result, clientID, clientSecret, string(tokenValue)); err != nil {
			log.Printf("Failed to handle rclone config: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			if err := templates.ComponentError(err.Error()).Render(ctx, w); err != nil {
				log.Printf("Failed to render template: %v", err)
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
			}
			return
		}

		if err := handleSystemdServices(ctx, result); err != nil {
			log.Printf("Failed to handle systemd services: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			if err := templates.ComponentError(err.Error()).Render(ctx, w); err != nil {
				log.Printf("Failed to render template: %v", err)
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
				return
			}
		} else {
			w.WriteHeader(http.StatusOK)
			if err := templates.ComponentSuccess().Render(ctx, w); err != nil {
				log.Printf("Failed to render template: %v", err)
				http.Error(w, "Failed to render template", http.StatusInternalServerError)
				return
			}
		}
	})
	return router
}

func checkAvailableDrives(ctx context.Context, oauthConfig *oauth2.Config, token *oauth2.Token) ([]models.Drive, error) {
	driveService, err := drive.NewService(
		ctx,
		option.WithScopes(drive.DriveMetadataReadonlyScope),
		option.WithTokenSource(oauthConfig.TokenSource(ctx, token)),
	)
	if err != nil {
		return nil, err
	}

	sharedDrives := []models.Drive{
		{
			Name: "My Drive",
			ID:   "my_drive",
		},
	}
	pageToken := ""
	for {
		req := driveService.Drives.List().PageSize(10)
		if pageToken != "" {
			req = req.PageToken(pageToken)
		}

		resp, err := req.Do()
		if err != nil {
			return nil, err
		}

		for _, d := range resp.Drives {
			sharedDrives = append(sharedDrives, models.Drive{
				Name: d.Name,
				ID:   d.Id,
			})
		}

		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return sharedDrives, nil
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version info",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Date: %s\n", Date)
		fmt.Printf("BuiltBy: %s\n", BuiltBy)
	},
}

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
