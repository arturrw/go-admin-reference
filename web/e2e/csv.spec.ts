import { readFileSync } from 'node:fs'
import { expect, test } from '@playwright/test'
import { loginAs } from './helpers'

test('export products, orders, customers and the dashboard as CSV', async ({ page }) => {
  const check = async (button: string, file: RegExp, header: string) => {
    const [download] = await Promise.all([page.waitForEvent('download'), page.getByRole('button', { name: button, exact: true }).click()])
    expect(download.suggestedFilename()).toMatch(file)
    const text = readFileSync((await download.path())!, 'utf8').replace(/^﻿/, '')
    expect(text.split(/\r?\n/)[0]).toContain(header)
    return text
  }

  await loginAs(page, 'viewer', '/products')
  await check('Export CSV', /^products-\d{4}-\d{2}-\d{2}\.csv$/, 'id,name,sku,category')
  await page.goto('/orders')
  await check('Export CSV', /^orders-.*\.csv$/, 'id,placed_at,status')
  await page.goto('/customers')
  await check('Export CSV', /^customers-.*\.csv$/, 'id,name,email')
  await page.goto('/')
  const dash = await check('Export', /^dashboard-30d-.*\.csv$/, 'Dashboard export')
  expect(dash).toContain('Net revenue')
})

test('import products from CSV, with row errors for a bad file', async ({ page }) => {
  await loginAs(page, 'editor', '/products')
  await page.getByRole('button', { name: 'Import CSV' }).click()
  const sheet = page.getByRole('dialog')

  await sheet.getByTestId('csv-input').setInputFiles({
    name: 'bad.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from('sku,name,category,price\nE2E-OK,Fine,Audio,10\nE2E-BAD,,Gadgets,abc\n'),
  })
  await sheet.getByRole('button', { name: 'Import', exact: true }).click()
  await expect(sheet.getByText(/nothing was imported/)).toBeVisible()
  await expect(sheet.getByText('Row 3').first()).toBeVisible()

  await sheet.getByTestId('csv-input').setInputFiles({
    name: 'good.csv',
    mimeType: 'text/csv',
    buffer: Buffer.from('sku,name,category,price,stock,status,tags\nE2E-LAMP-1,E2E Imported Lamp,lighting,49.90,12,active,led; desk\n'),
  })
  await sheet.getByRole('button', { name: 'Import', exact: true }).click()
  await expect(sheet.getByText('1 created · 0 updated')).toBeVisible()
  await sheet.getByRole('button', { name: 'Done' }).click()

  await page.getByPlaceholder('Search name or SKU…').fill('E2E-LAMP-1')
  await expect(page.getByText('E2E Imported Lamp')).toBeVisible()
})

test('changing the log level applies to the server', async ({ page }) => {
  await loginAs(page, 'owner', '/settings')
  const level = async () => (await (await page.request.get('/api/v1/settings/log-level')).json()).level
  const initial = await level()

  await page.getByRole('radio', { name: 'error' }).click()
  await expect(page.getByText('Log level set to error')).toBeVisible()
  expect(await level()).toBe('error')
  await expect(page.getByText(/Only server errors/)).toBeVisible()

  await page.getByRole('radio', { name: initial }).click()
  await expect.poll(level).toBe(initial)
})
