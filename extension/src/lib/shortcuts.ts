// Keyboard shortcuts shown in the help dialog. The labels are translation
// keys, the dialog translates them.
export interface Shortcut {
  keys: string
  label: string
}

export const SHORTCUTS: Shortcut[] = [
  { keys: 'j / n / ↓', label: 'Next article' },
  { keys: 'k / p / ↑', label: 'Previous article' },
  { keys: 'o / Enter', label: 'Open article' },
  { keys: 'm', label: 'Toggle read/unread' },
  { keys: 'f', label: 'Star / unstar' },
  { keys: 'v', label: 'Open original link' },
  { keys: 'g u', label: 'All articles' },
  { keys: 'g b', label: 'Starred' },
  { keys: 'g p', label: 'Podcasts' },
  { keys: 'a', label: 'Mark all read' },
  { keys: 'r', label: 'Refresh feeds' },
  { keys: '/', label: 'Focus search' },
  { keys: '?', label: 'Show this help' },
  { keys: 'Esc', label: 'Close dialogs' }
]
