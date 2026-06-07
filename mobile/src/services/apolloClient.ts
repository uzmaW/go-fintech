import { ApolloClient, InMemoryCache, HttpLink, split } from '@apollo/client';
import { GraphQLWsLink } from '@apollo/client/link/subscriptions';
import { getMainDefinition } from '@apollo/client/utilities';
import { createClient } from 'graphql-ws';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { setContext } from '@apollo/client/link/context';

// API Configuration - can be overridden via environment or AsyncStorage
const getApiUrl = (): string => {
  // Check AsyncStorage first for runtime override
  const envUrl = process.env.EXPO_PUBLIC_API_URL || process.env.REACT_NATIVE_API_URL;
  return envUrl || 'https://api.bank.com/graphql';
};

const getWsUrl = (): string => {
  const envUrl = process.env.EXPO_PUBLIC_WS_URL || process.env.REACT_NATIVE_WS_URL;
  return envUrl || 'wss://api.bank.com/graphql';
};

let cachedApiUrl: string | null = null;
let cachedWsUrl: string | null = null;

export const getApiConfig = async () => {
  if (!cachedApiUrl) {
    cachedApiUrl = await AsyncStorage.getItem('api_url') || getApiUrl();
  }
  if (!cachedWsUrl) {
    cachedWsUrl = await AsyncStorage.getItem('ws_url') || getWsUrl();
  }
  return { apiUrl: cachedApiUrl, wsUrl: cachedWsUrl };
};

export const setApiUrl = async (url: string): Promise<void> => {
  cachedApiUrl = url;
  await AsyncStorage.setItem('api_url', url);
};

export const setWsUrl = async (url: string): Promise<void> => {
  cachedWsUrl = url;
  await AsyncStorage.setItem('ws_url', url);
};

const createHttpLink = (uri: string) => new HttpLink({ uri });

const createWsLink = (url: string) => new GraphQLWsLink(
  createClient({
    url,
    connectionParams: async () => {
      const token = await AsyncStorage.getItem('authToken');
      return {
        authorization: token ? `Bearer ${token}` : '',
      };
    },
  })
);

const authLink = setContext(async (_, { headers }) => {
  const token = await AsyncStorage.getItem('authToken');
  return {
    headers: {
      ...headers,
      authorization: token ? `Bearer ${token}` : '',
    },
  };
});

export const createApolloClient = async () => {
  const { apiUrl, wsUrl } = await getApiConfig();

  const httpLink = createHttpLink(apiUrl);
  const wsLink = createWsLink(wsUrl);

  const splitLink = split(
    ({ query }) => {
      const definition = getMainDefinition(query);
      return (
        definition.kind === 'OperationDefinition' &&
        definition.operation === 'subscription'
      );
    },
    wsLink,
    authLink.concat(httpLink)
  );

  return new ApolloClient({
    link: splitLink,
    cache: new InMemoryCache({
      typePolicies: {
        Account: {
          fields: {
            transactions: {
              keyArgs: ['filter'],
              merge(existing = { edges: [] }, incoming) {
                return {
                  ...incoming,
                  edges: [...existing.edges, ...incoming.edges],
                };
              },
            },
          },
        },
      },
    }),
    defaultOptions: {
      watchQuery: {
        fetchPolicy: 'cache-and-network',
      },
    },
  });
};

// Default client instance with fallback URLs
const defaultHttpLink = new HttpLink({
  uri: getApiUrl(),
});

const defaultWsLink = new GraphQLWsLink(
  createClient({
    url: getWsUrl(),
    connectionParams: async () => {
      const token = await AsyncStorage.getItem('authToken');
      return {
        authorization: token ? `Bearer ${token}` : '',
      };
    },
  })
);

const defaultSplitLink = split(
  ({ query }) => {
    const definition = getMainDefinition(query);
    return (
      definition.kind === 'OperationDefinition' &&
      definition.operation === 'subscription'
    );
  },
  defaultWsLink,
  authLink.concat(defaultHttpLink)
);

export const apolloClient = new ApolloClient({
  link: defaultSplitLink,
  cache: new InMemoryCache({
    typePolicies: {
      Account: {
        fields: {
          transactions: {
            keyArgs: ['filter'],
            merge(existing = { edges: [] }, incoming) {
              return {
                ...incoming,
                edges: [...existing.edges, ...incoming.edges],
              };
            },
          },
        },
      },
    },
  }),
  defaultOptions: {
    watchQuery: {
      fetchPolicy: 'cache-and-network',
    },
  },
});