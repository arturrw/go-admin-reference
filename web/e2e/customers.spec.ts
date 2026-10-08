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
  await expect(page.getByRole('dialog').getByText('Timeline', { exact: true })).toBeVisible()
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

test('support adds a customer by hand; viewers can\'t', async ({ page }) => {
  await loginAs(page, 'support', '/customers')
  await page.getByRole('button', { name: 'Add customer' }).click()
  const form = page.getByRole('dialog', { name: 'Add customer' })
  const add = form.getByRole('button', { name: 'Add customer' })
  await expect(add).toBeDisabled() // name and email first

  await form.getByLabel('Name').fill('Greta Lindqvist')
  await form.getByLabel('Email').fill('greta.lindqvist@example.com')
  await form.getByLabel('Phone').fill('+46 8 123 456')
  await form.getByLabel('Street address').fill('Drottninggatan 5')
  await form.getByLabel('City').fill('Stockholm')
  await form.getByLabel('Postal code').fill('11151')
  await form.getByLabel('Country').selectOption('SE')
  await form.getByRole('textbox', { name: 'Add tag' }).fill('phone-order')
  await form.getByRole('textbox', { name: 'Add tag' }).press('Enter')
  await form.getByRole('switch', { name: 'Accepts marketing' }).click()
  await add.click()

  // The new customer's own sheet opens: New segment, no orders yet.
  const sheet = page.getByRole('dialog', { name: 'Greta Lindqvist' })
  await expect(sheet).toBeVisible()
  await expect(sheet.getByText('New', { exact: true }).first()).toBeVisible()
  await expect(sheet.getByText('Drottninggatan 5, 11151 Stockholm, SE')).toBeVisible()
  await expect(sheet.getByText('phone-order')).toBeVisible()
  await expect(sheet.getByText('Subscribed to marketing')).toBeVisible()
  await page.keyboard.press('Escape')

  // …and they are in the list and in the activity feed.
  await page.getByPlaceholder('Search customers…').fill('greta')
  await expect(page.locator('tbody tr')).toHaveCount(1)
  await page.goto('/')
  const feed = page.locator('.card', { has: page.getByRole('heading', { name: 'Activity' }) })
  await expect(feed.getByTestId('activity-item').first()).toContainText('added customer Greta Lindqvist')

  // The same email can't be added twice.
  await page.goto('/customers')
  await page.getByRole('button', { name: 'Add customer' }).click()
  await form.getByLabel('Name').fill('Greta Again')
  await form.getByLabel('Email').fill('GRETA.LINDQVIST@example.com')
  await add.click()
  await expect(form.getByText('is already a customer')).toBeVisible()

  await loginAs(page, 'viewer', '/customers')
  await expect(page.getByRole('button', { name: 'Add customer' })).toHaveCount(0)
})

test('emailing a segment prepares a Bcc draft for those who accept marketing', async ({ page, context }) => {
  await context.grantPermissions(['clipboard-read', 'clipboard-write'])
  await loginAs(page, 'support', '/customers')
  await page.getByRole('button', { name: /^VIP\b/ }).click() // the segment card filters the list
  const all = await (await page.request.get('/api/v1/customers?segment=VIP')).json()
  const optedIn: string[] = all.items.filter((c: { acceptsMarketing: boolean }) => c.acceptsMarketing).map((c: { email: string }) => c.email)
  expect(optedIn.length).toBeGreaterThan(0)

  await page.getByRole('button', { name: 'Email vip' }).click()
  const dialog = page.getByRole('dialog', { name: /Email the VIP segment/ })
  await expect(dialog.getByTestId('recipients')).toContainText(`${optedIn.length}`)
  await expect(dialog.getByTestId('recipients')).toContainText(`${all.items.length - optedIn.length} left out`)

  await dialog.getByLabel('Subject').fill('Spring picks & more')
  await dialog.getByLabel('Message').fill('Hi there,\nnew arrivals.')
  const href = (await dialog.getByTestId('mailto').getAttribute('href'))!
  expect(href.startsWith('mailto:?bcc=')).toBe(true)
  const bcc = decodeURIComponent(href.split('bcc=')[1].split('&')[0]).split(',')
  expect(bcc.sort()).toEqual([...optedIn].sort())
  expect(href).toContain('subject=Spring%20picks%20%26%20more')
  expect(href).toContain('body=Hi%20there%2C%0Anew%20arrivals.')

  await dialog.getByRole('button', { name: 'Copy addresses' }).click()
  expect(await page.evaluate(() => navigator.clipboard.readText())).toBe(optedIn.join(', '))

  // A link can't carry unlimited text: a long message leaves room for fewer addresses, and says so.
  await dialog.getByLabel('Message').fill('x'.repeat(1800))
  await expect(dialog.getByTestId('mailto-limit')).toBeVisible()
})