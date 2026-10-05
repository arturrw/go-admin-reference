import { expect, test } from '@playwright/test'
import { loginAs, PASSWORD, USERS } from './helpers'

test('owner opens a member, sees presence and activity, and grants a permission', async ({ page, browser }) => {
  // Jon (viewer) is active in another browser, so he shows as online.
  const other = await browser.newContext()
  const res = await other.request.post('/api/v1/auth/login', { data: { email: USERS.viewer, password: PASSWORD } })
  expect(res.ok()).toBeTruthy()

  await loginAs(page, 'owner', '/team')
  await page.getByRole('button', { name: 'Open Jon Berg' }).click()
  const sheet = page.getByRole('dialog', { name: 'Jon Berg' })
  await expect(sheet.getByTestId('presence')).toHaveText(/Online now/)
  await expect(sheet.getByText('Two-factor auth')).toBeVisible()

  await sheet.getByRole('tab', { name: 'Activity' }).click()
  await expect(sheet.getByTestId('activity-item').first()).toContainText('Jon Berg signed in')

  await sheet.getByRole('tab', { name: /Access/ }).click()
  const access = sheet.getByTestId('access')
  await access.getByRole('switch', { name: 'Manage orders' }).click()
  await access.getByRole('switch', { name: 'View customers' }).click()
  await expect(sheet.getByText('1 granted · 1 revoked vs viewer')).toBeVisible()
  await sheet.getByRole('button', { name: 'Save access' }).click()
  await expect(page.getByText('Access updated for Jon Berg')).toBeVisible()
  await expect(access.getByText('granted', { exact: true })).toBeVisible()
  await expect(access.getByText('revoked', { exact: true })).toBeVisible()

  // Jon's own session picks up the new permissions.
  const me = await (await other.request.get('/api/v1/auth/me')).json()
  expect(me.permissions).toContain('orders:write')
  expect(me.permissions).not.toContain('customers:read')
  expect((await other.request.get('/api/v1/customers')).status()).toBe(403)

  // Back to role defaults.
  await sheet.getByRole('button', { name: 'Role defaults' }).click()
  await sheet.getByRole('button', { name: 'Save access' }).click()
  await expect(sheet.getByText('Same as the viewer role')).toBeVisible()
  await other.close()
})

test('suspending a member asks for confirmation and signs them out', async ({ page, browser }) => {
  const other = await browser.newContext()
  await other.request.post('/api/v1/auth/login', { data: { email: USERS.support, password: PASSWORD } })

  await loginAs(page, 'owner', '/team')
  await page.getByRole('button', { name: 'Open Priya Shah' }).click()
  const sheet = page.getByRole('dialog', { name: 'Priya Shah' })
  await sheet.getByRole('button', { name: 'Suspend' }).click()
  await page.getByRole('dialog', { name: 'Suspend Priya Shah?' }).getByRole('button', { name: 'Suspend' }).click()
  await expect(page.getByText('Suspended Priya Shah')).toBeVisible()
  expect((await other.request.get('/api/v1/auth/me')).status()).toBe(401)

  await sheet.getByRole('button', { name: 'Reactivate' }).click()
  await expect(page.getByText('Reactivated Priya Shah')).toBeVisible()
  await other.close()
})

test('admins see member details but only the owner edits individual permissions', async ({ page }) => {
  await loginAs(page, 'admin', '/team')
  await page.getByRole('button', { name: 'Open Yuki Tanaka' }).click()
  const sheet = page.getByRole('dialog', { name: 'Yuki Tanaka' })
  await sheet.getByRole('tab', { name: /Access/ }).click()
  await expect(sheet.getByText('Only the owner can change individual permissions.')).toBeVisible()
  await expect(sheet.getByRole('switch')).toHaveCount(0)
})
