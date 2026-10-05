// Regenerates the README screenshots in docs/screenshots: builds the server,
// runs it on the in-memory demo data and captures full pages (and the open
// sheets) at desktop and phone sizes. Run `npm run screenshots` after
// `npm run build`.
import { execFileSync, spawn } from 'node:child_process'
import { mkdirSync, rmSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'
import { chromium } from '@playwright/test'

const root = resolve(import.meta.dirname, '../..')
const out = join(root, 'docs/screenshots')
const port = 8098
const base = `http://localhost:${port}`
const bin = join(tmpdir(), `goadmin-shots${process.platform === 'win32' ? '.exe' : ''}`)

execFileSync('go', ['build', '-o', bin, './cmd/server'], { cwd: root, stdio: 'inherit' })
const server = spawn(bin, [], {
  cwd: root,
  env: { ...process.env, ADDR: `:${port}`, DATABASE_URL: '', LOG_LEVEL: 'warn', UPLOAD_DIR: join(tmpdir(), 'goadmin-shots-uploads') },
  stdio: 'inherit',
})

const sleep = (ms) => new Promise((r) => setTimeout(r, ms))
for (let i = 0; ; i++) {
  try {
    if ((await fetch(`${base}/healthz`)).ok) break
  } catch {}
  if (i > 100) throw new Error('server did not start')
  await sleep(200)
}

mkdirSync(out, { recursive: true })
const browser = await chromium.launch({ channel: process.env.CI ? undefined : 'chrome' })

async function session(viewport, isMobile = false) {
  const ctx = await browser.newContext({ viewport, deviceScaleFactor: 1, isMobile, hasTouch: isMobile, colorScheme: 'dark' })
  const res = await ctx.request.post(`${base}/api/v1/auth/login`, { data: { email: 'artur@acme.io', password: 'goadmin' } })
  if (!res.ok()) throw new Error(`login: ${res.status()}`)
  return ctx.newPage()
}

/** Loads a page and waits until data, images and fonts have settled. */
async function open(page, path) {
  await page.goto(base + path)
  await page.waitForLoadState('networkidle').catch(() => {})
  await page.locator('main h1').waitFor()
  await page.evaluate(() => document.fonts.ready)
  await page.waitForFunction(() => !document.querySelector('.animate-pulse'))
  // Lazy images below the fold load once scrolled into view.
  await page.evaluate(async () => {
    for (let y = 0; y < document.body.scrollHeight; y += 600) {
      window.scrollTo(0, y)
      await new Promise((r) => setTimeout(r, 60))
    }
    window.scrollTo(0, 0)
  })
  await page.waitForLoadState('networkidle').catch(() => {})
  await sleep(400)
}

/**
 * Full pages are taken with the window stretched to the page height, so the
 * sticky sidebar and header render once at full length instead of being cut off.
 */
async function shot(page, name, fullPage = true) {
  const size = page.viewportSize()
  if (fullPage) {
    const height = await page.evaluate(() => document.documentElement.scrollHeight)
    await page.setViewportSize({ width: size.width, height: Math.max(height, size.height) })
    await sleep(300)
  }
  await page.screenshot({ path: join(out, `${name}.png`) })
  await page.setViewportSize(size)
  console.log('saved', name)
}

const desktop = await session({ width: 1440, height: 900 })
for (const [name, path] of [
  ['dashboard', '/'],
  ['products', '/products'],
  ['orders', '/orders'],
  ['customers', '/customers'],
  ['activity', '/activity'],
  ['team', '/team'],
  ['requests', '/requests'],
]) {
  await open(desktop, path)
  await shot(desktop, name)
}

// Sheets are fixed to the viewport, so they are captured as the screen shows them.
await open(desktop, '/customers?view=25')
await desktop.getByRole('dialog').getByText('Total spent').waitFor()
await sleep(500)
await shot(desktop, 'customer-sheet', false)

await open(desktop, '/')
await desktop.getByRole('button', { name: 'Orders: open details' }).click()
await desktop.getByRole('dialog').locator('tbody tr').nth(1).click()
await desktop.getByText('Orders placed this day').waitFor()
await sleep(800)
await shot(desktop, 'kpi-day', false)

await open(desktop, '/team')
await desktop.getByRole('button', { name: 'Open Diego Vega' }).click()
await desktop.getByRole('dialog').getByRole('tab', { name: /Access/ }).click()
await desktop.getByTestId('access').waitFor()
await sleep(500)
await shot(desktop, 'member-access', false)

const phone = await session({ width: 390, height: 844 }, true)
for (const [name, path] of [
  ['mobile-dashboard', '/'],
  ['mobile-orders', '/orders'],
]) {
  await open(phone, path)
  await shot(phone, name)
}

await browser.close()
const exited = new Promise((r) => server.once('exit', r))
server.kill()
await exited
rmSync(bin, { force: true })
