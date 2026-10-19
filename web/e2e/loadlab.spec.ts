import { expect, test } from '@playwright/test'

import { expectNoSeriousA11yViolations, signInAsOperator } from './helpers'

test('a short run completes and its money checks out', async ({ page }) => {
  await signInAsOperator(page)
  await page.getByRole('link', { name: 'Load lab' }).click()
  await expect(page.getByRole('heading', { name: 'Load lab' })).toBeVisible()
  await expectNoSeriousA11yViolations(page)

  const form = page.getByRole('form', { name: 'Start a run' })
  await form.getByLabel('Rate (req/s)').fill('100')
  await form.getByLabel('Duration (s)').fill('15')
  await form.getByLabel('Wallets').fill('5')
  await form.getByRole('button', { name: 'Start run' }).click()

  const status = page.getByRole('status', { name: 'Verification' })
  await expect(status).toBeVisible({ timeout: 60_000 })
  await expect(status).toContainText('Consistent')
})
