package main

import (
	"log"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
)

var mountCmd = &cobra.Command{
	Use:   "mount",
	Short: "Mount a drive",
	CompletionOptions: cobra.CompletionOptions{
		DisableDefaultCmd: true,
	},
	Args: cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return append([]string{"all"}, getRemotes()...), cobra.ShellCompDirectiveDefault
	},
	Run: mount,
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
	Args: cobra.MinimumNArgs(1),
	ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return append([]string{"all"}, getRemotes()...), cobra.ShellCompDirectiveDefault
	},
	Run: umount,
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
