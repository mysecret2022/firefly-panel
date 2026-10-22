<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NBadge,
  NButton,
  NButtonGroup,
  NCard,
  NCheckbox,
  NForm,
  NFormItem,
  NGrid,
  NGridItem,
  NImage,
  NInput,
  NModal,
  NPopconfirm,
  NRadio,
  NRadioGroup,
  NSpace,
  NSpin,
  NSwitch,
  NTabPane,
  NTabs,
  NTag,
  useDialog,
  useMessage,
} from 'naive-ui'
import { SvgIcon } from '@/components/common'
import type { CloudFileItem, CloudStorageConfig } from '@/api/panel/cloudStorage'
import {
  clearCloudImages,
  deleteCloudImages,
  getCloudConfig,
  listCloudBackups,
  listCloudImages,
  restoreEncryptedBackup,
  saveCloudConfig,
  syncAllImagesToLocal,
  syncImageToLocal,
  testCloudConnection,
  uploadEncryptedBackup,
} from '@/api/panel/cloudStorage'
import { auditAction, getAuditList } from '@/api/system/file'

const ms = useMessage()
const dialog = useDialog()

// 状态管理
const loading = ref(false)
const testLoading = ref(false)
const saveLoading = ref(false)
const backupUploading = ref(false)
const syncAllLoading = ref(false)
const activeTab = ref('images') // 默认进入云端漫游图床
const cloudCategory = ref<'all' | 'wallpapers' | 'icons'>('all')

// 配置表单
const form = ref<CloudStorageConfig>({
  type: 'github',
  enabled: false,
  github: {
    owner: '',
    repo: '',
    branch: 'main',
    token: '',
    basePath: 'panel-data',
    useCDN: false,
  },
  cloudflareR2: {
    accountId: '',
    accessKeyId: '',
    secretAccessKey: '',
    bucketName: '',
    publicDomain: '',
  },
})

// 仓库公私检测结果
const repoCheckResult = ref<{ tested: boolean; isPrivate: boolean; desc: string }>({
  tested: false,
  isPrivate: false,
  desc: '',
})

// 备份与图片列表
const cloudBackups = ref<CloudFileItem[]>([])
const cloudImages = ref<CloudFileItem[]>([])
const selectedCloudPaths = ref<string[]>([])
const backupsLoading = ref(false)
const imagesLoading = ref(false)

// 审核列表
const auditList = ref<any[]>([])
const selectedAuditIds = ref<number[]>([])
const auditLoading = ref(false)

// 加密备份表单
const backupPassword = ref('')
const backupPasswordConfirm = ref('')

// 解密还原弹窗
const restoreModalVisible = ref(false)
const selectedBackupFilename = ref('')
const restorePassword = ref('')
const restoreSubmitting = ref(false)

// 格式化文件大小
function formatSize(bytes: number) {
  if (!bytes || bytes === 0)
    return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`
}

// 格式化时间
function formatDate(d: string | Date) {
  if (!d)
    return '-'
  const date = new Date(d)
  return date.toLocaleString('zh-CN', { hour12: false })
}

// 计算云端总容量
const totalCloudSize = computed(() => {
  const sum = cloudImages.value.reduce((acc, cur) => acc + (cur.size || 0), 0)
  return formatSize(sum)
})

// 根据分类筛选云端图片
const filteredCloudImages = computed(() => {
  if (cloudCategory.value === 'wallpapers') {
    return cloudImages.value.filter(img =>
      img.path.includes('/wallpapers/')
      || img.name.includes('wallpaper')
      || img.size > 200 * 1024,
    )
  }
  if (cloudCategory.value === 'icons') {
    return cloudImages.value.filter(img =>
      img.path.includes('/icons/')
      || img.name.includes('icon')
      || img.size <= 200 * 1024,
    )
  }
  return cloudImages.value
})

// 加载配置
async function fetchConfig() {
  loading.value = true
  try {
    const res: any = await getCloudConfig()
    if (res.code === 0 && res.data) {
      if (!res.data.github) {
        res.data.github = {
          owner: '',
          repo: '',
          branch: 'main',
          token: '',
          basePath: 'panel-data',
          useCDN: false,
        }
      }
      if (!res.data.cloudflareR2) {
        res.data.cloudflareR2 = {
          accountId: '',
          accessKeyId: '',
          secretAccessKey: '',
          bucketName: '',
          publicDomain: '',
        }
      }
      form.value = res.data
    }
  }
  catch (error) {
    ms.error('获取配置失败')
  }
  finally {
    loading.value = false
  }
}

// 测试连接
async function handleTestConnection() {
  testLoading.value = true
  try {
    const res: any = await testCloudConnection(form.value)
    if (res.code === 0 && res.data) {
      repoCheckResult.value = {
        tested: true,
        isPrivate: res.data.isPrivate,
        desc: res.data.description,
      }
      ms.success(res.data.message || `连接测试成功: ${res.data.description}`)
    }
    else {
      repoCheckResult.value.tested = false
      ms.error(`连接失败: ${res.msg || '无法连接到存储后端'}`)
    }
  }
  catch (error: any) {
    repoCheckResult.value.tested = false
    ms.error(`连接错误: ${error.message || '请检查配置参数'}`)
  }
  finally {
    testLoading.value = false
  }
}

// 保存配置
async function handleSaveConfig() {
  saveLoading.value = true
  try {
    const res: any = await saveCloudConfig(form.value)
    if (res.code === 0) {
      ms.success('云存储配置已保存！')
    }
    else {
      ms.error(`保存失败: ${res.msg}`)
    }
  }
  catch (error) {
    ms.error('保存配置失败')
  }
  finally {
    saveLoading.value = false
  }
}

// 获取云端备份列表
async function fetchBackups() {
  backupsLoading.value = true
  try {
    const res: any = await listCloudBackups()
    if (res.code === 0) {
      cloudBackups.value = res.data || []
    }
  }
  catch (error) {
    ms.error('获取备份列表失败')
  }
  finally {
    backupsLoading.value = false
  }
}

// 获取云端图片列表
async function fetchImages() {
  imagesLoading.value = true
  try {
    const res: any = await listCloudImages()
    if (res.code === 0) {
      cloudImages.value = res.data || []
      selectedCloudPaths.value = []
    }
  }
  catch (error) {
    ms.error('获取云端图片列表失败')
  }
  finally {
    imagesLoading.value = false
  }
}

// 云端图片多选
function toggleSelectCloudImage(path: string) {
  const idx = selectedCloudPaths.value.indexOf(path)
  if (idx > -1)
    selectedCloudPaths.value.splice(idx, 1)
  else
    selectedCloudPaths.value.push(path)
}

function handleSelectAllCloudImages() {
  const currentPaths = filteredCloudImages.value.map(i => i.path)
  if (selectedCloudPaths.value.length === currentPaths.length && currentPaths.length > 0)
    selectedCloudPaths.value = []
  else
    selectedCloudPaths.value = currentPaths
}

// 批量删除选中的云端图片
function handleDeleteSelectedCloudImages() {
  if (selectedCloudPaths.value.length === 0) {
    ms.warning('请先勾选需要删除的云端图片')
    return
  }
  const count = selectedCloudPaths.value.length
  dialog.warning({
    title: '批量删除云端图片',
    content: `确定要从云端永久删除选中的 ${count} 张图片吗？(为避免触发 GitHub 频次限制，系统将保守限流删除)`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      imagesLoading.value = true
      try {
        const res: any = await deleteCloudImages(selectedCloudPaths.value)
        if (res.code === 0) {
          ms.success(res.data?.message || `已成功删除 ${res.data?.successCount} 张图片`)
          selectedCloudPaths.value = []
          fetchImages()
        }
        else {
          ms.error(`删除失败: ${res.msg}`)
        }
      }
      catch (e) {
        ms.error('操作失败')
      }
      finally {
        imagesLoading.value = false
      }
    },
  })
}

// 单张删除云端图片
function handleDeleteSingleCloudImage(p: string) {
  dialog.warning({
    title: '删除云端图片',
    content: '确定要从云端删除该图片吗？',
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      imagesLoading.value = true
      try {
        const res: any = await deleteCloudImages([p])
        if (res.code === 0) {
          ms.success('已从云端删除')
          fetchImages()
        }
        else {
          ms.error(`删除失败: ${res.msg}`)
        }
      }
      catch (e) {
        ms.error('操作失败')
      }
      finally {
        imagesLoading.value = false
      }
    },
  })
}

// 清空云端所有图片
function handleClearAllCloudImages() {
  if (cloudImages.value.length === 0) {
    ms.info('当前云端无图片')
    return
  }
  dialog.error({
    title: '危险：清空云端图床',
    content: `确定要清空云端存储中的全部 ${cloudImages.value.length} 张图片吗？此操作不可逆！`,
    positiveText: '彻底清空',
    negativeText: '取消',
    onPositiveClick: async () => {
      imagesLoading.value = true
      try {
        const res: any = await clearCloudImages()
        if (res.code === 0) {
          ms.success(res.data?.message || '已清空云端图床')
          selectedCloudPaths.value = []
          fetchImages()
        }
        else {
          ms.error(`清空失败: ${res.msg}`)
        }
      }
      catch (e) {
        ms.error('操作失败')
      }
      finally {
        imagesLoading.value = false
      }
    },
  })
}

// 获取待审核图片列表
async function fetchAuditList() {
  auditLoading.value = true
  try {
    const res: any = await getAuditList<{ list: any[]; count: number }>()
    if (res.code === 0 && res.data) {
      auditList.value = res.data.list || []
      selectedAuditIds.value = []
    }
  }
  catch (e) {}
  finally {
    auditLoading.value = false
  }
}

// 待审图片勾选
function toggleSelectAudit(id: number) {
  const idx = selectedAuditIds.value.indexOf(id)
  if (idx > -1)
    selectedAuditIds.value.splice(idx, 1)
  else
    selectedAuditIds.value.push(id)
}

function handleSelectAllAudit() {
  if (selectedAuditIds.value.length === auditList.value.length && auditList.value.length > 0)
    selectedAuditIds.value = []
  else
    selectedAuditIds.value = auditList.value.map(i => i.id)
}

// 审核操作 (approve / reject)
async function handleAuditAction(ids: number[], action: 'approve' | 'reject') {
  if (ids.length === 0) {
    ms.warning('请先勾选待审核的图片')
    return
  }
  auditLoading.value = true
  try {
    const res: any = await auditAction(ids, action)
    if (res.code === 0) {
      ms.success(res.data?.message || '操作成功')
      fetchAuditList()
      fetchImages()
    }
    else {
      ms.error(`处理失败: ${res.msg}`)
    }
  }
  catch (e) {
    ms.error('操作失败')
  }
  finally {
    auditLoading.value = false
  }
}

// 一键全部批准
function handleApproveAll() {
  const allIds = auditList.value.map(i => i.id)
  if (allIds.length === 0) {
    ms.info('当前暂无待审核申请')
    return
  }
  dialog.info({
    title: '一键全部批准',
    content: `确定批准并将这 ${allIds.length} 张图片全部推送到云端公共图库吗？`,
    positiveText: '确认全部批准',
    negativeText: '取消',
    onPositiveClick: () => {
      handleAuditAction(allIds, 'approve')
    },
  })
}

// 上传加密备份
async function handleUploadBackup() {
  if (!backupPassword.value) {
    ms.warning('请输入加密密码')
    return
  }
  if (backupPassword.value.length < 6) {
    ms.warning('为了数据安全，加密密码长度至少需要 6 位')
    return
  }
  if (backupPassword.value !== backupPasswordConfirm.value) {
    ms.error('两次输入的密码不一致，请核对')
    return
  }

  backupUploading.value = true
  try {
    const res: any = await uploadEncryptedBackup(backupPassword.value)
    if (res.code === 0) {
      ms.success(res.data?.message || '已成功加密并上传到云存储！')
      backupPassword.value = ''
      backupPasswordConfirm.value = ''
      fetchBackups()
    }
    else {
      ms.error(`备份失败: ${res.msg}`)
    }
  }
  catch (error: any) {
    ms.error(`上传备份异常: ${error.message}`)
  }
  finally {
    backupUploading.value = false
  }
}

// 打开还原弹窗
function openRestoreModal(item: CloudFileItem) {
  selectedBackupFilename.value = item.name
  restorePassword.value = ''
  restoreModalVisible.value = true
}

// 确认解密还原
async function handleConfirmRestore() {
  if (!restorePassword.value) {
    ms.warning('请输入解密密码')
    return
  }

  restoreSubmitting.value = true
  try {
    const res: any = await restoreEncryptedBackup(selectedBackupFilename.value, restorePassword.value)
    if (res.code === 0) {
      ms.success('数据已成功解密并恢复！页面即将重新加载以刷新视图...')
      restoreModalVisible.value = false
      setTimeout(() => {
        window.location.reload()
      }, 1500)
    }
    else {
      ms.error(`恢复失败: ${res.msg}`)
    }
  }
  catch (error: any) {
    ms.error(`恢复异常: ${error.message}`)
  }
  finally {
    restoreSubmitting.value = false
  }
}

// 同步单个图片到本地
async function handleSyncSingleImage(img: CloudFileItem) {
  try {
    const res: any = await syncImageToLocal(img.name)
    if (res.code === 0) {
      ms.success(`图片已成功下载到本地存储: ${img.name}`)
    }
    else {
      ms.error(`同步失败: ${res.msg}`)
    }
  }
  catch (error) {
    ms.error('下载图片异常')
  }
}

// 一键全部拉取到本地
async function handleSyncAllImages() {
  syncAllLoading.value = true
  try {
    const res: any = await syncAllImagesToLocal()
    if (res.code === 0) {
      ms.success(res.data?.message || '全部图片已同步完成')
    }
    else {
      ms.error(`同步失败: ${res.msg}`)
    }
  }
  catch (error) {
    ms.error('同步异常')
  }
  finally {
    syncAllLoading.value = false
  }
}

// 大图预览浮层 (单击放大，再点击任意位置/大图秒退缩小)
const previewVisible = ref(false)
const previewImage = ref<CloudFileItem | null>(null)

function openPreview(img: CloudFileItem) {
  previewImage.value = img
  previewVisible.value = true
}

function closePreview() {
  previewVisible.value = false
  previewImage.value = null
}

// 删除单个备份
function handleDeleteSingleBackup(b: CloudFileItem) {
  dialog.warning({
    title: '删除云端加密备份',
    content: `确定要永久删除备份包「${b.name}」吗？删除后将无法通过该包还原数据。`,
    positiveText: '确认删除',
    negativeText: '取消',
    onPositiveClick: async () => {
      backupsLoading.value = true
      try {
        const res: any = await deleteCloudImages([b.path])
        if (res.code === 0) {
          ms.success('备份包已成功从云端删除')
          fetchBackups()
        }
        else {
          ms.error(`删除失败: ${res.msg}`)
        }
      }
      catch (error) {
        ms.error('操作异常')
      }
      finally {
        backupsLoading.value = false
      }
    },
  })
}

// 下载备份文件
function handleDownloadBackup(b: CloudFileItem) {
  if (b.downloadUrl) {
    window.open(b.downloadUrl, '_blank')
  }
}

// Tab 切换监听
function onTabChange(tab: string) {
  if (tab === 'images')
    fetchImages()
  else if (tab === 'audit')
    fetchAuditList()
  else if (tab === 'backups')
    fetchBackups()
  else if (tab === 'config')
    fetchConfig()
}

onMounted(() => {
  fetchImages()
  fetchAuditList()
})
</script>

<template>
  <div class="h-full flex flex-col overflow-y-auto px-3 py-2 box-border">
    <NSpin :show="loading" class="h-full flex flex-col flex-1">
      <NTabs v-model:value="activeTab" type="line" animated class="h-full flex flex-col flex-1" pane-class="h-full flex flex-col flex-1" @update:value="onTabChange">
        <!-- 第 1 页: 云端漫游图床 (默认优先展示) -->
        <NTabPane name="images" tab="🖼️ 云端漫游图床" class="h-full flex flex-col flex-1 min-h-0">
          <NCard size="small" class="mt-1 shadow-sm rounded-xl flex-1 min-h-0 flex flex-col h-full" content-style="flex: 1; display: flex; flex-direction: column; min-height: 0; padding: 0.5rem; overflow: hidden;">
            <template #header>
              <div class="flex items-center py-0.5">
                <!-- 上层/左侧: 全部/壁纸图床/应用图床 -->
                <NTabs v-model:value="cloudCategory" type="segment" size="small" class="w-auto">
                  <NTabPane name="all" tab="全部" />
                  <NTabPane name="wallpapers" tab="🌄 壁纸图床" />
                  <NTabPane name="icons" tab="🎨 应用图床" />
                </NTabs>
              </div>
            </template>

            <template #header-extra>
              <div class="flex items-center gap-2 whitespace-nowrap overflow-x-auto py-0.5">
                <!-- 已漫游与容量小状态 (与刷新、全选等按钮排布在同一行) -->
                <div class="hidden md:flex items-center text-xs text-gray-500 dark:text-gray-400 gap-2 px-2 py-1 rounded-md bg-gray-100/70 dark:bg-zinc-800/60 border border-gray-200/60 dark:border-zinc-700/50">
                  <span>已漫游: <strong class="text-blue-500 font-semibold">{{ cloudImages.length }}</strong> 张</span>
                  <span>容量: <strong class="text-emerald-500 font-semibold">{{ totalCloudSize }}</strong></span>
                  <span class="flex items-center gap-1 text-emerald-600 dark:text-emerald-400 font-medium">
                    <span class="w-1.5 h-1.5 rounded-full bg-emerald-500 animate-pulse"></span>
                    云端畅通
                  </span>
                </div>

                <NButton size="small" secondary @click="fetchImages">
                  <template #icon>
                    <SvgIcon icon="material-symbols:refresh" />
                  </template>
                  刷新
                </NButton>
                <NButton size="small" secondary @click="handleSelectAllCloudImages">
                  {{ selectedCloudPaths.length === filteredCloudImages.length && filteredCloudImages.length > 0 ? '取消全选' : '全选当前' }}
                </NButton>
                <NButton
                  v-if="selectedCloudPaths.length > 0"
                  size="small"
                  type="error"
                  secondary
                  @click="handleDeleteSelectedCloudImages"
                >
                  <template #icon>
                    <SvgIcon icon="material-symbols:delete" />
                  </template>
                  批量删除 ({{ selectedCloudPaths.length }})
                </NButton>
                <NButton
                  v-if="cloudImages.length > 0"
                  size="small"
                  quaternary
                  type="error"
                  @click="handleClearAllCloudImages"
                >
                  清空云端
                </NButton>
                <NPopconfirm @positive-click="handleSyncAllImages">
                  <template #trigger>
                    <NButton size="small" type="primary" :loading="syncAllLoading">
                      <template #icon>
                        <SvgIcon icon="material-symbols:cloud-download" />
                      </template>
                      一键全部拉取到本地
                    </NButton>
                  </template>
                  确认将云端存储中的全部图片一次性下载并同步到本地存储吗？
                </NPopconfirm>
              </div>
            </template>

            <NAlert type="info" size="small" class="mb-2 shrink-0">
              支持按<strong>壁纸图床</strong>和<strong>应用图床</strong>分类筛选。多选批量删除已开启安全防频次限制限流保护，确保稳妥删除。
            </NAlert>

            <div class="flex-1 min-h-0 overflow-y-auto pr-0.5">
              <NSpin :show="imagesLoading">
                <div v-if="filteredCloudImages.length === 0" class="py-12 text-center text-slate-400">
                  <p>该分类下暂无云端图片</p>
                </div>

                <!-- 自适应弹性网格: 随弹窗宽度自增列数平铺成一排 -->
                <NGrid v-else cols="2 450:3 680:4 900:6 1200:8 1500:10" :x-gap="8" :y-gap="8">
                <NGridItem v-for="img in filteredCloudImages" :key="img.path">
                  <div
                    class="relative border rounded-xl p-1.5 bg-slate-50 dark:bg-zinc-800/80 hover:shadow-md transition-all group overflow-hidden border-gray-200 dark:border-zinc-700"
                    :class="{ 'ring-2 ring-blue-500': selectedCloudPaths.includes(img.path) }"
                  >
                    <!-- 大 Checkbox (左上角多选) -->
                    <div class="absolute top-1.5 left-1.5 z-10 bg-white/90 dark:bg-black/90 rounded-md p-0.5 shadow-sm">
                      <NCheckbox
                        :checked="selectedCloudPaths.includes(img.path)"
                        size="medium"
                        @update:checked="() => toggleSelectCloudImage(img.path)"
                      />
                    </div>

                    <!-- 纯净缩略图 (已移除覆盖在上面的图标字体Tag，点击全屏放大) -->
                    <div
                      class="w-full h-28 flex items-center justify-center bg-zinc-200/50 dark:bg-zinc-900 rounded-lg overflow-hidden cursor-pointer"
                      @click="openPreview(img)"
                    >
                      <img
                        :src="img.downloadUrl"
                        :alt="img.name"
                        loading="lazy"
                        class="w-full h-full object-cover transition-transform duration-200 group-hover:scale-105"
                      />
                    </div>

                    <!-- 图片文件名与大小 -->
                    <div class="mt-1.5 px-0.5 truncate font-mono text-xs text-slate-700 dark:text-zinc-200" :title="img.name">
                      {{ img.name }}
                    </div>
                    <div class="flex items-center justify-between text-xs text-slate-400 mt-1 pt-1 border-t border-gray-100 dark:border-zinc-700/60 px-0.5">
                      <span>{{ formatSize(img.size) }}</span>
                      <div class="flex items-center gap-1">
                        <NButton text type="primary" size="tiny" @click.stop="handleSyncSingleImage(img)">
                          下载
                        </NButton>
                        <NButton text type="error" size="tiny" @click.stop="handleDeleteSingleCloudImage(img.path)">
                          删除
                        </NButton>
                      </div>
                    </div>
                  </div>
                </NGridItem>
              </NGrid>
            </NSpin>
          </div>
          </NCard>
        </NTabPane>

        <!-- 第 2 页: 待审图片 (管理员一键审批中心) -->
        <NTabPane name="audit">
          <template #tab>
            <span>
              📋 待审图片
              <NBadge v-if="auditList.length > 0" :value="auditList.length" type="warning" />
            </span>
          </template>
          <NCard title="普通用户同步申请审批中心" size="small" class="mt-2 shadow-sm rounded-lg">
            <template #header-extra>
              <NSpace align="center">
                <NButton size="tiny" secondary @click="fetchAuditList">
                  刷新
                </NButton>
                <NButton size="tiny" secondary @click="handleSelectAllAudit">
                  {{ selectedAuditIds.length === auditList.length && auditList.length > 0 ? '取消全选' : '全选' }}
                </NButton>
                <NButton
                  v-if="selectedAuditIds.length > 0"
                  size="tiny"
                  type="success"
                  secondary
                  @click="handleAuditAction(selectedAuditIds, 'approve')"
                >
                  批准选中项 ({{ selectedAuditIds.length }})
                </NButton>
                <NButton
                  v-if="selectedAuditIds.length > 0"
                  size="tiny"
                  type="error"
                  secondary
                  @click="handleAuditAction(selectedAuditIds, 'reject')"
                >
                  驳回选中项 ({{ selectedAuditIds.length }})
                </NButton>
                <NButton
                  v-if="auditList.length > 0"
                  size="tiny"
                  type="primary"
                  @click="handleApproveAll"
                >
                  一键全部批准并推送到云端
                </NButton>
              </NSpace>
            </template>

            <NAlert type="info" size="small" class="mb-4">
              普通用户上传的壁纸和图标申请推送到云端公共图库时，需要经过系统管理员审批。批准后系统将自动将图片推送到云端漫游图床。
            </NAlert>

            <NSpin :show="auditLoading">
              <div v-if="auditList.length === 0" class="py-12 text-center text-slate-400">
                暂无待审核的图片申请
              </div>

              <NGrid v-else cols="2 s:3 m:4" responsive="screen" :x-gap="12" :y-gap="12">
                <NGridItem v-for="item in auditList" :key="item.id">
                  <div
                    class="relative border rounded-lg p-2 bg-slate-50 dark:bg-zinc-800/60 hover:shadow-md transition"
                    :class="{ 'ring-2 ring-blue-500': selectedAuditIds.includes(item.id) }"
                  >
                    <div class="absolute top-2 left-2 z-10 bg-white/80 dark:bg-black/80 rounded p-0.5">
                      <NCheckbox
                        :checked="selectedAuditIds.includes(item.id)"
                        size="small"
                        @update:checked="() => toggleSelectAudit(item.id)"
                      />
                    </div>

                    <div class="w-full h-28 flex items-center justify-center bg-zinc-200/50 dark:bg-zinc-900 rounded overflow-hidden">
                      <NImage :src="item.src" class="w-full h-full object-cover" />
                    </div>

                    <div class="mt-2 truncate font-mono text-xs text-slate-700 dark:text-zinc-200" :title="item.fileName">
                      {{ item.fileName }}
                    </div>

                    <div class="flex items-center justify-between text-xs text-slate-400 mt-1">
                      <span>申请人: <strong>{{ item.userName || `UID:${item.userId}` }}</strong></span>
                      <NTag size="tiny" type="warning" :bordered="false">待审核</NTag>
                    </div>

                    <div class="flex items-center justify-end gap-2 mt-2 pt-1 border-t border-gray-100 dark:border-zinc-700">
                      <NButton size="tiny" type="success" secondary @click="handleAuditAction([item.id], 'approve')">
                        批准
                      </NButton>
                      <NButton size="tiny" type="error" secondary @click="handleAuditAction([item.id], 'reject')">
                        驳回
                      </NButton>
                    </div>
                  </div>
                </NGridItem>
              </NGrid>
            </NSpin>
          </NCard>
        </NTabPane>

        <!-- 第 3 页: 存储配置 (还原为最初设计) -->
        <NTabPane name="config" tab="⚙️ 存储配置">
          <NCard title="云存储引擎配置" size="small" class="mt-2 shadow-sm rounded-lg">
            <template #header-extra>
              <NTag v-if="repoCheckResult.tested" :type="repoCheckResult.isPrivate ? 'success' : 'warning'" round size="small">
                {{ repoCheckResult.isPrivate ? '私有安全仓库 (Private)' : '公开仓库 (Public - 需加密)' }}
              </NTag>
              <NTag v-else type="default" round size="small">
                未测试连通性
              </NTag>
            </template>

            <!-- 核心双写开关 -->
            <div class="mb-4 p-3 bg-slate-50 dark:bg-zinc-800 rounded-lg flex items-center justify-between border border-slate-200 dark:border-zinc-700">
              <div>
                <div class="font-medium text-slate-800 dark:text-zinc-100 flex items-center gap-2">
                  <SvgIcon icon="material-symbols:cloud-sync-outline" class="text-xl text-primary" />
                  <span>启用本地 + 云端双写机制 (默认关闭)</span>
                </div>
                <div class="text-xs text-slate-500 dark:text-zinc-400 mt-1">
                  开启后，上传图标或壁纸时系统将自动上传一份到云端备份存储；换新 NAS 或设备时可随时一键漫游恢复。
                </div>
              </div>
              <NSwitch v-model:value="form.enabled" />
            </div>

            <!-- 驱动选择 -->
            <NForm label-placement="left" label-width="130" size="small">
              <NFormItem label="云存储平台">
                <NRadioGroup v-model:value="form.type">
                  <NRadio value="github">
                    GitHub 仓库 (免费·支持 jsdelivr CDN)
                  </NRadio>
                  <NRadio value="r2">
                    Cloudflare R2 (S3 兼容·每月 10GB 免费)
                  </NRadio>
                </NRadioGroup>
              </NFormItem>

              <!-- GitHub 配置项 -->
              <div v-if="form.type === 'github'" class="space-y-3 bg-zinc-50 dark:bg-zinc-900/40 p-3 rounded-lg border border-slate-100 dark:border-zinc-800">
                <NAlert type="info" size="small" class="mb-2">
                  提示：系统会自动检测填写的 GitHub 仓库是私有 (Private) 还是公开 (Public)。公开仓库外链任何人可看，备份包会经强加密；私有仓库则只有持有 Token 方可访问。
                </NAlert>
                <NGrid cols="1 s:2" responsive="screen" :x-gap="12">
                  <NGridItem>
                    <NFormItem label="用户名/组织 (Owner)">
                      <NInput v-model:value="form.github.owner" placeholder="例如: mysecret2022" />
                    </NFormItem>
                  </NGridItem>
                  <NGridItem>
                    <NFormItem label="仓库名 (Repo)">
                      <NInput v-model:value="form.github.repo" placeholder="例如: firefly-assets" />
                    </NFormItem>
                  </NGridItem>
                  <NGridItem>
                    <NFormItem label="默认分支 (Branch)">
                      <NInput v-model:value="form.github.branch" placeholder="main" />
                    </NFormItem>
                  </NGridItem>
                  <NGridItem>
                    <NFormItem label="存放基准路径">
                      <NInput v-model:value="form.github.basePath" placeholder="panel-data" />
                    </NFormItem>
                  </NGridItem>
                </NGrid>
                <NFormItem label="Personal Token">
                  <NInput
                    v-model:value="form.github.token"
                    type="password"
                    show-password-on="click"
                    placeholder="ghp_xxx (需具备 repo 读写权限)"
                  />
                </NFormItem>
                <NFormItem label="jsdelivr CDN 加速">
                  <div class="flex items-center gap-3">
                    <NSwitch v-model:value="form.github.useCDN" />
                    <span class="text-xs text-slate-500">（仅在公开仓库下有效，大幅提升国内图片访问速度）</span>
                  </div>
                </NFormItem>
              </div>

              <!-- Cloudflare R2 配置项 -->
              <div v-if="form.type === 'r2'" class="space-y-3 bg-zinc-50 dark:bg-zinc-900/40 p-3 rounded-lg border border-slate-100 dark:border-zinc-800">
                <NAlert type="info" size="small" class="mb-2">
                  Cloudflare R2 兼容标准 S3 协议，提供每月 10GB 永久免费存储额度与 0 出网流量费。
                </NAlert>
                <NGrid cols="1 s:2" responsive="screen" :x-gap="12">
                  <NGridItem>
                    <NFormItem label="Account ID">
                      <NInput v-model:value="form.cloudflareR2.accountId" placeholder="Cloudflare 仪表盘中的账户 ID" />
                    </NFormItem>
                  </NGridItem>
                  <NGridItem>
                    <NFormItem label="存储桶名称 (Bucket)">
                      <NInput v-model:value="form.cloudflareR2.bucketName" placeholder="例如: firefly-bucket" />
                    </NFormItem>
                  </NGridItem>
                  <NGridItem>
                    <NFormItem label="Access Key ID">
                      <NInput v-model:value="form.cloudflareR2.accessKeyId" placeholder="R2 API 访问密钥 ID" />
                    </NFormItem>
                  </NGridItem>
                  <NGridItem>
                    <NFormItem label="Secret Key">
                      <NInput
                        v-model:value="form.cloudflareR2.secretAccessKey"
                        type="password"
                        show-password-on="click"
                        placeholder="R2 访问秘钥 Secret"
                      />
                    </NFormItem>
                  </NGridItem>
                </NGrid>
                <NFormItem label="公开访问域名">
                  <NInput v-model:value="form.cloudflareR2.publicDomain" placeholder="如已绑定公开域名: https://img.domain.com (可选)" />
                </NFormItem>
              </div>

              <!-- 按钮行 -->
              <div class="mt-4 flex justify-end gap-3">
                <NButton :loading="testLoading" secondary type="info" @click="handleTestConnection">
                  <template #icon>
                    <SvgIcon icon="material-symbols:network-check" />
                  </template>
                  测试连接与探测属性
                </NButton>
                <NButton :loading="saveLoading" type="primary" @click="handleSaveConfig">
                  <template #icon>
                    <SvgIcon icon="material-symbols:save-outline" />
                  </template>
                  保存设置
                </NButton>
              </div>
            </NForm>
          </NCard>
        </NTabPane>

        <!-- 第 4 页: 云端加密备份与恢复 -->
        <NTabPane name="backups" tab="🔒 云端加密备份中心">
          <NCard title="安全加密全站备份" size="small" class="mt-2 shadow-sm rounded-lg">
            <NAlert type="warning" size="small" class="mb-4">
              <strong>隐私安全保障机制：</strong>上传云端备份前<strong>必须手动输入加密密码</strong>。备份文件离开本地前均经过行业级 <strong>AES-256-GCM</strong> 算法强加密，即使存储在公开仓库也绝无数据泄漏风险。
            </NAlert>

            <!-- 密码输入与上传卡片 -->
            <div class="p-3 bg-zinc-50 dark:bg-zinc-800/50 rounded-lg border border-slate-200 dark:border-zinc-700 mb-6">
              <div class="text-sm font-medium mb-3 text-slate-800 dark:text-zinc-200">
                立即创建并上传全站加密备份
              </div>
              <NGrid cols="1 s:2" responsive="screen" :x-gap="12" :y-gap="8">
                <NGridItem>
                  <NInput
                    v-model:value="backupPassword"
                    type="password"
                    show-password-on="click"
                    placeholder="输入加密密码 (必填，请务必牢记)"
                  />
                </NGridItem>
                <NGridItem>
                  <NInput
                    v-model:value="backupPasswordConfirm"
                    type="password"
                    show-password-on="click"
                    placeholder="再次确认加密密码"
                  />
                </NGridItem>
              </NGrid>
              <div class="mt-3 flex justify-end">
                <NButton type="primary" :loading="backupUploading" @click="handleUploadBackup">
                  <template #icon>
                    <SvgIcon icon="material-symbols:lock" />
                  </template>
                  AES-256 加密并上传到云端
                </NButton>
              </div>
            </div>

            <!-- 云端备份列表 -->
            <div class="flex items-center justify-between mb-2">
              <span class="font-medium text-sm text-slate-700 dark:text-zinc-300">云端备份历史 (已加密)</span>
              <NButton size="small" secondary @click="fetchBackups">
                <template #icon>
                  <SvgIcon icon="material-symbols:refresh" />
                </template>
                刷新列表
              </NButton>
            </div>

            <NSpin :show="backupsLoading">
              <div v-if="cloudBackups.length === 0" class="py-12 text-center text-slate-400">
                <p>云端暂无加密备份文件</p>
              </div>

              <div v-else class="space-y-2">
                <div
                  v-for="b in cloudBackups"
                  :key="b.path"
                  class="flex items-center justify-between p-3 border rounded-lg bg-zinc-50 dark:bg-zinc-800/50 hover:border-blue-400 transition"
                >
                  <div class="flex items-center gap-3">
                    <SvgIcon icon="material-symbols:encrypted-rounded" class="text-2xl text-emerald-500" />
                    <div>
                      <div class="font-mono text-xs font-semibold text-slate-800 dark:text-zinc-200">
                        {{ b.name }}
                      </div>
                      <div class="text-xs text-slate-400 flex gap-3 mt-1">
                        <span>大小: {{ formatSize(b.size) }}</span>
                        <span>时间: {{ formatDate(b.updatedAt) }}</span>
                      </div>
                    </div>
                  </div>

                  <div class="flex items-center gap-2">
                    <NButton size="small" secondary @click="handleDownloadBackup(b)">
                      <template #icon>
                        <SvgIcon icon="material-symbols:download-rounded" />
                      </template>
                      下载
                    </NButton>
                    <NButton type="warning" size="small" @click="openRestoreModal(b)">
                      <template #icon>
                        <SvgIcon icon="material-symbols:settings-backup-restore-rounded" />
                      </template>
                      解密还原
                    </NButton>
                    <NButton type="error" size="small" secondary @click="handleDeleteSingleBackup(b)">
                      <template #icon>
                        <SvgIcon icon="material-symbols:delete-outline" />
                      </template>
                      删除
                    </NButton>
                  </div>
                </div>
              </div>
            </NSpin>
          </NCard>
        </NTabPane>
      </NTabs>
    </NSpin>

    <!-- 还原解密弹窗 -->
    <NModal
      v-model:show="restoreModalVisible"
      preset="card"
      title="解密并还原全站数据"
      style="max-width: 460px; border-radius: 12px;"
    >
      <div class="space-y-4">
        <NAlert type="error" size="small">
          还原操作将覆盖当前系统中的卡片数据与配置，请确保已备份当前重要数据。
        </NAlert>
        <div>
          <div class="text-xs text-slate-500 mb-1">
            选择的备份文件:
          </div>
          <div class="font-mono text-xs p-2 bg-slate-100 dark:bg-zinc-800 rounded truncate">
            {{ selectedBackupFilename }}
          </div>
        </div>
        <div>
          <div class="text-xs text-slate-500 mb-1">
            请输入解密密码:
          </div>
          <NInput
            v-model:value="restorePassword"
            type="password"
            show-password-on="click"
            placeholder="输入创建备份时设置的密码"
          />
        </div>
      </div>
      <template #footer>
        <div class="flex justify-end gap-2">
          <NButton secondary @click="restoreModalVisible = false">
            取消
          </NButton>
          <NButton type="warning" :loading="restoreSubmitting" @click="handleConfirmRestore">
            确认解密并还原
          </NButton>
        </div>
      </template>
    </NModal>

    <!-- 全屏大图预览浮层 (单击任意位置/大图秒退缩小) -->
    <div
      v-if="previewVisible && previewImage"
      class="fixed inset-0 z-[9999] flex items-center justify-center bg-black/85 backdrop-blur-md cursor-zoom-out select-none transition-all duration-200"
      @click="closePreview"
    >
      <div class="relative max-w-[90vw] max-h-[90vh] flex flex-col items-center" @click.stop="closePreview">
        <img
          :src="previewImage.downloadUrl"
          :alt="previewImage.name"
          class="max-w-[90vw] max-h-[82vh] object-contain rounded-xl shadow-2xl transition-transform duration-200 hover:scale-[1.01]"
        />
        <div class="mt-3 flex items-center gap-3 px-4 py-1.5 rounded-full bg-black/60 border border-white/20 text-white text-xs backdrop-blur-lg shadow-lg">
          <span class="font-medium truncate max-w-[200px]">{{ previewImage.name }}</span>
          <span class="text-white/40">|</span>
          <span>{{ formatSize(previewImage.size) }}</span>
          <span class="text-white/40">|</span>
          <span class="text-white/80">单击任意处缩小退出</span>
        </div>
      </div>
    </div>
  </div>
</template>
