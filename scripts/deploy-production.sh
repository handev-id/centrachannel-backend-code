#!/usr/bin/env bash
# Install a tagged release's deployment config and immutable backend image.
set -euo pipefail
umask 077

if [[ $# -lt 1 || $# -gt 2 || ! "$1" =~ ^[0-9a-f]{40}$ ]]; then
  printf 'Usage: %s <40-character lowercase commit SHA> [production-compose-file]\n' "$0" >&2
  exit 2
fi
sha="$1"
source_compose="$(realpath "${2:-$(dirname "$0")/docker-compose.yml}")"
source_script="$(realpath "$0")"
root=/home/solodev/centrachannel/centrachannel-production
for command in docker curl flock python3; do
  command -v "$command" >/dev/null || { printf 'Missing prerequisite: %s\n' "$command" >&2; exit 1; }
done
[[ -f "$source_compose" ]] || { echo 'Production Compose source missing.' >&2; exit 1; }
cd "$root"
exec 9>.deploy-production.lock
flock -n 9 || { echo 'Another production deployment is running.' >&2; exit 1; }
[[ -f .env && ! -L .env ]] || { echo 'A regular production .env is required.' >&2; exit 1; }

# Do not source dotenv files as executable shell. Require canonical deployment keys.
python3 - <<'PY'
import pathlib, re
text = pathlib.Path('.env').read_text()
for key in ('IMAGE_NAME', 'IMAGE_TAG'):
    values = re.findall(r'^' + key + r'=(.*)$', text, re.M)
    if len(values) != 1:
        raise SystemExit(f'.env must contain exactly one unquoted {key}= line')
    if key == 'IMAGE_NAME' and values[0] != 'ghcr.io/handev-id/centrachannel-backend-code':
        raise SystemExit('Unexpected production IMAGE_NAME')
    if key == 'IMAGE_TAG' and pathlib.Path('docker-compose.yml').exists() and not re.fullmatch(r'[0-9a-f]{40}', values[0]):
        raise SystemExit('Existing deployment must have an immutable SHA IMAGE_TAG for rollback')
PY

backup="$(mktemp -d "$root/.deployment-backup.XXXXXXXX")"
cp -p .env "$backup/.env"
had_compose=false
had_script=false
if [[ -f docker-compose.yml ]]; then
  had_compose=true
  cp -p docker-compose.yml "$backup/docker-compose.yml"
fi
if [[ -f deploy-production.sh ]]; then
  had_script=true
  cp -p deploy-production.sh "$backup/deploy-production.sh"
fi

compose() {
  # Shell/SSH environment must not override the selected image in .env.
  env -u IMAGE_NAME -u IMAGE_TAG docker compose --env-file .env -p centrachannel-production -f docker-compose.yml "$@"
}
healthy() {
  local attempt id state
  for ((attempt=1; attempt<=45; attempt++)); do
    id="$(compose ps -q app 2>/dev/null)" || id=''
    state=''
    if [[ -n "$id" ]]; then
      state="$(docker inspect --format '{{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{end}}' "$id" 2>/dev/null)" || state=''
    fi
    if [[ "$state" == 'running healthy' ]] && curl --fail --silent --show-error --max-time 3 http://127.0.0.1:4000/health 2>/dev/null | grep -q '"status":"ok"'; then
      return 0
    fi
    sleep 2
  done
  return 1
}

startup_attempted=false
finish() {
  local status=$?
  trap - EXIT INT TERM
  if ((status == 0)); then
    printf 'Production deployed: %s; rollback snapshot: %s\n' "$sha" "$backup"
    return
  fi
  set +e
  echo 'Deployment failed; restoring previous configuration.' >&2
  # On the first release there is no previous app to recreate.
  if [[ "$had_compose" == false && "$startup_attempted" == true ]]; then
    compose stop app
  fi
  restore_ok=true
  if ! { cp -p "$backup/.env" .env.restore && mv .env.restore .env; }; then
    restore_ok=false
  fi
  if [[ "$had_compose" == true ]]; then
    if ! { cp -p "$backup/docker-compose.yml" docker-compose.yml.restore && mv docker-compose.yml.restore docker-compose.yml; }; then
      restore_ok=false
    fi
  elif ! rm -f docker-compose.yml; then
    restore_ok=false
  fi
  if [[ "$had_script" == true ]]; then
    if ! { cp -p "$backup/deploy-production.sh" deploy-production.sh.restore && mv deploy-production.sh.restore deploy-production.sh; }; then
      restore_ok=false
    fi
  elif ! rm -f deploy-production.sh; then
    restore_ok=false
  fi
  if [[ "$restore_ok" != true ]]; then
    printf 'ROLLBACK FAILED: could not restore configuration. Snapshot: %s\n' "$backup" >&2
    exit "$status"
  fi
  if [[ "$had_compose" == true && "$startup_attempted" == true ]]; then
    # The old immutable image is retained locally; never prune it on deploy.
    if compose up -d --no-build --pull never app && healthy; then
      echo 'Previous backend restored and healthy.' >&2
    else
      printf 'ROLLBACK FAILED: inspect production manually. Snapshot: %s\n' "$backup" >&2
    fi
  fi
  printf 'Failed release snapshot retained: %s\n' "$backup" >&2
  exit "$status"
}
trap finish EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

# Every failure from this point restores both runtime env and deployment files.
install -m 600 "$source_compose" docker-compose.yml.next
mv docker-compose.yml.next docker-compose.yml
if [[ "$source_script" != "$root/deploy-production.sh" ]]; then
  install -m 700 "$source_script" deploy-production.sh.next
  mv deploy-production.sh.next deploy-production.sh
fi
python3 - "$sha" <<'PY'
import pathlib, re, sys
path = pathlib.Path('.env')
updated = re.sub(r'^IMAGE_TAG=.*$', 'IMAGE_TAG=' + sys.argv[1], path.read_text(), flags=re.M)
tmp = pathlib.Path('.env.next')
tmp.write_text(updated)
tmp.chmod(0o600)
tmp.replace(path)
PY
compose pull app
startup_attempted=true
compose up -d --no-build app
if ! healthy; then
  echo 'Production failed startup/health verification.' >&2
  exit 1
fi
