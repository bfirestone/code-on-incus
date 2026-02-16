#!/bin/bash
set -euo pipefail

CODE_USER="bfirestone"
CODE_UID=1000

log() { echo "[coi-desktop] $*"; }

configure_dns_if_needed() {
    log "Checking DNS configuration..."
    if getent hosts archlinux.org > /dev/null 2>&1; then
        log "DNS resolution works."
        return 0
    fi
    log "DNS resolution failed, configuring static DNS..."
    rm -f /etc/resolv.conf
    cat > /etc/resolv.conf << 'EOF'
nameserver 8.8.8.8
nameserver 8.8.4.4
nameserver 1.1.1.1
EOF
    log "Static DNS configured."
}

install_base_dependencies() {
    log "Installing base dependencies..."
    pacman -Syu --noconfirm
    pacman -S --noconfirm --needed \
        curl wget git ca-certificates jq unzip sudo \
        tmux base-devel openssl readline zlib \
        libffi libyaml gmp sqlite postgresql-libs \
        libxml2 libxslt docker docker-compose \
        github-cli nodejs npm python
}

create_user() {
    log "Creating user $CODE_USER..."
    useradd -m -u "$CODE_UID" -G wheel,docker "$CODE_USER"
    echo "$CODE_USER ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/"$CODE_USER"
    chmod 440 /etc/sudoers.d/"$CODE_USER"
    mkdir -p "/home/$CODE_USER/.claude" "/home/$CODE_USER/.ssh"
    chmod 700 "/home/$CODE_USER/.ssh"
    chown -R "$CODE_USER:$CODE_USER" "/home/$CODE_USER"
    log "User '$CODE_USER' created (uid: $CODE_UID)"
}

configure_power_wrappers() {
    log "Configuring power management wrappers..."
    for cmd in shutdown poweroff reboot halt; do
        cat > "/usr/local/bin/${cmd}" << 'WRAPPER_EOF'
#!/bin/bash
exec sudo /usr/sbin/COMMAND_NAME "$@"
WRAPPER_EOF
        sed -i "s/COMMAND_NAME/${cmd}/" "/usr/local/bin/${cmd}"
        chmod 755 "/usr/local/bin/${cmd}"
    done
}

install_claude_cli() {
    log "Installing Claude CLI..."
    su - "$CODE_USER" -c 'curl -fsSL https://claude.ai/install.sh | bash'
    local CLAUDE_PATH="/home/$CODE_USER/.local/bin/claude"
    if [[ ! -x "$CLAUDE_PATH" ]]; then
        log "ERROR: Claude CLI not found at $CLAUDE_PATH"
        exit 1
    fi
    ln -sf "$CLAUDE_PATH" /usr/local/bin/claude
    log "Claude CLI installed"
}

install_dummy() {
    log "Installing dummy..."
    if [[ -f /tmp/dummy ]]; then
        cp /tmp/dummy /usr/local/bin/dummy
        chmod +x /usr/local/bin/dummy
        rm /tmp/dummy
    else
        log "No dummy found, skipping (optional for custom builds)"
    fi
}

cleanup() {
    log "Cleaning up..."
    pacman -Scc --noconfirm
}

main() {
    log "Starting coi-desktop image build (Arch Linux)..."
    configure_dns_if_needed
    install_base_dependencies
    create_user
    configure_power_wrappers
    install_claude_cli
    install_dummy
    cleanup
    log "coi-desktop image build complete!"
}

main "$@"
