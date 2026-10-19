import { Route, Routes } from 'react-router'

import { WalletDetail } from './WalletDetail'
import { WalletsHome } from './WalletsHome'

export function WalletsPage() {
  return (
    <Routes>
      <Route index element={<WalletsHome />} />
      <Route path=":id" element={<WalletDetail />} />
    </Routes>
  )
}
