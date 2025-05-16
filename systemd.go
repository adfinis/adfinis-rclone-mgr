package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/adfinis/adfinis-rclone-mount/models"
	"github.com/adrg/xdg"
	"github.com/coreos/go-systemd/v22/dbus"
)

func ensureFolderExists(path string) error {
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}
	return nil
}

func removeDriveCache(name string) error {
	cachePath := getDriveCachePath(name)
	if _, err := os.Stat(cachePath); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to stat cache path %s: %w", cachePath, err)
	}
	if err := os.RemoveAll(cachePath); err != nil {
		return fmt.Errorf("failed to remove cache path %s: %w", cachePath, err)
	}
	return nil
}

func getDriveDataPath(name string) string {
	return path.Join(xdg.Home, "google", name)
}

func getDriveCachePath(name string) string {
	return path.Join(xdg.CacheHome, "google", name)
}

func handleSystemdServices(ctx context.Context, drives []models.Drive) error {
	conn, err := dbus.NewUserConnectionContext(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	// make sure systmed know about the rclone mount service
	if err := conn.ReloadContext(ctx); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	var errs []error
	for _, drive := range drives {
		name := sanitizeDriveName(drive.Name)
		if err := ensureFolderExists(getDriveDataPath(name)); err != nil {
			return err
		}
		if err := ensureFolderExists(getDriveCachePath(name)); err != nil {
			return err
		}

		if drive.Enabled {
			if err := startService(ctx, conn, name); err != nil {
				errs = append(errs, err)
				continue
			}
			if drive.AutoMount {
				if err := enableService(ctx, conn, name); err != nil {
					errs = append(errs, err)
					continue
				}
			}

		} else {
			if err := stopService(ctx, conn, name); err != nil {
				errs = append(errs, err)
				continue
			}
			if err := disableService(ctx, conn, name); err != nil {
				errs = append(errs, err)
				continue
			}
			if err := removeDriveCache(name); err != nil {
				continue
			}
		}
	}
	// pretty error string
	if len(errs) > 0 {
		errStr := "Error while handling systemd services:\n"
		for _, err := range errs {
			errStr += fmt.Sprintf("- %s\n", err.Error())
		}
		return errors.New(errStr)
	}
	return nil
}

func enableService(ctx context.Context, conn *dbus.Conn, name string) error {
	serviceName := fmt.Sprintf("rclone@%s.service", name)
	_, _, err := conn.EnableUnitFilesContext(ctx, []string{serviceName}, false, true)
	if err != nil {
		return fmt.Errorf("failed to enable service %q: %w", serviceName, err)
	}
	return nil
}

func startService(ctx context.Context, conn *dbus.Conn, name string) error {
	serviceName := fmt.Sprintf("rclone@%s.service", name)
	ch := make(chan string)
	_, err := conn.StartUnitContext(ctx, serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to start service %q: %w", serviceName, err)
	}
	result := <-ch

	if result != "done" {
		return fmt.Errorf("failed to start service %q: %s", serviceName, result)
	}
	return nil
}

func stopService(ctx context.Context, conn *dbus.Conn, name string) error {
	serviceName := fmt.Sprintf("rclone@%s.service", name)
	ch := make(chan string)
	_, err := conn.StopUnitContext(ctx, serviceName, "replace", ch)
	if err != nil {
		return fmt.Errorf("failed to stop service %q: %w", serviceName, err)
	}
	result := <-ch

	if result != "done" {
		return fmt.Errorf("failed to stop service %q: %s", serviceName, result)
	}
	return nil
}

func disableService(ctx context.Context, conn *dbus.Conn, name string) error {
	serviceName := fmt.Sprintf("rclone@%s.service", name)
	_, err := conn.DisableUnitFilesContext(ctx, []string{serviceName}, false)
	if err != nil {
		return fmt.Errorf("failed to disable service %q: %w", serviceName, err)
	}
	return nil
}

func statusServices(ctx context.Context, conn *dbus.Conn, names []string) ([]dbus.UnitStatus, error) {
	unitNames := make([]string, len(names))
	for i, n := range names {
		unitNames[i] = fmt.Sprintf("rclone@%s.service", n)
	}
	return conn.ListUnitsByNamesContext(ctx, unitNames)
}
