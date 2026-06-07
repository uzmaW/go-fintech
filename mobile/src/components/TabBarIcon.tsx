import React from 'react';
import { View, Text, StyleSheet } from 'react-native';

interface TabBarIconProps {
  name: 'transactions' | 'dashboard' | 'security';
  color: string;
  focused: boolean;
}

export const TabBarIcon: React.FC<TabBarIconProps> = ({ name, color, focused }) => {
  const getIcon = () => {
    switch (name) {
      case 'transactions':
        return (
          <View style={styles.iconContainer}>
            <View style={[styles.icon, { borderColor: color }]}>
              <Text style={[styles.iconText, { color }]}>₿</Text>
            </View>
          </View>
        );
      case 'dashboard':
        return (
          <View style={styles.iconContainer}>
            <View style={[styles.icon, { borderColor: color }]}>
              <Text style={[styles.iconText, { color }]}>◉</Text>
            </View>
          </View>
        );
      case 'security':
        return (
          <View style={styles.iconContainer}>
            <View style={[styles.icon, { borderColor: color }]}>
              <Text style={[styles.iconText, { color }]}>⚙</Text>
            </View>
          </View>
        );
      default:
        return null;
    }
  };

  return (
    <View style={styles.container}>
      {getIcon()}
    </View>
  );
};

const styles = StyleSheet.create({
  container: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  iconContainer: {
    alignItems: 'center',
    justifyContent: 'center',
  },
  icon: {
    width: 28,
    height: 28,
    borderRadius: 14,
    borderWidth: 2,
    alignItems: 'center',
    justifyContent: 'center',
  },
  iconText: {
    fontSize: 14,
    fontWeight: '600',
  },
});