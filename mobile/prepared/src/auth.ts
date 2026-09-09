import * as AuthSession from "expo-auth-session";
import * as Crypto from "expo-crypto";
import * as WebBrowser from "expo-web-browser";
import { Platform } from "react-native";
import { LoopAwareApiError, isMissingSessionError, type LoopAwareApiClient } from "./api";
import type { Account, NativeGoogleConfig, NativeGooglePlatform } from "./types";

const defaultGoogleScopes = Object.freeze(["openid", "email", "profile"]);

WebBrowser.maybeCompleteAuthSession();

export class AuthController {
  constructor(private readonly api: LoopAwareApiClient) {}

  async restore(): Promise<Account | null> {
    try {
      return await this.api.me();
    } catch (error) {
      if (!isUnauthorizedError(error)) {
        throw error;
      }
    }
    try {
      await this.api.refreshSession();
      return await this.api.me();
    } catch (error) {
      if (isMissingSessionError(error)) {
        return null;
      }
      throw error;
    }
  }

  async signIn(): Promise<Account> {
    const platform = nativeGooglePlatform();
    const [nativeConfig, nonceToken] = await Promise.all([this.api.nativeGoogleConfig(platform), this.api.createAuthNonce()]);
    const nativeClient = resolveNativeGoogleClient(nativeConfig, platform);
    const authRequest = new AuthSession.AuthRequest({
      clientId: nativeClient.clientId,
      redirectUri: nativeClient.redirectUri,
      responseType: AuthSession.ResponseType.Code,
      scopes: nativeConfig.scopes?.length ? nativeConfig.scopes : [...defaultGoogleScopes],
      extraParams: { nonce: nonceToken || Crypto.randomUUID() },
      usePKCE: true,
    });
    const authResult = await authRequest.promptAsync({ authorizationEndpoint: nativeConfig.authorization_endpoint });
    if (authResult.type !== "success") {
      throw new Error(`native_google_sign_in_incomplete: ${authResult.type}`);
    }
    const authorizationCode = authResult.params.code;
    if (!authorizationCode) {
      throw new Error("native_google_sign_in_missing_code: Google did not return an authorization code");
    }
    if (!authRequest.codeVerifier) {
      throw new Error("native_google_sign_in_missing_pkce_verifier: AuthSession did not create a PKCE verifier");
    }
    const tokenResponse = await AuthSession.exchangeCodeAsync(
      {
        clientId: nativeClient.clientId,
        code: authorizationCode,
        redirectUri: nativeClient.redirectUri,
        extraParams: { code_verifier: authRequest.codeVerifier },
      },
      { tokenEndpoint: nativeConfig.token_endpoint },
    );
    if (!tokenResponse.idToken) {
      throw new Error("native_google_sign_in_missing_id_token: Google token exchange did not return an id_token");
    }
    await this.api.exchangeNativeGoogleCredential({
      googleIdToken: tokenResponse.idToken,
      nonceToken,
      platform,
      redirectUri: nativeClient.redirectUri,
    });
    return this.api.me();
  }

  async signOut(): Promise<void> {
    await this.api.logout();
  }
}

function nativeGooglePlatform(): NativeGooglePlatform {
  return Platform.OS === "android" ? "android" : "ios";
}

function resolveNativeGoogleClient(nativeConfig: NativeGoogleConfig, platform: NativeGooglePlatform): { clientId: string; redirectUri: string } {
  const platformClient = nativeConfig.clients?.find((client) => client.platform === platform);
  const clientId = platformClient?.client_id || nativeConfig.client_id || nativeConfig.client_ids?.[0] || "";
  const redirectUri = platformClient?.redirect_uris?.[0] || nativeConfig.redirect_uris?.[0] || "";
  if (!clientId) {
    throw new Error(`native_google_config_invalid.${platform}: missing client id`);
  }
  if (!redirectUri) {
    throw new Error(`native_google_config_invalid.${platform}: missing redirect URI`);
  }
  return { clientId, redirectUri };
}

function isUnauthorizedError(error: unknown): boolean {
  return error instanceof LoopAwareApiError && error.status === 401;
}
