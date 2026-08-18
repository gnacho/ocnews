#!/bin/sh
# install.sh — instala o actualiza ocnews-backend en Linux (systemd, sin Docker).
# Uso: curl -fsSL https://github.com/gnacho/ocnews/releases/latest/download/install.sh | sh
#      sh install.sh --version v0.1.1
#      sh install.sh --uninstall

set -eu

APP="ocnews"
USER="ocnews"
REPO="gnacho/ocnews"
INSTALL_DIR="/usr/local/bin"
DATA_DIR="/var/lib/${APP}"
CONFIG_DIR="/etc/${APP}"
ENV_FILE="${CONFIG_DIR}/env"
PORT="8094"

DRY_RUN=0
UNINSTALL=0
PURGE=0
VERSION=""

for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    --uninstall) UNINSTALL=1 ;;
    --purge) UNINSTALL=1; PURGE=1 ;;
    --version) shift; VERSION="${1:-}" ;;
    --version=*) VERSION="${arg#*=}" ;;
    *) echo "Uso: $0 [--dry-run] [--uninstall] [--purge] [--version vX.Y.Z]"; exit 1 ;;
  esac
done

if [ -t 0 ]; then
  INTERACTIVE=1
else
  INTERACTIVE=0
fi

run() {
  if [ "$DRY_RUN" -eq 1 ]; then
    echo "[dry-run] $*"
  else
    "$@"
  fi
}

root_required() {
  if [ "$(id -u)" -ne 0 ]; then
    if command -v sudo >/dev/null 2>&1; then
      sudo "$@"
    elif command -v doas >/dev/null 2>&1; then
      doas "$@"
    else
      echo "Este script necesita privilegios de root. Ejecuta como root o instala sudo/doas." >&2
      exit 1
    fi
  else
    "$@"
  fi
}

require_root() {
  if [ "$(id -u)" -ne 0 ]; then
    echo "Este paso requiere root." >&2
    exit 1
  fi
}

detect_arch() {
  arch=$(uname -m)
  case "$arch" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) echo "Arquitectura no soportada: $arch" >&2; exit 1 ;;
  esac
}

fetch_latest_version() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | \
    sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n1
}

random_pass() {
  tr -dc 'a-zA-Z0-9' </dev/urandom | head -c 16
}

free_port() {
  p="$PORT"
  for _ in $(seq 1 20); do
    if ! ss -Hln | awk '{print $5}' | grep -qE ":${p}$"; then
      echo "$p"
      return
    fi
    p=$((p + 1))
  done
  echo "$PORT"
}

stop_service() {
  if command -v systemctl >/dev/null 2>&1; then
    run systemctl stop "${APP}.service" >/dev/null 2>&1 || true
    run systemctl disable "${APP}.service" >/dev/null 2>&1 || true
  fi
}

uninstall() {
  echo "Desinstalando ${APP}..."
  stop_service
  run rm -f "${INSTALL_DIR}/${APP}"
  run rm -f "/etc/systemd/system/${APP}.service"
  run rm -f "/etc/systemd/system/${APP}.path"
  run systemctl daemon-reload >/dev/null 2>&1 || true
  if [ "$PURGE" -eq 1 ]; then
    run rm -rf "$DATA_DIR" "$CONFIG_DIR"
    run userdel "$USER" >/dev/null 2>&1 || true
  fi
  echo "${APP} desinstalado."
  if [ "$PURGE" -eq 1 ]; then
    echo "Datos y configuración eliminados."
  else
    echo "Datos conservados en ${DATA_DIR} y ${CONFIG_DIR}."
  fi
  exit 0
}

[ "$UNINSTALL" -eq 1 ] && uninstall

ARCH=$(detect_arch)
if [ -z "$VERSION" ]; then
  VERSION=$(fetch_latest_version)
  if [ -z "$VERSION" ]; then
    echo "No se pudo detectar la última versión. Usa --version vX.Y.Z." >&2
    exit 1
  fi
fi
VERSION_NORM=${VERSION#v}

TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

ASSET="${APP}_${VERSION_NORM}_linux_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${ASSET}"
CHECKSUM_URL="https://github.com/${REPO}/releases/download/${VERSION}/checksums.txt"

echo "Descargando ${APP} ${VERSION} para linux/${ARCH}..."
run curl -fsSL -o "${TMPDIR}/${ASSET}" "$URL"
run curl -fsSL -o "${TMPDIR}/checksums.txt" "$CHECKSUM_URL"

(cd "$TMPDIR" && sha256sum -c <(grep "${ASSET}" checksums.txt) >/dev/null 2>&1) || {
  echo "Error: el checksum no coincide para ${ASSET}." >&2
  exit 1
}

run tar -xzf "${TMPDIR}/${ASSET}" -C "$TMPDIR"

if [ -d "$DATA_DIR" ] && [ -f "$ENV_FILE" ]; then
  UPGRADE=1
else
  UPGRADE=0
fi

echo "Instalando binario..."
run root_required install -Dm755 "${TMPDIR}/${APP}" "${INSTALL_DIR}/${APP}"

run root_required mkdir -p "$DATA_DIR" "$CONFIG_DIR"

if ! id -u "$USER" >/dev/null 2>&1; then
  run root_required useradd --system --home-dir "$DATA_DIR" --shell /usr/sbin/nologin "$USER"
fi

run root_required chown -R "${USER}:${USER}" "$DATA_DIR"
run root_required chmod 750 "$DATA_DIR"

if [ "$UPGRADE" -eq 0 ]; then
  FINAL_PORT=$(free_port)
  ADMIN_PASS=$(random_pass)
  cat > "${TMPDIR}/env" <<EOF
OCNEWS_ADDR=:${FINAL_PORT}
OCNEWS_DATA_DIR=${DATA_DIR}
OCNEWS_AUTH_MODE=local
OCNEWS_FEED_INTERVAL=15m
OCNEWS_MAX_GAP=6h
OCNEWS_RETENTION_DAYS=90
OCNEWS_FETCH_TIMEOUT=20s
OCNEWS_LOG_LEVEL=info
AUTH_USER=admin
AUTH_PASS=${ADMIN_PASS}
EOF
  run root_required install -Dm600 "${TMPDIR}/env" "$ENV_FILE"
  run root_required chown "${USER}:${USER}" "$ENV_FILE"
else
  run root_required chown "${USER}:${USER}" "$ENV_FILE"
  FINAL_PORT=$(grep -E '^OCNEWS_ADDR=' "$ENV_FILE" | sed 's/.*://' || echo "$PORT")
fi

cat > "${TMPDIR}/${APP}.service" <<EOF
[Unit]
Description=ocnews RSS/Atom backend
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=${USER}
Group=${USER}
WorkingDirectory=${DATA_DIR}
EnvironmentFile=${ENV_FILE}
ExecStart=${INSTALL_DIR}/${APP}
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${DATA_DIR}
PrivateTmp=true
LockPersonality=true
RemoveIPC=true
UMask=007

[Install]
WantedBy=multi-user.target
EOF

run root_required install -Dm644 "${TMPDIR}/${APP}.service" "/etc/systemd/system/${APP}.service"

if command -v systemctl >/dev/null 2>&1; then
  run root_required systemctl daemon-reload
  run root_required systemctl enable "${APP}.service"
  run root_required systemctl restart "${APP}.service"
fi

IP=$(hostname -I 2>/dev/null | awk '{print $1}' || echo "localhost")

echo ""
echo "================ ${APP} ${UPGRADE:+actualizado }instalado ================"
echo "Versión:  ${VERSION}"
echo "Binario:  ${INSTALL_DIR}/${APP}"
echo "Datos:    ${DATA_DIR}"
echo "Config:   ${ENV_FILE}"
echo "URL:      http://${IP}:${FINAL_PORT}"
if [ "$UPGRADE" -eq 0 ]; then
  echo ""
  echo "Credenciales iniciales (muéstranlas UNA sola vez):"
  echo "  Usuario:  admin"
  echo "  Contraseña: ${ADMIN_PASS}"
fi
echo ""
echo "Comandos útiles:"
echo "  systemctl status ${APP}"
echo "  journalctl -u ${APP} -f"
echo ""
echo "Para desinstalar: sh install.sh --uninstall"
echo "Para purgar todo:   sh install.sh --purge"
