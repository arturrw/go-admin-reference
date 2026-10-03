const usd0 = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', maximumFractionDigits: 0 })
const usd2 = new Intl.NumberFormat('en-US', { style: 'currency', currency: 'USD', minimumFractionDigits: 2 })

/** Formats integer cents as dollars. */
export function money(cents: number, decimals: 0 | 2 = 0) {
  return (decimals ? usd2 : usd0).format(cents / 100)
}

export function compact(n: number) {
  if (Math.abs(n) >= 1e6) return (n / 1e6).toFixed(2) + 'M'
  if (Math.abs(n) >= 1e3) return (n / 1e3).toFixed(1) + 'k'
  return String(Math.round(n))
}

export const int = (n: number) => Math.round(n).toLocaleString('en-US')

export function timeAgo(iso: string | null | undefined, now = Date.now()) {
  if (!iso) return '—'
  const s = Math.max(0, (now - new Date(iso).getTime()) / 1000)
  if (s < 60) return 'just now'
  if (s < 3600) return `${Math.floor(s / 60)}m ago`
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`
  if (s < 86400 * 2) return 'yesterday'
  if (s < 86400 * 14) return `${Math.floor(s / 86400)}d ago`
  return `${Math.floor(s / 604800)}w ago`
}

export function duration(seconds: number) {
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d) return `${d}d ${String(h).padStart(2, '0')}h`
  if (h) return `${h}h ${String(m).padStart(2, '0')}m`
  return `${m}m ${Math.floor(seconds % 60)}s`
}

export const shortDate = (iso: string) =>
  new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric' })

export const monthYear = (iso: string) =>
  new Date(iso).toLocaleDateString('en-US', { month: 'short', year: 'numeric' })

export const initials = (name: string) =>
  name.split(/\s+/).map((p) => p[0]).join('').slice(0, 2).toUpperCase()

/** Stable hue derived from a string, used for avatars. */
export const hueOf = (s: string) => [...s].reduce((h, ch) => (h * 31 + ch.charCodeAt(0)) % 360, 7)

export const capitalize = (s: string) => s.charAt(0).toUpperCase() + s.slice(1)
