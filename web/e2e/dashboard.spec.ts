import { expect, test } from '@playwright/test'

import { expectNoSeriousA11yViolations, signInAsOperator } from './helpers'

test('the dashboard fills in with points once traffic flows', async ({ page }) => {
  await signInAsOperator(page)
  await expect(page.getByRole('heading', { name: 'Dashboard' })).toBeVisible()

  // Generate at least one request so the summary has something to report;
  // the first point needs two sampling intervals (10 s by default).
  await page.getByRole('link', { name: 'Wallets' }).click()
  await page.getByRole('button', { name: 'Open wallet' }).click()
  await expect(page.getByRole('heading', { name: /Wallet / })).toBeVisible({ timeout: 10_000 })

  await page.getByRole('link', { name: 'Dashboard' }).click()
  await expect(page.getByRole('region', { name: 'Right now' })).toBeVisible({ timeout: 20_000 })
  await expectNoSeriousA11yViolations(page)
})
