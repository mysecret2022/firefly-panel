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
  NInput,
  NModal,
  NPopconfirm,
  NRadio,
  NRadioGroup,
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
import {
  applySyncToCloud,
  changeFileCategory,
  deletes,
  getList,
  renameFile,
  uploadToCloud,
} from '@/api/system/file'
import type { FileItem } from '@/api/system/file'
import { set as savePanelConfig } from '@/api/panel/userConfig'
import { RoundCardModal, SvgIcon } from '@/components/common'
import cloudIcon from '@/assets/cloud.png'
import { t } from '@/locales'
import { useAuthStore, usePanelState } from '@/store'

const authStore = useAuthStore()
const panelStore = usePanelState()
const ms = useMessage()
const dialog = useDialog()

const loading = ref(false)
const rawList = ref<FileItem[]>([])
const selectedIds = ref<number[]>([])
const currentTab = ref<'all' | 'wallpapers' | 'icons'>('all')
const uploadCategory = ref<number>(0) // 0: 壁纸, 1: 图标

// 重命名弹窗
const renameModalVisible = ref(false)
const renameTargetId = ref<number | null>(null)
const renameNewName = ref('')

// 移动分类弹窗
const moveModalVisible = ref(false)
const moveTargetIds = ref<number[]>([])
const targetCategory = ref<number>(1)

// 动态可调排布列数
const customGridCols = ref<string>('auto')
const currentGridCols = computed(() => {
  if (customGridCols.value === 'auto')
    return '2 380:3 560:4 740:5 920:6 1100:7 1280:8 1460:9 1640:10 1820:11 2000:12'
  return customGridCols.value
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
  if (idx > -1)
    selectedIds.value.splice(idx, 1)
  else
    selectedIds.value.push(id)
}

function handleSelectAll() {
  if (isAllSelected.value) {
    const currentTabIds = new Set(filteredList.value.map(i => i.id))
    selectedIds.value = selectedIds.value.filter(id => !currentTabIds.has(id))
  }
  else {
    const newSet = new Set([...selectedIds.value, ...filteredList.value.map(i => i.id)])
    selectedIds.value = Array.from(newSet)
  }
}

// 设为壁纸
function handleSetWallpaper(imgSrc: string) {
  panelStore.panelConfig.backgroundImageSrc = imgSrc
  savePanelConfig({ panel: panelStore.panelConfig })
  ms.success('已应用为桌面背景壁纸！')
}

// 打开重命名弹窗
function openRenameModal(item: FileItem) {
  renameTargetId.value = item.id
  renameNewName.value = item.fileName
  renameModalVisible.value = true
}

async function handleConfirmRename() {
  if (!renameTargetId.value || !renameNewName.value.trim()) {
    ms.warning('文件名不能为空')
    return
  }
  loading.value = true
  try {
    const res: any = await renameFile(renameTargetId.value, renameNewName.value.trim())
    if (res.code === 0) {
      ms.success('重命名成功')
      renameModalVisible.value = false
      fetchFileList()
    }
    else {
      ms.error(`重命名失败: ${res.msg}`)
    }
  }
  catch (e) {
    ms.error('操作失败')
  }
  finally {
    loading.value = false
  }
}

// 打开移动分类弹窗
function openMoveModal(ids: number[]) {
  if (ids.length === 0) {
    ms.warning('请先勾选需要移动的图片')
    return
  }
  moveTargetIds.value = ids
  targetCategory.value = currentTab.value === 'wallpapers' ? 1 : 0
  moveModalVisible.value = true
}

async function handleConfirmMove() {
  loading.value = true
  try {
    const res: any = await changeFileCategory(moveTargetIds.value, targetCategory.value)
    if (res.code === 0) {
      ms.success(`已成功移动 ${moveTargetIds.value.length} 张图片分类`)
      moveModalVisible.value = false
      selectedIds.value = []
      fetchFileList()
    }
    else {
      ms.error(`移动失败: ${res.msg}`)
    }
  }
  catch (e) {
    ms.error('操作失败')
  }
  finally {
    loading.value = false
  }
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
  <div class="h-full flex flex-col flex-1 min-h-0 p-2 bg-slate-50 dark:bg-zinc-900 rounded-xl overflow-hidden box-border">
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

    <!-- 批量操作工具栏 (字体大 1 号，统一 size="small") -->
    <div class="flex flex-wrap items-center justify-between gap-2 py-2 text-sm text-gray-500 dark:text-gray-400">
      <div class="flex items-center gap-2">
        <NButton size="small" secondary @click="handleSelectAll">
          {{ isAllSelected ? '取消全选' : '全选当前页' }}
        </NButton>
        <span v-if="selectedIds.length > 0" class="text-green-600 dark:text-green-400 font-medium">
          已勾选 {{ selectedIds.length }} 张
        </span>

        <!-- 动态列数调节 -->
        <div class="hidden sm:flex items-center gap-1.5 ml-2 text-xs">
          <span>排布:</span>
          <NSelect
            v-model:value="customGridCols"
            size="tiny"
            class="w-24"
            :options="[
              { label: '自适应', value: 'auto' },
              { label: '4 列', value: '4' },
              { label: '6 列', value: '6' },
              { label: '8 列', value: '8' },
              { label: '10 列', value: '10' },
              { label: '12 列', value: '12' },
            ]"
          />
        </div>
      </div>

      <div class="flex items-center gap-2">
        <!-- 移动分类 (字号大 1 号) -->
        <NButton
          v-if="selectedIds.length > 0"
          size="small"
          secondary
          @click="openMoveModal(selectedIds)"
        >
          <template #icon>
            <SvgIcon icon="material-symbols:drive-file-move-outline" />
          </template>
          移动分类 ({{ selectedIds.length }})
        </NButton>

        <!-- 批量推送到云端 (字号大 1 号) -->
        <NButton
          v-if="selectedIds.length > 0"
          size="small"
          type="info"
          secondary
          :loading="loading"
          @click="handleSyncAction(selectedIds)"
        >
          <template #icon>
            <img :src="cloudIcon" style="width: 14px; height: 14px; object-fit: contain;" />
          </template>
          {{ authStore.userInfo?.role === 1 ? `批量推送到云端 (${selectedIds.length})` : `申请同步到图床 (${selectedIds.length})` }}
        </NButton>

        <!-- 批量删除 (字号大 1 号) -->
        <NButton
          v-if="selectedIds.length > 0"
          size="small"
          type="error"
          secondary
          :loading="loading"
          @click="handleDeleteSelected"
        >
          <template #icon>
            <SvgIcon icon="material-symbols:delete" />
          </template>
          批量删除 ({{ selectedIds.length }})
        </NButton>

        <!-- 清空全部 (字号大 1 号) -->
        <NButton
          v-if="filteredList.length > 0"
          size="small"
          quaternary
          type="error"
          @click="handleClearAllCurrent"
        >
          清空全部
        </NButton>
      </div>
    </div>

    <!-- 图片瀑布流 / 网格展示 (减小多余空白，图片更饱满) -->
    <div class="flex-1 min-h-0 overflow-y-auto mt-1 p-0.5">
      <NSpin :show="loading">
        <div v-if="filteredList.length === 0 && !loading" class="text-center py-16 text-gray-400 text-sm">
          暂无图片，点击右上角上传图片或图标吧
        </div>

        <NGrid v-else :cols="currentGridCols" :x-gap="8" :y-gap="8">
          <NGridItem v-for="item in filteredList" :key="item.id">
            <NCard
              size="small"
              class="relative rounded-xl border border-gray-200 dark:border-zinc-800 transition-all hover:shadow-md group overflow-hidden bg-white dark:bg-zinc-800 p-1"
              :class="{ 'ring-2 ring-blue-500': selectedIds.includes(item.id) }"
            >
              <!-- 勾选复选框 (左上角浮动，大 1 号 size="medium") -->
              <div class="absolute top-1.5 left-1.5 z-20 bg-white/90 dark:bg-black/90 rounded-md p-0.5 shadow-sm">
                <NCheckbox
                  :checked="selectedIds.includes(item.id)"
                  size="medium"
                  @update:checked="() => toggleSelect(item.id)"
                />
              </div>

              <!-- 状态 Tag (右上角浮动，已上云采用高亮艳丽红) -->
              <div class="absolute top-1.5 right-1.5 z-20">
                <NTag
                  v-if="item.syncStatus === 2"
                  size="tiny"
                  :color="{ color: '#ff3838', textColor: '#ffffff' }"
                  :bordered="false"
                  class="font-bold shadow-xs"
                >
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

              <!-- 图片预览区域 (使用 Naive UI 原生 NImage，支持滚轮/工具栏自由放大缩小旋转与全屏展示) -->
              <template #cover>
                <div class="h-32 w-full bg-slate-100/70 dark:bg-zinc-900/70 flex items-center justify-center p-1.5 box-border overflow-hidden">
                  <NImage
                    :src="item.src"
                    :preview-src="item.src"
                    object-fit="contain"
                    class="h-full w-full flex items-center justify-center"
                    :lazy="true"
                    show-toolbar-tooltip
                  />
                </div>
              </template>

              <!-- 卡片底部文件名与操作栏 -->
              <div class="pt-1">
                <div
                  class="text-xs truncate font-medium text-gray-700 dark:text-gray-300 cursor-pointer hover:text-blue-500 transition px-0.5"
                  :title="`${item.fileName} (点击重命名)`"
                  @click="openRenameModal(item)"
                >
                  {{ item.fileName }}
                </div>

                <div class="flex items-center justify-between mt-1 pt-1 border-t border-gray-100 dark:border-zinc-800">
                  <NButtonGroup size="tiny">
                    <!-- 重命名 -->
                    <NButton quaternary title="重命名文件名" @click="openRenameModal(item)">
                      <template #icon>
                        <SvgIcon icon="basil:edit-solid" />
                      </template>
                    </NButton>

                    <!-- 移动分类图标与 Tab 一致：壁纸用 🎨，图标用 🌄 -->
                    <NButton
                      quaternary
                      :title="item.fileType === 0 ? '一键移入【应用图标】' : '一键移入【壁纸图库】'"
                      @click="openMoveModal([item.id])"
                    >
                      <span class="text-xs">
                        {{ item.fileType === 0 ? '🎨' : '🌄' }}
                      </span>
                    </NButton>

                    <!-- 设为壁纸 (壁纸分类专属) -->
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

                    <!-- 推送到云端图床 / 申请同步 (使用 cloud.png 图标) -->
                    <NButton
                      quaternary
                      :title="authStore.userInfo?.role === 1 ? '推送到云端图床' : '申请同步到云端'"
                      @click="handleSyncAction([item.id])"
                    >
                      <template #icon>
                        <img :src="cloudIcon" style="width: 14px; height: 14px; object-fit: contain;" />
                      </template>
                    </NButton>

                    <!-- 删除此图片 -->
                    <NButton
                      quaternary
                      type="error"
                      title="删除此图片"
                      @click="handleDeleteSingle(item.id)"
                    >
                      <template #icon>
                        <SvgIcon icon="material-symbols:delete" />
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

    <!-- 重命名模态框 -->
    <NModal
      v-model:show="renameModalVisible"
      preset="card"
      title="修改文件名"
      style="max-width: 400px; border-radius: 12px;"
    >
      <div class="space-y-3">
        <div class="text-xs text-gray-500">
          请输入新的文件名：
        </div>
        <NInput v-model:value="renameNewName" placeholder="例如: my-icon.png" @keyup.enter="handleConfirmRename" />
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton secondary size="small" @click="renameModalVisible = false">
            取消
          </NButton>
          <NButton type="primary" size="small" :loading="loading" @click="handleConfirmRename">
            保存
          </NButton>
        </div>
      </template>
    </NModal>

    <!-- 移动分类模态框 -->
    <NModal
      v-model:show="moveModalVisible"
      preset="card"
      title="移动图片到其他分类"
      style="max-width: 400px; border-radius: 12px;"
    >
      <div class="space-y-3">
        <div class="text-xs text-gray-500">
          请选择目标分类：
        </div>
        <NRadioGroup v-model:value="targetCategory">
          <div class="space-y-2">
            <div>
              <NRadio :value="0">
                🌄 壁纸图库 (桌面大图壁纸)
              </NRadio>
            </div>
            <div>
              <NRadio :value="1">
                🎨 应用图标 (导航卡片小图标)
              </NRadio>
            </div>
          </div>
        </NRadioGroup>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton secondary size="small" @click="moveModalVisible = false">
            取消
          </NButton>
          <NButton type="primary" size="small" :loading="loading" @click="handleConfirmMove">
            确认移动
          </NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.group:hover .absolute {
  opacity: 1;
}
</style>
