# 二开评估与官方 v0.2.15 升级记录

评估时间：2026-10-09（北京时间）。目标为保留当前 fork 的二开功能，将官方最新 `main` 合入 fork `main`。本次仅更新源码；镜像发布和服务器部署分别执行。

## 升级基准

- 原 fork `main`：`1b4b319e8d6c72aefe43d733e0bb63b8285483fb`，官方共同基线为 `3040209f205472038c1ba745a1bedd2edd9053b1`（v0.2.13）。
- 本次官方 `main`：`3a6fd1c9db07203ca308aaba69e502bc1f35b307`，`backend/cmd/server/VERSION` 为 `0.2.15`。该提交也是评估时官方 `main` 的最新提交。
- 共同基线后，官方修改 301 个文件，fork 修改 75 个文件，重叠 5 个文件。70 个 fork 独有改动路径中，69 个在合并后与原 fork 内容一致；例外是按新基线更新的 `.github/fork.json`。
- 采用保留双方历史的 Git 三方合并。只有 `backend/go.mod` 出现文本冲突；保留 fork 的 goja/sourcemap 依赖，同时采用官方 Go 1.27.2 与更新的依赖版本。`backend/go.sum` 自动合并。

官方 [v0.2.14](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.14) 修复新安装管理员账号可预测及 EasyPay 回调伪造漏洞；[v0.2.15](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.15) 加入 Cline、Command Code 平台和平台清单，并修复协议转换、计费、WebSocket 用量及多处前端竞态。

## 二开保留与适配

| 范围 | 合并后的判断 |
| --- | --- |
| 多渠道内容审核、上下文、强审和预算 | fork 专用的审核服务、账本、迁移、管理接口和配置页面未被官方覆盖。原有 `240_fork_content_moderation_v2.sql` 保留。 |
| 新平台与审核入口 | 官方将 Cline、Command Code 注册到 OpenAI 网关；该网关的 Responses、Messages 处理路径仍在计费/转发前执行 `checkSecurityAudit`。已有审核顺序测试和平台路由测试覆盖相关接点；真实运营商效果仍需部署后试跑。 |
| 模型价格目录 | fork 的 models.dev 查价和 `modelsDevModel.Cost` 保留；自动合并的模型探测逻辑同时采用官方多协议账号判断。 |
| 柔和紫色主题 | fork 的主题文件与官方此次变更没有交叉，内容原样保留。 |
| 二开发行、更新源与 CI | fork 的工作流、版本规划和测试仍在；官方 Go 检查同步至 1.27.2、golangci-lint 至 v2.14.0。`.github/fork.json` 指向本次官方提交。 |

官方新增 `242_drop_platform_check_constraints.sql`，仅移除平台配额和组合路由表的两个平台 CHECK 约束，后续由应用层平台清单校验。它不删除 fork 的审核表，也不与 fork 的 `240` 号迁移冲突。部署前仍应备份数据库。全新自动安装需满足官方新的 `ADMIN_PASSWORD` 8–72 字节要求；已有用户的部署不受该安装条件阻断。

## 验证与边界

- 合并索引无未解决冲突，`git diff --cached --check` 通过；fork 版本规划 Python 测试 6 项通过。
- 前端 ESLint、TypeScript、关键测试与完整 Vitest 套件（365 个文件、2826 项），以及生产构建通过。
- Apple Container 生命周期、Compose 安全、Gateway 环境、运行资源和 Caddyfile 脚本检查通过。本机没有 Docker，简易 Compose 检查留给 CI。
- 本机没有 Go；后端单元、集成与静态检查以升级 PR 的 GitHub Actions 结果为准，通过后再合入 fork `main`。
- 本次没有连接线上数据库、真实支付或审核服务。源码合并不会自动发布 fork 镜像，也不会替换服务器容器。
