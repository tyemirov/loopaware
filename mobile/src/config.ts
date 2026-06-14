import Constants from "expo-constants";
import type { RuntimeConfig } from "./types";

const defaultApiBaseUrl = "https://loopaware-api.mprlab.com";
const defaultTauthBaseUrl = "https://tauth-api.mprlab.com";
const defaultTenantId = "loopaware";

export function loadRuntimeConfig(): RuntimeConfig {
  const loopAwareConfig = Constants.expoConfig?.extra?.loopAware as Partial<RuntimeConfig> | undefined;
  return {
    apiBaseUrl: parseHttpBaseUrl(loopAwareConfig?.apiBaseUrl, defaultApiBaseUrl, "apiBaseUrl"),
    tauthBaseUrl: parseHttpBaseUrl(loopAwareConfig?.tauthBaseUrl, defaultTauthBaseUrl, "tauthBaseUrl"),
    tauthTenantId: parseTenantId(loopAwareConfig?.tauthTenantId),
  };
}

function parseHttpBaseUrl(rawValue: unknown, fallbackValue: string, fieldName: string): string {
  const candidate = String(rawValue || fallbackValue).trim();
  const parsed = new URL(candidate);
  if (parsed.protocol !== "https:" && parsed.protocol !== "http:") {
    throw new Error(`mobile_config_invalid.${fieldName}: expected http or https URL`);
  }
  return parsed.origin + parsed.pathname.replace(/\/+$/, "");
}

function parseTenantId(rawValue: unknown): string {
  const tenantId = String(rawValue || defaultTenantId).trim();
  if (!tenantId) {
    throw new Error("mobile_config_invalid.tauthTenantId: tenant id is required");
  }
  return tenantId;
}
