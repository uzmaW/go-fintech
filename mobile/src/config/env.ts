// Environment configuration for mobile app

export interface ApiConfig {
  apiUrl: string;
  wsUrl: string;
}

// Default development URLs
export const DEFAULT_CONFIG: ApiConfig = {
  apiUrl: 'https://api.bank.com/graphql',
  wsUrl: 'wss://api.bank.com/graphql',
};

// Environment-specific overrides
export const ENV_CONFIG: Record<string, ApiConfig> = {
  development: {
    apiUrl: 'http://localhost:8080/graphql',
    wsUrl: 'ws://localhost:8080/graphql',
  },
  staging: {
    apiUrl: 'https://api-staging.bank.com/graphql',
    wsUrl: 'wss://api-staging.bank.com/graphql',
  },
  production: {
    apiUrl: 'https://api.bank.com/graphql',
    wsUrl: 'wss://api.bank.com/graphql',
  },
};

// Helper to get current environment
export const getCurrentEnv = (): string => {
  if (__DEV__) return 'development';
  // Can be extended to check AsyncStorage or process.env
  return 'production';
};

export const getConfig = (): ApiConfig => {
  const env = getCurrentEnv();
  return ENV_CONFIG[env] || DEFAULT_CONFIG;
};