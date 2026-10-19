import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router'
import { describe, expect, it } from 'vitest'

import { signedIn, WithSession } from '../test/session'
import { Layout } from './Layout'

describe('Layout', () => {
  it('names the operator, links every section and signs out', async () => {
    const session = signedIn()
    render(
      <WithSession session={session}>
        <MemoryRouter>
          <Layout />
        </MemoryRouter>
      </WithSession>,
    )

    const nav = screen.getByRole('navigation', { name: 'Sections' })
    for (const name of ['Dashboard', 'Wallets', 'Transactions', 'Load lab']) {
      expect(nav).toContainElement(screen.getByRole('link', { name }))
    }
    expect(screen.getByText('operator')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Sign out' }))
    expect(session.signOut).toHaveBeenCalledOnce()
  })
})
