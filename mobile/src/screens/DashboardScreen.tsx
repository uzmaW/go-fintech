import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  RefreshControl,
  Dimensions,
} from 'react-native';
import { useQuery, gql } from '@apollo/client';

const GET_DASHBOARD_METRICS = gql`
  query GetDashboardMetrics($timeRange: String!) {
    dashboardMetrics(timeRange: $timeRange) {
      totalBalance {
        formatted
        value
      }
      monthlyIncome {
        formatted
        value
      }
      monthlyExpenses {
        formatted
        value
      }
      savingsRate
      transactionCount
      pendingCount
    }
    recentTransactions(limit: 5) {
      id
      type
      amount {
        formatted
        value
      }
      description
      merchant {
        name
        logo
      }
      createdAt
      status
    }
    fraudAlerts(limit: 5) {
      id
      severity
      description
      amount {
        formatted
      }
      createdAt
    }
  }
`;

const GET_FRAUD_METRICS = gql`
  query GetSecurityMetrics($timeRange: String!) {
    securityMetrics(timeRange: $timeRange) {
      failedLogins
      blockedTransactions
      fraudAlerts
      activeSecurityEvents
      mfaAdoptionRate
      suspiciousActivityDetected
    }
  }
`;

interface DashboardScreenProps {
  navigation: any;
}

export const DashboardScreen: React.FC<DashboardScreenProps> = ({ navigation }) => {
  const [timeRange, setTimeRange] = useState('24h');
  const [refreshing, setRefreshing] = useState(false);

  const { data: metricsData, loading, refetch } = useQuery(GET_DASHBOARD_METRICS, {
    variables: { timeRange },
    pollInterval: 30000,
    notifyOnNetworkStatusChange: true,
  });

  const { data: fraudData } = useQuery(GET_FRAUD_METRICS, {
    variables: { timeRange },
    pollInterval: 30000,
  });

  useEffect(() => {
    const unsubscribe = navigation.addListener('focus', () => {
      refetch();
    });
    return unsubscribe;
  }, [navigation, refetch]);

  const handleRefresh = async () => {
    setRefreshing(true);
    await refetch();
    setRefreshing(false);
  };

  const metrics = metricsData?.dashboardMetrics;
  const recentTransactions = metricsData?.recentTransactions || [];
  const fraudAlerts = metricsData?.fraudAlerts || [];
  const securityMetrics = fraudData?.securityMetrics;

  const screenWidth = Dimensions.get('window').width;

  const renderMetricCard = (title: string, value: string, subtitle?: string, color?: string) => (
    <View style={[styles.metricCard, { width: (screenWidth - 48) / 2 }]}>
      <Text style={styles.metricTitle}>{title}</Text>
      <Text style={[styles.metricValue, color ? { color } : null]}>{value}</Text>
      {subtitle && <Text style={styles.metricSubtitle}>{subtitle}</Text>}
    </View>
  );

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.contentContainer}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} tintColor="#007AFF" />
      }
      showsVerticalScrollIndicator={false}
    >
      {/* Balance Card */}
      <View style={styles.balanceCard}>
        <Text style={styles.balanceLabel}>Total Balance</Text>
        <Text style={styles.balanceValue}>
          {metrics?.totalBalance?.formatted || '$0.00'}
        </Text>
        <View style={styles.balanceRow}>
          <View style={styles.balanceItem}>
            <Text style={styles.balanceItemLabel}>Income</Text>
            <Text style={[styles.balanceItemValue, { color: '#34C759' }]}>
              +{metrics?.monthlyIncome?.formatted || '$0.00'}
            </Text>
          </View>
          <View style={styles.balanceDivider} />
          <View style={styles.balanceItem}>
            <Text style={styles.balanceItemLabel}>Expenses</Text>
            <Text style={[styles.balanceItemValue, { color: '#FF3B30' }]}>
              -{metrics?.monthlyExpenses?.formatted || '$0.00'}
            </Text>
          </View>
        </View>
      </View>

      {/* Quick Actions */}
      <View style={styles.quickActions}>
        <TouchableOpacity
          style={styles.actionButton}
          onPress={() => navigation.navigate('SendMoney')}
          activeOpacity={0.7}
        >
          <View style={[styles.actionIcon, { backgroundColor: '#007AFF15' }]}>
            <Text style={styles.actionIconText}>$</Text>
          </View>
          <Text style={styles.actionLabel}>Send</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={styles.actionButton}
          onPress={() => navigation.navigate('AddMoney')}
          activeOpacity={0.7}
        >
          <View style={[styles.actionIcon, { backgroundColor: '#34C75915' }]}>
            <Text style={styles.actionIconText}>+</Text>
          </View>
          <Text style={styles.actionLabel}>Add</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={styles.actionButton}
          onPress={() => navigation.navigate('PayBills')}
          activeOpacity={0.7}
        >
          <View style={[styles.actionIcon, { backgroundColor: '#FF950015' }]}>
            <Text style={styles.actionIconText}>!</Text>
          </View>
          <Text style={styles.actionLabel}>Pay</Text>
        </TouchableOpacity>
        <TouchableOpacity
          style={styles.actionButton}
          onPress={() => navigation.navigate('More')}
          activeOpacity={0.7}
        >
          <View style={[styles.actionIcon, { backgroundColor: '#8E8E9315' }]}>
            <Text style={styles.actionIconText}>...</Text>
          </View>
          <Text style={styles.actionLabel}>More</Text>
        </TouchableOpacity>
      </View>

      {/* Metrics Grid */}
      <View style={styles.metricsGrid}>
        {renderMetricCard(
          'Transaction Count',
          metrics?.transactionCount?.toString() || '0',
          'This month'
        )}
        {renderMetricCard(
          'Pending',
          metrics?.pendingCount?.toString() || '0',
          'Awaiting approval'
        )}
        {renderMetricCard(
          'Savings Rate',
          `${metrics?.savingsRate || 0}%`,
          undefined,
          '#007AFF'
        )}
        {renderMetricCard(
          'MFA Adoption',
          `${securityMetrics?.mfaAdoptionRate || 0}%`,
          'Security score',
          '#34C759'
        )}
      </View>

      {/* Fraud Alerts Section */}
      {fraudAlerts.length > 0 && (
        <View style={styles.section}>
          <View style={styles.sectionHeader}>
            <Text style={styles.sectionTitle}>Fraud Alerts</Text>
            <TouchableOpacity onPress={() => navigation.navigate('Security')}>
              <Text style={styles.sectionLink}>View All</Text>
            </TouchableOpacity>
          </View>
          <View style={styles.alertContainer}>
            {fraudAlerts.map((alert: any) => (
              <View
                key={alert.id}
                style={[
                  styles.alertCard,
                  alert.severity === 'CRITICAL' && styles.alertCritical,
                  alert.severity === 'HIGH' && styles.alertHigh,
                ]}
              >
                <View style={styles.alertIcon}>
                  <Text style={styles.alertIconText}>!</Text>
                </View>
                <View style={styles.alertContent}>
                  <Text style={styles.alertDescription}>{alert.description}</Text>
                  <Text style={styles.alertAmount}>{alert.amount?.formatted}</Text>
                  <Text style={styles.alertTime}>
                    {new Date(alert.createdAt).toLocaleDateString()}
                  </Text>
                </View>
              </View>
            ))}
          </View>
        </View>
      )}

      {/* Security Metrics */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Security Overview</Text>
        <View style={styles.securityGrid}>
          <View style={styles.securityItem}>
            <Text style={styles.securityValue}>
              {securityMetrics?.failedLogins || 0}
            </Text>
            <Text style={styles.securityLabel}>Failed Logins</Text>
          </View>
          <View style={styles.securityItem}>
            <Text style={[styles.securityValue, { color: '#FF3B30' }]}>
              {securityMetrics?.blockedTransactions || 0}
            </Text>
            <Text style={styles.securityLabel}>Blocked Txns</Text>
          </View>
          <View style={styles.securityItem}>
            <Text style={[styles.securityValue, { color: '#FF9500' }]}>
              {securityMetrics?.fraudAlerts || 0}
            </Text>
            <Text style={styles.securityLabel}>Fraud Alerts</Text>
          </View>
          <View style={styles.securityItem}>
            <Text style={[styles.securityValue, { color: '#8E8E93' }]}>
              {securityMetrics?.activeSecurityEvents || 0}
            </Text>
            <Text style={styles.securityLabel}>Active Events</Text>
          </View>
        </View>
      </View>

      {/* Recent Transactions */}
      <View style={styles.section}>
        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Recent Transactions</Text>
          <TouchableOpacity onPress={() => navigation.navigate('Transactions')}>
            <Text style={styles.sectionLink}>See All</Text>
          </TouchableOpacity>
        </View>
        {recentTransactions.length > 0 ? (
          recentTransactions.map((tx: any) => (
            <TouchableOpacity
              key={tx.id}
              style={styles.recentTransaction}
              onPress={() => navigation.navigate('TransactionDetail', { id: tx.id })}
              activeOpacity={0.7}
            >
              <View style={styles.recentTxIcon}>
                <Text style={styles.recentTxInitial}>
                  {tx.merchant?.name?.charAt(0) || 'T'}
                </Text>
              </View>
              <View style={styles.recentTxDetails}>
                <Text style={styles.recentTxMerchant}>
                  {tx.merchant?.name || 'Transfer'}
                </Text>
                <Text style={styles.recentTxDesc} numberOfLines={1}>
                  {tx.description}
                </Text>
              </View>
              <Text
                style={[
                  styles.recentTxAmount,
                  tx.type === 'DEBIT' ? styles.debitText : styles.creditText,
                ]}
              >
                {tx.type === 'DEBIT' ? '-' : '+'}{tx.amount?.formatted}
              </Text>
            </TouchableOpacity>
          ))
        ) : (
          <Text style={styles.emptyText}>No recent transactions</Text>
        )}
      </View>

      {/* Time Range Selector */}
      <View style={styles.timeRangeSelector}>
        {['1h', '24h', '7d', '30d'].map((range) => (
          <TouchableOpacity
            key={range}
            style={[
              styles.timeRangeButton,
              timeRange === range && styles.timeRangeActive,
            ]}
            onPress={() => setTimeRange(range)}
          >
            <Text
              style={[
                styles.timeRangeText,
                timeRange === range && styles.timeRangeTextActive,
              ]}
            >
              {range}
            </Text>
          </TouchableOpacity>
        ))}
      </View>
    </ScrollView>
  );
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F2F2F7',
  },
  contentContainer: {
    padding: 16,
    paddingBottom: 32,
  },
  balanceCard: {
    backgroundColor: '#007AFF',
    borderRadius: 16,
    padding: 24,
    marginBottom: 16,
  },
  balanceLabel: {
    color: 'rgba(255, 255, 255, 0.8)',
    fontSize: 14,
    fontWeight: '500',
  },
  balanceValue: {
    color: '#FFFFFF',
    fontSize: 36,
    fontWeight: '700',
    marginTop: 8,
  },
  balanceRow: {
    flexDirection: 'row',
    marginTop: 24,
  },
  balanceItem: {
    flex: 1,
  },
  balanceItemLabel: {
    color: 'rgba(255, 255, 255, 0.6)',
    fontSize: 12,
  },
  balanceItemValue: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
    marginTop: 4,
  },
  balanceDivider: {
    width: 1,
    backgroundColor: 'rgba(255, 255, 255, 0.2)',
    marginHorizontal: 16,
  },
  quickActions: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    marginBottom: 16,
  },
  actionButton: {
    alignItems: 'center',
  },
  actionIcon: {
    width: 48,
    height: 48,
    borderRadius: 24,
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: 8,
  },
  actionIconText: {
    fontSize: 20,
    fontWeight: '600',
    color: '#007AFF',
  },
  actionLabel: {
    fontSize: 12,
    fontWeight: '500',
    color: '#000000',
  },
  metricsGrid: {
    flexDirection: 'row',
    flexWrap: 'wrap',
    justifyContent: 'space-between',
    marginBottom: 24,
  },
  metricCard: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    marginBottom: 12,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 2,
  },
  metricTitle: {
    fontSize: 12,
    color: '#8E8E93',
    fontWeight: '500',
  },
  metricValue: {
    fontSize: 24,
    fontWeight: '700',
    color: '#000000',
    marginTop: 4,
  },
  metricSubtitle: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 4,
  },
  section: {
    marginBottom: 24,
  },
  sectionHeader: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#000000',
  },
  sectionLink: {
    fontSize: 14,
    color: '#007AFF',
    fontWeight: '500',
  },
  alertContainer: {
    gap: 8,
  },
  alertCard: {
    flexDirection: 'row',
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    marginBottom: 8,
    borderLeftWidth: 4,
    borderLeftColor: '#FF9500',
  },
  alertCritical: {
    borderLeftColor: '#FF3B30',
    backgroundColor: '#FF3B3010',
  },
  alertHigh: {
    borderLeftColor: '#FF9500',
  },
  alertIcon: {
    width: 32,
    height: 32,
    borderRadius: 16,
    backgroundColor: '#FF3B3020',
    justifyContent: 'center',
    alignItems: 'center',
    marginRight: 12,
  },
  alertIconText: {
    color: '#FF3B30',
    fontSize: 16,
    fontWeight: '700',
  },
  alertContent: {
    flex: 1,
  },
  alertDescription: {
    fontSize: 14,
    fontWeight: '500',
    color: '#000000',
  },
  alertAmount: {
    fontSize: 14,
    fontWeight: '600',
    color: '#FF3B30',
    marginTop: 4,
  },
  alertTime: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 4,
  },
  securityGrid: {
    flexDirection: 'row',
    justifyContent: 'space-between',
  },
  securityItem: {
    alignItems: 'center',
    flex: 1,
    paddingVertical: 16,
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    marginHorizontal: 4,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 2,
  },
  securityValue: {
    fontSize: 20,
    fontWeight: '700',
    color: '#000000',
  },
  securityLabel: {
    fontSize: 10,
    color: '#8E8E93',
    marginTop: 4,
    textAlign: 'center',
  },
  recentTransaction: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    marginBottom: 8,
  },
  recentTxIcon: {
    width: 40,
    height: 40,
    borderRadius: 20,
    backgroundColor: '#007AFF',
    justifyContent: 'center',
    alignItems: 'center',
  },
  recentTxInitial: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
  recentTxDetails: {
    flex: 1,
    marginLeft: 12,
  },
  recentTxMerchant: {
    fontSize: 14,
    fontWeight: '600',
    color: '#000000',
  },
  recentTxDesc: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 2,
  },
  recentTxAmount: {
    fontSize: 14,
    fontWeight: '600',
  },
  debitText: {
    color: '#FF3B30',
  },
  creditText: {
    color: '#34C759',
  },
  emptyText: {
    fontSize: 14,
    color: '#8E8E93',
    textAlign: 'center',
    paddingVertical: 32,
  },
  timeRangeSelector: {
    flexDirection: 'row',
    justifyContent: 'center',
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 4,
    marginTop: 8,
  },
  timeRangeButton: {
    flex: 1,
    paddingVertical: 12,
    alignItems: 'center',
    borderRadius: 8,
  },
  timeRangeActive: {
    backgroundColor: '#007AFF',
  },
  timeRangeText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#8E8E93',
  },
  timeRangeTextActive: {
    color: '#FFFFFF',
  },
});