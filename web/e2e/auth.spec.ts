import { expect, test } from '@playwright/test'

import { expectNoSeriousA11yViolations, signInAsOperator } from './helpers'

test('an operator signs in, sees the console, and signs out', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: 'Wallet console' })).toBeVisible()
  await expectNoSeriousA11yViolations(page)

  await signInAsOperator(page)
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()
  await expectNoSeriousA11yViolations(page)

  await page.getByRole('button', { name: 'Sign out' }).click()
  await expect(page.getByRole('button', { name: 'Sign in' })).toBeVisible()
})
