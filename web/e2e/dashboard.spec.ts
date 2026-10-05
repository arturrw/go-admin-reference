import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('hovering the category donut shows that category’s share', async ({ page }) => {
  await loginAs(page, 'owner', '/')
  const card = page.locator('.card', { has: page.getByRole('heading', { name: 'Sales by category' }) })
  await expect(card.getByText('gross sales')).toBeVisible()

  const legendRow = card.getByText('Wearables', { exact: true })
  const share = await legendRow.locator('xpath=..').locator('b').innerText()
  await legendRow.hover()
  await expect(card.getByText('gross sales')).toBeHidden()
  // The centre now shows the hovered category and its percentage.
  const centre = card.locator('.pointer-events-none').first()
  await expect(centre).toContainText('Wearables')
  await expect(centre).toContainText(share)

  // Hovering the ring itself works the same way. The first segment starts at
  // 12 o'clock, so point just right of the top of the ring.
  await card.getByText('Sales by category').hover()
  const first = (await card.locator('circle[aria-label]').first().getAttribute('aria-label'))!.split(':')[0]
  const ring = card.locator('svg').first()
  const box = (await ring.boundingBox())!
  await page.mouse.move(box.x + box.width / 2 + 8, box.y + box.height * 0.1)
  await expect(centre).toContainText(first)
})

test('KPI cards open a detailed chart', async ({ page }) => {
  await loginAs(page, 'owner', '/')
  await page.getByRole('button', { name: 'Orders: open details' }).click()
  const sheet = page.getByRole('dialog')
  await expect(sheet.getByRole('heading', { name: 'Orders' })).toBeVisible()
  await expect(sheet.getByRole('img', { name: 'Orders over time' })).toBeVisible()
  await expect(sheet.getByText('Best day')).toBeVisible()
  // One row per day of the default 30-day range.
  await expect(sheet.locator('tbody tr')).toHaveCount(30)

  await sheet.getByRole('tab', { name: 'Conversion' }).click()
  await expect(sheet.getByRole('img', { name: 'Conversion over time' })).toBeVisible()
  await expect(sheet.getByText('Period average')).toBeVisible()
})

test('live card opens live details with the hottest product', async ({ page }) => {
  await loginAs(page, 'owner', '/')
  const card = page.getByRole('button', { name: 'Open live traffic details' })
  await expect(card.getByText('Hottest product')).toBeVisible()
  await card.click()

  const sheet = page.getByRole('dialog')
  await expect(sheet.getByText('Live storefront')).toBeVisible()
  await expect(sheet.getByText('Hottest product right now')).toBeVisible()
  await expect(sheet.getByText('Viewing now')).toBeVisible()
  await expect(sheet.getByText('Top pages right now')).toBeVisible()
  await sheet.getByRole('radio', { name: 'Hot product' }).click()
  await expect(sheet.getByRole('img', { name: 'Hot product over time' })).toBeVisible()

  await sheet.getByRole('link', { name: /Open product/ }).click()
  await expect(page).toHaveURL(/\/products\?edit=\d+/)
})

test('top products and recent orders open their sheets on the dashboard itself', async ({ page }) => {
  await loginAs(page, 'owner', '/')
  const dialog = page.getByRole('dialog')
  const top = page.locator('.card', { has: page.getByRole('heading', { name: 'Top products' }) })
  const first = top.getByRole('button', { name: /^Open / }).first()
  const name = (await first.locator('b').first().innerText()).trim()
  await first.click()
  await expect(dialog.getByRole('heading', { name })).toBeVisible()
  await expect(page).toHaveURL(/\/$/)
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()

  const orders = page.locator('.card', { has: page.getByRole('heading', { name: 'Recent orders' }) })
  const customerLink = orders.locator('tbody tr').first().getByRole('button', { name: /^Open customer / })
  const customer = (await customerLink.innerText()).trim().split('\n').pop()!
  await customerLink.click()
  await expect(dialog.getByRole('heading', { name: customer })).toBeVisible()
  await expect(page).toHaveURL(/\/$/)
  await page.keyboard.press('Escape')

  await orders.locator('tbody tr').first().locator('td').nth(2).click()
  await expect(dialog.getByText('Fulfillment')).toBeVisible()
  await expect(page).toHaveURL(/\/$/)

  // Sheets opened from a sheet stack: closing the customer goes back to the order.
  await dialog.getByRole('button', { name: /^Open customer / }).click()
  await expect(dialog.getByRole('heading', { name: customer })).toBeVisible()
  await dialog.getByRole('button', { name: 'Close' }).click()
  await expect(dialog.getByText('Fulfillment')).toBeVisible()
})
