package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

// LogEntry represents a parsed journalctl log line
type LogEntry struct {
	Message   string `json:"MESSAGE"`
	Timestamp string `json:"__REALTIME_TIMESTAMP"`
	Priority  string `json:"PRIORITY"`
	Unit      string `json:"_SYSTEMD_UNIT"`
}

func journaldReader(cmd *cobra.Command, args []string) {
	ctx := cmd.Context()
	driveName := args[0]
	logs, errs := startJournalReader(ctx, driveName)

	go func() {
		for l := range logs {
			handleLogEntry(l, driveName)
		}
	}()

	go func() {
		for err := range errs {
			fmt.Printf("Error: %v\n", err)
		}
	}()

	<-ctx.Done()
}

func startJournalReader(ctx context.Context, name string) (<-chan LogEntry, <-chan error) {
	logs := make(chan LogEntry)
	errs := make(chan error, 1)

	cmd := exec.Command(
		"journalctl",
		"--output=json",
		"--follow",
		"--user",
		"--since=now",
		fmt.Sprintf("--unit=%s", driveNameToUnitName(name)),
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		errs <- fmt.Errorf("failed to get stdout: %w", err)
		close(logs)
		close(errs)
		return logs, errs
	}

	if err := cmd.Start(); err != nil {
		errs <- fmt.Errorf("failed to start journalctl: %w", err)
		close(logs)
		close(errs)
		return logs, errs
	}

	go func() {
		defer close(logs)
		defer close(errs)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				_ = cmd.Process.Kill()
				return
			default:
				var entry LogEntry
				line := scanner.Text()
				if err := json.Unmarshal([]byte(line), &entry); err != nil {
					errs <- fmt.Errorf("failed to parse log line: %w", err)
					continue
				}
				logs <- entry
			}
		}

		if err := scanner.Err(); err != nil && err != io.EOF {
			errs <- fmt.Errorf("scanner error: %w", err)
		}

		if err := cmd.Wait(); err != nil {
			errs <- fmt.Errorf("journalctl command error: %w", err)
		}
	}()

	return logs, errs
}

var ignoredErrorsByDrive = map[string][]string{
	"shared_with_me": {
		"ERROR : IO error: open file failed: googleapi: Error 403: This file cannot be downloaded by the user., cannotDownloadFile", // weird error on shared_with_me
	},
	"all": {
		"Failed to copy: googleapi: Error 403", // occurs when trying to copy a file without permissions
		"failed to create directory",           // occurs when trying to create a directory without permissions
		"failed to make directory",             // occurs when trying to create a directory without permissions
	},
}

func shouldTriggerError(entry LogEntry, driveName string) bool {
	// anything that doesnt contain "ERROR" can be ignored
	if !strings.Contains(entry.Message, "ERROR") {
		return false
	}

	allIgnoredErrs := make([]string, 0)
	ignoredErrs, ok := ignoredErrorsByDrive[strings.ToLower(driveName)]
	if ok {
		allIgnoredErrs = append(allIgnoredErrs, ignoredErrs...)
	}
	allIgnoredErrs = append(allIgnoredErrs, ignoredErrorsByDrive["all"]...)

	for _, ie := range allIgnoredErrs {
		if strings.Contains(entry.Message, ie) {
			return false
		}
	}
	return true
}

var fileNameRegex = regexp.MustCompile(`ERROR\s+:\s+(.+?):\s`)

func fileNameFromEntry(entry LogEntry) string {
	s := fileNameRegex.FindStringSubmatch(entry.Message)
	if len(s) < 2 {
		return ""
	}
	return s[1]
}

func shouldTriggerFileMove(entry LogEntry) bool {
	return strings.Contains(entry.Message, "vfs cache: failed to upload") &&
		strings.Contains(entry.Message, "insufficientParentPermissions")
}

func handleLogEntry(entry LogEntry, driveName string) {
	if !shouldTriggerError(entry, driveName) {
		fmt.Println("Ignoring log entry:", entry.Message)
		return
	}

	// ask user to move file
	if shouldTriggerFileMove(entry) {
		requestFileMove(entry, driveName)
		return
	}

	// just send a notification
	title := fmt.Sprintf("Drive Error: %s", driveName)
	message := fmt.Sprintf("The following error occurred:\n\n%s", entry.Message)
	if err := sendDesktopNotificationError(title, message); err != nil {
		fmt.Printf("Failed to send notification: %v\n", err)
	}

	fmt.Println("Notified about error:", entry.Message)
}

func requestFileMove(entry LogEntry, driveName string) {
	fileName := fileNameFromEntry(entry)
	filePath := fileNameToPath(driveName, fileName)

	// make sure file still exists
	if _, err := os.Stat(filePath); err != nil {
		return
	}

	title := fmt.Sprintf("Drive Error: %s", driveName)
	message := fmt.Sprintf(`You have insufficient permissions to write a file:

- %s

Make sure to move the file you just created to another location immediately!
			`, getDriveDataPath(driveName))
	if err := sendDesktopNotificationError(title, message); err != nil {
		fmt.Printf("Failed to send notification: %v\n", err)
	}

	// open file selector to select the file location
	title = "Select File Location"
	message = fmt.Sprintf("Select a new location for the file:\n\n%s", filePath)

	newFilePath, err := openFileSelector(title, message, fileName)
	if err != nil {
		if strings.Contains(err.Error(), "exit status 1") {
			fmt.Println("File selector was cancelled, skipping...")
			return
		}
		fmt.Printf("Failed to open file selector: %v\n", err)
		return
	}
	if newFilePath == "" {
		fmt.Println("No file selected, skipping...")
		return
	}
	if err := moveFile(filePath, newFilePath); err != nil {
		title = "Error Moving File"
		message = fmt.Sprintf("Failed to move file:\n\n%s", err)
		if err := sendDesktopNotificationError(title, message); err != nil {
			fmt.Printf("Failed to send notification: %v\n", err)
		}
		fmt.Printf("Failed to move file: %v\n", err)
		return
	}
	title = "File Moved"
	message = fmt.Sprintf("File moved to:\n\n%s", newFilePath)
	if err := sendDesktopNotificationInfo(title, message); err != nil {
		fmt.Printf("Failed to send notification: %v\n", err)
	}
	fmt.Printf("File %q moved to: %s\n", filePath, newFilePath)
}

// moveFile moves a file from oldPath to newPath by copying the file and removing the old file.
// This is a workaround for the issue where os.Rename fails across different filesystems (and rclone mounts).
func moveFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() // nolint:errcheck

	info, err := in.Stat()
	if err != nil {
		return err
	}

	// create new file. This will overwrite the file if it exists
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		if strings.Contains(err.Error(), "is a directory") {
			return fmt.Errorf("Destination %q is a directory.\n\nYou cant replace a file with a directory!", dest) // nolint:staticcheck
		}
		return err
	}
	defer out.Close() // nolint:errcheck

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	// Close the files before removing
	in.Close()  // nolint:errcheck
	out.Close() // nolint:errcheck

	// Remove the original file
	if err = os.Remove(src); err != nil {
		return err
	}

	return nil
}

func sendDesktopNotificationError(title, message string) error {
	cmd := exec.Command("zenity", "--error", "--text", message, "--title", title)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

func sendDesktopNotificationInfo(title, message string) error {
	cmd := exec.Command("zenity", "--info", "--text", message, "--title", title)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send notification: %w", err)
	}
	return nil
}

func openFileSelector(title, message, fileName string) (string, error) {
	cmd := exec.Command(
		"zenity",
		"--file-selection",
		"--save",
		"--confirm-overwrite",
		"--title", title,
		"--text", message,
		"--filename", fileName,
	)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to open file selector: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
