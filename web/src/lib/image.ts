// Stock photos come from Unsplash's image CDN, which resizes on request. A
// 36px thumbnail shouldn't download an 800px photo, so ask for the size it is
// drawn at (2× for high-density screens), rounded up to a few cacheable steps.
const STEPS = [64, 128, 256, 480, 800]

export function imageAt(src: string | undefined, cssPx: number): string | undefined {
  if (!src || !src.startsWith('https://images.unsplash.com/')) return src
  const want = cssPx * 2
  const px = STEPS.find((s) => s >= want) ?? STEPS[STEPS.length - 1]
  const url = new URL(src)
  url.searchParams.set('w', String(px))
  url.searchParams.set('h', String(px))
  return url.toString()
}
