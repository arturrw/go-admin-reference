import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('unauthenticated visitors are sent to the login page', async ({ page }) => {
  await page.goto('/products')
  await expect(page).toHaveURL(/\/login\?redirect=/)
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
})

test('wrong password shows an error', async ({ page }) => {
  await page.goto('/login')
  await page.getByLabel('Email').fill('artur@acme.io')
  await page.getByLabel('Password').fill('wrong')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await expect(page.getByText('invalid email or password')).toBeVisible()
})

test('owner signs in with a demo account and lands on the page they asked for', async ({ page }) => {
  await page.goto('/customers')
  await page.getByRole('button', { name: /Artur DCS/ }).click()
  await expect(page).toHaveURL(/\/customers$/)
  await expect(page.getByTestId('current-user')).toHaveText('Artur DCS')
})

test('GoAdmin logo navigates to the dashboard', async ({ page }) => {
  await loginAs(page, 'owner', '/orders')
  await page.getByRole('link', { name: /GoAdmin — go to dashboard/ }).click()
  await expect(page).toHaveURL(/\/$/)
  await expect(page.getByRole('heading', { name: /Good (morning|afternoon|evening), Artur/ })).toBeVisible()
})

test('sign out ends the session', async ({ page }) => {
  await loginAs(page, 'owner')
  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page).toHaveURL(/\/login/)
  const res = await page.request.get('/api/v1/auth/me')
  expect(res.status()).toBe(401)
})

test('an expired session redirects to login without crashing the page', async ({ page }) => {
  const errors: string[] = []
  page.on('pageerror', (e) => errors.push(e.message))
  await loginAs(page, 'owner', '/products')
  await expect(page.getByRole('heading', { name: 'Products' })).toBeVisible()

  // Kill the session server-side, then trigger a client-side fetch.
  await page.request.post('/api/v1/auth/logout')
  await page.locator('aside nav').getByRole('link', { name: 'Orders' }).click()

  await expect(page).toHaveURL(/\/login\?redirect=/)
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
  expect(errors).toEqual([])
})
