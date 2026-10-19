import type { components } from './schema'

type Schemas = components['schemas']

export type Money = Schemas['Money']
export type Wallet = Schemas['Wallet']
export type LedgerEntry = Schemas['LedgerEntry']
export type LedgerPage = Schemas['LedgerPage']
export type Reconciliation = Schemas['Reconciliation']
export type Transaction = Schemas['Transaction']
export type TransactionStatus = Schemas['Status']
export type OpsSummary = Schemas['OpsSummary']
export type OpsPoint = Schemas['OpsPoint']
