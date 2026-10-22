<script setup lang="ts">
import { computed } from 'vue'
import { NEllipsis } from 'naive-ui'
import { ItemIcon } from '@/components/common'
import { PanelPanelConfigStyleEnum } from '@/enums'
import { usePanelState } from '@/store'

interface Prop {
  itemInfo?: Panel.ItemInfo
  size?: number
  forceBackground?: string
  iconTextColor?: string
  iconTextInfoHideDescription?: boolean
  iconTextIconHideTitle?: boolean
  style?: PanelPanelConfigStyleEnum | number
}

const props = withDefaults(defineProps<Prop>(), {
  size: 70,
  style: 0,
})

const defaultBackground = '#2a2a2a6b'
const panelState = usePanelState()

// 判断当前是否应该渲染为【长方形详情卡片】
const isInfoStyle = computed(() => {
  return Number(props.style) === 0 || props.style === PanelPanelConfigStyleEnum.info
})

// 计算卡片背景毛玻璃动态透明度
const cardDynamicStyle = computed(() => {
  const raw = panelState.panelConfig.cardGlassOpacity ?? 45
  const alpha = raw / 100

  // 当设为 0% 时，实现 100% 完全透明
  if (alpha === 0) {
    return {
      background: 'transparent !important',
      backdropFilter: 'none !important',
      WebkitBackdropFilter: 'none !important',
      borderColor: 'rgba(255, 255, 255, 0.08) !important',
      boxShadow: 'none !important',
    }
  }

  // 动态透明度背景与高斯模糊深度
  return {
    background: `rgba(30, 26, 68, ${alpha}) !important`,
    backdropFilter: `blur(${Math.max(6, Math.round(alpha * 26))}px) saturate(180%) !important`,
    WebkitBackdropFilter: `blur(${Math.max(6, Math.round(alpha * 26))}px) saturate(180%) !important`,
    borderColor: `rgba(255, 255, 255, ${Math.min(0.25, alpha * 0.25 + 0.04)}) !important`,
    boxShadow: `0 4px 18px rgba(0, 0, 0, ${alpha * 0.45}), inset 0 1px 1px rgba(255, 255, 255, ${alpha * 0.25}) !important`,
  }
})
</script>

<template>
  <div class="app-icon w-full">
    <!-- 方案 A：长方形横向胶囊毛玻璃卡片 (详情图标风格 1:1 还原) -->
    <div
      v-if="isInfoStyle"
      class="rect-app-card w-full flex items-center px-3 py-2.5 rounded-2xl select-none cursor-pointer"
      :style="cardDynamicStyle"
    >
      <!-- 左侧：应用图标 (46px 圆角方形自适应) -->
      <div class="rect-app-icon-wrap w-[46px] h-[46px] shrink-0 flex items-center justify-center rounded-xl overflow-hidden">
        <ItemIcon 
          :item-icon="itemInfo?.icon" 
          :size="46" 
          class="flex items-center justify-center"
        />
      </div>

      <!-- 右侧：文字信息区 (标题 + 描述双层排布) -->
      <div class="flex-1 min-w-0 ml-3 flex flex-col justify-center">
        <!-- 第一行：主标题 -->
        <div class="text-sm font-bold text-white tracking-wide truncate leading-tight">
          <NEllipsis :line-clamp="1">
            {{ itemInfo?.title }}
          </NEllipsis>
        </div>

        <!-- 第二行：副标题 / 描述 -->
        <div 
          v-if="!iconTextInfoHideDescription && itemInfo?.description" 
          class="text-xs text-white/70 dark:text-slate-300/80 truncate mt-1 leading-none font-normal"
        >
          <NEllipsis :line-clamp="1">
            {{ itemInfo?.description }}
          </NEllipsis>
        </div>
      </div>
    </div>

    <!-- 方案 B：传统极简小图标风格 (降级备用) -->
    <div v-else class="app-icon-small">
      <div
        class="app-icon-small-icon overflow-hidden rounded-2xl sunpanel w-[70px] h-[70px] mx-auto transition-all duration-200 hover:shadow-[0_0_20px_10px_rgba(0,0,0,0.2)]"
        :title="itemInfo?.description"
      >
        <ItemIcon :item-icon="itemInfo?.icon" />
      </div>
      <div
        v-if="!iconTextIconHideTitle"
        class="app-icon-small-title text-center app-icon-text-shadow cursor-pointer mt-[2px]"
        :style="{ color: iconTextColor }"
      >
        <span>{{ itemInfo?.title }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
/* 1. 长方形毛玻璃卡片核心基础样式 */
.rect-app-card {
  min-height: 64px;
  position: relative;
  overflow: hidden;
  background: rgba(30, 26, 68, 0.45) !important;
  backdrop-filter: blur(20px) saturate(180%) !important;
  -webkit-backdrop-filter: blur(20px) saturate(180%) !important;
  border: 1px solid rgba(255, 255, 255, 0.14) !important;
  box-shadow: 0 4px 18px rgba(0, 0, 0, 0.25), inset 0 1px 1px rgba(255, 255, 255, 0.15) !important;
  transition: all 0.32s cubic-bezier(0.34, 1.56, 0.64, 1) !important;
}

/* 2. 极光玻璃高光扫光流层 */
.rect-app-card::after {
  content: '';
  position: absolute;
  top: -50%;
  left: -120%;
  width: 60%;
  height: 200%;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 255, 255, 0.16) 45%,
    rgba(110, 231, 183, 0.22) 50%,
    transparent 100%
  );
  transform: rotate(25deg);
  pointer-events: none;
  transition: none;
}

/* 3. 鼠标悬停动效：微悬浮 + 极光扫光 */
.rect-app-card:hover {
  transform: translateY(-4px) scale(1.018);
  background: rgba(45, 38, 95, 0.62) !important;
  border-color: rgba(255, 255, 255, 0.32) !important;
  box-shadow: 
    0 12px 30px -4px rgba(0, 0, 0, 0.45),
    0 0 22px rgba(99, 242, 201, 0.25),
    inset 0 1px 1.5px rgba(255, 255, 255, 0.35) !important;
}

.rect-app-card:hover::after {
  left: 160%;
  transition: all 0.75s ease-out;
}

/* 4. 左侧图标微交互 */
.rect-app-icon-wrap {
  transition: all 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.rect-app-card:hover .rect-app-icon-wrap {
  transform: scale(1.08) rotate(1deg);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.35);
}

.rect-app-card:active {
  transform: translateY(-1px) scale(0.99);
  transition-duration: 0.1s !important;
}

/* 浅色日间模式自适应 */
html:not(.dark) .rect-app-card {
  background: rgba(255, 255, 255, 0.42) !important;
  border: 1px solid rgba(255, 255, 255, 0.5) !important;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08), inset 0 1px 1px rgba(255, 255, 255, 0.7) !important;
}
html:not(.dark) .rect-app-card:hover {
  background: rgba(255, 255, 255, 0.68) !important;
}
</style>
