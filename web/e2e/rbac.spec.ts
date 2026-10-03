import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

const nav = (page: import('@playwright/test').Page) => page.locator('aside nav')

test('owner sees every section', async ({ page }) => {
  await loginAs(page, 'owner')
  for (const item of ['Dashboard', 'Products', 'Orders', 'Customers', 'Team & roles', 'Request log', 'Settings']) {
    await expect(nav(page).getByRole('link', { name: item })).toBeVisible()
  }
})

test('viewer is read-only and cannot reach restricted sections', async ({ page }) => {
  await loginAs(page, 'viewer', '/products')
  await expect(nav(page).getByRole('link', { name: 'Request log' })).toHaveCount(0)
  await expect(nav(page).getByRole('link', { name: 'Team & roles' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Add product' })).toHaveCount(0)

  await page.goto('/products?edit=2')
  await expect(page.getByText(/Read-only — your role can view products/)).toBeVisible()
  await expect(page.getByRole('button', { name: 'Save' })).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Upload images' })).toHaveCount(0)

  await page.goto('/requests')
  await expect(page).toHaveURL(/\/forbidden$/)
  await expect(page.getByRole('heading', { name: 'No access' })).toBeVisible()

  // The API enforces the same rules even if the UI is bypassed.
  const res = await page.request.delete('/api/v1/products/2')
  expect(res.status()).toBe(403)
})

test('editor manages products but not the team', async ({ page }) => {
  await loginAs(page, 'editor', '/team')
  await expect(page.getByRole('button', { name: 'Invite member' })).toHaveCount(0)
  await expect(page.getByTestId('permission-matrix')).toBeVisible()
  await page.goto('/products')
  await expect(page.getByRole('button', { name: 'Add product' })).toBeVisible()
})

test('admin cannot edit or remove the owner', async ({ page }) => {
  await loginAs(page, 'admin', '/team')
  const ownerRow = page.locator('tr', { hasText: 'Artur DCS' })
  await expect(ownerRow.getByRole('button')).toHaveCount(0)
  const res = await page.request.delete('/api/v1/team/1')
  expect(res.status()).toBe(403)
})
