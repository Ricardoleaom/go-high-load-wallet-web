import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import { App } from './App'
import { signedOut, WithSession } from './test/session'

describe('App', () => {
  it('renders the sign-in screen when signed out', async () => {
    const session = signedOut()
    render(
      <WithSession session={session}>
        <App />
      </WithSession>,
    )

    expect(screen.getByRole('heading', { name: 'Wallet console' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Sign in' }))
    expect(session.signIn).toHaveBeenCalledOnce()
  })
})
