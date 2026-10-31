import { Image as FaviconImage } from "expo-image";
import { StatusBar } from "expo-status-bar";
import * as SecureStore from "expo-secure-store";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  ActivityIndicator,
  AccessibilityInfo,
  BackHandler,
  findNodeHandle,
  Modal,
  type DimensionValue,
  Image,
  Keyboard,
  Pressable,
  RefreshControl,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  useWindowDimensions,
  View,
} from "react-native";
import { SafeAreaProvider, SafeAreaView } from "react-native-safe-area-context";
import { LoopAwareApiClient, LoopAwareApiError } from "./src/api";
import { useAccountMenu } from "./src/account-menu";
import { useAccountPhoto, useResourceImage } from "./src/avatar";
import { ThemeProvider, useTheme, palettes, type ThemeColors } from "./src/theme";
import { AuthController } from "./src/auth";
import { loadRuntimeConfig } from "./src/config";
import { compactText, formatCount, formatDate, formatDateTime, formatPercent, sentenceCase } from "./src/format";
import type {
  Account,
  DashboardTab,
  FeedbackMessage,
  LocationPoint,
  PortfolioDashboard,
  SentryIssue,
  Site,
  SiteDashboard,
  Subscriber,
  TrafficInterval,
  VisitTrendPoint,
} from "./src/types";
import { locationWorldLandPath } from "./src/location-world-land";

const selectedSiteStorageKey = "loopaware.mobile.selectedSiteId";
const dashboardTabs: Array<{ key: DashboardTab; label: string }> = [
  { key: "overview", label: "Overview" },
  { key: "feedback", label: "Feedback" },
  { key: "traffic", label: "Traffic" },
  { key: "subscribers", label: "Subscribers" },
  { key: "health", label: "Health" },
  { key: "sentry", label: "LA Sentry" },
  { key: "reports", label: "Reports" },
];
const trafficIntervals: Array<{ key: TrafficInterval; label: string }> = [
  { key: "all", label: "All" },
  { key: "30days", label: "30d" },
  { key: "7days", label: "7d" },
  { key: "1day", label: "1d" },
];
const mobileFeedbackSourceKind: FeedbackMessage["source_kind"] = "mobile_app";
const siteSelectorLabelSize = 12;
const siteFilterTextSize = 14;
const siteFilterLineHeight = 20;
const siteFilterVerticalPadding = 8;
const siteFilterTouchSize = 44;
type HeadingFocus = { requested: boolean; onFocused: () => void };
type TrafficMeasure = "unique_visitors" | "requests";
type StatItem = { label: string; value: string; detail?: string };
type OverviewStatItem = StatItem & { destination: DashboardTab; actionLabel: string };

export default function App() {
  return <ThemeProvider><AppContent /></ThemeProvider>;
}

function AppContent() {
  const { styles, colors } = useAppStyles();
  const theme = useTheme();
  const [screen, setScreen] = useState<"dashboard" | "settings">("dashboard");
  useEffect(() => {
    if (screen !== "settings") return;
    const subscription = BackHandler.addEventListener("hardwareBackPress", () => { setScreen("dashboard"); return true; });
    return () => subscription.remove();
  }, [screen]);
  const runtimeConfig = useMemo(() => loadRuntimeConfig(), []);
  const apiClient = useMemo(() => new LoopAwareApiClient(runtimeConfig), [runtimeConfig]);
  const authController = useMemo(() => new AuthController(apiClient), [apiClient]);
  const [status, setStatus] = useState<"checking" | "signedOut" | "signedIn">("checking");
  const [account, setAccount] = useState<Account | null>(null);
  const [sites, setSites] = useState<Site[] | null>(null);
  const [selectedSiteId, setSelectedSiteId] = useState<string | null>(null);
  const [siteDashboard, setSiteDashboard] = useState<SiteDashboard | null>(null);
  const [portfolioDashboard, setPortfolioDashboard] = useState<PortfolioDashboard | null>(null);
  const [activeTab, setActiveTab] = useState<DashboardTab>("overview");
  const [overviewDestination, setOverviewDestination] = useState<DashboardTab | null>(null);
  const dashboardScroll = useRef<ScrollView>(null);
  const tabBarTop = useRef(0);
  const openOverviewTab = (tab: DashboardTab) => {
    Keyboard.dismiss();
    setActiveTab(tab);
    setOverviewDestination(tab);
    dashboardScroll.current?.scrollTo({ y: tabBarTop.current, animated: false });
  };
  const [trafficInterval, setTrafficInterval] = useState<TrafficInterval>("30days");
  const [busy, setBusy] = useState(false);
  const [refreshing, setRefreshing] = useState(false);
  const [errorMessage, setErrorMessage] = useState<string | null>(null);
  const dashboardLoadVersion = useRef(0);
  const authenticatedLoadVersion = useRef(0);
  const signInAttemptVersion = useRef(0);

  const loadSiteData = useCallback(
    async (site: Site, interval: TrafficInterval) => {
      const requestVersion = dashboardLoadVersion.current + 1;
      dashboardLoadVersion.current = requestVersion;
      setBusy(true);
      setErrorMessage(null);
      try {
        const dashboard = await apiClient.siteDashboard(site, interval);
        if (requestVersion !== dashboardLoadVersion.current) {
          return;
        }
        setSiteDashboard(dashboard);
      } catch (error) {
        if (requestVersion !== dashboardLoadVersion.current) {
          return;
        }
        setErrorMessage(errorToMessage(error));
      } finally {
        if (requestVersion === dashboardLoadVersion.current) {
          setBusy(false);
        }
      }
    },
    [apiClient],
  );

  const loadAuthenticatedState = useCallback(
    async (mode: "restore" | "current", preferredSiteId: string | null, interval: TrafficInterval) => {
      const requestVersion = dashboardLoadVersion.current + 1;
      dashboardLoadVersion.current = requestVersion;
      const accountRequestVersion = authenticatedLoadVersion.current + 1;
      authenticatedLoadVersion.current = accountRequestVersion;
      setBusy(true);
      setErrorMessage(null);
      try {
        const currentAccount = mode === "restore" ? await authController.restore() : await apiClient.me();
        if (accountRequestVersion !== authenticatedLoadVersion.current) {
          return;
        }
        if (!currentAccount) {
          setStatus("signedOut");
          setAccount(null);
          setSites(null);
          setSelectedSiteId(null);
          setSiteDashboard(null);
          setPortfolioDashboard(null);
          return;
        }
        setAccount(currentAccount);
        setStatus("signedIn");
        setPortfolioDashboard(null);

        async function loadSites() {
          try {
            const [sitesResponse, persistedSiteId] = await Promise.all([
              apiClient.sites(),
              SecureStore.getItemAsync(selectedSiteStorageKey).catch(() => null),
            ]);
            if (requestVersion !== dashboardLoadVersion.current) {
              return;
            }
            const nextSelectedSiteId = resolveSelectedSiteId(sitesResponse.sites, preferredSiteId, persistedSiteId);
            const selectedSite = sitesResponse.sites.find((site) => site.id === nextSelectedSiteId) || null;
            setSites(sitesResponse.sites);
            setSelectedSiteId(nextSelectedSiteId);
            setSiteDashboard(null);
            const nextSiteDashboard = selectedSite ? await apiClient.siteDashboard(selectedSite, interval) : null;
            if (requestVersion === dashboardLoadVersion.current) {
              setSiteDashboard(nextSiteDashboard);
            }
          } catch (error) {
            if (requestVersion === dashboardLoadVersion.current) {
              setErrorMessage(errorToMessage(error));
            }
          }
        }

        async function loadPortfolio() {
          try {
            const nextPortfolioDashboard = await apiClient.portfolioDashboard();
            if (accountRequestVersion === authenticatedLoadVersion.current) {
              setPortfolioDashboard(nextPortfolioDashboard);
            }
          } catch (error) {
            if (accountRequestVersion === authenticatedLoadVersion.current) {
              setErrorMessage(errorToMessage(error));
            }
          }
        }

        await Promise.all([loadSites(), loadPortfolio()]);
      } catch (error) {
        if (accountRequestVersion === authenticatedLoadVersion.current) {
          setStatus((currentStatus) => (currentStatus === "checking" ? "signedOut" : currentStatus));
          setErrorMessage(errorToMessage(error));
        }
      } finally {
        if (requestVersion === dashboardLoadVersion.current) {
          setBusy(false);
        }
        if (accountRequestVersion === authenticatedLoadVersion.current) {
          setRefreshing(false);
        }
      }
    },
    [apiClient, authController],
  );

  useEffect(() => {
    void loadAuthenticatedState("restore", null, "30days");
  }, [loadAuthenticatedState]);

  const selectedSite = useMemo(() => sites?.find((site) => site.id === selectedSiteId) || null, [selectedSiteId, sites]);
  const selectedSiteDashboard = useMemo(() => {
    if (!siteDashboard || siteDashboard.stats.site_id !== selectedSiteId || siteDashboard.stats.interval !== trafficInterval) {
      return null;
    }
    return siteDashboard;
  }, [selectedSiteId, siteDashboard, trafficInterval]);

  const handleSignIn = useCallback(async () => {
    const attemptVersion = ++signInAttemptVersion.current;
    setBusy(true);
    setErrorMessage(null);
    try {
      const currentAccount = await authController.signIn();
      if (attemptVersion !== signInAttemptVersion.current) {
        return;
      }
      setAccount(currentAccount);
      setStatus("signedIn");
      await loadAuthenticatedState("current", null, trafficInterval);
    } catch (error) {
      if (attemptVersion === signInAttemptVersion.current) {
        setStatus("signedOut");
        setErrorMessage(errorToMessage(error));
      }
    } finally {
      if (attemptVersion === signInAttemptVersion.current) {
        setBusy(false);
      }
    }
  }, [authController, loadAuthenticatedState, trafficInterval]);

  const handleSignOut = useCallback(async () => {
    setBusy(true);
    setErrorMessage(null);
    dashboardLoadVersion.current += 1;
    authenticatedLoadVersion.current += 1;
    signInAttemptVersion.current += 1;
    try {
      await authController.signOut();
      setAccount(null);
      setSites(null);
      setSelectedSiteId(null);
      setSiteDashboard(null);
      setPortfolioDashboard(null);
      setStatus("signedOut");
      setScreen("dashboard");
    } catch (error) {
      setErrorMessage(`Could not sign out. ${errorToMessage(error)}`);
    } finally {
      setBusy(false);
      setRefreshing(false);
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

  if (status === "checking" || theme.loading) {
    return (
      <SafeAreaProvider>
        <SafeAreaView style={styles.loadingScreen}>
          <StatusBar style={theme.appearance === "dark" ? "light" : "dark"} />
          <ActivityIndicator size="large" color={colors.teal} />
          <Text style={styles.loadingText}>Loading LoopAware</Text>
        </SafeAreaView>
      </SafeAreaProvider>
    );
  }

  return (
    <SafeAreaProvider>
      <SafeAreaView style={styles.safeArea}>
        <StatusBar style={theme.appearance === "dark" ? "light" : "dark"} />
        {status === "signedOut" ? (
          <SignedOutScreen busy={busy} errorMessage={errorMessage ?? theme.error} onSignIn={handleSignIn} runtimeHost={runtimeConfig.apiBaseUrl} />
        ) : screen === "settings" ? (
          <SettingsScreen account={account!} sites={sites} busy={busy} errorMessage={errorMessage}
            onBack={() => setScreen("dashboard")} onSignOut={handleSignOut} />
        ) : (
          <ScrollView
            ref={dashboardScroll}
            contentContainerStyle={styles.scrollContent}
            keyboardShouldPersistTaps="handled"
            keyboardDismissMode="on-drag"
            automaticallyAdjustKeyboardInsets
            refreshControl={<RefreshControl refreshing={refreshing} onRefresh={handleRefresh} tintColor={colors.teal} />}
          >
            <Header account={account!} apiClient={apiClient} onSettings={() => {
              setScreen("settings");
              void loadAuthenticatedState("current", selectedSiteId, trafficInterval);
            }} />
            {errorMessage ? <StatusBanner message={errorMessage} /> : null}
            {theme.error ? <StatusBanner message={theme.error} /> : null}
            {sites ? <SitesRail key={account!.email} apiClient={apiClient} sites={sites} selectedSiteId={selectedSiteId} onSelectSite={handleSelectSite} /> : null}
            {selectedSite ? (
              <>
                <SelectedSiteHeader site={selectedSite} />
                {selectedSiteDashboard ? (
                  <>
                    <View onLayout={event => { tabBarTop.current = event.nativeEvent.layout.y; }}>
                      <TabBar activeTab={activeTab} onChange={tab => { setOverviewDestination(null); setActiveTab(tab); }} />
                    </View>
                    {busy ? <InlineLoading /> : null}
                    <DashboardTabContent
                      activeTab={activeTab}
                      interval={trafficInterval}
                      onIntervalChange={handleSelectInterval}
                      portfolioDashboard={portfolioDashboard}
                      site={selectedSite}
                      siteDashboard={selectedSiteDashboard}
                      onNavigate={openOverviewTab}
                      headingFocus={{ requested: overviewDestination === activeTab, onFocused: () => setOverviewDestination(null) }}
                    />
                  </>
                ) : busy ? (
                  <InlineLoading />
                ) : (
                  <EmptyState title="Site unavailable" detail="This site's dashboard data could not be loaded." />
                )}
              </>
            ) : sites === null ? (
              busy ? <InlineLoading /> : <EmptyState title="Sites unavailable" detail="Your sites could not be loaded." />
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
  const { styles, colors } = useAppStyles();
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

function focusNative(element: View | Text | null) {
  if (!element) return;
  const tag = findNodeHandle(element);
  if (tag !== null) AccessibilityInfo.setAccessibilityFocus(tag);
}

function Header(props: { account: Account; apiClient: LoopAwareApiClient; onSettings: () => void }) {
  const { styles, colors } = useAppStyles();
  const avatar = useRef<View>(null);
  const menuItem = useRef<View>(null);
  const menu = useAccountMenu(props.onSettings, () => focusNative(avatar.current));
  return (
    <View style={styles.header}>
      <View style={styles.brandRowCompact}>
        <Image source={require("./assets/icon.png")} style={styles.headerIcon} />
        <Text style={styles.headerTitle}>LoopAware</Text>
      </View>
      <Pressable ref={avatar} accessibilityRole="button" accessibilityLabel="Account menu"
        accessibilityHint="Open account options" accessibilityState={{ expanded: menu.visible }}
        style={styles.avatar} onPress={menu.open}>
        <AccountAvatar key={`${props.account.email}:${props.account.avatar?.url ?? ""}`} account={props.account} apiClient={props.apiClient} />
      </Pressable>
      <Modal visible={menu.visible} transparent animationType="none" onRequestClose={() => menu.close("avatar")}
        onShow={() => focusNative(menuItem.current)} onDismiss={menu.onDismiss}>
        <SafeAreaView style={styles.menuOverlay} accessibilityViewIsModal onAccessibilityEscape={() => menu.close("avatar")}>
          <Pressable accessibilityRole="button" accessibilityLabel="Close account menu" style={styles.menuBackdrop} onPress={() => menu.close("avatar")} />
          <View style={styles.accountMenu}>
            <Pressable ref={menuItem} accessibilityRole="button" style={styles.menuItem} onPress={() => menu.close("settings")}>
              <Text style={styles.secondaryButtonText}>Settings</Text>
            </Pressable>
          </View>
        </SafeAreaView>
      </Modal>
    </View>
  );
}

function SettingsScreen(props: { account: Account; sites: Site[] | null; busy: boolean; errorMessage: string | null; onBack: () => void; onSignOut: () => void }) {
  const { styles, colors } = useAppStyles();
  const theme = useTheme();
  const heading = useRef<Text>(null);
  const focused = useRef(false);
  return <ScrollView contentContainerStyle={styles.scrollContent} accessibilityLabel="Account settings">
    <Pressable accessibilityRole="button" onPress={props.onBack} style={styles.backButton}>
      <Text style={styles.secondaryButtonText}>Back to dashboard</Text>
    </Pressable>
    <Text ref={heading} accessibilityRole="header" style={styles.settingsTitle} onLayout={() => {
      if (!focused.current) { focused.current = true; focusNative(heading.current); }
    }}>Settings</Text>
    {props.errorMessage ? <StatusBanner message={props.errorMessage} /> : null}
    <Section title="Account">
      <Text style={styles.labelText}>Email</Text><Text selectable style={styles.valueText}>{props.account.email}</Text>
      <Text style={styles.labelText}>Account role</Text><Text style={styles.valueText}>{sentenceCase(props.account.role)}</Text>
    </Section>
    <Section title="Site access">
      {props.sites === null ? <Text style={styles.helperText}>{props.busy ? "Loading site roles" : "Site roles could not be loaded."}</Text>
        : props.sites.length ? props.sites.map(site => <View key={site.id} style={styles.settingsRole}>
          <Text style={styles.valueText}>{site.name}</Text><Text style={styles.helperText}>Site role: {sentenceCase(site.access_role)}</Text>
        </View>) : <Text style={styles.helperText}>No site access</Text>}
    </Section>
    <Section title="Appearance">
      <Text style={styles.helperText}>Choose a theme. System follows your device appearance.</Text>
      <View accessibilityRole="radiogroup" style={styles.themeChoices}>
        {(["light", "dark", "system"] as const).map(choice => <Pressable key={choice} accessibilityRole="radio"
          accessibilityLabel={sentenceCase(choice)} accessibilityState={{ selected: theme.preference === choice, checked: theme.preference === choice }}
          onPress={() => theme.select(choice)} style={[styles.themeChoice, theme.preference === choice && styles.themeChoiceSelected]}>
          <Text style={[styles.secondaryButtonText, theme.preference === choice && styles.themeChoiceTextSelected]}>{sentenceCase(choice)}</Text>
        </Pressable>)}
      </View>
      <Text accessibilityLiveRegion="polite" style={styles.helperText}>{theme.saving ? "Saving theme…" : theme.error ? theme.error : "Changes save automatically"}</Text>
    </Section>
    {props.busy ? <InlineLoading /> : null}
    <Pressable accessibilityRole="button" disabled={props.busy} style={[styles.secondaryButton, props.busy && styles.disabledButton]} onPress={props.onSignOut}>
      <Text style={styles.secondaryButtonText}>Sign out</Text>
    </Pressable>
  </ScrollView>;
}

function AccountAvatar(props: { account: Account; apiClient: LoopAwareApiClient }) {
  const photo = useAccountPhoto(props.account, props.apiClient);
  return <AvatarImage key={photo.status === "loaded" ? photo.generation : "unavailable"}
    email={props.account.email} uri={photo.status === "loaded" ? photo.uri : undefined} />;
}

function AvatarImage(props: { email: string; uri?: string }) {
  const { styles } = useAppStyles();
  const [imageState, setImageState] = useState<"loading" | "ready" | "unavailable">(props.uri ? "loading" : "unavailable");
  return (
    <View style={styles.avatarContent} pointerEvents="none" accessible={false}>
      {imageState !== "ready" ? <Text style={styles.avatarInitial}>{Array.from(props.email)[0].toUpperCase()}</Text> : null}
      {props.uri && imageState !== "unavailable" ? <Image source={{ uri: props.uri }}
        style={[styles.avatarImage, imageState !== "ready" && styles.avatarImageLoading]}
        accessible={false} onLoad={() => setImageState("ready")} onError={() => setImageState("unavailable")} /> : null}
    </View>
  );
}

function SitesRail(props: { apiClient: LoopAwareApiClient; sites: Site[]; selectedSiteId: string | null; onSelectSite: (site: Site) => void }) {
  const { styles, colors } = useAppStyles();
  const { fontScale } = useWindowDimensions();
  const [filterOpen, setFilterOpen] = useState(false);
  const [query, setQuery] = useState("");
  const searchInput = useRef<TextInput>(null);
  const searchButton = useRef<View>(null);
  const searchText = query.trim().toLowerCase();
  const matchingSites = props.sites.filter(site => site.name.toLowerCase().includes(searchText) || site.allowed_origin.toLowerCase().includes(searchText));
  const closeFilter = () => {
    setFilterOpen(false);
    setQuery("");
    Keyboard.dismiss();
    focusNative(searchButton.current);
  };
  if (!props.sites.length) {
    return null;
  }
  return (
    <View style={styles.sitesSection} onAccessibilityEscape={filterOpen ? closeFilter : undefined}>
      <View style={{ ...styles.sitesHeading, minHeight: Math.max(siteFilterTouchSize, Math.ceil(siteFilterLineHeight * fontScale) + siteFilterVerticalPadding * 2) + 2 }}>
        <Text style={styles.sectionLabel}>Sites</Text>
        {filterOpen ? <View style={styles.siteFilterField}>
          <TextInput ref={searchInput} accessibilityRole="search" accessibilityLabel="Filter sites" autoFocus
            value={query} onChangeText={setQuery} placeholder="Site name or domain" placeholderTextColor={colors.muted}
            autoCapitalize="none" autoCorrect={false} autoComplete="off" returnKeyType="search"
            onSubmitEditing={() => Keyboard.dismiss()} style={styles.siteFilterInput} />
          {query.length ? <Pressable accessibilityRole="button" accessibilityLabel="Clear site filter" style={styles.siteFilterButton}
            onPress={() => { setQuery(""); searchInput.current?.focus(); }}>
            <Text style={styles.siteFilterClear}>×</Text>
          </Pressable> : null}
        </View> : null}
        <Pressable ref={searchButton} accessibilityRole="button" accessibilityLabel="Find sites"
          accessibilityHint={filterOpen ? "Close the filter and show all sites" : "Filter sites by name or domain"}
          accessibilityState={{ expanded: filterOpen }} style={styles.siteFilterButton}
          onPress={() => filterOpen ? closeFilter() : setFilterOpen(true)}>
          <View style={styles.searchIcon} accessibilityElementsHidden importantForAccessibility="no-hide-descendants">
            <View style={styles.searchIconRing} />
            <View style={styles.searchIconHandle} />
          </View>
        </Pressable>
      </View>
      {!matchingSites.length ? <EmptyInline text="No sites match your filter." /> : <ScrollView horizontal
        keyboardShouldPersistTaps="handled" showsHorizontalScrollIndicator={false} contentContainerStyle={styles.sitesRail}>
        {matchingSites.map((site) => {
          const selected = site.id === props.selectedSiteId;
          return (
            <Pressable key={site.id} accessibilityRole="button" accessibilityLabel={site.name} accessibilityState={{ selected }}
              style={[styles.siteChip, selected && styles.siteChipSelected]} onPress={() => { Keyboard.dismiss(); props.onSelectSite(site); }}>
              <SiteBadgeTitle key={`${site.id}:${site.favicon_url}`} site={site} selected={selected} apiClient={props.apiClient} />
              <Text style={[styles.siteChipMeta, selected && styles.siteChipMetaSelected]}>{sentenceCase(site.access_role)}</Text>
            </Pressable>
          );
        })}
      </ScrollView>}
    </View>
  );
}

function SiteBadgeTitle(props: { site: Site; selected: boolean; apiClient: LoopAwareApiClient }) {
  const load = useCallback((signal: AbortSignal) => props.apiClient.siteFavicon(props.site, signal), [props.apiClient, props.site.id, props.site.favicon_url]);
  const image = useResourceImage(load);
  return <SiteBadgeImage key={image.status === "loaded" ? image.generation : "unavailable"}
    site={props.site} selected={props.selected} uri={image.status === "loaded" ? image.uri : undefined} />;
}

function SiteBadgeImage(props: { site: Site; selected: boolean; uri?: string }) {
  const { styles } = useAppStyles();
  const [failed, setFailed] = useState(false);
  const showIcon = Boolean(props.uri) && !failed;
  return <View>
    <Text style={[styles.siteChipTitle, props.selected && styles.siteChipTitleSelected, showIcon && styles.siteChipIconTitle]} numberOfLines={1}>{props.site.name}</Text>
    {showIcon ? <FaviconImage source={{ uri: props.uri }} style={styles.siteChipIcon} accessible={false}
      contentFit="contain" cachePolicy="none" transition={0} onError={() => setFailed(true)} /> : null}
  </View>;
}

function SelectedSiteHeader(props: { site: Site }) {
  const { styles, colors } = useAppStyles();
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
  const { styles, colors } = useAppStyles();
  const scroll = useRef<ScrollView>(null);
  const viewport = useRef(0);
  const offset = useRef(0);
  const frames = useRef(new Map<DashboardTab, { x: number; width: number }>());
  const revealSelected = useCallback(() => {
    const frame = frames.current.get(props.activeTab);
    if (!frame || viewport.current === 0) return;
    const nextOffset = frame.x < offset.current ? frame.x
      : frame.x + frame.width > offset.current + viewport.current ? frame.x + frame.width - viewport.current : offset.current;
    if (nextOffset !== offset.current) {
      offset.current = nextOffset;
      scroll.current?.scrollTo({ x: nextOffset, animated: false });
    }
  }, [props.activeTab]);
  useEffect(revealSelected, [revealSelected]);
  return (
    <ScrollView ref={scroll} horizontal showsHorizontalScrollIndicator={false} contentContainerStyle={styles.tabBar}
      onLayout={event => { viewport.current = event.nativeEvent.layout.width; revealSelected(); }}
      onContentSizeChange={revealSelected} onScroll={event => { offset.current = event.nativeEvent.contentOffset.x; }} scrollEventThrottle={16}>
      {dashboardTabs.map((tab) => {
        const active = tab.key === props.activeTab;
        return (
          <Pressable key={tab.key} accessibilityRole="tab" accessibilityLabel={tab.label} accessibilityState={{ selected: active }}
            onLayout={event => { frames.current.set(tab.key, event.nativeEvent.layout); if (active) revealSelected(); }}
            style={[styles.tabButton, active && styles.tabButtonActive]} onPress={() => props.onChange(tab.key)}>
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
  onNavigate: (tab: DashboardTab) => void;
  headingFocus: HeadingFocus;
}) {
  const { styles, colors } = useAppStyles();
  switch (props.activeTab) {
    case "feedback":
      return <FeedbackSection messages={props.siteDashboard.messages} headingFocus={props.headingFocus} />;
    case "traffic":
      return <TrafficSection dashboard={props.siteDashboard} interval={props.interval} onIntervalChange={props.onIntervalChange} headingFocus={props.headingFocus} />;
    case "subscribers":
      return <SubscribersSection subscribers={props.siteDashboard.subscribers} headingFocus={props.headingFocus} />;
    case "health":
      return <HealthSection monitor={props.siteDashboard.healthMonitor} headingFocus={props.headingFocus} />;
    case "sentry":
      return <SentrySection issues={props.siteDashboard.sentryIssues} site={props.site} headingFocus={props.headingFocus} />;
    case "reports":
      return <ReportsSection dashboard={props.portfolioDashboard} siteDashboard={props.siteDashboard} site={props.site} />;
    case "overview":
    default:
      return <OverviewSection dashboard={props.siteDashboard} site={props.site} onNavigate={props.onNavigate} />;
  }
}

function OverviewSection(props: { dashboard: SiteDashboard; site: Site; onNavigate: (tab: DashboardTab) => void }) {
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.sectionStack}>
      <StatGrid
        onNavigate={props.onNavigate}
        items={[
          { destination: "feedback", actionLabel: "View feedback", label: "Feedback", value: formatCount(props.site.feedback_count), detail: `${formatCount(props.dashboard.messages.length)} loaded` },
          { destination: "subscribers", actionLabel: "View subscribers", label: "Subscribers", value: formatCount(props.site.subscriber_count), detail: `${formatCount(props.dashboard.subscribers.length)} loaded` },
          { destination: "traffic", actionLabel: "View traffic", label: props.site.traffic_profile === "detailed" ? "Unique visitors" : "Requests", value: formatCount(props.site.traffic_profile === "detailed" ? props.dashboard.stats.unique_visitor_count : props.dashboard.stats.visit_count), detail: props.site.traffic_profile === "detailed" ? `${formatCount(props.dashboard.stats.visit_count)} page views` : "Daily counts" },
          { destination: "health", actionLabel: "View health", label: "Health", value: sentenceCase(props.dashboard.healthMonitor?.status || "unknown"), detail: props.dashboard.healthMonitor?.enabled ? "Checks enabled" : "Checks off" },
          { destination: "sentry", actionLabel: "View Sentry", label: "Sentry", value: formatCount(props.dashboard.sentryIssues.length), detail: props.site.sentry_token_configured ? "Token configured" : "Token missing" },
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

function TrafficSection(props: { dashboard: SiteDashboard; interval: TrafficInterval; onIntervalChange: (interval: TrafficInterval) => void; headingFocus: HeadingFocus }) {
  const { styles, colors } = useAppStyles();
  const { engagement, attribution, devices, locations } = props.dashboard;
  if (engagement === null || attribution === null || devices === null || locations === null) {
    return (
      <View style={styles.sectionStack}>
        <DashboardHeading title="Traffic" focus={props.headingFocus} />
        <SegmentedControl items={trafficIntervals} selectedKey={props.interval} onChange={props.onIntervalChange} />
        <StatGrid items={[{ label: "Accepted requests", value: formatCount(props.dashboard.stats.visit_count), detail: "Daily totals in UTC" }]} />
        <Section title="Daily counts"><TrendRows trend={props.dashboard.trend.trend} measure="requests" /></Section>
        <EmptyInline text="Unique visitors, individual visits, pages, devices, and locations are unavailable for daily counts." />
      </View>
    );
  }
  return (
    <View style={styles.sectionStack}>
      <DashboardHeading title="Traffic" focus={props.headingFocus} />
      <SegmentedControl items={trafficIntervals} selectedKey={props.interval} onChange={props.onIntervalChange} />
      <StatGrid
        items={[
          { label: "Unique visitors", value: formatCount(props.dashboard.stats.unique_visitor_count), detail: "Distinct visitors in this window" },
          { label: "Page views", value: formatCount(props.dashboard.stats.visit_count), detail: "Includes repeat views" },
          { label: "Returning", value: formatPercent(engagement.returning_visitor_rate), detail: `${formatCount(engagement.returning_visitor_count)} returning visitors` },
          { label: "Pages/visitor", value: engagement.average_pages_per_visitor.toFixed(1), detail: `${formatCount(engagement.tracked_visitor_count)} tracked` },
        ]}
      />
      <Section title="Trend">
        <TrendRows trend={props.dashboard.trend.trend} measure="unique_visitors" />
      </Section>
      <Section title="Top pages">
        <Bars items={props.dashboard.stats.top_pages} labelFor={(item) => item.path || "/"} valueFor={(item) => item.unique_visitor_count} detailFor={item => `${formatCount(item.visit_count)} page views`} />
      </Section>
      <Section title="Attribution">
        <Bars items={attribution.sources} labelFor={(item) => item.value || "Direct"} valueFor={(item) => item.unique_visitor_count} detailFor={item => `${formatCount(item.visit_count)} page views`} />
      </Section>
      <Section title="Devices">
        <Bars items={devices.device_types} labelFor={(item) => sentenceCase(item.device_type)} valueFor={(item) => item.unique_visitor_count} detailFor={item => `${formatCount(item.visit_count)} page views`} />
      </Section>
      <Section title="Locations">
        <LocationMap locations={locations.locations} />
      </Section>
    </View>
  );
}

function FeedbackSection(props: { messages: FeedbackMessage[]; headingFocus: HeadingFocus }) {
  return (
    <Section title="Feedback" headingFocus={props.headingFocus}>
      {props.messages.length ? (
        props.messages.map((message) => (
          <ListItem
            key={message.id}
            title={compactText(message.message || message.sentiment, "No message")}
            meta={`${compactText(message.contact)} · ${formatDateTime(message.created_at)}`}
            detail={
              message.source_kind === mobileFeedbackSourceKind
                ? `${compactText(message.app_platform)} · ${compactText(message.screen_name || message.screen_path)}`
                : "Web widget"
            }
            right={sentenceCase(message.delivery)}
          />
        ))
      ) : (
        <EmptyInline text="No feedback yet." />
      )}
    </Section>
  );
}

function SubscribersSection(props: { subscribers: Subscriber[]; headingFocus: HeadingFocus }) {
  return (
    <Section title="Subscribers" headingFocus={props.headingFocus}>
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

function HealthSection(props: { monitor: SiteDashboard["healthMonitor"]; headingFocus: HeadingFocus }) {
  const { styles, colors } = useAppStyles();
  const monitor = props.monitor;
  if (!monitor) {
    return <View style={styles.sectionStack}><DashboardHeading title="Health" focus={props.headingFocus} /><EmptyInline text="Health monitor is not available." /></View>;
  }
  return (
    <View style={styles.sectionStack}>
      <DashboardHeading title="Health" focus={props.headingFocus} />
      <StatGrid
        items={[
          { label: "Status", value: sentenceCase(monitor.status), detail: monitor.enabled ? "Checks enabled" : "Checks off" },
          { label: "Failures", value: formatCount(monitor.consecutive_failures), detail: `Threshold ${formatCount(monitor.failure_threshold)}` },
          { label: "Response", value: monitor.last_duration_ms ? `${formatCount(monitor.last_duration_ms)} ms` : "No check", detail: monitor.last_status_code ? `HTTP ${monitor.last_status_code}` : "No status" },
        ]}
      />
      <Section title="Monitor">
        <InfoRow label="Target" value={compactText(monitor.target_url)} />
        <InfoRow label="Interval" value={`${formatCount(monitor.interval_seconds)} seconds`} />
        <InfoRow label="Timeout" value={`${formatCount(monitor.timeout_seconds)} seconds`} />
        <InfoRow label="Recipients" value={healthRecipientLabel(monitor)} />
      </Section>
      <Section title="Checks">
        <InfoRow label="Next check" value={formatDateTime(monitor.next_check_at)} />
        <InfoRow label="Last checked" value={formatDateTime(monitor.last_checked_at)} />
        <InfoRow label="Last up" value={formatDateTime(monitor.last_success_at)} />
        <InfoRow label="Last failure" value={formatDateTime(monitor.last_failure_at)} />
        <InfoRow label="Last error" value={compactText(monitor.last_error_message || monitor.last_error_code, "None")} />
      </Section>
    </View>
  );
}

function healthRecipientLabel(monitor: SiteDashboard["healthMonitor"]): string {
  if (!monitor) {
    return "";
  }
  if (monitor.recipient_mode === "selected") {
    return monitor.recipient_emails.join(", ");
  }
  return sentenceCase(monitor.recipient_mode);
}

function SentrySection(props: { issues: SentryIssue[]; site: Site; headingFocus: HeadingFocus }) {
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.sectionStack}>
      <DashboardHeading title="LA Sentry" focus={props.headingFocus} />
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
  const { styles, colors } = useAppStyles();
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

function TrendRows(props: { trend: VisitTrendPoint[]; measure: TrafficMeasure }) {
  if (!props.trend.length) {
    return <EmptyInline text="No trend data yet." />;
  }
  const valueFor = (point: VisitTrendPoint) => props.measure === "unique_visitors" ? point.unique_visitors! : point.page_views;
  const maxValue = Math.max(1, ...props.trend.map(valueFor));
  return (
    <>
      {props.trend.map((point) => (
        <DataBar key={point.date} label={point.date} value={valueFor(point)} maxValue={maxValue} measure={props.measure} detail={props.measure === "unique_visitors" ? `${formatCount(point.page_views)} page views` : undefined} />
      ))}
    </>
  );
}

function Bars<T>(props: { items: T[]; labelFor: (item: T) => string; valueFor: (item: T) => number; detailFor: (item: T) => string }) {
  if (!props.items.length) {
    return <EmptyInline text="No data yet." />;
  }
  const maxValue = Math.max(1, ...props.items.map(props.valueFor));
  return (
    <>
      {props.items.map((item, index) => (
        <DataBar key={`${props.labelFor(item)}-${index}`} label={props.labelFor(item)} value={props.valueFor(item)} maxValue={maxValue} measure="unique_visitors" detail={props.detailFor(item)} />
      ))}
    </>
  );
}

const locationMapWidth = 720;
const locationMapHeight = 300;
const locationMapTouchSize = 44;

function mappedLocation(location: LocationPoint): boolean {
  return location.label !== "UTC" && location.source !== "unknown" && location.signal !== "missing_signal"
    && (location.country !== "" || location.city !== "" || location.region !== "" || location.latitude !== 0 || location.longitude !== 0);
}

function locationPosition(location: LocationPoint): { left: `${number}%`; top: `${number}%` } {
  const degreeSize = locationMapHeight / 180;
  const longitude = Math.max(-180, Math.min(180, location.longitude));
  const latitude = Math.max(-80, Math.min(84, location.latitude));
  const horizontal = ((locationMapWidth / 2 + longitude * degreeSize) / locationMapWidth) * 100;
  const vertical = ((locationMapHeight / 2 - latitude * degreeSize) / locationMapHeight) * 100;
  return {
    left: `${Math.max(7, Math.min(93, horizontal))}%`,
    top: `${Math.max(16, Math.min(84, vertical))}%`,
  };
}

function locationCountSummary(location: LocationPoint): string {
  const count = location.unique_visitor_count;
  return `${location.label} ${formatCount(count)} unique visitor${count === 1 ? "" : "s"} ${formatCount(location.visit_count)} page views`;
}

function LocationMap(props: { locations: LocationPoint[] }) {
  const { styles, colors } = useAppStyles();
  const [selectedLabel, setSelectedLabel] = useState<string | null>(null);
  const [frameWidth, setFrameWidth] = useState<number | null>(null);
  const plotted = props.locations.filter(mappedLocation).slice(0, 10);
  const unlocated = props.locations.filter(location => !mappedLocation(location));
  const selected = plotted.find(location => location.label === selectedLabel) ?? plotted[0];
  const maximum = Math.max(1, ...plotted.map(location => location.visit_count));
  const mapSource = useMemo(() => {
    const mapSvg = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 720 300">`
      + `<rect width="720" height="300" fill="${colors.surfaceAlt}"/>`
      + `<path d="${locationWorldLandPath}" fill="${colors.muted}" fill-opacity="0.35" stroke="${colors.border}" stroke-width="0.8"/>`
      + `<path d="M160 0V300 M260 0V300 M360 0V300 M460 0V300 M560 0V300 M0 75H720 M0 150H720 M0 225H720" fill="none" stroke="${colors.border}" stroke-width="0.8" stroke-dasharray="3 8"/>`
      + `</svg>`;
    return { uri: `data:image/svg+xml,${encodeURIComponent(mapSvg)}` };
  }, [colors.surfaceAlt, colors.muted, colors.border]);

  if (!props.locations.length) return <EmptyInline text="No location data yet." />;

  return (
    <View style={styles.locationMapStack}>
      <View testID="location-map-frame" style={styles.locationMapFrame}
        onLayout={event => setFrameWidth(event.nativeEvent.layout.width)}>
        <View testID="location-map-canvas" style={[styles.locationMapCanvas, frameWidth === null ? null : { width: Math.min(locationMapWidth, frameWidth) }]}>
          <FaviconImage source={mapSource} style={styles.locationMapImage} contentFit="fill" accessibilityLabel="World map" />
          {plotted.map(location => {
            const size = 18 + Math.sqrt(location.visit_count / maximum) * 18;
            const active = location.label === selected?.label;
            return (
              <Pressable key={`${location.label}-${location.source}`} accessibilityRole="button"
                accessibilityLabel={`Show ${location.label} on map`}
                accessibilityValue={{ text: locationCountSummary(location) }}
                accessibilityState={{ selected: active }}
                onPress={() => setSelectedLabel(location.label)}
                style={[styles.locationMapMarker, locationPosition(location), active && styles.locationMapMarkerActive]}>
                <View style={[styles.locationMapBubble, { width: size, height: size, borderRadius: size / 2 }]}>
                  <Text maxFontSizeMultiplier={1.1} style={styles.locationMapBubbleText}>{formatCount(location.visit_count)}</Text>
                </View>
              </Pressable>
            );
          })}
        </View>
      </View>
      {plotted.length > 1 ? (
        <View style={styles.locationMapChoices}>
          {plotted.map(location => {
            const active = location.label === selected?.label;
            return (
              <Pressable key={`${location.label}-${location.source}`} accessibilityRole="button"
                accessibilityLabel={`Select ${location.label} location`}
                accessibilityValue={{ text: locationCountSummary(location) }}
                accessibilityState={{ selected: active }}
                onPress={() => setSelectedLabel(location.label)}
                style={[styles.locationMapChoice, active && styles.locationMapChoiceActive]}>
                <Text style={[styles.locationMapChoiceText, active && styles.locationMapChoiceTextActive]}>{location.label}</Text>
              </Pressable>
            );
          })}
        </View>
      ) : null}
      {selected ? (
        <View style={styles.locationMapDetail}>
          <Text style={styles.locationMapDetailText}>{locationCountSummary(selected)}</Text>
          <Text style={styles.locationMapDetailSource}>{selected.source.replace(/_/g, " ")} · {Math.round(selected.confidence * 100)}% location confidence</Text>
        </View>
      ) : null}
      {unlocated.map(location => (
        <View key={`${location.label}-${location.source}`} style={styles.locationMapUnknown}>
          <Text style={styles.locationMapDetailSource}>{`Location unavailable ${locationCountSummary(location)}`}</Text>
        </View>
      ))}
    </View>
  );
}

function StatGrid(props: { items: StatItem[] } | { items: OverviewStatItem[]; onNavigate: (tab: DashboardTab) => void }) {
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.statGrid}>
      {"onNavigate" in props ? props.items.map(item => (
        <Pressable key={item.label} accessibilityRole="button" accessibilityLabel={item.actionLabel}
          accessibilityValue={{ text: [item.label, item.value, item.detail].filter(Boolean).join(", ") }}
          style={({ pressed }) => [styles.statTile, pressed && styles.statTilePressed]} onPress={() => props.onNavigate(item.destination)}>
          <StatContents item={item} />
        </Pressable>
      )) : props.items.map(item => <View key={item.label} style={styles.statTile}><StatContents item={item} /></View>)}
    </View>
  );
}

function StatContents(props: { item: StatItem }) {
  const { styles } = useAppStyles();
  return <><Text style={styles.statLabel}>{props.item.label}</Text><Text style={styles.statValue}>{props.item.value}</Text>
    {props.item.detail ? <Text style={styles.statDetail}>{props.item.detail}</Text> : null}</>;
}

function DashboardHeading(props: { title: string; focus?: HeadingFocus }) {
  const { styles } = useAppStyles();
  const heading = useRef<Text>(null);
  return <Text ref={heading} accessibilityRole="header" style={styles.sectionTitle} onLayout={() => {
    if (props.focus?.requested) { focusNative(heading.current); props.focus.onFocused(); }
  }}>{props.title}</Text>;
}

function Section(props: { title: string; children: React.ReactNode; headingFocus?: HeadingFocus }) {
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.section}>
      <DashboardHeading title={props.title} focus={props.headingFocus} />
      {props.children}
    </View>
  );
}

function InfoRow(props: { label: string; value: string }) {
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.infoRow}>
      <Text style={styles.infoLabel}>{props.label}</Text>
      <Text style={styles.infoValue}>{props.value}</Text>
    </View>
  );
}

function ListItem(props: { title: string; meta?: string; detail?: string; right?: string }) {
  const { styles, colors } = useAppStyles();
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

function DataBar(props: { label: string; value: number; maxValue: number; measure: TrafficMeasure; detail?: string }) {
  const { styles, colors } = useAppStyles();
  const widthPercentage = `${Math.round((props.value / props.maxValue) * 100)}%` as DimensionValue;
  const unit = props.measure === "unique_visitors" ? "unique visitors" : "requests";
  return (
    <View style={styles.dataBarRow} accessible accessibilityLabel={props.label}
      accessibilityValue={{ text: `${formatCount(props.value)} ${unit}${props.detail ? `, ${props.detail}` : ""}` }}>
      <View style={styles.dataBarHeader}>
        <Text style={styles.dataBarLabel} numberOfLines={1}>
          {compactText(props.label)}
        </Text>
        <Text style={styles.dataBarValue}>{formatCount(props.value)} {props.measure === "unique_visitors" ? "unique" : "requests"}</Text>
      </View>
      <View style={styles.dataBarTrack}>
        <View style={[styles.dataBarFill, { width: widthPercentage }]} />
      </View>
      {props.detail ? <Text style={styles.dataBarDetail}>{props.detail}</Text> : null}
    </View>
  );
}

function SegmentedControl<T extends string>(props: { items: Array<{ key: T; label: string }>; selectedKey: T; onChange: (key: T) => void }) {
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.segmentedControl}>
      {props.items.map((item) => {
        const selected = item.key === props.selectedKey;
        return (
          <Pressable key={item.key} accessibilityRole="button" accessibilityState={{ selected }} style={[styles.segmentButton, selected && styles.segmentButtonActive]} onPress={() => props.onChange(item.key)}>
            <Text style={[styles.segmentButtonText, selected && styles.segmentButtonTextActive]}>{item.label}</Text>
          </Pressable>
        );
      })}
    </View>
  );
}

function PrimaryButton(props: { label: string; disabled?: boolean; onPress: () => void }) {
  const { styles, colors } = useAppStyles();
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
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.statusBanner}>
      <Text style={styles.statusBannerText}>{props.message}</Text>
    </View>
  );
}

function EmptyState(props: { title: string; detail: string }) {
  const { styles, colors } = useAppStyles();
  return (
    <View style={styles.emptyState}>
      <Text style={styles.emptyStateTitle}>{props.title}</Text>
      <Text style={styles.emptyStateDetail}>{props.detail}</Text>
    </View>
  );
}

function EmptyInline(props: { text: string }) {
  const { styles, colors } = useAppStyles();
  return <Text style={styles.emptyInline}>{props.text}</Text>;
}

function InlineLoading() {
  const { styles, colors } = useAppStyles();
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
  if (error instanceof LoopAwareApiError && error.code === "native_google_platform_not_configured") {
    return "Native Google sign-in is not configured for LoopAware in TAuth yet.";
  }
  if (error instanceof Error) {
    return error.message;
  }
  return "Request failed";
}

function createStyles(colors: ThemeColors) { return StyleSheet.create({
  labelText: { color: colors.muted, fontSize: 13, fontWeight: "700" },
  valueText: { color: colors.ink, fontSize: 16 },
  helperText: { color: colors.muted, fontSize: 14, lineHeight: 20 },
  avatar: { width: 44, height: 44, borderRadius: 22, alignItems: "center", justifyContent: "center", backgroundColor: colors.roleSurface, borderWidth: 1, borderColor: colors.teal },
  avatarContent: { width: 42, height: 42, alignItems: "center", justifyContent: "center" },
  avatarImage: { position: "absolute", width: 42, height: 42, borderRadius: 21 },
  avatarImageLoading: { opacity: 0 },
  avatarInitial: { fontSize: 20, fontWeight: "800", color: colors.tealDark },
  menuOverlay: { flex: 1, padding: 16, alignItems: "flex-end" },
  menuBackdrop: { position: "absolute", top: 0, bottom: 0, left: 0, right: 0, backgroundColor: "rgba(0,0,0,0.25)" },
  accountMenu: { marginTop: 58, minWidth: 180, borderRadius: 12, borderWidth: 1, borderColor: colors.border, backgroundColor: colors.surface },
  menuItem: { padding: 18, minHeight: 48, justifyContent: "center" },
  backButton: { minHeight: 44, justifyContent: "center", alignSelf: "flex-start" },
  settingsTitle: { color: colors.ink, fontSize: 28, fontWeight: "800" },
  settingsRole: { paddingVertical: 8, gap: 4 },
  themeChoices: { gap: 10 },
  themeChoice: { minHeight: 48, borderRadius: 8, borderWidth: 1, borderColor: colors.border, padding: 14, backgroundColor: colors.surface },
  themeChoiceSelected: { backgroundColor: colors.accentSurface, borderColor: colors.teal },
  themeChoiceTextSelected: { color: colors.tealDark },

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
    color: colors.onAccent,
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
    borderColor: colors.errorBorder,
    borderRadius: 8,
    padding: 12,
    backgroundColor: colors.errorSurface,
  },
  statusBannerText: {
    color: colors.red,
    fontSize: 13,
    lineHeight: 18,
  },
  sitesSection: {
    gap: 8,
  },
  sitesHeading: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    gap: 8,
  },
  siteFilterButton: {
    minWidth: siteFilterTouchSize,
    minHeight: siteFilterTouchSize,
    alignItems: "center",
    justifyContent: "center",
  },
  searchIcon: {
    width: siteSelectorLabelSize,
    height: siteSelectorLabelSize,
  },
  searchIconRing: {
    width: 9,
    height: 9,
    borderWidth: 1.5,
    borderRadius: 5,
    borderColor: colors.muted,
  },
  searchIconHandle: {
    position: "absolute",
    width: 6,
    height: 1.5,
    top: 9,
    left: 7,
    backgroundColor: colors.muted,
    transform: [{ rotate: "45deg" }],
  },
  siteFilterField: {
    flex: 1,
    minWidth: 0,
    flexDirection: "row",
    alignItems: "center",
    borderWidth: 1,
    borderColor: colors.border,
    borderRadius: 8,
    backgroundColor: colors.surface,
  },
  siteFilterInput: {
    flex: 1,
    minWidth: 0,
    minHeight: siteFilterTouchSize,
    paddingHorizontal: 12,
    paddingVertical: siteFilterVerticalPadding,
    fontSize: siteFilterTextSize,
    lineHeight: siteFilterLineHeight,
    color: colors.ink,
  },
  siteFilterClear: {
    fontSize: 22,
    color: colors.muted,
  },
  sectionLabel: {
    color: colors.muted,
    fontSize: siteSelectorLabelSize,
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
  siteChipIcon: { position: "absolute", left: 0, top: "50%", transform: [{ translateY: -8 }], width: 16, height: 16 },
  siteChipIconTitle: { paddingLeft: 22 },
  siteChipSelected: {
    borderColor: colors.teal,
    backgroundColor: colors.accentSurface,
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
    backgroundColor: colors.roleSurface,
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
  locationMapStack: {
    gap: 10,
  },
  locationMapFrame: {
    width: "100%",
    alignItems: "center",
  },
  locationMapCanvas: {
    width: "100%",
    maxWidth: locationMapWidth,
    aspectRatio: 2.4,
    borderRadius: 10,
    overflow: "hidden",
    backgroundColor: colors.surfaceAlt,
  },
  locationMapImage: {
    width: "100%",
    height: "100%",
  },
  locationMapMarker: {
    position: "absolute",
    width: locationMapTouchSize,
    height: locationMapTouchSize,
    alignItems: "center",
    justifyContent: "center",
    transform: [{ translateX: -locationMapTouchSize / 2 }, { translateY: -locationMapTouchSize / 2 }],
  },
  locationMapMarkerActive: {
    zIndex: 1,
  },
  locationMapBubble: {
    alignItems: "center",
    justifyContent: "center",
    backgroundColor: colors.teal,
    borderColor: colors.surface,
    borderWidth: 2,
  },
  locationMapBubbleText: {
    color: colors.onAccent,
    fontSize: 10,
    fontWeight: "800",
  },
  locationMapChoices: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: 8,
    justifyContent: "center",
  },
  locationMapChoice: {
    minHeight: locationMapTouchSize,
    maxWidth: "100%",
    justifyContent: "center",
    paddingHorizontal: 12,
    paddingVertical: 8,
    borderWidth: 1,
    borderRadius: 999,
    borderColor: colors.border,
    backgroundColor: colors.surfaceAlt,
  },
  locationMapChoiceActive: {
    borderColor: colors.teal,
    backgroundColor: colors.accentSurface,
  },
  locationMapChoiceText: {
    color: colors.muted,
    fontSize: 12,
    fontWeight: "700",
  },
  locationMapChoiceTextActive: {
    color: colors.tealDark,
  },
  locationMapDetail: {
    backgroundColor: colors.surfaceAlt,
    borderRadius: 8,
    padding: 10,
    gap: 4,
  },
  locationMapDetailText: {
    color: colors.ink,
    fontSize: 13,
    fontWeight: "700",
  },
  locationMapDetailSource: {
    color: colors.muted,
    fontSize: 12,
  },
  locationMapUnknown: {
    paddingHorizontal: 4,
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
  statTilePressed: { borderColor: colors.teal, opacity: 0.8 },
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
    backgroundColor: colors.accentSurface,
  },
  inlineLoadingText: {
    color: colors.tealDark,
    fontSize: 13,
    fontWeight: "700",
  },
}); }

const themeStyles = { light: createStyles(palettes.light), dark: createStyles(palettes.dark) };
function useAppStyles() {
  const theme = useTheme();
  return { styles: themeStyles[theme.appearance], colors: theme.colors };
}
