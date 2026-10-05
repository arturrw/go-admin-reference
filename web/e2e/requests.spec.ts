import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('request log shows analytics and a detailed, redacted view of each call', async ({ page }) => {
  await loginAs(page, 'owner')
  // Generate a write we can find: a failed validation with a password-like field.
  await page.request.post('/api/v1/team', { data: { name: '', email: 'bad', role: 'viewer' } })

  await page.goto('/requests')
  await expect(page.getByText('Top endpoints')).toBeVisible()
  await page.getByRole('combobox', { name: 'Method' }).selectOption('POST')

  const row = page.getByTestId('request-rows').getByRole('button', { name: /POST\s*\/api\/v1\/team/ }).first()
  await row.click()

  const sheet = page.getByRole('dialog')
  await expect(sheet.getByText('Request detail')).toBeVisible()
  await expect(sheet.getByText('artur@acme.io')).toBeVisible()

  await sheet.getByRole('tab', { name: /Request/ }).click()
  await expect(sheet.getByText('[redacted]').first()).toBeVisible() // Cookie header

  await sheet.getByRole('tab', { name: /Response/ }).click()
  await expect(sheet.getByText(/validation failed/)).toBeVisible()
})

test('traffic charts highlight the hovered bar and show its value', async ({ page }) => {
  await loginAs(page, 'owner', '/requests')
  const traffic = page.getByRole('img', { name: 'Requests per minute, last 30 minutes' })
  await traffic.getByTestId('bar').last().hover()
  await expect(traffic.getByRole('tooltip')).toContainText(/this minute · \d+ requests/)
  await traffic.getByTestId('bar').nth(10).hover()
  await expect(traffic.getByRole('tooltip')).toContainText('19 min ago')

  await page.goto('/')
  const live = page.getByRole('img', { name: 'Requests per second, last two minutes' })
  await live.getByTestId('bar').nth(20).hover()
  await expect(live.getByRole('tooltip')).toContainText(/req\/s · \d+ online/)
})