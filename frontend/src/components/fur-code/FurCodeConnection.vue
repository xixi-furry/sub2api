<template>
  <div class="connection" aria-hidden="true">
    <div class="connection-bar">
      <span class="window-dots"><i></i><i></i><i></i></span>
      <span>your workspace</span><span class="connection-command">⌘</span>
    </div>
    <div class="connection-body">
      <p class="connection-label">YOUR TOOLS. ONE ENTRY.</p>
      <div class="connection-tools">
        <span>Codex</span><span>Claude Code</span><span>OpenCode</span
        ><span>Pi</span>
      </div>
      <div class="connection-lines"><i></i><i></i><i></i><i></i></div>
      <div class="connection-stem"><span></span></div>
      <div class="connection-hub">
        <span class="connection-mark">&lt;/&gt;</span>
        <div>
          <strong>Fur Code<span>.</span></strong
          ><small>顺手的 AI 接入入口</small>
        </div>
        <span class="connection-arrow">↗</span>
      </div>
      <div class="connection-protocols">
        <span>Responses</span><i></i><span>Messages</span>
      </div>
      <div class="connection-prompt">
        <span>&gt;</span> build something you love<span
          class="connection-cursor"
        ></span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.connection {
  width: 100%;
  max-width: 510px;
  margin-inline: auto;
  overflow: hidden;
  border-radius: 20px;
  border: 1px solid #344361;
  background: #182238;
  color: #edf0ff;
  box-shadow:
    0 28px 60px -34px #28376266,
    0 0 0 7px #cbd3eb29;
  font-family: Inter, "Segoe UI", "Microsoft YaHei", sans-serif;
}
.connection-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 22px;
  border-bottom: 1px solid #34405a;
  color: #a9b6ce;
  font:
    11px Consolas,
    monospace;
}
.window-dots {
  display: flex;
  gap: 5px;
}
.window-dots i {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #63708b;
}
.window-dots i:first-child {
  background: #9caadd;
}
.connection-command {
  width: 28px;
  text-align: right;
  font-size: 16px;
}
.connection-body {
  padding: 35px 26px 0;
}
.connection-label {
  margin: 0 0 22px;
  color: #aab6d4;
  font:
    10px Consolas,
    monospace;
  letter-spacing: 2px;
}
.connection-tools {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
}
.connection-tools > span {
  padding: 12px 3px;
  text-align: center;
  border: 1px solid #45516b;
  border-radius: 8px;
  background: #24314a;
  font-size: 11px;
  white-space: nowrap;
}
.connection-lines {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  margin-inline: 12.5%;
  height: 37px;
  position: relative;
}
.connection-lines::after {
  content: "";
  position: absolute;
  inset: auto 0 0;
  border-bottom: 1px solid #6a77ae;
}
.connection-lines i {
  border-left: 1px solid #6a77ae;
}
.connection-lines i:nth-child(2) {
  transform: translateX(33.333%);
}
.connection-lines i:nth-child(3) {
  transform: translateX(66.666%);
}
.connection-lines i:last-child {
  border-left: 0;
  border-right: 1px solid #6a77ae;
}
.connection-stem {
  height: 30px;
  position: relative;
}
.connection-stem::before {
  content: "";
  position: absolute;
  left: 50%;
  inset-block: 0;
  border-left: 1px solid #6a77ae;
}
.connection-stem span {
  position: absolute;
  left: calc(50% - 2px);
  top: 12px;
  width: 5px;
  height: 5px;
  border-radius: 50%;
  background: #c1caff;
  box-shadow: 0 0 10px #a6b6ff;
}
.connection-hub {
  display: flex;
  align-items: center;
  gap: 15px;
  padding: 22px;
  border: 1px solid #7c88c2;
  border-radius: 12px;
  background: linear-gradient(115deg, #34416a, #263452);
  box-shadow: 0 12px 36px #0b152630;
}
.connection-mark {
  display: grid;
  place-items: center;
  width: 43px;
  height: 43px;
  flex-shrink: 0;
  border: 1px solid #7684b9;
  border-radius: 11px;
  background: #5463a3;
  font:
    700 18px Consolas,
    monospace;
  letter-spacing: -2px;
  padding-right: 2px;
}
.connection-hub strong {
  display: block;
  font-size: 24px;
  font-weight: 650;
  line-height: 1.2;
  letter-spacing: -0.8px;
}
.connection-hub strong span {
  color: #b5c1ff;
}
.connection-hub small {
  display: block;
  margin-top: 7px;
  color: #c1cbe2;
  font-size: 11px;
}
.connection-arrow {
  margin-left: auto;
  color: #b5c1ff;
  font-size: 24px;
}
.connection-protocols {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 13px;
  margin: 24px 0 28px;
  color: #a9b6ce;
  font:
    11px Consolas,
    monospace;
}
.connection-protocols i {
  width: 3px;
  height: 3px;
  border-radius: 50%;
  background: #7180a2;
}
.connection-prompt {
  display: flex;
  align-items: center;
  gap: 9px;
  border-top: 1px solid #34405a;
  padding: 20px 0 23px;
  color: #c4cee4;
  font:
    12px Consolas,
    monospace;
  white-space: nowrap;
}
.connection-prompt > span:first-child {
  color: #9faeef;
}
.connection-cursor {
  width: 5px;
  height: 13px;
  background: #aebcf2;
}
@media (prefers-reduced-motion: no-preference) {
  .connection {
    animation: connection-enter 0.7s ease-out both;
  }
  .connection-stem span {
    animation: connection-travel 1.4s ease-in-out 3;
  }
  .connection-cursor {
    animation: connection-blink 1s step-end 4;
  }
}
@keyframes connection-enter {
  from {
    opacity: 0;
    transform: translateY(12px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}
@keyframes connection-travel {
  0% {
    opacity: 0;
    transform: translateY(-12px);
  }
  35% {
    opacity: 1;
  }
  100% {
    opacity: 0;
    transform: translateY(13px);
  }
}
@keyframes connection-blink {
  50% {
    opacity: 0;
  }
}
@media (max-width: 1023px) {
  .connection-body {
    padding: 26px 18px 0;
  }
  .connection-tools {
    gap: 5px;
  }
  .connection-tools > span {
    font-size: 9px;
  }
  .connection-hub {
    padding: 18px 15px;
    gap: 11px;
  }
  .connection-hub strong {
    font-size: 22px;
  }
  .connection-prompt {
    font-size: 10px;
    gap: 7px;
  }
}
@media (max-width: 767px) {
  .connection {
    max-width: 450px;
    border-radius: 16px;
  }
  .connection-tools > span {
    font-size: 10px;
  }
  .connection-body {
    padding-top: 24px;
  }
}
@media (max-width: 374px) {
  .connection-body {
    padding-inline: 14px;
  }
  .connection-tools > span {
    font-size: 9px;
  }
  .connection-label {
    font-size: 9px;
    letter-spacing: 1.4px;
  }
}
</style>
