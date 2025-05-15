package main

import (
	"context"
	"fmt"
	"strings"

	"log"

	"github.com/adfinis/adfinis-rclone-mount/models"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rclone/rclone/fs/rc"

	_ "github.com/rclone/rclone/backend/drive" // make sure drive backend is registered
)

// dont ask...
func sanitizeDriveName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "?", "_")
	name = strings.ReplaceAll(name, "*", "_")
	name = strings.ReplaceAll(name, "\"", "_")
	name = strings.ReplaceAll(name, "<", "_")
	name = strings.ReplaceAll(name, ">", "_")
	name = strings.ReplaceAll(name, "|", "_")
	name = strings.ReplaceAll(name, "'", "_")
	name = strings.ReplaceAll(name, "&", "_")
	name = strings.ReplaceAll(name, "%", "_")
	return name
}

func handleRcloneConfig(ctx context.Context, drives []models.Drive, clientID, clientSecret, token string) error {
	// make sure we have a config file
	configfile.Install()

	// add drives
	for _, drive := range drives {
		driveName := sanitizeDriveName(drive.Name)
		if drive.Enabled {
			var configMap rc.Params
			// special case for personal drive
			if drive.ID == "my_drive" {
				configMap = rc.Params{
					"type":           "drive",
					"root_folder_id": "",
					"scope":          "drive",
					"client_id":      clientID,
					"client_secret":  clientSecret,
					"token":          token,
				}
			} else {
				// shared drive
				configMap = rc.Params{
					"type":           "drive",
					"team_drive":     drive.ID,
					"root_folder_id": "",
					"scope":          "drive",
					"client_id":      clientID,
					"client_secret":  clientSecret,
					"token":          token,
				}
			}
			_, err := config.CreateRemote(ctx, driveName, "drive", configMap, config.UpdateRemoteOpt{NonInteractive: true})
			if err != nil {
				return fmt.Errorf("failed to create remote %s: %w", driveName, err)
			}
			log.Printf("Added remote %q", driveName)
		} else {
			// remove the remote if it exists
			log.Printf("Removing remote %q", driveName)
			config.DeleteRemote(driveName)
		}
	}
	return nil
}
