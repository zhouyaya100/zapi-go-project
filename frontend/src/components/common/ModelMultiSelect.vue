<template>
  <el-select v-model="selected" multiple filterable allow-create default-first-option :placeholder="placeholder" :disabled="disabled" style="width: 100%" @change="$emit('update:modelValue', selected.join(','))">
    <el-option v-for="m in allModels" :key="m" :label="m" :value="m" />
  </el-select>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'

const props = defineProps<{ modelValue: string; allModels: string[]; placeholder?: string; disabled?: boolean }>()
defineEmits(['update:modelValue'])
const selected = ref<string[]>(props.modelValue ? props.modelValue.split(',').filter(Boolean) : [])
watch(() => props.modelValue, (v) => { selected.value = v ? v.split(',').filter(Boolean) : [] })
</script>
