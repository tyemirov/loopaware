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

  constructor(
    private readonly config: RuntimeConfig,
    private readonly fetcher: typeof fetch = fetch,
  ) {}

  async me(): Promise<Account> {
    return this.apiRequest<Account>("/api/me");
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
    const [messages, subscribers, stats, trend, attribution, engagement, devices, locations, sentryIssues, mobileApps, teamMembers, trafficReportSchedule, healthMonitor] =
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
        this.apiRequest<SiteHealthMonitor>(`/api/sites/${siteID}/health-monitor`),
      ]);

    return {
      messages: readCollection<FeedbackMessage>(messages, "messages"),
      subscribers: readCollection<Subscriber>(subscribers, "subscribers"),
      stats: {
        ...stats,
        top_pages: readCollection<TopPage>(stats, "top_pages"),
        recent_visits: readCollection<VisitLogEntry>(stats, "recent_visits"),
      },
      trend: {
        ...trend,
        trend: readCollection<VisitTrendPoint>(trend, "trend"),
      },
      attribution: {
        ...attribution,
        sources: readCollection<AttributionPoint>(attribution, "sources"),
        mediums: readCollection<AttributionPoint>(attribution, "mediums"),
        campaigns: readCollection<AttributionPoint>(attribution, "campaigns"),
      },
      engagement,
      devices: {
        ...devices,
        device_types: readCollection<DeviceTypePoint>(devices, "device_types"),
        top_resolutions: readCollection<AttributionPoint>(devices, "top_resolutions"),
        top_viewports: readCollection<AttributionPoint>(devices, "top_viewports"),
      },
      locations: {
        ...locations,
        locations: readCollection<LocationPoint>(locations, "locations"),
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
    try {
      return await this.request<T>(this.config.apiBaseUrl, path, options);
    } catch (error) {
      if (!isUnauthorizedApiError(error)) {
        throw error;
      }
      const recovered = await this.recoverAPIAuthentication();
      if (!recovered) {
        throw error;
      }
      return this.request<T>(this.config.apiBaseUrl, path, options);
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

function joinUrl(baseUrl: string, path: string): string {
  return `${baseUrl.replace(/\/+$/, "")}/${path.replace(/^\/+/, "")}`;
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
