#!/usr/bin/env bash
# bootstrap-vps.sh — Idempotent first-run setup for Voltana on Ubuntu 22.04 LTS.
# Run as root (or sudo bash bootstrap-vps.sh). Safe to re-run.
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[bootstrap]${NC} $*"; }
warn()  { echo -e "${YELLOW}[bootstrap]${NC} $*"; }
abort() { echo -e "${RED}[bootstrap] ERROR:${NC} $*" >&2; exit 1; }

# ── 0. Require root ───────────────────────────────────────────────────────────
[[ $EUID -eq 0 ]] || abort "Run as root: sudo bash scripts/bootstrap-vps.sh"

# ── 1. System packages ────────────────────────────────────────────────────────
info "Updating package index…"
apt-get update -q

info "Installing base packages…"
apt-get install -y -q \
    ca-certificates curl gnupg lsb-release \
    ufw git gettext-base certbot python3-certbot-nginx

# ── 2. Node.js 20 (LTS) via NodeSource ───────────────────────────────────────
NODE_OK=false
if command -v node &>/dev/null; then
    NODE_MAJOR=$(node --version | sed 's/v\([0-9]*\).*/\1/')
    [[ $NODE_MAJOR -ge 18 ]] && NODE_OK=true
fi

if $NODE_OK; then
    info "Node.js $(node --version) already installed — skipping."
else
    info "Installing Node.js 20 LTS…"
    curl -fsSL https://deb.nodesource.com/setup_20.x | bash -
    apt-get install -y -q nodejs
fi

# ── 4. Docker Engine + Compose v2 plugin ─────────────────────────────────────
if ! command -v docker &>/dev/null; then
    info "Installing Docker Engine…"
    install -m 0755 -d /etc/apt/keyrings
    curl -fsSL https://download.docker.com/linux/ubuntu/gpg \
        | gpg --dearmor -o /etc/apt/keyrings/docker.gpg
    chmod a+r /etc/apt/keyrings/docker.gpg
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.gpg] \
https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" \
        > /etc/apt/sources.list.d/docker.list
    apt-get update -q
    apt-get install -y -q docker-ce docker-ce-cli containerd.io docker-compose-plugin
    systemctl enable --now docker
else
    info "Docker already installed — skipping."
fi

# ── 5. Deploy user ────────────────────────────────────────────────────────────
DEPLOY_USER=voltana
if id "$DEPLOY_USER" &>/dev/null; then
    info "User $DEPLOY_USER already exists — skipping."
else
    info "Creating deploy user $DEPLOY_USER…"
    useradd --system --no-create-home --shell /usr/sbin/nologin "$DEPLOY_USER"
fi
# Always ensure voltana is in the docker group (idempotent)
usermod -aG docker "$DEPLOY_USER"
info "$DEPLOY_USER added to docker group."

# ── 6. Deploy directory ───────────────────────────────────────────────────────
DEPLOY_DIR=/opt/voltana
if [[ ! -d "$DEPLOY_DIR" ]]; then
    info "Creating $DEPLOY_DIR…"
    mkdir -p "$DEPLOY_DIR"
    chown "$DEPLOY_USER:$DEPLOY_USER" "$DEPLOY_DIR"
else
    info "$DEPLOY_DIR already exists — skipping."
fi

# ── 7. UFW firewall ───────────────────────────────────────────────────────────
info "Configuring UFW firewall…"
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp   comment 'SSH'
ufw allow 80/tcp   comment 'HTTP'
ufw allow 443/tcp  comment 'HTTPS'
ufw --force enable
info "UFW enabled. Active rules:"
ufw status verbose

# ── 8. certbot /var/www/certbot webroot dir ───────────────────────────────────
mkdir -p /var/www/certbot
info "Created /var/www/certbot (ACME webroot)."

# ── 9. Done ───────────────────────────────────────────────────────────────────
echo ""
info "Bootstrap complete."
warn "Next steps:"
warn "  1. Clone repo to $DEPLOY_DIR (or git pull if already cloned)."
warn "  2. Copy .env.example to $DEPLOY_DIR/.env and fill in all values."
warn "  3. Set DOMAIN= in the .env file."
warn "  4. Obtain TLS certificate (while ports 80/443 are free):"
warn "       certbot certonly --standalone -d \$DOMAIN"
warn "  5. Install and enable the systemd service:"
warn "       cp $DEPLOY_DIR/infra/systemd/voltana.service /etc/systemd/system/"
warn "       systemctl daemon-reload"
warn "       systemctl enable --now voltana"
warn "  6. Run the first deploy:"
warn "       sudo -u $DEPLOY_USER bash $DEPLOY_DIR/scripts/deploy.sh"
warn "  See docs/DEPLOY.md for the full step-by-step guide."
