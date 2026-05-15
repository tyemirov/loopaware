export {};

declare global {
  interface Window {
    __loopawareFetchIntercept?: {
      requests: Array<{ url: string; method: string; body: string; status: number }>;
      storageKey?: string;
      originalFetch?: typeof fetch;
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
      started?: () => boolean;
    };
    MPRUI?: {
      testing?: {
        authenticate?: (
          host: Element,
          profile: {
            user_id: string;
            user_email: string;
            email: string;
            display: string;
            avatar_url: string;
            roles: string[];
          }
        ) => unknown;
        unauthenticate?: (host: Element) => unknown;
      };
    };
  }
}
