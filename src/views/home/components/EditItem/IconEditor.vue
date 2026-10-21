<script setup lang="ts">
import { computed, defineProps, ref } from 'vue'
import {
  NButton,
  NColorPicker,
  NGrid,
  NGridItem,
  NImage,
  NInput,
  NModal,
  NRadio,
  NSpin,
  NUpload,
  useMessage,
} from 'naive-ui'
import type { UploadFileInfo } from 'naive-ui'
import { ItemIcon, SvgIcon } from '@/components/common'
import { useAuthStore } from '@/store'
import { apiRespErrMsg } from '@/utils/request/apiMessage'
import { getPublicIcons } from '@/api/system/file'

const props = defineProps<{
  itemIcon: Panel.ItemIcon | null
}>()
const emit = defineEmits<{
  (e: 'update:itemIcon', visible: Panel.ItemIcon): void // 定义修改父组件（prop内）的值的事件
}>()
const authStore = useAuthStore()
const ms = useMessage()

// 默认图标背景色
const defautSwatchesBackground = [
  '#00000000',
  '#000000',
  '#ffffff',
  '#18A058',
  '#2080F0',
  '#F0A020',
  'rgba(208, 48, 80, 1)',
  '#C418D1FF',
]

const initData: Panel.ItemIcon = {
  itemType: 2,
  backgroundColor: '#2a2a2a6b',
}

const itemIconInfo = computed({
  get() {
    const v = {
      ...initData,
      ...props.itemIcon,
      backgroundColor: props.itemIcon?.backgroundColor || initData.backgroundColor,
    }
    return v
  },
  set() {
    handleChange()
  },
})

// 图库选择器弹窗状态
const galleryModalVisible = ref(false)
const galleryLoading = ref(false)
const publicIcons = ref<any[]>([])

async function openGalleryPicker() {
  galleryModalVisible.value = true
  galleryLoading.value = true
  try {
    const res: any = await getPublicIcons()
    if (res.code === 0 && res.data) {
      publicIcons.value = res.data.list || []
    }
  }
  catch (e) {
    ms.error('获取公共图库失败')
  }
  finally {
    galleryLoading.value = false
  }
}

function handleSelectIconFromGallery(item: any) {
  itemIconInfo.value.itemType = 2 // 设为图片类型
  itemIconInfo.value.src = item.src
  handleChange()
  ms.success(`已选用图标: ${item.fileName}`)
  galleryModalVisible.value = false
}

function handleIconTypeRadioChange(type: number) {
  itemIconInfo.value.itemType = type
  handleChange()
}

function handleChange() {
  emit('update:itemIcon', itemIconInfo.value || null)
}

function handleResetBackgroundColor() {
  itemIconInfo.value.backgroundColor = initData.backgroundColor
  handleChange()
}

const handleUploadFinish = ({
  file,
  event,
}: {
  file: UploadFileInfo
  event?: ProgressEvent
}) => {
  const res = JSON.parse((event?.target as XMLHttpRequest).response)
  if (res.code === 0) {
    const imageUrl = res.data.imageUrl
    itemIconInfo.value.src = imageUrl
    emit('update:itemIcon', itemIconInfo.value || null)
  }
  else {
    apiRespErrMsg(res)
  }

  return file
}
</script>

<template>
  <div>
    <div class="mb-[10px]">
      <NRadio
        :checked="itemIconInfo.itemType === 1"
        :value="1"
        name="iconType"
        @change="handleIconTypeRadioChange(1)"
      >
        {{ $t('common.text') }}
      </NRadio>

      <NRadio
        :checked="itemIconInfo.itemType === 2"
        :value="2"
        name="iconType"
        @change="handleIconTypeRadioChange(2)"
      >
        {{ $t('common.image') }}
      </NRadio>

      <NRadio
        :checked="itemIconInfo.itemType === 3"
        :value="3"
        name="iconType"
        @change="handleIconTypeRadioChange(3)"
      >
        {{ $t('iconItem.onlineIcon') }}
      </NRadio>
    </div>

    <div class="h-[100px]">
      <div class="flex">
        <div>
          <div class="border rounded-2xl bg-slate-200 overflow-hidden transparent-grid">
            <ItemIcon :item-icon="itemIconInfo" />
          </div>
        </div>

        <!-- 文字 -->
        <div class="ml-[20px] flex-1">
          <div v-if="itemIconInfo.itemType === 1">
            <NInput v-model:value="itemIconInfo.text" class="mb-[5px]" size="small" type="text" @input="handleChange" />
          </div>

          <!-- 在线图标 / 管理员图库选择 -->
          <div v-if="itemIconInfo.itemType === 3">
            <div class="space-y-2">
              <NInput
                v-model:value="itemIconInfo.text"
                class="mb-[5px]"
                size="small"
                type="text"
                :placeholder="$t('iconItem.inputIconName')"
                @input="handleChange"
              />

              <div class="flex items-center gap-2">
                <NButton size="small" type="primary" secondary @click="openGalleryPicker">
                  <template #icon>
                    <SvgIcon icon="mingcute:more-1-fill" />
                  </template>
                  从管理员图库选择
                </NButton>
              </div>
            </div>
          </div>

          <!-- 本地图片或图库 -->
          <div v-if="itemIconInfo.itemType === 2">
            <NInput
              v-model:value="itemIconInfo.src"
              class="mb-[5px] w-full"
              size="small"
              type="text"
              :placeholder="$t('iconItem.inputIconUrlOrUpload')"
              @input="handleChange"
            />
            <div class="flex items-center gap-2">
              <NUpload
                action="/api/file/uploadImg"
                :show-file-list="false"
                name="imgfile"
                :data="{ fileType: 1 }"
                :headers="{ token: authStore.token as string }"
                @finish="handleUploadFinish"
              >
                <NButton size="small">
                  {{ $t('iconItem.selectUpload') }}
                </NButton>
              </NUpload>
              <NButton size="small" type="primary" secondary @click="openGalleryPicker">
                从管理员图库选择
              </NButton>
            </div>
          </div>
        </div>
      </div>

      <div class="flex items-center mt-[10px]">
        <div class="w-auto text-slate-500 mr-[10px]">
          {{ $t('common.backgroundColor') }}
        </div>
        <div class="w-[150px] flex items-center mr-[10px]">
          <NColorPicker
            v-model:value="itemIconInfo.backgroundColor"
            :show-alpha="true"
            size="small"
            :swatches="defautSwatchesBackground"
            @complete="handleChange"
          />
        </div>
        <NButton quaternary size="small" @click="handleResetBackgroundColor">
          {{ $t('common.reset') }}
        </NButton>
      </div>
    </div>

    <!-- 内置管理员图库 / 公共图标选择弹窗 -->
    <NModal
      v-model:show="galleryModalVisible"
      preset="card"
      title="选择管理员图库 / 公共图标"
      style="max-width: 600px; border-radius: 12px;"
    >
      <NSpin :show="galleryLoading">
        <div v-if="publicIcons.length === 0 && !galleryLoading" class="py-12 text-center text-slate-400">
          管理员尚未上传公共图标或图库
        </div>
        <div v-else class="max-h-[380px] overflow-y-auto p-1">
          <NGrid cols="3 s:4 m:6" :x-gap="8" :y-gap="8">
            <NGridItem v-for="item in publicIcons" :key="item.id">
              <div
                class="border rounded-lg p-2 bg-slate-50 dark:bg-zinc-800 hover:border-blue-500 hover:shadow-md cursor-pointer transition flex flex-col items-center justify-center text-center group"
                @click="handleSelectIconFromGallery(item)"
              >
                <div class="w-12 h-12 flex items-center justify-center rounded overflow-hidden mb-1">
                  <NImage
                    :src="item.src"
                    preview-disabled
                    class="w-full h-full object-contain"
                  />
                </div>
                <div class="text-[11px] truncate w-full text-slate-600 dark:text-zinc-300 font-mono" :title="item.fileName">
                  {{ item.fileName }}
                </div>
              </div>
            </NGridItem>
          </NGrid>
        </div>
      </NSpin>
    </NModal>
  </div>
</template>
