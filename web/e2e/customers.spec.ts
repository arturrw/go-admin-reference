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
  const orders = sheet.getByRole('button', { name: /^Open order #\d+$/ })
  expect(await orders.count()).toBeGreaterThan(0)

  // Order history opens the order on top of the customer, without leaving the page.
  await orders.first().click()
  await expect(page.getByRole('dialog').getByText('Fulfillment')).toBeVisible()
  await expect(page).toHaveURL(/\/customers\?view=\d+/)
})

test('support can add an internal note', async ({ page }) => {
  await loginAs(page, 'support', '/customers?view=3')
  const sheet = page.getByRole('dialog')
  await sheet.getByRole('tab', { name: /Notes/ }).click()
  await sheet.getByPlaceholder(/internal note/).fill('Called about a late delivery.')
  await sheet.getByRole('button', { name: 'Add note' }).click()
  await expect(sheet.getByText('Called about a late delivery.')).toBeVisible()
  await expect(sheet.getByText('Priya Shah').first()).toBeVisible()

  // Deleting asks for confirmation; cancelling keeps the note.
  const note = sheet.getByTestId('note').filter({ hasText: 'Called about a late delivery.' })
  await note.hover()
  await note.getByRole('button', { name: 'Delete note' }).click()
  const confirm = page.getByRole('dialog', { name: 'Delete this note?' })
  await expect(confirm).toContainText('Called about a late delivery.')
  await confirm.getByRole('button', { name: 'Cancel' }).click()
  await expect(note).toBeVisible()

  await note.hover()
  await note.getByRole('button', { name: 'Delete note' }).click()
  await confirm.getByRole('button', { name: 'Delete note' }).click()
  await expect(page.getByText('Note deleted')).toBeVisible()
  await expect(note).toHaveCount(0)

  // …and the deletion is in the dashboard's activity feed.
  await page.goto('/')
  const feed = page.locator('.card', { has: page.getByRole('heading', { name: 'Activity' }) })
  await expect(feed.getByTestId('activity-item').first()).toContainText("deleted Priya Shah's note")
})

test('overview widgets drill into filtered orders and products', async ({ page }) => {
  await loginAs(page, 'owner', '/customers?view=10')
  const sheet = page.getByRole('dialog')
  const tabs = sheet.getByRole('tablist')

  // A month bar opens the orders placed that month.
  const bar = sheet.getByRole('button', { name: /^\w+ \d{4}: \$[\d,]+, [1-9]\d* orders$/ }).last()
  const [, month, count] = (await bar.getAttribute('aria-label'))!.match(/^(\w+ \d{4}): .*, (\d+) orders$/)!
  await bar.hover()
  await expect(sheet.getByText(`${count} orders`).first()).toBeVisible()
  await bar.click()
  await expect(sheet.getByRole('tab', { name: /Orders/ })).toHaveAttribute('aria-selected', 'true')
  await expect(sheet.getByText(month)).toBeVisible()
  const orders = sheet.getByRole('button', { name: /^Open order #\d+$/ })
  await expect(orders).toHaveCount(Number(count))
  // The tab strip keeps its height next to a long list.
  expect((await tabs.boundingBox())!.height).toBeGreaterThan(25)

  await sheet.getByRole('button', { name: 'Clear filter' }).click()
  expect(await orders.count()).toBeGreaterThan(Number(count) - 1)

  // A favourite category opens the products bought in it.
  await sheet.getByRole('tab', { name: /Overview/ }).click()
  const cat = sheet.locator('section', { hasText: 'Favourite categories' }).getByRole('button', { name: /^\w+\s*\d+%$/ }).first()
  const category = (await cat.locator('span').first().innerText()).trim()
  await cat.click()
  await expect(sheet.getByRole('tab', { name: /Products/ })).toHaveAttribute('aria-selected', 'true')
  const products = sheet.getByRole('button', { name: /^Open (?!order )/ })
  expect(await products.count()).toBeGreaterThan(0)
  for (const t of await products.allInnerTexts()) expect(t).toContain(category)
  const product = (await products.first().locator('b').first().innerText()).trim()
  await products.first().click()
  await expect(page.getByRole('dialog').getByRole('heading', { name: product })).toBeVisible()
  await expect(page).toHaveURL(/\/customers\?view=10$/)
})
