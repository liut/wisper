# systemd User Service

A user-level systemd service for running webpawm in HTTP + SSE mode.

## Installation

```bash
mkdir -p ~/.config/systemd/user/ ~/.local/share/webpawm/
cp deploy/systemd/webpawm.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now webpawm.service
```

## Usage

```bash
systemctl --user status webpawm     # status
journalctl --user -u webpawm -f     # logs
systemctl --user restart webpawm    # restart
```

Default endpoints at `localhost:8087`: `/mcp/stream` (HTTP), `/mcp/sse` (SSE).

Edit `~/.config/webpawm/config.json` and restart to apply changes.
