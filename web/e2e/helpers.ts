import { expect, type Page } from '@playwright/test'

export const PASSWORD = 'goadmin'

export const USERS = {
  owner: 'artur@acme.io',
  admin: 'mark@acme.io',
  editor: 'yuki@acme.io',
  support: 'priya@acme.io',
  viewer: 'jon@acme.io',
} as const

/** Signs in through the API (fast path); page.request shares the browser's cookie jar. */
export async function loginAs(page: Page, role: keyof typeof USERS, path = '/') {
  const res = await page.request.post('/api/v1/auth/login', { data: { email: USERS[role], password: PASSWORD } })
  expect(res.ok()).toBeTruthy()
  await page.goto(path)
}

/** A small valid PNG generated in-browser-free: 2×2 red pixels. */
export const PNG = Buffer.from(
  'iVBORw0KGgoAAAANSUhEUgAAAAIAAAACCAIAAAD91JpzAAAAFklEQVR4nGP8z8DAwMDAxMDAwMDAAAANHQEDasKb6QAAAABJRU5ErkJggg==',
  'base64',
)
