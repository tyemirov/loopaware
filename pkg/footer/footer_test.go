package footer

import (
	"html/template"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testFooterHostID          = "footer-host"
	testFooterElementID       = "footer-element"
	testFooterInnerID         = "footer-inner"
	testFooterBaseClass       = "footer-base"
	testFooterInnerClass      = "footer-inner"
	testFooterWrapperClass    = "footer-wrapper"
	testFooterBrandClass      = "footer-brand"
	testFooterMenuWrapper     = "footer-menu-wrapper"
	testFooterPrefixClass     = "footer-prefix"
	testFooterPrefixText      = "LoopAware"
	testFooterToggleButtonID  = "footer-toggle"
	testFooterToggleClass     = "footer-toggle-class"
	testFooterMenuClass       = "footer-menu"
	testFooterMenuItemClass   = "footer-menu-item"
	testFooterPrivacyClass    = "footer-privacy"
	testFooterPrivacyHref     = "/privacy"
	testFooterPrivacyLabel    = "Privacy"
	testFooterPrivacyModal    = "<div>Privacy</div>"
	testFooterToggleLabel     = "More"
	testFooterThemeAttribute  = "data-theme"
	testFooterThemeAriaLabel  = "Theme"
	testFooterThemeLightMode  = "light"
	testFooterThemeDarkMode   = "dark"
	testFooterSmallSize       = "sm"
	testFooterThemeSwitcher   = "custom-switcher"
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
		MenuWrapperClass:  testFooterMenuWrapper,
		PrefixClass:       testFooterPrefixClass,
		PrefixText:        testFooterPrefixText,
		ToggleButtonID:    testFooterToggleButtonID,
		ToggleButtonClass: testFooterToggleClass,
		MenuClass:         testFooterMenuClass,
		MenuItemClass:     testFooterMenuItemClass,
		PrivacyLinkClass:  testFooterPrivacyClass,
		PrivacyLinkHref:   testFooterPrivacyHref,
		PrivacyLinkLabel:  testFooterPrivacyLabel,
		Links: []Link{
			{
				Label: testFooterExampleLinkName,
				URL:   testFooterExampleLinkURL,
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

func TestRenderFooterWithoutThemeToggleUsesToggleLabel(testingT *testing.T) {
	footerConfig := baseFooterConfig()
	footerConfig.ToggleLabel = testFooterToggleLabel
	footerConfig.Sticky = false
	footerConfig.ThemeToggleEnabled = false

	rendered, renderErr := Render(footerConfig)
	require.NoError(testingT, renderErr)

	renderedText := string(rendered)
	require.Contains(testingT, renderedText, testFooterToggleLabel)
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
