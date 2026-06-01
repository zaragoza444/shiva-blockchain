// Injected into the WebView so the wallet can call native share/clipboard.
export const INJECTED_BRIDGE_JS = `
(function() {
  if (window.OneXMobile) return;
  window.OneXMobile = {
    isApp: true,
    post: function(msg) {
      if (window.ReactNativeWebView) {
        window.ReactNativeWebView.postMessage(JSON.stringify(msg));
      }
    },
    copy: function(text) {
      this.post({ type: 'copy', text: String(text || '') });
    },
    share: function(text) {
      this.post({ type: 'share', text: String(text || '') });
    }
  };
  document.documentElement.classList.add('onex-mobile');
})();
true;
`;
