import type { Component } from 'vue'

import type { Selection } from '../api'

export interface NavEntry {
  key: string
  kind: 'all' | 'starred' | 'podcasts' | 'search' | 'folder' | 'feed'
  label: string
  icon: Component
  sel: Selection
  unread: number
  error?: string
  depth?: number
  favicon?: string
  searchQuery?: string
  searchId?: number
}
