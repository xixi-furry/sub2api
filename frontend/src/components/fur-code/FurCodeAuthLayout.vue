<template>
  <div class="fur-login" data-testid="fur-login-layout">
    <header class="login-nav">
      <RouterLink to="/home" class="login-brand"
        ><span class="login-logo" aria-hidden="true">&lt;/&gt;</span
        ><span>{{ siteName }}<b>.</b></span></RouterLink
      ><RouterLink to="/home" class="login-back"
        ><span aria-hidden="true">←</span> 返回首页</RouterLink
      >
    </header>
    <main class="login-main">
      <section class="login-story" aria-labelledby="fur-login-title">
        <p class="login-eyebrow">LESS SETUP. MORE CREATING.</p>
        <h1 id="fur-login-title">顺手的<br /><span>AI 接入入口。</span></h1>
        <p class="login-intro">
          为爱折腾的小动物而造。<br />接上熟悉的工具，继续你的下一个好想法。
        </p>
        <FurCodeConnection class="login-connection" />
        <div class="login-tools" aria-label="常用接入工具">
          <span>Codex</span><i></i><span>Claude Code</span><i></i
          ><span>OpenCode</span><i></i><span>Pi</span>
        </div>
      </section>
      <section class="login-form-section" aria-label="账号登录">
        <div class="login-card">
          <div class="login-welcome-mark" aria-hidden="true">&lt;/&gt;</div>
          <slot />
        </div>
        <div class="login-footer"><slot name="footer" /></div>
        <p class="login-note">从这里，继续你的创造。</p>
      </section>
    </main>
    <footer class="login-copyright">
      © {{ currentYear }} {{ siteName }} · 顺手的 AI 接入入口。
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useAppStore } from "@/stores";
import FurCodeConnection from "./FurCodeConnection.vue";
const appStore = useAppStore();
const siteName = computed(() =>
  appStore.siteName && appStore.siteName !== "Sub2API"
    ? appStore.siteName
    : "Fur Code",
);
const currentYear = new Date().getFullYear();
onMounted(() => {
  if (!appStore.publicSettingsLoaded) appStore.fetchPublicSettings();
});
</script>

<style scoped>
.fur-login {
  --login-bg: #fafbfe;
  --login-ink: #24304b;
  --login-muted: #647089;
  --login-card: #fff;
  --login-line: #e1e6f0;
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  color: var(--login-ink);
  background: var(--login-bg);
  font-family: Inter, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
  line-height: 1.65;
}
.login-nav {
  width: min(1160px, calc(100% - 64px));
  margin-inline: auto;
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 90px;
  gap: 20px;
  border-bottom: 1px solid var(--login-line);
}
.login-brand {
  display: inline-flex;
  align-items: center;
  gap: 11px;
  font-size: 24px;
  font-weight: 750;
  letter-spacing: -0.8px;
  min-height: 44px;
  color: var(--login-ink);
}
.login-brand b {
  color: #6674c6;
  margin-left: 2px;
}
.login-logo {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: 12px;
  background: #5563bd;
  color: #fff;
}
.login-logo {
  font:
    700 18px Consolas,
    monospace;
  letter-spacing: -2px;
  padding-right: 2px;
}
.login-back {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  font-size: 13px;
  color: var(--login-muted);
}
.login-back:hover {
  color: #5563bd;
}
.fur-login a:focus-visible {
  outline: 3px solid #8994ed;
  outline-offset: 5px;
  border-radius: 5px;
}
.login-main {
  width: min(1080px, calc(100% - 80px));
  margin: auto;
  display: grid;
  grid-template-columns: 1.1fr 1fr;
  gap: 93px;
  align-items: center;
  padding-block: 50px;
  flex: 1;
}
.login-story {
  position: relative;
  min-width: 0;
}
.login-eyebrow {
  font-family: Consolas, monospace;
  font-size: 10px;
  letter-spacing: 2px;
  color: #6674b5;
  font-weight: 600;
  margin: 0 0 18px;
}
.login-story h1 {
  font-size: 42px;
  line-height: 1.35;
  letter-spacing: -1.5px;
  font-weight: 750;
  margin: 0;
}
.login-story h1 span {
  color: #6573b9;
}
.login-intro {
  font-size: 14px;
  line-height: 1.95;
  color: var(--login-muted);
  margin: 20px 0 0;
}
.login-tools {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 15px;
  flex-wrap: wrap;
  color: var(--login-muted);
  font-size: 11px;
  margin-top: 15px;
}
.login-tools i {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: #b6bed3;
}
.login-form-section {
  width: 100%;
  min-width: 0;
}
.login-card {
  padding: 33px 36px 35px;
  background: var(--login-card);
  border: 1px solid var(--login-line);
  border-radius: 21px;
  box-shadow: 0 20px 70px -38px #3b518a42;
  position: relative;
}
.login-welcome-mark {
  width: 46px;
  height: 46px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 21px;
  border: 1px solid #e1e4f4;
  border-radius: 15px;
  background: #f0f2fc;
  color: #6370bc;
}
.login-welcome-mark {
  font:
    700 22px Consolas,
    monospace;
  letter-spacing: -2px;
  padding-right: 2px;
}
.login-connection {
  margin-top: 30px;
}
.login-footer {
  margin-top: 22px;
  text-align: center;
  font-size: 13px;
}
.login-note {
  text-align: center;
  font-size: 11px;
  color: var(--login-muted);
  margin-top: 20px;
}
.login-copyright {
  text-align: center;
  font-size: 11px;
  color: var(--login-muted);
  padding: 20px 24px 25px;
}
.login-card :deep(.input) {
  min-height: 49px;
  border-radius: 10px;
  font-size: 16px;
  background: var(--login-card);
  color: var(--login-ink);
  border-color: var(--login-line);
}
.login-card :deep(.input:focus) {
  border-color: #7886d4;
  --tw-ring-color: #7886d433;
}
.login-card :deep(.input-error) {
  border-color: #dc6262;
}
.login-card :deep(.input:disabled) {
  opacity: 0.65;
  cursor: not-allowed;
}
.login-card :deep(.input-label) {
  color: var(--login-ink);
  font-size: 13px;
}
.login-card :deep(.btn) {
  min-height: 47px;
  border-radius: 10px;
  font-size: 14px;
}
.login-card :deep(.btn-primary) {
  background: #5563bd;
  color: #fff;
  border-color: #5563bd;
}
.login-card :deep(.btn-primary:not(:disabled):hover) {
  background: #4251ac;
}
.login-card :deep(.btn:focus-visible) {
  outline: 3px solid #8994ed;
  outline-offset: 3px;
}
.login-card :deep(.password-toggle) {
  min-width: 44px;
  justify-content: center;
  padding-right: 0;
}
.login-card :deep(.fur-login-heading) {
  font-size: 26px;
  letter-spacing: -0.7px;
  color: var(--login-ink);
}
.login-card :deep(.fur-login-subheading) {
  color: var(--login-muted);
  font-size: 13px;
  line-height: 1.8;
}
.login-card :deep(.fur-login-error) {
  border: 1px solid #efb6b6;
  background: #fff3f3;
  color: #aa3434;
  padding: 11px 13px;
  font-size: 13px;
  border-radius: 9px;
}
.login-footer :deep(p) {
  color: var(--login-muted);
}
.dark .fur-login {
  --login-bg: #141e32;
  --login-ink: #e9edf8;
  --login-muted: #acb8cf;
  --login-card: #202d46;
  --login-line: #3a4662;
}
.dark .login-story h1 span,
.dark .login-eyebrow {
  color: #b0bcff;
}
.dark .login-welcome-mark {
  background: #303e65;
  border-color: #485579;
  color: #c4ceff;
}
.dark .login-card :deep(.fur-login-error) {
  background: #452b37;
  border-color: #78434e;
  color: #ffd0d0;
}
@media (max-width: 1023px) {
  .login-main {
    gap: 45px;
    width: calc(100% - 64px);
  }
  .login-story h1 {
    font-size: 35px;
  }
  .login-card {
    padding: 28px 25px;
  }
  .login-tools {
    gap: 9px;
    font-size: 10px;
  }
  .login-card :deep(.fur-login-heading) {
    font-size: 23px;
  }
}
@media (max-width: 767px) {
  .login-nav {
    width: calc(100% - 40px);
    min-height: 77px;
  }
  .login-brand {
    font-size: 21px;
  }
  .login-main {
    width: min(440px, calc(100% - 40px));
    grid-template-columns: 1fr;
    gap: 0;
    padding-block: 32px;
  }
  .login-story {
    display: none;
  }
  .login-card {
    padding: 29px 25px 30px;
    border-radius: 18px;
  }
  .login-form-section {
    align-self: start;
  }
  .login-copyright {
    font-size: 10px;
    padding-bottom: 22px;
  }
  .login-welcome-mark {
    margin-bottom: 19px;
  }
  .login-card :deep(.fur-login-heading) {
    font-size: 25px;
  }
}
@media (max-width: 374px) {
  .login-main,
  .login-nav {
    width: calc(100% - 32px);
  }
  .login-card {
    padding-inline: 21px;
  }
  .login-brand {
    font-size: 20px;
  }
  .login-back {
    font-size: 12px;
  }
  .login-card :deep(.fur-login-heading) {
    font-size: 23px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .fur-login * {
    animation: none !important;
    transition: none !important;
  }
}
</style>
