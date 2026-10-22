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

// 拉伸缩放状态 (左上角完全固定，右下角拖动变形)
let isResizing = false
let fixedLeft = 0
let fixedTop = 0

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
    card.style.width = `${Math.max(580, rect.width)}px`
    card.style.height = `${Math.max(480, rect.height)}px`
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

// 右下角拖动变形：左边和上边锁定不动，向右和向下拖动改变宽高 (保证最小 580x480 防挤压)
function handleResizeMouseDown(e: MouseEvent) {
  if (props.resizable === false || isMaximized.value)
    return
  const card = getCardEl()
  if (!card)
    return

  ensureFixedPosition(card)

  isResizing = true
  const rect = card.getBoundingClientRect()
  fixedLeft = rect.left
  fixedTop = rect.top
  card.style.transition = 'none'

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

  // 左上角严格固定在 (fixedLeft, fixedTop)，最小限制 580x480
  const newWidth = Math.max(580, Math.min(window.innerWidth - fixedLeft - 10, e.clientX - fixedLeft))
  const newHeight = Math.max(480, Math.min(window.innerHeight - fixedTop - 10, e.clientY - fixedTop))

  card.style.width = `${newWidth}px`
  card.style.height = `${newHeight}px`
  card.style.maxWidth = '99vw'
  card.style.maxHeight = '99vh'
}

function onResizeMouseUp() {
  isResizing = false
  document.removeEventListener('mousemove', onResizeMouseMove)
  document.removeEventListener('mouseup', onResizeMouseUp)
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

          // 添加右下角拉伸缩放手柄
          if (!card.querySelector('.custom-modal-resizer')) {
            const resizer = document.createElement('div')
            resizer.className = 'custom-modal-resizer'
            resizer.title = '向右下拖拽调整窗口大小 (左上固定)'
            resizer.innerHTML = `
              <svg width="12" height="12" viewBox="0 0 12 12" fill="none" xmlns="http://www.w3.org/2000/svg">
                <path d="M11 1L1 11M11 6L6 11M11 11L11 11" stroke="currentColor" stroke-width="1.6" stroke-linecap="round"/>
              </svg>
            `
            resizer.addEventListener('mousedown', handleResizeMouseDown)
            card.appendChild(resizer)
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
    style="border-radius: 1.1rem; min-width: 580px; min-height: 480px;"
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
/* 1. 最外层弹窗容器：挂载背景色与 Flexbox 纵向布局 */
.draggable-resizable-modal {
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
.draggable-resizable-modal .n-card-header,
.draggable-resizable-modal .n-card__footer,
.draggable-resizable-modal .n-card__action {
  flex-shrink: 0 !important;
}

/* 3. 中间内容容器移除写死固定高度，改为 flex: 1; min-height: 0; 吸收全部垂直空间，内部独立滚动 */
.draggable-resizable-modal .n-card__content {
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

/* 4. 清理继承约束：强制直属子内容容器自适应撑满父级剩余空间 */
.draggable-resizable-modal .n-card__content > * {
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

.custom-modal-resizer {
  position: absolute;
  right: 4px;
  bottom: 4px;
  width: 18px;
  height: 18px;
  cursor: nwse-resize;
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0.45;
  transition: opacity 0.2s, color 0.2s;
  z-index: 99;
  color: #888;
  user-select: none;
}

.custom-modal-resizer:hover {
  opacity: 1;
  color: #18a058;
}
</style>
