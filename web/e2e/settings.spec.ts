import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('create an API key, use it, then revoke it', async ({ page }) => {
  await loginAs(page, 'owner', '/settings')
  const keys = page.getByTestId('api-key')
  await expect(keys).toHaveCount(3)
  await expect(keys.first()).toContainText('ga_live_••••••••')

  await page.getByRole('button', { name: 'Create key' }).click()
  const dialog = page.getByRole('dialog', { name: 'Create API key' })
  await expect(dialog.getByRole('button', { name: 'Create key' })).toBeDisabled() // a name is required
  await dialog.getByLabel('Name').fill('E2E reporting')
  await dialog.getByRole('radio', { name: 'Read', exact: true }).click()
  await dialog.getByRole('button', { name: 'Create key' }).click()

  // The full key is shown once.
  const shown = page.getByRole('dialog', { name: 'Key “E2E reporting” created' })
  const secret = (await shown.getByTestId('api-key-secret').innerText()).trim()
  expect(secret).toMatch(/^ga_live_[0-9a-f]{48}$/)
  await shown.getByRole('button', { name: 'Done' }).click()
  await expect(keys).toHaveCount(4)
  const row = keys.filter({ hasText: 'E2E reporting' })
  await expect(row).toContainText(`${secret.slice(-4)}`)
  await expect(row).not.toContainText(secret)
  await expect(row).toContainText('never used')

  // The key works as a bearer token, and only for reads.
  const auth = { headers: { Authorization: `Bearer ${secret}` } }
  expect((await page.request.get('/api/v1/orders?limit=1', auth)).status()).toBe(200)
  expect((await page.request.get('/api/v1/team', auth)).status()).toBe(403)
  await page.reload()
  await expect(keys.filter({ hasText: 'E2E reporting' })).toContainText('used just now')

  // Revoking asks first, then cuts access at once.
  await keys.filter({ hasText: 'E2E reporting' }).getByRole('button', { name: /Revoke/ }).click()
  const confirm = page.getByRole('dialog', { name: /Revoke “E2E reporting”/ })
  await confirm.getByRole('button', { name: 'Cancel' }).click()
  await expect(keys).toHaveCount(4)
  await keys.filter({ hasText: 'E2E reporting' }).getByRole('button', { name: /Revoke/ }).click()
  await confirm.getByRole('button', { name: 'Revoke key' }).click()
  await expect(page.getByText('Key revoked')).toBeVisible()
  await expect(keys).toHaveCount(3)
  expect((await page.request.get('/api/v1/orders?limit=1', auth)).status()).toBe(401)

  // Both changes are in the activity feed.
  await page.goto('/')
  const feed = page.locator('.card', { has: page.getByRole('heading', { name: 'Activity' }) })
  await expect(feed.getByTestId('activity-item').first()).toContainText('revoked API key “E2E reporting”')
})

test('only owners and admins can manage API keys', async ({ page }) => {
  await loginAs(page, 'viewer', '/settings')
  await expect(page.getByTestId('api-key')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Create key' })).toBeDisabled()
  expect((await page.request.get('/api/v1/settings/api-keys')).status()).toBe(403)
})

test('general settings are saved and used by the UI', async ({ page }) => {
  await loginAs(page, 'owner', '/settings')
  const general = page.locator('section', { has: page.getByRole('heading', { name: 'General' }) })
  await expect(general.getByLabel('Listen address')).toBeDisabled()
  const save = general.getByRole('button', { name: 'Save changes' })
  await expect(save).toBeDisabled() // nothing changed yet

  await general.getByLabel('Service name').fill('Acme Backoffice')
  await general.getByLabel('Public base URL').fill('admin.acme.io')
  await save.click()
  await expect(general.getByText('must be an http(s) address')).toBeVisible()

  await general.getByLabel('Public base URL').fill('https://admin.acme.io/')
  await save.click()
  await expect(page.getByText('Settings saved')).toBeVisible()
  // The sidebar shows the new name, and the field normalises the URL.
  await expect(page.locator('aside')).toContainText('Acme Backoffice')
  await expect(general.getByLabel('Public base URL')).toHaveValue('https://admin.acme.io')

  await page.reload()
  await expect(general.getByLabel('Service name')).toHaveValue('Acme Backoffice')

  // Read-only roles see the name, not the form.
  await loginAs(page, 'viewer', '/settings')
  await expect(general.getByLabel('Service name')).toBeDisabled()
  await expect(general.getByLabel('Public base URL')).toHaveCount(0)
})

test('maintenance mode locks out other roles and shows owners a banner', async ({ page, browser }) => {
  test.setTimeout(90_000) // open tabs notice on the next 15 s poll
  const editor = await browser.newContext()
  const ep = await editor.newPage()
  await loginAs(ep, 'editor', '/products')
  await expect(ep.getByRole('heading', { name: 'Products' })).toBeVisible()

  await loginAs(page, 'owner', '/settings')
  const toggle = page.getByRole('switch', { name: 'Maintenance mode' })
  try {
    await toggle.click()
    await expect(toggle).toHaveAttribute('aria-checked', 'true')
    await expect(page.getByTestId('maintenance-banner')).toBeVisible()

    // The editor's open tab swaps to the holding page on its next poll.
    await expect(ep.getByTestId('maintenance-page')).toBeVisible({ timeout: 25_000 })
    await expect(ep.getByRole('heading', { name: 'Down for maintenance' })).toBeVisible()
    expect((await ep.request.get('/api/v1/orders')).status()).toBe(503)

    // The owner keeps working.
    await page.goto('/orders')
    await expect(page.getByRole('heading', { name: 'Orders' })).toBeVisible()
    await expect(page.getByTestId('maintenance-banner')).toBeVisible()
  } finally {
    await page.request.patch('/api/v1/settings', { data: { maintenance: false } })
  }
  await expect(page.getByTestId('maintenance-banner')).toBeHidden({ timeout: 25_000 })
  await ep.reload()
  await expect(ep.getByRole('heading', { name: 'Products' })).toBeVisible()
  await editor.close()
})

test('security settings: session lifetime, audit log and login alerts take effect', async ({ page }) => {
  await loginAs(page, 'owner', '/settings')
  const security = page.locator('section', { has: page.getByRole('heading', { name: 'Security' }) })

  // 2FA isn't implemented, and says so instead of pretending.
  await expect(security.getByText('planned')).toBeVisible()
  await expect(security.getByRole('switch', { name: 'Require 2FA' })).toBeDisabled()

  // The lifetime is stored and shown back after a reload.
  const lifetime = security.getByLabel('Session lifetime')
  await lifetime.selectOption({ label: '8 hours' })
  await expect(page.getByText('Settings saved')).toBeVisible()
  await page.reload()
  await expect(lifetime).toHaveValue(String(8 * 3600))

  // With the audit log off, login alerts can't work and are switched off with it.
  const audit = security.getByRole('switch', { name: 'Audit log' })
  const alerts = security.getByRole('switch', { name: 'Login alerts' })
  await audit.click()
  await expect(audit).toHaveAttribute('aria-checked', 'false')
  await expect(alerts).toBeDisabled()
  await audit.click()
  await expect(audit).toHaveAttribute('aria-checked', 'true')
  await expect(alerts).toBeEnabled()

  // Both switches are in the activity log.
  await page.goto('/activity')
  await page.getByLabel('Type').selectOption({ label: 'Settings' })
  await expect(page.getByTestId('activity-item').first()).toContainText('turned the audit log on')
  await expect(page.getByTestId('activity-item').nth(1)).toContainText('turned the audit log off')
})

test('a sign-in from a new device raises an alert in the activity log', async ({ page, browser }) => {
  // Jon (viewer) has signed in before in the demo data; Firefox on macOS is new for him.
  const jon = await browser.newContext({ userAgent: 'Mozilla/5.0 (Macintosh; Intel Mac OS X 14.5; rv:127.0) Gecko/20100101 Firefox/127.0' })
  const res = await jon.request.post('/api/v1/auth/login', { data: { email: 'jon@acme.io', password: 'goadmin' } })
  expect(res.ok()).toBeTruthy()
  await jon.close()

  await loginAs(page, 'owner', '/activity')
  await page.getByLabel('Type').selectOption({ label: 'Security alerts' })
  const alert = page.getByTestId('activity-item').first()
  await expect(alert).toContainText('New sign-in to Jon Berg')
  await expect(alert).toContainText('Firefox on macOS')
})

test('compact density tightens tables and cards and is remembered', async ({ page }) => {
  await loginAs(page, 'owner', '/orders')
  const rowHeight = async () => (await page.locator('tbody tr').first().boundingBox())!.height
  await expect(page.locator('tbody tr').first()).toBeVisible()
  const comfortable = await rowHeight()

  await page.goto('/settings')
  await page.getByRole('radio', { name: 'Compact' }).click()
  await expect(page.locator('html')).toHaveAttribute('data-density', 'compact')

  await page.goto('/orders')
  await expect(page.locator('tbody tr').first()).toBeVisible()
  expect(await rowHeight()).toBeLessThan(comfortable - 6)

  await page.reload() // survives a reload
  await expect(page.locator('html')).toHaveAttribute('data-density', 'compact')
  await page.goto('/settings')
  await page.getByRole('radio', { name: 'Comfortable' }).click()
  await page.goto('/orders')
  await expect(page.locator('tbody tr').first()).toBeVisible()
  expect(await rowHeight()).toBeGreaterThanOrEqual(comfortable - 1)
})

test('danger zone: clear the request log and sign everyone out, with confirmation', async ({ page, browser }) => {
  const viewer = await browser.newContext()
  await viewer.request.post('/api/v1/auth/login', { data: { email: 'jon@acme.io', password: 'goadmin' } })
  expect((await viewer.request.get('/api/v1/auth/me')).status()).toBe(200)

  await loginAs(page, 'owner', '/settings')
  const zone = page.locator('section', { has: page.getByRole('heading', { name: 'Danger zone' }) })
  await expect(zone.getByRole('button', { name: 'Delete' })).toBeDisabled() // a deliberate refusal, not a fake

  // Cancelling does nothing.
  await zone.getByRole('button', { name: 'Clear', exact: true }).click()
  await page.getByRole('dialog', { name: 'Clear the request log?' }).getByRole('button', { name: 'Cancel' }).click()
  await page.goto('/requests')
  await expect(page.getByTestId('request-rows').getByRole('button').first()).toBeVisible()

  await page.goto('/settings')
  await zone.getByRole('button', { name: 'Clear', exact: true }).click()
  await page.getByRole('dialog', { name: 'Clear the request log?' }).getByRole('button', { name: 'Clear log' }).click()
  await expect(page.getByText(/Request log cleared \(\d+ entries\)/)).toBeVisible()
  await page.goto('/requests')
  await expect(page.getByTestId('request-rows').getByRole('button')).not.toHaveCount(0) // only what happened since
  await expect(page.getByText('Requests (buffer)')).toBeVisible()
  expect(await page.getByTestId('request-rows').getByRole('button').count()).toBeLessThan(15)

  await page.goto('/settings')
  await zone.getByRole('button', { name: 'Sign out', exact: true }).click()
  await page.getByRole('dialog', { name: 'Sign everyone else out?' }).getByRole('button', { name: 'Sign everyone out' }).click()
  await expect(page.getByText(/Signed out \d+ sessions?/)).toBeVisible()
  expect((await viewer.request.get('/api/v1/auth/me')).status()).toBe(401)
  expect((await page.request.get('/api/v1/auth/me')).status()).toBe(200) // the owner stays signed in
  await viewer.close()

  // Admins can't reach it.
  await loginAs(page, 'admin', '/settings')
  await expect(zone.getByRole('button', { name: 'Clear', exact: true })).toBeDisabled()
})