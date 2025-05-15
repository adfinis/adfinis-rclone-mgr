from gi.repository import Nautilus, GObject
import os
import subprocess
import webbrowser
import json

"""
This extension adds a context menu item to Nautilus for opening files in Google Drive.
It generates a public link using rclone and opens it in the default web browser.

The extension expects rclone to mount the drives at ~/google/$drive_name.
"""

class GoogleDriveOpener(GObject.GObject, Nautilus.MenuProvider):
    RCLONE_MOUNT_PATH = os.path.expanduser("~/google")

    OPENDOCUMENT_FORMATS = [
        "application/vnd.oasis.opendocument.text",
        "application/vnd.oasis.opendocument.spreadsheet",
        "application/vnd.oasis.opendocument.presentation",
    ]

    def get_file_items(self, files):
        file_paths = []
        for file in files:
            file_path = file.get_location().get_path()
            if not os.path.abspath(file_path).startswith(self.RCLONE_MOUNT_PATH):
                return
            file_paths.append(file_path)

        item = Nautilus.MenuItem(
            name="GoogleDriveOpener::OpenPublicURL",
            label="Open in Google Drive",
            tip="Open the file in Google Drive",
        )
        item.connect("activate", self.open_rclone_url, file_paths)
        return [item]

    def open_rclone_url(self, menu, file_paths):
        for file_path in file_paths:
            try:
                relative_path = os.path.relpath(file_path, self.RCLONE_MOUNT_PATH)
                drive_name = relative_path.split(os.sep)[0]
                # remove drive_name and the last part of the path
                file_path = os.path.join("", *relative_path.split(os.sep)[1:-1])
                file_name = os.path.basename(relative_path)

                cmd = ['rclone', 'lsjson', f'{drive_name}:{file_path}']
                result = subprocess.run(cmd, capture_output=True, text=True, check=True)
                files = json.loads(result.stdout)

                if not files:
                    raise FileNotFoundError("No files returned from rclone lsjson")

                # find matching file by name
                files = [f for f in files if f['Path'] == file_name]
                if not files:
                    raise FileNotFoundError(f"File '{file_name}' not found in Google Drive")
                
                # assuming the first file is the one we want
                file = files[0]
                file_id = file['ID']

                # send warning if file is open document format
                if file['MimeType'] in self.OPENDOCUMENT_FORMATS:
                    subprocess.Popen(["zenity", "--warning", "--text", "You are about to open an Open Document Format file.\n\nOpening this with Google Docs will create a copy of the file!"])

                url = f'https://drive.google.com/open?id={file_id}'
                webbrowser.get("xdg-open").open(url)

            except subprocess.CalledProcessError as e:
                subprocess.Popen(["zenity", "--error", "--text", f"rclone failed:\n{e.stderr}"])
            except Exception as e:
                subprocess.Popen(["zenity", "--error", "--text", f"Unexpected error:\n{str(e)}"])
