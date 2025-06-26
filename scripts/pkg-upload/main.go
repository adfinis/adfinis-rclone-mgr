package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// PkgBotConfig represents the structure of .pkg-bot.yml
type PkgBotConfig struct {
	OnlyTest bool `yaml:"onlytest"`
	PkgBot   struct {
		Repo     string                         `yaml:"repo"`
		Branches []string                       `yaml:"branches"`
		Stages   []string                       `yaml:"stages"`
		Packages map[string]map[string][]string `yaml:"packages"`
	} `yaml:"pkgbot"`
}

var (
	repoMap = map[string]string{
		"adsy-public":  "public",
		"adsy-private": "private",
	}

	distMap = map[string][]string{
		"rhel6":    {"el/6"},
		"rhel7":    {"el/7"},
		"rhel8":    {"el/8"},
		"rhel9":    {"el/9"},
		"centos6":  {"el/6"},
		"centos7":  {"el/7"},
		"centos8":  {"el/8"},
		"precise":  {"ubuntu/precise"},
		"trusty":   {"ubuntu/trusty"},
		"vivid":    {"ubuntu/vivid"},
		"xenial":   {"ubuntu/xenial"},
		"bionic":   {"ubuntu/bionic"},
		"focal":    {"ubuntu/focal"},
		"jammy":    {"ubuntu/jammy"},
		"wheezy":   {"debian/wheezy"},
		"jessie":   {"debian/jessie"},
		"stretch":  {"debian/stretch"},
		"buster":   {"debian/buster"},
		"bullseye": {"debian/bullseye"},
		"bookworm": {"debian/bookworm"},
		"sles15":   {"sles/15.3", "sles/15.4", "sles/15.5", "sles/15.6"},
		"sles12":   {"sles/15.3", "sles/15.4", "sles/15.5", "sles/15.6"},
		"15":       {"sles/15.3", "sles/15.4", "sles/15.5", "sles/15.6"},
	}
)

func main() {
	uploadUser := os.Getenv("UPLOAD_USER")
	if uploadUser == "" {
		fmt.Fprintf(os.Stderr, "Missing UPLOAD_USER env var\n")
		os.Exit(1)
	}

	uploadSecret := os.Getenv("UPLOAD_SECRET")
	if uploadSecret == "" {
		fmt.Fprintf(os.Stderr, "Missing UPLOAD_SECRET env var\n")
		os.Exit(1)
	}

	pkgBotFile, err := searchPkgBotConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error finding .pkg-bot.yml: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Using .pkg-bot.yml at %s\n", pkgBotFile)

	configData, err := os.ReadFile(pkgBotFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading .pkg-bot.yml: %v\n", err)
		os.Exit(1)
	}

	var config PkgBotConfig
	if err := yaml.Unmarshal(configData, &config); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing .pkg-bot.yml: %v\n", err)
		os.Exit(1)
	}

	var uploadURL string
	if config.OnlyTest {
		fmt.Println("Only testing. Thus not using prod")
		uploadURL = "https://aptly-test.adfinis.com/upload"
	} else {
		uploadURL = "https://aptly.adfinis.com/upload"
	}

	repo, exists := repoMap[config.PkgBot.Repo]
	if !exists {
		fmt.Fprintf(os.Stderr, "Unknown repository: %s\n", config.PkgBot.Repo)
		os.Exit(1)
	}

	uploadedPackages := make(map[string][]string)
	for _, targets := range distMap {
		for _, target := range targets {
			uploadedPackages[target] = make([]string, 0)
		}
	}

	for _, packages := range config.PkgBot.Packages {
		for dist, wildcards := range packages {
			targets, exists := distMap[dist]
			if !exists {
				fmt.Fprintf(os.Stderr, "Unknown distribution: %s\n", dist)
				continue
			}

			for _, wildcard := range wildcards {
				files, err := filepath.Glob(wildcard)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error globbing %s: %v\n", wildcard, err)
					continue
				}

				for _, target := range targets {
					for _, file := range files {
						if !slices.Contains(uploadedPackages[target], file) {
							if err := uploadFile(file, uploadURL, repo, target, uploadUser, uploadSecret); err != nil {
								fmt.Fprintf(os.Stderr, "Error uploading %s: %v\n", file, err)
								continue
							}
							uploadedPackages[target] = append(uploadedPackages[target], file)
							fmt.Printf("- Uploaded %s to %s\n", file, target)
						} else {
							fmt.Fprintf(os.Stderr, "! duplicate %s for %s. This means the .pkgbot.yml contains e.g. centos8 AND rhel8 which is now the same\n", file, dist)
						}
					}
				}
			}
		}
	}
}

// uploadFile uploads a file to the specified URL using HTTP PUT with basic auth
func uploadFile(filePath, uploadURL, repo, target, username, password string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileName := filepath.Base(filePath)
	url := fmt.Sprintf("%s/%s/%s/%s", uploadURL, repo, target, fileName)

	req, err := http.NewRequest("PUT", url, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.SetBasicAuth(username, password)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// searchPkgBotConfig searches for a pkg-bot.yml file in the current directory and all parent directories
func searchPkgBotConfig() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	for {
		configPath := filepath.Join(currentDir, ".pkg-bot.yml")
		if _, err := os.Stat(configPath); err == nil {
			return configPath, nil
		}
		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir { // Reached root directory
			break
		}
		currentDir = parentDir
	}

	return "", fmt.Errorf(".pkg-bot.yml not found in any parent directories")
}
