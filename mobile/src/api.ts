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
  LocationDistribution,
} from "./types";

type RequestOptions = {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  headers?: Record<string, string>;
  body?: Record<string, unknown>;
};

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
  constructor(
    private readonly config: RuntimeConfig,
    private readonly fetcher: typeof fetch = fetch,
  ) {}

  async me(): Promise<Account> {
    return this.apiRequest<Account>("/api/me");
  }

  async sites(): Promise<SitesResponse> {
    return this.apiRequest<SitesResponse>("/api/sites");
  }

  async siteDashboard(site: Site, interval: TrafficInterval): Promise<SiteDashboard> {
    const siteID = encodeURIComponent(site.id);
    const intervalQuery = `?interval=${encodeURIComponent(interval)}`;
    const [messages, subscribers, stats, trend, attribution, engagement, devices, locations, sentryIssues, mobileApps, teamMembers, trafficReportSchedule] =
      await Promise.all([
        this.apiRequest<MessagesResponse>(`/api/sites/${siteID}/messages`),
        this.apiRequest<SubscribersResponse>(`/api/sites/${siteID}/subscribers`),
        this.apiRequest<VisitStats>(`/api/sites/${siteID}/visits/stats${intervalQuery}`),
        this.apiRequest<VisitTrend>(`/api/sites/${siteID}/visits/trend${intervalQuery}`),
        this.apiRequest<VisitAttribution>(`/api/sites/${siteID}/visits/attribution${intervalQuery}`),
        this.apiRequest<VisitEngagement>(`/api/sites/${siteID}/visits/engagement${intervalQuery}`),
        this.apiRequest<DeviceBreakdown>(`/api/sites/${siteID}/visits/devices${intervalQuery}`),
        this.apiRequest<LocationDistribution>(`/api/sites/${siteID}/visits/locations${intervalQuery}`),
        this.apiRequest<SentryIssuesResponse>(`/api/sites/${siteID}/sentry/issues`),
        this.apiRequest<MobileAppsResponse>(`/api/sites/${siteID}/mobile-apps`),
        site.access_role === "admin" ? this.apiRequest<TeamMembersResponse>(`/api/sites/${siteID}/team`) : Promise.resolve({ site_id: site.id, team_members: [] }),
        site.access_role === "admin"
          ? this.apiRequest<TrafficReportSchedule>(`/api/sites/${siteID}/traffic-report-schedule`)
          : Promise.resolve(null),
      ]);

    return {
      messages: messages.messages,
      subscribers: subscribers.subscribers,
      stats,
      trend,
      attribution,
      engagement,
      devices,
      locations,
      sentryIssues: sentryIssues.issues,
      mobileApps: mobileApps.mobile_apps,
      teamMembers: teamMembers.team_members,
      trafficReportSchedule,
    };
  }

  async portfolioDashboard(): Promise<PortfolioDashboard> {
    const [reportsResponse, defaultReport, schedule] = await Promise.all([
      this.apiRequest<PortfolioReportsResponse>("/api/reports/traffic/portfolio/reports"),
      this.apiRequest<PortfolioTrafficReport>("/api/reports/traffic/portfolio?days=30"),
      this.apiRequest<TrafficReportSchedule>("/api/reports/traffic/portfolio/schedule"),
    ]);
    return {
      reports: reportsResponse.reports,
      availableSites: reportsResponse.available_sites,
      defaultReport,
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

  private apiRequest<T>(path: string, options: RequestOptions = {}): Promise<T> {
    return this.request<T>(this.config.apiBaseUrl, path, options);
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
}

function joinUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
}
