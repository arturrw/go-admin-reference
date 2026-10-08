import { expect, test } from '@playwright/test'
import { loginAs, PASSWORD, USERS } from './helpers'

test('owner opens a member, sees presence and activity, and grants a permission', async ({ page, browser }) => {
  // Jon (viewer) is active in another browser, so he shows as online.
  const other = await browser.newContext()
  const res = await other.request.post('/api/v1/auth/login', { data: { email: USERS.viewer, password: PASSWORD } })
  expect(res.ok()).toBeTruthy()

  await loginAs(page, 'owner', '/team')
  await page.getByRole('button', { name: 'Open Jon Berg' }).click()
  const sheet = page.getByRole('dialog', { name: 'Jon Berg' })
  await expect(sheet.getByTestId('presence')).toHaveText(/Online now/)
  await expect(sheet.getByText('Two-factor auth')).toBeVisible()

  await sheet.getByRole('tab', { name: 'Activity' }).click()
  await expect(sheet.getByTestId('activity-item').first()).toContainText('Jon Berg signed in')

  await sheet.getByRole('tab', { name: /Access/ }).click()
  const access = sheet.getByTestId('access')
  await access.getByRole('switch', { name: 'Manage orders' }).click()
  await access.getByRole('switch', { name: 'View customers' }).click()
  await expect(sheet.getByText('1 granted · 1 revoked vs viewer')).toBeVisible()
  await sheet.getByRole('button', { name: 'Save access' }).click()
  await expect(page.getByText('Access updated for Jon Berg')).toBeVisible()
  await expect(access.getByText('granted', { exact: true })).toBeVisible()
  await expect(access.getByText('revoked', { exact: true })).toBeVisible()

  // Jon's own session picks up the new permissions.
  const me = await (await other.request.get('/api/v1/auth/me')).json()
  expect(me.permissions).toContain('orders:write')
  expect(me.permissions).not.toContain('customers:read')
  expect((await other.request.get('/api/v1/customers')).status()).toBe(403)

  // Back to role defaults.
  await sheet.getByRole('button', { name: 'Role defaults' }).click()
  await sheet.getByRole('button', { name: 'Save access' }).click()
  await expect(sheet.getByText('Same as the viewer role')).toBeVisible()
  await other.close()
})

test('suspending a member asks for confirmation and signs them out', async ({ page, browser }) => {
  const other = await browser.newContext()
  await other.request.post('/api/v1/auth/login', { data: { email: USERS.support, password: PASSWORD } })

  await loginAs(page, 'owner', '/team')
  await page.getByRole('button', { name: 'Open Priya Shah' }).click()
  const sheet = page.getByRole('dialog', { name: 'Priya Shah' })
  await sheet.getByRole('button', { name: 'Suspend' }).click()
  await page.getByRole('dialog', { name: 'Suspend Priya Shah?' }).getByRole('button', { name: 'Suspend' }).click()
  await expect(page.getByText('Suspended Priya Shah')).toBeVisible()
  expect((await other.request.get('/api/v1/auth/me')).status()).toBe(401)

  await sheet.getByRole('button', { name: 'Reactivate' }).click()
  await expect(page.getByText('Reactivated Priya Shah')).toBeVisible()
  await other.close()
})

test('admins see member details but only the owner edits individual permissions', async ({ page }) => {
  await loginAs(page, 'admin', '/team')
  await page.getByRole('button', { name: 'Open Yuki Tanaka' }).click()
  const sheet = page.getByRole('dialog', { name: 'Yuki Tanaka' })
  await sheet.getByRole('tab', { name: /Access/ }).click()
  await expect(sheet.getByText('Only the owner can change individual permissions.')).toBeVisible()
  await expect(sheet.getByRole('switch')).toHaveCount(0)
})

test('an invited member opens the link, picks a password and joins', async ({ page, browser }) => {
  await loginAs(page, 'owner', '/team')
  await page.getByRole('button', { name: 'Invite member' }).click()
  const form = page.getByRole('dialog', { name: 'Invite member' })
  await form.getByLabel('Full name').fill('Nina Hartmann')
  await form.getByLabel('Email').fill('nina.hartmann@acme.io')
  await form.getByRole('radio', { name: /editor/ }).click()
  await form.getByRole('button', { name: 'Create invitation' }).click()

  // The link is shown once; the email can't be sent from here.
  const link = page.getByRole('dialog', { name: 'Invitation for Nina Hartmann' })
  const url = (await link.getByTestId('invite-url').innerText()).trim()
  expect(url).toMatch(/\/invite\/[\w-]{20,}$/)
  await expect(link).toContainText("doesn't send email")
  await link.getByRole('button', { name: 'Done' }).click()
  await expect(page.locator('tr', { hasText: 'nina.hartmann@acme.io' })).toContainText('invited')

  // The invitee, on another browser, follows the link.
  const guest = await browser.newContext()
  const gp = await guest.newPage()
  await gp.goto(url)
  await expect(gp.getByText("You've been invited to join")).toBeVisible()
  await expect(gp.getByText('nina.hartmann@acme.io')).toBeVisible()
  await expect(gp.getByLabel('Your name')).toHaveValue('Nina Hartmann')
  const create = gp.getByRole('button', { name: 'Create account and sign in' })
  await expect(create).toBeDisabled()
  await gp.getByLabel('Choose a password').fill('short')
  await expect(create).toBeDisabled() // under 8 characters
  await gp.getByLabel('Choose a password').fill('correct horse battery')
  await gp.getByLabel('Repeat the password').fill('correct horse battery staple')
  await expect(gp.getByText('The passwords do not match.')).toBeVisible()
  await gp.getByLabel('Repeat the password').fill('correct horse battery')
  await create.click()

  // They land in the app with an editor's access.
  await expect(gp).toHaveURL(/\/$/)
  await expect(gp.getByTestId('current-user')).toHaveText('Nina Hartmann')
  await expect(gp.locator('aside nav').getByRole('link', { name: 'Products' })).toBeVisible()
  await expect(gp.locator('aside nav').getByRole('link', { name: 'Request log' })).toHaveCount(0)

  // The link is spent, and the owner sees the member as active.
  await gp.goto(url)
  await expect(gp.getByTestId('invite-invalid')).toBeVisible()
  await guest.close()
  await page.reload()
  await expect(page.locator('tr', { hasText: 'nina.hartmann@acme.io' })).not.toContainText('invited')
})

test('a replaced invitation link stops working', async ({ page, browser }) => {
  await loginAs(page, 'admin', '/team')
  // Sofia was invited in the demo data. Issue her a link, then another.
  await page.getByRole('button', { name: 'New invitation link for Sofia Rossi' }).click()
  const first = (await page.getByRole('dialog', { name: 'Invitation for Sofia Rossi' }).getByTestId('invite-url').innerText()).trim()
  await page.getByRole('dialog', { name: 'Invitation for Sofia Rossi' }).getByRole('button', { name: 'Done' }).click()
  await page.getByRole('button', { name: 'New invitation link for Sofia Rossi' }).click()
  const second = (await page.getByRole('dialog', { name: 'Invitation for Sofia Rossi' }).getByTestId('invite-url').innerText()).trim()
  expect(second).not.toBe(first)

  const guest = await browser.newContext()
  const gp = await guest.newPage()
  await gp.goto(first)
  await expect(gp.getByTestId('invite-invalid')).toBeVisible()
  await gp.goto(second)
  await expect(gp.getByText('sofia@acme.io')).toBeVisible()
  await guest.close()
})