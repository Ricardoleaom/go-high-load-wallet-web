import { expect, test } from '@playwright/test'

import { expectNoSeriousA11yViolations, signInAsOperator } from './helpers'

test('following a ledger entry to its transaction', async ({ page }) => {
  await signInAsOperator(page)
  await page.getByRole('link', { name: 'Wallets' }).click()
  await page.getByRole('button', { name: 'Open wallet' }).click()
  await expect(page.getByRole('heading', { name: /Wallet / })).toBeVisible({ timeout: 10_000 })

  // The opening entry links to its own transaction; following it lands on
  // Transactions with that ID already searched (the page reads it from
  // the address, which is how the ledger always links to a transaction).
  const ledger = page.getByRole('table', { name: 'Ledger' })
  await ledger.getByRole('link').first().click()

  await expect(page.getByRole('heading', { name: 'Transactions' })).toBeVisible()
  await expectNoSeriousA11yViolations(page)
  const result = page.getByRole('region', { name: /Transaction / })
  await expect(result).toBeVisible()
  await expect(result).toContainText('OPENING')
})
