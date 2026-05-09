<template>
  <el-container style="height: 100vh">
    <el-aside :width="isCollapsed ? '64px' : '220px'" style="transition: width 0.3s ease; overflow: hidden">
      <SideNav :collapsed="isCollapsed" />
    </el-aside>
    <el-container>
      <el-header style="padding: 0; height: 56px">
        <TopBar @toggle="isCollapsed = !isCollapsed" :collapsed="isCollapsed" />
      </el-header>
      <el-main class="main-content">
        <router-view v-slot="{ Component }">
          <transition name="fade-slide" mode="out-in">
            <component :is="Component" />
          </transition>
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import SideNav from './SideNav.vue'
import TopBar from './TopBar.vue'

const isCollapsed = ref(false)
</script>
