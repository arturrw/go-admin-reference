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
