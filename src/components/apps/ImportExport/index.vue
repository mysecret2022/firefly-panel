<script setup lang="ts">
import { onMounted, ref } from 'vue'
import type { UploadFileInfo } from 'naive-ui'
import {
  NAlert,
  NButton,
  NCard,
  NCheckbox,
  NCheckboxGroup,
  NDivider,
  NInput,
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

interface ItemGroup extends Panel.ItemIconGroup {
  items?: Panel.ItemInfo[]
}

const ms = useMessage()
const dialog = useDialog()

const jsonData = ref<string | null>(null)
const importWarning = ref<string[]>([])
const importRoundModalShow = ref(false)
const exportRoundModalShow = ref(false)
const loading = ref(false)
const uploadLoading = ref(false)
const restoreLoading = ref(false)
const version = ref('') // 当前软件版本
const debug = ref(false)

const importObj = ref<ImportJsonResult | null>(null)
const importItems = ref<string[]>(['icons'])
const checkedItems = ref<string[]>(['icons'])

// 全量备份导出
function handleExportFullBackup() {
  try {
    exportBackup()
    ms.success('全量备份已开始下载，请妥善保存备份文件！')
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
    title: '确认恢复数据？',
    content: `即将从【${file.name}】恢复数据，该操作将覆盖现有的分组、图标与系统配置，并解压还原资源附件。是否确认继续？`,
    positiveText: '确认恢复',
    negativeText: '取消',
    onPositiveClick: async () => {
      restoreLoading.value = true
      const formData = new FormData()
      formData.append('file', file)

      try {
        const res: any = await restoreBackup(formData)
        if (res.code === 0) {
          ms.success('数据恢复成功！正在重新加载页面...')
          setTimeout(() => {
            window.location.reload()
          }, 1500)
        }
        else {
          ms.error(`恢复失败: ${res.msg || '未知错误'}`)
        }
      }
      catch (err: any) {
        ms.error(`恢复异常: ${err.message || '网络或服务错误'}`)
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
  jsonData.value = exportResult.string()
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
  <div class="p-3">
    <NTabs type="line" animated>
      <!-- 全量整站备份与迁移 -->
      <NTabPane name="full-backup" tab="📦 全站备份与数据迁移 (ZIP)">
        <div class="space-y-4 pt-2">
          <NAlert type="info" :bordered="false">
            <template #header>
              全量备份与无缝迁移说明
            </template>
            打包内容包括：<strong>全部用户与权限、导航分组、卡片图标、系统设置、个性化配置</strong>以及 <strong>自定义上传的图标壁纸文件</strong>。
            在新服务器/NAS 部署后，直接上传备份包即可 100% 完整复原，无需重新配置。
          </NAlert>

          <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
            <NCard title="导出全站备份" size="small" class="shadow-sm">
              <p class="text-gray-500 text-sm mb-4">
                实时将整站数据与静态附件打包成 .zip 文件下载到本地。
              </p>
              <NButton type="primary" size="large" block @click="handleExportFullBackup">
                <template #icon>
                  <SvgIcon icon="fa6:solid-file-export" />
                </template>
                一键生成并下载整站备份 (.zip)
              </NButton>
            </NCard>

            <NCard title="恢复 / 迁移数据" size="small" class="shadow-sm">
              <p class="text-gray-500 text-sm mb-4">
                上传历史备份的 .zip 或 .json 文件，一键恢复还原所有数据。
              </p>
              <NUpload
                accept=".zip,.json"
                :default-upload="false"
                :show-file-list="false"
                @change="handleRestoreFileChange"
              >
                <NButton type="warning" size="large" block :loading="restoreLoading">
                  <template #icon>
                    <SvgIcon icon="fa6:solid-file-import" />
                  </template>
                  上传备份文件并一键恢复 (.zip / .json)
                </NButton>
              </NUpload>
            </NCard>
          </div>
        </div>
      </NTabPane>

      <!-- 单用户卡片导入导出 (JSON) -->
      <NTabPane name="json-cards" tab="🔖 单用户卡片导入导出 (JSON)">
        <div class="pt-2">
          <NAlert type="info" :bordered="false">
            <p>{{ $t('apps.exportImport.tip') }}</p>
          </NAlert>
          <div class="flex justify-center m-[40px]">
            <div class="m-[10px]">
              <NUpload
                accept=".sun-panel.json,.sunpanel.json"
                directory-dnd
                :default-upload="false"
                :show-file-list="false"
                @change="handleFileChange"
              >
                <NButton type="info" size="large" :loading="uploadLoading">
                  <template #icon>
                    <SvgIcon icon="fa6:solid-file-import" />
                  </template>
                  {{ $t('apps.exportImport.import') }}
                </NButton>
              </NUpload>
            </div>
            <div class="m-[10px]">
              <NButton type="info" size="large" @click="exportRoundModalShow = !exportRoundModalShow">
                <template #icon>
                  <SvgIcon icon="fa6:solid-file-export" />
                </template>
                {{ $t('apps.exportImport.export') }}
              </NButton>
            </div>
          </div>
        </div>
      </NTabPane>
    </NTabs>

    <!-- 弹窗部分 -->
    <RoundCardModal v-model:show="importRoundModalShow" style="max-width: 400px;" :title="$t('apps.exportImport.import')">
      <div v-if="importWarning.length > 0">
        <NAlert :title="$t('common.warning')" type="warning">
          <div v-for="(text, index) in importWarning" :key="index">
            {{ text }}
          </div>
        </NAlert>
      </div>
      <NDivider title-placement="left">
        {{ $t('apps.exportImport.selectImportData') }}
      </NDivider>

      <NSpace justify="center" style="margin-top: 20px;">
        <NCheckboxGroup v-model:value="checkedItems">
          <NCheckbox v-if="importItems.includes('icons')" value="icons" :label="$t('apps.exportImport.moduleIcon')" />
          <NCheckbox v-if="importItems.includes('style')" value="style" :label="$t('apps.exportImport.moduleStyle')" />
        </NCheckboxGroup>
      </NSpace>
      <NSpace justify="center">
        <div class="mt-[40px]">
          <NButton type="success" :disabled="checkedItems.length === 0" :loading="loading" @click="handleStartImport">
            {{ $t('common.continue') }}
          </NButton>
        </div>
      </NSpace>
    </RoundCardModal>

    <RoundCardModal v-model:show="exportRoundModalShow" style="max-width: 400px;" :title="$t('apps.exportImport.export')">
      <NDivider title-placement="left">
        {{ $t('apps.exportImport.selectExportData') }}
      </NDivider>

      <NSpace justify="center" style="margin-top: 20px;">
        <NCheckboxGroup v-model:value="checkedItems">
          <NCheckbox v-if="importItems.includes('icons')" value="icons" :label="$t('apps.exportImport.moduleIcon')" />
          <NCheckbox v-if="importItems.includes('style')" value="style" :label="$t('apps.exportImport.moduleStyle')" />
        </NCheckboxGroup>
      </NSpace>
      <NSpace justify="center">
        <div class="mt-[40px]">
          <NButton type="success" :disabled="checkedItems.length === 0" :loading="loading" @click="handleStartExport">
            {{ $t('common.continue') }}
          </NButton>
        </div>
      </NSpace>
    </RoundCardModal>
  </div>
</template>
