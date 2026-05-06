#!/usr/bin/env bash
set -euo pipefail

GENERATED_ENV=".env.generated"

docker compose --env-file "$GENERATED_ENV" down