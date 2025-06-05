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
   - **Arch Linux (AUR)**: `yay -S adfinis-rclone-mgr-bin`.
3. Stop nautilus to make sure the new extension gets picked up: `nautilus -q`

### Manual Installation
1. Install the dependencies
   ```bash
   # debian / ubuntu
   sudo apt install rclone python3-nautilus xclip zenity

   # fedora / rhel
   sudo dnf install rclone nautilus-python xclip zenity

   # arch
   sudo pacman -S rclone python-nautilus xclip zenity
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
> `adfinis-rclone-mgr` assumes you have your own client_id and client_secret.  
> Make sure to [create that](https://rclone.org/drive/#making-your-own-client-id) or ask your Google Workspace Admin to provide one.

1. Start the application:
   1. Via Desktop: Just search for `adfinis-rclone-mgr`
   2. Via Terminal:
   ```bash
   adfinis-rclone-mgr gdrive-config
   ```
2. Open the provided URL in your browser to configure Google Drive mounts.
3. Follow the on-screen instructions to log in, select drives, and generate configurations.
4. Use the Nautilus context menu to open files directly in Google Drive or to copy files and folders between Google Drives using the special "Copy on Google Drive" action.

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

- **Copy files or folders between Google Drives (server-side, no conversion):**
  ```bash
  adfinis-rclone-mgr cp <source>... <destination>
  ```
  This command allows you to copy files or folders from one Google Drive to another, or within the same drive, using rclone's server-side copy. This avoids any file format conversion and is the recommended way to copy Google Docs, Sheets, and Slides natively.

  > **Why not use normal copy?**
  >
  > If you use the standard copy methods (Ctrl+C, right-click + Copy, or drag & drop) in your file manager, rclone mounts will convert Google Docs, Sheets, and Slides to Microsoft Office formats (e.g., gdocs to .docx) during the copy. This can cause formatting issues. The `cp` command and the "Copy on Google Drive" Nautilus context menu entry ensure that all copy actions are performed server-side, preserving the native Google format and avoiding unwanted conversions.

### Daemon Mode

The `daemon` command replaces the old journald command. It runs a background process that reads logs from the systemd journal and handles IPC events for  features like server-side copy.

```bash
adfinis-rclone-mgr daemon <drive-name>
```

This is started automatically for each mounted drive.

## 🐞 Troubleshooting
If there are still pending io operations on a share, or if you have a Drive folder open in your file manager, unmounting a share might fail.  
In that case, make sure to close all open files and file manager windows and execute `adfinis-rclone-mgr umount <share-name> --force` or `fusermount -u ~/google/<share-name>`.

## 🧪 Development
Make sure to install all dependencies:
```
go mod tidy
```

If you make any changes to a `.templ` file, don't forget to generate a native go code based on the template:

```
go tool templ generate
```

To create a release, simply push a tag and the pipeline will do the rest.  
[Semantic versioning](https://semver.org/) and [conventional commits](https://www.conventionalcommits.org/en/v1.0.0/) are a must!

```
git tag vX.Y.Z
git push --tags
```

## 📜 License
This project is licensed under the [GNU General Public License v3.0](./LICENSE).  
You are free to use, modify, and distribute this software under the terms of the license.

---

Made with ❤️ by Adfinis.
