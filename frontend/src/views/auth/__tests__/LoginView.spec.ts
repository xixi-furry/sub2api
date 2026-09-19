import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import LoginView from '@/views/auth/LoginView.vue'

const { getPublicSettingsMock, pushMock, loginMock, isTotpMock } = vi.hoisted(() => ({
  getPublicSettingsMock: vi.fn(),
  pushMock: vi.fn(),
  loginMock: vi.fn(),
  isTotpMock: vi.fn()
}))

const publicSettings = {
  registration_enabled: true,
  turnstile_enabled: false,
  turnstile_site_key: '',
  tencent_captcha_enabled: false,
  tencent_captcha_app_id: '',
  aliyun_captcha_enabled: false,
  aliyun_captcha_scene_id: '',
  aliyun_captcha_prefix: '',
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  backend_mode_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
  password_reset_enabled: false,
  passkey_enabled: false,
  login_agreement_enabled: false,
  login_agreement_documents: []
}

vi.mock('vue-router', () => ({
  useRouter: () => ({
    push: pushMock,
    currentRoute: { value: { query: {} } }
  })
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      t: (key: string) => key
    }
  }),
  useI18n: () => ({
    t: (key: string) => key
  })
}))

vi.mock('@/stores', () => ({
  useAuthStore: () => ({
    login: loginMock,
    loginWithPasskey: vi.fn(),
    login2FA: vi.fn()
  }),
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showWarning: vi.fn()
  })
}))

vi.mock('@/api/auth', () => ({
  buildOAuthLoginStartURL: vi.fn(),
  getPublicSettings: (...args: unknown[]) => getPublicSettingsMock(...args),
  isTotp2FARequired: isTotpMock,
  isWeChatWebOAuthEnabled: vi.fn(() => false),
  startOAuthLogin: vi.fn()
}))

function mountLogin() {
  return mount(LoginView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /><slot name="footer" /></div>' },
        DingTalkOAuthSection: true,
        EmailOAuthButtons: true,
        Icon: true,
        LinuxDoOAuthSection: true,
        LoginAgreementPrompt: true,
        OidcOAuthSection: true,
        RouterLink: { template: '<a><slot /></a>' },
        TotpLoginModal: true,
        TurnstileWidget: true,
        WechatOAuthSection: true,
        transition: false
      }
    }
  })
}

describe('LoginView registration entry', () => {
  beforeEach(() => {
    getPublicSettingsMock.mockReset()
    pushMock.mockReset()
    loginMock.mockReset()
    isTotpMock.mockReset().mockReturnValue(false)
    getPublicSettingsMock.mockResolvedValue(publicSettings)
  })

  it('shows the registration entry when registration is enabled', async () => {
    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).toContain('auth.signUp')
  })

  it('hides the registration entry when registration is disabled', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({
      ...publicSettings,
      registration_enabled: false
    })

    const wrapper = mountLogin()
    await flushPromises()

    expect(wrapper.text()).not.toContain('auth.signUp')
  })

  it('keeps password visibility accessible and still submits the original credentials', async () => {
    loginMock.mockResolvedValueOnce({ access_token: 'test-session' })
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('builder@example.com')
    await wrapper.get('#password').setValue('test-password')
    await wrapper.get('[aria-label="显示密码"]').trigger('click')
    expect(wrapper.get('#password').attributes('type')).toBe('text')
    expect(wrapper.get('[aria-label="隐藏密码"]').attributes('aria-pressed')).toBe('true')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(loginMock).toHaveBeenCalledWith(expect.objectContaining({ email: 'builder@example.com', password: 'test-password' }))
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
  })

  it('shows a login failure inline without navigating to a protected page', async () => {
    loginMock.mockRejectedValueOnce(new Error('Login failed'))
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('builder@example.com')
    await wrapper.get('#password').setValue('test-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).not.toBe('')
    expect(pushMock).not.toHaveBeenCalled()
  })

  it('keeps two-factor verification ahead of dashboard navigation', async () => {
    isTotpMock.mockReturnValue(true)
    loginMock.mockResolvedValueOnce({ temp_token: 'test-pending', user_email_masked: 'b***@example.com' })
    const wrapper = mountLogin()
    await flushPromises()
    await wrapper.get('#email').setValue('builder@example.com')
    await wrapper.get('#password').setValue('test-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()
    expect(wrapper.findComponent({ name: 'TotpLoginModal' }).exists()).toBe(true)
    expect(pushMock).not.toHaveBeenCalled()
  })

  it('still blocks submission until the configured Turnstile challenge is verified', async () => {
    getPublicSettingsMock.mockResolvedValueOnce({ ...publicSettings, turnstile_enabled: true, turnstile_site_key: 'test-site-key' })
    const wrapper = mountLogin()
    await flushPromises()
    expect(wrapper.get('button[type="submit"]').attributes('disabled')).toBeDefined()
    expect(loginMock).not.toHaveBeenCalled()
  })
})
