<script setup lang="ts">
import { NButton, NCard, NForm, NFormItem, NGradientText, NInput, NSelect, useMessage } from 'naive-ui'
import { ref } from 'vue'
import { login } from '@/api'
import { useAppStore, useAuthStore } from '@/store'
import { SvgIcon } from '@/components/common'
import { router } from '@/router'
import { t } from '@/locales'
import { languageOptions } from '@/utils/defaultData'
import type { Language } from '@/store/modules/app/helper'

// const userStore = useUserStore()
const authStore = useAuthStore()
const appStore = useAppStore()
const ms = useMessage()
const loading = ref(false)
const languageValue = ref<Language>(appStore.language)

const usernameFocus = ref(false)
const passwordFocus = ref(false)

// const isShowCaptcha = ref<boolean>(false)
// const isShowRegister = ref<boolean>(false)

const form = ref<Login.LoginReqest>({
  username: '',
  password: '',
})

const loginPost = async () => {
  loading.value = true
  try {
    const res = await login<Login.LoginResponse>(form.value)
    if (res.code === 0) {
      authStore.setToken(res.data.token)
      authStore.setUserInfo(res.data)

      setTimeout(() => {
        ms.success(`Hi ${res.data.name},${t('login.welcomeMessage')}`)
        loading.value = false
        router.push({ path: '/' })
      }, 500)
    }
    else {
      loading.value = false
      // captchaRef.value.refresh()
    }
  }
  catch (error) {
    loading.value = false
    // 请检查网络或者服务器错误
    console.log(error)
  }
}

function handleSubmit() {
  // 点击登录按钮触发
  loginPost()
}

function handleChangeLanuage(value: Language) {
  languageValue.value = value
  appStore.setLanguage(value)
}
</script>

<template>
  <div class="login-container">
    <NCard class="login-card" style="border-radius: 20px;">
      <div class="mb-5 flex items-center justify-end">
        <div class="mr-2">
          <SvgIcon icon="ion-language" style="width: 20;height: 20;" />
        </div>
        <div class="min-w-[100px]">
          <NSelect v-model:value="languageValue" size="small" :options="languageOptions" @update-value="handleChangeLanuage" />
        </div>
      </div>

      <div class="login-title">
        <NGradientText :size="32" type="success" class="!font-bold tracking-wide">
          Firefly-panel
        </NGradientText>
      </div>
      <NForm :model="form" label-width="100px" @keydown.enter="handleSubmit">
        <NFormItem>
          <NInput
            v-model:value="form.username"
            :placeholder="(usernameFocus || form.username) ? '' : ($t('login.usernamePlaceholder') || '请输入账号')"
            :input-props="{ autocomplete: 'off' }"
            clearable
            @focus="usernameFocus = true"
            @blur="usernameFocus = false"
            @input="(val) => { if (val) usernameFocus = true }"
          >
            <template #prefix>
              <SvgIcon icon="ph:user-bold" />
            </template>
          </NInput>
        </NFormItem>

        <NFormItem>
          <NInput
            v-model:value="form.password"
            type="password"
            show-password-on="click"
            :placeholder="(passwordFocus || form.password) ? '' : ($t('login.passwordPlaceholder') || '请输入密码')"
            :input-props="{ autocomplete: 'off' }"
            clearable
            @focus="passwordFocus = true"
            @blur="passwordFocus = false"
            @input="(val) => { if (val) passwordFocus = true }"
          >
            <template #prefix>
              <SvgIcon icon="mdi:password-outline" />
            </template>
          </NInput>
        </NFormItem>

        <!-- <NFormItem v-if="isShowCaptcha">
          <div class="w-[120px] h-[34px] mr-[20px] rounded border flex cursor-pointer">
            <Captcha ref="captchaRef" src="/api/captcha/getImage" />
          </div>
          <NInput v-model:value="form.vcode" type="text" placeholder="请输入图像验证码" />
        </NFormItem> -->
        <NFormItem style="margin-top: 10px">
          <NButton type="primary" block :loading="loading" @click="handleSubmit">
            {{ $t('login.loginButton') }}
          </NButton>
        </NFormItem>

        <!-- <div class="flex justify-end">
          <NButton v-if="isShowRegister" quaternary type="info" class="flex" @click="$router.push({ path: '/register' })">
            注册
          </NButton>
          <NButton quaternary type="info" class="flex" @click="$router.push({ path: '/resetPassword' })">
            忘记密码?
          </NButton>
        </div> -->

        <div class="flex flex-col items-center justify-center text-xs text-slate-500 dark:text-slate-300 mt-4 space-y-1">
          <div>初始账号: <strong class="text-slate-700 dark:text-slate-100 font-mono">admin</strong> &nbsp;|&nbsp; 初始密码: <strong class="text-slate-700 dark:text-slate-100 font-mono">12345678</strong></div>
          <div>Powered By <a href="https://github.com/mysecret2022/firefly-panel" target="_blank" class="text-emerald-500 hover:underline">Firefly-panel</a></div>
        </div>
      </NForm>
    </NCard>
  </div>
</template>

<style>
.login-container {
    padding: 20px;
    display: flex;
    justify-content: center;
    align-items: center;
    height: 100vh;
    width: 100vw;
    background: url('@/assets/firefly.webp') no-repeat center center;
    background-size: cover;
    position: relative;
}

/* VisionOS 悬浮极光毛玻璃卡片核心样式 */
.login-card {
    margin: 20px;
    min-width: 380px;
    max-width: 440px;
    border-radius: 28px !important;
    position: relative;
    overflow: hidden;
    
    /* 高透磨砂质感 */
    background: rgba(18, 22, 34, 0.65) !important;
    backdrop-filter: blur(32px) saturate(210%) !important;
    -webkit-backdrop-filter: blur(32px) saturate(210%) !important;
    
    /* 极光双层微光边框与悬浮高光深度投影 */
    border: 1px solid rgba(255, 255, 255, 0.22) !important;
    box-shadow: 
        0 30px 60px -12px rgba(0, 0, 0, 0.55),
        0 0 50px -10px rgba(99, 242, 201, 0.25),
        0 0 35px -8px rgba(251, 191, 36, 0.18),
        inset 0 1.5px 2px rgba(255, 255, 255, 0.55),
        inset 0 -1px 2px rgba(255, 255, 255, 0.15) !important;
    transition: all 0.3s cubic-bezier(0.16, 1, 0.3, 1) !important;
}

/* 极光环形边缘折射伪元素 */
.login-card::before {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: 28px;
    padding: 1.5px;
    background: linear-gradient(
        135deg, 
        rgba(255, 255, 255, 0.7) 0%, 
        rgba(99, 242, 201, 0.55) 30%, 
        rgba(251, 191, 36, 0.45) 60%, 
        rgba(167, 139, 250, 0.55) 100%
    );
    -webkit-mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
    -webkit-mask-composite: xor;
    mask-composite: exclude;
    pointer-events: none;
    opacity: 0.85;
}

/* 白天浅色模式适配 */
html:not(.dark) .login-card {
    background: rgba(255, 255, 255, 0.62) !important;
    box-shadow: 
        0 25px 50px -12px rgba(0, 0, 0, 0.22),
        0 0 45px -10px rgba(99, 242, 201, 0.3),
        inset 0 1.5px 2px rgba(255, 255, 255, 0.8) !important;
}

.login-title {
    text-align: center;
    margin: 20px 0 24px 0;
}

/* 标题半透明透光渐变 */
.login-title .n-gradient-text {
    background: linear-gradient(180deg, #ffffff 0%, rgba(255, 255, 255, 0.72) 100%) !important;
    -webkit-background-clip: text !important;
    -webkit-text-fill-color: transparent !important;
    text-shadow: 0 4px 20px rgba(0, 0, 0, 0.35);
    filter: drop-shadow(0 2px 8px rgba(99, 242, 201, 0.35));
}

/* 玻璃质感输入框 */
.login-card .n-input {
    background: rgba(255, 255, 255, 0.08) !important;
    border-radius: 14px !important;
    border: 1px solid rgba(255, 255, 255, 0.15) !important;
    backdrop-filter: blur(10px) !important;
    transition: all 0.25s ease !important;
}

.login-card .n-input--focus,
.login-card .n-input:focus-within {
    border-color: rgba(99, 242, 201, 0.75) !important;
    box-shadow: 0 0 16px rgba(99, 242, 201, 0.35) !important;
    background: rgba(255, 255, 255, 0.14) !important;
}

/* 极光微光登录按钮 */
.login-card .n-button--primary-type {
    border-radius: 14px !important;
    background: linear-gradient(135deg, #10b981 0%, #059669 50%, #0d9488 100%) !important;
    border: 1px solid rgba(255, 255, 255, 0.3) !important;
    box-shadow: 0 4px 16px rgba(16, 185, 129, 0.4) !important;
    font-weight: 600 !important;
    letter-spacing: 0.05em !important;
}

@media (min-width: 600px) {
    .login-card {
        width: 420px;
        margin: 0px 10px;
    }
}

/* 输入内容或点击聚焦时占位提示消失 */
.login-card .n-input--focus .n-input__placeholder,
.login-card .n-input.n-input--focus .n-input__placeholder,
.login-card .n-input:focus-within .n-input__placeholder,
.login-card input:focus::placeholder,
.login-card input:not(:placeholder-shown)::placeholder {
    display: none !important;
    opacity: 0 !important;
    visibility: hidden !important;
    color: transparent !important;
}
</style>
