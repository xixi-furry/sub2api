<template>
  <div class="fur-home" data-testid="fur-home" lang="zh-CN">
    <a class="fur-skip" href="#fur-main">跳至主要内容</a>
    <header class="fur-header">
      <nav class="fur-wrap fur-nav" aria-label="主导航">
        <RouterLink to="/home" class="fur-brand"
          ><img v-if="siteLogo" :src="siteLogo" alt="" /><span
            v-else
            class="fur-logo"
            aria-hidden="true"
            >&lt;/&gt;</span
          ><span>{{ siteName }}<b>.</b></span></RouterLink
        >
        <div class="fur-navlinks">
          <a href="#fur-tools">挑选工具</a><a href="#fur-start">接入指引</a
          ><a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            >站点文档 ↗</a
          ><a v-else href="#fur-faq">常见问题</a>
        </div>
        <div class="fur-nav-actions">
          <button
            type="button"
            class="fur-theme"
            :aria-label="isDark ? '切换为浅色主题' : '切换为深色主题'"
            @click="$emit('toggleTheme')"
          >
            <Icon :name="isDark ? 'sun' : 'moon'" size="md" /></button
          ><RouterLink
            :to="destination"
            class="fur-button fur-small"
            data-testid="fur-auth-entry"
            >{{
              isAuthenticated
                ? isAdmin
                  ? "管理后台"
                  : "进入控制台"
                : "登录 / 开始"
            }}<span aria-hidden="true">↗</span></RouterLink
          >
        </div>
      </nav>
    </header>

    <main id="fur-main">
      <section class="fur-wrap fur-hero" aria-labelledby="fur-hero-title">
        <div class="fur-hero-copy">
          <p class="fur-eyebrow">BUILT FOR YOUR NEXT IDEA</p>
          <h1 id="fur-hero-title">顺手的<br /><span>AI 接入入口。</span></h1>
          <p class="fur-lead">
            给爱折腾的小动物，一个专注创造的地方。<br
              class="fur-desktop-break"
            />接上熟悉的工具，继续写代码、做项目，把想法变成作品。
          </p>
          <div class="fur-actions">
            <RouterLink :to="destination" class="fur-button"
              >{{ isAuthenticated ? "回到我的控制台" : "开始使用"
              }}<span aria-hidden="true">→</span></RouterLink
            ><a class="fur-button fur-secondary" href="#fur-tools"
              >查看接入指引</a
            >
          </div>
          <p class="fur-hero-note">
            <span class="fur-note-line"></span> 熟悉的工作流，顺手的每一次接入。
          </p>
        </div>
        <FurCodeConnection />
      </section>

      <section
        class="fur-wrap fur-tools"
        id="fur-tools"
        aria-labelledby="fur-tools-title"
      >
        <div class="fur-section-heading">
          <div>
            <p class="fur-eyebrow">YOUR TOOLS, YOUR WAY</p>
            <h2 id="fur-tools-title">熟悉的工具，顺畅的开始。</h2>
          </div>
          <p>选一个你常用的工具，<br />看看如何连接 Fur Code。</p>
        </div>
        <div class="fur-tool-grid" role="group" aria-label="选择接入工具">
          <button
            v-for="tool in tools"
            :key="tool.id"
            type="button"
            class="fur-tool-card"
            :class="{ 'is-selected': selectedTool === tool.id }"
            :aria-pressed="selectedTool === tool.id"
            :aria-label="`查看 ${tool.name} 接入指引`"
            aria-controls="fur-tool-guide"
            @click="selectTool(tool.id)"
          >
            <span class="fur-tool-top"
              ><span
                class="fur-tool-symbol"
                :class="`symbol-${tool.id}`"
                aria-hidden="true"
                >{{ tool.symbol }}</span
              ><span class="fur-tool-arrow" aria-hidden="true">↗</span></span
            ><strong>{{ tool.name }}</strong
            ><span class="fur-tool-sub">{{ tool.subtitle }}</span
            ><span class="fur-tool-caption">{{ tool.caption }}</span>
          </button>
        </div>
        <div
          class="fur-tool-guide"
          id="fur-tool-guide"
          aria-live="polite"
          aria-atomic="true"
        >
          <div>
            <span class="fur-guide-kicker"
              >{{ activeTool.name }} · 接入提示</span
            >
            <p>{{ activeTool.guide }}</p>
            <a
              :href="activeTool.docs"
              target="_blank"
              rel="noopener noreferrer"
              class="fur-text-link"
              >查看工具官方配置说明 <span aria-hidden="true">↗</span></a
            >
          </div>
          <div class="fur-address">
            <label for="fur-base-url">{{ activeTool.addressLabel }}</label>
            <div>
              <input
                id="fur-base-url"
                :value="activeTool.baseUrl"
                readonly
                spellcheck="false"
                @focus="selectAddress"
              /><button
                type="button"
                :aria-label="`复制 ${activeTool.name} API 地址`"
                @click="copyAddress"
              >
                <Icon name="copy" size="sm" /><span>{{
                  copyState === "copied" ? "已复制" : "复制"
                }}</span>
              </button>
            </div>
            <span class="fur-address-note">{{
              copyState === "failed"
                ? "复制未成功，可选中上方地址手动复制。"
                : "配合对应分组的 API Key 与可用模型使用。"
            }}</span>
          </div>
        </div>
        <p class="fur-tool-footnote">
          以上是客户端接入入口；可用模型、协议与额度，以你在控制台选择的分组为准。
        </p>
      </section>

      <section
        class="fur-steps-section"
        id="fur-start"
        aria-labelledby="fur-start-title"
      >
        <div class="fur-wrap fur-steps-layout">
          <div>
            <p class="fur-eyebrow">FROM IDEA TO FIRST REQUEST</p>
            <h2 id="fur-start-title">简单三步，<br />开始你的项目。</h2>
            <p class="fur-section-copy">
              写代码、搭网站、做点有趣的小工具。<br />先连接，再让灵感往前跑。
            </p>
            <RouterLink
              :to="
                isAuthenticated
                  ? '/keys'
                  : { path: '/login', query: { redirect: '/keys' } }
              "
              class="fur-text-link"
              >{{ isAuthenticated ? "前往密钥管理" : "登录后创建密钥" }}
              <span aria-hidden="true">→</span></RouterLink
            >
          </div>
          <ol class="fur-step-list">
            <li>
              <span class="fur-step-number">01</span>
              <div>
                <h3>登录并创建密钥</h3>
                <p>
                  登录控制台，查看可用分组与计费说明，再创建属于这个项目的 API
                  Key。
                </p>
              </div>
            </li>
            <li>
              <span class="fur-step-number">02</span>
              <div>
                <h3>配置常用工具</h3>
                <p>
                  选择 Codex、Claude Code、OpenCode 或
                  Pi，按对应协议配置地址、密钥和模型。
                </p>
              </div>
            </li>
            <li>
              <span class="fur-step-number">03</span>
              <div>
                <h3>发个请求，开始创造</h3>
                <p>从一个小任务开始；调用记录和用量都可以回到控制台查看。</p>
              </div>
            </li>
          </ol>
        </div>
      </section>

      <section
        class="fur-wrap fur-faq-section"
        id="fur-faq"
        aria-labelledby="fur-faq-title"
      >
        <div>
          <p class="fur-eyebrow">GOOD QUESTIONS</p>
          <h2 id="fur-faq-title">开始之前，<br />你可能想了解。</h2>
          <p class="fur-section-copy">第一次来，也可以慢慢熟悉。</p>
        </div>
        <div class="fur-faq">
          <details>
            <summary>第一次使用，从哪里开始？</summary>
            <p>
              先登录，再到密钥管理创建 API
              Key。已有工具的话，可以在上方选择对应接入提示；还不确定使用哪个模型，就先查看控制台的分组与模型说明。
            </p>
          </details>
          <details>
            <summary>四种工具可以共用同一套配置吗？</summary>
            <p>
              不完全相同。Codex、Claude Code、OpenCode 与 Pi
              的配置字段、认证方式和接口协议各有区别。请选择对应工具的指引，并确认密钥分组支持所需模型。
            </p>
          </details>
          <details>
            <summary>有哪些模型，怎么计算用量？</summary>
            <p>
              模型范围与计费取决于所选分组和当前开放情况，请以控制台展示的信息为准。使用后可查看请求记录和用量统计。
            </p>
          </details>
          <details>
            <summary>配置好了却无法调用，怎么办？</summary>
            <p>
              依次核对 API 地址、密钥、模型 ID、分组权限和可用额度。不要把 API
              Key 发到公开聊天或截图里；排查时保留错误信息，隐藏完整密钥。
            </p>
          </details>
        </div>
      </section>

      <section class="fur-wrap fur-invitation">
        <div>
          <p class="fur-eyebrow">YOUR NEXT IDEA STARTS HERE</p>
          <h2>带上好奇心，剩下的边做边学。</h2>
          <p>你的下一个小作品，从这里开始。</p>
        </div>
        <RouterLink :to="destination" class="fur-button"
          >{{ isAuthenticated ? "进入控制台" : "现在就出发" }}
          <span aria-hidden="true">→</span></RouterLink
        >
      </section>
    </main>
    <footer class="fur-wrap fur-footer">
      <RouterLink to="/home" class="fur-brand"
        ><span class="fur-logo" aria-hidden="true">&lt;/&gt;</span
        ><span>{{ siteName }}<b>.</b></span></RouterLink
      ><span>为爱折腾的小动物而造。</span>
      <div>
        <RouterLink v-if="showModelPlaza" to="/model-plaza">模型广场</RouterLink
        ><a
          v-if="docUrl"
          :href="docUrl"
          target="_blank"
          rel="noopener noreferrer"
          >站点文档</a
        ><a href="#fur-start">接入指引</a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from "vue";
import Icon from "@/components/icons/Icon.vue";
import FurCodeConnection from "./FurCodeConnection.vue";

const props = withDefaults(
  defineProps<{
    isAuthenticated: boolean;
    isAdmin: boolean;
    isDark: boolean;
    siteName?: string;
    siteLogo?: string;
    docUrl?: string;
    showModelPlaza?: boolean;
  }>(),
  { siteName: "Fur Code", siteLogo: "", docUrl: "", showModelPlaza: false },
);
defineEmits<{ toggleTheme: [] }>();

const destination = computed(() =>
  props.isAuthenticated
    ? props.isAdmin
      ? "/admin/dashboard"
      : "/dashboard"
    : "/login",
);
// Public landing-page hints contain no keys. Account-specific setup stays in the existing console.
const tools = [
  {
    id: "codex",
    name: "Codex",
    symbol: ">_",
    subtitle: "让想法落成代码",
    caption: "从终端里的一个小任务开始。",
    guide:
      "在用户级 config.toml 中配置自定义 model_provider，填写 base_url，并用 env_key 指向保存 Fur Code 密钥的环境变量；选择分组支持的模型。",
    addressLabel: "Responses 协议 · Base URL",
    baseUrl: "https://www.fur-code.com/v1",
    docs: "https://developers.openai.com/codex/config-advanced/",
  },
  {
    id: "claude",
    name: "Claude Code",
    symbol: "*",
    subtitle: "和代码好好聊聊",
    caption: "梳理项目，也照顾每个细节。",
    guide:
      "通过 ANTHROPIC_BASE_URL 指向 Fur Code，并按网关认证方式设置 Fur Code API Key。使用支持 Anthropic Messages 协议的分组；这里只配置地址是不够的。",
    addressLabel: "Anthropic 协议 · Base URL",
    baseUrl: "https://www.fur-code.com",
    docs: "https://code.claude.com/docs/en/llm-gateway",
  },
  {
    id: "opencode",
    name: "OpenCode",
    symbol: "</>",
    subtitle: "给终端多一点可能",
    caption: "选好模型，继续你的工作流。",
    guide:
      "通过 /connect 添加自定义提供商凭证，再在配置文件中填写 Base URL 与模型列表。Chat Completions 和 Responses 使用的提供商适配器不同，请与所选分组匹配。",
    addressLabel: "OpenAI 兼容协议 · Base URL",
    baseUrl: "https://www.fur-code.com/v1",
    docs: "https://opencode.ai/docs/providers/#custom-provider",
  },
  {
    id: "pi",
    name: "Pi",
    symbol: "π",
    subtitle: "轻巧一点，自由一点",
    caption: "把喜欢的工具拼成自己的样子。",
    guide:
      "在 ~/.pi/agent/models.json 添加自定义提供商，配置 baseUrl、凭证和模型列表；api 字段需与分组支持的协议一致。用 /model 选择模型。",
    addressLabel: "OpenAI 兼容协议示例 · Base URL",
    baseUrl: "https://www.fur-code.com/v1",
    docs: "https://github.com/earendil-works/pi/blob/main/packages/coding-agent/docs/models.md",
  },
] as const;
type ToolId = (typeof tools)[number]["id"];
const selectedTool = ref<ToolId>("codex");
const activeTool = computed(
  () => tools.find((tool) => tool.id === selectedTool.value) ?? tools[0],
);
const copyState = ref<"idle" | "copied" | "failed">("idle");
let copyTimer: ReturnType<typeof setTimeout> | undefined;
let copyAttempt = 0;
function selectTool(id: ToolId) {
  selectedTool.value = id;
  copyAttempt++;
  copyState.value = "idle";
  clearTimeout(copyTimer);
}
function selectAddress(event: FocusEvent) {
  (event.target as HTMLInputElement).select();
}
async function copyAddress() {
  const attempt = ++copyAttempt;
  clearTimeout(copyTimer);
  try {
    await navigator.clipboard.writeText(activeTool.value.baseUrl);
    if (attempt !== copyAttempt) return;
    copyState.value = "copied";
  } catch {
    if (attempt !== copyAttempt) return;
    copyState.value = "failed";
  }
  copyTimer = setTimeout(() => {
    copyState.value = "idle";
  }, 3500);
}
onBeforeUnmount(() => {
  copyAttempt++;
  clearTimeout(copyTimer);
});
</script>

<style scoped>
.fur-home {
  --fur-bg: #fafbfefc;
  --fur-ink: #24304b;
  --fur-muted: #647089;
  --fur-line: #e1e6f0;
  --fur-card: #fff;
  --fur-soft: #eef1fb;
  --fur-accent: #5563bd;
  --fur-accent-hover: #4251ac;
  min-height: 100vh;
  background: var(--fur-bg);
  color: var(--fur-ink);
  font-family: Inter, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  font-size: 16px;
  line-height: 1.65;
  isolation: isolate;
  overflow-wrap: break-word;
}
.fur-home * {
  box-sizing: border-box;
}
.fur-home h1,
.fur-home h2,
.fur-home h3,
.fur-home p {
  margin: 0;
}
.fur-home a {
  text-decoration: none;
}
.fur-home button,
.fur-home input {
  font: inherit;
}
.fur-home button {
  cursor: pointer;
}
.fur-home :is(a, button, input, summary):focus-visible {
  outline: 3px solid #8994ed;
  outline-offset: 5px;
}
.fur-home svg {
  flex-shrink: 0;
}
.fur-home [id] {
  scroll-margin-top: 25px;
}
.fur-wrap {
  width: min(1160px, calc(100% - 64px));
  margin-inline: auto;
}
.fur-skip {
  position: absolute;
  top: 10px;
  left: 20px;
  transform: translateY(-160%);
  padding: 12px 18px;
  background: var(--fur-card);
  color: var(--fur-ink);
  z-index: 10;
  border-radius: 9px;
}
.fur-skip:focus {
  transform: translateY(0);
}
.fur-header {
  border-bottom: 1px solid var(--fur-line);
}
.fur-nav {
  min-height: 90px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 25px;
}
.fur-brand {
  display: inline-flex;
  align-items: center;
  gap: 11px;
  min-height: 44px;
  font-size: 24px;
  font-weight: 750;
  letter-spacing: -0.9px;
  color: var(--fur-ink);
}
.fur-brand b {
  color: var(--fur-accent);
  margin-left: 2px;
}
.fur-logo,
.fur-brand > img {
  width: 38px;
  height: 38px;
  border-radius: 12px;
  object-fit: contain;
}
.fur-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--fur-accent);
  color: #fff;
}
.fur-logo {
  font:
    700 18px Consolas,
    monospace;
  letter-spacing: -2px;
  padding-right: 2px;
}
.fur-navlinks,
.fur-nav-actions {
  display: flex;
  align-items: center;
  gap: 29px;
}
.fur-navlinks a {
  display: inline-flex;
  align-items: center;
  min-height: 44px;
  font-size: 13px;
  color: var(--fur-muted);
  transition: color 0.2s;
}
.fur-navlinks a:hover {
  color: var(--fur-accent);
}
.fur-nav-actions {
  gap: 12px;
}
.fur-theme {
  width: 42px;
  min-height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  color: var(--fur-muted);
  background: none;
  border: 0;
}
.fur-theme:hover {
  background: var(--fur-soft);
}
.fur-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  min-height: 50px;
  padding: 12px 22px;
  background: var(--fur-accent);
  color: #fff;
  border: 1px solid transparent;
  border-radius: 11px;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  transition:
    background 0.2s,
    box-shadow 0.2s;
}
.fur-button > span {
  font-size: 20px;
  line-height: 1;
  transition: transform 0.2s;
}
.fur-button:hover {
  background: var(--fur-accent-hover);
  box-shadow: 0 5px 16px #5563bd20;
}
.fur-button:hover > span {
  transform: translateX(3px);
}
.fur-small {
  min-height: 44px;
  padding: 10px 17px;
  font-size: 13px;
  gap: 15px;
}
.fur-secondary {
  background: var(--fur-card);
  border-color: var(--fur-line);
  color: var(--fur-ink);
  box-shadow: none;
}
.fur-secondary:hover {
  background: var(--fur-soft);
  box-shadow: none;
}
.fur-hero {
  display: grid;
  grid-template-columns: 1.14fr 1fr;
  align-items: center;
  gap: 40px;
  padding-block: 88px 81px;
  position: relative;
}
.fur-hero::before {
  content: "";
  position: absolute;
  inset: 0 -2% 0 48%;
  background: radial-gradient(ellipse at 50% 50%, #dce3fa66, transparent 67%);
  z-index: -1;
  pointer-events: none;
}
.fur-eyebrow {
  display: flex;
  align-items: center;
  gap: 9px;
  font-family: Consolas, "SFMono-Regular", monospace;
  font-size: 11px;
  letter-spacing: 1.8px;
  font-weight: 600;
  color: var(--fur-accent);
  line-height: 1.8;
}
.fur-hero h1 {
  margin-top: 25px;
  font-size: clamp(36px, 4vw, 53px);
  line-height: 1.33;
  letter-spacing: -1.8px;
  font-weight: 750;
}
.fur-hero h1 > span {
  color: var(--fur-accent);
}
.fur-lead {
  font-size: 15px;
  line-height: 2.05;
  color: var(--fur-muted);
  max-width: 493px;
  margin-top: 23px !important;
}
.fur-actions {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-top: 31px;
}
.fur-hero-note {
  display: flex;
  align-items: center;
  gap: 9px;
  color: var(--fur-muted);
  font-size: 12px;
  margin-top: 24px !important;
}
.fur-note-line {
  width: 18px;
  height: 1px;
  background: #aab3c7;
}
.fur-tools {
  padding-block: 43px 70px;
  border-top: 1px solid var(--fur-line);
}
.fur-section-heading {
  display: flex;
  justify-content: space-between;
  align-items: flex-end;
  gap: 25px;
  margin-bottom: 29px;
}
.fur-section-heading > p {
  color: var(--fur-muted);
  font-size: 13px;
  line-height: 1.9;
}
.fur-home h2 {
  font-size: 31px;
  line-height: 1.45;
  letter-spacing: -0.8px;
  font-weight: 700;
}
.fur-section-heading h2 {
  margin-top: 14px;
}
.fur-tool-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 15px;
}
.fur-tool-card {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  padding: 23px;
  background: var(--fur-card);
  color: var(--fur-ink);
  border: 1px solid var(--fur-line);
  border-radius: 14px;
  text-align: left;
  transition:
    border-color 0.2s,
    box-shadow 0.2s,
    transform 0.2s;
}
.fur-tool-card:hover {
  transform: translateY(-4px);
  border-color: #aab3db;
  box-shadow: 0 10px 24px #3345780c;
}
.fur-tool-card.is-selected {
  border-color: var(--fur-accent);
  box-shadow: 0 0 0 1px var(--fur-accent);
  background: var(--fur-card);
}
.fur-tool-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}
.fur-tool-symbol {
  width: 42px;
  height: 42px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: Consolas, monospace;
  font-size: 20px;
  font-weight: 650;
  background: var(--fur-soft);
  color: #4b5db1;
}
.symbol-claude {
  background: #fbede4;
  color: #996341;
  font-size: 35px;
  padding-top: 9px;
}
.symbol-opencode {
  background: #ecf0f5;
  color: #53617d;
  font-size: 15px;
}
.symbol-pi {
  background: #f0eafa;
  color: #776096;
  font-size: 26px;
}
.fur-tool-arrow {
  font-size: 19px;
  color: #939db2;
}
.fur-tool-card strong {
  font-size: 18px;
  font-weight: 700;
  letter-spacing: -0.4px;
}
.fur-tool-sub {
  font-size: 13px;
  color: var(--fur-muted);
  margin-top: 3px;
}
.fur-tool-caption {
  font-size: 12px;
  color: var(--fur-muted);
  border-top: 1px solid var(--fur-line);
  padding-top: 15px;
  margin-top: 20px;
  line-height: 1.7;
}
.fur-tool-guide {
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 44px;
  background: var(--fur-soft);
  border: 1px solid var(--fur-line);
  border-radius: 13px;
  padding: 27px 29px;
  margin-top: 21px;
  align-items: center;
  min-height: 204px;
}
.fur-guide-kicker {
  font-size: 12px;
  font-weight: 650;
  color: var(--fur-accent);
}
.fur-tool-guide p {
  font-size: 13px;
  color: var(--fur-muted);
  line-height: 1.9;
  margin-top: 7px;
}
.fur-text-link {
  display: inline-flex;
  align-items: center;
  gap: 14px;
  min-height: 44px;
  color: var(--fur-accent);
  font-size: 13px;
  font-weight: 600;
}
.fur-text-link:hover {
  text-decoration: underline;
  text-underline-offset: 5px;
}
.fur-address {
  min-width: 0;
}
.fur-address label {
  display: block;
  color: var(--fur-muted);
  font-size: 11px;
  margin-bottom: 8px;
}
.fur-address > div {
  display: flex;
  gap: 7px;
  padding: 5px 6px 5px 11px;
  border: 1px solid var(--fur-line);
  border-radius: 9px;
  background: var(--fur-card);
}
.fur-address input {
  min-width: 0;
  flex: 1;
  border: 0;
  background: none;
  color: var(--fur-ink);
  font-family: Consolas, monospace;
  font-size: 12px;
  padding: 6px 0;
  outline-offset: 2px !important;
}
.fur-address button {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 44px;
  min-width: 69px;
  padding: 8px;
  border: 0;
  border-radius: 6px;
  background: var(--fur-soft);
  font-size: 11px;
  color: var(--fur-accent);
}
.fur-address button:hover {
  background: #dde4f6;
}
.fur-address-note {
  display: block;
  min-height: 20px;
  font-size: 11px;
  margin-top: 9px;
  color: var(--fur-muted);
}
.fur-tool-footnote {
  font-size: 11px;
  color: var(--fur-muted);
  margin-top: 13px !important;
  text-align: center;
}
.fur-steps-section {
  background: var(--fur-soft);
  border-block: 1px solid var(--fur-line);
}
.fur-steps-layout {
  display: grid;
  grid-template-columns: 1fr 1.1fr;
  gap: 100px;
  padding-block: 69px;
  align-items: center;
}
.fur-steps-layout h2,
.fur-faq-section h2 {
  margin-top: 17px;
}
.fur-section-copy {
  color: var(--fur-muted);
  font-size: 14px;
  line-height: 1.9;
  margin-top: 18px !important;
}
.fur-steps-layout .fur-text-link {
  margin-top: 17px;
}
.fur-step-list {
  list-style: none;
  display: grid;
  gap: 25px;
  padding: 0;
  margin: 0;
}
.fur-step-list li {
  display: flex;
  gap: 20px;
  align-items: flex-start;
}
.fur-step-number {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 39px;
  height: 39px;
  border-radius: 12px;
  background: var(--fur-card);
  border: 1px solid var(--fur-line);
  font-family: Consolas, monospace;
  font-size: 12px;
  color: var(--fur-accent);
}
.fur-step-list h3 {
  font-size: 16px;
  font-weight: 650;
}
.fur-step-list p {
  font-size: 13px;
  line-height: 1.9;
  color: var(--fur-muted);
  margin-top: 6px;
}
.fur-faq-section {
  display: grid;
  grid-template-columns: 0.7fr 1.3fr;
  gap: 90px;
  padding-block: 75px;
}
.fur-faq {
  border-top: 1px solid var(--fur-line);
}
.fur-faq details {
  border-bottom: 1px solid var(--fur-line);
}
.fur-faq summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  min-height: 70px;
  padding: 19px 0;
  list-style: none;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
.fur-faq summary::-webkit-details-marker {
  display: none;
}
.fur-faq summary::after {
  content: "+";
  font-size: 22px;
  color: var(--fur-accent);
  font-weight: 400;
}
.fur-faq details[open] summary::after {
  content: "−";
}
.fur-faq details p {
  color: var(--fur-muted);
  font-size: 13px;
  line-height: 1.9;
  padding: 0 25px 21px 0;
}
.fur-invitation {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 35px;
  padding: 36px 39px;
  border: 1px solid #dedff0;
  border-radius: 17px;
  background: linear-gradient(115deg, var(--fur-soft), var(--fur-card));
  margin-bottom: 54px;
  overflow: hidden;
}
.fur-invitation > div,
.fur-invitation > .fur-button {
  position: relative;
  z-index: 1;
}
.fur-invitation h2 {
  font-size: 25px;
  margin-top: 9px;
}
.fur-invitation p:not(.fur-eyebrow) {
  font-size: 13px;
  color: var(--fur-muted);
  margin-top: 9px;
}
.fur-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
  padding-block: 25px;
  border-top: 1px solid var(--fur-line);
  font-size: 11px;
  color: var(--fur-muted);
}
.fur-footer .fur-brand {
  font-size: 19px;
}
.fur-footer .fur-logo {
  width: 29px;
  height: 29px;
  border-radius: 9px;
}
.fur-footer .fur-logo {
  font-size: 14px;
}
.fur-footer > div {
  display: flex;
  gap: 21px;
}
.fur-footer > div a {
  display: flex;
  align-items: center;
  min-height: 44px;
  color: var(--fur-muted);
}
.fur-footer > div a:hover {
  color: var(--fur-accent);
}
.dark .fur-home {
  --fur-bg: #141e32;
  --fur-ink: #e9edf8;
  --fur-muted: #acb8cf;
  --fur-line: #35415b;
  --fur-card: #202d46;
  --fur-soft: #1b2943;
  --fur-accent: #b0bcff;
  --fur-accent-hover: #bec7ff;
}
.dark .fur-button {
  color: #182443;
}
.fur-home .fur-button:not(.fur-secondary) {
  color: #fff;
  background: #5563bd;
}
.fur-home .fur-button:not(.fur-secondary):hover {
  background: #4251ac;
}
.dark .fur-secondary {
  color: var(--fur-ink);
}
.dark .fur-tool-symbol {
  background: #303e65;
  color: #c4ceff;
}
.dark .symbol-claude {
  background: #4a3c39;
  color: #f1c8ad;
}
.dark .symbol-pi {
  background: #3d355b;
  color: #ddc9ff;
}
.dark .fur-address button:hover {
  background: #314264;
}
.dark .fur-invitation {
  border-color: var(--fur-line);
}
@media (prefers-reduced-motion: no-preference) {
  .fur-hero-copy {
    animation: fur-hero-enter 0.8s both;
  }
}
@keyframes fur-hero-enter {
  from {
    opacity: 0;
    transform: translateY(13px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@media (max-width: 1023px) {
  .fur-wrap {
    width: calc(100% - 48px);
  }
  .fur-nav {
    gap: 15px;
  }
  .fur-navlinks {
    gap: 20px;
  }
  .fur-brand {
    font-size: 22px;
  }
  .fur-hero {
    gap: 25px;
    padding-block: 48px 45px;
    grid-template-columns: 1.05fr 1fr;
  }
  .fur-hero h1 {
    font-size: 39px;
    letter-spacing: -1.6px;
  }
  .fur-lead {
    font-size: 14px;
  }
  .fur-tool-card {
    padding: 19px 16px;
  }
  .fur-tool-guide {
    gap: 25px;
    padding: 25px;
  }
  .fur-steps-layout {
    gap: 45px;
  }
  .fur-faq-section {
    gap: 40px;
  }
  .fur-tool-caption {
    font-size: 11px;
  }
  .fur-invitation h2 {
    font-size: 23px;
  }
}
@media (max-width: 767px) {
  .fur-wrap {
    width: calc(100% - 40px);
  }
  .fur-nav {
    flex-wrap: wrap;
    padding-top: 13px;
    gap: 12px;
    min-height: 0;
  }
  .fur-brand {
    font-size: 21px;
  }
  .fur-navlinks {
    order: 3;
    justify-content: center;
    width: 100%;
    border-top: 1px solid var(--fur-line);
    gap: 29px;
    padding-block: 2px;
    font-size: 12px;
  }
  .fur-nav-actions {
    gap: 4px;
  }
  .fur-small {
    font-size: 12px;
    padding: 9px 13px;
    gap: 10px;
  }
  .fur-theme {
    width: 34px;
  }
  .fur-hero {
    grid-template-columns: 1fr;
    gap: 32px;
    padding-block: 39px 33px;
  }
  .fur-hero h1 {
    font-size: clamp(33px, 7vw, 46px);
    letter-spacing: -1.5px;
    margin-top: 20px;
  }
  .fur-lead {
    font-size: 14px;
    max-width: 550px;
    margin-top: 20px !important;
  }
  .fur-eyebrow {
    font-size: 10px;
    letter-spacing: 1.3px;
  }
  .fur-desktop-break {
    display: none;
  }
  .fur-actions {
    margin-top: 24px;
  }
  .fur-button {
    font-size: 13px;
    padding: 12px 17px;
    gap: 12px;
  }
  .fur-hero-note {
    font-size: 11px;
    margin-top: 20px !important;
  }
  .fur-tools {
    padding-block: 33px 43px;
  }
  .fur-section-heading {
    display: block;
    margin-bottom: 23px;
  }
  .fur-section-heading > p {
    margin-top: 13px;
  }
  .fur-section-heading > p br {
    display: none;
  }
  .fur-home h2 {
    font-size: 25px;
  }
  .fur-tool-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 13px;
  }
  .fur-tool-card {
    padding: 19px 17px;
  }
  .fur-tool-card strong {
    font-size: 17px;
  }
  .fur-tool-caption {
    font-size: 11px;
    padding-top: 12px;
    margin-top: 16px;
  }
  .fur-tool-top {
    margin-bottom: 15px;
  }
  .fur-tool-guide {
    grid-template-columns: 1fr;
    gap: 18px;
    padding: 21px;
    min-height: 0;
  }
  .fur-address input {
    font-size: 11px;
  }
  .fur-tool-footnote {
    font-size: 11px;
    text-align: left;
    line-height: 1.8;
  }
  .fur-steps-layout {
    grid-template-columns: 1fr;
    gap: 27px;
    padding-block: 45px;
  }
  .fur-step-list {
    gap: 23px;
  }
  .fur-faq-section {
    grid-template-columns: 1fr;
    gap: 25px;
    padding-block: 45px;
  }
  .fur-invitation {
    padding: 27px 24px;
    display: block;
    margin-bottom: 37px;
  }
  .fur-invitation h2 {
    font-size: 22px;
  }
  .fur-invitation .fur-button {
    margin-top: 23px;
  }
  .fur-invitation .fur-eyebrow {
    font-size: 9px;
  }
  .fur-footer {
    flex-wrap: wrap;
    gap: 7px;
  }
  .fur-footer > span {
    order: 3;
    width: 100%;
  }
  .fur-footer > div {
    gap: 17px;
  }
  .fur-tool-sub {
    font-size: 12px;
  }
}
@media (max-width: 374px) {
  .fur-wrap {
    width: calc(100% - 32px);
  }
  .fur-brand {
    font-size: 19px;
    gap: 8px;
  }
  .fur-nav-actions {
    gap: 0;
  }
  .fur-hero h1 {
    font-size: 31px;
  }
  .fur-tool-card {
    padding: 16px 13px;
  }
  .fur-actions .fur-button {
    padding: 11px 13px;
    font-size: 12px;
  }
  .fur-small {
    padding-inline: 11px;
  }
  .fur-logo {
    width: 32px;
    height: 32px;
  }
  .fur-tool-caption {
    min-height: 51px;
  }
  .fur-address > div {
    padding-left: 8px;
  }
  .fur-address button {
    min-width: 57px;
  }
  .fur-address button svg {
    display: none;
  }
}
@media (prefers-reduced-motion: reduce) {
  .fur-home * {
    animation: none !important;
    transition: none !important;
  }
  .fur-tool-card:hover,
  .fur-button:hover > span {
    transform: none;
  }
}
</style>
