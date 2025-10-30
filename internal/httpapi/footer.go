package httpapi

import (
	"fmt"
	"html/template"

	"github.com/MarkoPoloResearchLab/loopaware/pkg/footer"
)

const (
	footerPrivacyLinkLabel = "Privacy • Terms"
	footerPrivacyLinkHref  = PrivacyPagePath

	footerLayoutClass             = "footer-layout w-100 d-flex flex-column flex-md-row align-items-start align-items-md-center justify-content-between gap-3"
	footerBrandWrapperClass       = "footer-brand d-inline-flex align-items-center gap-2 text-body-secondary small"
	footerMenuWrapperClass        = "footer-menu dropup"
	footerPrivacyLinkClass        = "footer-privacy-link text-body-secondary text-decoration-none small"
	footerPrefixClass             = "text-body-secondary fw-semibold"
	footerToggleButtonClass       = "btn btn-link dropdown-toggle text-decoration-none px-0 fw-semibold text-body-secondary"
	footerMenuClass               = "dropdown-menu dropdown-menu-end shadow"
	footerMenuItemClass           = "dropdown-item"
	footerThemeToggleWrapperClass = "footer-theme-toggle form-check form-switch m-0"
	footerThemeToggleInputClass   = "form-check-input"
	footerThemeToggleDataTheme    = "light"
	footerThemeToggleAriaLabel    = "Toggle theme"
)

type footerVariant string

const (
	footerVariantLanding   footerVariant = "landing"
	footerVariantPrivacy   footerVariant = "privacy"
	footerVariantDashboard footerVariant = "dashboard"
)

type footerVariantOverrides struct {
	ElementID          string
	InnerElementID     string
	BaseClass          string
	ToggleButtonID     string
	ThemeToggleEnabled bool
}

var (
	footerBaseConfig = footer.Config{
		InnerClass:              landingFooterInnerClass,
		WrapperClass:            footerLayoutClass,
		BrandWrapperClass:       footerBrandWrapperClass,
		MenuWrapperClass:        footerMenuWrapperClass,
		PrefixClass:             footerPrefixClass,
		PrefixText:              dashboardFooterBrandPrefix,
		ToggleButtonClass:       footerToggleButtonClass,
		ToggleLabel:             dashboardFooterBrandName,
		MenuClass:               footerMenuClass,
		MenuItemClass:           footerMenuItemClass,
		PrivacyLinkClass:        footerPrivacyLinkClass,
		PrivacyLinkHref:         footerPrivacyLinkHref,
		PrivacyLinkLabel:        footerPrivacyLinkLabel,
		Links:                   footerLinks,
		ThemeToggleEnabled:      false,
		ThemeToggleWrapperClass: footerThemeToggleWrapperClass,
		ThemeToggleInputClass:   footerThemeToggleInputClass,
		ThemeToggleDataTheme:    footerThemeToggleDataTheme,
		ThemeToggleAriaLabel:    footerThemeToggleAriaLabel,
		ThemeToggleID:           publicThemeToggleID,
	}
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{
		footerVariantLanding: {
			ElementID:          landingFooterElementID,
			InnerElementID:     landingFooterInnerID,
			BaseClass:          landingFooterBaseClass,
			ToggleButtonID:     landingFooterToggleID,
			ThemeToggleEnabled: true,
		},
		footerVariantPrivacy: {
			ElementID:          privacyFooterElementID,
			InnerElementID:     privacyFooterInnerID,
			BaseClass:          landingFooterBaseClass,
			ToggleButtonID:     dashboardFooterToggleButtonID,
			ThemeToggleEnabled: true,
		},
		footerVariantDashboard: {
			ElementID:      footerElementID,
			InnerElementID: footerInnerElementID,
			BaseClass:      footerBaseClass,
			ToggleButtonID: dashboardFooterToggleButtonID,
		},
	}
)

func footerConfigForVariant(variant footerVariant) (footer.Config, error) {
	overrides, ok := footerVariantOverridesByKey[variant]
	if !ok {
		return footer.Config{}, fmt.Errorf("unknown footer variant: %s", variant)
	}
	config := footerBaseConfig
	config.ElementID = overrides.ElementID
	config.InnerElementID = overrides.InnerElementID
	config.BaseClass = overrides.BaseClass
	config.ToggleButtonID = overrides.ToggleButtonID
	config.ThemeToggleEnabled = overrides.ThemeToggleEnabled
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
