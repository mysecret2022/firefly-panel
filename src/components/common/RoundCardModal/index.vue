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

let isDragging = false
let dragStartX = 0
let dragStartY = 0
let currentTranslateX = 0
let currentTranslateY = 0

let isResizing = false
let resizeStartX = 0
let resizeStartY = 0
let initialWidth = 0
let initialHeight = 0

function getCardEl(): HTMLElement | null {
  return document.querySelector(`.${modalUniqueClass}`) as HTMLElement | null
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

  isDragging = true
  dragStartX = e.clientX - currentTranslateX
  dragStartY = e.clientY - currentTranslateY
  card.style.transition = 'none'

  document.addEventListener('mousemove', onDragMouseMove)
  document.addEventListener('mouseup', onDragMouseUp)
  e.preventDefault()
}

function onDragMouseMove(e: MouseEvent) {
  if (!isDragging)
    return
  currentTranslateX = e.clientX - dragStartX
  currentTranslateY = e.clientY - dragStartY
  const card = getCardEl()
  if (card)
    card.style.transform = `translate(${currentTranslateX}px, ${currentTranslateY}px)`
}

function onDragMouseUp() {
  isDragging = false
  document.removeEventListener('mousemove', onDragMouseMove)
  document.removeEventListener('mouseup', onDragMouseUp)
}

function handleResizeMouseDown(e: MouseEvent) {
  if (props.resizable === false)
    return
  const card = getCardEl()
  if (!card)
    return

  isResizing = true
  resizeStartX = e.clientX
  resizeStartY = e.clientY
  initialWidth = card.offsetWidth
  initialHeight = card.offsetHeight
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

  const deltaX = e.clientX - resizeStartX
  const deltaY = e.clientY - resizeStartY

  const newWidth = Math.max(320, Math.min(window.innerWidth - 30, initialWidth + deltaX))
  const newHeight = Math.max(200, Math.min(window.innerHeight - 30, initialHeight + deltaY))

  card.style.width = `${newWidth}px`
  card.style.maxWidth = '98vw'
  card.style.height = `${newHeight}px`
  card.style.maxHeight = '95vh'
}

function onResizeMouseUp() {
  isResizing = false
  document.removeEventListener('mousemove', onResizeMouseMove)
  document.removeEventListener('mouseup', onResizeMouseUp)
}

function resetPositionAndSize() {
  const card = getCardEl()
  if (card) {
    currentTranslateX = 0
    currentTranslateY = 0
    card.style.transform = ''
    card.style.width = ''
    card.style.height = ''
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
            header.title = '按住拖动窗口，双击复位'
            header.addEventListener('mousedown', handleHeaderMouseDown)
            header.addEventListener('dblclick', resetPositionAndSize)
          }

          // 添加右下角拉伸缩放手柄
          if (!card.querySelector('.custom-modal-resizer')) {
            const resizer = document.createElement('div')
            resizer.className = 'custom-modal-resizer'
            resizer.title = '拖拽调整窗口大小'
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
  position: relative !important;
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
  right: 5px;
  bottom: 5px;
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
