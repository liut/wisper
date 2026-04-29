# systemd 用户服务

Linux 用户级 systemd 服务，用于运行 webpawm 的 HTTP + SSE 模式。

## 安装

```bash
mkdir -p ~/.config/systemd/user/ ~/.local/share/webpawm/
cp deploy/systemd/webpawm.service ~/.config/systemd/user/
systemctl --user daemon-reload
systemctl --user enable --now webpawm.service
```

## 常用操作

```bash
systemctl --user status webpawm     # 状态
journalctl --user -u webpawm -f     # 日志
systemctl --user restart webpawm    # 重启
```

默认端点 `localhost:8087`: `/mcp/stream` (HTTP), `/mcp/sse` (SSE)。

修改 `~/.config/webpawm/config.json` 后重启生效。
