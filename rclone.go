package main

import (
	"context"
	"fmt"
	"slices"

	"log"

	"github.com/adfinis/adfinis-rclone-mgr/v2/models"
	"github.com/rclone/rclone/fs/config"
	"github.com/rclone/rclone/fs/config/configfile"
	"github.com/rclone/rclone/fs/rc"
	"github.com/samber/lo"

	_ "github.com/rclone/rclone/backend/drive" // make sure drive backend is registered
)

func init() {
	// make sure we have a config file
	configfile.Install()
}

func handleRcloneConfig(ctx context.Context, drives []models.Drive, clientID, clientSecret, token string) ([]models.Drive, error) {
	// add drives and remove deleted drives
	for _, drive := range drives {
		driveName := sanitizeDriveName(drive.Name)
		if drive.Enabled {
			var configMap rc.Params
			switch drive.ID {
			case "my_drive":
				configMap = rc.Params{
					"type":           "drive",
					"root_folder_id": "",
					"scope":          "drive",
					"client_id":      clientID,
					"client_secret":  clientSecret,
					"token":          token,
				}
			case "shared_with_me":
				configMap = rc.Params{
					"type":           "drive",
					"root_folder_id": "",
					"scope":          "drive",
					"client_id":      clientID,
					"client_secret":  clientSecret,
					"token":          token,
					"shared_with_me": true,
				}
			default:
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
				return nil, fmt.Errorf("failed to create remote %s: %w", driveName, err)
			}
			log.Printf("Added remote %q", driveName)
		} else {
			// remove the remote if it exists
			log.Printf("Removing remote %q", driveName)
			config.DeleteRemote(driveName)
		}
	}

	// check for drives with a matching client secret that dont exist in gdrive anymore
	deletedDrives := make([]models.Drive, 0)
	allRemotes := config.GetRemotes()
	driveNames := lo.Map(drives, func(d models.Drive, _ int) string {
		return sanitizeDriveName(d.Name)
	})
	for _, remote := range allRemotes {
		if slices.Contains(driveNames, remote.Name) {
			continue
		}
		remoteClientID := config.GetValue(remote.Name, "client_id")
		if remoteClientID == clientID {
			config.DeleteRemote(remote.Name)
			log.Printf("Removed remote %q", remote.Name)
			deletedDrives = append(deletedDrives, models.Drive{
				Name: remote.Name,
			})
		}
	}

	return deletedDrives, nil
}

func getRemotes() []string {
	return config.GetRemoteNames()
}
