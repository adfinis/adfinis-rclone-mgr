package main

import (
	"fmt"
	"log"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version is the current version of adfinis-rclone-mgr.
	Version = "devel"
	// Commit is the git commit hash of the current version.
	Commit = "none"
	// Date is the build date of the current version.
	Date = "unknown"
	// BuiltBy is the user who built the current version.
	BuiltBy = "unknown"
)

var rootCmd = &cobra.Command{
	Use:   "adfinis-rclone-mgr",
	Short: "Manage rclone mounts for Google Drive like a champ",
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			log.Fatalln("Failed to show help:", err)
		}
	},
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	Version: Version,
}

func init() {
	rootCmd.AddCommand(
		gdriveConfigCmd,
		mountCmd,
		umountCmd,
		listCmd,
		journaldReaderCmd,
		versionCmd,
	)
}

var gdriveConfigCmd = &cobra.Command{
	Use:   "gdrive-config",
	Short: "Generate an rclone config for Google Drive",
	Run:   gdriveConfig,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a drive",
	Long: "The mount command lets you mount one or more drives by starting the corresponding systemd service.\n" +
		"Use 'mount all' to mount all drives at once.\n" +
		"Use 'mount <drive>' to mount a specific drive.\n" +
		"Use 'mount <drive1> <drive2>' to mount multiple drives at once.\n" +
		"You can use tab completion to see all available drives.\n",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: availableMountsForArgs,
	Run:               mount,
}

var umountCmd = &cobra.Command{
	Use:   "umount",
	Short: "Unmount a drive",
	Long: "The umount command lets you umount one or more drives by stopping the corresponding systemd service.\n" +
		"Use 'umount all' to umount all drives at once.\n" +
		"Use 'umount <drive>' to umount a specific drive.\n" +
		"Use 'umount <drive1> <drive2>' to umount multiple drives at once.\n" +
		"You can use tab completion to see all available drives.\n",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: availableMountsForArgs,
	Run:               umount,
}

var listCmdFlags struct {
	JSON bool
	YAML bool
}

func init() {
	listCmd.Flags().BoolVarP(&listCmdFlags.JSON, "json", "j", false, "Output in JSON format")
	listCmd.Flags().BoolVarP(&listCmdFlags.YAML, "yaml", "y", false, "Output in YAML format")

	listCmd.MarkFlagsMutuallyExclusive("json", "yaml")
}

var listCmd = &cobra.Command{
	Use:   "ls",
	Short: "List all available mounts and their status",
	Run:   list,
}

var journaldReaderCmd = &cobra.Command{
	Use:   "journald-reader",
	Short: "Read logs from systemd journal and send desktop notifications based on them",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: availableMountsForArgs,
	Run:               journaldReader,
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

func Execute() error {
	return rootCmd.Execute()
}

func main() {
	if err := Execute(); err != nil {
		os.Exit(1)
	}
}
