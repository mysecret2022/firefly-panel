<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { t } from '@/locales'

// 增加 color 属性支持
const props = withDefaults(defineProps<{
  hideSecond?: boolean
  color?: string
}>(), {
  color: '#ffffff',
})

interface CurrentDate {
  time: string
  date: string
  week: string
}

const currentDate = ref<CurrentDate>({
  time: '--:--',
  date: '------',
  week: '--',
})

function updateCurrentDate() {
  const now = new Date()
  const hours = String(now.getHours()).padStart(2, '0')
  const minutes = String(now.getMinutes()).padStart(2, '0')

  if (!props.hideSecond) {
    const seconds = String(now.getSeconds()).padStart(2, '0')
    currentDate.value.time = `${hours}:${minutes}:${seconds}`
  }
  else {
    currentDate.value.time = `${hours}:${minutes}`
  }

  const day = now.getDate()
  const month = now.getMonth() + 1

  const daysOfWeek = [
    t('deskModule.clock.sun'),
    t('deskModule.clock.mon'),
    t('deskModule.clock.tue'),
    t('deskModule.clock.wed'),
    t('deskModule.clock.thu'),
    t('deskModule.clock.fri'),
    t('deskModule.clock.sat'),
  ]
  currentDate.value.week = daysOfWeek[now.getDay()]
  currentDate.value.date = `${month}-${day}`
}

const updateClock = () => {
  updateCurrentDate()
}

const intervalId = setInterval(updateClock, 1000)

onMounted(() => {
  updateClock()
  updateCurrentDate()
})

onBeforeUnmount(() => {
  clearInterval(intervalId)
})
</script>

<template>
  <!-- 显式绑定与主标题一致的颜色与立体文字投影 -->
  <div 
    class="clock w-full text-center select-none"
    :style="{ 
      color: color || '#ffffff',
      textShadow: '0 2px 12px rgba(0,0,0,0.45)' 
    }"
  >
    <span class="clock-time text-2xl sm:text-2xl md:text-3xl font-[600] tracking-wider leading-none">
      {{ currentDate.time }}
    </span>
    <div class="hidden sm:hidden md:block text-xs opacity-85 mt-1 leading-none font-medium">
      <span class="clock-date mr-1.5">
        {{ currentDate.date }}
      </span>
      <span class="clock-week">
        {{ currentDate.week }}
      </span>
    </div>
  </div>
</template>
