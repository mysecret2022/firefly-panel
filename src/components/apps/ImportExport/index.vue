<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import type { UploadFileInfo } from 'naive-ui'
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NCheckboxGroup,
  NDivider,
  NInput,
  NSelect,
  NSpace,
  NTabPane,
  NTabs,
  NUpload,
  useDialog,
  useMessage,
} from 'naive-ui'
import { RoundCardModal, SvgIcon } from '@/components/common'
import type { IconGroup, ImportJsonResult } from '@/utils/jsonImportExport'
import { exportJson, importJsonString } from '@/utils/jsonImportExport'
import { get as getAbout } from '@/api/system/about'
import { edit as addGroup, getList as getGroupList } from '@/api/panel/itemIconGroup'
import { addMultiple as addMultipleIcons, getListByGroupId } from '@/api/panel/itemIcon'
import { exportBackup, restoreBackup } from '@/api/panel/backup'
import { listCloudBackups, restoreEncryptedBackup, testCloudConnection } from '@/api/panel/cloudStorage'
import type { CloudFileItem } from '@/api/panel/cloudStorage'
import { t } from '@/locales'
import { useAuthStore } from '@/store'

interface ItemGroup extends Panel.ItemIconGroup {
  items?: Panel.ItemInfo[]
}

const ms = useMessage()
const dialog = useDialog()
const authStore = useAuthStore()

// 默认激活 Tab：管理员优先进入全量备份，普通用户默认精准激活单账号卡片导入导出 (json-cards)，杜绝空白
const activeTab = ref(authStore.userInfo?.role === 1 ? 'full-backup' : 'json-cards')

watch(
  () => authStore.userInfo?.role,
  (role) => {
    if (role === 1) {
      if (!activeTab.value || activeTab.value === 'card-backup')
        activeTab.value = 'full-backup'
    }
    else {
      activeTab.value = 'json-cards'
    }
  },
  { immediate: true },
)
const importWarning = ref<string[]>([])
const importRoundModalShow = ref(false)
const exportRoundModalShow = ref(false)
const loading = ref(false)
const uploadLoading = ref(false)
const restoreLoading = ref(false)
const version = ref('1.0.2')

const importObj = ref<ImportJsonResult | null>(null)
const importItems = ref<string[]>(['icons'])
const checkedItems = ref<string[]>(['icons'])

// 全量备份导出
function handleExportFullBackup() {
  try {
    exportBackup()
    ms.success('整站全量备份已生成，正在下载...')
  }
  catch (error) {
    ms.error('导出失败，请重试')
  }
}

// 恢复全站备份文件选择
function handleRestoreFileChange(options: { file: UploadFileInfo; fileList: Array<UploadFileInfo> }) {
  const file = options.file.file
  if (!file)
    return

  dialog.warning({
    title: '确认恢复整站数据？',
    content: `即将从【${file.name}】还原数据。这将会覆盖现有数据表并还原图片附件，是否确认继续？`,
    positiveText: '确认恢复',
    negativeText: '取消',
    onPositiveClick: async () => {
      restoreLoading.value = true
      const formData = new FormData()
      formData.append('file', file)

      try {
        const res: any = await restoreBackup(formData)
        if (res.code === 0) {
          ms.success('数据恢复成功！页面即将自动刷新...')
          setTimeout(() => {
            window.location.reload()
          }, 1500)
        }
        else {
          ms.error(`恢复失败: ${res.msg || '未知错误'}`)
        }
      }
      catch (err: any) {
        ms.error(`恢复异常: ${err.message || '网络连接异常'}`)
      }
      finally {
        restoreLoading.value = false
      }
    },
  })
}

// 3. 从云端加密备份恢复 (云端漫游还原)
const cloudBackupList = ref<CloudFileItem[]>([])
const selectedCloudBackup = ref<string | null>(null)
const cloudDecryptPassword = ref('')
const cloudBackupLoading = ref(false)
const cloudRestoreLoading = ref(false)
const cloudTested = ref(false)
const cloudConnected = ref(false)
const cloudRestoreStepText = ref('')

function formatSize(bytes: number) {
  if (!bytes || bytes === 0)
    return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return `${(bytes / k ** i).toFixed(1)} ${sizes[i]}`
}

const cloudBackupOptions = computed(() => {
  if (cloudBackupList.value.length === 0) {
    return [
      {
        label: cloudConnected.value ? '未检测到有效云端快照' : '未检测到有效云端快照（请先刷新检测）',
        value: '',
        disabled: true,
      },
    ]
  }
  return cloudBackupList.value.map(item => ({
    label: `${item.name} (${formatSize(item.size)}${item.updatedAt ? ` · ${item.updatedAt}` : ''})`,
    value: item.name,
  }))
})

// 测试连通性并获取云端备份快照列表
async function fetchCloudBackups(showFeedback = false) {
  cloudBackupLoading.value = true
  try {
    const testRes: any = await testCloudConnection()
    cloudConnected.value = testRes.code === 0
    cloudTested.value = true

    if (testRes.code === 0) {
      const res: any = await listCloudBackups()
      if (res.code === 0 && Array.isArray(res.data)) {
        cloudBackupList.value = res.data
        if (res.data.length > 0) {
          // 默认选中第一个（最新快照）
          selectedCloudBackup.value = res.data[0].name
        }
        else {
          selectedCloudBackup.value = null
        }
        if (showFeedback)
          ms.success(`云端存储已连通，获取到 ${res.data.length} 个快照文件`)
      }
      else {
        cloudBackupList.value = []
        selectedCloudBackup.value = null
        if (showFeedback)
          ms.warning('云端已连接，但未检测到有效备份快照')
      }
    }
    else {
      cloudBackupList.value = []
      selectedCloudBackup.value = null
      if (showFeedback)
        ms.error(`云端连通失败: ${testRes.msg || '无法连接至配置的云存储'}`)
    }
  }
  catch (err: any) {
    cloudConnected.value = false
    cloudTested.value = true
    cloudBackupList.value = []
    selectedCloudBackup.value = null
    if (showFeedback)
      ms.error(`检测云端异常: ${err.message || '网络连接超时'}`)
  }
  finally {
    cloudBackupLoading.value = false
  }
}

// 执行云端加密还原
function handleCloudRestore() {
  if (!selectedCloudBackup.value) {
    ms.warning('请先选择需要还原的云端备份快照')
    return
  }
  if (!cloudDecryptPassword.value.trim()) {
    ms.warning('请输入 AES-256 解密密码')
    return
  }

  dialog.warning({
    title: '危险操作：确认还原云端加密数据？',
    content: '还原操作将解密并全量覆盖当前站点的所有本地数据与配置，过程不可逆，是否继续？',
    positiveText: '确认全量覆盖还原',
    negativeText: '取消',
    onPositiveClick: async () => {
      cloudRestoreLoading.value = true
      cloudRestoreStepText.value = '正在从云端拉取加密快照文件...'

      const stepTimer = setTimeout(() => {
        if (cloudRestoreLoading.value)
          cloudRestoreStepText.value = '正在进行 AES-256 校验与解密...'
      }, 1200)

      try {
        const res: any = await restoreEncryptedBackup(selectedCloudBackup.value!, cloudDecryptPassword.value.trim())
        clearTimeout(stepTimer)
        if (res.code === 0) {
          cloudRestoreStepText.value = '正在写入更新数据库并刷新生效...'
          ms.success(res.data?.message || '解密还原成功！数据已更新，页面即将自动刷新...')
          setTimeout(() => {
            window.location.reload()
          }, 1500)
        }
        else {
          cloudRestoreStepText.value = ''
          ms.error(res.msg || '解密失败：密码错误或备份文件损坏')
        }
      }
      catch (err: any) {
        clearTimeout(stepTimer)
        cloudRestoreStepText.value = ''
        ms.error(`还原异常: ${err.message || '网络连接超时或解密失败'}`)
      }
      finally {
        cloudRestoreLoading.value = false
      }
    },
  })
}

onMounted(() => {
  if (authStore.userInfo?.role === 1)
    fetchCloudBackups(false)
})

watch(activeTab, (tab) => {
  if (tab === 'full-backup' && authStore.userInfo?.role === 1 && !cloudTested.value)
    fetchCloudBackups(false)
})

// 导入图标 (JSON)
async function importIcons(): Promise<string | null> {
  const groups = importObj.value?.geticons()
  const batchSize = 50

  if (!groups)
    return null

  try {
    for (let i = 0; i < groups.length; i++) {
      const element = groups[i]
      const createGroupResponse = await addGroup<Panel.ItemIconGroup>({
        title: element.title,
        sort: element.sort,
      })

      if (createGroupResponse.code === 0) {
        const groupId = createGroupResponse.data?.id
        if (groupId) {
          let addIcons: Panel.ItemInfo[] = []
          for (let iconI = 0; iconI < element.children.length; iconI++) {
            const iconElement = element.children[iconI]
            addIcons.push({
              title: iconElement.title,
              sort: iconElement.sort,
              icon: iconElement.icon,
              url: iconElement.url,
              lanUrl: iconElement.lanUrl,
              description: iconElement.description,
              openMethod: iconElement.openMethod,
              itemIconGroupId: groupId,
            })

            if (addIcons.length === batchSize || iconI === element.children.length - 1) {
              const response = await addMultipleIcons(addIcons)
              if (response.code !== 0)
                return response.msg
              addIcons = []
            }
          }
        }
      }
      else {
        return createGroupResponse.msg
      }
    }
    return null
  }
  catch (error) {
    if (error instanceof Error)
      return `${t('common.failed')}: ${error.message}`
    else
      return t('common.unknownError')
  }
}

// 导出图标 (JSON)
async function exportIcons(): Promise<IconGroup[]> {
  const iconGroups: IconGroup[] = []
  const { code, data } = await getGroupList<Common.ListResponse<ItemGroup[]>>()

  if (code === 0) {
    await Promise.all(data.list.map(async (element) => {
      const group: IconGroup = {
        title: element.title as string,
        sort: element.sort as 0,
        children: [],
      }

      const res = await getListByGroupId<Common.ListResponse<Panel.ItemInfo[]>>(element.id)
      if (res.code === 0) {
        for (const iconElement of res.data.list) {
          group.children.push({
            icon: iconElement.icon,
            sort: iconElement.sort || 99999,
            title: iconElement.title,
            url: iconElement.url,
            lanUrl: iconElement.lanUrl || '',
            description: iconElement.description || '',
            openMethod: iconElement.openMethod || 1,
          })
        }
      }
      iconGroups.push(group)
    }))
    return iconGroups
  }
  return []
}

onMounted(() => {
  interface Version {
    versionName: string
    versionCode: number
  }

  getAbout<Version>().then((res) => {
    if (res.code === 0)
      version.value = res.data.versionName
  })
})

function handleFileChange(options: { file: UploadFileInfo; fileList: Array<UploadFileInfo> }) {
  uploadLoading.value = true
  if (options.file.file) {
    const reader = new FileReader()
    reader.onload = (e) => {
      if (e.target?.result) {
        try {
          const res = importJsonString(e.target.result as string)
          importObj.value = res
          importItems.value = res.getImportItemName()
          checkedItems.value = res.getImportItemName()
          importWarning.value = res.getWarnings()
          importRoundModalShow.value = true
        }
        catch (err: any) {
          ms.error(err.message || '文件解析失败')
        }
        finally {
          uploadLoading.value = false
        }
      }
    }
    reader.readAsText(options.file.file)
  }
}

async function handleStartExport() {
  loading.value = true
  const exportResult = exportJson(version.value)
  if (checkedItems.value.includes('icons')) {
    const iconGroups = await exportIcons()
    exportResult.addIconsData(iconGroups)
  }
  exportResult.exportFile()
  loading.value = false
  exportRoundModalShow.value = false
}

async function handleStartImport() {
  loading.value = true
  if (checkedItems.value.includes('icons')) {
    const errMsg = await importIcons()
    if (errMsg !== null)
      ms.error(`${t('common.failed')}:${errMsg}`)
    else
      ms.success(`${t('common.success')}, ${t('common.refreshPage')}`)
  }
  loading.value = false
  importRoundModalShow.value = false
}
</script>

<template>
  <div class="h-full w-full overflow-y-auto px-4 py-2 box-border">
    <NTabs v-model:value="activeTab" type="line" animated>
      <!-- 全量整站备份与数据迁移 (仅管理员可见) -->
      <NTabPane v-if="authStore.userInfo?.role === 1" name="full-backup" tab="📦 整站数据备份与恢复 (ZIP)">
        <div class="space-y-4 pt-2 pb-6">
          <NAlert type="info" :bordered="false" class="text-xs">
            <template #header>
              数据无缝迁移说明
            </template>
            全量打包包含：<strong>所有账号与权限、导航分组、卡片图标、样式设置</strong>以及<strong>自定义上传的图标和壁纸图片</strong>。在新设备或新容器部署后，上传备份包即可 100% 完整还原，无需重新配置。
          </NAlert>

          <!-- 导出备份卡片 (纵向全宽自适应，显示完整) -->
          <NCard title="1. 导出整站全量备份" size="small" :bordered="true" class="rounded-xl shadow-xs">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="text-gray-500 text-xs leading-relaxed">
                实时将当前数据库与上传的附件打包为一个 ZIP 压缩文件并保存到本地。
              </div>
              <div class="shrink-0">
                <NButton type="primary" size="medium" @click="handleExportFullBackup">
                  <template #icon>
                    <SvgIcon icon="fa6:solid-file-export" />
                  </template>
                  导出备份 (.zip)
                </NButton>
              </div>
            </div>
          </NCard>

          <!-- 恢复迁移卡片 (纵向全宽自适应，显示完整) -->
          <NCard title="2. 恢复整站数据 (迁移还原)" size="small" :bordered="true" class="rounded-xl shadow-xs">
            <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-3">
              <div class="text-gray-500 text-xs leading-relaxed">
                上传之前导出的 .zip 备份包（或 .json 文件），一键解压并覆盖还原所有数据。
              </div>
              <div class="shrink-0">
                <NUpload
                  accept=".zip,.json"
                  :default-upload="false"
                  :show-file-list="false"
                  @change="handleRestoreFileChange"
                >
                  <NButton type="warning" size="medium" :loading="restoreLoading">
                    <template #icon>
                      <SvgIcon icon="fa6:solid-file-import" />
                    </template>
                    上传并还原数据 (.zip)
                  </NButton>
                </NUpload>
              </div>
            </div>
          </NCard>

          <!-- 3. 从云端加密备份恢复 (云端漫游还原) -->
          <NCard title="3. 恢复云端加密数据 (云端漫游还原)" size="small" :bordered="true" class="rounded-xl shadow-xs">
            <div class="space-y-3">
              <div class="text-gray-500 text-xs leading-relaxed">
                从云端存储同步加密备份快照。此操作需使用“云存储与漫游”模块配置的 AES-256 密钥进行本地解密并覆盖还原。
              </div>

              <!-- 表单排版: 快照选择 + 密码输入 + 状态展示 (紧凑且规整) -->
              <div class="p-3 bg-slate-50/80 dark:bg-zinc-800/60 rounded-lg border border-slate-200/70 dark:border-zinc-700/60 space-y-3">
                <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
                  <!-- 快照版本选择 -->
                  <div>
                    <div class="text-xs font-medium text-slate-700 dark:text-zinc-200 mb-1 flex items-center justify-between">
                      <span>云端备份快照 (Snapshot)</span>
                      <NButton text type="primary" size="tiny" :loading="cloudBackupLoading" @click="() => fetchCloudBackups(true)">
                        <template #icon>
                          <SvgIcon icon="material-symbols:refresh" />
                        </template>
                        测试连通性 / 刷新快照列表
                      </NButton>
                    </div>
                    <NSelect
                      v-model:value="selectedCloudBackup"
                      size="small"
                      :options="cloudBackupOptions"
                      :loading="cloudBackupLoading"
                      :disabled="cloudRestoreLoading || cloudBackupList.length === 0"
                      placeholder="请选择云端备份快照"
                    />
                  </div>

                  <!-- AES-256 解密密码输入框 -->
                  <div>
                    <div class="text-xs font-medium text-slate-700 dark:text-zinc-200 mb-1 flex items-center justify-between">
                      <span>AES-256 解密密码</span>
                      <span class="text-[11px] text-slate-400">密钥用于内存解密验证</span>
                    </div>
                    <NInput
                      v-model:value="cloudDecryptPassword"
                      type="password"
                      size="small"
                      show-password-on="click"
                      :disabled="cloudRestoreLoading"
                      placeholder="请输入云端备份对应的加密密码"
                      autocomplete="new-password"
                    />
                  </div>
                </div>

                <!-- 底部操作与进度反馈 -->
                <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-2 pt-2 border-t border-slate-200/60 dark:border-zinc-700/50">
                  <div class="text-xs text-slate-500 dark:text-zinc-400 flex items-center gap-1.5">
                    <span v-if="cloudRestoreStepText" class="text-amber-500 flex items-center gap-1 font-medium">
                      <span class="w-1.5 h-1.5 rounded-full bg-amber-500 animate-pulse"></span>
                      {{ cloudRestoreStepText }}
                    </span>
                    <span v-else-if="cloudBackupList.length > 0" class="text-emerald-600 dark:text-emerald-400 flex items-center gap-1">
                      <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
                      已检测到 {{ cloudBackupList.length }} 个有效云端备份快照
                    </span>
                    <span v-else-if="cloudTested">
                      未检测到有效云端快照（请在“云存储与漫游”中生成备份或刷新连通性）
                    </span>
                    <span v-else>
                      点击右上角可测试连通性并拉取最新云端快照
                    </span>
                  </div>

                  <div class="flex items-center gap-2 shrink-0">
                    <NButton
                      type="warning"
                      size="medium"
                      :loading="cloudRestoreLoading"
                      :disabled="!selectedCloudBackup || cloudBackupList.length === 0"
                      @click="handleCloudRestore"
                    >
                      <template #icon>
                        <SvgIcon icon="material-symbols:cloud-download" />
                      </template>
                      拉取并解密还原
                    </NButton>
                  </div>
                </div>
              </div>
            </div>
          </NCard>
        </div>
      </NTabPane>

      <!-- 单用户卡片导入导出 (JSON) -->
      <NTabPane name="json-cards" tab="🔖 单账号卡片导入导出 (JSON)">
        <div class="space-y-4 pt-2 pb-6">
          <NAlert type="info" :bordered="false" class="text-xs">
            {{ $t('apps.exportImport.tip') }}
          </NAlert>

          <NCard title="导航卡片书签管理" size="small" :bordered="true" class="rounded-xl shadow-xs">
            <div class="text-gray-500 text-xs mb-4">
              导出当前账号下的所有图标与分组为 JSON 格式，或导入其他用户的卡片配置。
            </div>
            <div class="flex flex-wrap gap-4 items-center">
              <NUpload
                accept=".sun-panel.json,.sunpanel.json,.json"
                directory-dnd
                :default-upload="false"
                :show-file-list="false"
                @change="handleFileChange"
              >
                <NButton type="info" size="medium" :loading="uploadLoading">
                  <template #icon>
                    <SvgIcon icon="fa6:solid-file-import" />
                  </template>
                  {{ $t('apps.exportImport.import') }} JSON
                </NButton>
              </NUpload>

              <NButton type="info" size="medium" @click="exportRoundModalShow = !exportRoundModalShow">
                <template #icon>
                  <SvgIcon icon="fa6:solid-file-export" />
                </template>
                {{ $t('apps.exportImport.export') }} JSON
              </NButton>
            </div>
          </NCard>
        </div>
      </NTabPane>
    </NTabs>

    <!-- 弹窗部分 -->
    <RoundCardModal v-model:show="importRoundModalShow" style="max-width: 420px;" :title="$t('apps.exportImport.import')">
      <div v-if="importWarning.length > 0" class="mb-3">
        <NAlert :title="$t('common.warning')" type="warning">
          <div v-for="(text, index) in importWarning" :key="index">
            {{ text }}
          </div>
        </NAlert>
      </div>
      <NDivider title-placement="left">
        {{ $t('apps.exportImport.selectImportData') }}
      </NDivider>

      <NSpace justify="center" style="margin-top: 15px;">
        <NCheckboxGroup v-model:value="checkedItems">
          <NCheckbox v-if="importItems.includes('icons')" value="icons" :label="$t('apps.exportImport.moduleIcon')" />
          <NCheckbox v-if="importItems.includes('style')" value="style" :label="$t('apps.exportImport.moduleStyle')" />
        </NCheckboxGroup>
      </NSpace>
      <NSpace justify="center">
        <div class="mt-8">
          <NButton type="success" :disabled="checkedItems.length === 0" :loading="loading" @click="handleStartImport">
            {{ $t('common.continue') }}
          </NButton>
        </div>
      </NSpace>
    </RoundCardModal>

    <RoundCardModal v-model:show="exportRoundModalShow" style="max-width: 420px;" :title="$t('apps.exportImport.export')">
      <NDivider title-placement="left">
        {{ $t('apps.exportImport.selectExportData') }}
      </NDivider>

      <NSpace justify="center" style="margin-top: 15px;">
        <NCheckboxGroup v-model:value="checkedItems">
          <NCheckbox v-if="importItems.includes('icons')" value="icons" :label="$t('apps.exportImport.moduleIcon')" />
          <NCheckbox v-if="importItems.includes('style')" value="style" :label="$t('apps.exportImport.moduleStyle')" />
        </NCheckboxGroup>
      </NSpace>
      <NSpace justify="center">
        <div class="mt-8">
          <NButton type="success" :disabled="checkedItems.length === 0" :loading="loading" @click="handleStartExport">
            {{ $t('common.continue') }}
          </NButton>
        </div>
      </NSpace>
    </RoundCardModal>
  </div>
</template>
