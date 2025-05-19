# Mounting Google Drive with Rclone  
> The Adfinis way 🧙✨  

[![Go Report Card](https://goreportcard.com/badge/github.com/adfinis/adfinis-rclone-mgr)](https://goreportcard.com/report/github.com/adfinis/adfinis-rclone-mgr)
[![golangci-lint](https://github.com/adfinis/adfinis-rclone-mgr/actions/workflows/lint.yml/badge.svg)](https://github.com/adfinis/adfinis-rclone-mgr/actions/workflows/lint.yml)
[![testing](https://github.com/adfinis/adfinis-rclone-mgr/actions/workflows/test.yml/badge.svg)](https://github.com/adfinis/adfinis-rclone-mgr/actions/workflows/test.yml)
[![goreleaser](https://github.com/adfinis/adfinis-rclone-mgr/actions/workflows/release.yml/badge.svg)](https://github.com/adfinis/adfinis-rclone-mgr/actions/workflows/release.yml)

## 🚀 About the Project
This repository provides a streamlined way to mount Google Drive using Rclone, tailored for Adfinis workflows. It includes:
- A web-based configuration tool for Google Drive mounts.
- Systemd service templates for managing Rclone mounts.
- Nautilus integration for opening files directly in Google Drive and copying shareable links.
- CLI Tool to mount and umount shares.
- Better error handling in case of permission errors.

## 📦 Installation

### Using GoReleaser Packages
1. Download the latest release from the [Releases](https://github.com/adfinis/adfinis-rclone-mgr/releases) page.
2. Install the package for your distribution:
   - **Debian/Ubuntu**: `sudo apt install adfinis-rclone-mgr-<version>.deb`
   - **Fedora/RHEL**: `sudo dnf install adfinis-rclone-mgr-<version>.rpm`
   - **Arch Linux**: `sudo pacman -U adfinis-rclone-mgr-<version>.pkg.tar.zst`.
3. Stop nautilus to make sure the new extension gets picked up: `nautilus -q`

### Manual Installation
1. Install the dependencies
   ```bash
   # debian / ubuntu
   sudo apt install rclone python3-nautilus xclip

   # fedora / rhel
   sudo dnf install rclone nautilus-python xclip

   # arch
   sudo pacman -S rclone python-nautilus xclip
   ```
2. Clone the repository:
   ```bash
   git clone https://github.com/adfinis/adfinis-rclone-mgr.git
   cd adfinis-rclone-mgr
   ```
3. Build the binary:
   ```bash
   go build -o adfinis-rclone-mgr .
   ```
4. Install the assets:
   ```bash
   sudo cp assets/rclone@.service /usr/lib/systemd/user/
   sudo cp assets/adfinis-rclone-mgr@.service /usr/lib/systemd/user/
   sudo cp assets/google_drive_opener.py /usr/share/nautilus-python/extensions/
   sudo cp assets/adfinis-rclone-mgr.desktop /usr/share/applications/
   sudo cp assets/adfinis-rclone-mgr.png /usr/share/icons/hicolor/512x512/apps/
   ```
5. Optional: Autocompletion  
   ```
   ./adfinis-rclone-mgr completion --help
   ```

## 🛠️ Usage
1. Start the application:
   1. Via Desktop: Just search for `adfinis-rclone-mgr`
   2. Via Terminal:
   ```bash
   adfinis-rclone-mgr gdrive-config
   ```
2. Open the provided URL in your browser to configure Google Drive mounts.
3. Follow the on-screen instructions to log in, select drives, and generate configurations.
4. Use the Nautilus context menu to open files directly in Google Drive.

### Managing Mounts

You can now manage your Google Drive mounts directly from the terminal using the following commands:

- **List available shares:**
  ```bash
  adfinis-rclone-mgr ls
  ```

- **Mount a configured share:**
  ```bash
  adfinis-rclone-mgr mount <share-name>
  ```
  Replace `<share-name>` with the name of your configured Google Drive share.  
  You can use tab for autocompletion of the share names.

- **Unmount a share:**
  ```bash
  adfinis-rclone-mgr umount <share-name>
  ```
  This will safely unmount the specified share.

These commands allow you to quickly mount or unmount your Google Drive shares as needed.

## 📜 License
This project is licensed under the [GNU General Public License v3.0](./LICENSE).  
You are free to use, modify, and distribute this software under the terms of the license.

---

Made with ❤️ by Adfinis.
