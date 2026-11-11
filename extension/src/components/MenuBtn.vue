<template>
  <button
    type="button"
    class="news-menu-btn"
    :style="{
      appearance: 'none',
      background: 'transparent',
      border: 'none',
      font: 'inherit',
      textAlign: 'left',
      cursor: 'pointer',
      display: 'flex',
      alignItems: 'center',
      width: '100%',
      padding: '6px 10px',
      borderRadius: '6px',
      fontSize: '13px',
      color: danger ? 'var(--news-error)' : 'inherit'
    }"
    @click.stop="emit('click')"
  >
    {{ label }}
  </button>
</template>

<script setup lang="ts">
defineProps<{ label: string; danger?: boolean }>()

const emit = defineEmits<{ click: [] }>()
</script>
