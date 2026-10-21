<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import {
  NAlert,
  NButton,
  NButtonGroup,
  NCard,
  NCheckbox,
  NDivider,
  NEllipsis,
  NGrid,
  NGridItem,
  NImage,
  NImageGroup,
  NPopconfirm,
  NSelect,
  NSpin,
  NTabPane,
  NTabs,
  NTag,
  NUpload,
  useDialog,
  useMessage,
} from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import { applySyncToCloud, deletes, getList, uploadToCloud } from '@/api/system/file'
import type { FileItem } from '@/api/system/file'
import { set as savePanelConfig } from '@/api/panel/userConfig'
import { RoundCardModal, SvgIcon } from '@/components/common'
import { copyToClipboard, timeFormat } from '@/utils/cmn'
import { t } from '@/locales'
import { useAuthStore, usePanelState } from '@/store'

interface InfoModalState {
  title: string
  show: boolean
  fileInfo: FileItem | null
}

const authStore = useAuthStore()
const panelStore = usePanelState()
const ms = useMessage()
const dialog = useDialog()

const loading = ref(false)
const rawList = ref<FileItem[]>([])
const selectedIds = ref<number[]>([])
const currentTab = ref<'all' | 'wallpapers' | 'icons'>('all')
const uploadCategory = ref<number>(0) // 0: 壁纸, 1: 图标

const infoModalState = ref<InfoModalState>({
  show: false,
  title: '',
  fileInfo: null,
})

// 根据当前 Tab 过滤展示的列表
const filteredList = computed(() => {
  if (currentTab.value === 'wallpapers')
    return rawList.value.filter(item => item.fileType === 0)
  if (currentTab.value === 'icons')
    return rawList.value.filter(item => item.fileType === 1)
  return rawList.value
})

const isAllSelected = computed(() => {
  if (filteredList.value.length === 0)
    return false
  return filteredList.value.every(item => selectedIds.value.includes(item.id))
})

// 获取文件列表
async function fetchFileList() {
  loading.value = true
  try {
    const res: any = await getList<{ list: FileItem[]; count: number }>()
    if (res.code === 0 && res.data) {
      rawList.value = res.data.list || []
    }
  }
  catch (error) {
    ms.error(t('common.failed'))
  }
  finally {
    loading.value = false
  }
}

// 选中 / 反选
function toggleSelect(id: number) {
  const idx = selectedIds.value.indexOf(id)
  if (idx > -1) {
    selectedIds.value.splice(idx, 1)
  }
  else {
    selectedIds.value.push(id)
  }
}

function handleSelectAll() {
  if (isAllSelected.value) {
    // 取消全选
    const currentTabIds = new Set(filteredList.value.map(i => i.id))
    selectedIds.value = selectedIds.value.filter(id => !currentTabIds.has(id))
  }
  else {
    // 全选当前列表
    const newSet = new Set([...selectedIds.value, ...filteredList.value.map(i => i.id)])
    selectedIds.value = Array.from(newSet)
  }
}

// 复制图片链接
async function copyImageUrl(text: string) {
  const fullUrl = text.startsWith('http') ? text : `${window.location.origin}/${text.replace(/^\//, '')}`
  const res = await copyToClipboard(fullUrl)
  if (res)
    ms.success(t('apps.uploadsFileManager.copySuccess'))
  else
    ms.error(t('apps.uploadsFileManager.copyFailed'))
}

// 设为壁纸
function handleSetWallpaper(imgSrc: string) {
  panelStore.panelConfig.backgroundImageSrc = imgSrc
  savePanelConfig({ panel: panelStore.panelConfig })
  ms.success('已应用为桌面背景壁纸！')
}

// 单张删除
function handleDeleteSingle(id: number) {
  dialog.warning({
    title: t('common.warning'),
    content: `确定要删除该图片吗？删除后将无法恢复。`,
    positiveText: t('common.confirm'),
    negativeText: t('common.cancel'),
    onPositiveClick: async () => {
      await executeDeletes([id])
    },
  })
}

// 批量删除选中的图片
function handleDeleteSelected() {
  if (selectedIds.value.length === 0) {
    ms.warning('请先勾选需要删除的图片')
    return
  }

  const count = selectedIds.value.length
  dialog.warning({
    title: '批量删除确认',
    content: `您已勾选 ${count} 张图片，确定要全部永久删除吗？此操作不可逆！`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      await executeDeletes(selectedIds.value)
    },
  })
}

// 清空当前分类全部图片
function handleClearAllCurrent() {
  const currentIds = filteredList.value.map(i => i.id)
  if (currentIds.length === 0) {
    ms.info('当前列表暂无图片')
    return
  }

  dialog.error({
    title: '危险：清空全部图片',
    content: `确定要清空当前分类下的全部 ${currentIds.length} 张图片吗？此操作不可逆！`,
    positiveText: '彻底清空',
    negativeText: '取消',
    onPositiveClick: async () => {
      await executeDeletes(currentIds)
    },
  })
}

// 执行删除操作
async function executeDeletes(ids: number[]) {
  loading.value = true
  try {
    const res: any = await deletes(ids)
    if (res.code === 0) {
      ms.success(`成功删除 ${ids.length} 张图片`)
      selectedIds.value = selectedIds.value.filter(id => !ids.includes(id))
      fetchFileList()
    }
    else {
      ms.error(`删除失败: ${res.msg}`)
    }
  }
  catch (error) {
    ms.error('操作发生异常，请重试')
  }
  finally {
    loading.value = false
  }
}

// 上传到云端或申请同步
async function handleSyncAction(ids: number[]) {
  if (ids.length === 0) {
    ms.warning('请先勾选图片')
    return
  }

  loading.value = true
  try {
    if (authStore.userInfo?.role === 1) {
      // 管理员直接推送到云端图床
      const res: any = await uploadToCloud(ids)
      if (res.code === 0) {
        ms.success(res.data?.message || '已成功推送到云端图床！')
        fetchFileList()
      }
      else {
        ms.error(`推送到云端失败: ${res.msg}`)
      }
    }
    else {
      // 普通用户提交申请
      const res: any = await applySyncToCloud(ids)
      if (res.code === 0) {
        ms.success(res.data?.message || '已提交同步申请，等待管理员审核！')
        fetchFileList()
      }
      else {
        ms.error(`申请失败: ${res.msg}`)
      }
    }
  }
  catch (error) {
    ms.error('网络请求失败')
  }
  finally {
    loading.value = false
  }
}

// 文件上传完成
function handleUploadFinish({ file, event }: { file: UploadFileInfo; event?: ProgressEvent }) {
  try {
    const target = event?.target as XMLHttpRequest
    if (target && target.response) {
      const res = JSON.parse(target.response)
      if (res.code === 0) {
        ms.success('上传成功')
        fetchFileList()
        return
      }
    }
  }
  catch (e) {}
  fetchFileList()
}

// 切换 Tab 时更新默认上传类型
watch(currentTab, (val) => {
  if (val === 'icons')
    uploadCategory.value = 1
  else
    uploadCategory.value = 0
})

onMounted(() => {
  fetchFileList()
})
</script>

<template>
  <div class="h-full flex flex-col p-3 bg-slate-50 dark:bg-zinc-900 rounded-xl overflow-hidden box-border">
    <!-- 顶部说明与 Tab 切换 -->
    <div class="flex flex-col sm:flex-row sm:items-center justify-between pb-2 border-b border-gray-200 dark:border-zinc-800 gap-2">
      <NTabs v-model:value="currentTab" type="segment" size="small" class="w-full sm:w-auto">
        <NTabPane name="all" tab="🖼️ 全部图片" />
        <NTabPane name="wallpapers" tab="🌄 壁纸图库" />
        <NTabPane name="icons" tab="🎨 应用图标" />
      </NTabs>

      <!-- 上传区域 -->
      <div class="flex items-center gap-2">
        <NSelect
          v-model:value="uploadCategory"
          size="small"
          class="w-28"
          :options="[
            { label: '🌄 壁纸图库', value: 0 },
            { label: '🎨 应用图标', value: 1 },
          ]"
        />
        <NUpload
          action="/api/file/uploadImg"
          :show-file-list="false"
          name="imgfile"
          :data="{ fileType: uploadCategory }"
          :headers="{ token: authStore.token as string }"
          @finish="handleUploadFinish"
        >
          <NButton type="primary" size="small">
            <template #icon>
              <SvgIcon icon="tabler:file-upload" />
            </template>
            上传到此分类
          </NButton>
        </NUpload>
      </div>
    </div>

    <!-- 批量操作工具栏 -->
    <div class="flex flex-wrap items-center justify-between gap-2 py-2 text-xs text-gray-500 dark:text-gray-400">
      <div class="flex items-center gap-2">
        <NButton size="tiny" secondary @click="handleSelectAll">
          {{ isAllSelected ? '取消全选' : '全选当前页' }}
        </NButton>
        <span v-if="selectedIds.length > 0" class="text-green-600 dark:text-green-400 font-medium">
          已勾选 {{ selectedIds.length }} 张
        </span>
      </div>

      <div class="flex items-center gap-2">
        <NButton
          v-if="selectedIds.length > 0"
          size="tiny"
          type="info"
          secondary
          :loading="loading"
          @click="handleSyncAction(selectedIds)"
        >
          <template #icon>
            <SvgIcon icon="cloud" />
          </template>
          {{ authStore.userInfo?.role === 1 ? `批量推送到云端 (${selectedIds.length})` : `申请同步到图床 (${selectedIds.length})` }}
        </NButton>

        <NButton
          v-if="selectedIds.length > 0"
          size="tiny"
          type="error"
          secondary
          :loading="loading"
          @click="handleDeleteSelected"
        >
          <template #icon>
            <SvgIcon icon="material-symbols-delete" />
          </template>
          批量删除 ({{ selectedIds.length }})
        </NButton>

        <NButton
          v-if="filteredList.length > 0"
          size="tiny"
          quaternary
          type="error"
          @click="handleClearAllCurrent"
        >
          清空全部
        </NButton>
      </div>
    </div>

    <!-- 图片瀑布流 / 网格展示 -->
    <div class="flex-1 overflow-y-auto mt-1 p-1">
      <NSpin :show="loading">
        <div v-if="filteredList.length === 0 && !loading" class="text-center py-16 text-gray-400 text-sm">
          暂无图片，点击右上角上传图片或图标吧
        </div>

        <NGrid v-else cols="2 450:3 680:4 900:6" :x-gap="10" :y-gap="10">
          <NGridItem v-for="item in filteredList" :key="item.id">
            <NCard
              size="small"
              class="relative rounded-xl border border-gray-200 dark:border-zinc-800 transition-all hover:shadow-md group overflow-hidden bg-white dark:bg-zinc-800"
              :class="{ 'ring-2 ring-blue-500': selectedIds.includes(item.id) }"
            >
              <!-- 勾选复选框 (左上角浮动) -->
              <div class="absolute top-2 left-2 z-20 bg-white/80 dark:bg-black/80 rounded-md p-0.5 shadow-sm">
                <NCheckbox
                  :checked="selectedIds.includes(item.id)"
                  size="small"
                  @update:checked="() => toggleSelect(item.id)"
                />
              </div>

              <!-- 状态 Tag (右上角浮动) -->
              <div class="absolute top-2 right-2 z-20">
                <NTag v-if="item.syncStatus === 2" size="tiny" type="success" :bordered="false">
                  已上云
                </NTag>
                <NTag v-else-if="item.syncStatus === 1" size="tiny" type="warning" :bordered="false">
                  待审核
                </NTag>
                <NTag v-else-if="item.syncStatus === 3" size="tiny" type="error" :bordered="false">
                  已驳回
                </NTag>
                <NTag v-else size="tiny" type="default" :bordered="false">
                  仅本地
                </NTag>
              </div>

              <!-- 图片预览区域 -->
              <template #cover>
                <div class="h-28 w-full bg-slate-100 dark:bg-zinc-900 flex items-center justify-center p-2 box-border">
                  <NImage
                    :src="item.src"
                    :lazy="true"
                    object-fit="contain"
                    class="h-full w-full flex items-center justify-center"
                  />
                </div>
              </template>

              <!-- 卡片底部文件名与操作栏 -->
              <div class="pt-1">
                <div class="text-xs truncate font-medium text-gray-700 dark:text-gray-300" :title="item.fileName">
                  {{ item.fileName }}
                </div>

                <div class="flex items-center justify-between mt-2 pt-1 border-t border-gray-100 dark:border-zinc-800">
                  <NButtonGroup size="tiny">
                    <NButton quaternary title="复制链接" @click="copyImageUrl(item.src)">
                      <template #icon>
                        <SvgIcon icon="ion-copy" />
                      </template>
                    </NButton>

                    <NButton
                      v-if="item.fileType === 0"
                      quaternary
                      title="设置为桌面壁纸"
                      @click="handleSetWallpaper(item.src)"
                    >
                      <template #icon>
                        <SvgIcon icon="lucide:wallpaper" />
                      </template>
                    </NButton>

                    <NButton
                      quaternary
                      :title="authStore.userInfo?.role === 1 ? '推送到云端图床' : '申请同步到云端'"
                      @click="handleSyncAction([item.id])"
                    >
                      <template #icon>
                        <SvgIcon icon="cloud" />
                      </template>
                    </NButton>

                    <NButton
                      quaternary
                      type="error"
                      title="删除此图片"
                      @click="handleDeleteSingle(item.id)"
                    >
                      <template #icon>
                        <SvgIcon icon="material-symbols-delete" />
                      </template>
                    </NButton>
                  </NButtonGroup>
                </div>
              </div>
            </NCard>
          </NGridItem>
        </NGrid>
      </NSpin>
    </div>
  </div>
</template>

<style scoped>
.group:hover .absolute {
  opacity: 1;
}
</style>
