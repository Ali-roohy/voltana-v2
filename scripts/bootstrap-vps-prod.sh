#!/usr/bin/env bash
# bootstrap-vps-prod.sh — Idempotent first-run setup for Voltana on Ubuntu 24.04 LTS.
# Run as root: sudo bash scripts/bootstrap-vps-prod.sh
# Safe to re-run at any time (all steps are guarded by existence checks).
set -euo pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; NC='\033[0m'
info()  { echo -e "${GREEN}[bootstrap]${NC} $*"; }
warn()  { echo -e "${YELLOW}[bootstrap]${NC} $*"; }
abort() { echo -e "${RED}[bootstrap] ERROR:${NC} $*" >&2; exit 1; }

# ── 0. Require root ───────────────────────────────────────────────────────────
[[ $EUID -eq 0 ]] || abort "Run as root: sudo bash scripts/bootstrap-vps-prod.sh"

# ── 1. Ubuntu version check ───────────────────────────────────────────────────
UBUNTU_VER=$(. /etc/os-release && echo "$VERSION_ID")
if [[ "$UBUNTU_VER" != "24.04" ]]; then
    warn "This script targets Ubuntu 24.04 LTS; detected $UBUNTU_VER."
    warn "Continuing anyway — review any package name differences yourself."
fi

# ── 2. System packages ────────────────────────────────────────────────────────
info "Updating package index…"
apt-get update -q

info "Installing base packages…"
apt-get install -y -q \
    ca-certificates curl gnupg lsb-release \
    ufw git gettext-base certbot python3-certbot-nginx \
    unzip awscli

# ── 3. Node.js 20 (LTS) via NodeSource ───────────────────────────────────────
NODE_OK=false
if command -v node &>/dev/null; then
    NODE_MAJOR=$(node --version | sed 's/v\([0-9]*\).*/\1/')
    [[ $NODE_MAJOR -ge 20 ]] && NODE_OK=true
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
    info "Docker $(docker --version | awk '{print $3}' | tr -d ',') already installed — skipping."
fi

# ── 5. Deploy user ────────────────────────────────────────────────────────────
DEPLOY_USER=voltana
if id "$DEPLOY_USER" &>/dev/null; then
    info "User $DEPLOY_USER already exists — skipping."
else
    info "Creating deploy user $DEPLOY_USER…"
    useradd --system --no-create-home --shell /usr/sbin/nologin "$DEPLOY_USER"
fi
usermod -aG docker "$DEPLOY_USER"
info "$DEPLOY_USER added to docker group."

# ── 6. Deploy directory ───────────────────────────────────────────────────────
DEPLOY_DIR=/opt/voltana
if [[ ! -d "$DEPLOY_DIR" ]]; then
    info "Creating $DEPLOY_DIR…"
    mkdir -p "$DEPLOY_DIR"
fi
chown "$DEPLOY_USER:$DEPLOY_USER" "$DEPLOY_DIR"

# ── 7. Data directories (postgres bind-mount + backup store) ─────────────────
DATA_DIR=/var/lib/voltana
POSTGRES_DIR="$DATA_DIR/postgres"
BACKUP_DIR="$DATA_DIR/backups"
MAIL_DIR="$DATA_DIR/mail"

for DIR in "$DATA_DIR" "$POSTGRES_DIR" "$BACKUP_DIR" "$MAIL_DIR"; do
    if [[ ! -d "$DIR" ]]; then
        info "Creating $DIR…"
        mkdir -p "$DIR"
    fi
done
chown -R "$DEPLOY_USER:$DEPLOY_USER" "$DATA_DIR"
info "Data directories ready: $POSTGRES_DIR · $BACKUP_DIR · $MAIL_DIR (Poste.io)"

# ── 8. UFW firewall ───────────────────────────────────────────────────────────
info "Configuring UFW firewall…"
ufw --force reset
ufw default deny incoming
ufw default allow outgoing
ufw allow 22/tcp   comment 'SSH'
ufw allow 80/tcp   comment 'HTTP'
ufw allow 443/tcp  comment 'HTTPS'
# Mail (Poste.io) — TLS/MX subset only. Plaintext POP3/IMAP (110/143/995) and the
# admin UI (8443) are bound to 127.0.0.1 in compose, so they are intentionally
# absent here. NOTE: Docker publishes ports into its own iptables chain and
# bypasses UFW; these allows document intent — the localhost binds are the real guard.
ufw allow 25/tcp   comment 'SMTP (MX)'
ufw allow 465/tcp  comment 'SMTPS'
ufw allow 587/tcp  comment 'SMTP submission (STARTTLS)'
ufw allow 993/tcp  comment 'IMAPS'
ufw --force enable
info "UFW enabled. Active rules:"
ufw status verbose

# ── 9. certbot webroot directory ──────────────────────────────────────────────
mkdir -p /var/www/certbot
info "Created /var/www/certbot (ACME webroot)."

# ── 10. Install systemd backup timer ─────────────────────────────────────────
UNIT_SRC="$DEPLOY_DIR/infra/systemd"
if [[ -f "$UNIT_SRC/voltana-backup.service" ]] && [[ -f "$UNIT_SRC/voltana-backup.timer" ]]; then
    info "Installing backup systemd units…"
    cp "$UNIT_SRC/voltana-backup.service" /etc/systemd/system/
    cp "$UNIT_SRC/voltana-backup.timer"   /etc/systemd/system/
    systemctl daemon-reload
    systemctl enable voltana-backup.timer
    info "voltana-backup.timer enabled (daily at 03:00 UTC)."
else
    warn "Backup unit files not found in $UNIT_SRC — skipping timer install."
    warn "Run this step again after cloning the repo."
fi

# ── 11. Done ──────────────────────────────────────────────────────────────────
echo ""
info "Bootstrap complete (Ubuntu $UBUNTU_VER)."
echo ""
warn "Next steps — follow docs/DEPLOY_PRODUCTION.md for the full guide:"
warn "  1. Clone repo to $DEPLOY_DIR (or git pull if already cloned)."
warn "  2. Copy .env.production.example to $DEPLOY_DIR/.env and fill in all values."
warn "  3. Obtain TLS certificate (while ports 80/443 are free); include www + mail SANs:"
warn "       certbot certonly --standalone -d \$DOMAIN -d www.\$DOMAIN -d mail.\$DOMAIN"
warn "  4. Install and enable the main systemd service:"
warn "       cp $DEPLOY_DIR/infra/systemd/voltana.service /etc/systemd/system/"
warn "       systemctl daemon-reload"
warn "       systemctl enable --now voltana"
warn "  5. Run the first deploy:"
warn "       bash $DEPLOY_DIR/scripts/deploy.sh"
warn "  6. Verify: curl https://\$DOMAIN/health  →  {\"status\":\"ok\"}"
warn "  7. Re-run bootstrap to install backup timer once the repo is cloned."
