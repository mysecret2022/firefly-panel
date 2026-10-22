<script setup lang="ts">
import { NDivider, NGradientText, NTag } from 'naive-ui'
import srcSvglogo from '@/assets/about_logo.png'

const frontVersion = '1.0.13'
</script>

<template>
  <div class="h-full flex flex-col items-center justify-center overflow-y-auto pt-6 px-4">
    <!-- 头部 Logo 与应用名称 -->
    <div class="flex flex-col items-center justify-center">
      <img :src="srcSvglogo" width="80" height="80" alt="Logo">
      <div class="text-3xl font-bold mt-2">
        Firefly-panel
      </div>
      <div class="text-base mt-1">
        <NGradientText type="info" class="font-semibold">
          v{{ frontVersion }}
        </NGradientText>
      </div>
    </div>

    <NDivider style="margin: 20px 0" />

    <!-- 作者与版权信息 -->
    <div class="flex flex-col items-center justify-center text-base space-y-3">
      <div class="text-gray-700 dark:text-gray-300">
        <span class="font-semibold">原作者：</span>
        <a href="https://github.com/hslr-s" target="_blank" class="link">红烧猎人</a>
      </div>
      <div class="text-gray-700 dark:text-gray-300">
        <span class="font-semibold">二次开发作者：</span>
        <span class="font-bold text-green-600 dark:text-green-400">Roland</span>
      </div>
      <div class="mt-2 text-sm text-gray-400">
        基于 hslr-s/sun-panel 开源二次开发，集成整站备份、多云漫游图床与图库管理能力
      </div>

      <div class="mt-4">
        <a href="https://github.com/mysecret2022/my-sun-panel" target="_blank" class="link font-medium">
          GitHub 仓库地址: mysecret2022/my-sun-panel
        </a>
      </div>

      <div class="mt-4">
        <NTag :bordered="false" size="small" type="success">
          Firefly-panel v1.0.13
        </NTag>
      </div>
    </div>
  </div>
</template>

<style scoped>
.link {
  color: rgb(0, 102, 255);
  text-decoration: none;
}
.link:hover {
  text-decoration: underline;
}
</style>
