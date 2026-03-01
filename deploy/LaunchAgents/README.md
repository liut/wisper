# Wisper macOS LaunchAgent

本目录包含用于在 macOS 系统启动时自动运行 Wisper 服务的 LaunchAgent 配置文件。

## 文件说明

- `net.wisper.mcp-web.plist` - MCP Web 服务的启动配置文件

## 安装方法

### 1. 复制配置文件

```bash
mkdir -p ~/Library/LaunchAgents
cp net.wisper.mcp-web.plist ~/Library/LaunchAgents/
```

### 2. 自定义配置（可选）

编辑 plist 文件，根据需要修改以下配置：

- `ProgramArguments` - 命令行参数，可修改监听地址和端口
- `EnvironmentVariables` - 环境变量，如代理设置, 如果不需要可以删除这块
- `StandardOutPath` / `StandardErrorPath` - 日志输出路径

**注意**：如果修改了路径，需要将 `~/Library/Logs/` 改为实际的用户目录。

### 3. 加载服务

```bash
launchctl load ~/Library/LaunchAgents/net.wisper.mcp-web.plist
```

### 4. 启动服务

```bash
launchctl start net.wisper.mcp-web
```

## 管理命令

```bash
# 查看服务状态
launchctl list | grep wisper

# 停止服务
launchctl stop net.wisper.mcp-web

# 重新加载配置
launchctl unload ~/Library/LaunchAgents/net.wisper.mcp-web.plist
launchctl load ~/Library/LaunchAgents/net.wisper.mcp-web.plist

# 卸载服务
launchctl unload ~/Library/LaunchAgents/net.wisper.mcp-web.plist
rm ~/Library/LaunchAgents/net.wisper.mcp-web.plist
```

## 查看日志

```bash
# 实时查看输出日志
tail -f ~/Library/Logs/wisper-mcp-web.out.log

# 查看错误日志
tail -f ~/Library/Logs/wisper-mcp-web.err.log
```

## 注意事项

1. 确保 wisper 二进制文件路径正确（当前配置为 `/usr/local/bin/wisper`）
2. 如果使用代理，请确保代理地址和端口配置正确
3. 服务将在用户登录后自动启动
