# Mounting Google Drive with Rclone  
> The Adfinis way 🧙✨  

## 🚀 About the Project
This repository provides a streamlined way to mount Google Drive using Rclone, tailored for Adfinis workflows. It includes:
- A web-based configuration tool for Google Drive mounts.
- Systemd service templates for managing Rclone mounts.
- Nautilus integration for opening files directly in Google Drive.

## 📦 Installation

### Using GoReleaser Packages
1. Download the latest release from the [Releases](https://github.com/adfinis/adfinis-rclone-mount/releases) page.
2. Install the package for your distribution:
   - **Debian/Ubuntu**: `sudo apt install adfinis-rclone-mount-<version>.deb`
   - **Fedora/RHEL**: `sudo dnf install adfinis-rclone-mount-<version>.rpm`
   - **Arch Linux**: Use the `.pkg.tar.zst` file with `pacman -U`.

### Manual Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/adfinis/adfinis-rclone-mount.git
   cd adfinis-rclone-mount
   ```
2. Build the binary:
   ```bash
   go build -o adfinis-rclone-mount .
   ```
3. Install the assets:
   ```bash
   sudo cp assets/rclone@.service /usr/lib/systemd/user/
   sudo cp assets/google_drive_opener.py /usr/share/nautilus-python/extensions/
   sudo cp assets/adfinis-rclone-mount.desktop /usr/share/applications/
   sudo cp assets/adfinis-rclone-mount.png /usr/share/icons/hicolor/512x512/apps/
   ```

## 🛠️ Usage
1. Start the application:
   1. Via Desktop: Just search for `adfinis-rclone-mount`
   2. Via Terminal:
   ```bash
   ./adfinis-rclone-mount
   ```
2. Open the provided URL in your browser to configure Google Drive mounts.
3. Follow the on-screen instructions to log in, select drives, and generate configurations.
4. Use the Nautilus context menu to open files directly in Google Drive.

## 📜 License
This project is licensed under the [GNU General Public License v3.0](./LICENSE).  
You are free to use, modify, and distribute this software under the terms of the license.

---

Made with ❤️ by Adfinis.
