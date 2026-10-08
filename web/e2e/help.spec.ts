import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('help shows your role, what it allows and the shortcuts', async ({ page }) => {
  await loginAs(page, 'viewer', '/')
  await page.getByRole('button', { name: 'Help' }).click()
  const help = page.getByRole('dialog', { name: 'Help' })
  await expect(help).toContainText('viewer')
  const access = help.getByTestId('help-access')
  await expect(access.locator('div', { hasText: /^View dashboard/ }).first().getByLabel('allowed')).toBeVisible()
  // A viewer can't edit products or manage the team, and the list says so.
  await expect(access.getByText('Edit products').locator('xpath=ancestor::div[2]').getByLabel('not allowed')).toBeVisible()
  await expect(help.getByTestId('help-version')).toContainText('api v')
  await expect(help.getByRole('link', { name: 'API reference' })).toHaveAttribute('href', /docs\/API\.md$/)
  await page.keyboard.press('Escape')
  await expect(help).toBeHidden()
})

test('? opens help, but not while typing', async ({ page }) => {
  await loginAs(page, 'owner', '/products')
  await page.getByPlaceholder('Search name or SKU…').click()
  await page.keyboard.type('what?')
  await expect(page.getByRole('dialog', { name: 'Help' })).toHaveCount(0)
  await expect(page.getByPlaceholder('Search name or SKU…')).toHaveValue('what?')

  await page.locator('main h1').click()
  await page.keyboard.press('?')
  await expect(page.getByRole('dialog', { name: 'Help' })).toBeVisible()
})

test('help lists the owner exceptions made to a role', async ({ page }) => {
  // Diego (support) was granted "Edit products" in the demo data.
  const res = await page.request.post('/api/v1/auth/login', { data: { email: 'diego@acme.io', password: 'goadmin' } })
  expect(res.ok()).toBeTruthy()
  await page.goto('/')
  await page.getByRole('button', { name: 'Help' }).click()
  const help = page.getByRole('dialog', { name: 'Help' })
  await expect(help).toContainText('The owner made 1 exception(s) to your role.')
  await expect(help.getByTestId('help-access').getByText('Edit products').locator('xpath=ancestor::div[2]').getByLabel('allowed')).toBeVisible()
})