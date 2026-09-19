import { flushPromises, mount, RouterLinkStub } from "@vue/test-utils";
import { afterEach, describe, expect, it, vi } from "vitest";
import FurCodeHome from "../FurCodeHome.vue";

function mountHome(props = {}) {
  return mount(FurCodeHome, {
    props: { isAuthenticated: false, isAdmin: false, isDark: false, ...props },
    global: { stubs: { RouterLink: RouterLinkStub, FurCodeConnection: true } },
  });
}

afterEach(() => {
  vi.unstubAllGlobals();
  vi.useRealTimers();
});

describe("Fur Code account entry and setup helpers", () => {
  it("reacts to login, administrator status, and logout without keeping a stale destination", async () => {
    const wrapper = mountHome();
    const entry = () => wrapper.getComponent('[data-testid="fur-auth-entry"]');
    expect(entry().props("to")).toBe("/login");
    await wrapper.setProps({ isAuthenticated: true });
    expect(entry().props("to")).toBe("/dashboard");
    expect(entry().text()).toContain("进入控制台");
    await wrapper.setProps({ isAdmin: true });
    expect(entry().props("to")).toBe("/admin/dashboard");
    await wrapper.setProps({ isAuthenticated: false, isAdmin: false });
    expect(entry().props("to")).toBe("/login");
    wrapper.unmount();
  });

  it("preserves the keys destination through login", () => {
    const wrapper = mountHome();
    const keys = wrapper
      .findAllComponents(RouterLinkStub)
      .find((link) => link.text().includes("登录后创建密钥"));
    expect(keys?.props("to")).toEqual({
      path: "/login",
      query: { redirect: "/keys" },
    });
    expect(
      wrapper
        .findAllComponents(RouterLinkStub)
        .some((link) => link.props("to") === "/model-plaza"),
    ).toBe(false);
    wrapper.unmount();
  });

  it("copies the selected protocol address rather than always appending v1", async () => {
    vi.useFakeTimers();
    const writeText = vi.fn().mockResolvedValue(undefined);
    vi.stubGlobal("navigator", { clipboard: { writeText } });
    const wrapper = mountHome();
    await wrapper
      .get('[aria-label="查看 Claude Code 接入指引"]')
      .trigger("click");
    expect(
      (wrapper.get("#fur-base-url").element as HTMLInputElement).value,
    ).toBe("https://www.fur-code.com");
    await wrapper
      .get('[aria-label="复制 Claude Code API 地址"]')
      .trigger("click");
    await flushPromises();
    expect(writeText).toHaveBeenCalledWith("https://www.fur-code.com");
    expect(wrapper.text()).toContain("已复制");
    await wrapper.get('[aria-label="查看 Pi 接入指引"]').trigger("click");
    expect(
      (wrapper.get("#fur-base-url").element as HTMLInputElement).value,
    ).toBe("https://www.fur-code.com/v1");
    expect(wrapper.text()).not.toContain("已复制");
    wrapper.unmount();
  });

  it("offers manual copying when clipboard access is unavailable", async () => {
    vi.useFakeTimers();
    vi.stubGlobal("navigator", {
      clipboard: { writeText: vi.fn().mockRejectedValue(new Error("denied")) },
    });
    const wrapper = mountHome();
    await wrapper.get('[aria-label="复制 Codex API 地址"]').trigger("click");
    await flushPromises();
    expect(wrapper.text()).toContain("可选中上方地址手动复制");
    expect(wrapper.get("#fur-base-url").attributes("readonly")).toBeDefined();
    wrapper.unmount();
  });

  it("does not announce a stale copy success after switching tools", async () => {
    let finish!: () => void;
    vi.stubGlobal("navigator", {
      clipboard: {
        writeText: () =>
          new Promise<void>((resolve) => {
            finish = resolve;
          }),
      },
    });
    const wrapper = mountHome();
    await wrapper.get('[aria-label="复制 Codex API 地址"]').trigger("click");
    await wrapper
      .get('[aria-label="查看 Claude Code 接入指引"]')
      .trigger("click");
    finish();
    await flushPromises();
    expect(wrapper.text()).not.toContain("已复制");
    wrapper.unmount();
  });
});
