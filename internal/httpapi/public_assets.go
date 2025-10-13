package httpapi

import (
	"bytes"
	"html/template"
)

const (
	publicBrandName              = "LoopAware"
	publicLoginPath              = "/auth/google"
	publicThemeToggleID          = "public-theme-toggle"
	publicThemeStorageKey        = "loopaware_public_theme"
	publicLandingThemeStorageKey = "loopaware_landing_theme"
	publicLegacyThemeStorageKey  = "landing_theme"
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
      .landing-navbar {
        border-radius: 0;
        padding: 0.35rem 0;
      }
      .landing-brand {
        font-size: 1.25rem;
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
        background-color: #0f172a;
      }
      body[data-bs-theme="dark"] .landing-navbar {
        background-color: #0f172a;
      }
      body[data-bs-theme="dark"] .landing-navbar .navbar-brand,
      body[data-bs-theme="dark"] .landing-navbar .btn-primary {
        color: #f8fafc;
      }
      body[data-bs-theme="dark"] .landing-navbar .btn-primary {
        background-color: #2563eb;
        border-color: #2563eb;
      }
      body[data-bs-theme="dark"] .landing-navbar .form-check-input {
        background-color: #334155;
        border-color: #475569;
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
        background-color: #ffffff;
      }
      body[data-bs-theme="light"] .landing-navbar {
        background-color: #ffffff;
      }
      body[data-bs-theme="light"] .landing-navbar .navbar-brand {
        color: #0f172a;
      }
      body[data-bs-theme="light"] .landing-navbar .btn-primary {
        background-color: #2563eb;
        border-color: #2563eb;
        color: #ffffff;
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
	publicHeaderTemplate = template.Must(template.New("public_header").Parse(`<header class="landing-header">
  <nav class="navbar landing-navbar shadow-sm px-3">
    <div class="container d-flex align-items-center justify-content-between">
      <span class="navbar-brand d-flex align-items-center gap-3 landing-brand">
        <span class="landing-logo">
          <img src="{{.LogoDataURI}}" alt="LoopAware logo" class="landing-logo-image" />
        </span>
        <span>{{.BrandName}}</span>
      </span>
      <div class="d-flex align-items-center gap-3">
        <div class="form-check form-switch m-0" data-bs-theme="light">
          <input class="form-check-input" type="checkbox" id="{{.ThemeToggleID}}" aria-label="Toggle theme" />
        </div>
        <a class="btn btn-primary btn-sm" href="{{.LoginPath}}">Login</a>
      </div>
    </div>
  </nav>
</header>`))
	publicThemeScriptTemplate = template.Must(template.New("public_theme_script").Parse(`(function() {
  var publicThemeStorageKey = '{{.PublicThemeStorageKey}}';
  var landingThemeStorageKey = '{{.LandingThemeStorageKey}}';
  var legacyThemeStorageKey = '{{.LegacyThemeStorageKey}}';
  var themeToggleElement = document.getElementById('{{.ThemeToggleID}}');
  var rootElement = document.body;
  function applyPublicTheme(theme) {
    var normalizedTheme = theme === 'light' ? 'light' : 'dark';
    rootElement.setAttribute('data-bs-theme', normalizedTheme);
    rootElement.classList.toggle('bg-body', true);
    rootElement.classList.toggle('text-body', true);
    if (themeToggleElement) {
      themeToggleElement.checked = normalizedTheme === 'light';
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
  if (themeToggleElement) {
    themeToggleElement.addEventListener('change', function(event) {
      var nextTheme = event.target.checked ? 'light' : 'dark';
      applyPublicTheme(nextTheme);
      persistPublicTheme(nextTheme);
    });
  }
  initializePublicTheme();
})();`))
)

type publicHeaderTemplateData struct {
	LogoDataURI   template.URL
	BrandName     string
	ThemeToggleID string
	LoginPath     string
}

type publicThemeScriptTemplateData struct {
	ThemeToggleID          string
	PublicThemeStorageKey  string
	LandingThemeStorageKey string
	LegacyThemeStorageKey  string
}

func renderPublicHeader(logoDataURI template.URL) (template.HTML, error) {
	data := publicHeaderTemplateData{
		LogoDataURI:   logoDataURI,
		BrandName:     publicBrandName,
		ThemeToggleID: publicThemeToggleID,
		LoginPath:     publicLoginPath,
	}
	var buffer bytes.Buffer
	if err := publicHeaderTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}

func renderPublicThemeScript() (template.JS, error) {
	data := publicThemeScriptTemplateData{
		ThemeToggleID:          publicThemeToggleID,
		PublicThemeStorageKey:  publicThemeStorageKey,
		LandingThemeStorageKey: publicLandingThemeStorageKey,
		LegacyThemeStorageKey:  publicLegacyThemeStorageKey,
	}
	var buffer bytes.Buffer
	if err := publicThemeScriptTemplate.Execute(&buffer, data); err != nil {
		return "", err
	}
	return template.JS(buffer.String()), nil
}

func sharedPublicStyles() template.CSS {
	return template.CSS(publicSharedStylesCSS)
}

func privacyPageStyles() template.CSS {
	return template.CSS(privacyPageStylesCSS)
}
