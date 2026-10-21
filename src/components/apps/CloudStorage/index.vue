<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import {
  NAlert,
  NButton,
  NCard,
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

const ms = useMessage()
const dialog = useDialog()

// 状态管理
const loading = ref(false)
const testLoading = ref(false)
const saveLoading = ref(false)
const backupUploading = ref(false)
const syncAllLoading = ref(false)
const activeTab = ref('config')

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
const backupsLoading = ref(false)
const imagesLoading = ref(false)

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
  return `${(bytes / Math.pow(k, i)).toFixed(2)} ${sizes[i]}`
}

// 格式化日期
function formatDate(dateStr: string) {
  if (!dateStr)
    return '-'
  try {
    const d = new Date(dateStr)
    return d.toLocaleString()
  }
  catch {
    return dateStr
  }
}

// 加载配置
async function loadConfig() {
  loading.value = true
  try {
    const res = await getCloudConfig()
    if (res.data) {
      form.value = {
        ...form.value,
        ...res.data,
      }
    }
  }
  catch (error: any) {
    ms.error(error.message || '加载云存储配置失败')
  }
  finally {
    loading.value = false
  }
}

// 测试连接与公私检测
async function handleTestConnection() {
  testLoading.value = true
  try {
    const res = await testCloudConnection(form.value)
    if (res.data) {
      repoCheckResult.value = {
        tested: true,
        isPrivate: res.data.isPrivate,
        desc: res.data.description,
      }
      ms.success(res.data.message || '云存储连接测试成功')
    }
  }
  catch (error: any) {
    repoCheckResult.value.tested = false
    ms.error(`连接测试失败: ${error.message || '请检查配置'}`)
  }
  finally {
    testLoading.value = false
  }
}

// 保存配置
async function handleSaveConfig() {
  saveLoading.value = true
  try {
    await saveCloudConfig(form.value)
    ms.success('云存储配置已安全保存！')
  }
  catch (error: any) {
    ms.error(`保存失败: ${error.message || '未知错误'}`)
  }
  finally {
    saveLoading.value = false
  }
}

// 获取云端备份列表
async function fetchBackups() {
  backupsLoading.value = true
  try {
    const res = await listCloudBackups()
    cloudBackups.value = res.data || []
  }
  catch (error: any) {
    ms.error(`获取备份列表失败: ${error.message || '网络错误'}`)
  }
  finally {
    backupsLoading.value = false
  }
}

// 获取云端图床列表
async function fetchImages() {
  imagesLoading.value = true
  try {
    const res = await listCloudImages()
    cloudImages.value = res.data || []
  }
  catch (error: any) {
    ms.error(`获取图片列表失败: ${error.message || '网络错误'}`)
  }
  finally {
    imagesLoading.value = false
  }
}

// 上传加密备份
async function handleUploadBackup() {
  if (!backupPassword.value) {
    ms.warning('请输入加密密码，确保备份文件隐私安全')
    return
  }
  if (backupPassword.value !== backupPasswordConfirm.value) {
    ms.warning('两次输入的加密密码不一致，请仔细核对')
    return
  }

  backupUploading.value = true
  try {
    const res = await uploadEncryptedBackup(backupPassword.value)
    ms.success(res.data?.message || '全站 AES-256 加密备份已成功上传至云端！')
    backupPassword.value = ''
    backupPasswordConfirm.value = ''
    fetchBackups()
  }
  catch (error: any) {
    ms.error(`上传失败: ${error.message || '错误'}`)
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

// 提交解密还原
async function submitRestoreBackup() {
  if (!restorePassword.value) {
    ms.warning('请输入该备份的解密密码')
    return
  }

  restoreSubmitting.value = true
  try {
    const res = await restoreEncryptedBackup(selectedBackupFilename.value, restorePassword.value)
    ms.success(res.data?.message || '全站数据与文件已成功解密还原！')
    restoreModalVisible.value = false
    dialog.info({
      title: '还原成功',
      content: '备份数据已成功恢复到本地数据库与文件目录，建议刷新页面以查看最新卡片与配置。',
      positiveText: '立即刷新',
      onPositiveClick: () => {
        window.location.reload()
      },
    })
  }
  catch (error: any) {
    ms.error(`还原失败: ${error.message || '解密失败，密码不正确'}`)
  }
  finally {
    restoreSubmitting.value = false
  }
}

// 单张下载回本地
async function handleSyncSingleImage(item: CloudFileItem) {
  try {
    const res = await syncImageToLocal(item.name)
    ms.success(res.data?.message || `已下载 ${item.name} 到本地`)
  }
  catch (error: any) {
    ms.error(`下载失败: ${error.message || '错误'}`)
  }
}

// 一键全部下载回本地
async function handleSyncAllImages() {
  syncAllLoading.value = true
  try {
    const res = await syncAllImagesToLocal()
    ms.success(res.data?.message || '云端所有图片已成功下载同步到本地！')
  }
  catch (error: any) {
    ms.error(`全部同步失败: ${error.message || '错误'}`)
  }
  finally {
    syncAllLoading.value = false
  }
}

// 切换标签页事件
function handleTabChange(tab: string) {
  activeTab.value = tab
  if (tab === 'images')
    fetchImages()
  else if (tab === 'backups')
    fetchBackups()
}

onMounted(() => {
  loadConfig()
})
</script>

<template>
  <div class="p-3 max-w-4xl mx-auto space-y-4">
    <NSpin :show="loading">
      <NTabs v-model:value="activeTab" type="segment" animated @update:value="handleTabChange">
        <!-- TAB 1: 云存储配置 -->
        <NTabPane name="config" tab="云存储配置与双写">
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

        <!-- TAB 2: 云端漫游图床 -->
        <NTabPane name="images" tab="云端漫游图床">
          <NCard title="云端图库与多端漫游" size="small" class="mt-2 shadow-sm rounded-lg">
            <template #header-extra>
              <NSpace>
                <NButton size="small" secondary @click="fetchImages">
                  <template #icon>
                    <SvgIcon icon="material-symbols:refresh" />
                  </template>
                  刷新
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
                  确认将云端存储中的全部图片一次性下载并同步到当前机器的本地存储中吗？
                </NPopconfirm>
              </NSpace>
            </template>

            <NAlert type="info" size="small" class="mb-4">
              在更换新设备或全新部署 NAS 容器时，绑定该云存储后即可通过上方“一键全部拉取到本地”无缝同步所有图片，或在下方列表中单个选择下载。
            </NAlert>

            <NSpin :show="imagesLoading">
              <div v-if="cloudImages.length === 0" class="py-12 text-center text-slate-400">
                <SvgIcon icon="material-symbols:image-not-supported-outline" class="text-4xl mx-auto mb-2 text-slate-300" />
                <p>云端暂无图片，启用双写后新上传的图标/壁纸将自动同步至此</p>
              </div>

              <NGrid v-else cols="2 s:3 m:4" responsive="screen" :x-gap="12" :y-gap="12">
                <NGridItem v-for="img in cloudImages" :key="img.path">
                  <div class="border rounded-lg p-2 bg-slate-50 dark:bg-zinc-800/60 hover:shadow-md transition">
                    <div class="w-full h-28 flex items-center justify-center bg-zinc-200/50 dark:bg-zinc-900 rounded overflow-hidden">
                      <NImage
                        :src="img.downloadUrl"
                        class="w-full h-full object-cover"
                        preview-disabled
                        fallback-src="data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='40' height='40' viewBox='0 0 24 24'><text x='50%' y='50%' dominant-baseline='middle' text-anchor='middle' fill='%23aaa'>IMG</text></svg>"
                      />
                    </div>
                    <div class="mt-2 truncate font-mono text-xs text-slate-700 dark:text-zinc-200" :title="img.name">
                      {{ img.name }}
                    </div>
                    <div class="flex items-center justify-between text-xs text-slate-400 mt-1">
                      <span>{{ formatSize(img.size) }}</span>
                      <NButton text type="primary" size="tiny" @click="handleSyncSingleImage(img)">
                        下载到本地
                      </NButton>
                    </div>
                  </div>
                </NGridItem>
              </NGrid>
            </NSpin>
          </NCard>
        </NTabPane>

        <!-- TAB 3: 云端加密备份与恢复 -->
        <NTabPane name="backups" tab="云端加密备份中心">
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
              <div v-if="cloudBackups.length === 0" class="py-8 text-center text-slate-400">
                暂无云端备份文件
              </div>

              <div v-else class="space-y-2">
                <div
                  v-for="b in cloudBackups"
                  :key="b.path"
                  class="p-3 rounded-lg border border-slate-200 dark:border-zinc-700 bg-white dark:bg-zinc-800/80 flex items-center justify-between"
                >
                  <div class="space-y-1">
                    <div class="flex items-center gap-2">
                      <SvgIcon icon="material-symbols:shield-lock-outline" class="text-emerald-500 text-lg" />
                      <span class="font-mono text-sm font-medium text-slate-800 dark:text-zinc-100">{{ b.name }}</span>
                      <NTag size="tiny" type="success" round>
                        AES-256 加密
                      </NTag>
                    </div>
                    <div class="text-xs text-slate-400">
                      大小: {{ formatSize(b.size) }} | 云端时间: {{ formatDate(b.updatedAt) }}
                    </div>
                  </div>
                  <NButton size="small" type="warning" secondary @click="openRestoreModal(b)">
                    <template #icon>
                      <SvgIcon icon="material-symbols:settings-backup-restore" />
                    </template>
                    密码解密还原
                  </NButton>
                </div>
              </div>
            </NSpin>
          </NCard>
        </NTabPane>
      </NTabs>
    </NSpin>

    <!-- 解密还原密码输入弹窗 -->
    <NModal
      v-model:show="restoreModalVisible"
      preset="card"
      title="输入解密密码以还原全站数据"
      style="max-width: 480px"
      :mask-closable="false"
    >
      <div class="space-y-4">
        <NAlert type="warning" size="small">
          解密还原将覆盖当前数据库的卡片和分组配置，恢复到该备份创建时的状态。请输入创建备份时设置的加密密码：
        </NAlert>
        <div class="font-mono text-xs text-slate-500 break-all">
          目标备份: {{ selectedBackupFilename }}
        </div>
        <NInput
          v-model:value="restorePassword"
          type="password"
          show-password-on="click"
          placeholder="请输入解密密码"
          @keyup.enter="submitRestoreBackup"
        />
        <div class="flex justify-end gap-3 pt-2">
          <NButton @click="restoreModalVisible = false">
            取消
          </NButton>
          <NButton type="primary" :loading="restoreSubmitting" @click="submitRestoreBackup">
            校验并还原
          </NButton>
        </div>
      </div>
    </NModal>
  </div>
</template>
