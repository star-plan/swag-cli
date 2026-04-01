# Todos

- [x] 增加 reload 命令，重启swag容器
- [x] 增加测试功能，可以测试已配置的站点，是否可以正常访问
    测试链路：
    - [x] swag容器 -> 目标容器
    - [x] 已配置域名的可访问性

## 测试薄弱点

- [x] `internal/docker` 已补容器筛选、Docker 连接回退、`Exec`、`ReloadNginx`、`RestartContainer` 关键路径单测
- [x] `internal/tui` 已补主页管理菜单、配置显示合并、列表保护与主菜单能力入口等高风险交互逻辑测试
- [x] `internal/cli` 已补 `test` / `config` / `add` / `toggle` / `homepage` / `reload` 相关高价值辅助逻辑与关键行为分支测试
- [x] `internal/config` 需要补 `DefaultSiteConfPath` 双路径兼容、路径展开与 key 归一化相关测试

## 功能缺失 / UX 优化

- [x] TUI 目前只有“设置主页”，缺少“清理主页”入口，与 CLI `homepage clear` 不对齐
- [x] `test` 命令缺少按单站点筛选能力，排障时必须全量跑
- [x] `test` 命令失败时缺少更具体的错误上下文，诊断成本偏高
- [x] TUI 与 CLI 能力仍未完全对齐，例如 TUI 没有显式的 `reload` / `swag export` 入口
- [x] 已将主要 TUI 菜单与 `test` 命令用户输出统一为中文风格

## 推荐执行顺序

1. 先补 TUI 主页清理能力
2. 再增强 `test` 命令的可筛选性与诊断输出
3. 然后补 `internal/cli` / `internal/docker` 的高价值测试
