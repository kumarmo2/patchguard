#!/bin/sh
set -eu

# Always deploy the worktree containing this script, regardless of the caller's cwd.
source_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
releases=/opt/patchguard/releases

sh "$source_dir/deploy-ready.sh" --check

# Refresh the service account's vault before switching releases or restarting it.
if ! sudo -u review-bot -g review-bot env -u BW_SESSION -u BITWARDENCLI_DEBUG \
    HOME=/var/lib/review-bot PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin \
    timeout 60s bw sync --nointeraction; then
    printf '%s\n' 'Vaultwarden sync failed; deployment aborted. Check network access and bw login for review-bot.' >&2
    exit 1
fi

sudo install -d -m 0755 /opt/patchguard "$releases"
release=$(sudo mktemp -d "$releases/review-bot-api.XXXXXXXX")
sudo chmod 0755 "$release"

# Copy the actual working files, not just committed files; omit build output.
tar -C "$source_dir" --exclude='./target' --exclude='./.git' -cf - . | sudo tar -C "$release" -xf -
sudo install -d -o review-bot -g review-bot "$release/target"
sudo -u review-bot env HOME=/var/lib/review-bot PATH=/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin make -C "$release" build

# The drop-in selects this release without overwriting or deleting older deployments.
sudo install -m 0644 "$release/review-bot-api.service" /etc/systemd/system/review-bot-api.service
sudo install -d -m 0755 /etc/systemd/system/review-bot-api.service.d
printf '[Service]\nWorkingDirectory=%s\n' "$release" | sudo tee /etc/systemd/system/review-bot-api.service.d/deployment.conf >/dev/null
sudo systemctl daemon-reload
sudo systemctl enable review-bot-api.service
sudo systemctl restart review-bot-api.service
sudo systemctl --no-pager --full status review-bot-api.service
printf '\nDeployed %s\n' "$release"
