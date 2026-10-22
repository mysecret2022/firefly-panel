<script setup lang="ts">
import { VueDraggable } from 'vue-draggable-plus'
import { NBackTop, NButton, NButtonGroup, NDropdown, NModal, NSkeleton, NSpin, useDialog, useMessage } from 'naive-ui'
import { nextTick, onMounted, ref } from 'vue'
import { AppIcon, AppStarter, EditItem } from './components'
import { Clock, SearchBox, SystemMonitor } from '@/components/deskModule'
import { SvgIcon } from '@/components/common'
import { deletes, getListByGroupId, saveSort } from '@/api/panel/itemIcon'
import { getList as getGroupList } from '@/api/panel/itemIconGroup'

import { setTitle, updateLocalUserInfo } from '@/utils/cmn'
import { useAuthStore, usePanelState } from '@/store'
import { PanelPanelConfigStyleEnum, PanelStateNetworkModeEnum } from '@/enums'
import { VisitMode } from '@/enums/auth'
import { router } from '@/router'
import { t } from '@/locales'

interface ItemGroup extends Panel.ItemIconGroup {
  sortStatus?: boolean
  hoverStatus: boolean
  items?: Panel.ItemInfo[]
}

const ms = useMessage()
const dialog = useDialog()
const panelState = usePanelState()
const authStore = useAuthStore()

const scrollContainerRef = ref<HTMLElement | undefined>(undefined)

const editItemInfoShow = ref<boolean>(false)
const editItemInfoData = ref<Panel.ItemInfo | null>(null)
const windowShow = ref<boolean>(false)
const windowSrc = ref<string>('')
const windowTitle = ref<string>('')

const windowIframeRef = ref(null)
const windowIframeIsLoad = ref<boolean>(false)

const dropdownMenuX = ref(0)
const dropdownMenuY = ref(0)
const dropdownShow = ref(false)
const currentRightSelectItem = ref<Panel.ItemInfo | null>(null)
const currentAddItenIconGroupId = ref<number | undefined>()

const settingModalShow = ref(false)

const items = ref<ItemGroup[]>([])
const filterItems = ref<ItemGroup[]>([])

// 判断指定主机名是否属于内网局域网地址
function isPrivateIpOrLanHost(hostname: string): boolean {
  if (
    hostname === 'localhost'
    || hostname === '127.0.0.1'
    || hostname.endsWith('.local')
    || hostname.endsWith('.lan')
  ) {
    return true
  }

  // 192.168.x.x
  if (/^192\.168\.\d{1,3}\.\d{1,3}$/.test(hostname))
    return true

  // 10.x.x.x
  if (/^10\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(hostname))
    return true

  // 172.16.x.x - 172.31.x.x
  const match172 = hostname.match(/^172\.(\d{1,3})\.\d{1,3}\.\d{1,3}$/)
  if (match172) {
    const second = Number.parseInt(match172[1], 10)
    if (second >= 16 && second <= 31)
      return true
  }

  return false
}

// 内网连通性动态探测标记 (默认先检查当前地址栏)
const isLanDetected = ref<boolean>(isPrivateIpOrLanHost(window.location.hostname))
let hasProbedLan = false

// 探活探测（用于从外网域名访问时，探测当前客户端是否处于内网）
async function probeLanReachability(testUrl?: string) {
  // 如果本身就是通过内网 IP 打开的面板，直接确认为内网
  if (isPrivateIpOrLanHost(window.location.hostname)) {
    isLanDetected.value = true
    return
  }

  if (!testUrl || hasProbedLan)
    return

  try {
    hasProbedLan = true
    const controller = new AbortController()
    const timer = setTimeout(() => controller.abort(), 450)
    await fetch(testUrl, { mode: 'no-cors', signal: controller.signal })
    clearTimeout(timer)
    isLanDetected.value = true
  }
  catch {
    isLanDetected.value = false
  }
}

// 计算卡片最优跳转地址
function getOptimalUrl(item?: Panel.ItemInfo | null): string {
  if (!item)
    return ''

  // 1. 若用户手动切到 LAN 模式，或处于自动模式且处于内网环境，则优先选择 lanUrl
  const preferLan = panelState.networkMode === PanelStateNetworkModeEnum.lan
    || (panelState.networkMode !== PanelStateNetworkModeEnum.wan && isLanDetected.value)

  let targetUrl = item.url
  if (preferLan && item.lanUrl && item.lanUrl.trim() !== '') {
    targetUrl = item.lanUrl
  }
  else if (!targetUrl && item.lanUrl) {
    targetUrl = item.lanUrl
  }

  return targetUrl || ''
}

function openPage(openMethod: number, url: string, title?: string) {
  switch (openMethod) {
    case 1:
      window.location.href = url
      break
    case 2:
      window.open(url)
      break
    case 3:
      windowShow.value = true
      windowSrc.value = url
      windowTitle.value = title || url
      windowIframeIsLoad.value = true
      break

    default:
      break
  }
}

function handleItemClick(itemGroupIndex: number, item: Panel.ItemInfo) {
  if (items.value[itemGroupIndex] && items.value[itemGroupIndex].sortStatus) {
    handleEditItem(item)
    return
  }

  const jumpUrl = getOptimalUrl(item)
  openPage(item.openMethod, jumpUrl, item.title)
}

function handWindowIframeIdLoad(payload: Event) {
  windowIframeIsLoad.value = false
}

function getList() {
  // 获取组数据
  getGroupList<Common.ListResponse<ItemGroup[]>>().then(({ code, data, msg }) => {
    if (code === 0)
      items.value = data.list
    for (let i = 0; i < data.list.length; i++) {
      const element = data.list[i]
      if (element.id)
        updateItemIconGroupByNet(i, element.id)
    }
    filterItems.value = items.value
    // console.log(items)
  })
}

// 从后端获取组下面的图标
function updateItemIconGroupByNet(itemIconGroupIndex: number, itemIconGroupId: number) {
  getListByGroupId<Common.ListResponse<Panel.ItemInfo[]>>(itemIconGroupId).then((res) => {
    if (res.code === 0) {
      items.value[itemIconGroupIndex].items = res.data.list
      // 触发后台轻量探活
      if (!hasProbedLan && !isLanDetected.value) {
        const sampleLanUrl = res.data.list?.find(i => i.lanUrl && i.lanUrl.startsWith('http'))?.lanUrl
        if (sampleLanUrl)
          probeLanReachability(sampleLanUrl)
      }
    }
  })
}

function handleRightMenuSelect(key: string | number) {
  dropdownShow.value = false
  const jumpUrl = getOptimalUrl(currentRightSelectItem.value)
  switch (key) {
    case 'newWindows':
      window.open(jumpUrl)
      break
    case 'openWanUrl':
      if (currentRightSelectItem.value)
        openPage(currentRightSelectItem.value?.openMethod, currentRightSelectItem.value?.url, currentRightSelectItem.value?.title)
      break
    case 'openLanUrl':
      if (currentRightSelectItem.value && currentRightSelectItem.value.lanUrl)
        openPage(currentRightSelectItem.value?.openMethod, currentRightSelectItem.value.lanUrl, currentRightSelectItem.value?.title)
      break
    case 'edit':
      // 这里有个奇怪的问题，如果不使用{...}的方式 父组件的值会同步修改 标记一下
      handleEditItem({ ...currentRightSelectItem.value } as Panel.ItemInfo)
      break
    case 'delete':
      dialog.warning({
        title: t('common.warning'),
        content: t('common.deleteConfirmByName', { name: currentRightSelectItem.value?.title }),
        positiveText: t('common.confirm'),
        negativeText: t('common.cancel'),
        onPositiveClick: () => {
          deletes([currentRightSelectItem.value?.id as number]).then(({ code, msg }) => {
            if (code === 0) {
              ms.success(t('common.deleteSuccess'))
              getList()
            }
            else {
              ms.error(`${t('common.deleteFail')}:${msg}`)
            }
          })
        },
      })

      break
    default:
      break
  }
}

function handleContextMenu(e: MouseEvent, itemGroupIndex: number, item: Panel.ItemInfo) {
  if (items.value[itemGroupIndex] && items.value[itemGroupIndex].sortStatus)
    return

  e.preventDefault()
  currentRightSelectItem.value = item
  dropdownShow.value = false
  nextTick().then(() => {
    dropdownShow.value = true
    dropdownMenuX.value = e.clientX
    dropdownMenuY.value = e.clientY
  })
}

function onClickoutside() {
  // message.info('clickoutside')
  dropdownShow.value = false
}

function handleEditSuccess(item: Panel.ItemInfo) {
  getList()
}

function handleChangeNetwork(mode: PanelStateNetworkModeEnum) {
  panelState.setNetworkMode(mode)
  if (mode === PanelStateNetworkModeEnum.lan) {
    ms.success('已切换为【🏠 内网模式】（优先局域网直连）')
  }
  else if (mode === PanelStateNetworkModeEnum.wan) {
    ms.success('已切换为【🌐 外网模式】（优先公网域名访问）')
  }
  else {
    ms.success(`已切换为【🤖 自动感知模式】(${isLanDetected.value ? '已识别为内网环境' : '已识别为外网环境'})`)
  }
}

// 结束拖拽
// function handleEndDrag(event: any, itemIconGroup: Panel.ItemIconGroup) {
//   // console.log(event)
//   // console.log(items.value)
// }

function handleSaveSort(itemGroup: ItemGroup) {
  const saveItems: Common.SortItemRequest[] = []
  if (itemGroup.items) {
    for (let i = 0; i < itemGroup.items.length; i++) {
      const element = itemGroup.items[i]
      saveItems.push({
        id: element.id as number,
        sort: i + 1,
      })
    }

    saveSort({ itemIconGroupId: itemGroup.id as number, sortItems: saveItems }).then(({ code, msg }) => {
      if (code === 0) {
        ms.success(t('common.saveSuccess'))
        itemGroup.sortStatus = false
      }
      else {
        ms.error(`${t('common.saveFail')}:${msg}`)
      }
    })
  }
}

function getDropdownMenuOptions() {
  const dropdownMenuOptions = [
    {
      label: t('iconItem.newWindowOpen'),
      key: 'newWindows',
    },
  ]

  if (currentRightSelectItem.value?.lanUrl) {
    const isPreferLan = panelState.networkMode === PanelStateNetworkModeEnum.lan
      || (panelState.networkMode !== PanelStateNetworkModeEnum.wan && isLanDetected.value)
    if (isPreferLan) {
      dropdownMenuOptions.push({
        label: t('panelHome.openWanUrl'),
        key: 'openWanUrl',
      })
    }
    else {
      dropdownMenuOptions.push({
        label: t('panelHome.openLanUrl'),
        key: 'openLanUrl',
      })
    }
  }

  if (authStore.visitMode === VisitMode.VISIT_MODE_LOGIN) {
    dropdownMenuOptions.push({
      label: t('common.edit'),
      key: 'edit',
    }, {
      label: t('common.delete'),
      key: 'delete',
    })
  }

  return dropdownMenuOptions
}

onMounted(() => {
  // 更新用户信息
  updateLocalUserInfo()
  getList()

  // 遍历首个具有 lanUrl 的卡片进行后台快速探活
  const sampleLanUrl = items.value
    .flatMap(g => g.items || [])
    .find(i => i.lanUrl && i.lanUrl.startsWith('http'))?.lanUrl

  if (sampleLanUrl) {
    probeLanReachability(sampleLanUrl)
  }

  // 更新同步云端配置
  panelState.updatePanelConfigByCloud()

  // 设置标题
  if (panelState.panelConfig.logoText)
    setTitle(panelState.panelConfig.logoText)
})

// 前端搜索过滤
function itemFrontEndSearch(keyword?: string) {
  keyword = keyword?.trim()
  if (keyword !== '' && panelState.panelConfig.searchBoxSearchIcon) {
    const filteredData = ref<ItemGroup[]>([])
    for (let i = 0; i < items.value.length; i++) {
      const element = items.value[i].items?.filter((item: Panel.ItemInfo) => {
        return (
          item.title.toLowerCase().includes(keyword?.toLowerCase() ?? '')
          || item.url.toLowerCase().includes(keyword?.toLowerCase() ?? '')
          || item.description?.toLowerCase().includes(keyword?.toLowerCase() ?? '')
        )
      })
      if (element && element.length > 0)
        filteredData.value.push({ items: element, hoverStatus: false })
    }
    filterItems.value = filteredData.value
  }
  else {
    filterItems.value = items.value
  }
}

function handleSetHoverStatus(groupIndex: number, hoverStatus: boolean) {
  if (items.value[groupIndex])
    items.value[groupIndex].hoverStatus = hoverStatus
}

function handleSetSortStatus(groupIndex: number, sortStatus: boolean) {
  if (items.value[groupIndex])
    items.value[groupIndex].sortStatus = sortStatus

  // 并未保存排序重新更新数据
  if (!sortStatus) {
    // 单独更新组
    if (items.value[groupIndex] && items.value[groupIndex].id)
      updateItemIconGroupByNet(groupIndex, items.value[groupIndex].id as number)
  }
}

function handleEditItem(item: Panel.ItemInfo) {
  editItemInfoData.value = item
  editItemInfoShow.value = true
  currentAddItenIconGroupId.value = undefined
}

function handleAddItem(itemIconGroupId?: number) {
  editItemInfoData.value = null
  editItemInfoShow.value = true
  if (itemIconGroupId)
    currentAddItenIconGroupId.value = itemIconGroupId
}
</script>

<template>
  <div class="w-full h-full sun-main">
    <div
      class="cover wallpaper" :style="{
        filter: `blur(${panelState.panelConfig.backgroundBlur}px)`,
        background: `url(${panelState.panelConfig.backgroundImageSrc}) no-repeat`,
        backgroundSize: 'cover',
        backgroundPosition: 'center',
      }"
    />
    <div class="mask" :style="{ backgroundColor: `rgba(0,0,0,${panelState.panelConfig.backgroundMaskNumber})` }" />
    <!-- 滚动容器：保证内部充满高度实现页脚沉底 -->
    <div ref="scrollContainerRef" class="absolute w-full h-full overflow-auto flex flex-col">
      <div
        class="p-2.5 mx-auto w-full flex-1 flex flex-col justify-between box-border"
        :style="{
          marginTop: `${panelState.panelConfig.marginTop}%`,
          marginBottom: `${panelState.panelConfig.marginBottom}%`,
          maxWidth: (panelState.panelConfig.maxWidth ?? '1200') + panelState.panelConfig.maxWidthUnit,
        }"
      >
        <!-- 上半部分：头部与所有卡片组 -->
        <div>
          <!-- 头部 LOGO 与时钟 (颜色、透明度统一联动) -->
        <div class="mx-[auto] w-[80%]">
          <div 
            class="flex mx-[auto] items-center justify-center transition-all duration-300"
            :style="{ 
              opacity: ((panelState.panelConfig.headerOpacity ?? 90) / 100),
              color: panelState.panelConfig.logoColor || '#ffffff',
            }"
          >
            <!-- 主标题 -->
            <div class="logo">
              <span 
                class="text-2xl md:text-6xl font-bold select-none transition-all duration-300"
                :class="panelState.panelConfig.logoGlassEffect ? 'ios27-glass-text' : ''"
                :style="{
                  fontFamily: panelState.panelConfig.logoFontFamily || 'system-ui',
                  color: panelState.panelConfig.logoColor || '#ffffff',
                  textShadow: panelState.panelConfig.logoGlassEffect ? '' : '0 2px 12px rgba(0,0,0,0.45)',
                }"
              >
                {{ panelState.panelConfig.logoText }}
              </span>
            </div>

            <!-- 分割线 -->
            <div 
              class="divider text-base lg:text-2xl mx-[12px] select-none opacity-60 font-light"
              :style="{ color: panelState.panelConfig.logoColor || '#ffffff' }"
            >
              |
            </div>

            <!-- 时钟区域 (颜色同步传入) -->
            <div class="select-none flex items-center">
              <Clock 
                :hide-second="!panelState.panelConfig.clockShowSecond" 
                :color="panelState.panelConfig.logoColor || '#ffffff'"
              />
            </div>
          </div>

          <div v-if="panelState.panelConfig.searchBoxShow" class="flex mt-[20px] mx-auto sm:w-full lg:w-[80%]">
            <SearchBox @itemSearch="itemFrontEndSearch" />
          </div>
        </div>

        <!-- 应用盒子 -->
        <div :style="{ marginLeft: `${panelState.panelConfig.marginX}px`, marginRight: `${panelState.panelConfig.marginX}px` }">
          <!-- 系统监控状态 -->
          <div
            v-if="panelState.panelConfig.systemMonitorShow
              && ((panelState.panelConfig.systemMonitorPublicVisitModeShow && authStore.visitMode === VisitMode.VISIT_MODE_PUBLIC)
                || authStore.visitMode === VisitMode.VISIT_MODE_LOGIN)"
            class="flex mx-auto"
          >
            <SystemMonitor
              :allow-edit="authStore.visitMode === VisitMode.VISIT_MODE_LOGIN"
              :show-title="panelState.panelConfig.systemMonitorShowTitle"
            />
          </div>

          <!-- 组纵向排列 -->
          <div
            v-for="(itemGroup, itemGroupIndex) in filterItems" :key="itemGroupIndex"
            class="item-list mt-[50px]"
            :class="itemGroup.sortStatus ? 'shadow-2xl border shadow-[0_0_30px_10px_rgba(0,0,0,0.3)]  p-[10px] rounded-2xl' : ''"
            @mouseenter="handleSetHoverStatus(itemGroupIndex, true)"
            @mouseleave="handleSetHoverStatus(itemGroupIndex, false)"
          >
            <!-- 分组标题 -->
            <div class="text-white text-xl font-extrabold mb-[20px] ml-[10px] flex items-center">
              <span class="group-title text-shadow">
                {{ itemGroup.title }}
              </span>
              <div
                v-if="authStore.visitMode === VisitMode.VISIT_MODE_LOGIN"
                class="group-buttons ml-2 delay-100 transition-opacity flex"
                :class="itemGroup.hoverStatus ? 'opacity-100' : 'opacity-0'"
              >
                <span class="mr-2 cursor-pointer" :title="t('common.add')" @click="handleAddItem(itemGroup.id)">
                  <SvgIcon class="text-white font-xl" icon="typcn:plus" />
                </span>
                <span class="mr-2 cursor-pointer " :title="t('common.sort')" @click="handleSetSortStatus(itemGroupIndex, !itemGroup.sortStatus)">
                  <SvgIcon class="text-white font-xl" icon="ri:drag-drop-line" />
                </span>
              </div>
            </div>

            <!-- 详情图标 -->
            <div v-if="Number(panelState.panelConfig.iconStyle) === 0 || panelState.panelConfig.iconStyle === PanelPanelConfigStyleEnum.info">
              <div v-if="itemGroup.items">
                <VueDraggable
                  v-model="itemGroup.items" item-key="sort" :animation="300"
                  class="icon-info-box"
                  filter=".not-drag"
                  :disabled="!itemGroup.sortStatus"
                >
                  <div v-for="item, index in itemGroup.items" :key="index" :title="item.description" @contextmenu="(e) => handleContextMenu(e, itemGroupIndex, item)">
                    <AppIcon
                      :class="itemGroup.sortStatus ? 'cursor-move' : 'cursor-pointer'"
                      :item-info="item"
                      :icon-text-color="panelState.panelConfig.iconTextColor"
                      :icon-text-info-hide-description="panelState.panelConfig.iconTextInfoHideDescription || false"
                      :icon-text-icon-hide-title="panelState.panelConfig.iconTextIconHideTitle || false"
                      :style="PanelPanelConfigStyleEnum.info"
                      @click="handleItemClick(itemGroupIndex, item)"
                    />
                  </div>

                  <div v-if="itemGroup.items.length === 0" class="not-drag">
                    <AppIcon
                      :class="itemGroup.sortStatus ? 'cursor-move' : 'cursor-pointer'"
                      :item-info="{ icon: { itemType: 3, text: 'subway:add' }, title: t('common.add'), url: '', openMethod: 0 }"
                      :icon-text-color="panelState.panelConfig.iconTextColor"
                      :icon-text-info-hide-description="panelState.panelConfig.iconTextInfoHideDescription || false"
                      :icon-text-icon-hide-title="panelState.panelConfig.iconTextIconHideTitle || false"
                      :style="PanelPanelConfigStyleEnum.info"
                      @click="handleAddItem(itemGroup.id)"
                    />
                  </div>
                </VueDraggable>
              </div>
            </div>

            <!-- APP图标宫型盒子 -->
            <div v-if="Number(panelState.panelConfig.iconStyle) === 1 || panelState.panelConfig.iconStyle === PanelPanelConfigStyleEnum.icon">
              <div v-if="itemGroup.items">
                <VueDraggable
                  v-model="itemGroup.items" item-key="sort" :animation="300"
                  class="icon-small-box"

                  filter=".not-drag"
                  :disabled="!itemGroup.sortStatus"
                >
                  <div v-for="item, index in itemGroup.items" :key="index" :title="item.description" @contextmenu="(e) => handleContextMenu(e, itemGroupIndex, item)">
                    <AppIcon
                      :class="itemGroup.sortStatus ? 'cursor-move' : 'cursor-pointer'"
                      :item-info="item"
                      :icon-text-color="panelState.panelConfig.iconTextColor"
                      :icon-text-info-hide-description="!panelState.panelConfig.iconTextInfoHideDescription"
                      :icon-text-icon-hide-title="panelState.panelConfig.iconTextIconHideTitle || false"
                      :style="1"
                      @click="handleItemClick(itemGroupIndex, item)"
                    />
                  </div>

                  <div v-if="itemGroup.items.length === 0" class="not-drag">
                    <AppIcon
                      class="cursor-pointer"
                      :item-info="{ icon: { itemType: 3, text: 'subway:add' }, title: $t('common.add'), url: '', openMethod: 0 }"
                      :icon-text-color="panelState.panelConfig.iconTextColor"
                      :icon-text-info-hide-description="!panelState.panelConfig.iconTextInfoHideDescription"
                      :icon-text-icon-hide-title="panelState.panelConfig.iconTextIconHideTitle || false"
                      :style="1"
                      @click="handleAddItem(itemGroup.id)"
                    />
                  </div>
                </vuedraggable>
              </div>
            </div>

            <!-- 编辑栏 -->
            <div v-if="itemGroup.sortStatus" class="flex mt-[10px]">
              <div>
                <NButton color="#2a2a2a6b" @click="handleSaveSort(itemGroup)">
                  <template #icon>
                    <SvgIcon class="text-white font-xl" icon="material-symbols:save" />
                  </template>
                  <div>
                    {{ $t('common.saveSort') }}
                  </div>
                </NButton>
              </div>
            </div>
          </div>
        </div>
        </div>

        <!-- 下半部分：纯净沉底页脚 (最大化时稳居屏幕最底端) -->
        <div 
          class="footer-clean-wrap select-none mt-10 pb-4"
          :style="{ 
            transform: `translateY(${panelState.panelConfig.footerOffsetY ?? 0}px)`,
          }"
        >
          <div 
            class="footer"
            v-html="(panelState.panelConfig.footerHtml || '').replace('my-sun-panel', 'firefly-panel')" 
          />
        </div>
      </div>
    </div>

    <!-- 右键菜单 -->
    <NDropdown
      placement="bottom-start" trigger="manual" :x="dropdownMenuX" :y="dropdownMenuY"
      :options="getDropdownMenuOptions()" :show="dropdownShow" :on-clickoutside="onClickoutside" @select="handleRightMenuSelect"
    />

    <!-- 悬浮按钮 -->
    <div class="fixed-element shadow-[0_0_10px_2px_rgba(0,0,0,0.2)]">
      <NButtonGroup vertical>
        <!-- 网络模式三态切换按钮组 (🤖 自动 -> 🏠 内网 -> 🌐 外网 -> 🤖 自动) -->
        <NButton
          v-if="(panelState.networkMode === PanelStateNetworkModeEnum.auto || panelState.networkMode === null || panelState.networkMode === undefined) && panelState.panelConfig.netModeChangeButtonShow"
          color="#2a2a2a6b"
          :title="`当前网络：🤖 自动感知模式 (${isLanDetected ? '内网优先' : '外网优先'})\n点击切换至：🏠 强制内网直连`"
          @click="handleChangeNetwork(PanelStateNetworkModeEnum.lan)"
        >
          <template #icon>
            <SvgIcon class="text-emerald-400 font-xl" icon="material-symbols:auto-mode-rounded" />
          </template>
        </NButton>

        <NButton
          v-if="panelState.networkMode === PanelStateNetworkModeEnum.lan && panelState.panelConfig.netModeChangeButtonShow"
          color="#2a2a2a6b"
          title="当前网络：🏠 强制内网直连&#10;点击切换至：🌐 强制外网公网"
          @click="handleChangeNetwork(PanelStateNetworkModeEnum.wan)"
        >
          <template #icon>
            <SvgIcon class="text-sky-400 font-xl" icon="material-symbols:lan-outline-rounded" />
          </template>
        </NButton>

        <NButton
          v-if="panelState.networkMode === PanelStateNetworkModeEnum.wan && panelState.panelConfig.netModeChangeButtonShow"
          color="#2a2a2a6b"
          title="当前网络：🌐 强制外网公网&#10;点击切换至：🤖 自动感知模式"
          @click="handleChangeNetwork(PanelStateNetworkModeEnum.auto)"
        >
          <template #icon>
            <SvgIcon class="text-amber-400 font-xl" icon="mdi:wan" />
          </template>
        </NButton>

        <NButton v-if="authStore.visitMode === VisitMode.VISIT_MODE_LOGIN" color="#2a2a2a6b" @click="settingModalShow = !settingModalShow">
          <template #icon>
            <SvgIcon class="text-white font-xl" icon="majesticons-applications" />
          </template>
        </NButton>

        <NButton v-if="authStore.visitMode === VisitMode.VISIT_MODE_PUBLIC" color="#2a2a2a6b" :title="$t('panelHome.goToLogin')" @click="router.push('/login')">
          <template #icon>
            <SvgIcon class="text-white font-xl" icon="material-symbols:account-circle" />
          </template>
        </NButton>
      </NButtonGroup>

      <AppStarter v-model:visible="settingModalShow" />
      <!-- <Setting v-model:visible="settingModalShow" /> -->
    </div>

    <NBackTop
      :listen-to="() => scrollContainerRef"
      :right="10"
      :bottom="10"
      style="background-color:transparent;border: none;box-shadow: none;"
    >
      <div class="shadow-[0_0_10px_2px_rgba(0,0,0,0.2)]">
        <NButton color="#2a2a2a6b">
          <template #icon>
            <SvgIcon class="text-white font-xl" icon="icon-park-outline:to-top" />
          </template>
        </NButton>
      </div>
    </NBackTop>

    <EditItem v-model:visible="editItemInfoShow" :item-info="editItemInfoData" :item-group-id="currentAddItenIconGroupId" @done="handleEditSuccess" />

    <!-- 弹窗 -->
    <NModal
      v-model:show="windowShow" :mask-closable="false" preset="card"
      style="max-width: 1000px;height: 600px;border-radius: 1rem;" :bordered="true" size="small" role="dialog"
      aria-modal="true"
    >
      <template #header>
        <div class="flex items-center">
          <span class="mr-[20px]">
            {{ windowTitle }}
          </span>

          <NSpin v-if="windowIframeIsLoad" size="small" />
        </div>
      </template>
      <div class="w-full h-full rounded-2xl overflow-hidden border dark:border-zinc-700">
        <div v-if="windowIframeIsLoad" class="flex flex-col p-5">
          <NSkeleton height="50px" width="100%" class="rounded-lg" />
          <NSkeleton height="180px" width="100%" class="mt-[20px] rounded-lg" />
          <NSkeleton height="180px" width="100%" class="mt-[20px] rounded-lg" />
        </div>
        <iframe
          v-show="!windowIframeIsLoad" id="windowIframeId" ref="windowIframeRef" :src="windowSrc"
          class="w-full h-full" frameborder="0" @load="handWindowIframeIdLoad"
        />
      </div>
    </NModal>
  </div>
</template>

<style>
body,
html {
  overflow: hidden;
  background-color: rgb(54, 54, 54);
}
</style>

<style scoped>
.mask {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
}

.sun-main {
  user-select: none;
}

.cover {
  position: absolute;
  width: 100%;
  height: 100%;
  overflow: hidden;
  /* background: url(@/assets/start_sky.jpg) no-repeat; */

  transform: scale(1.05);
}

.text-shadow {
  text-shadow: 2px 2px 50px rgb(0, 0, 0);
}

.ios27-glass-text {
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.95) 0%, rgba(255, 255, 255, 0.55) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  text-shadow: 0 4px 18px rgba(0, 0, 0, 0.35), 0 1px 2px rgba(255, 255, 255, 0.4);
  filter: drop-shadow(0 2px 8px rgba(0, 0, 0, 0.28));
  letter-spacing: -0.02em;
  transition: all 0.3s ease;
}

.ios27-glass-divider {
  color: rgba(255, 255, 255, 0.4);
  text-shadow: 0 2px 10px rgba(0, 0, 0, 0.4);
  font-weight: 200;
}

/* 纯净沉底页脚容器 */
.footer-clean-wrap {
  width: 100%;
  display: flex;
  justify-content: center;
  align-items: center;
  margin-top: auto; /* 关键：自动推到容器最底部 */
}

/* 取消毛玻璃胶囊，呈现纯净透明文本 */
.footer,
.footer > div,
.ios27-glass-footer {
  display: flex !important;
  justify-content: center !important;
  align-items: center !important;
  margin: 0 auto !important;
  padding: 6px 0 !important;
  width: 100% !important;
  max-width: 100% !important;
  background: transparent !important;
  backdrop-filter: none !important;
  -webkit-backdrop-filter: none !important;
  border: none !important;
  border-radius: 0 !important;
  box-shadow: none !important;
  color: rgba(255, 255, 255, 0.72) !important;
  font-size: 0.85rem !important;
  text-shadow: 0 1px 4px rgba(0, 0, 0, 0.65) !important;
  transition: all 0.2s ease !important;
}

/* 页脚链接样式 */
.footer a,
.footer > div a,
.ios27-glass-footer a {
  color: rgba(255, 255, 255, 0.95) !important;
  font-weight: 600 !important;
  margin-left: 6px !important;
  text-decoration: none !important;
  text-shadow: 0 1px 4px rgba(0, 0, 0, 0.65) !important;
  transition: all 0.2s ease !important;
}

.footer a:hover,
.footer > div a:hover,
.ios27-glass-footer a:hover {
  color: #6ee7b7 !important;
  text-shadow: 0 0 12px rgba(110, 231, 183, 0.9) !important;
}

.footer:empty,
.footer > div:empty {
  display: none !important;
}

.app-icon-text-shadow {
  text-shadow: 2px 2px 5px rgb(0, 0, 0);
}

.fixed-element {
  position: fixed;
  /* 将元素固定在屏幕上 */
  right: 10px;
  /* 距离屏幕顶部的距离 */
  bottom: 50px;
  /* 距离屏幕左侧的距离 */
}

.icon-info-box {
  width: 100%;
  display: grid;
  /* 适配长方形卡片尺寸：每张卡片最小 215px，自适应铺满整行 */
  grid-template-columns: repeat(auto-fill, minmax(215px, 1fr));
  gap: 16px;
}

.icon-small-box {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(75px, 1fr));
  gap: 18px;

}

@media (max-width: 500px) {
  .icon-info-box{
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  }
}
</style>
