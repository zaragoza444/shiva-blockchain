import AsyncStorage from '@react-native-async-storage/async-storage';
import Constants from 'expo-constants';

const STORAGE_KEY = 'onex_wallet_url_override';

export const DEFAULT_WALLET_URL =
  process.env.EXPO_PUBLIC_WALLET_URL?.trim() || 'http://127.0.0.1:9338/wallet/';

export async function getWalletBaseUrl(): Promise<string> {
  let override = await AsyncStorage.getItem(STORAGE_KEY);
  if (!override) {
    override = await AsyncStorage.getItem('shiva_wallet_url_override');
    if (override) await AsyncStorage.setItem(STORAGE_KEY, override);
  }
  const base = (override?.trim() || DEFAULT_WALLET_URL).replace(/\/?$/, '/');
  return base;
}

export async function setWalletBaseUrl(url: string): Promise<void> {
  const trimmed = url.trim();
  if (!trimmed) {
    await AsyncStorage.removeItem(STORAGE_KEY);
    return;
  }
  await AsyncStorage.setItem(STORAGE_KEY, trimmed);
}

export function walletUrlWithHash(base: string, hash: string): string {
  const clean = base.replace(/#.*$/, '').replace(/\/?$/, '/');
  const h = hash.replace(/^#/, '');
  return h ? `${clean}#${h}` : clean;
}

export function resolveDeepLink(path: string | null): string | null {
  if (!path) return null;
  const p = path.replace(/^\/+/, '').toLowerCase();
  const map: Record<string, string> = {
    swap: 'swap',
    trade: 'swap',
    ai: 'ai',
    earn: 'earn',
    stake: 'earn',
    discover: 'discover',
    web3: 'web3',
    wallet: '',
    home: '',
  };
  if (p in map) return map[p];
  return null;
}

export function appVersion(): string {
  return Constants.expoConfig?.version ?? '1.0.0';
}
