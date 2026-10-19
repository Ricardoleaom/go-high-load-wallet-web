import { expect, test } from '@playwright/test'

import { expectNoSeriousA11yViolations, signInAsOperator } from './helpers'

test('opening a wallet, seeing its detail and reconciling it', async ({ page }) => {
  await signInAsOperator(page)
  await page.getByRole('link', { name: 'Wallets' }).click()
  await expect(page.getByRole('heading', { name: 'Wallets', exact: true })).toBeVisible()
  await expectNoSeriousA11yViolations(page)

  await page.getByRole('button', { name: 'Open wallet' }).click()
  await expect(page.getByRole('heading', { name: /Wallet / })).toBeVisible({ timeout: 10_000 })
  await expectNoSeriousA11yViolations(page)

  await page.getByRole('button', { name: 'Reconcile' }).click()
  const status = page.getByRole('status', { name: 'Reconciliation' })
  await expect(status).toBeVisible()
  await expect(status).toContainText('Consistent')
})
