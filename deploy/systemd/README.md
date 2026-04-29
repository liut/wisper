# systemd User Service

A user-level systemd service for running webpawm in HTTP + SSE mode.

## Installation

```bash
# Create user-level systemd directory
mkdir -p ~/.config/systemd/user/

# Copy the service file
cp deploy/systemd/webpawm.service ~/.config/systemd/user/

# Create log directory (required when ProtectHome=read-only)
mkdir -p ~/.local/share/webpawm/

# Reload and enable the service
systemctl --user daemon-reload
systemctl --user enable --now webpawm.service
```

## Common Commands

```bash
# Check status
systemctl --user status webpawm.service

# View logs
journalctl --user -u webpawm.service -f

# Restart
systemctl --user restart webpawm.service

# Stop and disable
systemctl --user disable --now webpawm.service
```

## Configuration

Edit `~/.config/webpawm/config.json` and restart the service to apply changes.
