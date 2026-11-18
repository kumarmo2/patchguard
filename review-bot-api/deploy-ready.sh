#!/bin/sh
set -eu

config_dir=/etc/patchguard
secret_dir=$config_dir/secrets
password_file=$secret_dir/bw-password
config_file=$config_dir/review-bot-api.env
service_home=/var/lib/review-bot
config_line="SECRET_PASSWORD_FILE=$password_file"

fail() {
    printf '%s\n' "$1" >&2
    exit 1
}

config_matches() {
    printf '%s\n' "$config_line" | sudo cmp -s "$config_file" -
}

check_setup() {
    if ! id review-bot >/dev/null 2>&1 || ! getent group review-bot >/dev/null 2>&1; then
        fail 'The review-bot account or group is missing. Run make deploy-ready first.'
    fi
    if ! sudo -u review-bot -g review-bot test -w "$service_home"; then
        fail "The service account cannot write to $service_home. Run make deploy-ready first."
    fi
    if ! config_matches; then
        fail "$config_file must contain only $config_line. Run make deploy-ready or update the existing file manually."
    fi
    if ! sudo -u review-bot -g review-bot test -f "$password_file" ||
       ! sudo -u review-bot -g review-bot test -r "$password_file" ||
       ! sudo -u review-bot -g review-bot test -s "$password_file"; then
        fail "The password file at $password_file is missing, empty, or unreadable by review-bot. Run make deploy-ready first."
    fi
    if sudo test -L "$password_file" ||
       [ "$(sudo stat -c '%U:%G %a' "$password_file")" != 'root:review-bot 640' ]; then
        fail "$password_file must be a regular file owned by root:review-bot with mode 0640."
    fi
}

case ${1:-} in
    --check)
        check_setup
        exit 0
        ;;
    '') ;;
    *) fail 'Usage: deploy-ready.sh [--check]' ;;
esac

source_file=${PASSWORD_SOURCE:-}
if [ -z "$source_file" ]; then
    fail 'Usage: make deploy-ready PASSWORD_SOURCE=/path/to/existing-password-file'
fi
case $source_file in
    /*) ;;
    *) source_file=$PWD/$source_file ;;
esac
if ! sudo test -f "$source_file" || ! sudo test -s "$source_file"; then
    fail 'PASSWORD_SOURCE must point to an existing, non-empty regular file.'
fi

if ! id review-bot >/dev/null 2>&1; then
    sudo useradd --system --user-group --home-dir "$service_home" --shell /usr/sbin/nologin review-bot
fi
if ! getent group review-bot >/dev/null 2>&1; then
    fail 'The review-bot group is missing; create it before running make deploy-ready.'
fi

sudo install -d -o review-bot -g review-bot -m 0750 "$service_home"
sudo install -d -o root -g root -m 0755 "$config_dir"
sudo install -d -o root -g review-bot -m 0750 "$secret_dir"

if sudo test -e "$config_file" && ! config_matches; then
    fail "$config_file already exists and differs; update it manually to contain only $config_line."
fi
if sudo test -e "$password_file"; then
    if ! sudo cmp -s "$source_file" "$password_file"; then
        fail "$password_file already exists and differs; refusing to overwrite it. Rotate it manually if intended."
    fi
else
    sudo install -o root -g review-bot -m 0640 "$source_file" "$password_file"
fi
if ! sudo test -e "$config_file"; then
    printf '%s\n' "$config_line" | sudo tee "$config_file" >/dev/null
fi

check_setup
printf '%s\n' 'Deployment prerequisites are ready. Run make deploy when you want to deploy.'
