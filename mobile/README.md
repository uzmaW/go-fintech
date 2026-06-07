# Go Fintech Mobile App

A React Native mobile application for the Go Fintech transaction processing system.

## Features

- **Transactions Tab**: View and manage transactions with real-time updates via GraphQL subscriptions
- **Dashboard Tab**: Overview of account balance, metrics, and recent activity
- **Security Tab**: Security monitoring, fraud alerts, and MFA settings

## Architecture

- **React Native** with Expo for cross-platform support
- **Apollo Client** for GraphQL with WebSocket subscriptions for real-time updates
- **React Navigation** for stack and tab navigation
- **AsyncStorage** for secure token storage

## Project Structure

```
mobile/
├── src/
│   ├── components/       # Reusable UI components
│   ├── navigation/        # Navigation configuration
│   ├── screens/          # Screen components
│   ├── services/          # Apollo client and API services
│   ├── types/            # TypeScript type definitions
│   └── App.tsx           # App entry point
├── package.json
└── tsconfig.json
```

## Getting Started

### Prerequisites

- Node.js 18+
- Expo CLI
- npm or yarn

### Installation

```bash
cd mobile
npm install
```

### Running the App

```bash
# Start Metro bundler
npm start

# Run on iOS
npm run ios

# Run on Android
npm run android
```

## GraphQL API

The app connects to `https://api.bank.com/graphql` for GraphQL operations and `wss://api.bank.com/graphql` for subscriptions.

## Screens

1. **TransactionsScreen**: FlatList with pull-to-refresh, pagination, and real-time subscription updates
2. **DashboardScreen**: Balance card, metrics grid, fraud alerts, and recent transactions
3. **SecurityScreen**: Security settings, metrics overview, fraud violations, and active security events
4. **TransactionDetailScreen**: Full transaction details with status timeline

## Dependencies

- `@apollo/client`: GraphQL client
- `@react-navigation/native`: Navigation framework
- `@react-navigation/bottom-tabs`: Bottom tab navigator
- `@react-navigation/stack`: Stack navigator
- `graphql-ws`: WebSocket link for subscriptions
- `@react-native-async-storage/async-storage`: Secure token storage