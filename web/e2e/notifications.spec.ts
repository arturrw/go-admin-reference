import { expect, test } from '@playwright/test'
import { loginAs, PASSWORD, USERS } from './helpers'

test('the bell shows what others did, opens records in place and marks them read', async ({ page, browser }) => {
  await loginAs(page, 'viewer', '/')
  const bell = page.getByRole('button', { name: /^Notifications/ })
  await expect(page.getByTestId('unread-count')).toBeVisible()

  await bell.click()
  const panel = page.getByRole('dialog', { name: 'Notifications' })
  await expect(panel.getByTestId('notification').first()).toHaveAttribute('data-unread', 'true')
  // Never your own actions or sign-ins.
  for (const t of await panel.getByTestId('activity-item').allInnerTexts()) expect(t).not.toMatch(/^Jon Berg /)

  // An entry opens its record on top of the page and closes the bell.
  await panel.locator('[data-testid=activity-item][role=button]').first().click()
  await expect(panel).toBeHidden()
  await expect(page.getByRole('dialog')).toBeVisible()
  await expect(page).toHaveURL(/\/$/)
  await page.keyboard.press('Escape')

  // Closing the bell marked everything read, on the server too.
  await expect(page.getByTestId('unread-count')).toBeHidden()
  await page.reload()
  await expect(page.getByTestId('unread-count')).toBeHidden()

  // Someone else changes something: it arrives as the one unread entry.
  const admin = await browser.newContext()
  await admin.request.post('/api/v1/auth/login', { data: { email: USERS.admin, password: PASSWORD } })
  await new Promise((r) => setTimeout(r, 50))
  const res = await admin.request.post('/api/v1/customers/7/notes', { data: { text: 'Bell e2e' } })
  expect(res.status()).toBe(201)
  await admin.close()

  await page.reload()
  await expect(page.getByTestId('unread-count')).toHaveText('1')
  await bell.click()
  const first = panel.getByTestId('notification').first()
  await expect(first).toHaveAttribute('data-unread', 'true')
  await expect(first).toContainText('Mark Liu')
  await expect(first).toContainText('added a note on')
  await panel.getByRole('button', { name: 'Mark all read' }).click()
  await expect(page.getByTestId('unread-count')).toBeHidden()
})

test('an alert about a new device reaches only that member', async ({ page, browser }) => {
  const ctx = await browser.newContext({ userAgent: 'Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0' })
  await ctx.request.post('/api/v1/auth/login', { data: { email: USERS.support, password: PASSWORD } })
  await ctx.close()

  await loginAs(page, 'support', '/')
  await page.getByRole('button', { name: /^Notifications/ }).click()
  await expect(page.getByRole('dialog', { name: 'Notifications' })).toContainText('New sign-in to Priya Shah')

  await loginAs(page, 'viewer', '/')
  await page.getByRole('button', { name: /^Notifications/ }).click()
  await expect(page.getByRole('dialog', { name: 'Notifications' })).not.toContainText('New sign-in to Priya Shah')
})