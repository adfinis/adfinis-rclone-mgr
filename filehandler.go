package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	clipboard "github.com/aymanbagabas/go-nativeclipboard"
	"github.com/ncruces/zenity"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func copy(cmd *cobra.Command, args []string) {
	// last arg is the destination directory
	dest := args[len(args)-1]
	runRcloneOp("copy", args[:len(args)-1], dest)
}

func move(cmd *cobra.Command, args []string) {
	// last arg is the destination directory
	dest := args[len(args)-1]
	runRcloneOp("move", args[:len(args)-1], dest)
}

// isSubdir checks if sub is a subdirectory (or the same) as root.
func isSubdir(root, sub string) (bool, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return false, err
	}
	absSub, err := filepath.Abs(sub)
	if err != nil {
		return false, err
	}
	rel, err := filepath.Rel(absRoot, absSub)
	if err != nil {
		return false, err
	}
	if strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return false, nil
	}
	return true, nil
}

// toRclonePath converts an absolute path under the Google root to rclone remote:path format.
func toRclonePath(root, abs string) (remote, subpath string, err error) {
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return "", "", err
	}
	split := strings.Split(rel, string(os.PathSeparator))
	if len(split) < 1 {
		return "", "", fmt.Errorf("invalid path: %q", abs)
	}
	remote = split[0]
	subpath = path.Join(strings.Join(split[1:], string(os.PathSeparator)))
	// assume something went horribly wrong if any of those start with ".."
	if strings.HasPrefix(subpath, "..") || strings.HasPrefix(remote, "..") {
		return "", "", fmt.Errorf("invalid path: %q", abs)
	}
	return remote, subpath, nil
}

// resolveActualFileName takes a file path that may end with .link.html and resolves it to the actual file name on google drive.
// This is needed because Google Drive native files are exported as .link.html locally but have different extensions when listed with rclone.
func resolveActualFileName(absGoogleRoot, filePath string) (string, error) {
	if !strings.HasSuffix(filePath, ".link.html") {
		return filePath, nil
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	baseName := strings.TrimSuffix(filepath.Base(absFilePath), ".link.html")
	dirPath := filepath.Dir(absFilePath)

	remote, subpath, err := toRclonePath(absGoogleRoot, dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to convert path: %w", err)
	}

	rclonePath := fmt.Sprintf("%s:%s", remote, subpath)
	cmd := exec.Command("rclone", "lsjson", rclonePath)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to list files with rclone lsjson %s: %w", rclonePath, err)
	}

	var files []struct {
		Path  string `json:"Path"`
		IsDir bool   `json:"IsDir"`
	}
	if err := json.Unmarshal(output, &files); err != nil {
		return "", fmt.Errorf("failed to parse rclone output: %w", err)
	}

	var matches []string
	for _, file := range files {
		if file.IsDir {
			continue
		}
		fileBase := strings.TrimSuffix(file.Path, filepath.Ext(file.Path))
		if fileBase == baseName {
			matches = append(matches, file.Path)
		}
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("file not found on remote: %s (searched in %s)", baseName, rclonePath)
	}

	if len(matches) > 1 {
		return "", fmt.Errorf("ambiguous file name: multiple files match '%s' - found: %v", baseName, matches)
	}

	return filepath.Join(dirPath, matches[0]), nil
}

// patchDestPath modifies the destination path to ensure the name of the source folder is added to the destination path.
// otherwise rclone would copy the files into the destination directory without creating a subdirectory.
func patchDestPath(src, dest string) string {
	if !strings.HasSuffix(dest, string(os.PathSeparator)) {
		dest += string(os.PathSeparator)
	}
	srcBase := filepath.Base(src)
	return filepath.Join(dest, srcBase)
}

func showZenityError(msg string) {
	log.Println("Error:", msg)
	if err := zenity.Error("msg"); err != nil {
		log.Println("Failed to show error dialog:", err)
	}
}

func selectDestination() (string, error) {
	zenityOut, err := zenity.SelectFile(zenity.Directory(), zenity.Title("Select destination directory for copy"))
	if err != nil {
		showZenityError("Copy cancelled or failed to select destination")
		return "", err
	}
	destDir := strings.TrimSpace(string(zenityOut))
	log.Println("Selected destination directory:", destDir)
	return destDir, nil
}

func selectDestAndRunOP(srcPaths []string, op string) {
	destDir, err := selectDestination()
	if err != nil {
		log.Println("Failed to select destination:", err)
		return
	}

	if destDir == "" {
		showZenityError("No destination directory selected")
		return
	}

	runRcloneOp(op, srcPaths, destDir)
}

// runRcloneOp runs rclone with the given operation ("copy" or "move")
func runRcloneOp(op string, srcPaths []string, destDir string) {
	absGoogleRoot, err := filepath.Abs(getGooglePath())
	if err != nil {
		showZenityError("Failed to resolve Google Drive root")
		return
	}
	absDestDir, err := filepath.Abs(destDir)
	if err != nil {
		showZenityError("Failed to resolve destination directory")
		return
	}
	// Check if absDestDir is under absGoogleRoot
	isSub, err := isSubdir(absGoogleRoot, absDestDir)
	if err != nil || !isSub {
		showZenityError("Destination must be inside your Google Drive mount")
		return
	}

	resolvedSrcPaths := make([]string, len(srcPaths))
	for i, src := range srcPaths {
		resolved, err := resolveActualFileName(absGoogleRoot, src)
		if err != nil {
			showZenityError(fmt.Sprintf("Failed to resolve file name for %s: %v", src, err))
			return
		}
		resolvedSrcPaths[i] = resolved
	}

	var filesCount int
	for _, src := range resolvedSrcPaths {
		if isDir(src) {
			if err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					filesCount++
				}
				return nil
			}); err != nil {
				showZenityError(fmt.Sprintf("Failed to count files in directory %s: %v", src, err))
				return
			}
		} else {
			filesCount++
		}
	}
	if filesCount == 0 {
		showZenityError("No source files provided")
		return
	}

	verb := op
	if op == "move" {
		verb = "mov"
	}

	caser := cases.Title(language.AmericanEnglish)

	progressDialog, err := zenity.Progress(
		zenity.Title(caser.String(op)+" on Google Drive"),
		zenity.AutoClose(),
	)
	if err != nil {
		showZenityError("Failed to start progress dialog")
		return
	}
	defer progressDialog.Close() // nolint:errcheck

	// Set initial text
	if err := progressDialog.Text(caser.String(verb) + "ing files..."); err != nil {
		showZenityError("Failed to set progress dialog text")
		return
	}

	var runningRcloneProc *exec.Cmd
	cancelled := make(chan struct{})
	go func() {
		defer close(cancelled)
		<-progressDialog.Done()
		log.Println("Progress dialog cancelled by user")
		if runningRcloneProc != nil {
			runningRcloneProc.Cancel() // nolint:errcheck
		}
	}()

	filesDone := 0
	var gotError bool
	for _, src := range resolvedSrcPaths {
		select {
		case <-cancelled:
			log.Println("Operation cancelled by user")
			return
		default:
		}

		isSub, err := isSubdir(absGoogleRoot, src)
		if err != nil || !isSub {
			showZenityError("Source must be inside your Google Drive mount")
			return
		}

		absSrc, err := filepath.Abs(src)
		if err != nil {
			showZenityError("Failed to resolve source path")
			return
		}

		srcDriveName, srcPath, err := toRclonePath(absGoogleRoot, absSrc)
		if err != nil {
			showZenityError("Failed to parse source path")
			return
		}
		srcRclone := fmt.Sprintf("%s:%s", srcDriveName, srcPath)
		destDriveName, destPath, err := toRclonePath(absGoogleRoot, absDestDir)
		if err != nil {
			showZenityError("Failed to parse destination path")
			return
		}
		destRclone := fmt.Sprintf("%s:%s", destDriveName, destPath)

		if isDir(src) {
			destRclone = patchDestPath(srcPath, destRclone)
		}
		log.Printf("%sing from %s to %s", verb, srcRclone, destRclone)

		cmd := exec.CommandContext(context.Background(), "rclone", op, "--drive-server-side-across-configs", srcRclone, destRclone, "-v")
		runningRcloneProc = cmd
		cmd.Stdout = os.Stdout
		// rclone -v writes to stderr
		stderr, err := cmd.StderrPipe()
		if err != nil {
			showZenityError(fmt.Sprintf("Failed to create stderr pipe for rclone %s: %v", op, err))
			return
		}
		if err := cmd.Start(); err != nil {
			showZenityError(fmt.Sprintf("rclone %s failed: %v", op, err))
			return
		}

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if strings.Contains(line, "Failed") {
				gotError = true
			}
			if isProgressString(line) {
				filesDone++
				percent := int(float64(filesDone) / float64(filesCount) * 100)
				if err := progressDialog.Value(percent); err != nil {
					log.Printf("Failed to update progress: %v", err)
				}
			}
		}

		if err := cmd.Wait(); err != nil {
			showZenityError(fmt.Sprintf("rclone %s failed: %v", op, err))
			return
		}

		if cmd.ProcessState.ExitCode() != 0 {
			gotError = true
		}
	}
	progressDialog.Close() // nolint:errcheck
	if gotError {
		showZenityError(fmt.Sprintf("One or more files failed to %s. Check the console for details.", op))
	} else {
		msg := "File(s) copied successfully"
		if op == "move" {
			msg = "File(s) moved successfully"
		}
		if err := zenity.Info(msg); err != nil {
			log.Println("Failed to show success dialog:", err)
		}
	}
}

func isProgressString(s string) bool {
	return strings.Contains(s, "Moved (server-side)") ||
		strings.Contains(s, "Copied (server-side copy)") ||
		strings.Contains(s, "Copied (server side copy)") // old rclone versions
}

type rcloneFileMetadata struct {
	Path     string `json:"Path"`
	IsDir    bool   `json:"IsDir"`
	ID       string `json:"ID"`
	MimeType string `json:"MimeType"`
}

// getRcloneFileMetadata resolves a file path (including .link.html files) and returns its metadata from rclone.
func getRcloneFileMetadata(absGoogleRoot, filePath string) (*rcloneFileMetadata, error) {
	// Convert to absolute path first
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Check if file is under Google Drive mount
	isSub, err := isSubdir(absGoogleRoot, absFilePath)
	if err != nil || !isSub {
		return nil, fmt.Errorf("file must be inside your Google Drive mount")
	}

	baseName := filepath.Base(absFilePath)
	isLinkHTML := strings.HasSuffix(baseName, ".link.html")
	if isLinkHTML {
		baseName = strings.TrimSuffix(baseName, ".link.html")
	}

	dirPath := filepath.Dir(absFilePath)
	remote, subpath, err := toRclonePath(absGoogleRoot, dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to convert path: %w", err)
	}

	rclonePath := fmt.Sprintf("%s:%s", remote, subpath)
	cmd := exec.Command("rclone", "lsjson", rclonePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list files with rclone lsjson %s: %w", rclonePath, err)
	}

	var files []rcloneFileMetadata
	if err := json.Unmarshal(output, &files); err != nil {
		return nil, fmt.Errorf("failed to parse rclone output: %w", err)
	}

	if isLinkHTML {
		// Find files that match the base name (without extension)
		var matches []rcloneFileMetadata
		for _, file := range files {
			if file.IsDir {
				continue
			}
			fileBase := strings.TrimSuffix(file.Path, filepath.Ext(file.Path))
			if fileBase == baseName {
				matches = append(matches, file)
			}
		}

		if len(matches) == 0 {
			return nil, fmt.Errorf("file not found on remote: %s (searched in %s)", baseName, rclonePath)
		}

		if len(matches) > 1 {
			matchNames := make([]string, len(matches))
			for i, m := range matches {
				matchNames[i] = m.Path
			}
			return nil, fmt.Errorf("ambiguous file name: multiple files match '%s' - found: %v", baseName, matchNames)
		}

		return &matches[0], nil
	}

	// For non-.link.html files, find exact match
	for _, file := range files {
		if file.Path == baseName {
			return &file, nil
		}
	}

	return nil, fmt.Errorf("file '%s' not found in Google Drive", baseName)
}

// getGoogleDriveLink returns the Google Drive web link for a file.
func getGoogleDriveLink(absGoogleRoot, filePath string) (string, error) {
	metadata, err := getRcloneFileMetadata(absGoogleRoot, filePath)
	if err != nil {
		return "", err
	}

	if metadata.ID == "" {
		return "", fmt.Errorf("file has no ID")
	}

	return fmt.Sprintf("https://drive.google.com/open?id=%s", metadata.ID), nil
}

// printLinks prints Google Drive links for the given files.
func printLinks(cmd *cobra.Command, args []string) {
	absGoogleRoot, err := filepath.Abs(getGooglePath())
	if err != nil {
		log.Fatalf("Failed to resolve Google Drive root: %v", err)
	}

	for _, filePath := range args {
		link, err := getGoogleDriveLink(absGoogleRoot, filePath)
		if err != nil {
			log.Printf("Error getting link for %s: %v", filePath, err)
			continue
		}
		fmt.Println(link)
	}
}

// openInBrowser opens files in Google Drive web interface.
func openInBrowser(filePaths []string) {
	absGoogleRoot, err := filepath.Abs(getGooglePath())
	if err != nil {
		showZenityError("Failed to resolve Google Drive root")
		return
	}

	for _, filePath := range filePaths {
		metadata, err := getRcloneFileMetadata(absGoogleRoot, filePath)
		if err != nil {
			showZenityError(fmt.Sprintf("Failed to get file metadata for %s: %v", filePath, err))
			continue
		}

		// Show warning for OpenDocument formats
		openDocFormats := []string{
			"application/vnd.oasis.opendocument.text",
			"application/vnd.oasis.opendocument.spreadsheet",
			"application/vnd.oasis.opendocument.presentation",
		}
		for _, format := range openDocFormats {
			if metadata.MimeType == format {
				if err := zenity.Warning(
					"You are about to open an Open Document Format file.\n\nOpening this with Google Docs will create a copy of the file!",
				); err != nil {
					log.Printf("Failed to show warning dialog: %v", err)
				}
				break
			}
		}

		url := fmt.Sprintf("https://drive.google.com/open?id=%s", metadata.ID)
		cmd := exec.Command("xdg-open", url)
		if err := cmd.Start(); err != nil {
			showZenityError(fmt.Sprintf("Failed to open browser: %v", err))
		}
	}
}

// copyLinkToClipboard copies the Google Drive link to the clipboard.
func copyLinkToClipboard(filePaths []string) {
	if len(filePaths) == 0 {
		showZenityError("No files provided")
		return
	}

	absGoogleRoot, err := filepath.Abs(getGooglePath())
	if err != nil {
		showZenityError("Failed to resolve Google Drive root")
		return
	}

	link, err := getGoogleDriveLink(absGoogleRoot, filePaths[0])
	if err != nil {
		showZenityError(fmt.Sprintf("Failed to get link: %v", err))
		return
	}

	// Copy to clipboard using go-nativeclipboard
	if _, err := clipboard.Text.Write([]byte(link)); err != nil {
		log.Printf("Failed to copy to clipboard: %v", err)
		showZenityError(fmt.Sprintf("Failed to copy link to clipboard: %v", err))
	}
}

func duplicateFiles(sources []string) {
	absGoogleRoot, err := filepath.Abs(getGooglePath())
	if err != nil {
		showZenityError("Failed to resolve Google Drive root")
		return
	}

	resolvedSources := make([]string, len(sources))
	for i, src := range sources {
		resolved, err := resolveActualFileName(absGoogleRoot, src)
		if err != nil {
			showZenityError(fmt.Sprintf("Failed to resolve file name for %s: %v", src, err))
			return
		}
		resolvedSources[i] = resolved
	}

	// Count files for progress tracking
	var filesCount int
	for _, src := range resolvedSources {
		if isDir(src) {
			if err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
				if err == nil && !d.IsDir() {
					filesCount++
				}
				return nil
			}); err != nil {
				showZenityError(fmt.Sprintf("Failed to count files in directory %s: %v", src, err))
				return
			}
		} else {
			filesCount++
		}
	}
	if filesCount == 0 {
		showZenityError("No source files provided")
		return
	}

	progressDialog, err := zenity.Progress(
		zenity.Title("Duplicate on Google Drive"),
		zenity.AutoClose(),
	)
	if err != nil {
		showZenityError("Failed to start progress dialog")
		return
	}
	defer progressDialog.Close() // nolint:errcheck

	// Set initial text
	if err := progressDialog.Text("Duplicating files..."); err != nil {
		showZenityError("Failed to set progress dialog text")
		return
	}

	var runningRcloneProc *exec.Cmd
	cancelled := make(chan struct{})
	go func() {
		defer close(cancelled)
		<-progressDialog.Done()
		log.Println("Progress dialog cancelled by user")
		if runningRcloneProc != nil {
			runningRcloneProc.Cancel() // nolint:errcheck
		}
	}()

	filesDone := 0
	for _, src := range resolvedSources {
		select {
		case <-cancelled:
			log.Println("Operation cancelled by user")
			return
		default:
		}

		// Verify source is under Google Drive mount
		isSub, err := isSubdir(absGoogleRoot, src)
		if err != nil || !isSub {
			showZenityError("Source must be inside your Google Drive mount")
			return
		}

		absSrc, err := filepath.Abs(src)
		if err != nil {
			showZenityError("Failed to resolve source path")
			return
		}

		srcDriveName, srcPath, err := toRclonePath(absGoogleRoot, absSrc)
		if err != nil {
			showZenityError("Failed to parse source path")
			return
		}

		// Generate destination path with "copy_of_" prefix
		var destPath string
		if isDir(src) {
			// For directories, add "copy_of_" prefix to the directory name
			baseName := filepath.Base(srcPath)
			destPath = filepath.Join(filepath.Dir(srcPath), "copy_of_"+baseName)
		} else {
			// For files, add "copy_of_" prefix while preserving extension
			ext := filepath.Ext(srcPath)
			baseName := strings.TrimSuffix(filepath.Base(srcPath), ext)
			destPath = filepath.Join(filepath.Dir(srcPath), "copy_of_"+baseName+ext)
		}

		srcRclone := fmt.Sprintf("%s:%s", srcDriveName, srcPath)
		destRclone := fmt.Sprintf("%s:%s", srcDriveName, destPath)

		log.Printf("Duplicating from %s to %s", srcRclone, destRclone)

		// Use copyto for individual file duplication, copy for directories
		var cmd *exec.Cmd
		if isDir(src) {
			cmd = exec.CommandContext(context.Background(), "rclone", "copy", "--drive-server-side-across-configs", srcRclone, destRclone, "-v")
		} else {
			cmd = exec.CommandContext(context.Background(), "rclone", "copyto", "--drive-server-side-across-configs", srcRclone, destRclone, "-v")
		}

		runningRcloneProc = cmd
		cmd.Stdout = os.Stdout
		// rclone -v writes to stderr
		stderr, err := cmd.StderrPipe()
		if err != nil {
			showZenityError(fmt.Sprintf("Failed to create stderr pipe for rclone duplicate: %v", err))
			return
		}
		if err := cmd.Start(); err != nil {
			showZenityError(fmt.Sprintf("rclone duplicate failed: %v", err))
			return
		}

		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println(line)
			if isProgressString(line) {
				filesDone++
				percent := int(float64(filesDone) / float64(filesCount) * 100)
				if err := progressDialog.Value(percent); err != nil {
					log.Printf("Failed to update progress: %v", err)
				}
			}
		}

		// Wait for the command to complete
		if err := cmd.Wait(); err != nil {
			showZenityError(fmt.Sprintf("rclone duplicate failed: %v", err))
			return
		}
	}

	if err := zenity.Info("File(s) duplicated successfully"); err != nil {
		log.Println("Failed to show success dialog:", err)
	}
}
