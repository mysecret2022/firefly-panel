<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, useAttrs, watch } from 'vue'
import { NModal } from 'naive-ui'

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

const bindAttrs = computed<{ class: string; style: string }>(() => ({
  class: `${(attrs.class as string) || ''} ${modalUniqueClass} draggable-resizable-modal`,
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
  if (card.style.position !== 'fixed') {
    const rect = card.getBoundingClientRect()
    card.style.position = 'fixed'
    card.style.left = `${rect.left}px`
    card.style.top = `${rect.top}px`
    card.style.margin = '0'
    card.style.transform = 'none'
    card.style.width = `${rect.width}px`
    card.style.height = `${rect.height}px`
  }
}

function handleHeaderMouseDown(e: MouseEvent) {
  if (props.draggable === false)
    return
  const target = e.target as HTMLElement
  if (target.closest('.n-base-close, button, a, input, select, textarea, .n-dropdown-trigger'))
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
  if (!isDragging)
    return
  const card = getCardEl()
  if (!card)
    return

  const deltaX = e.clientX - dragMouseStartX
  const deltaY = e.clientY - dragMouseStartY

  const newLeft = Math.max(10, Math.min(window.innerWidth - 60, cardStartLeft + deltaX))
  const newTop = Math.max(10, Math.min(window.innerHeight - 60, cardStartTop + deltaY))

  card.style.left = `${newLeft}px`
  card.style.top = `${newTop}px`
}

function onDragMouseUp() {
  isDragging = false
  document.removeEventListener('mousemove', onDragMouseMove)
  document.removeEventListener('mouseup', onDragMouseUp)
}

// 右下角拖动变形：左边和上边锁定不动，向右和向下拖动改变宽高
function handleResizeMouseDown(e: MouseEvent) {
  if (props.resizable === false)
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
  if (!isResizing)
    return
  const card = getCardEl()
  if (!card)
    return

  // 左上角严格固定在 (fixedLeft, fixedTop)，只根据鼠标当前坐标计算新的 width 和 height
  const newWidth = Math.max(320, Math.min(window.innerWidth - fixedLeft - 10, e.clientX - fixedLeft))
  const newHeight = Math.max(200, Math.min(window.innerHeight - fixedTop - 10, e.clientY - fixedTop))

  card.style.width = `${newWidth}px`
  card.style.height = `${newHeight}px`
  card.style.maxWidth = '98vw'
  card.style.maxHeight = '98vh'
}

function onResizeMouseUp() {
  isResizing = false
  document.removeEventListener('mousemove', onResizeMouseMove)
  document.removeEventListener('mouseup', onResizeMouseUp)
}

// 双击标题栏一键复位
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

          // 绑定 Header 拖拽
          const header = card.querySelector('.n-card-header') as HTMLElement
          if (header && !header.dataset.dragAttached) {
            header.dataset.dragAttached = 'true'
            header.style.cursor = 'move'
            header.style.userSelect = 'none'
            header.title = '按住拖动窗口，双击居中复位'
            header.addEventListener('mousedown', handleHeaderMouseDown)
            header.addEventListener('dblclick', resetPositionAndSize)
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
    style="border-radius: 1rem;"
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
.draggable-resizable-modal {
  display: flex !important;
  flex-direction: column !important;
  box-sizing: border-box;
}

.draggable-resizable-modal .n-card__content {
  flex: 1 1 auto;
  overflow: auto;
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
