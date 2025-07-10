package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

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

	var filesCount int
	for _, src := range srcPaths {
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
	for _, src := range srcPaths {
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
			if isProgressString(line) {
				filesDone++
				percent := int(float64(filesDone) / float64(filesCount) * 100)
				if err := progressDialog.Value(percent); err != nil {
					log.Printf("Failed to update progress: %v", err)
				}
			}
		}
	}
	progressDialog.Close() // nolint:errcheck

	msg := "File(s) copied successfully"
	if op == "move" {
		msg = "File(s) moved successfully"
	}
	if err := zenity.Info(msg); err != nil {
		log.Println("Failed to show success dialog:", err)
	}
}

func isProgressString(s string) bool {
	return strings.Contains(s, "Moved (server-side)") ||
		strings.Contains(s, "Copied (server-side copy)") ||
		strings.Contains(s, "Copied (server side copy)") // old rclone versions
}

func duplicateFiles(sources []string) {
	absGoogleRoot, err := filepath.Abs(getGooglePath())
	if err != nil {
		showZenityError("Failed to resolve Google Drive root")
		return
	}

	// Count files for progress tracking
	var filesCount int
	for _, src := range sources {
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
	for _, src := range sources {
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
