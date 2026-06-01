import * as Clipboard from 'expo-clipboard';
import * as Linking from 'expo-linking';
import * as Sharing from 'expo-sharing';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  View,
} from 'react-native';
import { useSafeAreaInsets } from 'react-native-safe-area-context';
import { WebView } from 'react-native-webview';
import type { WebViewNavigation } from 'react-native-webview/lib/WebViewTypes';
import { getWalletBaseUrl, walletUrlWithHash } from '../config';
import { INJECTED_BRIDGE_JS } from '../injectedBridge';

type Props = {
  deepLinkHash: string | null;
  onOpenSettings: () => void;
};

export function WebWalletScreen({ deepLinkHash, onOpenSettings }: Props) {
  const insets = useSafeAreaInsets();
  const webRef = useRef<WebView>(null);
  const [url, setUrl] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadUrl = useCallback(async (hash?: string | null) => {
    try {
      setError(null);
      const base = await getWalletBaseUrl();
      const target = walletUrlWithHash(base, hash ?? deepLinkHash ?? '');
      setUrl(target);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to load wallet URL');
    }
  }, [deepLinkHash]);

  useEffect(() => {
    loadUrl();
  }, [loadUrl]);

  useEffect(() => {
    if (deepLinkHash && url) {
      const next = walletUrlWithHash(url, deepLinkHash);
      webRef.current?.injectJavaScript(`window.location.href=${JSON.stringify(next)}; true;`);
    }
  }, [deepLinkHash, url]);

  const shouldOpenExternally = (requestUrl: string, base: string) => {
    try {
      const req = new URL(requestUrl);
      const baseUrl = new URL(base);
      if (req.origin !== baseUrl.origin) return true;
      if (!req.pathname.startsWith('/wallet') && !req.pathname.startsWith('/bridge')) {
        return true;
      }
    } catch {
      return true;
    }
    return false;
  };

  const onShouldStartLoadWithRequest = (event: { url: string }) => {
    if (!url) return true;
    if (shouldOpenExternally(event.url, url)) {
      Linking.openURL(event.url);
      return false;
    }
    return true;
  };

  const onNavigationStateChange = (nav: WebViewNavigation) => {
    if (nav.loading) setLoading(true);
    else setLoading(false);
  };

  const onMessage = async (event: { nativeEvent: { data: string } }) => {
    try {
      const msg = JSON.parse(event.nativeEvent.data) as { type?: string; text?: string };
      if (msg.type === 'copy' && msg.text) {
        await Clipboard.setStringAsync(msg.text);
      }
      if (msg.type === 'share' && msg.text) {
        if (await Sharing.isAvailableAsync()) {
          await Sharing.shareAsync(msg.text, { dialogTitle: 'OneX Wallet' });
        } else {
          await Clipboard.setStringAsync(msg.text);
        }
      }
    } catch {
      /* ignore malformed messages */
    }
  };

  if (error) {
    return (
      <View style={[styles.center, { paddingTop: insets.top }]}>
        <Text style={styles.errTitle}>Cannot open wallet</Text>
        <Text style={styles.errBody}>{error}</Text>
        <Pressable style={styles.btn} onPress={() => loadUrl()}>
          <Text style={styles.btnText}>Retry</Text>
        </Pressable>
        <Pressable style={styles.btnSecondary} onPress={onOpenSettings}>
          <Text style={styles.btnText}>Settings</Text>
        </Pressable>
      </View>
    );
  }

  if (!url) {
    return (
      <View style={[styles.center, { paddingTop: insets.top }]}>
        <ActivityIndicator size="large" color="#00c853" />
      </View>
    );
  }

  return (
    <View style={[styles.root, { paddingTop: insets.top }]}>
      {loading && (
        <View style={styles.loadingBar}>
          <ActivityIndicator color="#00c853" />
        </View>
      )}
      <WebView
        ref={webRef}
        source={{ uri: url }}
        style={styles.web}
        onLoadEnd={() => setLoading(false)}
        onError={() => {
          setLoading(false);
          setError('WebView failed to load. Check wallet URL in Settings.');
        }}
        onNavigationStateChange={onNavigationStateChange}
        onShouldStartLoadWithRequest={onShouldStartLoadWithRequest}
        onMessage={onMessage}
        injectedJavaScriptBeforeContentLoaded={INJECTED_BRIDGE_JS}
        javaScriptEnabled
        domStorageEnabled
        allowsBackForwardNavigationGestures
        setSupportMultipleWindows={false}
        originWhitelist={['*']}
        pullToRefreshEnabled
      />
      <Pressable style={[styles.gear, { top: insets.top + 8 }]} onPress={onOpenSettings}>
        <Text style={styles.gearText}>⚙</Text>
      </Pressable>
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#000' },
  flex: { flex: 1 },
  web: { flex: 1, backgroundColor: '#000' },
  center: { flex: 1, backgroundColor: '#000', justifyContent: 'center', alignItems: 'center', padding: 24 },
  loadingBar: { position: 'absolute', top: 0, left: 0, right: 0, zIndex: 2, padding: 8, alignItems: 'center' },
  errTitle: { color: '#fff', fontSize: 18, fontWeight: '600', marginBottom: 8 },
  errBody: { color: '#909090', textAlign: 'center', marginBottom: 20 },
  btn: { backgroundColor: '#fff', paddingHorizontal: 24, paddingVertical: 12, borderRadius: 12, marginBottom: 10 },
  btnSecondary: { borderWidth: 1, borderColor: '#444', paddingHorizontal: 24, paddingVertical: 12, borderRadius: 12 },
  btnText: { color: '#000', fontWeight: '600' },
  gear: { position: 'absolute', right: 16, zIndex: 3, width: 36, height: 36, borderRadius: 18, backgroundColor: '#1a1a1a', alignItems: 'center', justifyContent: 'center' },
  gearText: { color: '#fff', fontSize: 18 },
});
