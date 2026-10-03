export const ACCENTS = ['#C6F432', '#4FE3F0', '#FF8A3D', '#FF5FA2', '#A78BFA', '#F5F5F4'] as const

const KEY = 'goadmin.accent'
const DEFAULT = ACCENTS[0]

export function getAccent(): string {
  try {
    return localStorage.getItem(KEY) ?? DEFAULT
  } catch {
    return DEFAULT
  }
}

export function setAccent(color: string) {
  document.documentElement.style.setProperty('--color-accent', color)
  try {
    localStorage.setItem(KEY, color)
  } catch {
    // Storage can be unavailable (private mode); the change still applies for this session.
  }
}

/** Re-applies a previously chosen accent on boot. */
export function restoreAccent() {
  const c = getAccent()
  if (c !== DEFAULT) document.documentElement.style.setProperty('--color-accent', c)
}
