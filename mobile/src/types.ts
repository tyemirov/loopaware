export type NativeGooglePlatform = "ios" | "android";

export type TrafficInterval = "all" | "30days" | "1day";

export type DashboardTab = "overview" | "feedback" | "traffic" | "subscribers" | "sentry" | "reports";

export type RuntimeConfig = {
  apiBaseUrl: string;
  tauthBaseUrl: string;
  tauthTenantId: string;
};

export type Account = {
  email: string;
  name: string;
  role: string;
  avatar?: {
    url?: string;
  };
};

export type Site = {
  id: string;
  name: string;
  allowed_origin: string;
  subscribe_allowed_origins: string;
  widget_allowed_origins: string;
  traffic_allowed_origins: string;
  owner_email: string;
  favicon_url: string;
  widget: string;
  created_at: number;
  feedback_count: number;
  subscriber_count: number;
  visit_count: number;
  unique_visitor_count: number;
  sentry_token_configured: boolean;
  widget_bubble_side: string;
  widget_bubble_bottom_offset: number;
  widget_accent_color: string;
  widget_show_message_input: boolean;
  widget_show_sentiment_buttons: boolean;
  access_role: "admin" | "team_member";
};

export type SitesResponse = {
  sites: Site[];
};

export type FeedbackMessage = {
  id: string;
  contact: string;
  message: string;
  sentiment: string;
  ip: string;
  user_agent: string;
  created_at: number;
  delivery: string;
  source_kind: string;
  mobile_client_id: string;
  screen_name: string;
  screen_path: string;
  app_platform: string;
  app_identifier: string;
  app_version: string;
  app_build: string;
  app_environment: string;
  context?: Record<string, unknown>;
};

export type MessagesResponse = {
  site_id: string;
  messages: FeedbackMessage[];
};

export type Subscriber = {
  id: string;
  email: string;
  name: string;
  status: string;
  created_at: number;
  confirmed_at: number;
  unsubscribed_at: number;
};

export type SubscribersResponse = {
  site_id: string;
  subscribers: Subscriber[];
};

export type TopPage = {
  path: string;
  visit_count: number;
};

export type VisitLogEntry = {
  url: string;
  path: string;
  ip: string;
  country: string;
  browser: string;
  user_agent: string;
  referrer: string;
  visitor_id: string;
  occurred_at: number;
};

export type VisitStats = {
  site_id: string;
  interval: TrafficInterval;
  visit_count: number;
  unique_visitor_count: number;
  top_pages: TopPage[];
  recent_visits: VisitLogEntry[];
};

export type VisitTrendPoint = {
  date: string;
  page_views: number;
  unique_visitors: number;
};

export type VisitTrend = {
  site_id: string;
  interval: string;
  days: number;
  trend: VisitTrendPoint[];
};

export type AttributionPoint = {
  value: string;
  visit_count: number;
};

export type VisitAttribution = {
  site_id: string;
  interval: TrafficInterval;
  limit: number;
  sources: AttributionPoint[];
  mediums: AttributionPoint[];
  campaigns: AttributionPoint[];
};

export type VisitEngagement = {
  site_id: string;
  interval: string;
  days: number;
  tracked_visitor_count: number;
  returning_visitor_count: number;
  returning_visitor_rate: number;
  average_pages_per_visitor: number;
  depth_distribution: {
    single_page: number;
    two_to_three_pages: number;
    four_to_seven_pages: number;
    eight_or_more_pages: number;
  };
  observed_time_distribution: {
    under_30_seconds: number;
    between_30_and_119_seconds: number;
    between_120_and_599_seconds: number;
    at_least_600_seconds: number;
  };
};

export type DeviceTypePoint = {
  device_type: string;
  visit_count: number;
};

export type DeviceBreakdown = {
  site_id: string;
  interval: TrafficInterval;
  limit: number;
  device_types: DeviceTypePoint[];
  top_resolutions: AttributionPoint[];
  top_viewports: AttributionPoint[];
};

export type LocationPoint = {
  label: string;
  source: string;
  signal: string;
  country: string;
  region: string;
  city: string;
  latitude: number;
  longitude: number;
  confidence: number;
  visit_count: number;
};

export type LocationDistribution = {
  site_id: string;
  interval: TrafficInterval;
  limit: number;
  locations: LocationPoint[];
};

export type SentryIssue = {
  id: string;
  title: string;
  status: string;
  level: string;
  platform: string;
  environment: string;
  release: string;
  first_seen_at: number;
  last_seen_at: number;
  occurrence_count: number;
};

export type SentryIssuesResponse = {
  site_id: string;
  issues: SentryIssue[];
};

export type MobileAppRegistration = {
  id: string;
  client_id: string;
  platform: string;
  app_identifier: string;
  display_name: string;
  enabled: boolean;
  created_at: number;
  updated_at: number;
};

export type MobileAppsResponse = {
  site_id: string;
  mobile_apps: MobileAppRegistration[];
};

export type TeamMember = {
  id: string;
  email: string;
  added_by_email: string;
  created_at: number;
};

export type TeamMembersResponse = {
  site_id: string;
  team_members: TeamMember[];
};

export type TrafficReportSchedule = {
  site_id: string;
  report_id: string;
  enabled: boolean;
  frequency: string;
  recipient_email: string;
  recipient_mode: string;
  recipient_emails: string[];
  timezone: string;
  send_hour: number;
  send_minute: number;
  weekday: number;
  month_day: number;
  next_send_at: number;
  last_sent_at: number;
  last_status: string;
  last_error: string;
  email_enabled: boolean;
  persisted: boolean;
};

export type PortfolioReportDefinition = {
  id: string;
  name: string;
  site_ids: string[];
  is_default: boolean;
};

export type PortfolioTrafficSite = {
  site_id: string;
  site_name: string;
  visit_count: number;
  unique_visitor_count: number;
};

export type PortfolioReportsResponse = {
  reports: PortfolioReportDefinition[];
  available_sites: PortfolioTrafficSite[];
};

export type PortfolioTrafficReport = {
  report_id: string;
  report_name: string;
  scope: string;
  days: number;
  site_count: number;
  visit_count: number;
  unique_visitor_count: number;
  trend: VisitTrendPoint[];
  sites: PortfolioTrafficSite[];
};

export type SiteDashboard = {
  messages: FeedbackMessage[];
  subscribers: Subscriber[];
  stats: VisitStats;
  trend: VisitTrend;
  attribution: VisitAttribution;
  engagement: VisitEngagement;
  devices: DeviceBreakdown;
  locations: LocationDistribution;
  sentryIssues: SentryIssue[];
  mobileApps: MobileAppRegistration[];
  teamMembers: TeamMember[];
  trafficReportSchedule: TrafficReportSchedule | null;
};

export type PortfolioDashboard = {
  reports: PortfolioReportDefinition[];
  availableSites: PortfolioTrafficSite[];
  defaultReport: PortfolioTrafficReport;
  schedule: TrafficReportSchedule | null;
};

export type NativeGoogleConfig = {
  client_id: string;
  client_ids?: string[];
  platform: NativeGooglePlatform;
  redirect_uris: string[];
  clients?: Array<{
    platform: NativeGooglePlatform;
    client_id: string;
    redirect_uris: string[];
  }>;
  authorization_endpoint: string;
  token_endpoint: string;
  scopes: string[];
  response_type: string;
  pkce_required: boolean;
  code_challenge_methods_supported: string[];
};

export type NativeGoogleCredential = {
  googleIdToken: string;
  nonceToken: string;
  platform: NativeGooglePlatform;
  redirectUri: string;
};
