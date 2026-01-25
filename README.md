# SWAG CLI - Nginx 反向代理配置助手

一个专为 [LinuxServer.io SWAG](https://docs.linuxserver.io/general/swag) (Secure Web Application Gateway) 容器设计的命令行管理工具。

旨在简化 Nginx 反向代理配置的生成、管理和维护流程。通过简单的 CLI 命令或交互式向导，自动发现 Docker 容器并生成对应的反向代理配置。

## ✨ 主要功能

- **🤖 自动发现**: 自动检测指定 Docker 网络中的运行容器，无需手动查找 IP 或端口。
- **📝 配置生成**: 基于最佳实践模板，自动生成 `.subdomain.conf` 反向代理配置文件。
- **🖥️ 交互式向导**: 提供友好的 TUI (终端界面) 引导用户完成站点添加和管理。
- **🔌 站点管理**:
  - `list`: 查看所有已配置站点及其关联容器的实时状态 (在线/离线)。
  - `toggle`: 快速启用或禁用特定站点 (无需删除文件)。
  - `test`: 内置连接性测试，检查 SWAG 到目标容器的连通性以及外部访问状态。
- **🔄 自动重载**: 操作完成后自动重启 SWAG 容器以应用更改。

## 🛠️ 安装说明

### 前置要求

- **Go**: 1.25 或更高版本
- **Docker**: 本机需安装 Docker 且当前用户有权限访问 Docker Socket (通常需加入 `docker` 用户组)。
- **SWAG**: 需有一个正在运行的 [SWAG 容器](https://docs.linuxserver.io/images/docker-swag)。

### 源码编译安装

1. 克隆仓库:
   ```bash
   git clone https://github.com/your-username/swag-cli.git
   cd swag-cli
   ```

2. 编译并安装:
   ```bash
   go install ./cmd/swag-cli
   ```

3. 验证安装:
   ```bash
   swag-cli --version
   ```

## 🚀 使用指南

### 1. 初始化配置 (推荐)

首次使用建议设置全局配置，避免每次命令重复输入参数。

```bash
# 设置 SWAG 配置目录 (由 docker-compose 映射的 config 卷路径)
swag-cli config set swag-dir /path/to/your/appdata/swag

# 设置 SWAG 容器名称 (默认为 swag)
swag-cli config set swag-container swag

# 设置 Docker 网络名称 (swag 和其他容器所在的网络，默认为 swag)
swag-cli config set network swag
```

你也可以一键导出/导入这份全局配置，用于多机器迁移或备份恢复：

```bash
# 导出到文件（默认导出到当前目录并带时间戳文件名）
swag-cli config export

# 导出到指定文件
swag-cli config export ./swag-cli.config.json

# 导出到 stdout（可用于重定向/管道）
swag-cli config export --stdout > swag-cli.config.json

# 从文件导入（默认会展示变更并要求确认；可用 -y 跳过确认）
swag-cli config import ./swag-cli.config.json
swag-cli config import -y ./swag-cli.config.json
```

### 2. 交互模式 (TUI)

直接运行命令不带参数，即可进入交互式向导模式：

```bash
swag-cli
```
在交互模式下，你可以：
- 从列表中选择容器添加代理
- 查看当前所有站点状态
- 启用/禁用/删除现有站点

### 3. 下行指令模式 (CLI)

**添加新站点**
```bash
# 基本用法 (默认使用容器名作为子域名)
swag-cli add my-app

# 指定子域名和端口
swag-cli add my-app --subdomain app --port 8080 --proto http
```

**设置根域名主页 (Homepage / Root Domain)**
```bash
# 将 example.com 的主页反代到容器 my-app:8080
swag-cli homepage set my-app --domain example.com --port 8080 --proto http

# 只计算变更，不写入（用于确认定位的文件与修改逻辑）
swag-cli homepage set my-app --domain example.com --port 8080 --proto http --dry-run

# 清理主页反代，恢复 default 的 try_files 行为（并恢复 server_name 为 '_'）
swag-cli homepage clear --restore-server-name-underscore
```
说明：
- 子域名（如 `a.example.com`）仍通过 `config/nginx/proxy-confs/*.subdomain.conf` 管理（`add/toggle/list`）。  
- 根域名主页通过修改 `config/nginx/site-confs/default`（或兼容路径 `site-conf/default`）的 `location /` 来实现。
- 工具会在 `config/nginx/site-confs/.bak/` 下自动保存 default 的备份，避免被 `include /config/nginx/site-confs/*;` 误加载。

**列出所有站点**
```bash
swag-cli list
```
*输出将显示配置类型、目标地址以及容器的运行状态。*

**测试连通性**
```bash
swag-cli test
```
*检查内部容器连通性 (SWAG -> 目标容器) 和外部 URL 可访问性。*

**启用/禁用站点**
```bash
swag-cli toggle my-app
```

**重启 SWAG**
```bash
swag-cli reload
```

## ⚙️ 命令帮助

查看任何命令的详细帮助信息：
```bash
swag-cli help [command]
```

## 🤝 贡献参与

欢迎提交 Issue 或 Pull Request 来改进此项目！

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

## 📄 许可证

本项目采用 MIT 许可证。详见 [LICENSE](LICENSE) 文件。
