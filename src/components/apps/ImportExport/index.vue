<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import type { UploadFileInfo } from 'naive-ui'
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NCheckboxGroup,
  NDivider,
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
