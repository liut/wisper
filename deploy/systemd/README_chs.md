# systemd 用户服务

Linux 用户级 systemd 服务，用于管理 webpawm 的 HTTP + SSE 模式。

## 安装

```bash
# 创建用户级 systemd 目录
mkdir -p ~/.config/systemd/user/

# 复制 service 文件
cp deploy/systemd/webpawm.service ~/.config/systemd/user/

# 创建日志目录（ProtectHome=read-only 时需要显式放行写入路径）
mkdir -p ~/.local/share/webpawm/

# 重载并启用服务
systemctl --user daemon-reload
systemctl --user enable --now webpawm.service
```

## 常用操作

```bash
# 查看状态
systemctl --user status webpawm.service

# 查看日志
journalctl --user -u webpawm.service -f

# 重启
systemctl --user restart webpawm.service

# 停止并禁用
systemctl --user disable --now webpawm.service
```

## 配置

修改 `~/.config/webpawm/config.json` 后重启服务生效。
