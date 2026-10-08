import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('a refund asks for a reason that stays in the order and the customer history', async ({ page }) => {
  const res = await page.request.post('/api/v1/auth/login', { data: { email: 'priya@acme.io', password: 'goadmin' } })
  expect(res.ok()).toBeTruthy()
  const list = await (await page.request.get('/api/v1/orders?status=delivered&limit=1')).json()
  const order = list.items[0]

  await loginAs(page, 'support', `/orders?view=${order.id}`)
  const sheet = page.getByRole('dialog', { name: `Order #${order.id}` })
  await sheet.getByRole('button', { name: 'Refund' }).click()

  const dialog = page.getByRole('dialog', { name: `Refund order #${order.id}` })
  await dialog.getByRole('radio', { name: 'Other' }).click()
  // "Other" needs a written reason before the refund can go through.
  await expect(dialog.getByRole('button', { name: /^Refund \$/ })).toBeDisabled()
  await dialog.getByRole('radio', { name: 'Wrong item sent' }).click()
  await dialog.getByLabel('Refund details').fill('sent the blue one')
  await dialog.getByRole('button', { name: /^Refund \$/ }).click()

  await expect(dialog).toBeHidden()
  await expect(sheet.getByText('Wrong item sent: sent the blue one')).toBeVisible()
  await expect(sheet.getByText(/Refunded by Priya Shah/)).toBeVisible()
  await expect(sheet.getByRole('button', { name: 'Refund' })).toBeDisabled()

  // The customer's purchase history shows the reason too.
  await sheet.getByRole('button', { name: /^Open customer / }).click()
  const customer = page.getByRole('dialog')
  await customer.getByRole('tab', { name: /Orders/ }).click()
  await expect(customer.getByRole('button', { name: `Open order #${order.id}` }).getByTestId('refund-reason')).toContainText('Wrong item sent: sent the blue one')

  // …and so does the orders list, as does every seeded refund.
  await page.goto('/orders')
  await page.getByRole('radio', { name: /Refunded/ }).click()
  const row = page.locator('tbody tr', { hasText: `#${order.id}` })
  await expect(row.getByTestId('refund-reason')).toHaveText('Wrong item sent: sent the blue one')
  for (const reason of await page.getByTestId('refund-reason').allInnerTexts()) expect(reason).not.toBe('No reason recorded')
})

test('the invoice opens as a printable page with the order numbers', async ({ page, context }) => {
  await loginAs(page, 'viewer', '/orders')
  await page.locator('tbody tr').first().click()
  const sheet = page.getByRole('dialog')
  const id = (await sheet.getByRole('heading').first().innerText()).replace(/\D/g, '')
  const total = (await sheet.locator('dd').last().innerText()).trim()

  const [invoice] = await Promise.all([context.waitForEvent('page'), sheet.getByRole('link', { name: 'Invoice' }).click()])
  await invoice.waitForLoadState()
  await expect(invoice.getByRole('heading', { name: 'Invoice' })).toBeVisible()
  await expect(invoice.getByText(`INV-${id}`).first()).toBeVisible()
  await expect(invoice.locator('.grand')).toContainText(total)
  await expect(invoice.getByRole('button', { name: 'Print / Save as PDF' })).toBeVisible()
  // The print button really prints.
  await invoice.evaluate(() => {
    ;(window as unknown as { printed: number }).printed = 0
    window.print = () => ((window as unknown as { printed: number }).printed += 1)
  })
  await invoice.getByRole('button', { name: 'Print / Save as PDF' }).click()
  expect(await invoice.evaluate(() => (window as unknown as { printed: number }).printed)).toBe(1)
})
test('the timeline is the order history and moves forward one step at a time', async ({ page }) => {
  await page.request.post('/api/v1/auth/login', { data: { email: 'priya@acme.io', password: 'goadmin' } })
  const list = await (await page.request.get('/api/v1/orders?status=pending&limit=1')).json()
  const id = list.items[0].id

  await loginAs(page, 'support', `/orders?view=${id}`)
  const sheet = page.getByRole('dialog', { name: `Order #${id}` })
  const timeline = sheet.getByTestId('order-timeline')
  await expect(timeline).toContainText('Placed')
  await expect(timeline).toContainText('Awaiting payment')
  await expect(sheet.getByRole('button', { name: 'Mark shipped' })).toHaveCount(0)

  await sheet.getByRole('button', { name: 'Mark paid' }).click()
  await expect(timeline).toContainText('Paid')
  await expect(timeline).toContainText('Priya Shah')
  await expect(timeline).toContainText('Awaiting shipment')

  await sheet.getByRole('button', { name: 'Mark shipped' }).click()
  await expect(timeline).toContainText('Awaiting delivery')
  await sheet.getByRole('button', { name: 'Mark delivered' }).click()
  await expect(timeline).not.toContainText('Awaiting')
  await expect(timeline.getByText('Delivered')).toBeVisible()
  await expect(sheet.getByRole('button', { name: 'Delivered' })).toBeDisabled()
})
test('a sale entered by hand takes stock and shows up in the dashboard and the product', async ({ page }) => {
  await loginAs(page, 'support', '/orders')
  const api = page.request
  const stockOf = async (id: number) => (await (await api.get(`/api/v1/products/${id}`)).json()) as { stock: number; sold30d: number; name: string; priceCents: number }
  const products = (await (await api.get('/api/v1/products?status=active&sort=name')).json()).items as { id: number; name: string; stock: number; priceCents: number }[]
  const product = products.find((p) => p.stock >= 5)!
  const before = await stockOf(product.id)
  const revenue = async () => ((await (await api.get('/api/v1/dashboard?range=30')).json()) as { revenueCents: number }).revenueCents
  const revBefore = await revenue()

  await page.getByRole('button', { name: 'New order' }).click()
  const form = page.getByRole('dialog', { name: 'New order' })
  await expect(form.getByRole('button', { name: 'Create order' })).toBeDisabled()
  await form.getByRole('option').first().click()
  await form.getByLabel('Product 1', { exact: true }).selectOption({ label: `${product.name} · $${(product.priceCents / 100).toLocaleString('en-US', { minimumFractionDigits: 2 })} · ${product.stock} in stock` })
  await form.getByLabel('Quantity 1', { exact: true }).fill('2')
  await form.getByLabel('Payment').selectOption('PayPal')
  await form.getByRole('button', { name: 'Create order' }).click()

  // The new order opens, pending, with its first timeline entry.
  const sheet = page.getByRole('dialog', { name: /^Order #/ })
  await expect(sheet.getByText('pending')).toBeVisible()
  await expect(sheet.getByTestId('order-timeline')).toContainText('Priya Shah')
  await expect(sheet.getByText(product.name)).toBeVisible()

  const after = await stockOf(product.id)
  expect(after.stock).toBe(before.stock - 2)
  expect(after.sold30d).toBe(before.sold30d + 2)
  expect(await revenue()).toBe(revBefore + 2 * product.priceCents)
})
