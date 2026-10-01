# 二开评估与官方 v0.2.11 升级记录

评估时间：2026-10-01（北京时间）。升级对象为 `xixi-furry/sub2api` 的当前 `main`，原发布版本为 `v0.2.10-fix1`。

## 升级基线与合并

- 升级前 fork `main`：`1e067711936ad26bdaf9040b7d814b647075dd7e`，已包含官方 `v0.2.10` 和本仓库审核、预算、主题定制。
- 前次官方基线：`a60a29549f488a854966aaec9541abbe006cac22`。
- 本次合入官方最新 `main`：`d6adebd22de00478cd021119ba755f37bcb94fb5`，`backend/cmd/server/VERSION` 为 `0.2.11`。官方 [v0.2.11 Release](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.11) 标签指向 `96f4c115c9749078f90cbf210a01d39baf3f53b6`；本次还包含其后主分支的 Axios 1.20.0 与 Grok CLI 修复。
- 两边从前次基线出发分别修改 101、73 个文件，仅 `backend/internal/server/routes/admin.go` 重叠。普通三方 Git merge 自动完成，无文本冲突。官方 Claude 限额重置兑换路由和本仓库风控 v2 路由均保留。
- `.github/fork.json` 已更新到本次官方提交，供后续自动同步与 fork 版本规划使用。本次无官方数据库迁移。

## 新版功能与二开适配

| 范围 | 评估结果 |
| --- | --- |
| 审核提示词、payload 与多渠道审核 | 官方没有修改 fork 专用的 `content_moderation_v2_*`、审核账本和风控设置页面。网关的 `checkSecurityAudit` 仍位于新增余额在途预占之前；审核拦截不会占用用户余额预留。现有审核配置不会因源码升级自动启用或清空。 |
| 审核预算与余额计费 | fork 的 `fork_moderation_*` 审核预算与上游用户余额预占分别记账。上游预占默认开启，用于减少余额模式并发透支；低余额用户的并发请求可能更早被拒绝。实际运营配置可通过 `billing.inflight_reservation` 调整。 |
| API Key 限制 | 上游默认限制单用户最多 200 个有效 Key、每小时最多创建 60 次。已有大量 Key 的用户仍可使用现存 Key，但达到上限后不能继续创建；可通过 `api_key_create.max_active_per_user` / `max_per_user_per_hour` 调整，`0` 表示不限制。 |
| 账号、模型与客户端 | 保留官方 Claude 原生限额重置兑换、GPT-6.1 Sol、Codex 远程模型目录、套餐识别与 Grok CLI 修复。新增管理路由沿用管理鉴权和审计保护。 |
| 柔和紫色主题 | 官方本轮没有修改 fork 主题相关文件，合并后原样保留。 |
| 发行及更新源 | fork 的更新源隔离、版本规划和双架构发布工作流没有被上游覆盖；本次只更新源码基线，不自动发布镜像或替换服务器容器。 |

## 验证与边界

- `git diff --check` 无空白错误；fork 版本规划测试 6 项通过。
- 前端 ESLint、TypeScript 检查、完整 Vitest 套件（338 个文件、2596 项）与生产构建通过。完整套件中发现一组原有订阅套餐卡片测试仍按旧版 i18n 翻译键断言，已改为校验实际显示的翻译；组件业务逻辑未变。
- 本机 Apple Container、Compose 安全、Gateway 环境、运行资源和 Caddyfile 脚本检查通过。简易 Compose 检查需要本机缺少的 Docker，由仓库 CI 验证。
- 本机缺少 Go 和 Docker。后端单元、集成、静态检查及简易 Compose 检查以本次升级 PR 的 GitHub Actions 结果为准，全部通过后才可更新远端 `main`。
- 本次没有连接线上数据库、真实审核模型或 Claude 账号。源码和模拟回归不能替代升级后的后台试跑；发布、镜像更新及服务器部署均是独立步骤。
