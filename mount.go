package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a drive",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: availableMountsForArgs,
	Run:               mount,
}

func availableMountsForArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if lo.Contains(args, "all") {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}
	// remove args from the list of available mounts
	remotes := getRemotes()
	wantedRemotes := make([]string, 0, len(remotes))
	for _, remote := range remotes {
		if !lo.Contains(args, remote) {
			wantedRemotes = append(wantedRemotes, remote)
		}
	}
	return getMountsWithStatus(cmd.Context(), wantedRemotes), cobra.ShellCompDirectiveNoFileComp
}

func getMountsWithStatus(ctx context.Context, remotes []string) []string {
	conn, err := dbus.NewUserConnectionContext(ctx)
	if err != nil {
		log.Fatalln("Failed to start dbus connection:", err)
	}
	statuses, err := statusServices(ctx, conn, remotes)
	if err != nil {
		log.Fatalln("Failed to get service status:", err)
	}

	mounts := make([]string, len(statuses)+1)
	mounts[0] = "all\tmount all drives"
	for i, status := range statuses {
		mounts[i+1] = fmt.Sprintf("%s\t%s", unitNameToDriveName(status.Name), status.ActiveState)
	}
	return mounts
}

func mount(cmd *cobra.Command, args []string) {
	conn, err := dbus.NewUserConnectionContext(cmd.Context())
	if err != nil {
		log.Fatalln("Failed to start dbus connection:", err)
	}
	if lo.Contains(args, "all") {
		args = getRemotes()
	}
	for _, arg := range args {
		if err := startService(cmd.Context(), conn, arg); err != nil {
			log.Printf("Failed to mount drive: %v", err)
			continue
		}
		log.Println("Mounted Drive:", arg)
	}
}

var umountCmd = &cobra.Command{
	Use:   "umount",
	Short: "Unmount a drive",

	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args:              cobra.MinimumNArgs(1),
	ValidArgsFunction: availableMountsForArgs,
	Run:               umount,
}

func umount(cmd *cobra.Command, args []string) {
	conn, err := dbus.NewUserConnectionContext(cmd.Context())
	if err != nil {
		log.Fatalln("Failed to start dbus connection:", err)
	}
	if lo.Contains(args, "all") {
		args = getRemotes()
	}
	for _, arg := range args {
		if err := stopService(cmd.Context(), conn, arg); err != nil {
			log.Printf("Failed to umount drive: %v", err)
			continue
		}
		log.Println("Umounted Drive:", arg)
	}
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
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Run: list,
}

func list(cmd *cobra.Command, _ []string) {
	conn, err := dbus.NewUserConnectionContext(cmd.Context())
	if err != nil {
		log.Fatalln("Failed to start dbus connection:", err)
	}
	statuses, err := statusServices(cmd.Context(), conn, getRemotes())
	if err != nil {
		log.Fatalln("Failed to get service status:", err)
	}

	if listCmdFlags.JSON {
		renderJSON(statuses)
	} else if listCmdFlags.YAML {
		renderYAML(statuses)
	} else {
		renderTable(statuses)
	}
}

func renderTable(statuses []dbus.UnitStatus) {
	rows := make([][]string, len(statuses))
	for i, status := range statuses {
		var prefix string
		if status.ActiveState == "active" {
			prefix = "✅"
		} else if status.ActiveState == "failed" {
			prefix = "❌"
		} else if status.ActiveState == "inactive" {
			prefix = "⬜"
		} else {
			prefix = "❓"
		}
		rows[i] = []string{
			prefix,
			status.Name,
			status.ActiveState,
			fmt.Sprintf("~/google/%s", unitNameToDriveName(status.Name)),
		}
	}

	re := lipgloss.NewRenderer(os.Stdout)

	cellStyle := re.NewStyle().Padding(0, 1)
	headerStyle := cellStyle.Bold(true).Align(lipgloss.Center)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(lipgloss.Color("2e4b98"))).
		StyleFunc(func(row, col int) lipgloss.Style {
			switch {
			case row == table.HeaderRow:
				return headerStyle
			default:
				return cellStyle
			}
		}).
		Headers("Ok?", "Name", "Status", "Mount Path").
		Rows(rows...)

	fmt.Println()
	fmt.Println(t)
	fmt.Println()
}

type serviceStatus struct {
	Name      string
	Status    string
	MountPath string
}

func statusesToServiceStatuses(statuses []dbus.UnitStatus) []serviceStatus {
	serviceStatuses := make([]serviceStatus, len(statuses))
	for i, status := range statuses {
		serviceStatuses[i] = serviceStatus{
			Name:      unitNameToDriveName(status.Name),
			Status:    status.ActiveState,
			MountPath: fmt.Sprintf("~/google/%s", unitNameToDriveName(status.Name)),
		}
	}
	return serviceStatuses
}

func renderJSON(statuses []dbus.UnitStatus) {
	s := statusesToServiceStatuses(statuses)
	jsonData, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		log.Fatalln("Failed to marshal JSON:", err)
	}
	fmt.Println(string(jsonData))
}

func renderYAML(statuses []dbus.UnitStatus) {
	s := statusesToServiceStatuses(statuses)
	yamlData, err := yaml.Marshal(s)
	if err != nil {
		log.Fatalln("Failed to marshal YAML:", err)
	}
	fmt.Println(string(yamlData))
}

func unitNameToDriveName(name string) string {
	name = strings.Split(name, "@")[1]
	name = strings.Split(name, ".service")[0]
	return name
}
