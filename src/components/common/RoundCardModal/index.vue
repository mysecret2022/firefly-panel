<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, useAttrs, watch } from 'vue'
import { NButton, NModal } from 'naive-ui'
import { SvgIcon } from '@/components/common'

const props = withDefaults(
  defineProps<{
    title?: string
    show: boolean
    size?: 'medium' | 'small' | 'large' | 'huge' | undefined
    draggable?: boolean
    resizable?: boolean
  }>(),
  {
    draggable: true,
    resizable: true,
  },
)

const emit = defineEmits<Emit>()
interface Emit {
  (e: 'update:show', show: boolean): void
}

const attrs = useAttrs()
const modalUniqueClass = `round-modal-${Math.random().toString(36).slice(2, 9)}`

const isMaximized = ref(false)
const prevRect = ref<{ left: number; top: number; width: number; height: number } | null>(null)

const bindAttrs = computed<{ class: string; style: string }>(() => ({
  class: `${(attrs.class as string) || ''} ${modalUniqueClass} draggable-resizable-modal ${isMaximized.value ? 'is-maximized' : ''}`,
  style: (attrs.style as string) || '',
}))

const showModal = computed({
  get: () => props.show,
  set: (show: boolean) => {
    emit('update:show', show)
  },
})

// 拖拽移动状态
let isDragging = false
let dragMouseStartX = 0
let dragMouseStartY = 0
let cardStartLeft = 0
let cardStartTop = 0

// 8 方向自由缩放调整状态 (E, W, S, N, SE, SW, NE, NW)
type ResizeDirection = 'n' | 's' | 'e' | 'w' | 'ne' | 'nw' | 'se' | 'sw'
let isResizing = false
let currentResizeDir: ResizeDirection = 'se'
let resizeStartX = 0
let resizeStartY = 0
let resizeStartLeft = 0
let resizeStartTop = 0
let resizeStartWidth = 0
let resizeStartHeight = 0

function getCardEl(): HTMLElement | null {
  return document.querySelector(`.${modalUniqueClass}`) as HTMLElement | null
}

// 确保卡片转换为 fixed 定位，便于绝对坐标拖拽与定点缩放
function ensureFixedPosition(card: HTMLElement) {
  if (card.style.position !== 'fixed' && !isMaximized.value) {
    const rect = card.getBoundingClientRect()
    card.style.position = 'fixed'
    card.style.left = `${rect.left}px`
    card.style.top = `${rect.top}px`
    card.style.margin = '0'
    card.style.transform = 'none'
    card.style.width = `${Math.max(320, rect.width)}px`
    card.style.height = `${Math.max(300, rect.height)}px`
  }
}

function handleHeaderMouseDown(e: MouseEvent) {
  if (props.draggable === false || isMaximized.value)
    return
  const target = e.target as HTMLElement
  if (target.closest('.n-base-close, button, a, input, select, textarea, .n-dropdown-trigger, .modal-header-btn'))
    return

  const card = getCardEl()
  if (!card)
    return

  ensureFixedPosition(card)

  isDragging = true
  dragMouseStartX = e.clientX
  dragMouseStartY = e.clientY
  cardStartLeft = parseFloat(card.style.left) || card.getBoundingClientRect().left
  cardStartTop = parseFloat(card.style.top) || card.getBoundingClientRect().top
  card.style.transition = 'none'

  document.addEventListener('mousemove', onDragMouseMove)
  document.addEventListener('mouseup', onDragMouseUp)
  e.preventDefault()
}

function onDragMouseMove(e: MouseEvent) {
  if (!isDragging || isMaximized.value)
    return
  const card = getCardEl()
  if (!card)
    return

  const deltaX = e.clientX - dragMouseStartX
  const deltaY = e.clientY - dragMouseStartY

  const newLeft = Math.max(0, Math.min(window.innerWidth - 80, cardStartLeft + deltaX))
  const newTop = Math.max(0, Math.min(window.innerHeight - 80, cardStartTop + deltaY))

  card.style.left = `${newLeft}px`
  card.style.top = `${newTop}px`
}

function onDragMouseUp() {
  isDragging = false
  document.removeEventListener('mousemove', onDragMouseMove)
  document.removeEventListener('mouseup', onDragMouseUp)
}

// 8 方向拉伸按下：记录起始点并锁定反向锚点
function handleResizeStart(dir: ResizeDirection, e: MouseEvent) {
  if (props.resizable === false || isMaximized.value)
    return
  const card = getCardEl()
  if (!card)
    return

  ensureFixedPosition(card)

  isResizing = true
  currentResizeDir = dir
  resizeStartX = e.clientX
  resizeStartY = e.clientY

  const rect = card.getBoundingClientRect()
  resizeStartLeft = parseFloat(card.style.left) || rect.left
  resizeStartTop = parseFloat(card.style.top) || rect.top
  resizeStartWidth = parseFloat(card.style.width) || rect.width
  resizeStartHeight = parseFloat(card.style.height) || rect.height
  card.style.transition = 'none'

  // 全局防文本选中并锁定对应光标
  document.body.style.userSelect = 'none'
  const cursorMap: Record<ResizeDirection, string> = {
    n: 'ns-resize',
    s: 'ns-resize',
    e: 'ew-resize',
    w: 'ew-resize',
    ne: 'nesw-resize',
    sw: 'nesw-resize',
    nw: 'nwse-resize',
    se: 'nwse-resize',
  }
  document.body.style.cursor = cursorMap[dir]

  document.addEventListener('mousemove', onResizeMouseMove)
  document.addEventListener('mouseup', onResizeMouseUp)
  e.preventDefault()
  e.stopPropagation()
}

function onResizeMouseMove(e: MouseEvent) {
  if (!isResizing || isMaximized.value)
    return
  const card = getCardEl()
  if (!card)
    return

  const dx = e.clientX - resizeStartX
  const dy = e.clientY - resizeStartY
  const startRight = resizeStartLeft + resizeStartWidth
  const startBottom = resizeStartTop + resizeStartHeight
  const minWidth = 320
  const minHeight = 300

  let newWidth = resizeStartWidth
  let newHeight = resizeStartHeight
  let newLeft = resizeStartLeft
  let newTop = resizeStartTop

  // 1. 水平方向缩放与锚定计算
  if (currentResizeDir.includes('e')) {
    // 东（E/NE/SE）：左边严格不动，调整宽度
    newWidth = Math.max(minWidth, Math.min(window.innerWidth - resizeStartLeft - 10, resizeStartWidth + dx))
  }
  else if (currentResizeDir.includes('w')) {
    // 西（W/NW/SW）：右边绝对固定在 startRight，计算新宽度与新 left
    newWidth = Math.min(startRight - 10, Math.max(minWidth, resizeStartWidth - dx))
    newLeft = startRight - newWidth
  }

  // 2. 垂直方向缩放与锚定计算
  if (currentResizeDir.includes('s')) {
    // 南（S/SE/SW）：上边严格不动，调整高度
    newHeight = Math.max(minHeight, Math.min(window.innerHeight - resizeStartTop - 10, resizeStartHeight + dy))
  }
  else if (currentResizeDir.includes('n')) {
    // 北（N/NE/NW）：底边绝对固定在 startBottom，计算新高度与新 top
    newHeight = Math.min(startBottom - 10, Math.max(minHeight, resizeStartHeight - dy))
    newTop = startBottom - newHeight
  }

  // 直接批量应用，保证极致流畅
  card.style.width = `${newWidth}px`
  card.style.height = `${newHeight}px`
  card.style.left = `${newLeft}px`
  card.style.top = `${newTop}px`
  card.style.maxWidth = '99vw'
  card.style.maxHeight = '99vh'
}

function onResizeMouseUp() {
  if (isResizing) {
    isResizing = false
    document.body.style.userSelect = ''
    document.body.style.cursor = ''
    document.removeEventListener('mousemove', onResizeMouseMove)
    document.removeEventListener('mouseup', onResizeMouseUp)
  }
}

// 最大化与还原切换
function toggleMaximize() {
  const card = getCardEl()
  if (!card)
    return

  if (!isMaximized.value) {
    // 记录最大化前的尺寸与坐标
    const rect = card.getBoundingClientRect()
    prevRect.value = {
      left: parseFloat(card.style.left) || rect.left,
      top: parseFloat(card.style.top) || rect.top,
      width: parseFloat(card.style.width) || rect.width,
      height: parseFloat(card.style.height) || rect.height,
    }
    card.style.transition = 'all 0.2s ease'
    card.style.position = 'fixed'
    card.style.left = '10px'
    card.style.top = '10px'
    card.style.width = 'calc(100vw - 20px)'
    card.style.height = 'calc(100vh - 20px)'
    card.style.maxWidth = 'calc(100vw - 20px)'
    card.style.maxHeight = 'calc(100vh - 20px)'
    card.style.margin = '0'
    isMaximized.value = true
  }
  else {
    // 恢复
    card.style.transition = 'all 0.2s ease'
    if (prevRect.value) {
      card.style.left = `${prevRect.value.left}px`
      card.style.top = `${prevRect.value.top}px`
      card.style.width = `${prevRect.value.width}px`
      card.style.height = `${prevRect.value.height}px`
      card.style.maxWidth = '99vw'
      card.style.maxHeight = '99vh'
    }
    else {
      resetPositionAndSize()
    }
    isMaximized.value = false
  }
}

// 双击标题栏在最大化与还原之间切换
function handleHeaderDblClick() {
  toggleMaximize()
}

// 双击重置
function resetPositionAndSize() {
  const card = getCardEl()
  if (card) {
    card.style.position = ''
    card.style.left = ''
    card.style.top = ''
    card.style.transform = ''
    card.style.margin = ''
    card.style.width = ''
    card.style.height = ''
    card.style.maxWidth = ''
    card.style.maxHeight = ''
    card.style.transition = ''
    isMaximized.value = false
  }
}

watch(
  () => props.show,
  (val) => {
    if (val) {
      nextTick(() => {
        setTimeout(() => {
          const card = getCardEl()
          if (!card)
            return

          // 固化外壳基准尺寸，与内部内容完全解耦，防止切换 Tab 或动态增减内容时窗口跳变抖动
          if (!isMaximized.value) {
            const rect = card.getBoundingClientRect()
            if (!card.style.width)
              card.style.width = `${Math.round(rect.width)}px`
            if (!card.style.height)
              card.style.height = `${Math.round(rect.height)}px`
          }

          // 绑定 Header 拖拽与双击
          const header = card.querySelector('.n-card-header') as HTMLElement
          if (header && !header.dataset.dragAttached) {
            header.dataset.dragAttached = 'true'
            header.style.cursor = 'move'
            header.style.userSelect = 'none'
            header.title = '按住拖动窗口，双击最大化/还原'
            header.addEventListener('mousedown', handleHeaderMouseDown)
            header.addEventListener('dblclick', handleHeaderDblClick)

            // 注入最大化按钮到右上角 (位于关闭按钮前)
            const closeBtn = card.querySelector('.n-card-header__close, .n-base-close')
            if (closeBtn && !header.querySelector('.custom-maximize-btn')) {
              const maxBtn = document.createElement('div')
              maxBtn.className = 'custom-maximize-btn modal-header-btn'
              maxBtn.title = '最大化 / 还原窗口'
              maxBtn.innerHTML = `
                <svg width="14" height="14" viewBox="0 0 14 14" fill="none" xmlns="http://www.w3.org/2000/svg">
                  <rect x="2" y="2" width="10" height="10" rx="1.5" stroke="currentColor" stroke-width="1.3"/>
                </svg>
              `
              maxBtn.addEventListener('click', (e) => {
                e.stopPropagation()
                toggleMaximize()
              })
              closeBtn.parentElement?.insertBefore(maxBtn, closeBtn)
            }
          }

          // 挂载原生桌面级 8 方向缩放手柄 (E, W, S, N, NE, NW, SE, SW)
          if (props.resizable !== false && !card.querySelector('.modal-resize-handle')) {
            const directions: { dir: ResizeDirection; title: string }[] = [
              { dir: 'n', title: '上下拖动调整高度 (底边锁定)' },
              { dir: 's', title: '上下拖动调整高度 (顶边锁定)' },
              { dir: 'w', title: '左右拖动调整宽度 (右边锁定)' },
              { dir: 'e', title: '左右拖动调整宽度 (左边锁定)' },
              { dir: 'nw', title: '拖动对角调整大小 (右下锁定)' },
              { dir: 'ne', title: '拖动对角调整大小 (左下锁定)' },
              { dir: 'sw', title: '拖动对角调整大小 (右上锁定)' },
              { dir: 'se', title: '拖动对角调整大小 (左上锁定)' },
            ]

            directions.forEach(({ dir, title }) => {
              const handle = document.createElement('div')
              handle.className = `modal-resize-handle modal-resize-${dir}`
              handle.title = title
              if (dir === 'se') {
                handle.innerHTML = `
                  <div class="custom-modal-resizer-icon">
                    <svg width="11" height="11" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                      <path d="M11 1L1 11M11 6L6 11M11 11L11 11" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
                    </svg>
                  </div>
                `
              }
              handle.addEventListener('mousedown', (e) => handleResizeStart(dir, e))
              card.appendChild(handle)
            })
          }
        }, 100)
      })
    }
    else {
      resetPositionAndSize()
    }
  },
  { immediate: true },
)

onBeforeUnmount(() => {
  document.removeEventListener('mousemove', onDragMouseMove)
  document.removeEventListener('mouseup', onDragMouseUp)
  document.removeEventListener('mousemove', onResizeMouseMove)
  document.removeEventListener('mouseup', onResizeMouseUp)
})
</script>

<template>
  <NModal
    v-model:show="showModal"
    preset="card"
    :size="size"
    v-bind="bindAttrs"
    style="border-radius: 1.1rem; min-width: 320px; min-height: 300px;"
    :style="$parent"
    :title="title"
  >
    <template #cover>
      <slot name="cover" />
    </template>
    <template #header>
      <slot name="header" />
    </template>
    <template #header-extra>
      <slot name="header-extra" />
    </template>
    <template #footer>
      <slot name="footer" />
    </template>
    <template #action>
      <slot name="action" />
    </template>
    <slot />
  </NModal>
</template>

<style>
/* 1. 最外层弹窗容器：具备显式基准尺寸，与内部内容完全解耦，杜绝内容伸缩导致的抖动 */
.draggable-resizable-modal {
  width: 760px;
  height: 560px;
  min-width: 420px !important;
  min-height: 360px !important;
  max-width: 90vw !important;
  max-height: 85vh !important;
  display: flex !important;
  flex-direction: column !important;
  box-sizing: border-box !important;
  backdrop-filter: blur(20px) saturate(180%) !important;
  -webkit-backdrop-filter: blur(20px) saturate(180%) !important;
  background-color: rgba(255, 255, 255, 0.94) !important;
  background-size: cover !important;
  background-repeat: no-repeat !important;
  border: 1px solid rgba(255, 255, 255, 0.4) !important;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.15), 0 0 1px rgba(0, 0, 0, 0.1) !important;
  overflow: hidden !important;
}

.dark .draggable-resizable-modal {
  background-color: rgba(24, 24, 28, 0.92) !important;
  border: 1px solid rgba(255, 255, 255, 0.08) !important;
  box-shadow: 0 16px 36px rgba(0, 0, 0, 0.4) !important;
}

/* 2. 顶部 Header / 底部 Footer 容器锁定固定高度，防止被压缩 */
.draggable-resizable-modal > .n-card-header,
.draggable-resizable-modal > .n-card__footer,
.draggable-resizable-modal > .n-card__action {
  flex-shrink: 0 !important;
}

/* 弹窗头部支持折行响应式，避免标题与操作区/标签碰撞挤压 */
.draggable-resizable-modal > .n-card-header {
  flex-wrap: wrap !important;
  gap: 8px !important;
}

.draggable-resizable-modal > .n-card-header > .n-card-header__main {
  min-width: 0 !important;
  word-break: break-word !important;
  overflow-wrap: anywhere !important;
}

/* 3. 中间内容容器移除写死固定高度，改为 flex: 1; min-height: 0; 吸收全部垂直空间，内部独立滚动 */
.draggable-resizable-modal > .n-card__content {
  flex: 1 1 0% !important;
  min-height: 0 !important;
  height: auto !important;
  display: flex !important;
  flex-direction: column !important;
  overflow-y: auto !important;
  overflow-x: hidden !important;
  padding: 0.5rem !important;
  box-sizing: border-box !important;
}

/* 4. 清理继承约束：强制直属子内容容器自适应撑满父级剩余空间（使用 > 严格限定直接子级，禁止污染深层嵌套 NCard） */
.draggable-resizable-modal > .n-card__content > * {
  flex: 1 1 0% !important;
  min-height: 0 !important;
  height: auto !important;
  display: flex !important;
  flex-direction: column !important;
  box-sizing: border-box !important;
}

/* 头部最大化按钮 */
.custom-maximize-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  margin-right: 8px;
  border-radius: 6px;
  color: #666;
  cursor: pointer;
  transition: all 0.15s ease;
}

.custom-maximize-btn:hover {
  background-color: rgba(0, 0, 0, 0.06);
  color: #18a058;
}

.dark .custom-maximize-btn {
  color: #aaa;
}

.dark .custom-maximize-btn:hover {
  background-color: rgba(255, 255, 255, 0.1);
  color: #63e2b7;
}

/* 8 方向自由缩放手柄 (原生桌面级交互) */
.modal-resize-handle {
  position: absolute;
  box-sizing: border-box;
  user-select: none;
  z-index: 100;
  touch-action: none;
}

/* 4 条主边框 handle (感应区域 8px，居中透明，不干扰内部交互) */
.modal-resize-n {
  top: 0;
  left: 14px;
  right: 14px;
  height: 8px;
  cursor: ns-resize;
}

.modal-resize-s {
  bottom: 0;
  left: 14px;
  right: 14px;
  height: 8px;
  cursor: ns-resize;
}

.modal-resize-w {
  left: 0;
  top: 14px;
  bottom: 14px;
  width: 8px;
  cursor: ew-resize;
}

.modal-resize-e {
  right: 0;
  top: 14px;
  bottom: 14px;
  width: 8px;
  cursor: ew-resize;
}

/* 4 个对角 handle (16px x 16px，更高层级，对准四角) */
.modal-resize-nw {
  top: 0;
  left: 0;
  width: 16px;
  height: 16px;
  cursor: nwse-resize;
  z-index: 102;
}

.modal-resize-ne {
  top: 0;
  right: 0;
  width: 16px;
  height: 16px;
  cursor: nesw-resize;
  z-index: 102;
}

.modal-resize-sw {
  bottom: 0;
  left: 0;
  width: 16px;
  height: 16px;
  cursor: nesw-resize;
  z-index: 102;
}

.modal-resize-se {
  bottom: 0;
  right: 0;
  width: 18px;
  height: 18px;
  cursor: nwse-resize;
  z-index: 102;
  display: flex;
  align-items: flex-end;
  justify-content: flex-end;
}

/* 右下角视觉图标 */
.custom-modal-resizer-icon {
  width: 14px;
  height: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0.45;
  transition: opacity 0.2s, color 0.2s;
  color: #888;
  pointer-events: none;
  margin-right: 3px;
  margin-bottom: 3px;
}

.modal-resize-se:hover .custom-modal-resizer-icon {
  opacity: 1;
  color: #18a058;
}

/* 弹窗最大化时隐藏所有手柄 */
.is-maximized .modal-resize-handle {
  display: none !important;
}
</style>
