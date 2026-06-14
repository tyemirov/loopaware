import { StatusBar } from "expo-status-bar";
import * as SecureStore from "expo-secure-store";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  ActivityIndicator,
  type DimensionValue,
  Image,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  View,
} from "react-native";
import { SafeAreaProvider, SafeAreaView } from "react-native-safe-area-context";
import { LoopAwareApiClient } from "./src/api";
import { AuthController } from "./src/auth";
import { loadRuntimeConfig } from "./src/config";
import { compactText, formatCount, formatDate, formatDateTime, formatPercent, sentenceCase } from "./src/format";
import type {
  Account,
  DashboardTab,
  FeedbackMessage,
  PortfolioDashboard,
  SentryIssue,
  Site,
  SiteDashboard,
  Subscriber,
  TrafficInterval,
  VisitLogEntry,
  VisitTrendPoint,
} from "./src/types";

const selectedSiteStorageKey = "loopaware.mobile.selectedSiteId";
const dashboardTabs: Array<{ key: DashboardTab; label: string }> = [
  { key: "overview", label: "Overview" },
  { key: "feedback", label: "Feedback" },
  { key: "traffic", label: "Traffic" },
  { key: "subscribers", label: "Subscribers" },
  { key: "sentry", label: "LA Sentry" },
  { key: "reports", label: "Reports" },
];
const trafficIntervals: Array<{ key: TrafficInterval; label: string }> = [
  { key: "all", label: "All" },
  { key: "30days", label: "30d" },
  { key: "1day", label: "1d" },
];

export default function App() {
  const runtimeConfig = useMemo(() => loadRuntimeConfig(), []);
  const apiClient = useMemo(() => new LoopAwareApiClient(runtimeConfig), [runtimeConfig]);
  const authController = useMemo(() => new AuthController(apiClient), [apiClient]);
  const [status, setStatus] = useState<"checking" | "signedOut" | "signedIn">("checking");
  const [account, setAccount] = useState<Account | null>(null);
  const [sites, setSites] = useState<Site[]>([]);
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null);
  const [siteDashboard, setSiteDashboard] = useState<SiteDashboard | null>(null);
  const [portfolioDashboard, setPortfolioDashboard] = useState<PortfolioDashboard | null>(null);
  const [activeTab, setActiveTab] = useState<DashboardTab>("overview");
  const [trafficInterval, setTrafficInterval] = useState<TrafficInterval>("30days");
  const [busy, setBusy] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const loadSiteData = useCallback(
    async (site: Site, interval: TrafficInterval) => {
      setBusy(true);
      setErrorMessage(null);
      try {
        const dashboard = await apiClient.siteDashboard(site, interval);
        setSiteDashboard(dashboard);
      } catch (error) {
        setErrorMessage(errorToMessage(error));
      } finally {
        setBusy(false);
      }
    },
    [apiClient],
  );

  const loadAuthenticatedState = useCallback(
    async (mode: "restore" | "current", preferredSiteId: string | null, interval: TrafficInterval) => {
      setBusy(true);
      setErrorMessage(null);
      try {
        const currentAccount = mode === "restore" ? await authController.restore() : await apiClient.me();
        if (!currentAccount) {
          setStatus("signedOut");
          setAccount(null);
          setSites([]);
          setSelectedSiteId(null);
          setSiteDashboard(null);
          setPortfolioDashboard(null);
          return;
        }
        const [sitesResponse, persistedSiteId] = await Promise.all([
          apiClient.sites(),
          SecureStore.getItemAsync(selectedSiteStorageKey).catch(() => null),
        ]);
        const nextSelectedSiteId = resolveSelectedSiteId(sitesResponse.sites, preferredSiteId, persistedSiteId);
        const selectedSite = sitesResponse.sites.find((site) => site.id === nextSelectedSiteId) || null;
        const [nextSiteDashboard, nextPortfolioDashboard] = await Promise.all([
          selectedSite ? apiClient.siteDashboard(selectedSite, interval) : Promise.resolve(null),
          apiClient.portfolioDashboard(),
        ]);
        setAccount(currentAccount);
        setSites(sitesResponse.sites);
        setSelectedSiteId(nextSelectedSiteId);
        setSiteDashboard(nextSiteDashboard);
        setPortfolioDashboard(nextPortfolioDashboard);
        setStatus("signedIn");
      } catch (error) {
        setStatus((currentStatus) => (currentStatus === "checking" ? "signedOut" : currentStatus));
        setErrorMessage(errorToMessage(error));
      } finally {
        setBusy(false);
        setRefreshing(false);
      }
    },
    [apiClient, authController],
  );

  useEffect(() => {
    void loadAuthenticatedState("restore", null, "30days");
  }, [loadAuthenticatedState]);

  const selectedSite = useMemo(() => sites.find((site) => site.id === selectedSiteId) || null, [selectedSiteId, sites]);

  const handleSignIn = useCallback(async () => {
    setBusy(true);
    setErrorMessage(null);
    try {
      const currentAccount = await authController.signIn();
      setAccount(currentAccount);
      await loadAuthenticatedState("current", null, trafficInterval);
    } catch (error) {
      setStatus("signedOut");
      setErrorMessage(errorToMessage(error));
    } finally {
      setBusy(false);
    }
  }, [authController, loadAuthenticatedState]);

  const handleSignOut = useCallback(async () => {
    setBusy(true);
    setErrorMessage(null);
    try {
      await authController.signOut();
    } catch (error) {
      setErrorMessage(errorToMessage(error));
    } finally {
      setAccount(null);
      setSites([]);
      setSelectedSiteId(null);
      setSiteDashboard(null);
      setPortfolioDashboard(null);
      setStatus("signedOut");
      setBusy(false);
    }
  }, [authController]);

  const handleSelectSite = useCallback(
    async (site: Site) => {
      setSelectedSiteId(site.id);
      await SecureStore.setItemAsync(selectedSiteStorageKey, site.id).catch(() => undefined);
      await loadSiteData(site, trafficInterval);
    },
    [loadSiteData, trafficInterval],
  );

  const handleSelectInterval = useCallback(
    async (interval: TrafficInterval) => {
      setTrafficInterval(interval);
      if (selectedSite) {
        await loadSiteData(selectedSite, interval);
      }
    },
    [loadSiteData, selectedSite],
  );

  const handleRefresh = useCallback(async () => {
    setRefreshing(true);
    await loadAuthenticatedState("current", selectedSiteId, trafficInterval);
  }, [loadAuthenticatedState, selectedSiteId, trafficInterval]);

  if (status === "checking") {
    return (
      <SafeAreaProvider>
        <SafeAreaView style={styles.loadingScreen}>
          <StatusBar style="dark" />
          <ActivityIndicator size="large" color={colors.teal} />
          <Text style={styles.loadingText}>Loading LoopAware</Text>
        </SafeAreaView>
      </SafeAreaProvider>
    );
  }

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.safeArea}>
        <StatusBar style="dark" />
        {status === "signedOut" ? (
          <SignedOutScreen busy={busy} errorMessage={errorMessage} onSignIn={handleSignIn} runtimeHost={runtimeConfig.apiBaseUrl} />
        ) : (
          <ScrollView
            contentContainerStyle={styles.scrollContent}
            refreshControl={<RefreshControl refreshing={refreshing} onRefresh={handleRefresh} tintColor={colors.teal} />}
          >
            <Header account={account} busy={busy} onSignOut={handleSignOut} />
            {errorMessage ? <StatusBanner message={errorMessage} /> : null}
            <SitesRail sites={sites} selectedSiteId={selectedSiteId} onSelectSite={handleSelectSite} />
            {selectedSite && siteDashboard ? (
              <>
                <SelectedSiteHeader site={selectedSite} />
                <TabBar activeTab={activeTab} onChange={setActiveTab} />
                {busy ? <InlineLoading /> : null}
                <DashboardTabContent
                  activeTab={activeTab}
                  interval={trafficInterval}
                  onIntervalChange={handleSelectInterval}
                  portfolioDashboard={portfolioDashboard}
                  site={selectedSite}
                  siteDashboard={siteDashboard}
                />
              </>
            ) : (
              <EmptyState title="No sites" detail="Visible sites will appear here after the web dashboard has data for this account." />
            )}
          </ScrollView>
        )}
      </SafeAreaView>
    </SafeAreaProvider>
  );
}

function SignedOutScreen(props: { busy: boolean; errorMessage: string | null; onSignIn: () => void; runtimeHost: string }) {
  return (
    <View style={styles.signedOutScreen}>
      <View style={styles.brandRow}>
        <Image source={require("./assets/icon.png")} style={styles.brandIcon} />
        <View>
          <Text style={styles.brandName}>LoopAware</Text>
          <Text style={styles.brandSubtle}>{props.runtimeHost.replace(/^https?:\/\//, "")}</Text>
        </View>
      </View>
      <View style={styles.signInPanel}>
        <Text style={styles.signInTitle}>Operator dashboard</Text>
        <Text style={styles.signInCopy}>Sign in with the Google account you use on LoopAware web.</Text>
        {props.errorMessage ? <StatusBanner message={props.errorMessage} /> : null}
        <PrimaryButton label={props.busy ? "Opening Google" : "Sign in with Google"} disabled={props.busy} onPress={props.onSignIn} />
      </View>
    </View>
  );
}

function Header(props: { account: Account | null; busy: boolean; onSignOut: () => void }) {
  return (
    <View style={styles.header}>
      <View style={styles.brandRowCompact}>
        <Image source={require("./assets/icon.png")} style={styles.headerIcon} />
        <View>
          <Text style={styles.headerTitle}>LoopAware</Text>
          <Text style={styles.headerSubtitle}>{compactText(props.account?.email, "Signed in")}</Text>
        </View>
      </View>
      <Pressable disabled={props.busy} style={({ pressed }) => [styles.secondaryButton, pressed && styles.pressedButton]} onPress={props.onSignOut}>
        <Text style={styles.secondaryButtonText}>Sign out</Text>
      </Pressable>
    </View>
  );
}

function SitesRail(props: { sites: Site[]; selectedSiteId: string | null; onSelectSite: (site: Site) => void }) {
  if (!props.sites.length) {
    return null;
  }
  return (
    <View style={styles.sitesSection}>
      <Text style={styles.sectionLabel}>Sites</Text>
      <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.sitesRail}>
        {props.sites.map((site) => {
          const selected = site.id === props.selectedSiteId;
          return (
            <Pressable key={site.id} style={[styles.siteChip, selected && styles.siteChipSelected]} onPress={() => props.onSelectSite(site)}>
              <Text style={[styles.siteChipTitle, selected && styles.siteChipTitleSelected]} numberOfLines={1}>
                {site.name}
              </Text>
              <Text style={[styles.siteChipMeta, selected && styles.siteChipMetaSelected]}>{sentenceCase(site.access_role)}</Text>
            </Pressable>
          );
        })}
      </ScrollView>
    </View>
  );
}

function SelectedSiteHeader(props: { site: Site }) {
  return (
    <View style={styles.siteHeader}>
      <View>
        <Text style={styles.siteName}>{props.site.name}</Text>
        <Text style={styles.siteOrigin}>{props.site.allowed_origin}</Text>
      </View>
      <Text style={styles.rolePill}>{props.site.access_role === "admin" ? "Admin" : "Team"}</Text>
    </View>
  );
}

function TabBar(props: { activeTab: DashboardTab; onChange: (tab: DashboardTab) => void }) {
  return (
    <ScrollView horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.tabBar}>
      {dashboardTabs.map((tab) => {
        const active = tab.key === props.activeTab;
        return (
          <Pressable key={tab.key} style={[styles.tabButton, active && styles.tabButtonActive]} onPress={() => props.onChange(tab.key)}>
            <Text style={[styles.tabButtonText, active && styles.tabButtonTextActive]}>{tab.label}</Text>
          </Pressable>
        );
      })}
    </ScrollView>
  );
}

function DashboardTabContent(props: {
  activeTab: DashboardTab;
  interval: TrafficInterval;
  onIntervalChange: (interval: TrafficInterval) => void;
  portfolioDashboard: PortfolioDashboard | null;
  site: Site;
  siteDashboard: SiteDashboard;
}) {
  switch (props.activeTab) {
    case "feedback":
      return <FeedbackSection messages={props.siteDashboard.messages} />;
    case "traffic":
      return <TrafficSection dashboard={props.siteDashboard} interval={props.interval} onIntervalChange={props.onIntervalChange} />;
    case "subscribers":
      return <SubscribersSection subscribers={props.siteDashboard.subscribers} />;
    case "sentry":
      return <SentrySection issues={props.siteDashboard.sentryIssues} site={props.site} />;
    case "reports":
      return <ReportsSection dashboard={props.portfolioDashboard} siteDashboard={props.siteDashboard} site={props.site} />;
    case "overview":
    default:
      return <OverviewSection dashboard={props.siteDashboard} site={props.site} />;
  }
}

function OverviewSection(props: { dashboard: SiteDashboard; site: Site }) {
  return (
    <View style={styles.sectionStack}>
      <StatGrid
        items={[
          { label: "Feedback", value: formatCount(props.site.feedback_count), detail: `${formatCount(props.dashboard.messages.length)} loaded` },
          { label: "Subscribers", value: formatCount(props.site.subscriber_count), detail: `${formatCount(props.dashboard.subscribers.length)} loaded` },
          { label: "Visits", value: formatCount(props.dashboard.stats.visit_count), detail: `${formatCount(props.dashboard.stats.unique_visitor_count)} unique` },
          { label: "Sentry", value: formatCount(props.dashboard.sentryIssues.length), detail: props.site.sentry_token_configured ? "Token configured" : "Token missing" },
        ]}
      />
      <Section title="Site settings">
        <InfoRow label="Owner" value={props.site.owner_email} />
        <InfoRow label="Widget origins" value={compactText(props.site.widget_allowed_origins || props.site.allowed_origin)} />
        <InfoRow label="Subscribe origins" value={compactText(props.site.subscribe_allowed_origins || props.site.allowed_origin)} />
        <InfoRow label="Traffic origins" value={compactText(props.site.traffic_allowed_origins || props.site.allowed_origin)} />
        <InfoRow label="Created" value={formatDate(props.site.created_at)} />
      </Section>
      <Section title="Native apps">
        {props.dashboard.mobileApps.length ? (
          props.dashboard.mobileApps.map((mobileApp) => (
            <ListItem
              key={mobileApp.id}
              title={mobileApp.display_name}
              meta={`${sentenceCase(mobileApp.platform)} · ${mobileApp.app_identifier}`}
              right={mobileApp.enabled ? "Enabled" : "Disabled"}
            />
          ))
        ) : (
          <EmptyInline text="No registered mobile feedback clients." />
        )}
      </Section>
      {props.site.access_role === "admin" ? (
        <Section title="Team">
          {props.dashboard.teamMembers.length ? (
            props.dashboard.teamMembers.map((member) => <ListItem key={member.id} title={member.email} meta={`Added ${formatDate(member.created_at)}`} />)
          ) : (
            <EmptyInline text="No team members assigned." />
          )}
        </Section>
      ) : null}
    </View>
  );
}

function TrafficSection(props: { dashboard: SiteDashboard; interval: TrafficInterval; onIntervalChange: (interval: TrafficInterval) => void }) {
  return (
    <View style={styles.sectionStack}>
      <SegmentedControl items={trafficIntervals} selectedKey={props.interval} onChange={props.onIntervalChange} />
      <StatGrid
        items={[
          { label: "Page views", value: formatCount(props.dashboard.stats.visit_count), detail: `${props.dashboard.stats.interval} window` },
          { label: "Visitors", value: formatCount(props.dashboard.stats.unique_visitor_count), detail: "Unique visitors" },
          { label: "Returning", value: formatPercent(props.dashboard.engagement.returning_visitor_rate), detail: `${formatCount(props.dashboard.engagement.returning_visitor_count)} visitors` },
          { label: "Pages/visitor", value: props.dashboard.engagement.average_pages_per_visitor.toFixed(1), detail: `${formatCount(props.dashboard.engagement.tracked_visitor_count)} tracked` },
        ]}
      />
      <Section title="Trend">
        <TrendRows trend={props.dashboard.trend.trend} />
      </Section>
      <Section title="Top pages">
        <Bars items={props.dashboard.stats.top_pages} labelFor={(item) => item.path || "/"} valueFor={(item) => item.visit_count} />
      </Section>
      <Section title="Attribution">
        <Bars items={props.dashboard.attribution.sources} labelFor={(item) => item.value || "Direct"} valueFor={(item) => item.visit_count} />
      </Section>
      <Section title="Devices">
        <Bars items={props.dashboard.devices.device_types} labelFor={(item) => sentenceCase(item.device_type)} valueFor={(item) => item.visit_count} />
      </Section>
      <Section title="Locations">
        <Bars items={props.dashboard.locations.locations} labelFor={(item) => item.label} valueFor={(item) => item.visit_count} />
      </Section>
      <Section title="Recent visits">
        {props.dashboard.stats.recent_visits.length ? (
          props.dashboard.stats.recent_visits.map((visit, index) => <VisitItem key={`${visit.visitor_id}-${visit.occurred_at}-${index}`} visit={visit} />)
        ) : (
          <EmptyInline text="No recent visits in this interval." />
        )}
      </Section>
    </View>
  );
}

function FeedbackSection(props: { messages: FeedbackMessage[] }) {
  return (
    <Section title="Feedback">
      {props.messages.length ? (
        props.messages.map((message) => (
          <ListItem
            key={message.id}
            title={compactText(message.message || message.sentiment, "No message")}
            meta={`${compactText(message.contact)} · ${formatDateTime(message.created_at)}`}
            detail={message.source_kind === "mobile" ? `${compactText(message.app_platform)} · ${compactText(message.screen_name || message.screen_path)}` : "Web widget"}
            right={sentenceCase(message.delivery)}
          />
        ))
      ) : (
        <EmptyInline text="No feedback yet." />
      )}
    </Section>
  );
}

function SubscribersSection(props: { subscribers: Subscriber[] }) {
  return (
    <Section title="Subscribers">
      {props.subscribers.length ? (
        props.subscribers.map((subscriber) => (
          <ListItem
            key={subscriber.id}
            title={subscriber.email}
            meta={`${compactText(subscriber.name, "No name")} · ${formatDateTime(subscriber.created_at)}`}
            right={sentenceCase(subscriber.status)}
          />
        ))
      ) : (
        <EmptyInline text="No subscribers yet." />
      )}
    </Section>
  );
}

function SentrySection(props: { issues: SentryIssue[]; site: Site }) {
  return (
    <View style={styles.sectionStack}>
      <StatGrid
        items={[
          { label: "Issues", value: formatCount(props.issues.length), detail: props.site.sentry_token_configured ? "Ingest ready" : "No token" },
          { label: "Open", value: formatCount(props.issues.filter((issue) => issue.status === "unresolved").length), detail: "Unresolved" },
        ]}
      />
      <Section title="Issues">
        {props.issues.length ? (
          props.issues.map((issue) => (
            <ListItem
              key={issue.id}
              title={issue.title}
              meta={`${sentenceCase(issue.level)} · ${compactText(issue.environment)} · ${compactText(issue.release)}`}
              detail={`Last seen ${formatDateTime(issue.last_seen_at)}`}
              right={formatCount(issue.occurrence_count)}
            />
          ))
        ) : (
          <EmptyInline text="No LA Sentry issues." />
        )}
      </Section>
    </View>
  );
}

function ReportsSection(props: { dashboard: PortfolioDashboard | null; siteDashboard: SiteDashboard; site: Site }) {
  const schedule = props.siteDashboard.trafficReportSchedule;
  return (
    <View style={styles.sectionStack}>
      <Section title="Selected-site report">
        {schedule ? (
          <>
            <InfoRow label="Enabled" value={schedule.enabled ? "Yes" : "No"} />
            <InfoRow label="Frequency" value={sentenceCase(schedule.frequency)} />
            <InfoRow label="Recipients" value={schedule.recipient_mode === "selected" ? schedule.recipient_emails.join(", ") : sentenceCase(schedule.recipient_mode)} />
            <InfoRow label="Next email" value={formatDateTime(schedule.next_send_at)} />
          </>
        ) : (
          <EmptyInline text={props.site.access_role === "admin" ? "Schedule not configured." : "Schedule is visible to site admins."} />
        )}
      </Section>
      <Section title="All-sites traffic">
        {props.dashboard ? (
          <>
            <StatGrid
              items={[
                { label: "Sites", value: formatCount(props.dashboard.defaultReport.site_count), detail: "Included" },
                { label: "Visits", value: formatCount(props.dashboard.defaultReport.visit_count), detail: "Last 30 days" },
                { label: "Visitors", value: formatCount(props.dashboard.defaultReport.unique_visitor_count), detail: "Unique" },
              ]}
            />
            {props.dashboard.defaultReport.sites.map((site) => (
              <ListItem
                key={site.site_id}
                title={site.site_name}
                meta={`${formatCount(site.visit_count)} visits`}
                right={`${formatCount(site.unique_visitor_count)} unique`}
              />
            ))}
          </>
        ) : (
          <EmptyInline text="All-sites report data unavailable." />
        )}
      </Section>
    </View>
  );
}

function TrendRows(props: { trend: VisitTrendPoint[] }) {
  if (!props.trend.length) {
    return <EmptyInline text="No trend data yet." />;
  }
  const recentTrend = props.trend.slice(-10);
  const maxValue = Math.max(1, ...recentTrend.map((point) => point.page_views));
  return (
    <>
      {recentTrend.map((point) => (
        <DataBar key={point.date} label={point.date} value={point.page_views} maxValue={maxValue} detail={`${formatCount(point.unique_visitors)} unique`} />
      ))}
    </>
  );
}

function Bars<T>(props: { items: T[]; labelFor: (item: T) => string; valueFor: (item: T) => number }) {
  if (!props.items.length) {
    return <EmptyInline text="No data yet." />;
  }
  const maxValue = Math.max(1, ...props.items.map(props.valueFor));
  return (
    <>
      {props.items.map((item, index) => (
        <DataBar key={`${props.labelFor(item)}-${index}`} label={props.labelFor(item)} value={props.valueFor(item)} maxValue={maxValue} />
      ))}
    </>
  );
}

function VisitItem(props: { visit: VisitLogEntry }) {
  return (
    <ListItem
      title={props.visit.path || props.visit.url || "Visit"}
      meta={`${compactText(props.visit.browser)} · ${compactText(props.visit.country)} · ${formatDateTime(props.visit.occurred_at)}`}
      detail={props.visit.referrer ? `Referrer: ${props.visit.referrer}` : compactText(props.visit.visitor_id, "Anonymous visitor")}
    />
  );
}

function StatGrid(props: { items: Array<{ label: string; value: string; detail?: string }> }) {
  return (
    <View style={styles.statGrid}>
      {props.items.map((item) => (
        <View key={item.label} style={styles.statTile}>
          <Text style={styles.statLabel}>{item.label}</Text>
          <Text style={styles.statValue}>{item.value}</Text>
          {item.detail ? <Text style={styles.statDetail}>{item.detail}</Text> : null}
        </View>
      ))}
    </View>
  );
}

function Section(props: { title: string; children: React.ReactNode }) {
  return (
    <View style={styles.section}>
      <Text style={styles.sectionTitle}>{props.title}</Text>
      {props.children}
    </View>
  );
}

function InfoRow(props: { label: string; value: string }) {
  return (
    <View style={styles.infoRow}>
      <Text style={styles.infoLabel}>{props.label}</Text>
      <Text style={styles.infoValue}>{props.value}</Text>
    </View>
  );
}

function ListItem(props: { title: string; meta?: string; detail?: string; right?: string }) {
  return (
    <View style={styles.listItem}>
      <View style={styles.listItemMain}>
        <Text style={styles.listItemTitle} numberOfLines={2}>
          {props.title}
        </Text>
        {props.meta ? <Text style={styles.listItemMeta}>{props.meta}</Text> : null}
        {props.detail ? <Text style={styles.listItemDetail}>{props.detail}</Text> : null}
      </View>
      {props.right ? <Text style={styles.listItemRight}>{props.right}</Text> : null}
    </View>
  );
}

function DataBar(props: { label: string; value: number; maxValue: number; detail?: string }) {
  const widthPercentage = `${Math.max(5, Math.round((props.value / props.maxValue) * 100))}%` as DimensionValue;
  return (
    <View style={styles.dataBarRow}>
      <View style={styles.dataBarHeader}>
        <Text style={styles.dataBarLabel} numberOfLines={1}>
          {compactText(props.label)}
        </Text>
        <Text style={styles.dataBarValue}>{formatCount(props.value)}</Text>
      </View>
      <View style={styles.dataBarTrack}>
        <View style={[styles.dataBarFill, { width: widthPercentage }]} />
      </View>
      {props.detail ? <Text style={styles.dataBarDetail}>{props.detail}</Text> : null}
    </View>
  );
}

function SegmentedControl<T extends string>(props: { items: Array<{ key: T; label: string }>; selectedKey: T; onChange: (key: T) => void }) {
  return (
    <View style={styles.segmentedControl}>
      {props.items.map((item) => {
        const selected = item.key === props.selectedKey;
        return (
          <Pressable key={item.key} style={[styles.segmentButton, selected && styles.segmentButtonActive]} onPress={() => props.onChange(item.key)}>
            <Text style={[styles.segmentButtonText, selected && styles.segmentButtonTextActive]}>{item.label}</Text>
          </Pressable>
        );
      })}
    </View>
  );
}

function PrimaryButton(props: { label: string; disabled?: boolean; onPress: () => void }) {
  return (
    <Pressable
      disabled={props.disabled}
      style={({ pressed }) => [styles.primaryButton, props.disabled && styles.disabledButton, pressed && styles.pressedButton]}
      onPress={props.onPress}
    >
      <Text style={styles.primaryButtonText}>{props.label}</Text>
    </Pressable>
  );
}

function StatusBanner(props: { message: string }) {
  return (
    <View style={styles.statusBanner}>
      <Text style={styles.statusBannerText}>{props.message}</Text>
    </View>
  );
}

function EmptyState(props: { title: string; detail: string }) {
  return (
    <View style={styles.emptyState}>
      <Text style={styles.emptyStateTitle}>{props.title}</Text>
      <Text style={styles.emptyStateDetail}>{props.detail}</Text>
    </View>
  );
}

function EmptyInline(props: { text: string }) {
  return <Text style={styles.emptyInline}>{props.text}</Text>;
}

function InlineLoading() {
  return (
    <View style={styles.inlineLoading}>
      <ActivityIndicator color={colors.teal} />
      <Text style={styles.inlineLoadingText}>Refreshing data</Text>
    </View>
  );
}

function resolveSelectedSiteId(sites: Site[], preferredSiteId?: string | null, persistedSiteId?: string | null): string | null {
  const candidates = [preferredSiteId, persistedSiteId].filter(Boolean);
  for (const candidate of candidates) {
    if (sites.some((site) => site.id === candidate)) {
      return candidate || null;
    }
  }
  return sites[0]?.id || null;
}

function errorToMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "Request failed";
}

const colors = {
  background: "#f5f7f6",
  ink: "#17211f",
  muted: "#65726e",
  border: "#d8e0dd",
  surface: "#ffffff",
  surfaceAlt: "#eef3f1",
  teal: "#157f78",
  tealDark: "#0e5e59",
  amber: "#b7791f",
  red: "#b42318",
  green: "#287d3c",
};

const styles = StyleSheet.create({
  safeArea: {
    flex: 1,
    backgroundColor: colors.background,
  },
  loadingScreen: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.background,
  },
  loadingText: {
    marginTop: 12,
    color: colors.muted,
    fontSize: 15,
  },
  scrollContent: {
    padding: 16,
    paddingBottom: 40,
    gap: 14,
  },
  signedOutScreen: {
    flex: 1,
    justifyContent: "space-between",
    padding: 22,
    backgroundColor: colors.background,
  },
  brandRow: {
    flexDirection: "row",
    alignItems: "center",
    gap: 14,
    marginTop: 20,
  },
  brandRowCompact: {
    flexDirection: "row",
    alignItems: "center",
    gap: 10,
    flex: 1,
  },
  brandIcon: {
    width: 56,
    height: 56,
    borderRadius: 14,
  },
  headerIcon: {
    width: 38,
    height: 38,
    borderRadius: 10,
  },
  brandName: {
    color: colors.ink,
    fontSize: 30,
    fontWeight: "800",
  },
  brandSubtle: {
    color: colors.muted,
    fontSize: 14,
    marginTop: 2,
  },
  signInPanel: {
    gap: 14,
    marginBottom: 24,
  },
  signInTitle: {
    color: colors.ink,
    fontSize: 28,
    fontWeight: "800",
  },
  signInCopy: {
    color: colors.muted,
    fontSize: 16,
    lineHeight: 22,
  },
  header: {
    flexDirection: "row",
    alignItems: "center",
    gap: 12,
  },
  headerTitle: {
    color: colors.ink,
    fontSize: 20,
    fontWeight: "800",
  },
  headerSubtitle: {
    color: colors.muted,
    fontSize: 12,
    marginTop: 1,
  },
  primaryButton: {
    alignItems: "center",
    justifyContent: "center",
    borderRadius: 8,
    minHeight: 50,
    paddingHorizontal: 16,
    backgroundColor: colors.teal,
  },
  primaryButtonText: {
    color: colors.surface,
    fontSize: 16,
    fontWeight: "800",
  },
  secondaryButton: {
    alignItems: "center",
    justifyContent: "center",
    borderRadius: 8,
    minHeight: 38,
    paddingHorizontal: 12,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface,
  },
  secondaryButtonText: {
    color: colors.ink,
    fontSize: 13,
    fontWeight: "700",
  },
  disabledButton: {
    opacity: 0.6,
  },
  pressedButton: {
    opacity: 0.82,
  },
  statusBanner: {
    borderWidth: 1,
    borderColor: "#f2b8b5",
    borderRadius: 8,
    padding: 12,
    backgroundColor: "#fff2f1",
  },
  statusBannerText: {
    color: colors.red,
    fontSize: 13,
    lineHeight: 18,
  },
  sitesSection: {
    gap: 8,
  },
  sectionLabel: {
    color: colors.muted,
    fontSize: 12,
    fontWeight: "800",
    textTransform: "uppercase",
  },
  sitesRail: {
    gap: 10,
    paddingRight: 6,
  },
  siteChip: {
    width: 168,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.border,
    padding: 12,
    backgroundColor: colors.surface,
  },
  siteChipSelected: {
    borderColor: colors.teal,
    backgroundColor: "#e6f4f1",
  },
  siteChipTitle: {
    color: colors.ink,
    fontSize: 14,
    fontWeight: "800",
  },
  siteChipTitleSelected: {
    color: colors.tealDark,
  },
  siteChipMeta: {
    color: colors.muted,
    fontSize: 12,
    marginTop: 5,
  },
  siteChipMetaSelected: {
    color: colors.tealDark,
  },
  siteHeader: {
    flexDirection: "row",
    alignItems: "flex-start",
    justifyContent: "space-between",
    gap: 12,
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    padding: 14,
    backgroundColor: colors.surface,
  },
  siteName: {
    color: colors.ink,
    fontSize: 22,
    fontWeight: "800",
  },
  siteOrigin: {
    color: colors.muted,
    fontSize: 13,
    marginTop: 4,
  },
  rolePill: {
    overflow: "hidden",
    borderRadius: 999,
    paddingHorizontal: 10,
    paddingVertical: 5,
    color: colors.tealDark,
    backgroundColor: "#d9f0ec",
    fontSize: 12,
    fontWeight: "800",
  },
  tabBar: {
    gap: 8,
    paddingVertical: 2,
  },
  tabButton: {
    minHeight: 38,
    justifyContent: "center",
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.border,
    paddingHorizontal: 13,
    backgroundColor: colors.surface,
  },
  tabButtonActive: {
    borderColor: colors.ink,
    backgroundColor: colors.ink,
  },
  tabButtonText: {
    color: colors.muted,
    fontSize: 13,
    fontWeight: "800",
  },
  tabButtonTextActive: {
    color: colors.surface,
  },
  sectionStack: {
    gap: 14,
  },
  section: {
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    padding: 14,
    backgroundColor: colors.surface,
    gap: 10,
  },
  sectionTitle: {
    color: colors.ink,
    fontSize: 17,
    fontWeight: "800",
  },
  statGrid: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 10,
  },
  statTile: {
    flexGrow: 1,
    flexBasis: "47%",
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    padding: 12,
    backgroundColor: colors.surface,
  },
  statLabel: {
    color: colors.muted,
    fontSize: 12,
    fontWeight: "800",
    textTransform: "uppercase",
  },
  statValue: {
    color: colors.ink,
    fontSize: 26,
    fontWeight: "800",
    marginTop: 8,
  },
  statDetail: {
    color: colors.muted,
    fontSize: 12,
    marginTop: 4,
  },
  infoRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    gap: 14,
    paddingVertical: 4,
  },
  infoLabel: {
    color: colors.muted,
    fontSize: 13,
    fontWeight: "700",
    flex: 0.4,
  },
  infoValue: {
    color: colors.ink,
    fontSize: 13,
    flex: 0.6,
    textAlign: "right",
  },
  listItem: {
    flexDirection: "row",
    alignItems: "flex-start",
    gap: 10,
    borderTopWidth: 1,
    borderTopColor: colors.surfaceAlt,
    paddingTop: 10,
  },
  listItemMain: {
    flex: 1,
    gap: 4,
  },
  listItemTitle: {
    color: colors.ink,
    fontSize: 14,
    fontWeight: "800",
  },
  listItemMeta: {
    color: colors.muted,
    fontSize: 12,
  },
  listItemDetail: {
    color: colors.muted,
    fontSize: 12,
    lineHeight: 17,
  },
  listItemRight: {
    color: colors.tealDark,
    fontSize: 12,
    fontWeight: "800",
  },
  segmentedControl: {
    flexDirection: "row",
    borderRadius: 8,
    borderWidth: 1,
    borderColor: colors.border,
    overflow: "hidden",
    backgroundColor: colors.surface,
  },
  segmentButton: {
    flex: 1,
    minHeight: 38,
    justifyContent: "center",
    alignItems: "center",
  },
  segmentButtonActive: {
    backgroundColor: colors.teal,
  },
  segmentButtonText: {
    color: colors.muted,
    fontSize: 13,
    fontWeight: "800",
  },
  segmentButtonTextActive: {
    color: colors.surface,
  },
  dataBarRow: {
    gap: 6,
  },
  dataBarHeader: {
    flexDirection: "row",
    justifyContent: "space-between",
    gap: 12,
  },
  dataBarLabel: {
    color: colors.ink,
    fontSize: 13,
    fontWeight: "700",
    flex: 1,
  },
  dataBarValue: {
    color: colors.muted,
    fontSize: 12,
    fontWeight: "800",
  },
  dataBarTrack: {
    height: 8,
    borderRadius: 999,
    backgroundColor: colors.surfaceAlt,
    overflow: "hidden",
  },
  dataBarFill: {
    height: 8,
    borderRadius: 999,
    backgroundColor: colors.teal,
  },
  dataBarDetail: {
    color: colors.muted,
    fontSize: 12,
  },
  emptyInline: {
    color: colors.muted,
    fontSize: 13,
    lineHeight: 18,
  },
  emptyState: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    padding: 26,
    backgroundColor: colors.surface,
  },
  emptyStateTitle: {
    color: colors.ink,
    fontSize: 18,
    fontWeight: "800",
  },
  emptyStateDetail: {
    color: colors.muted,
    fontSize: 13,
    lineHeight: 18,
    textAlign: "center",
    marginTop: 8,
  },
  inlineLoading: {
    flexDirection: "row",
    alignItems: "center",
    gap: 8,
    borderRadius: 8,
    padding: 10,
    backgroundColor: "#e6f4f1",
  },
  inlineLoadingText: {
    color: colors.tealDark,
    fontSize: 13,
    fontWeight: "700",
  },
});
