// White is the default; the rest are offered in Settings → Appearance.
export const ACCENTS = ['#F5F5F4', '#C6F432', '#4FE3F0', '#FF8A3D', '#FF5FA2', '#A78BFA'] as const

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

// Density: "compact" tightens card and table padding (see index.css). It is
// personal, like the accent, and remembered in this browser.
export type Density = 'comfortable' | 'compact'

const DENSITY_KEY = 'goadmin.density'

export function getDensity(): Density {
  try {
    return localStorage.getItem(DENSITY_KEY) === 'compact' ? 'compact' : 'comfortable'
  } catch {
    return 'comfortable'
  }
}

export function setDensity(d: Density) {
  document.documentElement.dataset.density = d
  try {
    localStorage.setItem(DENSITY_KEY, d)
  } catch {
    // Storage can be unavailable; the change still applies for this session.
  }
}

export function restoreDensity() {
  document.documentElement.dataset.density = getDensity()
}
