import sys
from gi.repository import GObject
import os
import subprocess
import httpx

# hacky but most reliable way to detect if we're running within Nemo. We always fall
# back to Nautilus
if sys.argv[0] == "nemo":
    from gi.repository import Nemo as FileManager
else:
    from gi.repository import Nautilus as FileManager

"""
This extension adds a context menu item to Nautilus and Nemo for opening files in Google
Drive.
It generates a public link using rclone and opens it in the default web browser.

The extension expects rclone to mount the drives at ~/google/$drive_name.
"""


class GoogleDriveOpener(GObject.GObject, FileManager.MenuProvider):
    RCLONE_MOUNT_PATH = os.path.expanduser("~/google")

    def get_file_items(self, *args):
        # `args` will be `[files: List[Nautilus.FileInfo]]` in Nautilus 4.0 API,
        # and `[window: Gtk.Widget, files: List[Nautilus.FileInfo]]` in Nautilus 3.0 API.
        files = args[-1]
        file_paths = []
        for file in files:
            file_path = file.get_location().get_path()
            abs_path = os.path.abspath(file_path)
            if (
                not abs_path.startswith(self.RCLONE_MOUNT_PATH)
                or abs_path == self.RCLONE_MOUNT_PATH
            ):
                return
            # Check if the path is only one level below google directory
            rel_path = os.path.relpath(abs_path, self.RCLONE_MOUNT_PATH)
            if os.sep not in rel_path:
                return
            file_paths.append(file_path)

        items = []
        open_google_drive_item = FileManager.MenuItem(
            name="GoogleDriveOpener::OpenPublicURL",
            label="Open in Google Drive",
            tip="Open the file in Google Drive",
        )
        open_google_drive_item.connect("activate", self.open_rclone_url, file_paths)
        items.append(open_google_drive_item)

        copy_file_item = FileManager.MenuItem(
            name="GoogleDriveOpener::CopyFile",
            label="Copy on Google Drive",
            tip="Copy on Google Drive",
        )
        copy_file_item.connect("activate", self.copy_file, file_paths)
        items.append(copy_file_item)

        move_file_item = FileManager.MenuItem(
            name="GoogleDriveOpener::MoveFile",
            label="Move on Google Drive",
            tip="Move on Google Drive",
        )
        move_file_item.connect("activate", self.move_file, file_paths)
        items.append(move_file_item)

        duplicate_file_item = FileManager.MenuItem(
            name="GoogleDriveOpener::DuplicateFile",
            label="Duplicate on Google Drive",
            tip="Duplicate on Google Drive",
        )
        duplicate_file_item.connect("activate", self.duplicate_file, file_paths)
        items.append(duplicate_file_item)

        # add copy button if only one file is selected
        if len(file_paths) == 1:
            copy_file_link_item = FileManager.MenuItem(
                name="GoogleDriveOpener::CopyShareLink",
                label="Copy File Link",
                tip="Copy the file link to the clipboard",
            )
            copy_file_link_item.connect("activate", self.copy_file_link, file_paths)
            items.append(copy_file_link_item)

        return items

    def open_rclone_url(self, menu, file_paths):
        self._send_file_op(file_paths, "open")

    def copy_file_link(self, menu, file_paths):
        self._send_file_op(file_paths, "link")

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

    def duplicate_file(self, menu, file_paths):
        self._send_file_op(file_paths, "duplicate")
