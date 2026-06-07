export interface Money {
  formatted: string;
  value: number;
}

export interface Merchant {
  name: string;
  logo?: string;
}

export interface Transaction {
  id: string;
  type: 'DEBIT' | 'CREDIT';
  amount: Money;
  description: string;
  merchant?: Merchant;
  createdAt: string;
  status: 'PENDING' | 'COMPLETED' | 'FAILED' | 'CANCELLED';
}

export interface TransactionEdge {
  node: Transaction;
  cursor: string;
}

export interface PageInfo {
  hasNextPage: boolean;
  endCursor?: string;
}

export interface TransactionConnection {
  edges: TransactionEdge[];
  pageInfo: PageInfo;
  totalCount: number;
}

export interface DashboardMetrics {
  totalBalance: Money;
  monthlyIncome: Money;
  monthlyExpenses: Money;
  savingsRate: number;
  transactionCount: number;
  pendingCount: number;
}

export interface FraudAlert {
  id: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  description: string;
  amount: Money;
  createdAt: string;
}

export interface SecurityMetrics {
  failedLogins: number;
  blockedTransactions: number;
  fraudAlerts: number;
  activeSecurityEvents: number;
  mfaAdoptionRate: number;
  suspiciousActivityDetected: number;
}

export interface SecurityEvent {
  eventId: string;
  eventType: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  userId: string;
  details: Record<string, any>;
  createdAt: string;
}

export interface FraudViolation {
  transactionId: string;
  ruleName: string;
  severity: 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';
  amount: Money;
  createdAt: string;
}

export interface PaginationInput {
  limit: number;
  offset: number;
}