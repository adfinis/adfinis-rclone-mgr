package main

import (
	"fmt"
	"log"
	"os"

	mcobra "github.com/muesli/mango-cobra"
	"github.com/muesli/roff"
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
	Long: "adfinis-rclone-mgr is a command line tool to manage rclone mounts for Google Drive.\n" +
		"It provides commands to mount, unmount, list, and configure Google Drive mounts.\n" +
		"It also provides a daemon to read error logs from systemd journal and send desktop notifications based on them.",
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
		daemonCmd,
		copyCmd,
		versionCmd,
		manCmd,
	)
}

var gdriveConfigCmd = &cobra.Command{
	Use:   "gdrive-config",
	Short: "Generate an rclone config for Google Drive",
	Long: "The gdrive-config command generates an rclone config for Google Drive.\n" +
		"It will open a browser window to authenticate with Google Drive.\n" +
		"After authentication, it will generate a config file for rclone.\n" +
		"The config file will be saved in the default location for rclone configs (~/.config/rclone/rclone.conf).\n" +
		"Existing rclone remotes won't be overwritten unless the name conflicts with the name of a Google Drive.\n",
	Run: gdriveConfig,
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

var umountCmdFlags struct {
	Force bool
}

func init() {
	umountCmd.Flags().BoolVarP(&umountCmdFlags.Force, "force", "f", false, "Force unmount the drive(s)")
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

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Daemon to read logs from systemd journal and handle ipc events",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args:              cobra.ExactArgs(1),
	ValidArgsFunction: availableMountsForArgs,
	Run:               daemon,
}

var copyCmd = &cobra.Command{
	Use:   "cp",
	Short: "Copy files or folders from one drive to another",
	Long: "The cp command allows you to copy files or folders from one Google Drive to another.\n" +
		"It supports copying single files, multiple files, or entire folders.\n" +
		"You can use it as a drop-in replacement for the linux cp command, but with the added benefit of working across Google Drives.\n",
	Args: cobra.MinimumNArgs(2),
	Run:  copy,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version details",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("Version: %s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Date: %s\n", Date)
		fmt.Printf("BuiltBy: %s\n", BuiltBy)
	},
}

var manCmd = &cobra.Command{
	Use:                   "man",
	Short:                 "generates the manpages",
	SilenceUsage:          true,
	DisableFlagsInUseLine: true,
	Hidden:                true,
	Args:                  cobra.NoArgs,
	RunE: func(_ *cobra.Command, _ []string) error {
		manPage, err := mcobra.NewManPage(1, rootCmd)
		if err != nil {
			return err
		}
		_, err = fmt.Fprint(os.Stdout, manPage.Build(roff.NewDocument()))
		return err
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
