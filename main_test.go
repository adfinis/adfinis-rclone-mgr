package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmd_UsageAndShort(t *testing.T) {
	assert.Equal(t, "adfinis-rclone-mgr", rootCmd.Use)
	assert.Contains(t, rootCmd.Short, "Google Drive")
	assert.Equal(t, Version, rootCmd.Version)
}

func TestRootCmd_HasSubcommands(t *testing.T) {
	subNames := []string{}
	for _, c := range rootCmd.Commands() {
		subNames = append(subNames, c.Name())
	}
	expected := []string{
		"gdrive-config",
		"mount",
		"umount",
		"ls",
		"journald-reader",
		"version",
		"man",
	}
	for _, name := range expected {
		assert.Contains(t, subNames, name)
	}
}

func TestRootCmd_HelpOutput(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{})
	_ = rootCmd.Execute()
	out := buf.String()
	assert.Contains(t, out, "adfinis-rclone-mgr is a command line tool to manage rclone mounts for Google Drive")
	assert.Contains(t, out, "Usage:")
	assert.Contains(t, out, "Available Commands:")
}

func TestRootCmd_UnknownCommand(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"doesnotexist"})
	err := rootCmd.Execute()
	assert.Error(t, err)
	out := buf.String()
	assert.True(t, strings.Contains(out, "unknown command") || strings.Contains(out, "Usage:"))
}
