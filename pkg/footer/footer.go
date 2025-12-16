package footer

import (
	"bytes"
	"encoding/json"
	"html/template"
)

// Link describes a navigation entry displayed inside the footer dropdown.
type Link struct {
	Label  string `json:"label"`
	URL    string `json:"url"`
	Rel    string `json:"rel,omitempty"`
	Target string `json:"target,omitempty"`
}

// Config captures the markup and style hooks required to render the footer.
type Config struct {
	HostElementID     string
	ElementID         string
	InnerElementID    string
	BaseClass         string
	InnerClass        string
	WrapperClass      string
	BrandWrapperClass string
	MenuWrapperClass  string
	PrefixClass       string
	PrefixText        string
	ToggleButtonID    string
	ToggleButtonClass string
	ToggleLabel       string
	MenuClass         string
	MenuItemClass     string
	PrivacyLinkClass  string
	PrivacyLinkHref   string
	PrivacyLinkLabel  string
	PrivacyModalHTML  string
	Links             []Link
	Sticky            bool
	Size              string

	ThemeToggleEnabled bool
	ThemeSwitcher      string
	ThemeMode          string
	ThemeAttribute     string
	ThemeAriaLabel     string
	ThemeModes         []string
}

type footerThemeConfig struct {
	Attribute string   `json:"attribute,omitempty"`
	AriaLabel string   `json:"ariaLabel,omitempty"`
	Modes     []string `json:"modes,omitempty"`
}

var footerTemplate = template.Must(template.New("footer").Parse(`<mpr-footer id="{{.HostID}}" element-id="{{.ElementID}}" base-class="{{.BaseClass}}" inner-element-id="{{.InnerElementID}}" inner-class="{{.InnerClass}}" wrapper-class="{{.WrapperClass}}" brand-wrapper-class="{{.BrandWrapperClass}}" menu-wrapper-class="{{.MenuWrapperClass}}" prefix-class="{{.PrefixClass}}" prefix-text="{{.PrefixText}}" toggle-button-id="{{.ToggleButtonID}}" toggle-button-class="{{.ToggleButtonClass}}" toggle-label="{{.ToggleLabel}}" menu-class="{{.MenuClass}}" menu-item-class="{{.MenuItemClass}}" privacy-link-class="{{.PrivacyLinkClass}}" privacy-link-href="{{.PrivacyLinkHref}}" privacy-link-label="{{.PrivacyLinkLabel}}" links='{{.LinksJSON}}' sticky="{{.StickyValue}}"{{if .Size}} size="{{.Size}}"{{end}}{{if .PrivacyModalHTML}} privacy-modal-content="{{.PrivacyModalHTML}}"{{end}}{{if .ThemeToggleEnabled}} theme-switcher="{{.ThemeSwitcher}}" theme-config='{{.ThemeConfigJSON}}'{{if .ThemeMode}} theme-mode="{{.ThemeMode}}"{{end}}{{end}}></mpr-footer>`))

// Render returns the footer HTML for the provided configuration.
func Render(config Config) (template.HTML, error) {
	linksJSON, marshalErr := json.Marshal(config.Links)
	if marshalErr != nil {
		return "", marshalErr
	}

	stickyValue := "true"
	if !config.Sticky {
		stickyValue = "false"
	}

	themeSwitcher := config.ThemeSwitcher
	if config.ThemeToggleEnabled && themeSwitcher == "" {
		themeSwitcher = "toggle"
	}

	themeConfigJSON := "{}"
	if config.ThemeToggleEnabled {
		themeConfig, jsonErr := json.Marshal(footerThemeConfig{
			Attribute: config.ThemeAttribute,
			AriaLabel: config.ThemeAriaLabel,
			Modes:     config.ThemeModes,
		})
		if jsonErr != nil {
			return "", jsonErr
		}
		themeConfigJSON = string(themeConfig)
	}

	renderConfig := struct {
		HostID             string
		ElementID          string
		BaseClass          string
		InnerElementID     string
		InnerClass         string
		WrapperClass       string
		BrandWrapperClass  string
		MenuWrapperClass   string
		PrefixClass        string
		PrefixText         string
		ToggleButtonID     string
		ToggleButtonClass  string
		ToggleLabel        string
		MenuClass          string
		MenuItemClass      string
		PrivacyLinkClass   string
		PrivacyLinkHref    string
		PrivacyLinkLabel   string
		PrivacyModalHTML   string
		LinksJSON          string
		StickyValue        string
		Size               string
		ThemeToggleEnabled bool
		ThemeSwitcher      string
		ThemeConfigJSON    string
		ThemeMode          string
	}{
		HostID:             config.HostElementID,
		ElementID:          config.ElementID,
		BaseClass:          config.BaseClass,
		InnerElementID:     config.InnerElementID,
		InnerClass:         config.InnerClass,
		WrapperClass:       config.WrapperClass,
		BrandWrapperClass:  config.BrandWrapperClass,
		MenuWrapperClass:   config.MenuWrapperClass,
		PrefixClass:        config.PrefixClass,
		PrefixText:         config.PrefixText,
		ToggleButtonID:     config.ToggleButtonID,
		ToggleButtonClass:  config.ToggleButtonClass,
		ToggleLabel:        config.ToggleLabel,
		MenuClass:          config.MenuClass,
		MenuItemClass:      config.MenuItemClass,
		PrivacyLinkClass:   config.PrivacyLinkClass,
		PrivacyLinkHref:    config.PrivacyLinkHref,
		PrivacyLinkLabel:   config.PrivacyLinkLabel,
		PrivacyModalHTML:   config.PrivacyModalHTML,
		LinksJSON:          string(linksJSON),
		StickyValue:        stickyValue,
		Size:               config.Size,
		ThemeToggleEnabled: config.ThemeToggleEnabled,
		ThemeSwitcher:      themeSwitcher,
		ThemeConfigJSON:    themeConfigJSON,
		ThemeMode:          config.ThemeMode,
	}

	var buffer bytes.Buffer
	if err := footerTemplate.Execute(&buffer, renderConfig); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}
