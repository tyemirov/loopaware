package footer

import (
	"encoding/json"
	"html"
	"html/template"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderCurrentSharedMenu(testingT *testing.T) {
	config := baseFooterConfig()
	config.MenuLabel = testFooterMenuLabel
	rendered, err := Render(config)
	require.NoError(testingT, err)
	match := regexp.MustCompile(`\smenu='([^']*)'`).FindStringSubmatch(string(rendered))
	require.Len(testingT, match, 2, "footer must emit the current menu attribute")
	var menu map[string]any
	require.NoError(testingT, json.Unmarshal([]byte(html.UnescapeString(match[1])), &menu))
	require.Equal(testingT, map[string]any{
		"label":     testFooterMenuLabel,
		"placement": "top",
		"sections": []any{map[string]any{
			"id": "mpr-projects", "label": "Projects", "mode": "static",
			"links": []any{map[string]any{"label": testFooterExampleLinkName, "href": testFooterExampleLinkURL}},
		}},
	}, menu)
	for _, obsolete := range []string{"links-collection=", "toggle-button-id=", "toggle-button-class=", "toggle-label=", "menu-wrapper-class=", "menu-class=", "menu-item-class="} {
		require.NotContains(testingT, string(rendered), obsolete)
	}
}

const (
	testFooterHostID          = "footer-host"
	testFooterElementID       = "footer-element"
	testFooterInnerID         = "footer-inner"
	testFooterBaseClass       = "footer-base"
	testFooterInnerClass      = "footer-inner"
	testFooterWrapperClass    = "footer-wrapper"
	testFooterBrandClass      = "footer-brand"
	testFooterPrefixClass     = "footer-prefix"
	testFooterPrefixText      = "LoopAware"
	testFooterPrivacyClass    = "footer-privacy"
	testFooterPrivacyHref     = "/privacy"
	testFooterPrivacyLabel    = "Privacy"
	testFooterPrivacyModal    = "<div>Privacy</div>"
	testFooterMenuLabel       = "More"
	testFooterThemeAttribute  = "data-theme"
	testFooterThemeAriaLabel  = "Theme"
	testFooterThemeLightMode  = "light"
	testFooterThemeDarkMode   = "dark"
	testFooterSmallSize       = "small"
	testFooterThemeSwitcher   = "square"
	testFooterExampleLinkName = "Docs"
	testFooterExampleLinkURL  = "/docs"
	testFooterTemplateName    = "footer"
	testFooterTemplateOption  = "missingkey=error"
	testFooterTemplateError   = "{{.MissingValue}}"
)

func baseFooterConfig() Config {
	return Config{
		HostElementID:     testFooterHostID,
		ElementID:         testFooterElementID,
		InnerElementID:    testFooterInnerID,
		BaseClass:         testFooterBaseClass,
		InnerClass:        testFooterInnerClass,
		WrapperClass:      testFooterWrapperClass,
		BrandWrapperClass: testFooterBrandClass,
		PrefixClass:       testFooterPrefixClass,
		PrefixText:        testFooterPrefixText,
		MenuLabel:         testFooterMenuLabel,
		PrivacyLinkClass:  testFooterPrivacyClass,
		PrivacyLinkHref:   testFooterPrivacyHref,
		PrivacyLinkLabel:  testFooterPrivacyLabel,
		Links: []Link{
			{
				Label: testFooterExampleLinkName,
				Href:  testFooterExampleLinkURL,
			},
		},
	}
}

func TestRenderFooterWithThemeToggle(testingT *testing.T) {
	footerConfig := baseFooterConfig()
	footerConfig.PrivacyModalHTML = testFooterPrivacyModal
	footerConfig.Sticky = true
	footerConfig.Size = testFooterSmallSize
	footerConfig.ThemeToggleEnabled = true
	footerConfig.ThemeMode = testFooterThemeLightMode
	footerConfig.ThemeAttribute = testFooterThemeAttribute
	footerConfig.ThemeAriaLabel = testFooterThemeAriaLabel
	footerConfig.ThemeModes = []string{testFooterThemeLightMode, testFooterThemeDarkMode}

	rendered, renderErr := Render(footerConfig)
	require.NoError(testingT, renderErr)

	renderedText := string(rendered)
	require.Contains(testingT, renderedText, testFooterPrefixText)
	require.Contains(testingT, renderedText, `sticky="true"`)
	require.Contains(testingT, renderedText, `size="`+testFooterSmallSize+`"`)
	require.Contains(testingT, renderedText, `theme-switcher="toggle"`)
	require.Contains(testingT, renderedText, testFooterThemeAttribute)
	require.Contains(testingT, renderedText, testFooterThemeLightMode)
	require.Contains(testingT, renderedText, "privacy-modal-content")
}

func TestRenderFooterWithoutThemeToggleUsesMenuLabel(testingT *testing.T) {
	footerConfig := baseFooterConfig()
	footerConfig.MenuLabel = testFooterMenuLabel
	footerConfig.Sticky = false
	footerConfig.ThemeToggleEnabled = false

	rendered, renderErr := Render(footerConfig)
	require.NoError(testingT, renderErr)

	renderedText := string(rendered)
	require.Contains(testingT, renderedText, testFooterMenuLabel)
	require.Contains(testingT, renderedText, `sticky="false"`)
	require.False(testingT, strings.Contains(renderedText, "theme-switcher="))
}

func TestRenderFooterReportsTemplateError(testingT *testing.T) {
	originalTemplate := footerTemplate
	testingT.Cleanup(func() {
		footerTemplate = originalTemplate
	})
	footerTemplate = template.Must(template.New(testFooterTemplateName).Option(testFooterTemplateOption).Parse(testFooterTemplateError))

	footerConfig := baseFooterConfig()
	_, renderErr := Render(footerConfig)
	require.Error(testingT, renderErr)
}

func TestRenderFooterUsesCustomThemeSwitcher(testingT *testing.T) {
	footerConfig := baseFooterConfig()
	footerConfig.ThemeToggleEnabled = true
	footerConfig.ThemeSwitcher = testFooterThemeSwitcher
	footerConfig.ThemeMode = testFooterThemeLightMode
	footerConfig.ThemeModes = []string{testFooterThemeLightMode, testFooterThemeDarkMode}

	rendered, renderErr := Render(footerConfig)
	require.NoError(testingT, renderErr)

	renderedText := string(rendered)
	require.Contains(testingT, renderedText, `theme-switcher="`+testFooterThemeSwitcher+`"`)
}
