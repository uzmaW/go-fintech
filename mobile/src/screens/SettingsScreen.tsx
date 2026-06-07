import React, { useState, useEffect } from 'react';
import {
  View,
  Text,
  TextInput,
  TouchableOpacity,
  StyleSheet,
  Alert,
  ScrollView,
} from 'react-native';
import AsyncStorage from '@react-native-async-storage/async-storage';
import { ENV_CONFIG, DEFAULT_CONFIG } from '../config/env';
import { getCurrentEnv } from '../config/env';

interface SettingsScreenProps {
  navigation: any;
}

export const SettingsScreen: React.FC<SettingsScreenProps> = ({ navigation }) => {
  const [apiUrl, setApiUrl] = useState('');
  const [wsUrl, setWsUrl] = useState('');
  const [env, setEnv] = useState('production');
  const [saved, setSaved] = useState(false);

  useEffect(() => {
    loadSavedUrls();
  }, []);

  const loadSavedUrls = async () => {
    try {
      const savedApiUrl = await AsyncStorage.getItem('api_url');
      const savedWsUrl = await AsyncStorage.getItem('ws_url');
      const currentEnv = await AsyncStorage.getItem('env') || getCurrentEnv();

      setApiUrl(savedApiUrl || DEFAULT_CONFIG.apiUrl);
      setWsUrl(savedWsUrl || DEFAULT_CONFIG.wsUrl);
      setEnv(currentEnv);
    } catch (error) {
      console.error('Failed to load saved URLs:', error);
    }
  };

  const handleSave = async () => {
    try {
      await AsyncStorage.setItem('api_url', apiUrl);
      await AsyncStorage.setItem('ws_url', wsUrl);
      await AsyncStorage.setItem('env', env);
      setSaved(true);
      Alert.alert('Success', 'API URLs have been updated. Restart the app to apply changes.');
      setTimeout(() => setSaved(false), 2000);
    } catch (error) {
      Alert.alert('Error', 'Failed to save settings');
    }
  };

  const handleReset = async () => {
    Alert.alert(
      'Reset Settings',
      'This will reset all API URLs to defaults. Continue?',
      [
        { text: 'Cancel', style: 'cancel' },
        {
          text: 'Reset',
          style: 'destructive',
          onPress: async () => {
            setApiUrl(DEFAULT_CONFIG.apiUrl);
            setWsUrl(DEFAULT_CONFIG.wsUrl);
            setEnv('production');
            await AsyncStorage.multiRemove(['api_url', 'ws_url', 'env']);
          },
        },
      ]
    );
  };

  const handlePresetSelect = (preset: string) => {
    const config = ENV_CONFIG[preset] || DEFAULT_CONFIG;
    setApiUrl(config.apiUrl);
    setWsUrl(config.wsUrl);
    setEnv(preset);
  };

  return (
    <ScrollView style={styles.container} contentContainerStyle={styles.contentContainer}>
      <View style={styles.section}>
        <Text style={styles.sectionTitle}>Environment Presets</Text>
        <View style={styles.presetsRow}>
          {Object.keys(ENV_CONFIG).map((preset) => (
            <TouchableOpacity
              key={preset}
              style={[
                styles.presetButton,
                env === preset && styles.presetButtonActive,
              ]}
              onPress={() => handlePresetSelect(preset)}
            >
              <Text
                style={[
                  styles.presetButtonText,
                  env === preset && styles.presetButtonTextActive,
                ]}
              >
                {preset.charAt(0).toUpperCase() + preset.slice(1)}
              </Text>
            </TouchableOpacity>
          ))}
        </View>
      </View>

      <View style={styles.section}>
        <Text style={styles.sectionTitle}>API Configuration</Text>
        <View style={styles.inputGroup}>
          <Text style={styles.inputLabel}>GraphQL API URL</Text>
          <TextInput
            style={styles.textInput}
            value={apiUrl}
            onChangeText={setApiUrl}
            placeholder="https://api.bank.com/graphql"
            placeholderTextColor="#8E8E93"
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="url"
          />
        </View>

        <View style={styles.inputGroup}>
          <Text style={styles.inputLabel}>WebSocket URL</Text>
          <TextInput
            style={styles.textInput}
            value={wsUrl}
            onChangeText={setWsUrl}
            placeholder="wss://api.bank.com/graphql"
            placeholderTextColor="#8E8E93"
            autoCapitalize="none"
            autoCorrect={false}
            keyboardType="url"
          />
        </View>
      </View>

      <View style={styles.section}>
        <TouchableOpacity
          style={[styles.saveButton, saved && styles.saveButtonSuccess]}
          onPress={handleSave}
          activeOpacity={0.8}
        >
          <Text style={styles.saveButtonText}>
            {saved ? 'Saved!' : 'Save Settings'}
          </Text>
        </TouchableOpacity>

        <TouchableOpacity
          style={styles.resetButton}
          onPress={handleReset}
          activeOpacity={0.8}
        >
          <Text style={styles.resetButtonText}>Reset to Defaults</Text>
        </TouchableOpacity>
      </View>

      <View style={styles.infoSection}>
        <Text style={styles.infoTitle}>Current Configuration</Text>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>Environment:</Text>
          <Text style={styles.infoValue}>{env}</Text>
        </View>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>API URL:</Text>
          <Text style={styles.infoValue} numberOfLines={1}>{apiUrl}</Text>
        </View>
        <View style={styles.infoRow}>
          <Text style={styles.infoLabel}>WebSocket:</Text>
          <Text style={styles.infoValue} numberOfLines={1}>{wsUrl}</Text>
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
  sectionTitle: {
    fontSize: 18,
    fontWeight: '600',
    color: '#000000',
    marginBottom: 12,
  },
  presetsRow: {
    flexDirection: 'row',
    gap: 8,
  },
  presetButton: {
    flex: 1,
    paddingVertical: 12,
    paddingHorizontal: 16,
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    borderWidth: 2,
    borderColor: '#E5E5EA',
    alignItems: 'center',
  },
  presetButtonActive: {
    borderColor: '#007AFF',
    backgroundColor: '#007AFF15',
  },
  presetButtonText: {
    fontSize: 14,
    fontWeight: '500',
    color: '#8E8E93',
  },
  presetButtonTextActive: {
    color: '#007AFF',
  },
  inputGroup: {
    marginBottom: 16,
  },
  inputLabel: {
    fontSize: 14,
    fontWeight: '500',
    color: '#000000',
    marginBottom: 8,
  },
  textInput: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    paddingHorizontal: 16,
    paddingVertical: 14,
    fontSize: 16,
    color: '#000000',
    borderWidth: 1,
    borderColor: '#E5E5EA',
  },
  saveButton: {
    backgroundColor: '#007AFF',
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    marginBottom: 12,
  },
  saveButtonSuccess: {
    backgroundColor: '#34C759',
  },
  saveButtonText: {
    color: '#FFFFFF',
    fontSize: 16,
    fontWeight: '600',
  },
  resetButton: {
    borderRadius: 12,
    paddingVertical: 16,
    alignItems: 'center',
    borderWidth: 1,
    borderColor: '#FF3B30',
  },
  resetButtonText: {
    color: '#FF3B30',
    fontSize: 16,
    fontWeight: '500',
  },
  infoSection: {
    backgroundColor: '#FFFFFF',
    borderRadius: 12,
    padding: 16,
  },
  infoTitle: {
    fontSize: 14,
    fontWeight: '600',
    color: '#8E8E93',
    marginBottom: 12,
  },
  infoRow: {
    flexDirection: 'row',
    justifyContent: 'space-between',
    paddingVertical: 8,
    borderBottomWidth: 1,
    borderBottomColor: '#E5E5EA',
  },
  infoLabel: {
    fontSize: 14,
    color: '#8E8E93',
  },
  infoValue: {
    fontSize: 14,
    fontWeight: '500',
    color: '#000000',
    maxWidth: '60%',
    textAlign: 'right',
  },
});