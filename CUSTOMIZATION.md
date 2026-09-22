# 本次二开说明

## 基线

2026-09-22 重新基于官方 Wei-Shaw/sub2api 的 main：
`1c0a69c0ceddb2fd21581c17ab09f6c500b89ba1`（2026-09-21）。

原 fork 的旧二开不再继承。当前新增内容只有审核自定义配置和柔和紫色主题，官方布局、登录流程及其他业务功能沿用上游。

## 在哪里填写两块内容

1. 管理后台 → 安全审计 → 内容审计 → 内容审计设置。
2. 「基础」中选择 **OpenAI Compatible**，填写审核服务商的 Base URL、模型 ID 和 API Key。
3. 「自定义审核」中选择 **Chat Completions**。
4. 将系统提示词粘贴到「系统提示词」。
5. 将 JavaScript payload 代码粘贴到「请求 payload 代码」。支持原示例的 `const wrappedUserContent = ...; const requestBody = ...;` 写法，无须改成函数。
6. 回到「基础」运行审核测试，确认结果后保存。测试使用当前未保存的提示词、代码、模型、Key 和阈值。

这两段内容存入现有 settings 配置，无需新增数据库迁移。切换到 TypeSafe AI 后，OpenAI 兼容引擎的配置仍会保留。

### payload 变量

| 变量 | 内容 |
| --- | --- |
| `text` | 网关提取的待审核文字 |
| `input` | 原生审核输入，可能是字符串或含 text / image_url 的数组 |
| `config.model` | 基础设置中的模型 ID |
| `config.auditPrompt` | 系统提示词编辑框内容 |
| `isModerationEndpoint` | 选择 Moderations 时为 true，Chat Completions 时为 false |
| `requestBody` | 脚本必须定义的结果，可以是 JSON 字符串或对象 |

脚本只负责构造请求体，后端沿用现有 Key、代理、超时和重试设置发送。不要把 Key 写入脚本。无需 fetch、require 或文件操作；这些 API 不开放。每次运行独立，执行限时 100 ms、栈深度 256，脚本及提示词各限制 64 KiB，请求体限制 16 MiB。脚本仅允许管理员配置，解释器不是进程级内存沙箱。

payload 留空时使用内置请求构造：系统提示词加包裹过的待审内容，temperature 为 0，不开启流式输出。默认构造会转发图片输入；自定义脚本若只使用 text，则不会转发图片。

「填入 payload 示例」提供可编辑的请求构造示例，不包含审核政策。具体规则由管理员填写。自定义脚本的 requestBody 必须包含 model，聊天接口还必须包含非空 messages；stream 只能省略或为 false。

### 返回格式

支持系统提示词示例的：

```json
{"confidence": 0.90, "reason": "命中自定义审核规则"}
```

也支持原 payload 所要求的：

```json
{"flagged": true, "reason": "命中自定义审核规则"}
```

- confidence 必须是 0–1 的数值，达到「拦截置信度阈值」即命中。默认 0.85，可自行调整。
- 只有 flagged 时直接使用布尔值；两者同时返回时以 confidence 为准。
- 聊天模型结果记为 custom 类别；原来的分类阈值继续用于 Moderations / TypeSafe AI。
- reason 会显示在测试结果和审核记录详情，并进行现有敏感信息脱敏。
- 无效 JSON、空结果、截断、拒答、脚本错误或超时均视为审核异常，不冒充“审核通过”。是否继续转发沿用上游既有的失败放行策略。
- 原生 Moderations 和 TypeSafe AI 的审核逻辑仍可使用。

聊天接口 Base URL 支持主机根地址（自动补 /v1/chat/completions）、以 /v1 或自定义路径结尾的基地址，以及完整 /chat/completions 地址。不要在 URL 中填写 Key。

原有的抽样比例、提取最新输入、输入长度限制、观察 / 前置拦截、关键词、缓存、通知及封禁策略仍然适用。提示词不是审核效果保证，需要用实际业务样本校准。

## 主题

通过 Tailwind 共享颜色调整主色、灰阶、深色背景、阴影和登录背景，保留官方布局和功能。使用原有系统字体栈，没有增加字体文件。

## 构建与上线

仍使用官方项目的构建方式，前端使用 **pnpm 9**、后端使用 **Go 1.27.0**。例如在仓库根目录构建本地镜像：

```sh
docker build -t sub2api:custom .
```

部署时将原服务的 image 改为自己构建的镜像。继续使用官方预构建镜像不会包含这些修改。本次仅更新代码，未修改运行中的服务器、数据或镜像。

## 验证

- Go 内容审核、配置、代理、引擎及管理接口相关测试。
- 新增脚本执行 / 超时、返回解析、配置持久化、引擎切换和网关拦截测试。
- 前端 lint、TypeScript 类型检查、生产构建。
- Linux amd64 后端编译通过（包含前端静态资源）。
- 审核页面与翻译测试 19 项；Makefile 指定的前端关键测试 274 项。
- 1440 px 桌面浅色 / 深色、390 px 手机浏览器检查，无页面异常或编辑器横向溢出。
- 视觉预览使用本地模拟数据；未使用真实审核模型 Key，因此未验证具体服务商的审核质量或延迟。

新增 JavaScript 解释器采用 [Goja](https://github.com/dop251/goja)，版本及依赖校验保存在 go.mod / go.sum。
