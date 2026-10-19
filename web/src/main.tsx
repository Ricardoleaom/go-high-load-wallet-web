import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { App } from './App'
import { OidcSession } from './auth/OidcSession'
import './index.css'

const root = document.getElementById('root')
if (!root) {
  throw new Error('index.html has no #root element')
}

createRoot(root).render(
  <StrictMode>
    <OidcSession>
      <App />
    </OidcSession>
  </StrictMode>,
)
