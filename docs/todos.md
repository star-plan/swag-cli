# Todos

- [x] 增加 reload 命令，重启swag容器
- [x] 增加测试功能，可以测试已配置的站点，是否可以正常访问
    测试链路：
    - [x] swag容器 -> 目标容器
    - [x] 已配置域名的可访问性

## 测试薄弱点

- [ ] `internal/docker` 已补容器筛选、`Exec`、`ReloadNginx`、`RestartContainer` 基础单测；仍需覆盖 Docker 连接回退路径
- [ ] `internal/tui` 覆盖率极低；需要补齐主页管理、配置显示、列表保护等高风险交互逻辑
- [ ] `internal/cli` 覆盖率偏低；`test` 相关辅助逻辑已补，但仍需覆盖 `add` / `toggle` / `homepage` / `reload` / `config` 的关键行为分支
- [x] `internal/config` 需要补 `DefaultSiteConfPath` 双路径兼容、路径展开与 key 归一化相关测试

## 功能缺失 / UX 优化

- [x] TUI 目前只有“设置主页”，缺少“清理主页”入口，与 CLI `homepage clear` 不对齐
- [x] `test` 命令缺少按单站点筛选能力，排障时必须全量跑
- [x] `test` 命令失败时缺少更具体的错误上下文，诊断成本偏高
- [x] TUI 与 CLI 能力仍未完全对齐，例如 TUI 没有显式的 `reload` / `swag export` 入口
- [ ] 输出语言中英混合，长期看建议统一语言风格

## 推荐执行顺序

1. 先补 TUI 主页清理能力
2. 再增强 `test` 命令的可筛选性与诊断输出
3. 然后补 `internal/cli` / `internal/docker` 的高价值测试
