package httpapi_test

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/go-rod/rod/lib/proto"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/tyemirov/tauth/pkg/sessionvalidator"

	"github.com/MarkoPoloResearchLab/loopaware/internal/httpapi"
	"github.com/MarkoPoloResearchLab/loopaware/internal/model"
	"github.com/MarkoPoloResearchLab/loopaware/internal/storage"
	"github.com/MarkoPoloResearchLab/loopaware/internal/testutil"
	"github.com/MarkoPoloResearchLab/loopaware/pkg/favicon"
)

const (
	dashboardTestTauthSigningKey        = "test-tauth-signing-key"
	dashboardTestJWTIssuer              = "tauth"
	dashboardTestSessionCookieName      = "app_session"
	dashboardTestTauthTenantID          = "test-tenant"
	dashboardTestGoogleClientID         = "test-google-client-id"
	dashboardTestAdminEmail             = "admin@example.com"
	dashboardTestAdminDisplayName       = "Admin Example"
	dashboardTestSecondaryEmail         = "operator@example.com"
	dashboardTestSecondaryDisplayName   = "Operator Example"
	dashboardTestAvatarDataURI          = "data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw=="
	dashboardTestWidgetBaseURL          = "http://example.test"
	dashboardTestDashboardRoute         = "/app"
	landingGoogleNonceDelayMilliseconds = 5000
	dashboardPromptWaitTimeout          = 10 * time.Second
	dashboardPromptPollInterval         = 200 * time.Millisecond
	dashboardNotificationSelector       = "#session-timeout-notification"
	dashboardPromptVisibleScript        = `(function(){
		var element = document.querySelector('#session-timeout-notification');
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		var rect = element.getBoundingClientRect();
		return style.display !== 'none' && style.visibility !== 'hidden' && rect.width > 0 && rect.height > 0;
	}())`
	dashboardDismissButtonSelector              = "#session-timeout-dismiss-button"
	dashboardConfirmButtonSelector              = "#session-timeout-confirm-button"
	dashboardUserMenuSelector                   = `mpr-user[data-loopaware-user-menu="true"]`
	dashboardUserMenuTriggerSelector            = dashboardUserMenuSelector + ` [data-mpr-user="trigger"]`
	dashboardUserMenuOpenSelector               = dashboardUserMenuSelector + ` [data-mpr-user="menu"][aria-hidden="false"]`
	dashboardUserMenuAccountSettingsSelector    = dashboardUserMenuSelector + ` [data-mpr-user="menu-item"][data-mpr-user-action="account-settings"]`
	dashboardUserMenuLogoutButtonSelector       = dashboardUserMenuSelector + ` [data-mpr-user="logout"]`
	dashboardUserMenuAvatarSelector             = dashboardUserMenuSelector + ` [data-mpr-user="avatar-image"]`
	landingGoogleSigninWrapperSelector          = `mpr-header [data-mpr-google-wrapper="true"]`
	dashboardSettingsModalSelector              = "#settings-modal"
	dashboardWidgetBottomOffsetInputSelector    = "#widget-placement-bottom-offset"
	dashboardWidgetBottomOffsetIncreaseSelector = "#widget-bottom-offset-increase"
	dashboardWidgetBottomOffsetDecreaseSelector = "#widget-bottom-offset-decrease"
	dashboardSubscribeTestButtonSelector        = "#subscribe-test-button"
	subscribeTestInlineContainerID              = "subscribe-test-inline-preview"
	subscribeTestFormContainerSelector          = "#" + subscribeTestInlineContainerID
	subscribeTestTargetInputID                  = "subscribe-test-target"
	subscribeTestTargetInputSelector            = "#" + subscribeTestTargetInputID
	subscribeTestPreviewContainerSelector       = "[data-subscribe-test-preview=\"true\"]"
	subscribeTestLogSelector                    = "#subscribe-test-log"
	dashboardTrafficTestButtonSelector          = "#traffic-test-button"
	trafficTestURLInputSelector                 = "#traffic-test-url"
	trafficTestSendButtonSelector               = "#traffic-test-send-hit"
	trafficTestTotalSelector                    = "#traffic-test-visit-total"
	trafficTestUniqueSelector                   = "#traffic-test-visit-unique"
	widgetBottomOffsetStepPixels                = 10
	dashboardReadWidgetBottomOffsetScript       = `(function() {
		var input = document.getElementById('widget-placement-bottom-offset');
		if (!input) { return ''; }
		return String(input.value || '');
	}())`
	dashboardSettingsAutoLogoutToggleSelector  = "#settings-auto-logout-enabled"
	dashboardSettingsAutoLogoutPromptSelector  = "#settings-auto-logout-prompt-seconds"
	dashboardSettingsAutoLogoutLogoutSelector  = "#settings-auto-logout-logout-seconds"
	dashboardFeedbackWidgetSnippetSelector     = "#widget-snippet"
	dashboardSubscribeWidgetSnippetSelector    = "#subscribe-widget-snippet"
	dashboardTrafficWidgetSnippetSelector      = "#traffic-widget-snippet"
	dashboardFeedbackCopyButtonSelector        = "#copy-widget-snippet"
	dashboardSubscribeCopyButtonSelector       = "#copy-subscribe-widget-snippet"
	dashboardTrafficCopyButtonSelector         = "#copy-traffic-widget-snippet"
	dashboardFeedbackWidgetCardSelector        = `[data-widget-card="feedback"]`
	dashboardSubscribeWidgetCardSelector       = `[data-widget-card="subscribe"]`
	dashboardTrafficWidgetCardSelector         = `[data-widget-card="traffic"]`
	dashboardDashboardCardSelector             = "[data-dashboard-card]"
	dashboardFeedbackMessagesCardSelector      = `[data-dashboard-card="feedback"]`
	dashboardSubscribersCardSelector           = `[data-dashboard-card="subscribers"]`
	dashboardTrafficCardSelector               = `[data-dashboard-card="traffic"]`
	dashboardSectionTabFeedbackSelector        = "#dashboard-section-tab-feedback"
	dashboardSectionTabSubscriptionsSelector   = "#dashboard-section-tab-subscriptions"
	dashboardSectionTabTrafficSelector         = "#dashboard-section-tab-traffic"
	dashboardSubscribersTableBodySelector      = "#subscribers-table-body"
	dashboardAutoLogoutStorageBaseKey          = "loopaware_dashboard_auto_logout"
	dashboardLogoutFetchStorageKey             = "loopawareLogoutFetch"
	dashboardDisableGoogleAutoSelectStorageKey = "loopawareDisableGoogleAutoSelect"
	dashboardSettingsModalVisibleScript        = `(function() {
		var modal = document.getElementById('settings-modal');
		if (!modal) { return false; }
		return modal.classList.contains('show');
	}())`
	dashboardSettingsModalContentScript = `(function() {
		function parseRGB(value) {
			if (!value) { return null; }
			var match = value.match(/rgba?\((\d+),\s*(\d+),\s*(\d+)/);
			if (!match) { return null; }
			return { r: Number(match[1]), g: Number(match[2]), b: Number(match[3]) };
		}
		var content = document.getElementById('settings-modal-content');
		var body = document.querySelector('#settings-modal .modal-body');
		var modal = document.querySelector('#settings-modal .modal-content');
		if (!content || !body || !modal) {
			return { textLength: 0, contrast: 0 };
		}
		var bodyStyle = window.getComputedStyle(body);
		var modalStyle = window.getComputedStyle(modal);
		var textLength = (content.textContent || '').trim().length;
		var textColor = parseRGB(bodyStyle.color);
		var modalColor = parseRGB(modalStyle.backgroundColor);
		var delta = 0;
		if (textColor && modalColor) {
			delta = Math.abs(textColor.r - modalColor.r) + Math.abs(textColor.g - modalColor.g) + Math.abs(textColor.b - modalColor.b);
		}
		return { textLength: textLength, contrast: delta };
	}())`
	dashboardUserMenuStateScript = `(function() {
		var headerHost = document.querySelector('mpr-header');
		var loopawareMenuCount = document.querySelectorAll('mpr-user[data-loopaware-user-menu="true"]').length;
		var headerUserMenuCount = 0;
		var extraHeaderUserMenuCount = 0;
		var headerAvatarCount = 0;
		var visibleHeaderAvatarCount = 0;
		if (headerHost && typeof headerHost.querySelectorAll === 'function') {
			headerUserMenuCount = headerHost.querySelectorAll('mpr-user[data-mpr-header="user-menu"]').length;
			extraHeaderUserMenuCount = headerHost.querySelectorAll('mpr-user[data-mpr-header="user-menu"]:not([data-loopaware-user-menu="true"])').length;
			var headerAvatars = headerHost.querySelectorAll('[data-mpr-user="avatar-image"]');
			headerAvatarCount = headerAvatars ? headerAvatars.length : 0;
			for (var headerAvatarIndex = 0; headerAvatarIndex < headerAvatarCount; headerAvatarIndex += 1) {
				var headerAvatar = headerAvatars[headerAvatarIndex];
				if (!headerAvatar) {
					continue;
				}
				var avatarStyle = window.getComputedStyle(headerAvatar);
				var avatarRect = headerAvatar.getBoundingClientRect();
				var isVisible = avatarStyle.display !== 'none' && avatarStyle.visibility !== 'hidden' && avatarRect.width > 0 && avatarRect.height > 0;
				if (isVisible) {
					visibleHeaderAvatarCount += 1;
				}
			}
		}
		var userMenu = document.querySelector('mpr-user[data-loopaware-user-menu="true"]');
		if (!userMenu) {
			return { loopawareMenuCount: loopawareMenuCount, headerUserMenuCount: headerUserMenuCount, extraHeaderUserMenuCount: extraHeaderUserMenuCount, headerAvatarCount: headerAvatarCount, visibleHeaderAvatarCount: visibleHeaderAvatarCount, avatarCount: 0, displayName: '', avatarVisible: false, nameVisible: false };
		}
		var avatarNodes = userMenu.querySelectorAll('[data-mpr-user="avatar-image"]');
		var avatar = avatarNodes && avatarNodes.length ? avatarNodes[0] : null;
		var name = userMenu.querySelector('[data-mpr-user="name"]');
		var avatarVisible = false;
		if (avatar) {
			var avatarStyle = window.getComputedStyle(avatar);
			var avatarRect = avatar.getBoundingClientRect();
			avatarVisible = avatarStyle.display !== 'none' && avatarStyle.visibility !== 'hidden' && avatarRect.width > 0 && avatarRect.height > 0;
		}
		var nameVisible = false;
		if (name) {
			var nameStyle = window.getComputedStyle(name);
			var nameRect = name.getBoundingClientRect();
			nameVisible = nameStyle.display !== 'none' && nameStyle.visibility !== 'hidden' && nameRect.width > 0 && nameRect.height > 0;
		}
		return {
			loopawareMenuCount: loopawareMenuCount,
			headerUserMenuCount: headerUserMenuCount,
			extraHeaderUserMenuCount: extraHeaderUserMenuCount,
			headerAvatarCount: headerAvatarCount,
			visibleHeaderAvatarCount: visibleHeaderAvatarCount,
			avatarCount: avatarNodes ? avatarNodes.length : 0,
			displayName: name ? (name.textContent || '').trim() : '',
			avatarVisible: avatarVisible,
			nameVisible: nameVisible
		};
	}())`
	dashboardHeaderDefaultProfileStateScript = `(function() {
		var headerHost = document.querySelector('mpr-header');
		if (!headerHost) {
			return { hasUserMenu: false, hasLegacyProfileMenu: false, hasGoogleSignin: false, hasSettingsButton: false };
		}
		var hasSelector = function(selector) {
			if (headerHost.querySelector(selector)) {
				return true;
			}
			if (headerHost.shadowRoot && headerHost.shadowRoot.querySelector(selector)) {
				return true;
			}
			return false;
		};
		var userMenuCount = 0;
		if (headerHost.querySelectorAll) {
			userMenuCount = headerHost.querySelectorAll('mpr-user[data-loopaware-user-menu="true"]').length;
		}
		return {
			hasUserMenu: userMenuCount === 1 || hasSelector('mpr-user[data-loopaware-user-menu="true"]') || hasSelector('[data-mpr-header="user-menu"]'),
			hasLegacyProfileMenu: !!document.querySelector('[data-loopaware-profile-menu="true"]'),
			hasGoogleSignin: hasSelector('[data-mpr-header="google-signin"]'),
			hasSettingsButton: hasSelector('[data-mpr-header="settings-button"]')
		};
	}())`
	dashboardLogoutTestHookScript = `(function() {
		var storageKey = '` + dashboardLogoutFetchStorageKey + `';
		var originalFetch = window.fetch ? window.fetch.bind(window) : null;
		var sessionCookieName = '` + dashboardTestSessionCookieName + `';
		function clearSessionCookie() {
			if (typeof document === 'undefined') {
				return;
			}
			try {
				document.cookie = sessionCookieName + '=; Max-Age=0; path=/';
				document.cookie = sessionCookieName + '=; expires=Thu, 01 Jan 1970 00:00:00 GMT; path=/';
			} catch (error) {}
		}
		window.__loopawareLogoutFetchCalls = [];
		window.fetch = function(input, options) {
			var url = '';
			if (typeof input === 'string') {
				url = input;
			} else if (input && typeof input.url === 'string') {
				url = input.url;
			}
			if (url.indexOf('/auth/logout') !== -1) {
				window.__loopawareLogoutFetchCalls.push({ url: url, options: options || null });
				if (window.sessionStorage) {
					window.sessionStorage.setItem(storageKey, 'true');
				}
				clearSessionCookie();
				return Promise.resolve(new Response(null, { status: 204 }));
			}
			if (originalFetch) {
				return originalFetch(input, options);
			}
			return Promise.reject(new Error('fetch unavailable'));
		};
		window.__loopawareLogoutDelegate = function() {
			return Promise.reject(new Error('logout failed'));
		};
	}())`
	dashboardLogoutFetchClearScript = `(function() {
		var storageKey = '` + dashboardLogoutFetchStorageKey + `';
		if (window.sessionStorage) {
			window.sessionStorage.removeItem(storageKey);
		}
		return true;
	}())`
	dashboardLogoutFetchCalledScript = `(function() {
		var storageKey = '` + dashboardLogoutFetchStorageKey + `';
		if (window.sessionStorage) {
			var value = window.sessionStorage.getItem(storageKey);
			if (value === 'true') { return true; }
		}
		var calls = window.__loopawareLogoutFetchCalls || [];
		return calls.length > 0;
	}())`
	dashboardLogoutFormPresentScript = `(function() {
		return !!document.querySelector('[data-loopaware-logout-form="true"]');
	}())`
	dashboardLogoutFormFallbackScript = `(function() {
		var originalFetch = window.fetch ? window.fetch.bind(window) : null;
		window.fetch = function(input, options) {
			var url = '';
			if (typeof input === 'string') {
				url = input;
			} else if (input && typeof input.url === 'string') {
				url = input.url;
			}
			if (url.indexOf('/auth/logout') !== -1) {
				return Promise.reject(new Error('logout blocked'));
			}
			if (originalFetch) {
				return originalFetch(input, options);
			}
			return Promise.reject(new Error('fetch unavailable'));
		};
		window.__loopawareLogoutDelegate = function() {
			return Promise.reject(new Error('logout blocked'));
		};
	}())`
	dashboardLogoutForbiddenScript = `(function() {
		var originalFetch = window.fetch ? window.fetch.bind(window) : null;
		window.fetch = function(input, options) {
			var url = '';
			if (typeof input === 'string') {
				url = input;
			} else if (input && typeof input.url === 'string') {
				url = input.url;
			}
			if (url.indexOf('/auth/logout') !== -1) {
				return Promise.resolve(new Response(null, { status: 403 }));
			}
			if (originalFetch) {
				return originalFetch(input, options);
			}
			return Promise.reject(new Error('fetch unavailable'));
		};
		window.__loopawareLogoutDelegate = function() {
			return Promise.resolve();
		};
	}())`
	dashboardBodyModalOpenScript = `(function() {
		return document.body && document.body.classList.contains('modal-open');
	}())`
	dashboardSettingsHooksReadyScript    = "typeof window.__loopawareDashboardSettingsTestHooks !== 'undefined'"
	dashboardSessionTimeoutVisibleScript = `(function() {
                var container = document.getElementById('session-timeout-notification');
                if (!container) { return false; }
                return container.getAttribute('aria-hidden') === 'false';
        }())`
	dashboardSessionTimeoutAtBottomScript = `(function() {
		var container = document.getElementById('session-timeout-notification');
		if (!container) { return false; }
		var rect = container.getBoundingClientRect();
		var viewportHeight = window.innerHeight || document.documentElement.clientHeight || 0;
		if (!viewportHeight) { return false; }
		return Math.abs(rect.bottom - viewportHeight) <= 2;
	}())`
	dashboardReadAutoLogoutSettingsScript = `(function() {
                if (!window.__loopawareDashboardSettingsTestHooks) { return null; }
                return window.__loopawareDashboardSettingsTestHooks.readAutoLogoutSettings();
	}())`
	dashboardReadSessionTimeoutStartRequestedScript = `(function() {
                if (!window.__loopawareDashboardSettingsTestHooks || typeof window.__loopawareDashboardSettingsTestHooks.readSessionTimeoutStartRequested !== 'function') {
                        return false;
                }
                return !!window.__loopawareDashboardSettingsTestHooks.readSessionTimeoutStartRequested();
	}())`
	dashboardReadAutoLogoutMinimumsScript = `(function() {
		if (!window.__loopawareDashboardSettingsTestHooks) {
			return { minPromptSeconds: 0, minLogoutSeconds: 0, minimumGapSeconds: 0, maxPromptSeconds: 0, maxLogoutSeconds: 0 };
		}
		return {
			minPromptSeconds: window.__loopawareDashboardSettingsTestHooks.minPromptSeconds || 0,
			minLogoutSeconds: window.__loopawareDashboardSettingsTestHooks.minLogoutSeconds || 0,
			minimumGapSeconds: window.__loopawareDashboardSettingsTestHooks.minimumGapSeconds || 0,
			maxPromptSeconds: window.__loopawareDashboardSettingsTestHooks.maxPromptSeconds || 0,
			maxLogoutSeconds: window.__loopawareDashboardSettingsTestHooks.maxLogoutSeconds || 0
		};
	}())`
	dashboardUserEmailSelector                = "#user-email"
	dashboardFooterSelector                   = "#dashboard-footer"
	dashboardTrafficStatusSelector            = "#traffic-status"
	dashboardVisitCountSelector               = "#visit-count"
	dashboardUniqueVisitorCountSelector       = "#unique-visitor-count"
	dashboardTopPagesTableBodySelector        = "#top-pages-table-body"
	dashboardTopPagesPlaceholderText          = "No visits yet."
	dashboardForcePromptScript                = "if (window.__loopawareDashboardIdleTestHooks && typeof window.__loopawareDashboardIdleTestHooks.forcePrompt === 'function') { window.__loopawareDashboardIdleTestHooks.forcePrompt(); }"
	dashboardForceLogoutScript                = "if (window.__loopawareDashboardIdleTestHooks && typeof window.__loopawareDashboardIdleTestHooks.forceLogout === 'function') { window.__loopawareDashboardIdleTestHooks.forceLogout(); }"
	dashboardNotificationBackgroundScript     = `window.getComputedStyle(document.querySelector("#session-timeout-notification")).backgroundColor`
	dashboardLocationPathScript               = "window.location.pathname"
	dashboardClearAutoLogoutStorageKeysScript = `(function() {
		var baseKey = '%s';
		var emails = [%q, %q];
		if (!window.localStorage) {
			return true;
		}
		window.localStorage.removeItem(baseKey);
		emails.forEach(function(value) {
			var normalizedEmail = '';
			if (typeof value === 'string') {
				normalizedEmail = value.trim().toLowerCase();
			}
			if (!normalizedEmail) {
				return;
			}
			var storageKey = baseKey + ':' + encodeURIComponent(normalizedEmail);
			window.localStorage.removeItem(storageKey);
		});
		return true;
	}())`
	dashboardLegacyAutoLogoutStoragePresentScript = `(function() {
		var storageKey = '%s';
		if (!window.localStorage) {
			return false;
		}
		return window.localStorage.getItem(storageKey) !== null;
	}())`
	dashboardSetLegacyAutoLogoutSettingsScript = `(function() {
		var storageKey = '%s';
		var payload = {
			enabled: true,
			prompt_seconds: %d,
			logout_seconds: %d
		};
		if (window.localStorage) {
			window.localStorage.setItem(storageKey, JSON.stringify(payload));
		}
		return true;
	}())`
	dashboardDisableGoogleAutoSelectTrackingScript = `(function() {
		var storageKey = '%s';
		if (window.localStorage) {
			window.localStorage.removeItem(storageKey);
		}
		if (!window.google) { window.google = {}; }
		if (!window.google.accounts) { window.google.accounts = {}; }
		if (!window.google.accounts.id) { window.google.accounts.id = {}; }
		var original = window.google.accounts.id.disableAutoSelect;
		window.google.accounts.id.disableAutoSelect = function() {
			if (window.localStorage) {
				window.localStorage.setItem(storageKey, 'true');
			}
			if (typeof original === 'function') {
				return original();
			}
			return undefined;
		};
	}())`
	landingMarkAuthenticatedScript = `(function() {
		var profile = { display: 'Test User', email: 'test@example.com', avatar_url: 'data:image/gif;base64,R0lGODlhAQABAIAAAP///wAAACH5BAEAAAAALAAAAAABAAEAAAICRAEAOw==' };
		window.__loopawareTestProfile = profile;
		window.initAuthClient = function(options) {
			if (options && typeof options.onAuthenticated === 'function') {
				options.onAuthenticated(profile);
			}
			return Promise.resolve(profile);
		};
		window.getCurrentUser = function() {
			return Promise.resolve(profile);
		};
		function markAuthenticated() {
			var headerHost = document.querySelector('mpr-header');
			if (!headerHost) {
				return false;
			}
			headerHost.setAttribute('data-user-display', profile.display);
			headerHost.setAttribute('data-user-email', profile.email);
			headerHost.setAttribute('data-user-avatar-url', profile.avatar_url);
			return true;
		}
		function scheduleMark() {
			var remainingAttempts = 60;
			function attemptMark() {
				markAuthenticated();
				remainingAttempts -= 1;
				if (remainingAttempts > 0) {
					window.setTimeout(attemptMark, 100);
				}
			}
			attemptMark();
		}
		if (document.readyState === 'loading') {
			document.addEventListener('DOMContentLoaded', scheduleMark);
		} else {
			scheduleMark();
		}
	}())`
	landingDelayedNonceScript = `(function() {
		var delayMs = %d;
		window.__loopawareNonceResolved = false;
		window.requestNonce = function() {
			return new Promise(function(resolve) {
				window.setTimeout(function() {
					window.__loopawareNonceResolved = true;
					resolve('test-nonce');
				}, delayMs);
			});
		};
		if (!window.google) { window.google = {}; }
		if (!window.google.accounts) { window.google.accounts = {}; }
		if (!window.google.accounts.id) { window.google.accounts.id = {}; }
		window.google.accounts.id.initialize = function(config) {};
		window.google.accounts.id.renderButton = function(target) {
			if (!target || typeof document === 'undefined') {
				return;
			}
			var button = document.createElement('div');
			button.textContent = 'Sign in';
			target.appendChild(button);
		};
		window.google.accounts.id.prompt = function() {};
	}())`
	dashboardTestTauthStubScript = `(function() {
		if (typeof window === 'undefined') {
			return;
		}

		var runtimeKey = '__loopawareTestTauthRuntime';
		var sessionCookieName = '` + dashboardTestSessionCookieName + `';

		var runtime = window[runtimeKey];
		if (!runtime || typeof runtime !== 'object') {
			runtime = { tenantId: '', profile: null, options: null };
			window[runtimeKey] = runtime;
		}

		function readCookieValue(name) {
			if (typeof document === 'undefined' || typeof document.cookie !== 'string') {
				return '';
			}
			var prefix = String(name || '').trim() + '=';
			if (prefix === '=') {
				return '';
			}
			var parts = document.cookie.split(';');
			for (var index = 0; index < parts.length; index += 1) {
				var entry = parts[index];
				if (!entry) {
					continue;
				}
				var trimmed = entry.trim();
				if (trimmed.indexOf(prefix) !== 0) {
					continue;
				}
				return trimmed.slice(prefix.length);
			}
			return '';
		}

		function decodeBase64Url(input) {
			if (!input || typeof input !== 'string' || typeof window.atob !== 'function') {
				return '';
			}
			var normalized = input.replace(/-/g, '+').replace(/_/g, '/');
			var padding = normalized.length % 4;
			if (padding === 2) {
				normalized += '==';
			} else if (padding === 3) {
				normalized += '=';
			} else if (padding !== 0) {
				return '';
			}
			try {
				return window.atob(normalized);
			} catch (error) {
				return '';
			}
		}

		function parseSessionClaims() {
			var token = readCookieValue(sessionCookieName);
			if (!token) {
				return null;
			}
			var parts = token.split('.');
			if (!parts || parts.length < 2) {
				return null;
			}
			var payload = decodeBase64Url(parts[1]);
			if (!payload) {
				return null;
			}
			try {
				return JSON.parse(payload);
			} catch (error) {
				return null;
			}
		}

		function resolveProfileFromClaims(claims) {
			if (!claims || typeof claims !== 'object') {
				return null;
			}
			var email = typeof claims.user_email === 'string' ? claims.user_email.trim() : '';
			var display = typeof claims.user_display_name === 'string' ? claims.user_display_name.trim() : '';
			var avatarUrl = typeof claims.user_avatar_url === 'string' ? claims.user_avatar_url.trim() : '';
			var userId = typeof claims.user_id === 'string' ? claims.user_id.trim() : '';
			var roles = Array.isArray(claims.user_roles) ? claims.user_roles.slice() : [];
			if (!email && !display && !avatarUrl && !userId) {
				return null;
			}
			if (!display) {
				display = email;
			}
			return {
				user_id: userId,
				user_email: email,
				email: email,
				display: display,
				avatar_url: avatarUrl,
				roles: roles
			};
		}

		function hydrateProfile() {
			var claims = parseSessionClaims();
			runtime.profile = resolveProfileFromClaims(claims);
			return runtime.profile;
		}

		function setAuthTenantId(tenantId) {
			runtime.tenantId = String(tenantId || '');
		}

		function getCurrentUser() {
			return runtime.profile;
		}

		function initAuthClient(options) {
			runtime.options = options || null;
			var profile = hydrateProfile();
			try {
				if (profile && options && typeof options.onAuthenticated === 'function') {
					options.onAuthenticated(profile);
				}
				if (!profile && options && typeof options.onUnauthenticated === 'function') {
					options.onUnauthenticated();
				}
			} catch (error) {}
			return Promise.resolve();
		}

		function apiFetch(url, initOptions) {
			var merged = Object.assign({}, initOptions || {});
			merged.credentials = 'include';
			return window.fetch(url, merged);
		}

		function getAuthEndpoints() {
			return {
				baseUrl: '',
				meUrl: '/api/me',
				nonceUrl: '/auth/nonce',
				googleUrl: '/auth/google',
				refreshUrl: '/auth/refresh',
				logoutUrl: '/auth/logout'
			};
		}

		function requestNonce() {
			return Promise.resolve('test-nonce');
		}

		function exchangeGoogleCredential() {
			return Promise.reject(new Error('tauth.exchange_not_supported'));
		}

		function logout() {
			runtime.profile = null;
			return Promise.resolve();
		}

		hydrateProfile();

		if (typeof window.setAuthTenantId !== 'function') {
			window.setAuthTenantId = setAuthTenantId;
		}
		if (typeof window.getCurrentUser !== 'function') {
			window.getCurrentUser = getCurrentUser;
		}
		if (typeof window.initAuthClient !== 'function') {
			window.initAuthClient = initAuthClient;
		}
		if (typeof window.apiFetch !== 'function') {
			window.apiFetch = apiFetch;
		}
		if (typeof window.getAuthEndpoints !== 'function') {
			window.getAuthEndpoints = getAuthEndpoints;
		}
		if (typeof window.requestNonce !== 'function') {
			window.requestNonce = requestNonce;
		}
		if (typeof window.exchangeGoogleCredential !== 'function') {
			window.exchangeGoogleCredential = exchangeGoogleCredential;
		}
		if (typeof window.logout !== 'function') {
			window.logout = logout;
		}
	}())`
	landingSigninDisabledScript = `(function() {
		var header = document.querySelector('mpr-header');
		if (!header) { return false; }
		return !!header.querySelector('[data-loopaware-signin-disabled="true"]');
	}())`
	landingSigninDisabledWhileNoncePendingScript = `(function() {
		if (window.__loopawareNonceResolved) { return false; }
		var header = document.querySelector('mpr-header');
		if (!header) { return false; }
		return !!header.querySelector('[data-loopaware-signin-disabled="true"]');
	}())`
	landingNonceResolvedScript = `(function() {
		return window.__loopawareNonceResolved === true;
	}())`
	dashboardIdleHooksReadyScript  = "typeof window.__loopawareDashboardIdleTestHooks !== 'undefined'"
	dashboardSelectFirstSiteScript = `(function() {
	                var list = document.getElementById('sites-list');
	                if (!list) { return false; }
	                var item = list.querySelector('[data-site-id]');
                if (!item) { return false; }
                item.click();
                return true;
        }())`
	dashboardTrafficStatusHiddenScript = `(function() {
                var status = document.querySelector('#traffic-status');
                if (!status) { return false; }
                return status.classList.contains('d-none');
        }())`
	dashboardEditSiteAllowedOriginsSelector       = "#edit-site-origin"
	dashboardSubscribeAllowedOriginsInputSelector = "input[data-subscribe-origin-placeholder=\"true\"]"
	dashboardFirstSiteOriginTextScript            = `(function() {
		var item = document.querySelector('#sites-list [data-site-id]');
		if (!item) { return ''; }
		var elements = item.querySelectorAll('div');
		if (!elements || elements.length < 2) { return ''; }
		return (elements[1].textContent || '').trim();
	}())`
	dashboardTopPagesPlaceholderScript = `(function() {
                var body = document.querySelector('#top-pages-table-body');
                if (!body) { return ''; }
                var row = body.querySelector('tr');
                if (!row) { return ''; }
                return row.textContent.trim();
        }())`
	dashboardTopPagesRowsScript = `(function() {
                var body = document.querySelector('#top-pages-table-body');
                if (!body) { return []; }
                var rows = body.querySelectorAll('tr');
                var results = [];
                rows.forEach(function(row) {
                        var cells = row.querySelectorAll('td');
                        if (cells.length < 2) { return; }
                        results.push({ path: cells[0].textContent.trim(), count: cells[1].textContent.trim() });
                });
                return results;
        }())`
	dashboardVisitCountsScript = `(function() {
                var total = document.getElementById('visit-count');
                var unique = document.getElementById('unique-visitor-count');
                return {
                        totalVisible: !!(total && !total.classList.contains('d-none')),
                        totalText: total ? total.textContent.trim() : '',
                        uniqueVisible: !!(unique && !unique.classList.contains('d-none')),
                        uniqueText: unique ? unique.textContent.trim() : ''
                };
        }())`
	dashboardNoMessagesPlaceholderScript = `(function() {
                var body = document.getElementById('feedback-table-body');
                if (!body) { return false; }
                return body.textContent.indexOf('No feedback yet.') !== -1;
        }())`
	dashboardFeedbackRenderedScript = `(function() {
		var body = document.getElementById('feedback-table-body');
		if (!body) { return false; }
		var rows = body.querySelectorAll('tr');
		if (!rows.length) { return false; }
		for (var index = 0; index < rows.length; index++) {
			var cells = rows[index].querySelectorAll('td');
			if (cells.length < 4) { continue; }
			if (cells[1].textContent.indexOf('auto@example.com') !== -1 && cells[2].textContent.indexOf('Auto refresh message') !== -1 && cells[3].textContent.trim() === 'mailed') {
				return true;
			}
		}
		return false;
	}())`
	dashboardFeedbackCountScript = `(function() {
		var badge = document.getElementById('feedback-count');
		if (!badge) { return false; }
		if (badge.classList.contains('d-none')) { return false; }
		var textContent = (badge.textContent || '').trim();
		return textContent === '1';
	}())`
	dashboardDocumentThemeScript        = "document.documentElement.getAttribute('data-bs-theme') || ''"
	dashboardDocumentMprThemeScript     = "document.documentElement.getAttribute('data-mpr-theme') || ''"
	dashboardStoredDashboardThemeScript = "localStorage.getItem('loopaware_dashboard_theme') || ''"
	dashboardStoredPublicThemeScript    = "localStorage.getItem('loopaware_public_theme') || ''"
	dashboardStoredLandingThemeScript   = "localStorage.getItem('loopaware_landing_theme') || ''"
	dashboardSeedPublicThemeScript      = `localStorage.setItem('loopaware_public_theme','dark');localStorage.removeItem('loopaware_dashboard_theme');localStorage.removeItem('loopaware_theme');`
	dashboardClearThemeStorageScript    = `localStorage.removeItem('loopaware_public_theme');localStorage.removeItem('loopaware_landing_theme');localStorage.removeItem('loopaware_dashboard_theme');localStorage.removeItem('loopaware_theme');`
	dashboardSiteFaviconSelector        = "#sites-list [data-site-id] img"
	dashboardSiteFaviconVisibleScript   = `(function() {
			var list = document.getElementById('sites-list');
			if (!list) { return false; }
		var item = list.querySelector('[data-site-id]');
		if (!item) { return false; }
		var icon = item.querySelector('img');
		if (!icon || icon.classList.contains('d-none')) { return false; }
		var bounds = icon.getBoundingClientRect();
		return bounds.width > 0 && bounds.height > 0 && !!icon.src;
	}())`
	dashboardCaptureWindowOpenScript = `(function() {
		window.__loopawareOpenedWindows = [];
		window.open = function(url, target, features) {
			window.__loopawareOpenedWindows.push({
				url: typeof url === 'string' ? url : '',
				target: typeof target === 'string' ? target : '',
				features: typeof features === 'string' ? features : ''
			});
			return null;
		};
		return true;
	}())`
	dashboardOpenedWindowCallsScript = `(function() {
    return window.__loopawareOpenedWindows || [];
}())`
	dashboardDispatchSyntheticMousemoveScript = "document.dispatchEvent(new Event('mousemove'))"
	widgetTestSummaryOffsetScript             = `document.getElementById('widget-test-summary-offset') ? document.getElementById('widget-test-summary-offset').textContent : ''`
	widgetTestBottomOffsetInputSelector       = "#widget-test-bottom-offset"
	widgetTestOffsetIncreaseSelector          = "#widget-test-offset-increase"
	widgetTestOffsetDecreaseSelector          = "#widget-test-offset-decrease"
	widgetTestSaveButtonSelector              = "#widget-test-save"
	widgetTestReadOffsetInputScript           = `(function() {
		var input = document.getElementById('widget-test-bottom-offset');
		if (!input) { return ''; }
		return String(input.value || '');
	}())`
	widgetTestReadSummaryOffsetScript = `(function() {
	var summary = document.getElementById('widget-test-summary-offset');
	if (!summary) { return ''; }
	return String(summary.textContent || '');
}())`
	widgetTestBubbleBottomScript = `(function() {
	var bubble = document.getElementById('mp-feedback-bubble');
	if (!bubble) { return ''; }
	return bubble.style.bottom || '';
}())`
	widgetTestPanelBottomScript = `(function() {
var panel = document.getElementById('mp-feedback-panel');
if (!panel) { return ''; }
return panel.style.bottom || '';
}())`
	dashboardWidgetBottomOffsetPresenceScript = `(function() {
	return !!document.getElementById('widget-placement-bottom-offset');
}())`
	widgetTestEnsurePreviewElementsScript = `(function() {
	var bubble = document.getElementById('mp-feedback-bubble');
	if (!bubble) {
		bubble = document.createElement('div');
		bubble.id = 'mp-feedback-bubble';
		bubble.style.position = 'fixed';
		document.body.appendChild(bubble);
	}
	var panel = document.getElementById('mp-feedback-panel');
	if (!panel) {
		panel = document.createElement('div');
		panel.id = 'mp-feedback-panel';
			panel.style.position = 'fixed';
			document.body.appendChild(panel);
		}
		return true;
	}())`
	widgetBubblePresenceScript = `(function() {
		return !!document.getElementById('mp-feedback-bubble');
	}())`
)

var dashboardTestLandingPath = httpapi.LandingPagePath

type stubDashboardNotifier struct{}

func (stubDashboardNotifier) NotifyFeedback(ctx context.Context, site model.Site, feedback model.Feedback) (string, error) {
	return model.FeedbackDeliveryMailed, nil
}

func (stubDashboardNotifier) NotifySubscription(ctx context.Context, site model.Site, subscriber model.Subscriber) error {
	return nil
}

type dashboardIntegrationHarness struct {
	router             *gin.Engine
	authManager        *httpapi.AuthManager
	faviconManager     *httpapi.SiteFaviconManager
	database           *gorm.DB
	sqlDB              *sql.DB
	server             *httptest.Server
	baseURL            string
	subscriptionEvents *httpapi.SubscriptionTestEventBroadcaster
}

type dashboardHarnessOptions struct {
	subscriptionNotifier httpapi.SubscriptionNotifier
	emailSender          httpapi.EmailSender
	userLoadDelay        time.Duration
}

type dashboardHarnessOption func(*dashboardHarnessOptions)

func withSubscriptionNotifier(notifier httpapi.SubscriptionNotifier) dashboardHarnessOption {
	return func(options *dashboardHarnessOptions) {
		if notifier != nil {
			options.subscriptionNotifier = notifier
		}
	}
}

func withEmailSender(sender httpapi.EmailSender) dashboardHarnessOption {
	return func(options *dashboardHarnessOptions) {
		if sender != nil {
			options.emailSender = sender
		}
	}
}

func withUserLoadDelay(delay time.Duration) dashboardHarnessOption {
	return func(options *dashboardHarnessOptions) {
		if delay > 0 {
			options.userLoadDelay = delay
		}
	}
}

func TestDashboardSessionTimeoutPromptHonorsThemeAndLogout(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Scenario Site",
		AllowedOrigin: "https://widget.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	var waitNavigation func()

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	currentPath := evaluateScriptString(t, page, dashboardLocationPathScript)
	require.Equal(t, dashboardTestDashboardRoute, currentPath)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
        var element = document.querySelector(%q);
        if (!element) { return false; }
        var style = window.getComputedStyle(element);
        if (!style) { return false; }
        return style.display !== 'none' && style.visibility !== 'hidden';
    }())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardPromptVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSessionTimeoutAtBottomScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	lightBackgroundColor := evaluateScriptString(t, page, dashboardNotificationBackgroundScript)
	lightColor := mustParseRGBColor(t, lightBackgroundColor)

	evaluateScriptInto(t, page, `(function(){
		var footer = document.getElementById('dashboard-footer');
		if (!footer || typeof footer.dispatchEvent !== 'function') { return false; }
		footer.dispatchEvent(new CustomEvent('mpr-footer:theme-change', { detail: { theme: 'dark' } }));
		return true;
	}())`, nil)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardPromptVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	darkBackgroundColor := evaluateScriptString(t, page, dashboardNotificationBackgroundScript)
	darkColor := mustParseRGBColor(t, darkBackgroundColor)
	require.NotEqual(t, lightColor, darkColor)

	dismissClickScript := fmt.Sprintf("document.querySelector(%q).click()", dashboardDismissButtonSelector)
	evaluateScriptInto(t, page, dismissClickScript, nil)
	require.Eventually(t, func() bool {
		notificationElement, elementErr := page.Element(dashboardNotificationSelector)
		if elementErr != nil {
			return true
		}
		visible, visibleErr := notificationElement.Visible()
		if visibleErr != nil {
			return false
		}
		return !visible
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	confirmClickScript := fmt.Sprintf("document.querySelector(%q).click()", dashboardConfirmButtonSelector)
	evaluateScriptInto(t, page, confirmClickScript, nil)
	waitNavigation()

	require.Eventually(t, func() bool {
		info, infoErr := page.Info()
		if infoErr != nil {
			return false
		}
		parsed, parseErr := url.Parse(info.URL)
		if parseErr != nil {
			return false
		}
		return parsed.Path == dashboardTestLandingPath
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardSessionTimeoutStartsAfterUserSettingsLoad(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail, withUserLoadDelay(2*time.Second))
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	dashboardPage := buildHeadlessPage(testingT)

	setPageCookie(testingT, dashboardPage, harness.baseURL, sessionCookie)
	navigateToPage(testingT, dashboardPage, harness.baseURL+dashboardTestDashboardRoute)

	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, dashboardPage, dashboardSettingsHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.False(testingT, evaluateScriptBoolean(testingT, dashboardPage, dashboardReadSessionTimeoutStartRequestedScript))

	userEmailVisibleScript := fmt.Sprintf(`(function(){
        var element = document.querySelector(%q);
        if (!element) { return false; }
        var style = window.getComputedStyle(element);
        if (!style) { return false; }
        return style.display !== 'none' && style.visibility !== 'hidden';
    }())`, dashboardUserEmailSelector)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, dashboardPage, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, dashboardPage, dashboardReadSessionTimeoutStartRequestedScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardSettingsModalOpensAndDismissesViaBackdrop(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Settings Modal Site",
		AllowedOrigin: "https://modal.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	openDashboardUserMenu(t, page)
	openDashboardUserMenu(t, page)
	clickSelector(t, page, dashboardUserMenuAccountSettingsSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	waitForVisibleElement(t, page, dashboardSettingsModalSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardBodyModalOpenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var modalContent struct {
		TextLength float64 `json:"textLength"`
		Contrast   float64 `json:"contrast"`
	}
	evaluateScriptInto(t, page, dashboardSettingsModalContentScript, &modalContent)
	require.Greater(t, modalContent.TextLength, 0.0)
	require.Greater(t, modalContent.Contrast, 30.0)

	var modalBounds viewportBounds
	evaluateScriptInto(t, page, `(function(){
		var dialog = document.querySelector('.modal-dialog');
		if (!dialog) {
			return { left: 0, top: 0, width: 0, height: 0 };
		}
		var rect = dialog.getBoundingClientRect();
		return { left: rect.left, top: rect.top, width: rect.width, height: rect.height };
	}())`, &modalBounds)

	clickX := modalBounds.Left - 10.0
	if clickX < 0 {
		clickX = modalBounds.Left + modalBounds.Width + 10.0
	}
	if clickX > float64(headlessViewportWidth-1) {
		clickX = float64(headlessViewportWidth - 1)
	}
	clickY := modalBounds.Top - 10.0
	if clickY < 0 {
		clickY = modalBounds.Top + modalBounds.Height + 10.0
	}
	if clickY > float64(headlessViewportHeight-1) {
		clickY = float64(headlessViewportHeight - 1)
	}

	t.Logf("settings modal bounds: left=%.2f top=%.2f width=%.2f height=%.2f", modalBounds.Left, modalBounds.Top, modalBounds.Width, modalBounds.Height)
	t.Logf("click outside coords: x=%.2f y=%.2f", clickX, clickY)
	clickTarget := evaluateScriptString(t, page, fmt.Sprintf(`(function(x, y){
		var element = document.elementFromPoint(x, y);
		if (!element) { return 'none'; }
		var descriptor = element.tagName.toLowerCase();
		if (element.id) {
			descriptor += '#' + element.id;
		}
		if (element.className) {
			descriptor += '.' + element.className.split(/\s+/).filter(Boolean).join('.');
		}
		return descriptor;
	}(%f, %f))`, clickX, clickY))
	t.Logf("click target: %s", clickTarget)
	backdropConfig := evaluateScriptString(t, page, `(function(){
		if (!window.bootstrap) { return 'none'; }
		var element = document.getElementById('settings-modal');
		if (!element) { return 'none'; }
		var instance = window.bootstrap.Modal.getInstance(element);
		if (!instance) { return 'none'; }
		return String(instance._config && instance._config.backdrop);
	}())`)
	t.Logf("settings modal backdrop config: %s", backdropConfig)
	require.True(t, evaluateScriptBoolean(t, page, `(function(){
		var element = document.getElementById('settings-modal');
		return !!(element && element.dataset && element.dataset.loopawareSettingsModalDismissAttached === 'true');
	}())`))

	require.True(t, evaluateScriptBoolean(t, page, fmt.Sprintf(`(function(x, y){
		var dialog = document.querySelector('.modal-dialog');
		if (!dialog) { return false; }
		var rect = dialog.getBoundingClientRect();
		return x < rect.left || x > rect.right || y < rect.top || y > rect.bottom;
	}(%f, %f))`, clickX, clickY)))

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
		if (!window.bootstrap) { return false; }
		var element = document.getElementById('settings-modal');
		if (!element) { return false; }
		var instance = window.bootstrap.Modal.getOrCreateInstance(element);
		if (!instance) { return false; }
		element.dataset.loopawareSettingsModalDismissTriggered = 'true';
		instance.hide();
		element.classList.remove('show');
		element.setAttribute('aria-hidden', 'true');
		element.style.display = 'none';
		if (document.body) {
			document.body.classList.remove('modal-open');
		}
		return true;
	}())`))
	require.True(t, evaluateScriptBoolean(t, page, `(function(){
		var element = document.getElementById('settings-modal');
		return !!(element && element.dataset && element.dataset.loopawareSettingsModalDismissTriggered === 'true');
	}())`))

	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardBodyModalOpenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardLogoutFallsBackToFetchWhenLogoutFails(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardLogoutFetchClearScript, nil)
	evaluateScriptInto(t, page, dashboardLogoutTestHookScript, nil)

	openDashboardUserMenu(t, page)
	openDashboardUserMenu(t, page)
	clickSelector(t, page, dashboardUserMenuLogoutButtonSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardLogoutFetchCalledScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardLocationPathScript) == dashboardTestLandingPath
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardLogoutFallsBackToFormWhenLogoutAndFetchFail(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardLogoutFormFallbackScript, nil)

	openDashboardUserMenu(t, page)
	openDashboardUserMenu(t, page)
	clickSelector(t, page, dashboardUserMenuLogoutButtonSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardLogoutFormPresentScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardLocationPathScript) == dashboardTestLandingPath
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardLogoutDoesNotRedirectWhenLogoutEndpointForbidden(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardLogoutForbiddenScript, nil)

	openDashboardUserMenu(t, page)
	openDashboardUserMenu(t, page)
	clickSelector(t, page, dashboardUserMenuLogoutButtonSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var menu = document.querySelector('mpr-user[data-loopaware-user-menu="true"]');
			if (!menu) { return ''; }
			return menu.getAttribute('data-mpr-user-error') || '';
		}())`) == "loopaware.logout_failed"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Equal(t, dashboardTestDashboardRoute, evaluateScriptString(t, page, dashboardLocationPathScript))
}

func TestDashboardLogoutDisablesGoogleAutoSelect(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(testingT)

	setPageCookie(testingT, page, harness.baseURL, sessionCookie)

	navigateToPage(testingT, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(testingT, func() bool {
		return evaluateScriptString(testingT, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(testingT, page, fmt.Sprintf(dashboardDisableGoogleAutoSelectTrackingScript, dashboardDisableGoogleAutoSelectStorageKey), nil)

	openDashboardUserMenu(testingT, page)
	openDashboardUserMenu(testingT, page)
	clickSelector(testingT, page, dashboardUserMenuLogoutButtonSelector)

	disableAutoSelectScript := fmt.Sprintf(`(function(){
		if (!window.localStorage) { return ''; }
		return window.localStorage.getItem(%q) || '';
	}())`, dashboardDisableGoogleAutoSelectStorageKey)
	require.Eventually(testingT, func() bool {
		return evaluateScriptString(testingT, page, disableAutoSelectScript) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(testingT, func() bool {
		return evaluateScriptString(testingT, page, dashboardLocationPathScript) == dashboardTestLandingPath
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardUserMenuHasSingleAvatarAndExpectedItems(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookieWithAvatar(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName, dashboardTestAvatarDataURI)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	openDashboardUserMenu(t, page)
	openDashboardUserMenu(t, page)

	var state struct {
		LoopawareMenuCount       int    `json:"loopawareMenuCount"`
		HeaderUserMenuCount      int    `json:"headerUserMenuCount"`
		ExtraHeaderUserMenuCount int    `json:"extraHeaderUserMenuCount"`
		HeaderAvatarCount        int    `json:"headerAvatarCount"`
		VisibleHeaderAvatarCount int    `json:"visibleHeaderAvatarCount"`
		AvatarCount              int    `json:"avatarCount"`
		DisplayName              string `json:"displayName"`
		AvatarVisible            bool   `json:"avatarVisible"`
		NameVisible              bool   `json:"nameVisible"`
	}
	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, dashboardUserMenuStateScript, &state)
		return state.AvatarVisible && state.AvatarCount == 1 && state.HeaderUserMenuCount == 1 && state.ExtraHeaderUserMenuCount == 0 && state.HeaderAvatarCount == 1 && state.VisibleHeaderAvatarCount == 1
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	evaluateScriptInto(t, page, dashboardUserMenuStateScript, &state)
	require.Equal(t, 1, state.LoopawareMenuCount)
	require.Equal(t, 1, state.HeaderUserMenuCount)
	require.Equal(t, 0, state.ExtraHeaderUserMenuCount)
	require.Equal(t, 1, state.HeaderAvatarCount)
	require.Equal(t, 1, state.VisibleHeaderAvatarCount)
	require.Equal(t, 1, state.AvatarCount)
	require.True(t, state.AvatarVisible)
	require.False(t, state.NameVisible)

	var menuCount int
	evaluateScriptInto(t, page, `(function(){
		return document.querySelectorAll('mpr-user[data-loopaware-user-menu="true"]').length;
	}())`, &menuCount)
	require.Equal(t, 1, menuCount)

	var headerState struct {
		HasUserMenu          bool `json:"hasUserMenu"`
		HasLegacyProfileMenu bool `json:"hasLegacyProfileMenu"`
	}
	evaluateScriptInto(t, page, dashboardHeaderDefaultProfileStateScript, &headerState)
	require.True(t, headerState.HasUserMenu)
	require.False(t, headerState.HasLegacyProfileMenu)

	accountSettingsLabel := evaluateScriptString(t, page, fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return ''; }
		return String(element.textContent || '').trim();
	}())`, dashboardUserMenuAccountSettingsSelector))
	require.Equal(t, "Account Settings", accountSettingsLabel)

	logoutLabel := evaluateScriptString(t, page, fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return ''; }
		return String(element.textContent || '').trim();
	}())`, dashboardUserMenuLogoutButtonSelector))
	require.Equal(t, "Logout", logoutLabel)
}

func TestDashboardHeaderUsesSingleUserMenu(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookieWithAvatar(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName, dashboardTestAvatarDataURI)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-loopaware-auth-bound') || '';
		}())`) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var state struct {
		HasUserMenu          bool `json:"hasUserMenu"`
		HasLegacyProfileMenu bool `json:"hasLegacyProfileMenu"`
	}
	evaluateScriptInto(t, page, dashboardHeaderDefaultProfileStateScript, &state)

	require.True(t, state.HasUserMenu)
	require.False(t, state.HasLegacyProfileMenu)

	var menuCount int
	evaluateScriptInto(t, page, `(function(){
		return document.querySelectorAll('mpr-user[data-loopaware-user-menu="true"]').length;
	}())`, &menuCount)
	require.Equal(t, 1, menuCount)
}

func TestDashboardShowsDistinctWidgetSnippets(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Widget Snippet Site",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)
	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	snippetScript := fmt.Sprintf(`(function(){
		var feedback = document.querySelector(%q);
		var subscribe = document.querySelector(%q);
		var traffic = document.querySelector(%q);
		var copyFeedback = document.querySelector(%q);
		var copySubscribe = document.querySelector(%q);
		var copyTraffic = document.querySelector(%q);
		return {
			feedbackCard: !!document.querySelector(%q),
			subscribeCard: !!document.querySelector(%q),
			trafficCard: !!document.querySelector(%q),
			feedback: feedback ? (feedback.value || '').trim() : '',
			subscribe: subscribe ? (subscribe.value || '').trim() : '',
			traffic: traffic ? (traffic.value || '').trim() : '',
			feedbackCopyDisabled: copyFeedback ? !!copyFeedback.disabled : true,
			subscribeCopyDisabled: copySubscribe ? !!copySubscribe.disabled : true,
			trafficCopyDisabled: copyTraffic ? !!copyTraffic.disabled : true
		};
	}())`, dashboardFeedbackWidgetSnippetSelector, dashboardSubscribeWidgetSnippetSelector, dashboardTrafficWidgetSnippetSelector, dashboardFeedbackCopyButtonSelector, dashboardSubscribeCopyButtonSelector, dashboardTrafficCopyButtonSelector, dashboardFeedbackWidgetCardSelector, dashboardSubscribeWidgetCardSelector, dashboardTrafficWidgetCardSelector)

	var snippets struct {
		FeedbackCard          bool   `json:"feedbackCard"`
		SubscribeCard         bool   `json:"subscribeCard"`
		TrafficCard           bool   `json:"trafficCard"`
		Feedback              string `json:"feedback"`
		Subscribe             string `json:"subscribe"`
		Traffic               string `json:"traffic"`
		FeedbackCopyDisabled  bool   `json:"feedbackCopyDisabled"`
		SubscribeCopyDisabled bool   `json:"subscribeCopyDisabled"`
		TrafficCopyDisabled   bool   `json:"trafficCopyDisabled"`
	}

	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, snippetScript, &snippets)
		return snippets.FeedbackCard && snippets.SubscribeCard && snippets.TrafficCard &&
			strings.Contains(snippets.Feedback, site.ID) &&
			strings.Contains(snippets.Subscribe, site.ID) &&
			strings.Contains(snippets.Traffic, site.ID)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, snippets.FeedbackCard)
	require.True(t, snippets.SubscribeCard)
	require.True(t, snippets.TrafficCard)
	require.Contains(t, snippets.Feedback, "/widget.js?site_id="+site.ID)
	require.Contains(t, snippets.Subscribe, "/subscribe.js?site_id="+site.ID)
	require.Contains(t, snippets.Traffic, "/pixel.js?site_id="+site.ID)
	require.False(t, snippets.FeedbackCopyDisabled)
	require.False(t, snippets.SubscribeCopyDisabled)
	require.False(t, snippets.TrafficCopyDisabled)
}

func TestDashboardSectionTabsTogglePanes(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Dashboard Tabs Site",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)
	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	visibilityScript := fmt.Sprintf(`(function(){
		function hidden(selector) {
			var node = document.querySelector(selector);
			if (!node) { return null; }
			return node.classList.contains('d-none');
		}
		return {
			feedbackWidgetHidden: hidden(%q),
			subscribeWidgetHidden: hidden(%q),
			trafficWidgetHidden: hidden(%q),
			feedbackPaneHidden: hidden(%q),
			subscribePaneHidden: hidden(%q),
			trafficPaneHidden: hidden(%q)
		};
	}())`, dashboardFeedbackWidgetCardSelector, dashboardSubscribeWidgetCardSelector, dashboardTrafficWidgetCardSelector, dashboardFeedbackMessagesCardSelector, dashboardSubscribersCardSelector, dashboardTrafficCardSelector)

	var state struct {
		FeedbackWidgetHidden  *bool `json:"feedbackWidgetHidden"`
		SubscribeWidgetHidden *bool `json:"subscribeWidgetHidden"`
		TrafficWidgetHidden   *bool `json:"trafficWidgetHidden"`
		FeedbackPaneHidden    *bool `json:"feedbackPaneHidden"`
		SubscribePaneHidden   *bool `json:"subscribePaneHidden"`
		TrafficPaneHidden     *bool `json:"trafficPaneHidden"`
	}

	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, visibilityScript, &state)
		return state.FeedbackWidgetHidden != nil &&
			state.SubscribeWidgetHidden != nil &&
			state.TrafficWidgetHidden != nil &&
			state.FeedbackPaneHidden != nil &&
			state.SubscribePaneHidden != nil &&
			state.TrafficPaneHidden != nil
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.False(t, *state.FeedbackWidgetHidden)
	require.True(t, *state.SubscribeWidgetHidden)
	require.True(t, *state.TrafficWidgetHidden)
	require.False(t, *state.FeedbackPaneHidden)
	require.True(t, *state.SubscribePaneHidden)
	require.True(t, *state.TrafficPaneHidden)

	tabLayoutScript := `(function() {
		var tabs = document.getElementById('dashboard-section-tabs');
		var feedback = document.getElementById('dashboard-section-tab-feedback');
		var subscriptions = document.getElementById('dashboard-section-tab-subscriptions');
		var traffic = document.getElementById('dashboard-section-tab-traffic');
		if (!tabs || !feedback || !subscriptions || !traffic) {
			return null;
		}
		var tabsRect = tabs.getBoundingClientRect();
		var feedbackRect = feedback.getBoundingClientRect();
		var subscriptionsRect = subscriptions.getBoundingClientRect();
		var trafficRect = traffic.getBoundingClientRect();
		return {
			tabsWidth: tabsRect.width,
			feedbackWidth: feedbackRect.width,
			subscriptionsWidth: subscriptionsRect.width,
			trafficWidth: trafficRect.width
		};
	}())`

	var tabLayout struct {
		TabsWidth          float64 `json:"tabsWidth"`
		FeedbackWidth      float64 `json:"feedbackWidth"`
		SubscriptionsWidth float64 `json:"subscriptionsWidth"`
		TrafficWidth       float64 `json:"trafficWidth"`
	}

	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, tabLayoutScript, &tabLayout)
		return tabLayout.TabsWidth > 0 && tabLayout.FeedbackWidth > 0 && tabLayout.SubscriptionsWidth > 0 && tabLayout.TrafficWidth > 0
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	expectedTabWidth := tabLayout.TabsWidth / 3
	tabWidthDelta := math.Max(2.0, expectedTabWidth*0.06)
	require.InDelta(t, expectedTabWidth, tabLayout.FeedbackWidth, tabWidthDelta)
	require.InDelta(t, expectedTabWidth, tabLayout.SubscriptionsWidth, tabWidthDelta)
	require.InDelta(t, expectedTabWidth, tabLayout.TrafficWidth, tabWidthDelta)
	require.InDelta(t, tabLayout.FeedbackWidth, tabLayout.SubscriptionsWidth, tabWidthDelta)
	require.InDelta(t, tabLayout.SubscriptionsWidth, tabLayout.TrafficWidth, tabWidthDelta)

	waitForVisibleElement(t, page, dashboardSectionTabSubscriptionsSelector)
	clickSelector(t, page, dashboardSectionTabSubscriptionsSelector)
	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, visibilityScript, &state)
		return state.SubscribePaneHidden != nil && !*state.SubscribePaneHidden
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, *state.FeedbackPaneHidden)
	require.False(t, *state.SubscribeWidgetHidden)
	require.False(t, *state.SubscribePaneHidden)

	waitForVisibleElement(t, page, dashboardSectionTabTrafficSelector)
	clickSelector(t, page, dashboardSectionTabTrafficSelector)
	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, visibilityScript, &state)
		return state.TrafficPaneHidden != nil && !*state.TrafficPaneHidden
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, *state.FeedbackPaneHidden)
	require.True(t, *state.SubscribePaneHidden)
	require.False(t, *state.TrafficWidgetHidden)
	require.False(t, *state.TrafficPaneHidden)

	waitForVisibleElement(t, page, dashboardSectionTabFeedbackSelector)
	clickSelector(t, page, dashboardSectionTabFeedbackSelector)
	require.Eventually(t, func() bool {
		evaluateScriptInto(t, page, visibilityScript, &state)
		return state.FeedbackPaneHidden != nil && !*state.FeedbackPaneHidden
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.False(t, *state.FeedbackWidgetHidden)
	require.False(t, *state.FeedbackPaneHidden)
	require.True(t, *state.SubscribePaneHidden)
	require.True(t, *state.TrafficPaneHidden)
}

func TestDashboardSeparatesSubscribersAndTrafficCards(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Dashboard Cards Site",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)
	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	cardScript := fmt.Sprintf(`(function(){
		var sequence = Array.prototype.slice.call(document.querySelectorAll(%q)).map(function(node) {
			return (node.getAttribute('data-dashboard-card') || '').trim();
		});
		var feedbackCard = document.querySelector(%q);
		var subscribersCard = document.querySelector(%q);
		var trafficCard = document.querySelector(%q);
		return {
			cards: sequence,
			subscribersInFeedback: !!(feedbackCard && feedbackCard.querySelector(%q)),
			trafficInFeedback: !!(feedbackCard && feedbackCard.querySelector(%q)),
			subscribersInOwnCard: !!(subscribersCard && subscribersCard.querySelector(%q)),
			trafficInOwnCard: !!(trafficCard && trafficCard.querySelector(%q))
		};
	}())`, dashboardDashboardCardSelector, dashboardFeedbackMessagesCardSelector, dashboardSubscribersCardSelector, dashboardTrafficCardSelector, dashboardSubscribersTableBodySelector, dashboardTopPagesTableBodySelector, dashboardSubscribersTableBodySelector, dashboardTopPagesTableBodySelector)

	var layout struct {
		Cards                 []string `json:"cards"`
		SubscribersInFeedback bool     `json:"subscribersInFeedback"`
		TrafficInFeedback     bool     `json:"trafficInFeedback"`
		SubscribersInOwnCard  bool     `json:"subscribersInOwnCard"`
		TrafficInOwnCard      bool     `json:"trafficInOwnCard"`
	}

	evaluateScriptInto(t, page, cardScript, &layout)

	require.Equal(t, []string{"feedback", "subscribers", "traffic"}, layout.Cards)
	require.False(t, layout.SubscribersInFeedback)
	require.False(t, layout.TrafficInFeedback)
	require.True(t, layout.SubscribersInOwnCard)
	require.True(t, layout.TrafficInOwnCard)
}

func TestDashboardSettingsAutoLogoutConfiguration(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSettingsHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	openDashboardUserMenu(t, page)
	clickSelector(t, page, dashboardUserMenuAccountSettingsSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	toggleElement := waitForVisibleElement(t, page, dashboardSettingsAutoLogoutToggleSelector)
	toggleCheckedScript := `(function(){var toggle=document.querySelector('#settings-auto-logout-enabled');return !!(toggle&&toggle.checked);}())`
	if evaluateScriptBoolean(t, page, toggleCheckedScript) {
		require.NoError(t, toggleElement.Click(proto.InputMouseButtonLeft, 1))
	}
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, toggleCheckedScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		var current struct {
			Enabled       bool    `json:"enabled"`
			PromptSeconds float64 `json:"promptSeconds"`
			LogoutSeconds float64 `json:"logoutSeconds"`
		}
		evaluateScriptInto(t, page, dashboardReadAutoLogoutSettingsScript, &current)
		return !current.Enabled
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	closeButton := waitForVisibleElement(t, page, "#settings-modal .btn-close")
	require.NoError(t, closeButton.Click(proto.InputMouseButtonLeft, 1))
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardBodyModalOpenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSessionTimeoutVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	openDashboardUserMenu(t, page)
	clickSelector(t, page, dashboardUserMenuAccountSettingsSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	toggleElement = waitForVisibleElement(t, page, dashboardSettingsAutoLogoutToggleSelector)
	if !evaluateScriptBoolean(t, page, toggleCheckedScript) {
		require.NoError(t, toggleElement.Click(proto.InputMouseButtonLeft, 1))
	}
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, toggleCheckedScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var minimums struct {
		MinPromptSeconds  float64 `json:"minPromptSeconds"`
		MinLogoutSeconds  float64 `json:"minLogoutSeconds"`
		MinimumGapSeconds float64 `json:"minimumGapSeconds"`
		MaxPromptSeconds  float64 `json:"maxPromptSeconds"`
		MaxLogoutSeconds  float64 `json:"maxLogoutSeconds"`
	}
	evaluateScriptInto(t, page, dashboardReadAutoLogoutMinimumsScript, &minimums)
	minPrompt := int(minimums.MinPromptSeconds)
	minLogout := int(minimums.MinLogoutSeconds)
	minGap := int(minimums.MinimumGapSeconds)
	maxPrompt := int(minimums.MaxPromptSeconds)
	maxLogout := int(minimums.MaxLogoutSeconds)
	if minPrompt <= 0 {
		minPrompt = 10
	}
	if minLogout <= 0 {
		minLogout = 20
	}
	if minGap < 1 {
		minGap = 5
	}
	promptSeconds := minPrompt + minGap + 1
	if maxPrompt > 0 && promptSeconds > maxPrompt {
		promptSeconds = maxPrompt
	}
	if promptSeconds < minPrompt {
		promptSeconds = minPrompt
	}
	logoutSeconds := promptSeconds + minGap + 2
	if logoutSeconds < minLogout {
		logoutSeconds = minLogout
	}
	if maxLogout > 0 && logoutSeconds > maxLogout {
		logoutSeconds = maxLogout
	}
	if logoutSeconds <= promptSeconds+minGap {
		logoutSeconds = promptSeconds + minGap + 1
	}

	setInputValue(t, page, dashboardSettingsAutoLogoutPromptSelector, fmt.Sprintf("%d", promptSeconds))
	setInputValue(t, page, dashboardSettingsAutoLogoutLogoutSelector, fmt.Sprintf("%d", logoutSeconds))

	require.Eventually(t, func() bool {
		var current struct {
			Enabled       bool    `json:"enabled"`
			PromptSeconds float64 `json:"promptSeconds"`
			LogoutSeconds float64 `json:"logoutSeconds"`
		}
		evaluateScriptInto(t, page, dashboardReadAutoLogoutSettingsScript, &current)
		if !current.Enabled {
			return false
		}
		return int(current.PromptSeconds) == promptSeconds && int(current.LogoutSeconds) == logoutSeconds
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	closeButton = waitForVisibleElement(t, page, "#settings-modal .btn-close")
	require.NoError(t, closeButton.Click(proto.InputMouseButtonLeft, 1))
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSettingsModalVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)
	dismissClickScript := fmt.Sprintf("document.querySelector(%q).click()", dashboardDismissButtonSelector)
	evaluateScriptInto(t, page, dismissClickScript, nil)
	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, dashboardSessionTimeoutVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardAutoLogoutSettingsAreUserScoped(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	dashboardPage := buildHeadlessPage(testingT)

	navigateToPage(testingT, dashboardPage, harness.baseURL+dashboardTestLandingPath)
	clearScript := fmt.Sprintf(dashboardClearAutoLogoutStorageKeysScript, dashboardAutoLogoutStorageBaseKey, dashboardTestAdminEmail, dashboardTestSecondaryEmail)
	evaluateScriptInto(testingT, dashboardPage, clearScript, nil)

	legacyPromptSeconds := 300
	legacyLogoutSeconds := 900
	setLegacyScript := fmt.Sprintf(dashboardSetLegacyAutoLogoutSettingsScript, dashboardAutoLogoutStorageBaseKey, legacyPromptSeconds, legacyLogoutSeconds)
	evaluateScriptInto(testingT, dashboardPage, setLegacyScript, nil)

	adminSessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	setPageCookie(testingT, dashboardPage, harness.baseURL, adminSessionCookie)

	navigateToPage(testingT, dashboardPage, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, dashboardPage, dashboardSettingsHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	adminEmailScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return ''; }
		return (element.textContent || '').trim();
	}())`, dashboardUserEmailSelector)
	require.Eventually(testingT, func() bool {
		return evaluateScriptString(testingT, dashboardPage, adminEmailScript) == dashboardTestAdminEmail
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(testingT, func() bool {
		var current struct {
			Enabled       bool    `json:"enabled"`
			PromptSeconds float64 `json:"promptSeconds"`
			LogoutSeconds float64 `json:"logoutSeconds"`
		}
		evaluateScriptInto(testingT, dashboardPage, dashboardReadAutoLogoutSettingsScript, &current)
		if !current.Enabled {
			return false
		}
		return int(current.PromptSeconds) == legacyPromptSeconds && int(current.LogoutSeconds) == legacyLogoutSeconds
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(testingT, func() bool {
		legacyPresentScript := fmt.Sprintf(dashboardLegacyAutoLogoutStoragePresentScript, dashboardAutoLogoutStorageBaseKey)
		return !evaluateScriptBoolean(testingT, dashboardPage, legacyPresentScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	secondarySessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestSecondaryEmail, dashboardTestSecondaryDisplayName)
	setPageCookie(testingT, dashboardPage, harness.baseURL, secondarySessionCookie)

	navigateToPage(testingT, dashboardPage, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, dashboardPage, dashboardSettingsHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	secondaryEmailScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return ''; }
		return (element.textContent || '').trim();
	}())`, dashboardUserEmailSelector)
	require.Eventually(testingT, func() bool {
		return evaluateScriptString(testingT, dashboardPage, secondaryEmailScript) == dashboardTestSecondaryEmail
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(testingT, func() bool {
		var current struct {
			Enabled       bool    `json:"enabled"`
			PromptSeconds float64 `json:"promptSeconds"`
			LogoutSeconds float64 `json:"logoutSeconds"`
		}
		evaluateScriptInto(testingT, dashboardPage, dashboardReadAutoLogoutSettingsScript, &current)
		if !current.Enabled {
			return false
		}
		return !(int(current.PromptSeconds) == legacyPromptSeconds && int(current.LogoutSeconds) == legacyLogoutSeconds)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	adminSessionCookie = createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	setPageCookie(testingT, dashboardPage, harness.baseURL, adminSessionCookie)

	navigateToPage(testingT, dashboardPage, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, dashboardPage, dashboardSettingsHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(testingT, func() bool {
		return evaluateScriptString(testingT, dashboardPage, adminEmailScript) == dashboardTestAdminEmail
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(testingT, func() bool {
		var current struct {
			Enabled       bool    `json:"enabled"`
			PromptSeconds float64 `json:"promptSeconds"`
			LogoutSeconds float64 `json:"logoutSeconds"`
		}
		evaluateScriptInto(testingT, dashboardPage, dashboardReadAutoLogoutSettingsScript, &current)
		if !current.Enabled {
			return false
		}
		return int(current.PromptSeconds) == legacyPromptSeconds && int(current.LogoutSeconds) == legacyLogoutSeconds
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardSessionTimeoutAutoLogout(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	evaluateScriptInto(t, page, dashboardForceLogoutScript, nil)
	waitNavigation()

	require.Eventually(t, func() bool {
		info, infoErr := page.Info()
		if infoErr != nil {
			return false
		}
		parsed, parseErr := url.Parse(info.URL)
		if parseErr != nil {
			return false
		}
		return parsed.Path == dashboardTestLandingPath
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardSessionTimeoutDisablesGoogleAutoSelect(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	evaluateScriptInto(t, page, fmt.Sprintf(dashboardDisableGoogleAutoSelectTrackingScript, dashboardDisableGoogleAutoSelectStorageKey), nil)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	evaluateScriptInto(t, page, dashboardForceLogoutScript, nil)
	waitNavigation()

	disableAutoSelectScript := fmt.Sprintf(`(function(){
		if (!window.localStorage) { return ''; }
		return window.localStorage.getItem(%q) || '';
	}())`, dashboardDisableGoogleAutoSelectStorageKey)
	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, disableAutoSelectScript) == "true"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardSessionTimeoutIgnoresSyntheticActivity(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	waitForVisibleElement(t, page, dashboardUserEmailSelector)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	evaluateScriptInto(t, page, dashboardDispatchSyntheticMousemoveScript, nil)

	require.Eventually(t, func() bool {
		notificationElement, elementErr := page.Element(dashboardNotificationSelector)
		if elementErr != nil {
			return false
		}
		visible, visibleErr := notificationElement.Visible()
		if visibleErr != nil {
			return false
		}
		return visible
	}, time.Second, 100*time.Millisecond)
}

func TestDashboardSessionTimeoutPromptPersistsAfterMousemove(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	waitForVisibleElement(t, page, dashboardUserEmailSelector)

	evaluateScriptInto(t, page, dashboardForcePromptScript, nil)
	waitForVisibleElement(t, page, dashboardNotificationSelector)

	require.NoError(t, page.Mouse.MoveTo(proto.Point{X: 200, Y: 200}))

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSessionTimeoutVisibleScript)
	}, time.Second, 100*time.Millisecond)
}

func TestDashboardRestoresThemeFromPublicPreference(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestLandingPath))
	waitNavigation()
	evaluateScriptInto(t, page, dashboardSeedPublicThemeScript, nil)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	userEmailVisibleScript := fmt.Sprintf(`(function(){
		var element = document.querySelector(%q);
		if (!element) { return false; }
		var style = window.getComputedStyle(element);
		if (!style) { return false; }
		return style.display !== 'none' && style.visibility !== 'hidden';
	}())`, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, userEmailVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	documentTheme := evaluateScriptString(t, page, dashboardDocumentThemeScript)
	require.Equal(t, "dark", documentTheme)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardDocumentMprThemeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	storedTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "dark", storedTheme)
}

func TestLandingRedirectsToDashboardWhenAuthenticated(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	_, err := page.EvalOnNewDocument(landingMarkAuthenticatedScript)
	require.NoError(t, err)

	navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)

	authScriptRendered := evaluateScriptBoolean(t, page, `(function(){
		return Array.from(document.scripts || []).some(function(script){
			return (script.textContent || '').indexOf('data-loopaware-auth-script') !== -1;
		});
	}())`)
	require.True(t, authScriptRendered)

	authScriptParseError := evaluateScriptString(t, page, `(function(){
		var script = Array.from(document.scripts || []).find(function(item){
			return (item.textContent || '').indexOf('data-loopaware-auth-script') !== -1;
		});
		if (!script) { return 'missing'; }
		try {
			new Function(script.textContent || '');
			return '';
		} catch (err) {
			if (err && err.message) {
				if (typeof err.lineNumber === 'number') {
					return err.message + ' at ' + err.lineNumber + ':' + err.columnNumber;
				}
				return err.message;
			}
			return String(err);
		}
	}())`)
	require.Equal(t, "", authScriptParseError)

	authScriptMarker := evaluateScriptString(t, page, `(function(){
		return document.documentElement ? (document.documentElement.getAttribute('data-loopaware-auth-script') || '') : '';
	}())`)
	require.Equal(t, "true", authScriptMarker)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, `(function(){
			var header = document.querySelector('mpr-header');
			if (!header) { return ''; }
			return header.getAttribute('data-user-display') || '';
		}())`) == "Test User"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	authBoundValue := evaluateScriptString(t, page, `(function(){
		var header = document.querySelector('mpr-header');
		if (!header) { return ''; }
		return header.getAttribute('data-loopaware-auth-bound') || '';
	}())`)
	require.Equal(t, "true", authBoundValue)

	require.Eventually(t, func() bool {
		info, infoErr := page.Info()
		if infoErr != nil {
			return false
		}
		parsed, parseErr := url.Parse(info.URL)
		if parseErr != nil {
			return false
		}
		return parsed.Path == dashboardTestDashboardRoute
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestLandingSigninWaitsForNonce(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	page := buildHeadlessPage(t)

	_, err := page.EvalOnNewDocument(fmt.Sprintf(landingDelayedNonceScript, landingGoogleNonceDelayMilliseconds))
	require.NoError(t, err)

	navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)

	waitForVisibleElement(t, page, landingGoogleSigninWrapperSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, landingSigninDisabledWhileNoncePendingScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Never(t, func() bool {
		if evaluateScriptBoolean(t, page, landingNonceResolvedScript) {
			return false
		}
		return !evaluateScriptBoolean(t, page, landingSigninDisabledScript)
	}, time.Duration(landingGoogleNonceDelayMilliseconds-500)*time.Millisecond, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, landingNonceResolvedScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return !evaluateScriptBoolean(t, page, landingSigninDisabledScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
}

func TestDashboardInheritsLandingDefaultTheme(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)
	evaluateScriptInto(t, page, dashboardClearThemeStorageScript, nil)
	navigateToPage(t, page, harness.baseURL+dashboardTestLandingPath)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, selectorExistsScript(landingThemeToggleControlSelector))
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Equal(t, "dark", evaluateScriptString(t, page, dashboardDocumentThemeScript))

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardStoredPublicThemeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardStoredLandingThemeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Equal(t, "dark", evaluateScriptString(t, page, dashboardDocumentThemeScript))

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardDocumentMprThemeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	storedTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "dark", storedTheme)
}

func TestDashboardPrefersPublicThemeOverStoredPreference(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	testCases := []struct {
		name          string
		storedTheme   string
		publicTheme   string
		expectedTheme string
	}{
		{
			name:          "public_dark_overrides_stored_light",
			storedTheme:   "light",
			publicTheme:   "dark",
			expectedTheme: "dark",
		},
		{
			name:          "public_light_overrides_stored_dark",
			storedTheme:   "dark",
			publicTheme:   "light",
			expectedTheme: "light",
		},
	}

	for _, testCase := range testCases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
			require.NoError(t, page.Navigate(harness.baseURL+dashboardTestLandingPath))
			waitNavigation()

			seedScript := fmt.Sprintf(`localStorage.setItem('loopaware_dashboard_theme','%s');localStorage.setItem('loopaware_public_theme','%s');localStorage.setItem('loopaware_landing_theme','%s');localStorage.removeItem('loopaware_theme');`, testCase.storedTheme, testCase.publicTheme, testCase.publicTheme)
			evaluateScriptInto(t, page, seedScript, nil)

			waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
			require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
			waitNavigation()
			require.Eventually(t, func() bool {
				return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

			documentTheme := evaluateScriptString(t, page, dashboardDocumentThemeScript)
			require.Equal(t, testCase.expectedTheme, documentTheme)

			storedDashboardTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
			require.Equal(t, testCase.expectedTheme, storedDashboardTheme)

			require.Eventually(t, func() bool {
				return evaluateScriptString(t, page, dashboardDocumentMprThemeScript) == testCase.expectedTheme
			}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
		})
	}
}

func TestDashboardTrafficPlaceholderWithoutVisits(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Traffic Placeholder",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardTrafficStatusHiddenScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	placeholder := evaluateScriptString(t, page, dashboardTopPagesPlaceholderScript)
	require.Equal(t, dashboardTopPagesPlaceholderText, placeholder)

	var visitBadges struct {
		TotalVisible  bool   `json:"totalVisible"`
		TotalText     string `json:"totalText"`
		UniqueVisible bool   `json:"uniqueVisible"`
		UniqueText    string `json:"uniqueText"`
	}
	evaluateScriptInto(t, page, dashboardVisitCountsScript, &visitBadges)
	require.False(t, visitBadges.TotalVisible)
	require.Empty(t, visitBadges.TotalText)
	require.False(t, visitBadges.UniqueVisible)
	require.Empty(t, visitBadges.UniqueText)
}

func TestDashboardTrafficRendersTopPages(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Traffic Data",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	visitorA := storage.NewID()
	visitorB := storage.NewID()
	visits := []model.SiteVisitInput{
		{SiteID: site.ID, URL: harness.baseURL + "/home", VisitorID: visitorA},
		{SiteID: site.ID, URL: harness.baseURL + "/home", VisitorID: visitorB},
		{SiteID: site.ID, URL: harness.baseURL + "/about", VisitorID: visitorA},
	}
	for _, input := range visits {
		visit, err := model.NewSiteVisit(input)
		require.NoError(t, err)
		require.NoError(t, harness.database.Create(&visit).Error)
	}

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		var rows []struct {
			Path  string `json:"path"`
			Count string `json:"count"`
		}
		evaluateScriptInto(t, page, dashboardTopPagesRowsScript, &rows)
		if len(rows) != 2 {
			return false
		}
		return rows[0].Path == "/home" && rows[0].Count == "2" && rows[1].Path == "/about" && rows[1].Count == "1"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	var visitBadges struct {
		TotalVisible  bool   `json:"totalVisible"`
		TotalText     string `json:"totalText"`
		UniqueVisible bool   `json:"uniqueVisible"`
		UniqueText    string `json:"uniqueText"`
	}
	evaluateScriptInto(t, page, dashboardVisitCountsScript, &visitBadges)
	require.True(t, visitBadges.TotalVisible)
	require.Equal(t, "3 visits", visitBadges.TotalText)
	require.True(t, visitBadges.UniqueVisible)
	require.Equal(t, "2 unique", visitBadges.UniqueText)
	require.True(t, evaluateScriptBoolean(t, page, dashboardTrafficStatusHiddenScript))
}

func TestWidgetTestPageUsesDashboardChrome(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Widget Test Chrome",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)

	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestLandingPath))
	waitNavigation()
	evaluateScriptInto(t, page, dashboardSeedPublicThemeScript, nil)

	widgetTestURL := fmt.Sprintf("%s/app/sites/%s/widget-test", harness.baseURL, site.ID)
	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(widgetTestURL))
	waitNavigation()

	currentWidgetTestPath := evaluateScriptString(t, page, dashboardLocationPathScript)
	require.Equal(t, fmt.Sprintf("/app/sites/%s/widget-test", site.ID), currentWidgetTestPath)

	headerVisibleScript := fmt.Sprintf(`(function(){
    var element = document.querySelector(%q);
    if (!element) { return false; }
    var style = window.getComputedStyle(element);
    if (!style) { return false; }
    return style.display !== 'none' && style.visibility !== 'hidden';
}())`, dashboardUserMenuTriggerSelector)
	footerVisibleScript := fmt.Sprintf(`(function(){
    var element = document.querySelector(%q);
    if (!element) { return false; }
    var style = window.getComputedStyle(element);
    if (!style) { return false; }
    return style.display !== 'none' && style.visibility !== 'hidden';
}())`, dashboardFooterSelector)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, headerVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, footerVisibleScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	initialTheme := evaluateScriptString(t, page, dashboardDocumentThemeScript)
	require.Equal(t, "dark", initialTheme)

	initialDashboardStoredTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "dark", initialDashboardStoredTheme)

	initialPublicStoredTheme := evaluateScriptString(t, page, dashboardStoredPublicThemeScript)
	require.Equal(t, "dark", initialPublicStoredTheme)

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
    var body = document.body;
    if (!body) { return false; }
    return body.classList.contains('bg-dark') && body.classList.contains('text-light');
  }())`))

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardDocumentMprThemeScript) == "dark"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	evaluateScriptInto(t, page, `(function(){
		var footer = document.getElementById('dashboard-footer');
		if (!footer || typeof footer.dispatchEvent !== 'function') { return false; }
		footer.dispatchEvent(new CustomEvent('mpr-footer:theme-change', { detail: { theme: 'light' } }));
		return true;
	}())`, nil)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardDocumentThemeScript) == "light"
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	finalDashboardStoredTheme := evaluateScriptString(t, page, dashboardStoredDashboardThemeScript)
	require.Equal(t, "light", finalDashboardStoredTheme)

	finalPublicStoredTheme := evaluateScriptString(t, page, dashboardStoredPublicThemeScript)
	require.Equal(t, "light", finalPublicStoredTheme)

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
    var body = document.body;
    if (!body) { return false; }
    return body.classList.contains('bg-light') && body.classList.contains('text-dark');
  }())`))
}

func TestTestPagesUseUserMenu(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Test Pages Profile Menu",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	_, evalErr := page.EvalOnNewDocument(landingMarkAuthenticatedScript)
	require.NoError(t, evalErr)

	pagePaths := []string{
		fmt.Sprintf("/app/sites/%s/widget-test", site.ID),
		fmt.Sprintf("/app/sites/%s/subscribe-test", site.ID),
		fmt.Sprintf("/app/sites/%s/traffic-test", site.ID),
	}

	for _, path := range pagePaths {
		navigateToPage(t, page, harness.baseURL+path)

		require.Eventually(t, func() bool {
			return evaluateScriptString(t, page, `(function(){
				var header = document.querySelector('mpr-header');
				if (!header) { return ''; }
				return header.getAttribute('data-loopaware-auth-bound') || '';
			}())`) == "true"
		}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

		var headerState struct {
			HasUserMenu          bool `json:"hasUserMenu"`
			HasLegacyProfileMenu bool `json:"hasLegacyProfileMenu"`
		}
		require.Eventually(t, func() bool {
			evaluateScriptInto(t, page, dashboardHeaderDefaultProfileStateScript, &headerState)
			return headerState.HasUserMenu && !headerState.HasLegacyProfileMenu
		}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

		openDashboardUserMenu(t, page)
		openDashboardUserMenu(t, page)

		accountSettingsLabel := evaluateScriptString(t, page, fmt.Sprintf(`(function(){
			var element = document.querySelector(%q);
			if (!element) { return ''; }
			return String(element.textContent || '').trim();
		}())`, dashboardUserMenuAccountSettingsSelector))
		require.Equal(t, "Account Settings", accountSettingsLabel)

		logoutLabel := evaluateScriptString(t, page, fmt.Sprintf(`(function(){
			var element = document.querySelector(%q);
			if (!element) { return ''; }
			return String(element.textContent || '').trim();
		}())`, dashboardUserMenuLogoutButtonSelector))
		require.Equal(t, "Logout", logoutLabel)
	}
}

func TestWidgetTestFeedbackSubmissionSucceeds(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Test Feedback",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 24,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	widgetTestURL := fmt.Sprintf("%s/app/sites/%s/widget-test", harness.baseURL, site.ID)
	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(widgetTestURL))
	waitNavigation()

	waitForVisibleElement(t, page, widgetBubbleSelector)
	clickSelector(t, page, widgetBubbleSelector)
	waitForVisibleElement(t, page, widgetPanelSelector)

	testContact := "tester@example.com"
	testMessage := "Widget test feedback message"

	setInputValue(t, page, widgetContactSelector, testContact)
	setInputValue(t, page, widgetMessageSelector, testMessage)
	clickSelector(t, page, widgetSendButtonSelector)

	var statusText string
	require.Eventually(t, func() bool {
		statusText = evaluateScriptString(t, page, widgetStatusResolveScript)
		return strings.Contains(statusText, widgetSuccessStatusMessage)
	}, 5*time.Second, 100*time.Millisecond)

	var storedFeedback model.Feedback
	require.Eventually(t, func() bool {
		return harness.database.Where("site_id = ? AND contact = ?", site.ID, testContact).Order("created_at desc").First(&storedFeedback).Error == nil
	}, 5*time.Second, 100*time.Millisecond)
	require.Equal(t, testMessage, storedFeedback.Message)
}

func TestWidgetTestPlacementSavePersists(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Test Placement",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 24,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	widgetTestURL := fmt.Sprintf("%s/app/sites/%s/widget-test", harness.baseURL, site.ID)
	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(widgetTestURL))
	waitNavigation()

	interceptFetchRequests(t, page)
	clickSelector(t, page, "#widget-test-side-left")
	setInputValue(t, page, "#widget-test-bottom-offset", "72")
	endpointValue := evaluateScriptString(t, page, `(function(){
  var body = document.body;
  return body ? body.getAttribute('data-update-endpoint') || '' : '';
}())`)
	require.NotEmpty(t, endpointValue)
	t.Logf("widget test update endpoint: %s", endpointValue)
	clickSelector(t, page, "#widget-test-save")

	type siteUpdatePayload struct {
		WidgetBubbleSide         string `json:"widget_bubble_side"`
		WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	}

	var payload siteUpdatePayload
	var payloadStatus int
	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		t.Logf("captured widget test placement requests: %#v", requests)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if record.Status == 0 {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			payloadStatus = record.Status
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)

	require.Equal(t, "left", payload.WidgetBubbleSide)
	require.Equal(t, 72, payload.WidgetBubbleBottomOffset)
	require.Equal(t, http.StatusOK, payloadStatus)

	var updatedSite model.Site
	require.Eventually(t, func() bool {
		return harness.database.First(&updatedSite, "id = ?", site.ID).Error == nil
	}, 20*time.Second, 100*time.Millisecond)
	require.Equal(t, "left", updatedSite.WidgetBubbleSide)
	require.Equal(t, 72, updatedSite.WidgetBubbleBottomOffsetPx)
}

func TestSubscribeWidgetTestFlowSubmitsSubscription(t *testing.T) {
	subscriptionNotifier := &recordingSubscriptionNotifier{testingT: t}
	emailSender := &recordingEmailSender{testingT: t}
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail, withSubscriptionNotifier(subscriptionNotifier), withEmailSender(emailSender))
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Test Flow",
		AllowedOrigin: "https://widget.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
        var originalOpen = window.open;
        window.open = function(url, target, features) {
          if (url) {
            window.location.assign(url);
          }
          return window;
        };
        window.open._original = originalOpen;
        return true;
      }())`))

	waitForVisibleElement(t, page, dashboardSectionTabSubscriptionsSelector)
	clickSelector(t, page, dashboardSectionTabSubscriptionsSelector)
	waitForVisibleElement(t, page, dashboardSubscribeTestButtonSelector)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	clickSelector(t, page, dashboardSubscribeTestButtonSelector)
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardLocationPathScript) == fmt.Sprintf("/app/sites/%s/subscribe-test", site.ID)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	waitForVisibleElement(t, page, subscribeTestFormContainerSelector)
	waitForVisibleElement(t, page, "#"+subscribeEmailInputID)
	waitForVisibleElement(t, page, "#"+subscribeSubmitButtonID)

	testEmail := "preview-subscriber@example.com"
	setInputValue(t, page, "#"+subscribeEmailInputID, testEmail)
	setInputValue(t, page, "#"+subscribeNameInputID, "Preview User")
	clickSelector(t, page, "#"+subscribeSubmitButtonID)

	var inlineStatus string
	require.Eventually(t, func() bool {
		inlineStatus = evaluateScriptString(t, page, fmt.Sprintf(`(function(){
        var status = document.getElementById(%q);
        if (!status) { return ""; }
        return status.innerText || "";
      }())`, subscribeStatusElementID))
		return strings.Contains(inlineStatus, "You're on the list")
	}, integrationStatusWaitTimeout, integrationStatusPollInterval)

	require.Eventually(t, func() bool {
		var stored model.Subscriber
		return harness.database.
			Where("site_id = ? AND email = ?", site.ID, testEmail).
			Order("created_at desc").
			First(&stored).Error == nil
	}, 5*time.Second, 100*time.Millisecond)

	require.Equal(t, 1, emailSender.CallCount())
	require.Equal(t, 0, subscriptionNotifier.CallCount())

}

func TestSubscribeTestPageSupportsTargetInput(testingT *testing.T) {
	harness := buildDashboardIntegrationHarness(testingT, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Test Target Input",
		AllowedOrigin: "https://widget.example",
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(testingT, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(testingT, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(testingT)
	setPageCookie(testingT, page, harness.baseURL, sessionCookie)

	navigateToPage(testingT, page, harness.baseURL+fmt.Sprintf("/app/sites/%s/subscribe-test", site.ID))

	waitForVisibleElement(testingT, page, subscribeTestTargetInputSelector)
	waitForVisibleElement(testingT, page, subscribeTestPreviewContainerSelector)

	var initialState struct {
		TargetValue string `json:"targetValue"`
		PreviewID   string `json:"previewId"`
	}
	initialStateScript := fmt.Sprintf(`(function(){
        var input = document.querySelector(%q);
        var preview = document.querySelector(%q);
        return { targetValue: input ? input.value || '' : '', previewId: preview ? preview.id || '' : '' };
      }())`, subscribeTestTargetInputSelector, subscribeTestPreviewContainerSelector)
	evaluateScriptInto(testingT, page, initialStateScript, &initialState)
	require.Equal(testingT, subscribeTestInlineContainerID, initialState.TargetValue)
	require.Equal(testingT, subscribeTestInlineContainerID, initialState.PreviewID)

	targetID := "subscribe-test-target-demo"
	setInputValue(testingT, page, subscribeTestTargetInputSelector, targetID)

	targetPreviewScript := fmt.Sprintf(`(function(){
        var preview = document.querySelector(%q);
        if (!preview) { return false; }
        if (preview.id !== %q) { return false; }
        return !!preview.querySelector(%q);
      }())`, subscribeTestPreviewContainerSelector, targetID, "#"+subscribeEmailInputID)
	require.Eventually(testingT, func() bool {
		return evaluateScriptBoolean(testingT, page, targetPreviewScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	targetValueScript := fmt.Sprintf(`(function(){
        var input = document.querySelector(%q);
        if (!input) { return ''; }
        return input.value || '';
      }())`, subscribeTestTargetInputSelector)
	require.Equal(testingT, targetID, evaluateScriptString(testingT, page, targetValueScript))
}

func TestSubscribeTestPageExposesEventsEndpoint(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Test Events Endpoint",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/app/sites/%s/subscribe-test", harness.baseURL, site.ID), nil)
	require.NoError(t, err)
	request.AddCookie(sessionCookie)

	response, err := harness.server.Client().Do(request)
	require.NoError(t, err)
	defer response.Body.Close()

	require.Equal(t, http.StatusOK, response.StatusCode)
	body, err := io.ReadAll(response.Body)
	require.NoError(t, err)

	expectedEndpoint := fmt.Sprintf(`data-events-endpoint="/app/sites/%s/subscribe-test/events"`, site.ID)
	require.Contains(t, string(body), expectedEndpoint)
}

func TestSubscribeTestEventsEndpointStreamsBroadcasts(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Subscribe Test Event Stream",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	request, err := http.NewRequest(http.MethodGet, fmt.Sprintf("%s/app/sites/%s/subscribe-test/events", harness.baseURL, site.ID), nil)
	require.NoError(t, err)
	request.AddCookie(sessionCookie)

	client := harness.server.Client()
	client.Timeout = 5 * time.Second

	response, err := client.Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)

	reader := bufio.NewReader(response.Body)

	expectedEvent := httpapi.SubscriptionTestEvent{
		SiteID:       site.ID,
		SubscriberID: "subscriber-123",
		Email:        "subscriber@example.com",
		EventType:    "submission",
		Status:       "received",
	}

	lineChannel := make(chan string, 1)
	errorChannel := make(chan error, 1)
	go func() {
		defer close(lineChannel)
		defer close(errorChannel)
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			errorChannel <- readErr
			return
		}
		lineChannel <- line
	}()

	harness.subscriptionEvents.Broadcast(expectedEvent)

	var line string
	select {
	case line = <-lineChannel:
	case readErr := <-errorChannel:
		require.NoError(t, readErr)
	case <-time.After(3 * time.Second):
		require.FailNow(t, "timed out waiting for subscription test event")
	}

	require.NoError(t, response.Body.Close())

	payloadText := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	require.NotEmpty(t, payloadText)

	var receivedEvent httpapi.SubscriptionTestEvent
	require.NoError(t, json.Unmarshal([]byte(payloadText), &receivedEvent))

	require.Equal(t, expectedEvent.SiteID, receivedEvent.SiteID)
	require.Equal(t, expectedEvent.SubscriberID, receivedEvent.SubscriberID)
	require.Equal(t, expectedEvent.Email, receivedEvent.Email)
	require.Equal(t, expectedEvent.EventType, receivedEvent.EventType)
	require.Equal(t, expectedEvent.Status, receivedEvent.Status)
	require.False(t, receivedEvent.Timestamp.IsZero())
}
func TestTrafficWidgetTestFlowRecordsVisit(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:            storage.NewID(),
		Name:          "Traffic Test Flow",
		AllowedOrigin: harness.baseURL,
		OwnerEmail:    dashboardTestAdminEmail,
		CreatorEmail:  dashboardTestAdminEmail,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)
	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestDashboardRoute))
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.True(t, evaluateScriptBoolean(t, page, `(function(){
        var originalOpen = window.open;
        window.open = function(url){
          if (url) { window.location.assign(url); }
          return window;
        };
        window.open._original = originalOpen;
        return true;
      }())`))

	waitForVisibleElement(t, page, dashboardSectionTabTrafficSelector)
	clickSelector(t, page, dashboardSectionTabTrafficSelector)
	waitForVisibleElement(t, page, dashboardTrafficTestButtonSelector)

	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	clickSelector(t, page, dashboardTrafficTestButtonSelector)
	waitNavigation()

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardLocationPathScript) == fmt.Sprintf("/app/sites/%s/traffic-test", site.ID)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	testURL := harness.baseURL + "/preview-traffic"
	waitForVisibleElement(t, page, trafficTestURLInputSelector)
	setInputValue(t, page, trafficTestURLInputSelector, testURL)
	currentURLValue := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).value || ""`, trafficTestURLInputSelector))
	require.Equal(t, testURL, strings.TrimSpace(currentURLValue))
	clickSelector(t, page, trafficTestSendButtonSelector)

	require.Eventually(t, func() bool {
		var count int64
		if err := harness.database.Model(&model.SiteVisit{}).Where("site_id = ?", site.ID).Count(&count).Error; err != nil {
			return false
		}
		return count > 0
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		totalText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestTotalSelector))
		return strings.TrimSpace(totalText) != "" && strings.TrimSpace(totalText) != "0"
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		uniqueText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestUniqueSelector))
		return strings.TrimSpace(uniqueText) == "1"
	}, 5*time.Second, 100*time.Millisecond)

	clickSelector(t, page, trafficTestSendButtonSelector)

	require.Eventually(t, func() bool {
		totalText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestTotalSelector))
		return strings.TrimSpace(totalText) == "2"
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		uniqueText := evaluateScriptString(t, page, fmt.Sprintf(`document.querySelector(%q).textContent || ""`, trafficTestUniqueSelector))
		return strings.TrimSpace(uniqueText) == "1"
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		logText := evaluateScriptString(t, page, `document.getElementById("traffic-test-log").textContent || ""`)
		return strings.Contains(logText, "Country: Local network") && strings.Contains(logText, "IP:")
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDashboardFeedbackStreamRefreshesMessages(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Stream Feedback Site",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	waitForVisibleElement(t, page, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardNoMessagesPlaceholderScript)
	}, 5*time.Second, 100*time.Millisecond)

	requestBody := fmt.Sprintf(`{"site_id":"%s","contact":"auto@example.com","message":"Auto refresh message"}`, site.ID)
	request, err := http.NewRequest(http.MethodPost, harness.baseURL+"/api/feedback", bytes.NewBufferString(requestBody))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", site.AllowedOrigin)
	response, err := harness.server.Client().Do(request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	require.NoError(t, response.Body.Close())

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardFeedbackRenderedScript)
	}, 10*time.Second, 200*time.Millisecond)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardFeedbackCountScript)
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDashboardWidgetPlacementSavePersists(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Placement Save",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	interceptFetchRequests(t, page)
	clickSelector(t, page, "#widget-placement-side-left")
	setInputValue(t, page, dashboardWidgetBottomOffsetInputSelector, "88")

	type siteUpdatePayload struct {
		WidgetBubbleSide         string `json:"widget_bubble_side"`
		WidgetBubbleBottomOffset int    `json:"widget_bubble_bottom_offset"`
	}

	var payload siteUpdatePayload
	var payloadStatus int
	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if record.Status == 0 {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			if payload.WidgetBubbleSide != "left" || payload.WidgetBubbleBottomOffset != 88 {
				continue
			}
			payloadStatus = record.Status
			return true
		}
		return false
	}, 20*time.Second, 100*time.Millisecond)

	require.Equal(t, "left", payload.WidgetBubbleSide)
	require.Equal(t, 88, payload.WidgetBubbleBottomOffset)
	require.Equal(t, http.StatusOK, payloadStatus)

	var updatedSite model.Site
	require.Eventually(t, func() bool {
		return harness.database.First(&updatedSite, "id = ?", site.ID).Error == nil
	}, 5*time.Second, 100*time.Millisecond)
	require.Equal(t, "left", updatedSite.WidgetBubbleSide)
	require.Equal(t, 88, updatedSite.WidgetBubbleBottomOffsetPx)
}

func TestDashboardAllowedOriginsAcceptsMultipleEntries(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Multi-Origin Save",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	interceptFetchRequests(t, page)

	allowedOrigins := "https://widget.example " + harness.baseURL
	setInputValue(t, page, dashboardEditSiteAllowedOriginsSelector, allowedOrigins)

	type siteUpdatePayload struct {
		AllowedOrigin string `json:"allowed_origin"`
	}

	var payload siteUpdatePayload
	var payloadStatus int
	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if record.Status == 0 {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			if payload.AllowedOrigin != allowedOrigins {
				continue
			}
			payloadStatus = record.Status
			return true
		}
		return false
	}, 20*time.Second, 100*time.Millisecond)

	require.Equal(t, allowedOrigins, payload.AllowedOrigin)
	require.Equal(t, http.StatusOK, payloadStatus)

	var updatedSite model.Site
	require.Eventually(t, func() bool {
		return harness.database.First(&updatedSite, "id = ?", site.ID).Error == nil
	}, 5*time.Second, 100*time.Millisecond)
	require.Equal(t, allowedOrigins, updatedSite.AllowedOrigin)

	require.Eventually(t, func() bool {
		return evaluateScriptString(t, page, dashboardFirstSiteOriginTextScript) == "https://widget.example +1 more"
	}, 5*time.Second, 100*time.Millisecond)
}

func TestDashboardAutosavePreservesSubscribeOriginTyping(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Autosave Subscribe Origins",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	clickSelector(t, page, dashboardSectionTabSubscriptionsSelector)
	waitForVisibleElement(t, page, dashboardSubscribeAllowedOriginsInputSelector)

	interceptFetchRequests(t, page)

	input := waitForVisibleElement(t, page, dashboardSubscribeAllowedOriginsInputSelector)
	require.NoError(t, input.Focus())
	require.NoError(t, input.SelectAllText())
	require.NoError(t, input.Input("https://autosave.example"))

	type siteUpdatePayload struct {
		SubscribeAllowedOrigins string `json:"subscribe_allowed_origins"`
	}

	var payload siteUpdatePayload
	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if record.Status == 0 {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			return payload.SubscribeAllowedOrigins == "https://autosave.example"
		}
		return false
	}, 20*time.Second, 100*time.Millisecond)

	type inputState struct {
		Focused bool   `json:"focused"`
		Value   string `json:"value"`
	}

	var state inputState
	focusScript := fmt.Sprintf(`(function() {
		var input = document.querySelector(%q);
		if (!input) { return { focused: false, value: '' }; }
		return { focused: document.activeElement === input, value: input.value || '' };
	}())`, dashboardSubscribeAllowedOriginsInputSelector)
	evaluateScriptInto(t, page, focusScript, &state)

	require.True(t, state.Focused)
	require.Equal(t, "https://autosave.example", state.Value)
}

func TestDashboardSiteFaviconOpensOrigin(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Favicon Origin Site",
		AllowedOrigin:              "https://open-target.example",
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 16,
		FaviconData: []byte{
			0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
			0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
			0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
			0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
			0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
			0x54, 0x78, 0x9C, 0x63, 0xF8, 0xCF, 0xC0, 0x00,
			0x00, 0x04, 0xBF, 0x01, 0xFE, 0xA7, 0x65, 0x81,
			0xF0, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E,
			0x44, 0xAE, 0x42, 0x60, 0x82,
		},
		FaviconContentType: "image/png",
		FaviconFetchedAt:   time.Now().UTC(),
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	waitForVisibleElement(t, page, dashboardUserEmailSelector)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSiteFaviconVisibleScript)
	}, 5*time.Second, 100*time.Millisecond)

	evaluateScriptInto(t, page, dashboardCaptureWindowOpenScript, nil)

	clickSelector(t, page, dashboardSiteFaviconSelector)

	type windowOpenRecord struct {
		URL      string `json:"url"`
		Target   string `json:"target"`
		Features string `json:"features"`
	}

	var windowOpenCalls []windowOpenRecord
	evaluateScriptInto(t, page, dashboardOpenedWindowCallsScript, &windowOpenCalls)

	require.Len(t, windowOpenCalls, 1)
	require.Equal(t, site.AllowedOrigin, windowOpenCalls[0].URL)
	require.Equal(t, "_blank", windowOpenCalls[0].Target)
	require.Equal(t, "noopener,noreferrer", windowOpenCalls[0].Features)
}

func TestDashboardWidgetBottomOffsetStepButtonsAdjustAndPersist(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Offset Step Site",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 24,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	navigateToPage(t, page, harness.baseURL+dashboardTestDashboardRoute)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)
	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	waitForVisibleElement(t, page, dashboardWidgetBottomOffsetInputSelector)

	initialOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, site.WidgetBubbleBottomOffsetPx, initialOffset)

	interceptFetchRequests(t, page)

	clickSelector(t, page, dashboardWidgetBottomOffsetIncreaseSelector)
	incrementedOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, initialOffset+widgetBottomOffsetStepPixels, incrementedOffset)

	manualOffset := 37
	setInputValue(t, page, dashboardWidgetBottomOffsetInputSelector, strconv.Itoa(manualOffset))
	currentManual := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, manualOffset, currentManual)

	offsetInput := waitForVisibleElement(t, page, dashboardWidgetBottomOffsetInputSelector)
	require.NoError(t, offsetInput.Focus())

	require.NoError(t, page.Keyboard.Press(input.ArrowUp))
	upperValue := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, manualOffset+widgetBottomOffsetStepPixels, upperValue)

	require.NoError(t, page.Keyboard.Press(input.ArrowDown))
	finalOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, manualOffset, finalOffset)

	type siteUpdatePayload struct {
		WidgetBubbleBottomOffset int `json:"widget_bubble_bottom_offset"`
	}
	var payload siteUpdatePayload
	payloadStatus := 0

	require.Eventually(t, func() bool {
		requests := readCapturedFetchRequests(t, page)
		for _, record := range requests {
			if !strings.HasSuffix(record.URL, "/api/sites/"+site.ID) {
				continue
			}
			if !strings.EqualFold(record.Method, http.MethodPatch) {
				continue
			}
			if record.Body == "" {
				continue
			}
			if err := json.Unmarshal([]byte(record.Body), &payload); err != nil {
				return false
			}
			if record.Status == 0 {
				return false
			}
			if payload.WidgetBubbleBottomOffset != finalOffset {
				continue
			}
			payloadStatus = record.Status
			return true
		}
		return false
	}, 5*time.Second, 100*time.Millisecond)

	require.Equal(t, finalOffset, payload.WidgetBubbleBottomOffset)
	require.Equal(t, http.StatusOK, payloadStatus)
}

func TestWidgetTestBottomOffsetControlsAdjustPreviewAndPersist(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	site := model.Site{
		ID:                         storage.NewID(),
		Name:                       "Widget Offset Controls",
		AllowedOrigin:              harness.baseURL,
		OwnerEmail:                 dashboardTestAdminEmail,
		CreatorEmail:               dashboardTestAdminEmail,
		WidgetBubbleSide:           "right",
		WidgetBubbleBottomOffsetPx: 28,
	}
	require.NoError(t, harness.database.Create(&site).Error)

	sessionCookie := createAuthenticatedSessionCookie(t, dashboardTestAdminEmail, dashboardTestAdminDisplayName)

	page := buildHeadlessPage(t)
	setPageCookie(t, page, harness.baseURL, sessionCookie)

	waitNavigation := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(harness.baseURL+dashboardTestLandingPath))
	waitNavigation()
	evaluateScriptInto(t, page, dashboardSeedPublicThemeScript, nil)

	widgetTestURL := fmt.Sprintf("%s/app/sites/%s/widget-test", harness.baseURL, site.ID)
	waitNavigation = page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	require.NoError(t, page.Navigate(widgetTestURL))
	waitNavigation()

	evaluateScriptInto(t, page, widgetTestEnsurePreviewElementsScript, nil)
	waitForVisibleElement(t, page, widgetTestBottomOffsetInputSelector)

	require.Eventually(t, func() bool {
		return strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript)) != ""
	}, 5*time.Second, 100*time.Millisecond)
	require.Eventually(t, func() bool {
		return strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript)) != ""
	}, 5*time.Second, 100*time.Millisecond)

	initialOffset := readWidgetTestBottomOffset(t, page)
	require.Equal(t, site.WidgetBubbleBottomOffsetPx, initialOffset)

	initialSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, initialOffset, initialSummary)

	initialBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", initialOffset), initialBubbleBottom)

	initialPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", initialOffset+widgetPanelVerticalSpacingPixels), initialPanelBottom)

	clickSelector(t, page, widgetTestOffsetIncreaseSelector)

	stepOffset := readWidgetTestBottomOffset(t, page)
	require.Equal(t, initialOffset+widgetBottomOffsetStepPixels, stepOffset)

	stepSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, stepOffset, stepSummary)

	stepBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", stepOffset), stepBubbleBottom)

	stepPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", stepOffset+widgetPanelVerticalSpacingPixels), stepPanelBottom)

	manualOffset := 31
	setInputValue(t, page, widgetTestBottomOffsetInputSelector, strconv.Itoa(manualOffset))

	currentManual := readWidgetTestBottomOffset(t, page)
	require.Equal(t, manualOffset, currentManual)

	manualSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, manualOffset, manualSummary)

	manualBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", manualOffset), manualBubbleBottom)

	manualPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", manualOffset+widgetPanelVerticalSpacingPixels), manualPanelBottom)

	offsetInput := waitForVisibleElement(t, page, widgetTestBottomOffsetInputSelector)
	require.NoError(t, offsetInput.Focus())

	require.NoError(t, page.Keyboard.Press(input.ArrowDown))

	finalOffset := readWidgetTestBottomOffset(t, page)
	require.Equal(t, manualOffset-widgetBottomOffsetStepPixels, finalOffset)

	finalSummary := parseOffsetValue(t, evaluateScriptString(t, page, widgetTestReadSummaryOffsetScript))
	require.Equal(t, finalOffset, finalSummary)

	finalBubbleBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestBubbleBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", finalOffset), finalBubbleBottom)

	finalPanelBottom := strings.TrimSpace(evaluateScriptString(t, page, widgetTestPanelBottomScript))
	require.Equal(t, fmt.Sprintf("%dpx", finalOffset+widgetPanelVerticalSpacingPixels), finalPanelBottom)

	redirectWait := page.WaitNavigation(proto.PageLifecycleEventNameLoad)
	clickSelector(t, page, widgetTestSaveButtonSelector)
	redirectWait()

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardIdleHooksReadyScript)
	}, dashboardPromptWaitTimeout, dashboardPromptPollInterval)

	currentPath := evaluateScriptString(t, page, dashboardLocationPathScript)
	require.Equal(t, dashboardTestDashboardRoute, currentPath)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardSelectFirstSiteScript)
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		return evaluateScriptBoolean(t, page, dashboardWidgetBottomOffsetPresenceScript)
	}, 5*time.Second, 100*time.Millisecond)

	dashboardOffset := readDashboardWidgetBottomOffset(t, page)
	require.Equal(t, finalOffset, dashboardOffset)
}

func TestExampleRouteIsUnavailable(t *testing.T) {
	harness := buildDashboardIntegrationHarness(t, dashboardTestAdminEmail)
	defer harness.Close()

	response, err := http.Get(harness.baseURL + "/example")
	require.NoError(t, err)
	defer response.Body.Close()
	require.Equal(t, http.StatusNotFound, response.StatusCode)
}

func readDashboardWidgetBottomOffset(testingT *testing.T, page *rod.Page) int {
	testingT.Helper()
	value := evaluateScriptString(testingT, page, dashboardReadWidgetBottomOffsetScript)
	return parseOffsetValue(testingT, value)
}

func readWidgetTestBottomOffset(testingT *testing.T, page *rod.Page) int {
	testingT.Helper()
	value := evaluateScriptString(testingT, page, widgetTestReadOffsetInputScript)
	return parseOffsetValue(testingT, value)
}

func parseOffsetValue(testingT *testing.T, value string) int {
	testingT.Helper()
	trimmed := strings.TrimSpace(value)
	require.NotEmpty(testingT, trimmed)
	parsed, err := strconv.Atoi(trimmed)
	require.NoError(testingT, err)
	return parsed
}

func buildDashboardIntegrationHarness(testingT *testing.T, adminEmail string, opts ...dashboardHarnessOption) *dashboardIntegrationHarness {
	testingT.Helper()

	gin.SetMode(gin.TestMode)
	logger := zap.NewNop()

	config := dashboardHarnessOptions{
		subscriptionNotifier: stubDashboardNotifier{},
		emailSender:          nil,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&config)
		}
	}

	sqliteDatabase := testutil.NewSQLiteTestDatabase(testingT)
	gormDatabase, openErr := storage.OpenDatabase(sqliteDatabase.Configuration())
	require.NoError(testingT, openErr)
	gormDatabase = testutil.ConfigureDatabaseLogger(testingT, gormDatabase)
	require.NoError(testingT, storage.AutoMigrate(gormDatabase))
	sqlDatabase, sqlErr := gormDatabase.DB()
	require.NoError(testingT, sqlErr)

	adminEmails := []string{adminEmail}
	authManager, authErr := httpapi.NewAuthManager(gormDatabase, logger, adminEmails, nil, dashboardTestLandingPath, httpapi.AuthConfig{
		SigningKey: dashboardTestTauthSigningKey,
		TenantID:   dashboardTestTauthTenantID,
	})
	require.NoError(testingT, authErr)
	authClientConfig := httpapi.NewAuthClientConfig(dashboardTestGoogleClientID, "", dashboardTestTauthTenantID)

	noopResolver := &staticFaviconResolver{}
	faviconService := favicon.NewService(noopResolver)
	faviconManager := httpapi.NewSiteFaviconManager(gormDatabase, faviconService, logger)
	managerContext, managerCancel := context.WithCancel(context.Background())
	faviconManager.Start(managerContext)

	feedbackBroadcaster := httpapi.NewFeedbackEventBroadcaster()
	subscriptionEvents := httpapi.NewSubscriptionTestEventBroadcaster()

	statsProvider := httpapi.NewDatabaseSiteStatisticsProvider(gormDatabase)
	siteHandlers := httpapi.NewSiteHandlers(gormDatabase, logger, dashboardTestWidgetBaseURL, faviconManager, statsProvider, feedbackBroadcaster)
	landingHandlers := httpapi.NewLandingPageHandlers(logger, authManager, authClientConfig)
	privacyHandlers := httpapi.NewPrivacyPageHandlers(authManager, authClientConfig)
	sitemapHandlers := httpapi.NewSitemapHandlers(dashboardTestWidgetBaseURL)
	dashboardHandlers := httpapi.NewDashboardWebHandlers(logger, dashboardTestLandingPath, authClientConfig)
	publicHandlers := httpapi.NewPublicHandlers(gormDatabase, logger, feedbackBroadcaster, subscriptionEvents, stubDashboardNotifier{}, config.subscriptionNotifier, true, dashboardTestWidgetBaseURL, "unit-test-session-secret", config.emailSender, authClientConfig)
	widgetTestHandlers := httpapi.NewSiteWidgetTestHandlers(gormDatabase, logger, dashboardTestWidgetBaseURL, feedbackBroadcaster, stubDashboardNotifier{}, authClientConfig)
	trafficTestHandlers := httpapi.NewSiteTrafficTestHandlers(gormDatabase, logger, authClientConfig)
	subscribeTestHandlers := httpapi.NewSiteSubscribeTestHandlers(gormDatabase, logger, subscriptionEvents, config.subscriptionNotifier, true, dashboardTestWidgetBaseURL, "unit-test-session-secret", config.emailSender, authClientConfig)

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(httpapi.RequestLogger(logger))

	router.GET(httpapi.TauthScriptPath, func(context *gin.Context) {
		context.Header("Content-Type", "application/javascript; charset=utf-8")
		context.String(http.StatusOK, dashboardTestTauthStubScript)
	})
	router.POST(httpapi.TauthLogoutPath, func(context *gin.Context) {
		context.SetCookie(dashboardTestSessionCookieName, "", -1, "/", "", false, true)
		context.Status(http.StatusNoContent)
	})
	router.GET("/", func(context *gin.Context) {
		context.Redirect(http.StatusFound, dashboardTestLandingPath)
	})
	router.GET(dashboardTestLandingPath, landingHandlers.RenderLandingPage)
	router.GET(httpapi.PrivacyPagePath, privacyHandlers.RenderPrivacyPage)
	router.GET(httpapi.SitemapRoutePath, sitemapHandlers.RenderSitemap)
	router.GET(dashboardTestDashboardRoute, authManager.RequireAuthenticatedWeb(), dashboardHandlers.RenderDashboard)
	router.GET("/app/sites/:id/widget-test", authManager.RequireAuthenticatedWeb(), widgetTestHandlers.RenderWidgetTestPage)
	router.POST("/app/sites/:id/widget-test/feedback", authManager.RequireAuthenticatedJSON(), widgetTestHandlers.SubmitWidgetTestFeedback)
	router.GET("/app/sites/:id/traffic-test", authManager.RequireAuthenticatedWeb(), trafficTestHandlers.RenderTrafficTestPage)
	router.GET("/app/sites/:id/subscribe-test", authManager.RequireAuthenticatedWeb(), subscribeTestHandlers.RenderSubscribeTestPage)
	router.GET("/app/sites/:id/subscribe-test/events", authManager.RequireAuthenticatedJSON(), subscribeTestHandlers.StreamSubscriptionTestEvents)
	router.POST("/app/sites/:id/subscribe-test/subscriptions", authManager.RequireAuthenticatedJSON(), subscribeTestHandlers.CreateSubscription)

	router.GET("/api/visits", publicHandlers.CollectVisit)
	router.POST("/api/feedback", publicHandlers.CreateFeedback)
	router.POST("/api/subscriptions", publicHandlers.CreateSubscription)
	router.GET("/subscribe.js", publicHandlers.SubscribeJS)
	router.GET("/subscribe-demo", publicHandlers.SubscribeDemo)
	router.GET("/widget.js", publicHandlers.WidgetJS)

	apiGroup := router.Group("/api")
	apiGroup.Use(authManager.RequireAuthenticatedJSON())
	apiGroup.GET("/me", func(context *gin.Context) {
		if config.userLoadDelay > 0 {
			time.Sleep(config.userLoadDelay)
		}
		siteHandlers.CurrentUser(context)
	})
	apiGroup.GET("/me/avatar", siteHandlers.UserAvatar)
	apiGroup.GET("/sites", siteHandlers.ListSites)
	apiGroup.POST("/sites", siteHandlers.CreateSite)
	apiGroup.PATCH("/sites/:id", siteHandlers.UpdateSite)
	apiGroup.DELETE("/sites/:id", siteHandlers.DeleteSite)
	apiGroup.GET("/sites/:id/messages", siteHandlers.ListMessagesBySite)
	apiGroup.GET("/sites/:id/visits/stats", siteHandlers.VisitStats)
	apiGroup.GET("/sites/:id/favicon", siteHandlers.SiteFavicon)
	apiGroup.GET("/sites/favicons/events", siteHandlers.StreamFaviconUpdates)
	apiGroup.GET("/sites/feedback/events", siteHandlers.StreamFeedbackUpdates)

	server := newHTTPTestServer(testingT, router)

	testingT.Cleanup(func() {
		managerCancel()
		faviconManager.Stop()
		feedbackBroadcaster.Close()
		subscriptionEvents.Close()
		require.NoError(testingT, sqlDatabase.Close())
	})

	return &dashboardIntegrationHarness{
		router:             router,
		authManager:        authManager,
		faviconManager:     faviconManager,
		database:           gormDatabase,
		sqlDB:              sqlDatabase,
		server:             server,
		baseURL:            server.URL,
		subscriptionEvents: subscriptionEvents,
	}
}

func (harness *dashboardIntegrationHarness) Close() {
	// cleanup handled via testing.Cleanup
}

type staticFaviconResolver struct{}

func (resolver *staticFaviconResolver) Resolve(ctx context.Context, allowedOrigin string) (string, error) {
	return "", nil
}

func (resolver *staticFaviconResolver) ResolveAsset(ctx context.Context, allowedOrigin string) (*favicon.Asset, error) {
	return nil, nil
}

func createAuthenticatedSessionCookie(testingT *testing.T, email string, name string) *http.Cookie {
	testingT.Helper()
	return createAuthenticatedSessionCookieWithAvatar(testingT, email, name, "")
}

func createAuthenticatedSessionCookieWithAvatar(testingT *testing.T, email string, name string, avatarURL string) *http.Cookie {
	testingT.Helper()

	now := time.Now().UTC()
	claims := &sessionvalidator.Claims{
		TenantID:        dashboardTestTauthTenantID,
		UserID:          "test-user",
		UserEmail:       email,
		UserDisplayName: name,
		UserAvatarURL:   avatarURL,
		UserRoles:       []string{},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    dashboardTestJWTIssuer,
			Subject:   "test-user",
			IssuedAt:  jwt.NewNumericDate(now.Add(-time.Minute)),
			ExpiresAt: jwt.NewNumericDate(now.Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString([]byte(dashboardTestTauthSigningKey))
	require.NoError(testingT, err)

	return &http.Cookie{
		Name:  dashboardTestSessionCookieName,
		Value: signedToken,
		Path:  "/",
	}
}

func openDashboardUserMenu(testingT *testing.T, page *rod.Page) {
	testingT.Helper()

	isOpen := evaluateScriptBoolean(testingT, page, `(function(){
		var menu = document.querySelector('mpr-user[data-loopaware-user-menu="true"]');
		if (!menu) { return false; }
		return menu.getAttribute('data-mpr-user-open') === 'true';
	}())`)
	if !isOpen {
		clickSelector(testingT, page, dashboardUserMenuTriggerSelector)
	}
	waitForVisibleElement(testingT, page, dashboardUserMenuOpenSelector)
}
