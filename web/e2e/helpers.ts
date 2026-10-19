import { AxeBuilder } from '@axe-core/playwright'
import { expect, type Page } from '@playwright/test'

// signInAsOperator drives the real PKCE flow against the backend's
// Keycloak with the realm's demo operator.
export async function signInAsOperator(page: Page): Promise<void> {
  await page.goto('/')
  await page.getByRole('button', { name: 'Sign in' }).click()
  await page.getByLabel('Username or email').fill('operator')
  await page.getByLabel('Password', { exact: true }).fill('operator')
  await page.getByRole('button', { name: 'Sign In' }).click()
  // The redirect back through the IdP can take longer than Playwright's
  // default 5 s under a busy machine.
  await expect(page.getByText('operator')).toBeVisible({ timeout: 20_000 })
}

// expectNoSeriousA11yViolations scans the current page with axe-core and
// fails on anything of serious or critical impact; moderate/minor issues
// are left for later polish rather than blocking the suite.
export async function expectNoSeriousA11yViolations(page: Page): Promise<void> {
  const results = await new AxeBuilder({ page }).withTags(['wcag2a', 'wcag2aa']).analyze()
  const serious = results.violations.filter((v) => v.impact === 'serious' || v.impact === 'critical')
  expect(serious, JSON.stringify(serious, null, 2)).toEqual([])
}
