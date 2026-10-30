import type {
  Account,
  MessagesResponse,
  MobileAppsResponse,
  NativeGoogleConfig,
  NativeGoogleCredential,
  NativeGooglePlatform,
  PortfolioDashboard,
  PortfolioReportsResponse,
  PortfolioTrafficReport,
  RuntimeConfig,
  SentryIssuesResponse,
  Site,
  SiteDashboard,
  SiteHealthMonitor,
  SitesResponse,
  SubscribersResponse,
  TeamMembersResponse,
  TrafficInterval,
  TrafficReportSchedule,
  VisitAttribution,
  VisitEngagement,
  VisitStats,
  VisitTrend,
  DeviceBreakdown,
  AttributionPoint,
  DeviceTypePoint,
  FeedbackMessage,
  LocationPoint,
  MobileAppRegistration,
  PortfolioReportDefinition,
  PortfolioTrafficSite,
  LocationDistribution,
  SentryIssue,
  Subscriber,
  TeamMember,
  TopPage,
  VisitLogEntry,
  VisitTrendPoint,
} from "./types";

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  headers?: Record<string, string>;
  body?: Record<string, unknown>;
};

const maxAvatarBytes = 1 << 20;
const uniqueVisitorsSortQuery = "&sort=unique_visitors";
const maxFaviconBytes = 128 * 1024;
const faviconTransferLimit = 4;
type ImageContract = { subject: string; limit: number; sizeLabel: string; invalidCode: string; loadCode: string };
type NativeImageBlob = Blob & { close: () => void };

export class LoopAwareApiError extends Error {
  readonly status: number;
  readonly code: string;

  constructor(status: number, code: string, message: string) {
    super(message);
    this.name = "LoopAwareApiError";
    this.status = status;
    this.code = code;
  }
}

export class LoopAwareApiClient {
  private authenticationRecoveryPromise: Promise<boolean> | null = null;
  private faviconTransfers = 0;
  private readonly faviconQueue: Array<() => void> = [];

  constructor(
    private readonly config: RuntimeConfig,
    private readonly fetcher: typeof fetch = fetch,
  ) {}

  async me(): Promise<Account> {
    const account = await this.apiRequest<Account>("/api/me");
    const avatar = account.avatar;
    if (avatar === undefined) return account;
    if (typeof avatar !== "object" || avatar === null || (avatar.url !== undefined && typeof avatar.url !== "string")) {
      throw new LoopAwareApiError(200, "mobile_api_invalid_avatar", "Load account avatar: expected an image address.");
    }
    const avatarURL = avatar.url?.trim();
    if (!avatarURL) return { ...account, avatar: undefined };
    let url: URL;
    try {
      url = new URL(avatarURL, `${this.config.apiBaseUrl}/`);
    } catch {
      throw new LoopAwareApiError(200, "mobile_api_invalid_avatar", "Load account avatar: invalid image address.");
    }
    if (url.protocol !== "https:" && url.protocol !== "http:" && !(url.protocol === "data:" && avatarURL.startsWith("data:image/"))) {
      throw new LoopAwareApiError(200, "mobile_api_invalid_avatar", "Load account avatar: unsupported image address.");
    }
    return { ...account, avatar: { url: url.href } };
  }

  async accountAvatar(account: Account, signal: AbortSignal): Promise<string | null> {
    const uri = account.avatar?.url;
    if (!uri) return null;
    if (uri.startsWith("data:image/")) return uri;
    return this.imageData(uri, signal, { subject: "account photo", limit: maxAvatarBytes, sizeLabel: "1 MiB", invalidCode: "mobile_api_invalid_avatar", loadCode: "mobile_avatar_load_failed" });
  }

  async siteFavicon(site: Site, signal: AbortSignal): Promise<string | null> {
    if (!site.favicon_url) return null;
    let uri: URL;
    try { uri = new URL(site.favicon_url, `${this.config.apiBaseUrl}/`); }
    catch { throw new LoopAwareApiError(200, "mobile_api_invalid_favicon", `Load site ${site.id} favicon: invalid image address.`); }
    if (uri.origin !== new URL(this.config.apiBaseUrl).origin || !["https:", "http:"].includes(uri.protocol)) {
      throw new LoopAwareApiError(200, "mobile_api_invalid_favicon", `Load site ${site.id} favicon: expected an API image address.`);
    }
    await this.acquireFaviconSlot(signal);
    try {
      return await this.imageData(uri.href, signal, { subject: `site ${site.id} favicon`, limit: maxFaviconBytes, sizeLabel: "128 KiB", invalidCode: "mobile_api_invalid_favicon", loadCode: "mobile_favicon_load_failed" });
    } finally {
      this.faviconTransfers--;
      this.faviconQueue.shift()?.();
    }
  }

  private acquireFaviconSlot(signal: AbortSignal): Promise<void> {
    return new Promise((resolve, reject) => {
      const cancel = () => {
        const index = this.faviconQueue.indexOf(start);
        if (index >= 0) this.faviconQueue.splice(index, 1);
        reject(new Error("Load site favicon: request cancelled."));
      };
      const start = () => {
        signal.removeEventListener("abort", cancel);
        this.faviconTransfers++;
        resolve();
      };
      if (signal.aborted) { cancel(); return; }
      if (this.faviconTransfers < faviconTransferLimit) start();
      else { this.faviconQueue.push(start); signal.addEventListener("abort", cancel, { once: true }); }
    });
  }

  private async imageData(uri: string, signal: AbortSignal, contract: ImageContract): Promise<string> {
    const apiOrigin = new URL(uri).origin === new URL(this.config.apiBaseUrl).origin;
    const load = async () => {
      const response = await this.fetcher(uri, {
        credentials: apiOrigin ? "include" : "omit", cache: "no-store", signal,
        headers: { Accept: "image/*" },
      });
      if (!response.ok) {
        throw new LoopAwareApiError(response.status, contract.loadCode, `Load ${contract.subject}: HTTP ${response.status}.`);
      }
      const blob = await response.blob() as NativeImageBlob;
      try {
        const imageType = blob.type.startsWith("image/") || (apiOrigin && blob.type === "application/octet-stream");
        if (!imageType || blob.size === 0 || blob.size > contract.limit) {
          throw new LoopAwareApiError(502, contract.invalidCode, `Load ${contract.subject}: expected an image of at most ${contract.sizeLabel}.`);
        }
        return await readImageDataURL(blob, signal, contract.subject);
      } finally {
        blob.close();
      }
    };
    try {
      return await (apiOrigin ? this.withAPIAuthentication(load, signal) : load());
    } catch (error) {
      if (error instanceof LoopAwareApiError) throw error;
      throw new Error(`Load ${contract.subject}: image transfer failed.`, { cause: error });
    }
  }

  async sites(): Promise<SitesResponse> {
    const sitesResponse = await this.apiRequest<SitesResponse>("/api/sites");
    return {
      ...sitesResponse,
      sites: readCollection<Site>(sitesResponse, "sites"),
    };
  }

  async siteDashboard(site: Site, interval: TrafficInterval): Promise<SiteDashboard> {
    const siteID = encodeURIComponent(site.id);
    const intervalQuery = `?interval=${encodeURIComponent(interval)}`;
    const rankedIntervalQuery = site.traffic_profile === "detailed" ? `${intervalQuery}${uniqueVisitorsSortQuery}` : intervalQuery;
    const [messages, subscribers, stats, trend, attribution, engagement, devices, locations, sentryIssues, mobileApps, teamMembers, trafficReportSchedule, healthMonitor] =
      await Promise.all([
        this.apiRequest<MessagesResponse>(`/api/sites/${siteID}/messages`),
        this.apiRequest<SubscribersResponse>(`/api/sites/${siteID}/subscribers`),
        this.apiRequest<VisitStats>(`/api/sites/${siteID}/visits/stats${rankedIntervalQuery}`),
        this.apiRequest<VisitTrend>(`/api/sites/${siteID}/visits/trend${intervalQuery}`),
        site.traffic_profile === "aggregate" ? Promise.resolve(null) : this.apiRequest<VisitAttribution>(`/api/sites/${siteID}/visits/attribution${rankedIntervalQuery}`),
        site.traffic_profile === "aggregate" ? Promise.resolve(null) : this.apiRequest<VisitEngagement>(`/api/sites/${siteID}/visits/engagement${intervalQuery}`),
        site.traffic_profile === "aggregate" ? Promise.resolve(null) : this.apiRequest<DeviceBreakdown>(`/api/sites/${siteID}/visits/devices${rankedIntervalQuery}`),
        site.traffic_profile === "aggregate" ? Promise.resolve(null) : this.apiRequest<LocationDistribution>(`/api/sites/${siteID}/visits/locations${rankedIntervalQuery}`),
        this.apiRequest<SentryIssuesResponse>(`/api/sites/${siteID}/sentry/issues`),
        this.apiRequest<MobileAppsResponse>(`/api/sites/${siteID}/mobile-apps`),
        site.access_role === "admin" ? this.apiRequest<TeamMembersResponse>(`/api/sites/${siteID}/team`) : Promise.resolve({ site_id: site.id, team_members: [] }),
        site.access_role === "admin"
          ? this.apiRequest<TrafficReportSchedule>(`/api/sites/${siteID}/traffic-report-schedule`)
          : Promise.resolve(null),
        this.apiRequest<SiteHealthMonitor>(`/api/sites/${siteID}/health-monitor`),
      ]);

    if (site.traffic_profile === "detailed") readTrafficCount(stats.unique_visitor_count, "site unique visitors");
    return {
      messages: readCollection<FeedbackMessage>(messages, "messages"),
      subscribers: readCollection<Subscriber>(subscribers, "subscribers"),
      stats: {
        ...stats,
        top_pages: readVisitorCollection<TopPage>(stats, "top_pages"),
        recent_visits: readCollection<VisitLogEntry>(stats, "recent_visits"),
      },
      trend: {
        ...trend,
        trend: readSiteTrend(trend, site),
      },
      attribution: attribution === null ? null : {
        ...attribution,
        sources: readVisitorCollection<AttributionPoint>(attribution, "sources"),
        mediums: readVisitorCollection<AttributionPoint>(attribution, "mediums"),
        campaigns: readVisitorCollection<AttributionPoint>(attribution, "campaigns"),
      },
      engagement,
      devices: devices === null ? null : {
        ...devices,
        device_types: readVisitorCollection<DeviceTypePoint>(devices, "device_types"),
        top_resolutions: readVisitorCollection<AttributionPoint>(devices, "top_resolutions"),
        top_viewports: readVisitorCollection<AttributionPoint>(devices, "top_viewports"),
      },
      locations: locations === null ? null : {
        ...locations,
        locations: readVisitorCollection<LocationPoint>(locations, "locations"),
      },
      sentryIssues: readCollection<SentryIssue>(sentryIssues, "issues"),
      mobileApps: readCollection<MobileAppRegistration>(mobileApps, "mobile_apps"),
      teamMembers: readCollection<TeamMember>(teamMembers, "team_members"),
      trafficReportSchedule,
      healthMonitor,
    };
  }

  async portfolioDashboard(): Promise<PortfolioDashboard> {
    const [reportsResponse, defaultReport, schedule] = await Promise.all([
      this.apiRequest<PortfolioReportsResponse>("/api/reports/traffic/portfolio/reports"),
      this.apiRequest<PortfolioTrafficReport>("/api/reports/traffic/portfolio?days=30"),
      this.apiRequest<TrafficReportSchedule>("/api/reports/traffic/portfolio/schedule"),
    ]);
    return {
      reports: readCollection<PortfolioReportDefinition>(reportsResponse, "reports"),
      availableSites: readCollection<PortfolioTrafficSite>(reportsResponse, "available_sites"),
      defaultReport: {
        ...defaultReport,
        trend: readCollection<VisitTrendPoint>(defaultReport, "trend"),
        sites: readCollection<PortfolioTrafficSite>(defaultReport, "sites"),
      },
      schedule,
    };
  }

  async nativeGoogleConfig(platform: NativeGooglePlatform): Promise<NativeGoogleConfig> {
    return this.tauthRequest<NativeGoogleConfig>(`/auth/google/native/config?platform=${encodeURIComponent(platform)}`);
  }

  async createAuthNonce(): Promise<string> {
    const response = await this.tauthRequest<{ nonce: string }>("/auth/nonce", { method: "POST" });
    return response.nonce;
  }

  async exchangeNativeGoogleCredential(credential: NativeGoogleCredential): Promise<void> {
    await this.tauthRequest<Record<string, unknown>>("/auth/google/native", {
      method: "POST",
      body: {
        google_id_token: credential.googleIdToken,
        nonce_token: credential.nonceToken,
        platform: credential.platform,
        redirect_uri: credential.redirectUri,
      },
    });
  }

  async refreshSession(): Promise<void> {
    await this.tauthRequest<Record<string, unknown>>("/auth/refresh", { method: "POST" });
  }

  async logout(): Promise<void> {
    await this.tauthRequest<Record<string, unknown>>("/auth/logout", { method: "POST" });
  }

  private async apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
    return this.withAPIAuthentication(() => this.request<T>(this.config.apiBaseUrl, path, options));
  }

  private async withAPIAuthentication<T>(operation: () => Promise<T>, signal?: AbortSignal): Promise<T> {
    try {
      return await operation();
    } catch (error) {
      if (signal?.aborted || !isUnauthorizedApiError(error)) {
        throw error;
      }
      const recovered = await this.recoverAPIAuthentication();
      if (signal?.aborted || !recovered) {
        throw error;
      }
      return operation();
    }
  }

  private tauthRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
    return this.request<T>(this.config.tauthBaseUrl, path, {
      ...options,
      headers: {
        "X-TAuth-Tenant": this.config.tauthTenantId,
        ...(options.headers || {}),
      },
    });
  }

  private async request<T>(baseUrl: string, path: string, options: RequestOptions): Promise<T> {
    const headers: Record<string, string> = {
      Accept: "application/json",
      "Content-Type": "application/json",
      ...(options.headers || {}),
    };
    const response = await this.fetcher(joinUrl(baseUrl, path), {
      method: options.method || "GET",
      headers,
      credentials: "include",
      body: options.body ? JSON.stringify(options.body) : undefined,
    });
    const data = await response.json().catch(() => ({}));
    if (!response.ok) {
      const errorCode = typeof data.error === "string" ? data.error : "api_error";
      const errorMessage = typeof data.message === "string" ? data.message : `Request failed with ${response.status}`;
      throw new LoopAwareApiError(response.status, errorCode, errorMessage);
    }
    return data as T;
  }

  private recoverAPIAuthentication(): Promise<boolean> {
    if (!this.authenticationRecoveryPromise) {
      this.authenticationRecoveryPromise = this.refreshAPIAuthentication().finally(() => {
        this.authenticationRecoveryPromise = null;
      });
    }
    return this.authenticationRecoveryPromise;
  }

  private async refreshAPIAuthentication(): Promise<boolean> {
    try {
      await this.refreshSession();
      return true;
    } catch (error) {
      if (isMissingSessionError(error)) {
        return false;
      }
      throw error;
    }
  }
}

function readImageDataURL(blob: Blob, signal: AbortSignal, subject: string): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    const cancelled = () => {
      reader.abort();
      reject(new Error(`Load ${subject}: request cancelled.`));
    };
    if (signal.aborted) { cancelled(); return; }
    signal.addEventListener("abort", cancelled, { once: true });
    reader.onload = () => {
      signal.removeEventListener("abort", cancelled);
      if (typeof reader.result !== "string") {
        reject(new Error(`Load ${subject}: invalid image conversion result.`));
      } else resolve(reader.result);
    };
    reader.onerror = () => {
      signal.removeEventListener("abort", cancelled);
      reject(new Error(`Load ${subject}: image conversion failed.`, { cause: reader.error }));
    };
    try {
      reader.readAsDataURL(blob);
    } catch (error) {
      signal.removeEventListener("abort", cancelled);
      reject(new Error(`Load ${subject}: image conversion failed.`, { cause: error }));
    }
  });
}

function joinUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
}

function readSiteTrend(payload: VisitTrend, site: Site): VisitTrendPoint[] {
  const recentTrend = readCollection<VisitTrendPoint>(payload, "trend").slice(-10);
  if (site.traffic_profile === "aggregate") {
    return recentTrend;
  }
  return recentTrend.map((point) => {
    const uniqueVisitors = point.unique_visitors;
    if (typeof uniqueVisitors !== "number" || !Number.isSafeInteger(uniqueVisitors) || uniqueVisitors < 0) {
      throw new LoopAwareApiError(502, "mobile_api_invalid_trend", `Read site ${site.id} trend for ${point.date}: invalid unique visitor count.`);
    }
    return { ...point, unique_visitors: uniqueVisitors };
  }).sort((left, right) => right.unique_visitors - left.unique_visitors || right.date.localeCompare(left.date));
}

function readTrafficCount(value: unknown, subject: string): number {
  if (typeof value !== "number" || !Number.isSafeInteger(value) || value < 0) {
    throw new LoopAwareApiError(502, "mobile_api_invalid_traffic", `Read ${subject}: expected a nonnegative integer count.`);
  }
  return value;
}

function readVisitorCollection<T extends { visit_count: number; unique_visitor_count: number }>(payload: object, fieldName: string): T[] {
  return readCollection<T>(payload, fieldName).map(item => {
    if (typeof item !== "object" || item === null) {
      throw new LoopAwareApiError(502, "mobile_api_invalid_traffic", `Read ${fieldName}: expected a traffic bucket.`);
    }
    readTrafficCount(item.unique_visitor_count, `${fieldName} unique visitors`);
    readTrafficCount(item.visit_count, `${fieldName} page views`);
    return item;
  });
}

function readCollection<T>(payload: object, fieldName: string): T[] {
  const value = (payload as Record<string, unknown>)[fieldName];
  if (value === null) {
    return [];
  }
  if (!Array.isArray(value)) {
    throw new LoopAwareApiError(502, "mobile_api_invalid_collection", `LoopAware API returned invalid ${fieldName} payload.`);
  }
  return value as T[];
}

export function isMissingSessionError(error: unknown): boolean {
  if (!(error instanceof LoopAwareApiError)) {
    return false;
  }
  if (error.status === 401) {
    return true;
  }
  return error.status === 404 && error.code === "api_error";
}

function isUnauthorizedApiError(error: unknown): boolean {
  return error instanceof LoopAwareApiError && error.status === 401;
}
