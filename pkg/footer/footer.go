package footer

import (
	"bytes"
	"encoding/json"
	"html/template"
)

// Link describes a navigation entry displayed inside the footer dropdown.
type Link struct {
	Label  string `json:"label"`
	Href   string `json:"href"`
	Rel    string `json:"rel,omitempty"`
	Target string `json:"target,omitempty"`
}

type footerMenuSection struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Mode  string `json:"mode"`
	Links []Link `json:"links"`
}

type footerMenu struct {
	Label     string              `json:"label"`
	Placement string              `json:"placement"`
	Sections  []footerMenuSection `json:"sections"`
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
	PrefixClass       string
	PrefixText        string
	MenuLabel         string
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
	Attribute   string   `json:"attribute,omitempty"`
	AriaLabel   string   `json:"ariaLabel,omitempty"`
	Modes       []string `json:"modes,omitempty"`
	InitialMode string   `json:"initialMode,omitempty"`
}

var footerTemplate = template.Must(template.New("footer").Parse(`<mpr-footer id="{{.HostID}}" element-id="{{.ElementID}}" base-class="{{.BaseClass}}" inner-element-id="{{.InnerElementID}}" inner-class="{{.InnerClass}}" wrapper-class="{{.WrapperClass}}" brand-wrapper-class="{{.BrandWrapperClass}}" prefix-class="{{.PrefixClass}}" prefix-text="{{.PrefixText}}" privacy-link-class="{{.PrivacyLinkClass}}" privacy-link-href="{{.PrivacyLinkHref}}" privacy-link-label="{{.PrivacyLinkLabel}}" menu='{{.MenuJSON}}' sticky="{{.StickyValue}}"{{if .Size}} size="{{.Size}}"{{end}}{{if .PrivacyModalHTML}} privacy-modal-content="{{.PrivacyModalHTML}}"{{end}}{{if .ThemeToggleEnabled}} theme-switcher="{{.ThemeSwitcher}}" theme-config='{{.ThemeConfigJSON}}'{{end}}></mpr-footer>`))

// Render returns the footer HTML for the provided configuration.
func Render(config Config) (template.HTML, error) {
	menuJSON, marshalErr := json.Marshal(footerMenu{
		Label:     config.MenuLabel,
		Placement: "top",
		Sections:  []footerMenuSection{{ID: "mpr-projects", Label: "Projects", Mode: "static", Links: config.Links}},
	})
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
			Attribute:   config.ThemeAttribute,
			AriaLabel:   config.ThemeAriaLabel,
			Modes:       config.ThemeModes,
			InitialMode: config.ThemeMode,
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
		PrefixClass        string
		PrefixText         string
		PrivacyLinkClass   string
		PrivacyLinkHref    string
		PrivacyLinkLabel   string
		PrivacyModalHTML   string
		MenuJSON           string
		StickyValue        string
		Size               string
		ThemeToggleEnabled bool
		ThemeSwitcher      string
		ThemeConfigJSON    string
	}{
		HostID:             config.HostElementID,
		ElementID:          config.ElementID,
		BaseClass:          config.BaseClass,
		InnerElementID:     config.InnerElementID,
		InnerClass:         config.InnerClass,
		WrapperClass:       config.WrapperClass,
		BrandWrapperClass:  config.BrandWrapperClass,
		PrefixClass:        config.PrefixClass,
		PrefixText:         config.PrefixText,
		PrivacyLinkClass:   config.PrivacyLinkClass,
		PrivacyLinkHref:    config.PrivacyLinkHref,
		PrivacyLinkLabel:   config.PrivacyLinkLabel,
		PrivacyModalHTML:   config.PrivacyModalHTML,
		MenuJSON:           string(menuJSON),
		StickyValue:        stickyValue,
		Size:               config.Size,
		ThemeToggleEnabled: config.ThemeToggleEnabled,
		ThemeSwitcher:      themeSwitcher,
		ThemeConfigJSON:    themeConfigJSON,
	}

	var buffer bytes.Buffer
	if err := footerTemplate.Execute(&buffer, renderConfig); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}
