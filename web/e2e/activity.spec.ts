import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('a change shows up in the dashboard feed and the full activity log', async ({ page }) => {
  await loginAs(page, 'admin', '/products?edit=8')
  const name = (await page.getByLabel('Name', { exact: true }).inputValue()).trim()
  await page.getByLabel('Stock').fill('321')
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText('Product updated')).toBeVisible()

  await page.goto('/')
  const feed = page.locator('.card', { has: page.getByRole('heading', { name: 'Activity' }) })
  const entry = feed.getByTestId('activity-item').first()
  await expect(entry).toContainText('Mark Liu')
  await expect(entry).toContainText(`updated ${name}: stock`)
  await expect(entry).toContainText('→ 321')

  // The entry opens the product it touched, without leaving the dashboard.
  await entry.click()
  await expect(page.getByRole('dialog').getByRole('heading', { name })).toBeVisible()
  await page.keyboard.press('Escape')

  await feed.getByRole('link', { name: 'View all' }).click()
  await expect(page).toHaveURL(/\/activity$/)
  await expect(page.getByRole('heading', { name: 'Activity log' })).toBeVisible()
  await expect(page.getByTestId('activity-item').first()).toContainText(`updated ${name}`)

  // Filter to one member and one kind.
  await page.getByLabel('Member').selectOption({ label: 'Yuki Tanaka' })
  await expect(page.getByText('Showing everything Yuki Tanaka did')).toBeVisible()
  for (const t of await page.getByTestId('activity-item').allInnerTexts()) expect(t).toContain('Yuki Tanaka')
  await page.getByLabel('Type').selectOption({ label: 'Sign-ins' })
  await expect(page.getByTestId('activity-item').first()).toContainText('signed in')
})

test('the full activity log is hidden from roles without team access', async ({ page }) => {
  await loginAs(page, 'support', '/')
  const feed = page.locator('.card', { has: page.getByRole('heading', { name: 'Activity' }) })
  await expect(feed.getByTestId('activity-item').first()).toBeVisible()
  await expect(feed.getByRole('link', { name: 'View all' })).toHaveCount(0)
  await page.goto('/activity')
  await expect(page).toHaveURL(/\/forbidden$/)
})
