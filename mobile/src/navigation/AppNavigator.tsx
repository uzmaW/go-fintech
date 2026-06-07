import React from 'react';
import { NavigationContainer } from '@react-navigation/native';
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs';
import { createStackNavigator } from '@react-navigation/stack';
import { ApolloProvider } from '@apollo/client';
import { apolloClient } from '../services/apolloClient';
import { TransactionsScreen } from '../screens/TransactionsScreen';
import { DashboardScreen } from '../screens/DashboardScreen';
import { SecurityScreen } from '../screens/SecurityScreen';
import { SettingsScreen } from '../screens/SettingsScreen';
import { TransactionDetailScreen } from '../screens/TransactionDetailScreen';
import { TabBarIcon } from '../components/TabBarIcon';

const Tab = createBottomTabNavigator();
const Stack = createStackNavigator();

const TransactionsStack = () => (
  <Stack.Navigator
    screenOptions={{
      headerStyle: {
        backgroundColor: '#FFFFFF',
        shadowColor: '#000000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.1,
        shadowRadius: 3,
        elevation: 4,
      },
      headerTitleStyle: {
        fontWeight: '600',
        fontSize: 17,
      },
      headerTintColor: '#007AFF',
      headerBackTitleVisible: false,
    }}
  >
    <Stack.Screen
      name="TransactionsList"
      component={TransactionsScreen}
      options={{ title: 'Transactions' }}
    />
    <Stack.Screen
      name="TransactionDetail"
      component={TransactionDetailScreen}
      options={{ title: 'Transaction Details' }}
    />
  </Stack.Navigator>
);

const DashboardStack = () => (
  <Stack.Navigator
    screenOptions={{
      headerStyle: {
        backgroundColor: '#FFFFFF',
        shadowColor: '#000000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.1,
        shadowRadius: 3,
        elevation: 4,
      },
      headerTitleStyle: {
        fontWeight: '600',
        fontSize: 17,
      },
      headerTintColor: '#007AFF',
      headerBackTitleVisible: false,
    }}
  >
    <Stack.Screen
      name="DashboardMain"
      component={DashboardScreen}
      options={{ title: 'Dashboard' }}
    />
    <Stack.Screen
      name="TransactionDetail"
      component={TransactionDetailScreen}
      options={{ title: 'Transaction Details' }}
    />
    <Stack.Screen
      name="SendMoney"
      component={PlaceholderScreen}
      options={{ title: 'Send Money' }}
    />
    <Stack.Screen
      name="AddMoney"
      component={PlaceholderScreen}
      options={{ title: 'Add Money' }}
    />
    <Stack.Screen
      name="PayBills"
      component={PlaceholderScreen}
      options={{ title: 'Pay Bills' }}
    />
    <Stack.Screen
      name="Security"
      component={SecurityScreen}
      options={{ title: 'Security' }}
    />
    <Stack.Screen
      name="FraudDashboard"
      component={PlaceholderScreen}
      options={{ title: 'Fraud Dashboard' }}
    />
    <Stack.Screen
      name="AllSecurityEvents"
      component={PlaceholderScreen}
      options={{ title: 'Security Events' }}
    />
  </Stack.Navigator>
);

const SecurityStack = () => (
  <Stack.Navigator
    screenOptions={{
      headerStyle: {
        backgroundColor: '#FFFFFF',
        shadowColor: '#000000',
        shadowOffset: { width: 0, height: 1 },
        shadowOpacity: 0.1,
        shadowRadius: 3,
        elevation: 4,
      },
      headerTitleStyle: {
        fontWeight: '600',
        fontSize: 17,
      },
      headerTintColor: '#007AFF',
      headerBackTitleVisible: false,
    }}
  >
    <Stack.Screen
      name="SecurityMain"
      component={SecurityScreen}
      options={{ title: 'Security' }}
    />
    <Stack.Screen
      name="Settings"
      component={SettingsScreen}
      options={{ title: 'Settings' }}
    />
    <Stack.Screen
      name="TransactionDetail"
      component={TransactionDetailScreen}
      options={{ title: 'Transaction Details' }}
    />
  </Stack.Navigator>
);

const PlaceholderScreen: React.FC<{ navigation: any }> = ({ navigation }) => (
  <React.Fragment>
    {/* Placeholder for additional screens */}
  </React.Fragment>
);

export const AppNavigator: React.FC = () => {
  return (
    <ApolloProvider client={apolloClient}>
      <NavigationContainer>
        <Tab.Navigator
          screenOptions={{
            tabBarActiveTintColor: '#007AFF',
            tabBarInactiveTintColor: '#8E8E93',
            tabBarStyle: {
              backgroundColor: '#FFFFFF',
              borderTopColor: '#E5E5EA',
              paddingTop: 8,
              height: 88,
            },
            tabBarLabelStyle: {
              fontSize: 12,
              fontWeight: '500',
            },
            headerShown: false,
          }}
        >
          <Tab.Screen
            name="Transactions"
            component={TransactionsStack}
            options={{
              tabBarIcon: ({ color, focused }) => (
                <TabBarIcon name="transactions" color={color} focused={focused} />
              ),
            }}
          />
          <Tab.Screen
            name="Dashboard"
            component={DashboardStack}
            options={{
              tabBarIcon: ({ color, focused }) => (
                <TabBarIcon name="dashboard" color={color} focused={focused} />
              ),
            }}
          />
          <Tab.Screen
            name="Security"
            component={SecurityStack}
            options={{
              tabBarIcon: ({ color, focused }) => (
                <TabBarIcon name="security" color={color} focused={focused} />
              ),
            }}
          />
        </Tab.Navigator>
      </NavigationContainer>
    </ApolloProvider>
  );
};