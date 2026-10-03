import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('clicking a customer opens their full profile and purchase history', async ({ page }) => {
  await loginAs(page, 'owner', '/customers')
  const firstRow = page.locator('table tbody tr').first()
  const name = (await firstRow.locator('b').first().textContent())!.trim()
  await firstRow.click()

  const sheet = page.getByRole('dialog')
  await expect(sheet.getByRole('heading', { name })).toBeVisible()
  await expect(sheet.getByText('Total spent')).toBeVisible()
  await expect(sheet.getByText('Contact', { exact: true })).toBeVisible()
  await expect(page).toHaveURL(/view=\d+/)

  await sheet.getByRole('tab', { name: /Orders/ }).click()
  const orders = sheet.locator('a[href*="/orders?view="]')
  expect(await orders.count()).toBeGreaterThan(0)

  // Order history links through to the order sheet.
  await orders.first().click()
  await expect(page).toHaveURL(/\/orders\?view=\d+/)
  await expect(page.getByRole('dialog').getByText('Fulfillment')).toBeVisible()
})

test('support can add an internal note', async ({ page }) => {
  await loginAs(page, 'support', '/customers?view=3')
  const sheet = page.getByRole('dialog')
  await sheet.getByRole('tab', { name: /Notes/ }).click()
  await sheet.getByPlaceholder(/internal note/).fill('Called about a late delivery.')
  await sheet.getByRole('button', { name: 'Add note' }).click()
  await expect(sheet.getByText('Called about a late delivery.')).toBeVisible()
  await expect(sheet.getByText('Priya Shah').first()).toBeVisible()
})
