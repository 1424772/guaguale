#!/usr/bin/env bash

set -euo pipefail

project_dir="${PROJECT_DIR:-/opt/guaguale}"
backup_dir="${BACKUP_DIR:-/opt/guaguale/backups/mysql}"
retention_days="${RETENTION_DAYS:-14}"
timestamp="$(date +'%Y%m%d_%H%M%S')"
backup_file="${backup_dir}/guaguale_${timestamp}.sql.gz"

umask 077
mkdir -p "${backup_dir}"
cd "${project_dir}"

docker compose exec -T mysql sh -c \
  'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" exec mysqldump --single-transaction --quick --routines --events -uroot "$MYSQL_DATABASE"' \
  | gzip -9 > "${backup_file}"

if [[ ! -s "${backup_file}" ]]; then
  rm -f "${backup_file}"
  echo "database backup is empty" >&2
  exit 1
fi

find "${backup_dir}" -type f -name 'guaguale_*.sql.gz' -mtime "+${retention_days}" -delete
echo "database backup created: ${backup_file}"

