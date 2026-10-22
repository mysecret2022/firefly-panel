<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { NLayout, NLayoutContent, NLayoutSider, NSpace } from 'naive-ui'
import { useAuthStore } from '@/store'
import { AppLoader, RoundCardModal, SvgIcon } from '@/components/common'
import { t } from '@/locales'
import cloudIcon from '@/assets/cloud.png'

interface App {
  name: string
  componentName: string
  icon: string
  iconImg?: string
  auth?: number
}
const props = defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', visible: boolean): void
}>()

const componentName = ref('UserInfo')
const collapsed = ref(false)
const screenWidth = ref(0)
const isSmallScreen = ref(false)
const defaultTitle = t('appLauncher.title')
const title = ref('')
const height = ref('520px')

const apps = ref<App[]>([
  {
    name: t('apps.userInfo.appName'),
    componentName: 'UserInfo',
    icon: 'material-symbols-person-edit-outline-rounded',
  },
  {
    name: t('apps.baseSettings.appName'),
    componentName: 'Style',
    icon: 'ion-color-palette-outline',
  },
  {
    name: t('apps.itemGroupManage.appName'),
    componentName: 'ItemGroupManage',
    icon: 'material-symbols-ad-group-outline-rounded',
  },
  {
    name: t('apps.uploadsFileManager.appName'),
    componentName: 'UploadFileManager',
    icon: 'tabler:file-upload',
  },
  {
    name: t('apps.exportImport.appName'),
    componentName: 'ImportExport',
    icon: 'icon-park-outline-import-and-export',
  },
])

const authStore = useAuthStore()

const show = computed({
  get: () => props.visible,
  set: (visible: boolean) => {
    emit('update:visible', visible)
  },
})

function handleClickApp(item: App) {
  componentName.value = item.componentName
  if (isSmallScreen.value)
    collapsed.value = true
}

function getScreenWidth() {
  return window.innerWidth
}

function handleResize() {
  screenWidth.value = getScreenWidth()
  if (screenWidth.value < 640) {
    collapsed.value = true
    isSmallScreen.value = true
  }
  else {
    collapsed.value = false
    isSmallScreen.value = false
  }
}

onMounted(() => {
  const adminApp: App = {
    name: t('adminSettingUsers.appName'),
    componentName: 'Users',
    icon: 'lucide-users',
    auth: 1,
  }
  const cloudStorageApp: App = {
    name: '云存储与漫游',
    componentName: 'CloudStorage',
    icon: '',
    iconImg: cloudIcon,
    auth: 1,
  }
  const aboutApp: App = {
    name: t('apps.about.appName'),
    componentName: 'About',
    icon: 'lucide-info',
  }
  // 初始化
  if (authStore.userInfo?.role === 1) {
    apps.value.push(adminApp)
    apps.value.push(cloudStorageApp)
    apps.value.push(aboutApp)
  }
  else {
    apps.value.push(aboutApp)
  }

  window.addEventListener('resize', handleResize)
  handleResize()
})

onUnmounted(() => {
  window.removeEventListener('resize', handleResize)
})
</script>

<template>
  <div>
    <RoundCardModal
      v-model:show="show"
      style="max-width: 900px;"
      size="small"
    >
      <template #header>
        <div class="flex items-center select-none" @click="collapsed = !collapsed">
          <div class="text-3xl cursor-pointer" style="color:var(--n-color-target)">
            <SvgIcon class=" transition-all duration-500" :icon="collapsed ? 'tabler-layout-sidebar-right-collapse-filled' : 'tabler-layout-sidebar-left-collapse-filled'" />
          </div>
          <div class="ml-1">
            {{ title === '' ? defaultTitle : title }}
          </div>
        </div>
      </template>
      <div class="w-full h-full app-starter-modal-content flex flex-col flex-1 min-h-0">
        <NLayout has-sider style="border-radius:0.75rem;" class="h-full flex-1 flex min-h-0">
          <NLayoutSider
            v-model:collapsed="collapsed"
            collapse-mode="width"
            :collapsed-width="0"
            :width="isSmallScreen ? '100%' : 240"
            class="h-full flex flex-col min-h-0"
            content-style="overflow: hidden; height: 100%; display: flex; flex-direction: column;"
          >
            <div class="w-full h-full dark:bg-[#2c2c32] flex flex-col flex-1 min-h-0">
              <div
                class="p-[5px] bg-slate-200 dark:bg-zinc-900 rounded-xl overflow-y-auto flex-1 min-h-0"
                :style="{
                  width: isSmallScreen ? '100%' : '220px',
                  minWidth: '200px',
                }"
              >
                <div
                  v-for=" (item, index) in apps"
                  :key="index"
                  :style="{ color: componentName === item.componentName ? 'var(--n-color-target)' : '' }"
                  @click="handleClickApp(item)"
                >
                  <div
                    class="bg-white dark:bg-zinc-800 p-[10px] rounded-lg mb-[5px] font-bold cursor-pointer flex items-center hover:bg-slate-50 focus:bg-slate-50"
                  >
                    <div class="flex items-center justify-center">
                      <div class="text-lg flex items-center justify-center w-[1.125rem] h-[1.125rem]">
                        <img v-if="item.iconImg" :src="item.iconImg" class="w-full h-full object-contain" alt="" />
                        <SvgIcon v-else :icon="item.icon" />
                      </div>
                      <span class="ml-2">{{ item.name }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </NLayoutSider>
          <NLayoutContent content-style="display: flex; flex-direction: column; min-height: 0; flex: 1; height: 100%;" class="h-full flex flex-col flex-1 min-h-0">
            <div class="rounded-2xl h-full overflow-hidden transition-all duration-500 min-w-[300px] flex-1 flex flex-col min-h-0" :class="(isSmallScreen && !collapsed) ? 'opacity-0' : 'opacity-100'">
              <AppLoader :component-name="componentName" class="h-full flex-1 flex flex-col min-h-0" />
            </div>
          </NLayoutContent>
        </NLayout>
      </div>
    </RoundCardModal>
  </div>
</template>

<style scoped>
.text-shadow {
  text-shadow: 0px 0px 5px gray;
}
</style>

<style>
.dark .app-starter-modal-content .n-layout{
    background-color: #2c2c32;
}
</style>
