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
