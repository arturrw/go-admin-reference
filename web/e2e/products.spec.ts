import { expect, test } from '@playwright/test'
import { loginAs, PNG } from './helpers'

test('product cards show Unsplash photos', async ({ page }) => {
  await loginAs(page, 'owner', '/products')
  const firstImage = page.locator('article img').first()
  await expect(firstImage).toBeVisible()
  await expect(firstImage).toHaveAttribute('src', /^https:\/\/images\.unsplash\.com\/photo-/)
})

test('upload, set as cover and delete a product image', async ({ page }) => {
  await loginAs(page, 'admin', '/products?edit=5')
  const gallery = page.getByTestId('image-gallery')
  const thumbs = gallery.getByTestId('thumbs').locator('img')
  await expect(thumbs).toHaveCount(3)

  await page.getByTestId('image-input').setInputFiles({ name: 'studio.png', mimeType: 'image/png', buffer: PNG })
  await expect(page.getByText('Image uploaded')).toBeVisible()
  await expect(thumbs).toHaveCount(4)
  await expect(thumbs.nth(3)).toHaveAttribute('src', /\/media\/uploads\/.+\.png$/)

  await thumbs.nth(3).hover()
  await gallery.getByRole('button', { name: 'Make cover' }).last().click()
  await expect(thumbs.first()).toHaveAttribute('src', /\/media\/uploads\//)

  await thumbs.first().hover()
  await gallery.getByRole('button', { name: 'Delete image' }).first().click()
  await expect(thumbs).toHaveCount(3)
  await expect(thumbs.first()).toHaveAttribute('src', /images\.unsplash\.com/)
})

test('rejects non-image uploads', async ({ page }) => {
  await loginAs(page, 'admin', '/products?edit=6')
  await page.getByTestId('image-input').setInputFiles({ name: 'notes.txt', mimeType: 'text/plain', buffer: Buffer.from('hello') })
  await expect(page.getByText(/use JPEG, PNG, WebP or GIF/)).toBeVisible()
})

test('edit description, tags and pricing', async ({ page }) => {
  await loginAs(page, 'editor', '/products?edit=7')
  await page.getByLabel('Description').fill('Updated by the e2e suite.')
  const tags = page.getByRole('textbox', { name: 'Add tag' })
  await tags.fill('e2e-tag')
  await tags.press('Enter')
  await page.getByLabel('Cost per item').fill('10')
  await page.getByRole('button', { name: 'Save' }).click()
  await expect(page.getByText('Product updated')).toBeVisible()

  const res = await page.request.get('/api/v1/products/7')
  const p = await res.json()
  expect(p.description).toBe('Updated by the e2e suite.')
  expect(p.tags).toContain('e2e-tag')
  expect(p.costCents).toBe(1000)
})

test('create a product, then keep editing it to add images', async ({ page }) => {
  await loginAs(page, 'owner', '/products')
  await page.getByRole('button', { name: 'Add product' }).click()
  const sheet = page.getByRole('dialog')
  await sheet.getByLabel('Name', { exact: true }).fill('E2E Lamp')
  await sheet.getByLabel('Vendor').fill('Test Vendor')
  await sheet.getByRole('button', { name: 'Create product' }).click()
  await expect(page.getByText('Product created')).toBeVisible()
  await expect(page).toHaveURL(/edit=\d+/)
  await expect(page.getByRole('button', { name: 'Upload images' })).toBeVisible()
})
