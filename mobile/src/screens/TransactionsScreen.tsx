import React, { useState } from 'react';
import {
  View,
  Text,
  FlatList,
  TouchableOpacity,
  StyleSheet,
  RefreshControl,
  ActivityIndicator,
} from 'react-native';
import { useQuery, useSubscription, gql } from '@apollo/client';
import { Transaction, TransactionEdge } from '../types';

const GET_TRANSACTIONS = gql`
  query GetTransactions($accountId: ID!, $pagination: PaginationInput!) {
    transactions(accountId: $accountId, pagination: $pagination) {
      edges {
        node {
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
      }
      pageInfo {
        hasNextPage
        endCursor
      }
      totalCount
    }
  }
`;

const TRANSACTION_CREATED_SUBSCRIPTION = gql`
  subscription OnTransactionCreated($accountId: ID!) {
    transactionCreated(accountId: $accountId) {
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
  }
`;

interface TransactionsScreenProps {
  navigation: any;
}

export const TransactionsScreen: React.FC<TransactionsScreenProps> = ({ navigation }) => {
  const [refreshing, setRefreshing] = useState(false);
  const [loadingMore, setLoadingMore] = useState(false);

  const { data, loading, fetchMore, refetch } = useQuery(GET_TRANSACTIONS, {
    variables: {
      accountId: 'default-account',
      pagination: { limit: 20, offset: 0 },
    },
    notifyOnNetworkStatusChange: true,
  });

  useSubscription(TRANSACTION_CREATED_SUBSCRIPTION, {
    variables: { accountId: 'default-account' },
    onSubscriptionData: () => {
      refetch();
    },
  });

  const handleRefresh = async () => {
    setRefreshing(true);
    await refetch();
    setRefreshing(false);
  };

  const handleLoadMore = async () => {
    if (loadingMore || !data?.transactions?.pageInfo?.hasNextPage) return;

    setLoadingMore(true);
    try {
      await fetchMore({
        variables: {
          pagination: {
            limit: 20,
            offset: data.transactions.edges.length,
          },
        },
      });
    } finally {
      setLoadingMore(false);
    }
  };

  const renderTransaction = ({ item }: { item: TransactionEdge }) => (
    <TouchableOpacity
      style={styles.transactionCard}
      onPress={() => navigation.navigate('TransactionDetail', { id: item.node.id })}
      activeOpacity={0.7}
    >
      <View style={styles.merchantLogo}>
        <Text style={styles.merchantInitial}>
          {item.node.merchant?.name?.charAt(0) || 'T'}
        </Text>
      </View>
      <View style={styles.transactionDetails}>
        <Text style={styles.merchantName}>
          {item.node.merchant?.name || 'Transfer'}
        </Text>
        <Text style={styles.description} numberOfLines={1}>
          {item.node.description}
        </Text>
        <Text style={styles.date}>
          {new Date(item.node.createdAt).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            hour: '2-digit',
            minute: '2-digit',
          })}
        </Text>
      </View>
      <View style={styles.amountContainer}>
        <Text
          style={[
            styles.amount,
            item.node.type === 'DEBIT' ? styles.debitAmount : styles.creditAmount,
          ]}
        >
          {item.node.type === 'DEBIT' ? '-' : '+'}{item.node.amount.formatted}
        </Text>
        <View style={[styles.statusBadge, getStatusStyle(item.node.status)]}>
          <Text style={styles.statusText}>{item.node.status}</Text>
        </View>
      </View>
    </TouchableOpacity>
  );

  const renderFooter = () => {
    if (!loadingMore) return null;
    return (
      <View style={styles.loadingFooter}>
        <ActivityIndicator size="small" color="#007AFF" />
      </View>
    );
  };

  const renderEmpty = () => (
    <View style={styles.emptyContainer}>
      <Text style={styles.emptyTitle}>No Transactions</Text>
      <Text style={styles.emptySubtitle}>
        Your transactions will appear here once you start using your account.
      </Text>
    </View>
  );

  if (loading && !data) {
    return (
      <View style={styles.loadingContainer}>
        <ActivityIndicator size="large" color="#007AFF" />
        <Text style={styles.loadingText}>Loading transactions...</Text>
      </View>
    );
  }

  return (
    <View style={styles.container}>
      <FlatList
        data={data?.transactions?.edges || []}
        keyExtractor={(item) => item.node.id}
        renderItem={renderTransaction}
        contentContainerStyle={styles.listContent}
        refreshControl={
          <RefreshControl refreshing={refreshing} onRefresh={handleRefresh} tintColor="#007AFF" />
        }
        onEndReached={handleLoadMore}
        onEndReachedThreshold={0.5}
        ListFooterComponent={renderFooter}
        ListEmptyComponent={renderEmpty}
        ItemSeparatorComponent={() => <View style={styles.separator} />}
      />
    </View>
  );
};

const getStatusStyle = (status: string) => {
  switch (status) {
    case 'COMPLETED':
      return { backgroundColor: '#34C75920', borderColor: '#34C759' };
    case 'PENDING':
      return { backgroundColor: '#FF950020', borderColor: '#FF9500' };
    case 'FAILED':
      return { backgroundColor: '#FF3B3020', borderColor: '#FF3B30' };
    default:
      return { backgroundColor: '#8E8E9320', borderColor: '#8E8E93' };
  }
};

const styles = StyleSheet.create({
  container: {
    flex: 1,
    backgroundColor: '#F2F2F7',
  },
  listContent: {
    padding: 16,
  },
  loadingContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    backgroundColor: '#F2F2F7',
  },
  loadingText: {
    marginTop: 12,
    fontSize: 16,
    color: '#8E8E93',
  },
  transactionCard: {
    flexDirection: 'row',
    alignItems: 'center',
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
    shadowColor: '#000000',
    shadowOffset: { width: 0, height: 1 },
    shadowOpacity: 0.05,
    shadowRadius: 2,
    elevation: 2,
  },
  merchantLogo: {
    width: 48,
    height: 48,
    borderRadius: 24,
    backgroundColor: '#007AFF',
    justifyContent: 'center',
    alignItems: 'center',
  },
  merchantInitial: {
    color: '#FFFFFF',
    fontSize: 20,
    fontWeight: '600',
  },
  transactionDetails: {
    flex: 1,
    marginLeft: 12,
  },
  merchantName: {
    fontSize: 16,
    fontWeight: '600',
    color: '#000000',
  },
  description: {
    fontSize: 14,
    color: '#8E8E93',
    marginTop: 2,
  },
  date: {
    fontSize: 12,
    color: '#8E8E93',
    marginTop: 4,
  },
  amountContainer: {
    alignItems: 'flex-end',
  },
  amount: {
    fontSize: 16,
    fontWeight: '600',
  },
  debitAmount: {
    color: '#FF3B30',
  },
  creditAmount: {
    color: '#34C759',
  },
  statusBadge: {
    paddingHorizontal: 8,
    paddingVertical: 4,
    borderRadius: 6,
    borderWidth: 1,
    marginTop: 6,
  },
  statusText: {
    fontSize: 10,
    fontWeight: '600',
    textTransform: 'uppercase',
  },
  separator: {
    height: 12,
  },
  loadingFooter: {
    paddingVertical: 16,
    alignItems: 'center',
  },
  emptyContainer: {
    flex: 1,
    justifyContent: 'center',
    alignItems: 'center',
    paddingTop: 100,
    paddingHorizontal: 32,
  },
  emptyTitle: {
    fontSize: 20,
    fontWeight: '600',
    color: '#000000',
    marginBottom: 8,
  },
  emptySubtitle: {
    fontSize: 16,
    color: '#8E8E93',
    textAlign: 'center',
    lineHeight: 22,
  },
});