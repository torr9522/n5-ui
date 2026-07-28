# Development Skill Tree

## Backend

- Go
- Gin
- Go embed
- SQLite 数据库
- systemd 部署与日志排查

## Frontend

- Vue 2
- HTML template
- Ant Design Vue
- 前端状态与弹窗交互
- 前端 UAT / Headless Chromium

## Xray

- Xray inbound 配置
- VMess 分享链接格式
- VLESS 分享链接格式
- Trojan 分享链接格式
- Shadowsocks 分享链接格式
- TLS / SNI / Host Header 差异

## Release And Operations

- GitHub 发布流程
- Git 分支与回滚
- 安装脚本风险点
- Go build 产物验证
- systemd 服务安装与 reboot 验证

## Required Verification Skills

- 检查 Go embed 后 binary 是否包含最新 UI 字符串。
- 使用浏览器真实路径验证复制链接和二维码。
- 区分数据库配置、Xray 运行配置、分享链接临时导出字段。
- 验证 install.sh、release asset、源码 build 三种安装路径的差异。
