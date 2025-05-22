#!/usr/bin/env bash

INSTANCE="$1"
MOUNT_POINT="$HOME/google/$INSTANCE"
CACHE_DIR="$HOME/.cache/google/$INSTANCE"

CMD=(/usr/bin/rclone mount \
    --cache-dir "$CACHE_DIR" \
    --vfs-cache-mode writes \
    --vfs-cache-max-size 10G)

if [[ "$INSTANCE" == "Shared_With_Me" ]]; then
    CMD+=(--drive-shared-with-me)
fi

CMD+=("$INSTANCE:" "$MOUNT_POINT")

# Execute the command
exec "${CMD[@]}"
