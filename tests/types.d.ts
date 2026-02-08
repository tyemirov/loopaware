export {};

declare global {
  interface Window {
    apiFetch?: (input: RequestInfo | URL, init?: RequestInit) => Promise<Response>;
    __loopawareFetchIntercept?: {
      requests: Array<{ url: string; method: string; body: string; status: number }>;
      storageKey?: string;
      originalFetch?: typeof fetch;
      originalApiFetch?: typeof fetch;
    };
    __loopawareDashboardSettingsTestHooks?: {
      readAutoLogoutSettings: () => { enabled: boolean; promptSeconds: number; logoutSeconds: number };
      minPromptSeconds: number;
      maxPromptSeconds: number;
      minLogoutSeconds: number;
      maxLogoutSeconds: number;
      minimumGapSeconds: number;
      readSessionTimeoutStartRequested?: () => boolean;
    };
    __loopawareDashboardIdleTestHooks?: {
      forcePrompt: () => void;
      forceLogout: () => void;
    };
  }
}
