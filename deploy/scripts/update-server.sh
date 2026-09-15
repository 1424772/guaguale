#!/usr/bin/env bash

set -euo pipefail

project_dir="${PROJECT_DIR:-/opt/guaguale}"

cd "${project_dir}"
git pull --ff-only origin main
docker compose config --quiet
docker compose up -d --build --remove-orphans
docker compose ps

