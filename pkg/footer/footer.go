package footer

import (
	"bytes"
	"encoding/json"
	"fmt"
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
	ElementID               string
	InnerElementID          string
	BaseClass               string
	InnerClass              string
	WrapperClass            string
	BrandWrapperClass       string
	MenuWrapperClass        string
	PrefixClass             string
	PrefixText              string
	ToggleButtonID          string
	ToggleButtonClass       string
	ToggleLabel             string
	MenuClass               string
	MenuItemClass           string
	PrivacyLinkClass        string
	PrivacyLinkHref         string
	PrivacyLinkLabel        string
	LeadingHTML             template.HTML
	Links                   []Link
	ThemeToggleEnabled      bool
	ThemeToggleID           string
	ThemeToggleWrapperClass string
	ThemeToggleInputClass   string
	ThemeToggleDataTheme    string
	ThemeToggleAriaLabel    string
}

var footerTemplate = template.Must(template.New("footer").Parse(`<footer id="{{.ElementID}}" class="{{.BaseClass}}" data-mpr-footer="root" data-mpr-footer-config='{{.ComponentJSON}}' x-data="mprFooter()" x-init="init(JSON.parse($el.getAttribute('data-mpr-footer-config')))">
  <div id="{{.InnerElementID}}" class="{{.InnerClass}}" data-mpr-footer="inner">
    <div class="{{.WrapperClass}}" data-mpr-footer="layout">
      <a class="{{.PrivacyLinkClass}}" data-mpr-footer="privacy-link" href="{{.PrivacyLinkHref}}">{{.PrivacyLinkLabel}}</a>
      <div class="{{.BrandWrapperClass}}" data-mpr-footer="brand">{{.LeadingHTML}}
        <span class="{{.PrefixClass}}" data-mpr-footer="prefix">{{.PrefixText}}</span>
        <div class="{{.MenuWrapperClass}}" data-mpr-footer="menu-wrapper">
          <button id="{{.ToggleButtonID}}" class="{{.ToggleButtonClass}}" data-mpr-footer="toggle-button" type="button" data-bs-toggle="dropdown" aria-expanded="false">{{.ToggleLabel}}</button>
          <ul class="{{.MenuClass}}" data-mpr-footer="menu" aria-labelledby="{{.ToggleButtonID}}">
            {{range .Links}}
            <li><a class="{{$.MenuItemClass}}" data-mpr-footer="menu-link" href="{{.URL}}" target="{{if .Target}}{{.Target}}{{else}}_blank{{end}}" rel="{{if .Rel}}{{.Rel}}{{else}}noopener noreferrer{{end}}">{{.Label}}</a></li>
            {{end}}
          </ul>
        </div>
      </div>
    </div>
  </div>
</footer>`))

// Render returns the footer HTML for the provided configuration.
func Render(config Config) (template.HTML, error) {
	normalizedConfig := config

	if normalizedConfig.ThemeToggleEnabled && normalizedConfig.ThemeToggleID != "" {
		normalizedConfig.LeadingHTML = template.HTML(fmt.Sprintf(
			`<div class="%s" data-mpr-footer="theme-toggle" data-bs-theme="%s"><input class="%s" type="checkbox" id="%s" aria-label="%s" data-mpr-footer="theme-toggle-input" /></div>`,
			normalizedConfig.ThemeToggleWrapperClass,
			normalizedConfig.ThemeToggleDataTheme,
			normalizedConfig.ThemeToggleInputClass,
			normalizedConfig.ThemeToggleID,
			normalizedConfig.ThemeToggleAriaLabel,
		))
	} else {
		normalizedConfig.LeadingHTML = template.HTML("")
	}

	componentDescriptor := struct {
		ElementID         string `json:"elementId"`
		InnerElementID    string `json:"innerElementId"`
		BaseClass         string `json:"baseClass"`
		InnerClass        string `json:"innerClass"`
		WrapperClass      string `json:"wrapperClass"`
		BrandWrapperClass string `json:"brandWrapperClass"`
		MenuWrapperClass  string `json:"menuWrapperClass"`
		PrefixClass       string `json:"prefixClass"`
		PrefixText        string `json:"prefixText"`
		ToggleButtonID    string `json:"toggleButtonId"`
		ToggleButtonClass string `json:"toggleButtonClass"`
		ToggleLabel       string `json:"toggleLabel"`
		MenuClass         string `json:"menuClass"`
		MenuItemClass     string `json:"menuItemClass"`
		PrivacyLinkClass  string `json:"privacyLinkClass"`
		PrivacyLinkHref   string `json:"privacyLinkHref"`
		PrivacyLinkLabel  string `json:"privacyLinkLabel"`
		ThemeToggle       struct {
			Enabled      bool   `json:"enabled"`
			WrapperClass string `json:"wrapperClass"`
			InputClass   string `json:"inputClass"`
			DataTheme    string `json:"dataTheme"`
			InputID      string `json:"inputId"`
			AriaLabel    string `json:"ariaLabel"`
		} `json:"themeToggle"`
		Links []Link `json:"links"`
	}{
		ElementID:         normalizedConfig.ElementID,
		InnerElementID:    normalizedConfig.InnerElementID,
		BaseClass:         normalizedConfig.BaseClass,
		InnerClass:        normalizedConfig.InnerClass,
		WrapperClass:      normalizedConfig.WrapperClass,
		BrandWrapperClass: normalizedConfig.BrandWrapperClass,
		MenuWrapperClass:  normalizedConfig.MenuWrapperClass,
		PrefixClass:       normalizedConfig.PrefixClass,
		PrefixText:        normalizedConfig.PrefixText,
		ToggleButtonID:    normalizedConfig.ToggleButtonID,
		ToggleButtonClass: normalizedConfig.ToggleButtonClass,
		ToggleLabel:       normalizedConfig.ToggleLabel,
		MenuClass:         normalizedConfig.MenuClass,
		MenuItemClass:     normalizedConfig.MenuItemClass,
		PrivacyLinkClass:  normalizedConfig.PrivacyLinkClass,
		PrivacyLinkHref:   normalizedConfig.PrivacyLinkHref,
		PrivacyLinkLabel:  normalizedConfig.PrivacyLinkLabel,
		Links:             normalizedConfig.Links,
	}

	componentDescriptor.ThemeToggle.Enabled = normalizedConfig.ThemeToggleEnabled
	componentDescriptor.ThemeToggle.WrapperClass = normalizedConfig.ThemeToggleWrapperClass
	componentDescriptor.ThemeToggle.InputClass = normalizedConfig.ThemeToggleInputClass
	componentDescriptor.ThemeToggle.DataTheme = normalizedConfig.ThemeToggleDataTheme
	componentDescriptor.ThemeToggle.InputID = normalizedConfig.ThemeToggleID
	componentDescriptor.ThemeToggle.AriaLabel = normalizedConfig.ThemeToggleAriaLabel

	componentJSON, marshalErr := json.Marshal(componentDescriptor)
	if marshalErr != nil {
		return "", marshalErr
	}

	renderConfig := struct {
		Config
		ComponentJSON string
	}{
		Config:        normalizedConfig,
		ComponentJSON: string(componentJSON),
	}

	var buffer bytes.Buffer
	if err := footerTemplate.Execute(&buffer, renderConfig); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}
