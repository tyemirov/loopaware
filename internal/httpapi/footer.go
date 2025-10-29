package httpapi

import (
	"fmt"
	"html/template"

	"github.com/MarkoPoloResearchLab/loopaware/pkg/footer"
)

const (
	footerPrivacyLinkLabel = "Privacy • Terms"
	footerPrivacyLinkHref  = PrivacyPagePath

	footerLayoutClass       = "footer-layout w-100 d-flex flex-column flex-md-row align-items-start align-items-md-center justify-content-between gap-3"
	footerBrandWrapperClass = "footer-brand d-inline-flex align-items-center gap-2 text-body-secondary small"
	footerMenuWrapperClass  = "footer-menu dropup"
	footerPrivacyLinkClass  = "footer-privacy-link text-body-secondary text-decoration-none small"
	footerPrefixClass       = "text-body-secondary fw-semibold"
	footerToggleButtonClass = "btn btn-link dropdown-toggle text-decoration-none px-0 fw-semibold text-body-secondary"
	footerMenuClass         = "dropdown-menu dropdown-menu-end shadow"
	footerMenuItemClass     = "dropdown-item"
)

type footerVariant string

const (
	footerVariantLanding   footerVariant = "landing"
	footerVariantPrivacy   footerVariant = "privacy"
	footerVariantDashboard footerVariant = "dashboard"
)

type footerVariantOverrides struct {
	ElementID      string
	InnerElementID string
	BaseClass      string
	ToggleButtonID string
	LeadingHTML    template.HTML
}

var (
	footerThemeToggleHTML = template.HTML(fmt.Sprintf(`<div class="footer-theme-toggle form-check form-switch m-0" data-bs-theme="light"><input class="form-check-input" type="checkbox" id="%s" aria-label="Toggle theme" /></div>`, publicThemeToggleID))
	footerBaseConfig      = footer.Config{
		InnerClass:        landingFooterInnerClass,
		WrapperClass:      footerLayoutClass,
		BrandWrapperClass: footerBrandWrapperClass,
		MenuWrapperClass:  footerMenuWrapperClass,
		PrefixClass:       footerPrefixClass,
		PrefixText:        dashboardFooterBrandPrefix,
		ToggleButtonClass: footerToggleButtonClass,
		ToggleLabel:       dashboardFooterBrandName,
		MenuClass:         footerMenuClass,
		MenuItemClass:     footerMenuItemClass,
		PrivacyLinkClass:  footerPrivacyLinkClass,
		PrivacyLinkHref:   footerPrivacyLinkHref,
		PrivacyLinkLabel:  footerPrivacyLinkLabel,
		Links:             footerLinks,
	}
	footerVariantOverridesByKey = map[footerVariant]footerVariantOverrides{
		footerVariantLanding: {
			ElementID:      landingFooterElementID,
			InnerElementID: landingFooterInnerID,
			BaseClass:      landingFooterBaseClass,
			ToggleButtonID: landingFooterToggleID,
			LeadingHTML:    footerThemeToggleHTML,
		},
		footerVariantPrivacy: {
			ElementID:      privacyFooterElementID,
			InnerElementID: privacyFooterInnerID,
			BaseClass:      landingFooterBaseClass,
			ToggleButtonID: dashboardFooterToggleButtonID,
			LeadingHTML:    footerThemeToggleHTML,
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
	config.LeadingHTML = overrides.LeadingHTML
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
