import { expect, type Page, test } from '@playwright/test'
import { loginAs } from './helpers'

test.use({ viewport: { width: 375, height: 812 }, hasTouch: true, isMobile: true })

/** No page may scroll sideways on a phone; wide tables scroll inside their card. */
async function expectNoSideScroll(page: Page) {
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth)).toBeLessThanOrEqual(0)
}

test('every page fits a phone screen', async ({ page }) => {
  await loginAs(page, 'owner', '/')
  for (const path of ['/', '/products', '/orders', '/customers', '/team', '/activity', '/requests', '/settings']) {
    await page.goto(path)
    await expect(page.locator('main h1')).toBeVisible()
    await expectNoSideScroll(page)
  }
})

test('the menu opens from the header and navigates', async ({ page }) => {
  await loginAs(page, 'owner', '/')
  const nav = page.locator('aside nav')
  await expect(nav).not.toBeInViewport()
  await page.getByRole('button', { name: 'Open menu' }).tap()
  await expect(nav).toBeInViewport()
  await nav.getByRole('link', { name: 'Orders' }).tap()
  await expect(page).toHaveURL(/\/orders$/)
  await expect(nav).not.toBeInViewport()
})

test('orders show status and total as compact rows and open the order', async ({ page }) => {
  await loginAs(page, 'owner', '/orders')
  const row = page.getByTestId('order-row').first()
  await expect(row).toBeVisible()
  await expect(row).toContainText(/\$[\d,]+\.\d{2}/)
  await row.tap()
  await expect(page.getByRole('dialog').getByText('Timeline', { exact: true })).toBeVisible()
  await expectNoSideScroll(page)
})

test('sheets fit the screen', async ({ page }) => {
  await loginAs(page, 'owner', '/customers?view=25')
  const sheet = page.getByRole('dialog')
  await expect(sheet.getByText('Total spent')).toBeVisible()
  // Polled: the sheet slides in from the right.
  await expect.poll(async () => { const b = (await sheet.boundingBox())!; return b.x >= 0 && b.x + b.width <= 375 }).toBe(true)
})
