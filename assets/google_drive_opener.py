from gi.repository import Nautilus, GObject
import os
import subprocess
import webbrowser
import json
import httpx

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

    def get_file_items(self, *args):
        # `args` will be `[files: List[Nautilus.FileInfo]]` in Nautilus 4.0 API,
        # and `[window: Gtk.Widget, files: List[Nautilus.FileInfo]]` in Nautilus 3.0 API.
        files = args[-1]
        file_paths = []
        for file in files:
            file_path = file.get_location().get_path()
            if not os.path.abspath(file_path).startswith(self.RCLONE_MOUNT_PATH):
                return
            file_paths.append(file_path)

        items = []
        open_google_drive_item = Nautilus.MenuItem(
            name="GoogleDriveOpener::OpenPublicURL",
            label="Open in Google Drive",
            tip="Open the file in Google Drive",
        )
        open_google_drive_item.connect("activate", self.open_rclone_url, file_paths)
        items.append(open_google_drive_item)

        copy_file_item = Nautilus.MenuItem(
            name="GoogleDriveOpener::CopyFile",
            label="Copy on Google Drive",
            tip="Copy on Google Drive",
        )
        copy_file_item.connect("activate", self.copy_file, file_paths)
        items.append(copy_file_item)

        move_file_item = Nautilus.MenuItem(
            name="GoogleDriveOpener::MoveFile",
            label="Move on Google Drive",
            tip="Move on Google Drive",
        )
        move_file_item.connect("activate", self.move_file, file_paths)
        items.append(move_file_item)

        # add copy button if only one file is selected
        if len(file_paths) == 1:
            copy_file_link_item = Nautilus.MenuItem(
                name="GoogleDriveOpener::CopyShareLink",
                label="Copy File Link",
                tip="Copy the file link to the clipboard",
            )
            copy_file_link_item.connect("activate", self.copy_file_link, file_paths)
            items.append(copy_file_link_item)

        return items

    def _get_rclone_file(self, file_path):
        try:
            relative_path = os.path.relpath(file_path, self.RCLONE_MOUNT_PATH)
            drive_name = relative_path.split(os.sep)[0]
            # remove drive_name and the last part of the path
            file_path = os.path.join("", *relative_path.split(os.sep)[1:-1])
            file_name = os.path.basename(relative_path)

            cmd = ["rclone", "lsjson", f"{drive_name}:{file_path}"]
            result = subprocess.run(cmd, capture_output=True, text=True, check=True)
            files = json.loads(result.stdout)

            if not files:
                raise FileNotFoundError("No files returned from rclone lsjson")

            # find matching file by name
            files = [f for f in files if f["Path"] == file_name]
            if not files:
                raise FileNotFoundError(f"File '{file_name}' not found in Google Drive")

            # assuming the first file is the one we want
            file = files[0]
            return file

        except subprocess.CalledProcessError as e:
            subprocess.Popen(
                ["zenity", "--error", "--text", f"rclone failed:\n{e.stderr}"]
            )
        except Exception as e:
            subprocess.Popen(
                ["zenity", "--error", "--text", f"Unexpected error:\n{str(e)}"]
            )

    def open_rclone_url(self, menu, file_paths):
        for file_path in file_paths:
            try:
                file = self._get_rclone_file(file_path)
                file_id = file["ID"]

                # send warning if file is open document format
                if file["MimeType"] in self.OPENDOCUMENT_FORMATS:
                    subprocess.Popen(
                        [
                            "zenity",
                            "--warning",
                            "--text",
                            "You are about to open an Open Document Format file.\n\nOpening this with Google Docs will create a copy of the file!",
                        ]
                    )

                url = f"https://drive.google.com/open?id={file_id}"
                webbrowser.get("xdg-open").open(url)

            except Exception as e:
                subprocess.Popen(
                    ["zenity", "--error", "--text", f"Unexpected error:\n{str(e)}"]
                )

    def _copy_to_clipboard(self, text):
        try:
            subprocess.run(["xclip", "-selection", "clipboard"], input=text.encode())
            subprocess.run(["xclip", "-selection", "primary"], input=text.encode())
        except Exception as e:
            print(f"Clipboard copy failed: {e}")

    def copy_file_link(self, menu, file_paths):
        file_path = file_paths[0]
        try:
            file = self._get_rclone_file(file_path)
            file_id = file["ID"]
            url = f"https://drive.google.com/open?id={file_id}"
            self._copy_to_clipboard(url)
        except Exception as e:
            subprocess.Popen(
                ["zenity", "--error", "--text", f"Unexpected error:\n{str(e)}"]
            )

    def _send_file_op(self, file_paths, op):
        try:
            if not file_paths:
                return
            relative_path = os.path.relpath(file_paths[0], self.RCLONE_MOUNT_PATH)
            drive_name = relative_path.split(os.sep)[0]
            sock_path = os.path.join(
                os.environ.get("XDG_RUNTIME_DIR", "/run/user/%d" % os.getuid()),
                "adfinis-rclone-mgr",
                f"{drive_name}.sock",
            )
            url = f"http://localhost/gdrive/{op}"
            transport = httpx.HTTPTransport(uds=sock_path)
            with httpx.Client(transport=transport) as client:
                resp = client.post(url, json={"sources": file_paths}, timeout=600)
                if resp.status_code != 200:
                    raise Exception(f"Server error: {resp.text}")
        except Exception as e:
            subprocess.Popen(
                [
                    "zenity",
                    "--error",
                    "--text",
                    f"Failed to {op} file(s) via daemon: {str(e)}",
                ]
            )

    def copy_file(self, menu, file_paths):
        self._send_file_op(file_paths, "copy")

    def move_file(self, menu, file_paths):
        self._send_file_op(file_paths, "move")
