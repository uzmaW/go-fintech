import React from 'react';
import {
  View,
  Text,
  ScrollView,
  TouchableOpacity,
  StyleSheet,
  Alert,
} from 'react-native';
import { useQuery, gql } from '@apollo/client';
import { LoadingSpinner } from '../components/LoadingSpinner';

const GET_TRANSACTION_DETAIL = gql`
  query GetTransactionDetail($id: ID!) {
    transaction(id: $id) {
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
        category
      }
      createdAt
      updatedAt
      status
      initiatedBy
      authorizationCode
      metadata
    }
  }
`;

interface TransactionDetailScreenProps {
  route: {
    params: {
      id: string;
    };
  };
  navigation: any;
}

export const TransactionDetailScreen: React.FC<TransactionDetailScreenProps> = ({
  route,
  navigation,
}) => {
  const { id } = route.params;

  const { data, loading, error } = useQuery(GET_TRANSACTION_DETAIL, {
    variables: { id },
  });

  const handleReportIssue = () => {
    Alert.alert(
      'Report Issue',
      'What type of issue would you like to report?',
      [
        { text: 'Cancel', style: 'cancel' },
        { text: 'Unauthorized Transaction', onPress: () => console.log('Unauthorized') },
        { text: 'Incorrect Amount', onPress: () => console.log('Incorrect Amount') },
        { text: 'Duplicate Transaction', onPress: () => console.log('Duplicate') },
      ]
    );
  };

  if (loading) {
    return <LoadingSpinner message="Loading transaction details..." />;
  }

  if (error) {
    return (
      <View style={styles.errorContainer}>
        <Text style={styles.errorText}>Failed to load transaction</Text>
        <TouchableOpacity style={styles.retryButton} onPress={() => navigation.goBack()}>
          <Text style={styles.retryButtonText}>Go Back</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const tx = data?.transaction;

  if (!tx) {
    return (
      <View style={styles.errorContainer}>
        <Text style={styles.errorText}>Transaction not found</Text>
        <TouchableOpacity style={styles.retryButton} onPress={() => navigation.goBack()}>
          <Text style={styles.retryButtonText}>Go Back</Text>
        </TouchableOpacity>
      </View>
    );
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'COMPLETED':
        return '#34C759';
      case 'PENDING':
        return '#FF9500';
      case 'FAILED':
        return '#FF3B30';
      default:
        return '#8E8E93';
    }
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.contentContainer}>
      {/* Amount Header */}
      <View style={styles.header}>
        <View
          style={[
            styles.statusBadge,
            { backgroundColor: getStatusColor(tx.status) + '20', borderColor: getStatusColor(tx.status) },
          ]}
        >
          <Text style={[styles.statusText, { color: getStatusColor(tx.status) }]}>
            {tx.status}
          </Text>
        </View>
        <Text
          style={[
            styles.amount,
            tx.type === 'DEBIT' ? styles.debitAmount : styles.creditAmount,
          ]}
        >
          {tx.type === 'DEBIT' ? '-' : '+'}{tx.amount.formatted}
        </Text>
        <Text style={styles.description}>{tx.description}</Text>
      </View>

      {/* Transaction Details */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Transaction Details</Text>
        <View style={styles.card}>
          <DetailRow label="Transaction ID" value={tx.id} />
          <DetailRow label="Date & Time" value={new Date(tx.createdAt).toLocaleString()} />
          {tx.merchant && (
            <DetailRow label="Merchant" value={tx.merchant.name} />
          )}
          {tx.merchant?.category && (
            <DetailRow label="Category" value={tx.merchant.category} />
          )}
          <DetailRow label="Type" value={tx.type} />
          <DetailRow label="Initiated By" value={tx.initiatedBy || 'User'} />
          {tx.authorizationCode && (
            <DetailRow label="Authorization Code" value={tx.authorizationCode} />
          )}
        </View>
      </View>

      {/* Status Timeline */}
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Status History</Text>
        <View style={styles.card}>
          <TimelineItem
            title="Transaction Initiated"
            time={tx.createdAt}
            completed
          />
          {tx.status === 'COMPLETED' && (
            <TimelineItem title="Completed" time={tx.updatedAt} completed />
          )}
          {tx.status === 'PENDING' && (
            <TimelineItem title="Pending" time={tx.updatedAt} active />
          )}
          {tx.status === 'FAILED' && (
            <TimelineItem title="Failed" time={tx.updatedAt} failed />
          )}
        </View>
      </View>

      {/* Actions */}
      <View style={styles.section}>
        <TouchableOpacity
          style={styles.reportButton}
          onPress={handleReportIssue}
          activeOpacity={0.7}
        >
          <Text style={styles.reportButtonText}>Report an Issue</Text>
        </TouchableOpacity>
      </View>
    </ScrollView>
  );
};

interface DetailRowProps {
  label: string;
  value: string;
}

const DetailRow: React.FC<DetailRowProps> = ({ label, value }) => (
  <View style={styles.detailRow}>
    <Text style={styles.detailLabel}>{label}</Text>
    <Text style={styles.detailValue}>{value}</Text>
  </View>
);

interface TimelineItemProps {
  title: string;
  time: string;
  completed?: boolean;
  active?: boolean;
  failed?: boolean;
}

const TimelineItem: React.FC<TimelineItemProps> = ({ title, time, completed, active, failed }) => {
  const getDotColor = () => {
    if (failed) return '#FF3B30';
    if (active) return '#FF9500';
    if (completed) return '#34C759';
    return '#8E8E93';
  };

  return (
    <View style={styles.timelineItem}>
      <View style={[styles.timelineDot, { backgroundColor: getDotColor() }]} />
      <View style={styles.timelineContent}>
        <Text style={styles.timelineTitle}>{title}</Text>
        <Text style={styles.timelineTime}>{new Date(time).toLocaleString()}</Text>
      </View>
    </View>
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
  header: {
    backgroundColor: '#FFFFFF',
    borderRadius: 16,
    padding: 24,
    alignItems: 'center',
    marginBottom: 24,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 2 },
    shadowOpacity: 0.1,
    shadowRadius: 4,
    elevation: 4,
  },
  statusBadge: {
    paddingHorizontal: 12,
    paddingVertical: 6,
    borderRadius: 8,
    borderWidth: 1,
    marginBottom: 16,
  },
  statusText: {
    fontSize: 12,
    fontWeight: '600',
    textTransform: 'uppercase',
  },
  amount: {
    fontSize: 40,
    fontWeight: '700',
  },
  debitAmount: {
    color: '#FF3B30',
  },
  creditAmount: {
    color: '#34C759',
  },
  description: {
    fontSize: 16,
    color: '#8E8E93',
    marginTop: 8,
  },
  section: {
    marginBottom: 24,
  },
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#000000',
    marginBottom: 12,
  },
  card: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 2,
  },
  detailRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 12,
    borderBottomWidth: 1,
    borderBottomColor: '#E5E5EA',
  },
  detailLabel: {
    fontSize: 14,
    color: '#8E8E93',
  },
  detailValue: {
    fontSize: 14,
    fontWeight: '500',
    color: '#000000',
    textAlign: 'right',
    flex: 1,
    marginLeft: 16,
  },
  timelineItem: {
    flexDirection: 'row',
    alignItems: 'flex-start',
    paddingVertical: 12,
  },
  timelineDot: {
    width: 12,
    height: 12,
    borderRadius: 6,
    marginRight: 12,
    marginTop: 4,
  },
  timelineContent: {
    flex: 1,
  },
  timelineTitle: {
    fontSize: 14,
    fontWeight: '500',
    color: '#000000',
  },
  timelineTime: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 2,
  },
  reportButton: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#FF3B30',
  },
  reportButtonText: {
    fontSize: 16,
    fontWeight: '600',
    color: '#FF3B30',
  },
  errorContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    padding: 32,
    backgroundColor: '#F2F2F7',
  },
  errorText: {
    fontSize: 16,
    color: '#8E8E93',
    marginBottom: 16,
  },
  retryButton: {
    backgroundColor: '#007AFF',
    paddingHorizontal: 24,
    paddingVertical: 12,
    borderRadius: 8,
  },
  retryButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
});