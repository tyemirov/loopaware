package httpapi

import (
	"bytes"
	"fmt"
	"html/template"
)

const (
	publicBrandName              = "LoopAware"
	publicThemeStorageKey        = "loopaware_public_theme"
	publicLandingThemeStorageKey = "loopaware_landing_theme"
	publicLegacyThemeStorageKey  = "landing_theme"
	publicLandingPath            = LandingPagePath
	publicDashboardPath          = "/app"
	publicSignInLabel            = "Login"
	publicSignOutLabel           = "Logout"
	publicHeroScrollTarget       = "#top"
	publicHeroAttributeName      = "data-public-hero"
	publicHeroAttributeValue     = "true"
	publicHeroScrollAttribute    = "data-scroll-to-top"
	publicHeroScrollValue        = "true"
	publicSharedStylesCSS        = `.landing-body {
        transition: background-color 0.3s ease, color 0.3s ease;
      }
      .landing-header {
        position: sticky;
        top: 0;
        z-index: 1030;
        padding: 0;
        transition: background-color 0.3s ease;
      }
      .landing-brand {
        font-size: 1.25rem;
        font-weight: 600;
      }
      .landing-logo {
        display: inline-flex;
        align-items: center;
        justify-content: center;
      }
      .landing-logo-image {
        width: 48px;
        height: 48px;
      }
      .landing-card {
        transition: transform 0.2s ease, box-shadow 0.2s ease;
        cursor: default;
      }
      .landing-card:hover,
      .landing-card:focus-visible {
        transform: translateY(-4px);
        box-shadow: 0 1.25rem 1.5rem -1rem rgba(15, 23, 42, 0.35);
      }
      .landing-card:focus-visible {
        outline: 0;
      }
      body[data-bs-theme="dark"] .landing-header {
        --mpr-color-surface-primary: #0f172a;
        --mpr-color-text-primary: #f8fafc;
        --mpr-color-border: rgba(148, 163, 184, 0.25);
        --mpr-chip-bg: rgba(148, 163, 184, 0.18);
      }
      body[data-bs-theme="dark"] .landing-card {
        background-color: rgba(15, 23, 42, 0.8);
        color: #e2e8f0;
      }
      body[data-bs-theme="dark"] .landing-card p {
        color: #cbd5f5;
      }
      body[data-bs-theme="dark"] .landing-footer {
        background-color: #0f172a;
        color: #94a3b8;
      }
      body[data-bs-theme="light"] .landing-header {
        --mpr-color-surface-primary: #ffffff;
        --mpr-color-text-primary: #0f172a;
        --mpr-color-border: rgba(148, 163, 184, 0.2);
        --mpr-chip-bg: rgba(148, 163, 184, 0.18);
      }
      body[data-bs-theme="light"] .landing-card {
        background-color: #f8fafc;
        color: #0f172a;
      }
      body[data-bs-theme="light"] .landing-card p {
        color: #475569;
      }
      body[data-bs-theme="light"] .landing-footer {
        background-color: #ffffff;
        color: #475569;
      }`
	privacyPageStylesCSS = `body{font:16px/1.5 system-ui,Segoe UI,Roboto,Helvetica,Arial,sans-serif;margin:0}
      .privacy-container{max-width:800px;margin:40px auto}
      .privacy-heading{font-size:1.6rem;margin-bottom:.2rem}`
)

var (
	publicHeaderTemplate = template.Must(template.New("public_header").Parse(`<mpr-header class="landing-header" google-site-id="{{.GoogleClientID}}"{{if .TauthBaseURL}} tauth-url="{{.TauthBaseURL}}"{{end}} tauth-tenant-id="{{.TauthTenantID}}" tauth-login-path="{{.TauthLoginPath}}" tauth-logout-path="{{.TauthLogoutPath}}" tauth-nonce-path="{{.TauthNoncePath}}" sign-in-label="{{.SignInLabel}}" sign-out-label="{{.SignOutLabel}}">
  <a slot="brand" class="landing-brand d-inline-flex align-items-center gap-3 text-decoration-none" href="{{.HeroTarget}}" {{.HeroDataAttribute}}{{if .HeroScrollAttribute}} {{.HeroScrollAttribute}}{{end}}>
    <span class="landing-logo">
      <img src="{{.LogoDataURI}}" alt="LoopAware logo" class="landing-logo-image" />
    </span>
    <span>{{.BrandName}}</span>
  </a>
</mpr-header>`))
	publicThemeScriptTemplate = template.Must(template.New("public_theme_script").Parse(`(function() {
  var publicThemeStorageKey = '{{.PublicThemeStorageKey}}';
  var landingThemeStorageKey = '{{.LandingThemeStorageKey}}';
  var legacyThemeStorageKey = '{{.LegacyThemeStorageKey}}';
  var rootElement = document.body;
  var documentRoot = document.documentElement;
  var footerElement = document.querySelector('mpr-footer');
  function applyPublicTheme(theme) {
    var normalizedTheme = theme === 'light' ? 'light' : 'dark';
    rootElement.setAttribute('data-bs-theme', normalizedTheme);
    rootElement.setAttribute('data-mpr-theme', normalizedTheme);
    if (documentRoot) {
      documentRoot.setAttribute('data-bs-theme', normalizedTheme);
      documentRoot.setAttribute('data-mpr-theme', normalizedTheme);
    }
    rootElement.classList.toggle('bg-body', true);
    rootElement.classList.toggle('text-body', true);
    if (footerElement) {
      footerElement.setAttribute('theme-mode', normalizedTheme);
    }
  }
  function loadPublicTheme() {
    var storedTheme = localStorage.getItem(publicThemeStorageKey);
    if (storedTheme === null) {
      var landingStoredTheme = localStorage.getItem(landingThemeStorageKey);
      if (landingStoredTheme === null) {
        var legacyStoredTheme = localStorage.getItem(legacyThemeStorageKey);
        if (legacyStoredTheme === 'light' || legacyStoredTheme === 'dark') {
          landingStoredTheme = legacyStoredTheme;
          localStorage.setItem(landingThemeStorageKey, landingStoredTheme);
        }
      }
      if (landingStoredTheme === 'light' || landingStoredTheme === 'dark') {
        storedTheme = landingStoredTheme;
        localStorage.setItem(publicThemeStorageKey, storedTheme);
      }
    }
    return storedTheme;
  }
  function persistPublicTheme(theme) {
    localStorage.setItem(publicThemeStorageKey, theme);
    localStorage.setItem(landingThemeStorageKey, theme);
  }
  function initializePublicTheme() {
    var storedTheme = loadPublicTheme();
    applyPublicTheme(storedTheme);
  }
  if (footerElement) {
    footerElement.addEventListener('mpr-footer:theme-change', function(event) {
      var nextTheme = event && event.detail && event.detail.theme === 'dark' ? 'dark' : 'light';
      applyPublicTheme(nextTheme);
      persistPublicTheme(nextTheme);
    });
  }
  initializePublicTheme();
  var heroElement = document.querySelector('[{{.HeroAttributeName}}]');
  if (heroElement) {
    var shouldScrollToTop = heroElement.getAttribute('{{.HeroScrollAttributeName}}') === '{{.HeroScrollAttributeValue}}';
    if (shouldScrollToTop) {
      heroElement.addEventListener('click', function(event) {
        event.preventDefault();
        window.scrollTo({ top: 0, behavior: 'smooth' });
      });
    }
  }
})();`))
)

type publicHeaderTemplateData struct {
	LogoDataURI         template.URL
	BrandName           string
	HeroTarget          string
	HeroDataAttribute   template.HTMLAttr
	HeroScrollAttribute template.HTMLAttr
	GoogleClientID      string
	TauthBaseURL        string
	TauthTenantID       string
	TauthLoginPath      string
	TauthLogoutPath     string
	TauthNoncePath      string
	SignInLabel         string
	SignOutLabel        string
}

type publicThemeScriptTemplateData struct {
	PublicThemeStorageKey    string
	LandingThemeStorageKey   string
	LegacyThemeStorageKey    string
	HeroAttributeName        string
	HeroScrollAttributeName  string
	HeroScrollAttributeValue string
}

type publicPageType string

const (
	publicPageLanding publicPageType = "landing"
	publicPagePrivacy publicPageType = "privacy"
)

type publicHeroBehavior struct {
	Target       string
	ShouldScroll bool
}

func renderPublicHeader(logoDataURI template.URL, isAuthenticated bool, pageType publicPageType, authConfig AuthClientConfig) (template.HTML, error) {
	heroBehavior := resolvePublicHeroBehavior(isAuthenticated, pageType)
	data := publicHeaderTemplateData{
		LogoDataURI:       logoDataURI,
		BrandName:         publicBrandName,
		HeroTarget:        heroBehavior.Target,
		HeroDataAttribute: template.HTMLAttr(fmt.Sprintf(`%s="%s"`, publicHeroAttributeName, publicHeroAttributeValue)),
		GoogleClientID:    authConfig.GoogleClientID,
		TauthBaseURL:      authConfig.TauthBaseURL,
		TauthTenantID:     authConfig.TauthTenantID,
		TauthLoginPath:    TauthLoginPath,
		TauthLogoutPath:   TauthLogoutPath,
		TauthNoncePath:    TauthNoncePath,
		SignInLabel:       publicSignInLabel,
		SignOutLabel:      publicSignOutLabel,
	}
	if heroBehavior.ShouldScroll {
		data.HeroScrollAttribute = template.HTMLAttr(fmt.Sprintf(`%s="%s"`, publicHeroScrollAttribute, publicHeroScrollValue))
	}
	var buffer bytes.Buffer
	if err := publicHeaderTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}

func renderPublicThemeScript() (template.JS, error) {
	data := publicThemeScriptTemplateData{
		PublicThemeStorageKey:    publicThemeStorageKey,
		LandingThemeStorageKey:   publicLandingThemeStorageKey,
		LegacyThemeStorageKey:    publicLegacyThemeStorageKey,
		HeroAttributeName:        publicHeroAttributeName,
		HeroScrollAttributeName:  publicHeroScrollAttribute,
		HeroScrollAttributeValue: publicHeroScrollValue,
	}
	var buffer bytes.Buffer
	if err := publicThemeScriptTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return template.JS(buffer.String()), nil
}

func resolvePublicHeroBehavior(isAuthenticated bool, pageType publicPageType) publicHeroBehavior {
	if isAuthenticated {
		return publicHeroBehavior{Target: publicDashboardPath}
	}
	if pageType == publicPageLanding {
		return publicHeroBehavior{Target: publicHeroScrollTarget, ShouldScroll: true}
	}
	return publicHeroBehavior{Target: publicLandingPath}
}

func sharedPublicStyles() template.CSS {
	return template.CSS(publicSharedStylesCSS)
}

func privacyPageStyles() template.CSS {
	return template.CSS(privacyPageStylesCSS)
}
