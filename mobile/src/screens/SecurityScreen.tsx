import React, { useState } from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  RefreshControl,
  Switch,
  Alert,
} from 'react-native';
import { useQuery, gql } from '@apollo/client';

const GET_SECURITY_METRICS = gql`
  query GetSecurityMetrics($timeRange: String!) {
    securityMetrics(timeRange: $timeRange) {
      failedLogins
      blockedTransactions
      fraudAlerts
      activeSecurityEvents
      mfaAdoptionRate
      suspiciousActivityDetected
    }
    securityEvents(limit: 50, unresolved: true) {
      eventId
      eventType
      severity
      userId
      details
      createdAt
    }
    fraudViolations(limit: 20) {
      transactionId
      ruleName
      severity
      amount {
        formatted
        value
      }
      createdAt
    }
  }
`;

interface SecurityScreenProps {
  navigation: any;
}

export const SecurityScreen: React.FC<SecurityScreenProps> = ({ navigation }) => {
  const [timeRange, setTimeRange] = useState('24h');
  const [refreshing, setRefreshing] = useState(false);
  const [mfaEnabled, setMfaEnabled] = useState(true);
  const [biometricEnabled, setBiometricEnabled] = useState(true);

  const { data, loading, refetch } = useQuery(GET_SECURITY_METRICS, {
    variables: { timeRange },
    pollInterval: 30000,
    notifyOnNetworkStatusChange: true,
  });

  const handleRefresh = async () => {
    setRefreshing(true);
    await refetch();
    setRefreshing(false);
  };

  const handleResolveEvent = (eventId: string) => {
    Alert.alert(
      'Resolve Event',
      'Are you sure you want to mark this event as resolved?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Resolve',
          style: 'default',
          onPress: () => {
            console.log(`Resolving event ${eventId}`);
          },
        },
      ]
    );
  };

  const handleReviewTransaction = (transactionId: string) => {
    navigation.navigate('TransactionDetail', { id: transactionId });
  };

  const securityMetrics = data?.securityMetrics;
  const securityEvents = data?.securityEvents || [];
  const fraudViolations = data?.fraudViolations || [];

  const getSeverityColor = (severity: string) => {
    switch (severity) {
      case 'CRITICAL':
        return '#FF3B30';
      case 'HIGH':
        return '#FF9500';
      case 'MEDIUM':
        return '#007AFF';
      default:
        return '#8E8E93';
    }
  };

  const getSeverityBgColor = (severity: string) => {
    switch (severity) {
      case 'CRITICAL':
        return '#FF3B3015';
      case 'HIGH':
        return '#FF950015';
      case 'MEDIUM':
        return '#007AFF15';
      default:
        return '#8E8E9315';
    }
  };

  return (
    <ScrollView
      style={styles.container}
      contentContainerStyle={styles.contentContainer}
      refreshControl={
        <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} tintColor="#007AFF" />
      }
      showsVerticalScrollIndicator={false}
    >
      {/* Security Settings */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Security Settings</Text>
        <View style={styles.settingsCard}>
          <View style={styles.settingRow}>
            <View style={styles.settingInfo}>
              <Text style={styles.settingLabel}>Two-Factor Authentication</Text>
              <Text style={styles.settingDescription}>
                Require 2FA for all transactions
              </Text>
            </View>
            <Switch
              value={mfaEnabled}
              onValueChange={setMfaEnabled}
              trackColor={{ false: '#E5E5EA', true: '#34C759' }}
              thumbColor="#FFFFFF"
            />
          </View>
          <View style={styles.settingDivider} />
          <View style={styles.settingRow}>
            <View style={styles.settingInfo}>
              <Text style={styles.settingLabel}>Biometric Login</Text>
              <Text style={styles.settingDescription}>
                Use Face ID or fingerprint
              </Text>
            </View>
            <Switch
              value={biometricEnabled}
              onValueChange={setBiometricEnabled}
              trackColor={{ false: '#E5E5EA', true: '#34C759' }}
              thumbColor="#FFFFFF"
            />
          </View>
        </View>
      </View>

      {/* Security Overview */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Security Overview</Text>
        <View style={styles.metricsRow}>
          <View style={styles.metricCard}>
            <Text style={[styles.metricValue, { color: '#FF3B30' }]}>
              {securityMetrics?.failedLogins || 0}
            </Text>
            <Text style={styles.metricLabel}>Failed Logins</Text>
          </View>
          <View style={styles.metricCard}>
            <Text style={[styles.metricValue, { color: '#FF9500' }]}>
              {securityMetrics?.blockedTransactions || 0}
            </Text>
            <Text style={styles.metricLabel}>Blocked Txns</Text>
          </View>
        </View>
        <View style={styles.metricsRow}>
          <View style={styles.metricCard}>
            <Text style={[styles.metricValue, { color: '#FF3B30' }]}>
              {securityMetrics?.fraudAlerts || 0}
            </Text>
            <Text style={styles.metricLabel}>Fraud Alerts</Text>
          </View>
          <View style={styles.metricCard}>
            <Text style={styles.metricValue}>
              {securityMetrics?.activeSecurityEvents || 0}
            </Text>
            <Text style={styles.metricLabel}>Active Events</Text>
          </View>
        </View>
        <View style={styles.mfaCard}>
          <View style={styles.mfaInfo}>
            <Text style={styles.mfaTitle}>MFA Adoption Rate</Text>
            <Text style={styles.mfaDescription}>
              {securityMetrics?.mfaAdoptionRate || 0}% of users have enabled 2FA
            </Text>
          </View>
          <View style={styles.mfaProgress}>
            <View
              style={[
                styles.mfaProgressBar,
                { width: `${securityMetrics?.mfaAdoptionRate || 0}%` },
              ]}
            />
          </View>
        </View>
      </View>

      {/* Fraud Violations */}
      <View style={styles.section}>
        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Fraud Violations</Text>
          <TouchableOpacity onPress={() => navigation.navigate('FraudDashboard')}>
            <Text style={styles.sectionLink}>View All</Text>
          </TouchableOpacity>
        </View>
        {fraudViolations.length > 0 ? (
          fraudViolations.map((violation: any) => (
            <TouchableOpacity
              key={violation.transactionId}
              style={styles.violationCard}
              onPress={() => handleReviewTransaction(violation.transactionId)}
              activeOpacity={0.7}
            >
              <View
                style={[
                  styles.violationSeverity,
                  {
                    backgroundColor: getSeverityBgColor(violation.severity),
                    borderColor: getSeverityColor(violation.severity),
                  },
                ]}
              >
                <Text
                  style={[
                    styles.violationSeverityText,
                    { color: getSeverityColor(violation.severity) },
                  ]}
                >
                  {violation.severity}
                </Text>
              </View>
              <View style={styles.violationContent}>
                <Text style={styles.violationRule}>{violation.ruleName}</Text>
                <Text style={styles.violationAmount}>
                  {violation.amount?.formatted || '$0.00'}
                </Text>
                <Text style={styles.violationTime}>
                  {new Date(violation.createdAt).toLocaleDateString('en-US', {
                    month: 'short',
                    day: 'numeric',
                    hour: '2-digit',
                    minute: '2-digit',
                  })}
                </Text>
              </View>
              <View style={styles.violationArrow}>
                <Text style={styles.violationArrowText}>›</Text>
              </View>
            </TouchableOpacity>
          ))
        ) : (
          <View style={styles.emptyCard}>
            <Text style={styles.emptyText}>No fraud violations detected</Text>
            <Text style={styles.emptySubtext}>Your transactions are secure</Text>
          </View>
        )}
      </View>

      {/* Active Security Events */}
      <View style={styles.section}>
        <View style={styles.sectionHeader}>
          <Text style={styles.sectionTitle}>Active Security Events</Text>
          <TouchableOpacity onPress={() => navigation.navigate('AllSecurityEvents')}>
            <Text style={styles.sectionLink}>View All</Text>
          </TouchableOpacity>
        </View>
        {securityEvents.length > 0 ? (
          securityEvents.map((event: any) => (
            <View key={event.eventId} style={styles.eventCard}>
              <View style={styles.eventHeader}>
                <View
                  style={[
                    styles.eventSeverity,
                    {
                      backgroundColor: getSeverityBgColor(event.severity),
                      borderColor: getSeverityColor(event.severity),
                    },
                  ]}
                >
                  <Text
                    style={[
                      styles.eventSeverityText,
                      { color: getSeverityColor(event.severity) },
                    ]}
                  >
                    {event.severity}
                  </Text>
                </View>
                <Text style={styles.eventType}>{event.eventType}</Text>
              </View>
              <Text style={styles.eventDetails} numberOfLines={2}>
                {JSON.stringify(event.details)}
              </Text>
              <View style={styles.eventFooter}>
                <Text style={styles.eventTime}>
                  {new Date(event.createdAt).toLocaleDateString()}
                </Text>
                <TouchableOpacity
                  style={styles.resolveButton}
                  onPress={() => handleResolveEvent(event.eventId)}
                >
                  <Text style={styles.resolveButtonText}>Resolve</Text>
                </TouchableOpacity>
              </View>
            </View>
          ))
        ) : (
          <View style={styles.emptyCard}>
            <Text style={styles.emptyText}>No active security events</Text>
            <Text style={styles.emptySubtext}>All systems operating normally</Text>
          </View>
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

      {/* Security Tips */}
      <View style={styles.tipsCard}>
        <Text style={styles.tipsTitle}>Security Tips</Text>
        <View style={styles.tipRow}>
          <Text style={styles.tipIcon}>🔒</Text>
          <Text style={styles.tipText}>
            Keep your account secure by enabling two-factor authentication
          </Text>
        </View>
        <View style={styles.tipRow}>
          <Text style={styles.tipIcon}>📱</Text>
          <Text style={styles.tipText}>
            Never share your verification codes with anyone
          </Text>
        </View>
        <View style={styles.tipRow}>
          <Text style={styles.tipIcon}>⚠️</Text>
          <Text style={styles.tipText}>
            Report suspicious activity immediately
          </Text>
        </View>
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
    marginBottom: 12,
  },
  sectionLink: {
    fontSize: 14,
    color: '#007AFF',
    fontWeight: '500',
  },
  settingsCard: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 2,
  },
  settingRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingVertical: 8,
  },
  settingInfo: {
    flex: 1,
    marginRight: 16,
  },
  settingLabel: {
    fontSize: 16,
    fontWeight: '500',
    color: '#000000',
  },
  settingDescription: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 2,
  },
  settingDivider: {
    height: 1,
    backgroundColor: '#E5E5EA',
    marginVertical: 8,
  },
  metricsRow: {
    flexDirection: 'row',
    gap: 12,
    marginBottom: 12,
  },
  metricCard: {
    flex: 1,
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 2,
  },
  metricValue: {
    fontSize: 28,
    fontWeight: '700',
    color: '#000000',
  },
  metricLabel: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 4,
  },
  mfaCard: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 2,
  },
  mfaInfo: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: 12,
  },
  mfaTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#000000',
  },
  mfaDescription: {
    fontSize: 12,
    color: '#8E8E93',
  },
  mfaProgress: {
    height: 8,
    backgroundColor: '#E5E5EA',
    borderRadius: 4,
    overflow: 'hidden',
  },
  mfaProgressBar: {
    height: '100%',
    backgroundColor: '#34C759',
    borderRadius: 4,
  },
  violationCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    marginBottom: 8,
    borderLeftWidth: 4,
    borderLeftColor: '#FF3B30',
  },
  violationSeverity: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 6,
    borderWidth: 1,
    marginRight: 12,
  },
  violationSeverityText: {
    fontSize: 10,
    fontWeight: '700',
    textTransform: 'uppercase',
  },
  violationContent: {
    flex: 1,
  },
  violationRule: {
    fontSize: 14,
    fontWeight: '600',
    color: '#000000',
  },
  violationAmount: {
    fontSize: 14,
    fontWeight: '500',
    color: '#FF3B30',
    marginTop: 4,
  },
  violationTime: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 2,
  },
  violationArrow: {
    marginLeft: 8,
  },
  violationArrowText: {
    fontSize: 20,
    color: '#8E8E93',
  },
  eventCard: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    marginBottom: 8,
  },
  eventHeader: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 8,
  },
  eventSeverity: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 6,
    borderWidth: 1,
    marginRight: 12,
  },
  eventSeverityText: {
    fontSize: 10,
    fontWeight: '700',
    textTransform: 'uppercase',
  },
  eventType: {
    fontSize: 14,
    fontWeight: '600',
    color: '#000000',
  },
  eventDetails: {
    fontSize: 14,
    color: '#8E8E93',
    marginBottom: 12,
  },
  eventFooter: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
  },
  eventTime: {
    fontSize: 12,
    color: '#8E8E93',
  },
  resolveButton: {
    backgroundColor: '#007AFF',
    paddingHorizontal: 16,
    paddingVertical: 8,
    borderRadius: 8,
  },
  resolveButtonText: {
    color: '#FFFFFF',
    fontSize: 14,
    fontWeight: '600',
  },
  emptyCard: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 32,
    alignItems: 'center',
  },
  emptyText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#000000',
  },
  emptySubtext: {
    fontSize: 14,
    color: '#8E8E93',
    marginTop: 4,
  },
  timeRangeSelector: {
    flexDirection: 'row',
    justifyContent: 'center',
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 4,
    marginBottom: 24,
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
  tipsCard: {
    backgroundColor: '#007AFF15',
    borderRadius: 12,
    padding: 16,
    borderWidth: 1,
    borderColor: '#007AFF30',
  },
  tipsTitle: {
    fontSize: 16,
    fontWeight: '600',
    color: '#007AFF',
    marginBottom: 12,
  },
  tipRow: {
    flexDirection: 'row',
    alignItems: 'center',
    marginBottom: 12,
  },
  tipIcon: {
    fontSize: 16,
    marginRight: 12,
  },
  tipText: {
    flex: 1,
    fontSize: 14,
    color: '#000000',
    lineHeight: 20,
  },
});