package httpapi

import (
	"fmt"
	"html/template"

	"github.com/MarkoPoloResearchLab/loopaware/pkg/footer"
)

const (
	footerPrivacyLinkLabel = "Privacy • Terms"
	footerPrivacyLinkHref  = PrivacyPagePath

	footerBrandText            = dashboardFooterBrandPrefix + " " + dashboardFooterBrandName
	footerInnerClass           = "mpr-footer__inner"
	footerLayoutClass          = "mpr-footer__layout"
	footerBrandWrapperClass    = "mpr-footer__brand"
	footerMenuWrapperClass     = "mpr-footer__menu-wrapper"
	footerPrivacyLinkClass     = "mpr-footer__privacy"
	footerPrefixClass          = "mpr-footer__prefix"
	footerToggleButtonClass    = "mpr-footer__menu-button"
	footerMenuClass            = "mpr-footer__menu"
	footerMenuItemClass        = "mpr-footer__menu-item"
	footerThemeToggleAriaLabel = "Toggle theme"
	footerThemeAttribute       = "data-bs-theme"
	footerThemeSwitcher        = "toggle"
)

type footerVariant string

const (
	footerVariantLanding   footerVariant = "landing"
	footerVariantPrivacy   footerVariant = "privacy"
	footerVariantDashboard footerVariant = "dashboard"
)

var footerThemeModes = []string{"light", "dark"}

type footerVariantOverrides struct {
	ElementID          string
	InnerElementID     string
	BaseClass          string
	ToggleButtonID     string
	ThemeToggleEnabled bool
	ThemeMode          string
}

var (
	footerBaseConfig = footer.Config{
		InnerClass:         footerInnerClass,
		WrapperClass:       footerLayoutClass,
		BrandWrapperClass:  footerBrandWrapperClass,
		MenuWrapperClass:   footerMenuWrapperClass,
		PrefixClass:        footerPrefixClass,
		PrefixText:         footerBrandText,
		ToggleButtonClass:  footerToggleButtonClass,
		ToggleLabel:        footerBrandText,
		MenuClass:          footerMenuClass,
		MenuItemClass:      footerMenuItemClass,
		PrivacyLinkClass:   footerPrivacyLinkClass,
		PrivacyLinkHref:    footerPrivacyLinkHref,
		PrivacyLinkLabel:   footerPrivacyLinkLabel,
		Links:              footerLinks,
		Sticky:             false,
		ThemeToggleEnabled: false,
		ThemeSwitcher:      footerThemeSwitcher,
		ThemeAttribute:     footerThemeAttribute,
		ThemeAriaLabel:     footerThemeToggleAriaLabel,
		ThemeModes:         footerThemeModes,
	}
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{
		footerVariantLanding: {
			ElementID:          landingFooterElementID,
			InnerElementID:     landingFooterInnerID,
			BaseClass:          landingFooterBaseClass,
			ToggleButtonID:     landingFooterToggleID,
			ThemeToggleEnabled: true,
			ThemeMode:          "dark",
		},
		footerVariantPrivacy: {
			ElementID:          privacyFooterElementID,
			InnerElementID:     privacyFooterInnerID,
			BaseClass:          landingFooterBaseClass,
			ToggleButtonID:     dashboardFooterToggleButtonID,
			ThemeToggleEnabled: true,
			ThemeMode:          "dark",
		},
		footerVariantDashboard: {
			ElementID:          footerElementID,
			InnerElementID:     footerInnerElementID,
			BaseClass:          footerBaseClass,
			ToggleButtonID:     dashboardFooterToggleButtonID,
			ThemeToggleEnabled: true,
			ThemeMode:          "light",
		},
	}
)

func footerConfigForVariant(variant footerVariant) (footer.Config, error) {
	overrides, ok := footerVariantOverridesByKey[variant]
	if !ok {
		return footer.Config{}, fmt.Errorf("unknown footer variant: %s", variant)
	}
	config := footerBaseConfig
	config.HostElementID = overrides.ElementID
	config.ElementID = overrides.ElementID + "-root"
	config.InnerElementID = overrides.InnerElementID
	config.BaseClass = overrides.BaseClass
	config.ToggleButtonID = overrides.ToggleButtonID
	config.ThemeToggleEnabled = overrides.ThemeToggleEnabled
	config.ThemeMode = overrides.ThemeMode
	return config, nil
}

func renderFooterHTMLForVariant(variant footerVariant) (template.HTML, error) {
	config, configErr := footerConfigForVariant(variant)
	if configErr != nil {
		return "", configErr
	}
	return footer.Render(config)
}

var footerLinks = []footer.Link{
	{Label: "Marco Polo Research Lab", URL: "https://mprlab.com"},
	{Label: "Gravity Notes", URL: "https://gravity.mprlab.com"},
	{Label: "LoopAware", URL: "https://loopaware.mprlab.com"},
	{Label: "Allergy Wheel", URL: "https://allergy.mprlab.com"},
	{Label: "Social Threader", URL: "https://threader.mprlab.com"},
	{Label: "RSVP", URL: "https://rsvp.mprlab.com"},
	{Label: "Countdown Calendar", URL: "https://countdown.mprlab.com"},
	{Label: "LLM Crossword", URL: "https://llm-crossword.mprlab.com"},
	{Label: "Prompt Bubbles", URL: "https://prompts.mprlab.com"},
	{Label: "Wallpapers", URL: "https://wallpapers.mprlab.com"},
}
