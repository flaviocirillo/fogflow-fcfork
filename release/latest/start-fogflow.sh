#!/usr/bin/env bash
set -euo pipefail

COMPOSE_FILE="${COMPOSE_FILE:-docker-compose.yml}"

ENV_FILE="${1:-fogflow.config}"
GENERATED_ENV=".env.generated"
GENERATED_COMPOSE_FILE=".docker-compose.generated.yml"

if [ -f "$ENV_FILE" ]; then
  set -a
  source "$ENV_FILE"
  set +a
fi

detect_host_ip() {
  ip route get 1.1.1.1 2>/dev/null | awk '{for(i=1;i<=NF;i++) if ($i=="src") {print $(i+1); exit}}'
}

is_enabled() {
  case "${1:-}" in
    true|TRUE|True|1|yes|YES|Yes|y|Y)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

HOST_IP="${HOST_IP:-$(detect_host_ip)}"
HOST_IP="${HOST_IP:-172.17.0.1}"

FOGFLOW_DAEMON="${FOGFLOW_DAEMON:-false}"
FOGFLOW_MODE="${FOGFLOW_MODE:-full}"

DOCKER_SUBNET="${DOCKER_SUBNET:-172.32.0.0/16}"
DOCKER_GATEWAY="${DOCKER_GATEWAY:-172.32.0.1}"

LONGITUDE="${LONGITUDE:-139}"
LATITUDE="${LATITUDE:-35}"
SITE_ID="${SITE_ID:-001}"

LOG_INFO="${LOG_INFO:-stdout}"
LOG_ERROR="${LOG_ERROR:-stdout}"
LOG_PROTOCOL="${LOG_PROTOCOL:-stdout}"
LOG_DEBUG="${LOG_DEBUG:-discard}"

DISCOVERY_PORT="${DISCOVERY_PORT:-8090}"
DISCOVERY_STORE_ON_DISK="${DISCOVERY_STORE_ON_DISK:-false}"
DISCOVERY_DB_DIR="${DISCOVERY_DB_DIR:-/tmp/fogflow/discoveryDB}"
DISCOVERY_DELAY_STORE_ON_FILE="${DISCOVERY_DELAY_STORE_ON_FILE:-3}"

BROKER_PORT="${BROKER_PORT:-8070}"
BROKER_HEARTBEAT_INTERVAL="${BROKER_HEARTBEAT_INTERVAL:-30}"

MASTER_NGSI_AGENT_PORT="${MASTER_NGSI_AGENT_PORT:-1060}"
MASTER_REST_API_PORT="${MASTER_REST_API_PORT:-8010}"
MASTER_INFINITE_RECONNECTION_TRIES="${MASTER_INFINITE_RECONNECTION_TRIES:-true}"

WORKER_CONTAINER_AUTOREMOVE="${WORKER_CONTAINER_AUTOREMOVE:-false}"
WORKER_START_ACTUAL_TASK="${WORKER_START_ACTUAL_TASK:-true}"
WORKER_CAPACITY="${WORKER_CAPACITY:-8}"
WORKER_HEARTBEAT_INTERVAL="${WORKER_HEARTBEAT_INTERVAL:-30}"
WORKER_DETECTION_DURATION="${WORKER_DETECTION_DURATION:-10}"
WORKER_INFINITE_RECONNECTION_TRIES="${WORKER_INFINITE_RECONNECTION_TRIES:-true}"
WORKER_CONTAINER_MANAGEMENT="${WORKER_CONTAINER_MANAGEMENT:-docker}"

case "$WORKER_CONTAINER_MANAGEMENT" in
  docker)
    WORKER_PROFILE="worker_docker"
    ;;
  airflow)
    WORKER_PROFILE="worker_airflow"
    ;;
  *)
    echo "ERROR: invalid WORKER_CONTAINER_MANAGEMENT=${WORKER_CONTAINER_MANAGEMENT}"
    echo "Allowed values: docker, airflow"
    exit 1
    ;;
esac

DESIGNER_WEB_PORT="${DESIGNER_WEB_PORT:-8080}"
DESIGNER_AGENT_PORT="${DESIGNER_AGENT_PORT:-1030}"
DESIGNER_LD_AGENT_PORT="${DESIGNER_LD_AGENT_PORT:-1090}"
DESIGNER_DO_NOT_INIT_APPLICATIONS="${DESIGNER_DO_NOT_INIT_APPLICATIONS:-true}"
DESIGNER_STORE_ON_DISK="${DESIGNER_STORE_ON_DISK:-false}"
DESIGNER_DB_DIR="${DESIGNER_DB_DIR:-/tmp/fogflow/designerDB}"

RABBITMQ_PORT="${RABBITMQ_PORT:-5672}"
RABBITMQ_USER="${RABBITMQ_USER:-admin}"
RABBITMQ_PASS="${RABBITMQ_PASS:-mypass}"

HTTPS_ENABLED="${HTTPS_ENABLED:-false}"
PERSISTENT_STORAGE_PORT="${PERSISTENT_STORAGE_PORT:-9082}"

NGINX_PORT="${NGINX_PORT:-80}"
NGINX_SERVER_NAME="${NGINX_SERVER_NAME:-_}"

AIRFLOW_DAGS="${AIRFLOW_DAGS:-/tmp/fogflow/airflow/dags}"
PYMODULE_REPO="${PYMODULE_REPO:-/tmp/fogflow/pymodules}"

generate_compose_file() {
  python3 - "$COMPOSE_FILE" "$GENERATED_COMPOSE_FILE" "$DESIGNER_STORE_ON_DISK" "$DISCOVERY_STORE_ON_DISK" <<'PY'
from pathlib import Path
import sys

source_path = Path(sys.argv[1])
target_path = Path(sys.argv[2])
designer_store_on_disk = sys.argv[3].lower() in {"true", "1", "yes", "y"}
discovery_store_on_disk = sys.argv[4].lower() in {"true", "1", "yes", "y"}

service_name = None
output_lines = []

for line in source_path.read_text().splitlines():
    stripped = line.strip()

    if line.startswith("  ") and not line.startswith("    ") and stripped.endswith(":"):
        service_name = stripped[:-1]

    if service_name == "designer" and stripped == "- ${DESIGNER_DB_DIR}:/app/public/data/meta" and not designer_store_on_disk:
        output_lines.append("      # - ${DESIGNER_DB_DIR}:/app/public/data/meta")
        continue

    if service_name == "discovery" and stripped == "- ${DISCOVERY_DB_DIR}:/discoveryDB" and not discovery_store_on_disk:
        output_lines.append("      # - ${DISCOVERY_DB_DIR}:/discoveryDB")
        continue

    output_lines.append(line)

target_path.write_text("\n".join(output_lines) + "\n")
PY
}

cat > .config.json <<EOF
{
  "my_hostip": "${HOST_IP}",
  "physical_location": {
    "longitude": ${LONGITUDE},
    "latitude": ${LATITUDE}
  },
  "site_id": "${SITE_ID}",
  "logging": {
    "info": "${LOG_INFO}",
    "error": "${LOG_ERROR}",
    "protocol": "${LOG_PROTOCOL}",
    "debug": "${LOG_DEBUG}"
  },
  "discovery": {
    "http_port": ${DISCOVERY_PORT},
    "storeOnDisk": ${DISCOVERY_STORE_ON_DISK},
    "delayStoreOnFile": ${DISCOVERY_DELAY_STORE_ON_FILE}
  },
  "broker": {
    "http_port": ${BROKER_PORT},
    "heartbeat_interval": ${BROKER_HEARTBEAT_INTERVAL}
  },
  "master": {
    "ngsi_agent_port": ${MASTER_NGSI_AGENT_PORT},
    "rest_api_port": ${MASTER_REST_API_PORT},
    "infinite_reconnection_tries": ${MASTER_INFINITE_RECONNECTION_TRIES}
  },
  "worker": {
    "container_autoremove": ${WORKER_CONTAINER_AUTOREMOVE},
    "start_actual_task": ${WORKER_START_ACTUAL_TASK},
    "capacity": ${WORKER_CAPACITY},
    "heartbeat_interval": ${WORKER_HEARTBEAT_INTERVAL},
    "detection_duration": ${WORKER_DETECTION_DURATION},
    "infinite_reconnection_tries": ${WORKER_INFINITE_RECONNECTION_TRIES},
    "container_management": "${WORKER_CONTAINER_MANAGEMENT}"
  },
  "designer": {
    "webSrvPort": ${DESIGNER_WEB_PORT},
    "agentPort": ${DESIGNER_AGENT_PORT},
    "ldAgentPort": ${DESIGNER_LD_AGENT_PORT},
    "doNotInitApplications": ${DESIGNER_DO_NOT_INIT_APPLICATIONS}
  },
  "rabbitmq": {
    "port": ${RABBITMQ_PORT},
    "username": "${RABBITMQ_USER}",
    "password": "${RABBITMQ_PASS}"
  },
  "https": {
    "enabled": ${HTTPS_ENABLED}
  },
  "persistent_storage": {
    "port": ${PERSISTENT_STORAGE_PORT}
  }
}
EOF

cat > .nginx.conf <<EOF
events {
  worker_connections 4096;
}

http {
  server {
    listen 80;
    server_name ${NGINX_SERVER_NAME};

    location / {
      proxy_pass http://designer:${DESIGNER_WEB_PORT}/;
    }

    location /ngsi9/ {
      proxy_pass http://discovery:${DISCOVERY_PORT}/ngsi9/;
    }

    location /ngsi10/ {
      proxy_pass http://cloud_broker:${BROKER_PORT}/ngsi10/;
    }

    location /ngsi-ld/ {
      proxy_pass http://cloud_broker:${BROKER_PORT}/ngsi-ld/;
    }
  }
}
EOF

cat > "$GENERATED_ENV" <<EOF
HOST_IP=${HOST_IP}

FOGFLOW_DAEMON=${FOGFLOW_DAEMON}
FOGFLOW_MODE=${FOGFLOW_MODE}

DESIGNER_WEB_PORT=${DESIGNER_WEB_PORT}
DESIGNER_AGENT_PORT=${DESIGNER_AGENT_PORT}
DESIGNER_LD_AGENT_PORT=${DESIGNER_LD_AGENT_PORT}
DESIGNER_DB_DIR=${DESIGNER_DB_DIR}

DISCOVERY_PORT=${DISCOVERY_PORT}
DISCOVERY_DB_DIR=${DISCOVERY_DB_DIR}
BROKER_PORT=${BROKER_PORT}
MASTER_NGSI_AGENT_PORT=${MASTER_NGSI_AGENT_PORT}
MASTER_REST_API_PORT=${MASTER_REST_API_PORT}

RABBITMQ_PORT=${RABBITMQ_PORT}
RABBITMQ_USER=${RABBITMQ_USER}
RABBITMQ_PASS=${RABBITMQ_PASS}

NGINX_PORT=${NGINX_PORT}

AIRFLOW_DAGS=${AIRFLOW_DAGS}
PYMODULE_REPO=${PYMODULE_REPO}

DOCKER_SUBNET=${DOCKER_SUBNET}
DOCKER_GATEWAY=${DOCKER_GATEWAY}
EOF


echo "----------------------------------------"
echo "🚀 FogFlow Configuration Summary"
echo "----------------------------------------"

echo "🌐 HOST"
echo "  HOST_IP: ${HOST_IP}"
echo ""

if [ "$FOGFLOW_MODE" = "full" ]; then
  echo "🎨 DESIGNER"
  echo "  Web UI:        http://localhost:${DESIGNER_WEB_PORT}"
  echo "  Agent Port:    ${DESIGNER_AGENT_PORT}"
  echo "  LD Agent Port: ${DESIGNER_LD_AGENT_PORT}"
  echo "  Store on disk: ${DESIGNER_STORE_ON_DISK}"
  echo ""

  echo "🔎 DISCOVERY"
  echo "  HTTP Port:     ${DISCOVERY_PORT}"
  echo "  Store on disk: ${DISCOVERY_STORE_ON_DISK}"
  echo ""

  echo "🧠 MASTER"
  echo "  NGSI Agent:    ${MASTER_NGSI_AGENT_PORT}"
  echo "  REST API:      ${MASTER_REST_API_PORT}"
  echo ""
fi

echo "📡 BROKER"
echo "  HTTP Port:     ${BROKER_PORT}"
echo "  Heartbeat:     ${BROKER_HEARTBEAT_INTERVAL}s"
echo ""

echo "⚙️ WORKER"
echo "  Worker profile: ${WORKER_PROFILE}"
echo "  Capacity:      ${WORKER_CAPACITY}"
echo "  Heartbeat:     ${WORKER_HEARTBEAT_INTERVAL}s"
echo "  Detection:     ${WORKER_DETECTION_DURATION}s"
echo "  Mode:          ${WORKER_CONTAINER_MANAGEMENT}"
echo ""

if [ "$FOGFLOW_MODE" = "full" ]; then
  echo "🐇 RABBITMQ"
  echo "  Port:          ${RABBITMQ_PORT}"
  echo "  User:          ${RABBITMQ_USER}"
  echo ""

  echo "🌍 NGINX"
  echo "  External URL:  http://localhost:${NGINX_PORT}"
  echo ""
fi

if [ "$FOGFLOW_MODE" = "edge" ]; then
  echo "⚡ EDGE MODE ACTIVE"
  echo "  Running: Broker + Worker only"
  echo ""
fi

echo "----------------------------------------"
echo "Mode:    ${FOGFLOW_MODE}"
echo "Daemon:  ${FOGFLOW_DAEMON}"
echo "----------------------------------------"

if [ "$WORKER_CONTAINER_MANAGEMENT" = "docker" ] && [ ! -S /var/run/docker.sock ]; then
  echo "ERROR: WORKER_CONTAINER_MANAGEMENT=docker but /var/run/docker.sock is not available"
  exit 1
fi

if [ "$WORKER_CONTAINER_MANAGEMENT" = "airflow" ]; then
  mkdir -p "$AIRFLOW_DAGS"
  mkdir -p "$PYMODULE_REPO"
fi

case "$FOGFLOW_MODE" in
  full)
    PROFILE="full"
    if is_enabled "$DISCOVERY_STORE_ON_DISK"; then
      mkdir -p "$DISCOVERY_DB_DIR"
    fi
    if is_enabled "$DESIGNER_STORE_ON_DISK"; then
      mkdir -p "$DESIGNER_DB_DIR"
    fi
    ;;
  edge)
    PROFILE="edge"
    ;;
  *)
    echo "ERROR: invalid FOGFLOW_MODE=${FOGFLOW_MODE}"
    echo "Allowed values: full, edge"
    exit 1
    ;;
esac

generate_compose_file

case "$FOGFLOW_DAEMON" in
  true|1|yes|y)
    docker compose -f "$GENERATED_COMPOSE_FILE" \
      --env-file "$GENERATED_ENV" \
      --profile "$PROFILE" \
      --profile "$WORKER_PROFILE" \
      up -d
    ;;
  false|0|no|n)
    docker compose -f "$GENERATED_COMPOSE_FILE" \
      --env-file "$GENERATED_ENV" \
      --profile "$PROFILE" \
      --profile "$WORKER_PROFILE" \
      up
    ;;
  *)
    echo "ERROR: invalid FOGFLOW_DAEMON=${FOGFLOW_DAEMON}"
    echo "Allowed values: true/false, 1/0, yes/no, y/n"
    exit 1
    ;;
esac