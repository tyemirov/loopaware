package httpapi

import (
	"bytes"
	"html/template"
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

type FooterLink struct {
	Label string
	URL   string
}

type FooterConfig struct {
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
}

var (
	footerTemplate = template.Must(template.New("footer").Parse(`<footer id="{{.ElementID}}" class="{{.BaseClass}}">
  <div id="{{.InnerElementID}}" class="{{.InnerClass}}">
    <div class="{{.WrapperClass}}">
      <a class="{{.PrivacyLinkClass}}" href="{{.PrivacyLinkHref}}">{{.PrivacyLinkLabel}}</a>
      <div class="{{.BrandWrapperClass}}">
        <span class="{{.PrefixClass}}">{{.PrefixText}}</span>
        <div class="{{.MenuWrapperClass}}">
          <button id="{{.ToggleButtonID}}" class="{{.ToggleButtonClass}}" type="button" data-bs-toggle="dropdown" aria-expanded="false">{{.ToggleLabel}}</button>
          <ul class="{{.MenuClass}}" aria-labelledby="{{.ToggleButtonID}}">
            {{range .Links}}
            <li><a class="{{$.MenuItemClass}}" href="{{.URL}}" target="_blank" rel="noopener noreferrer">{{.Label}}</a></li>
            {{end}}
          </ul>
        </div>
      </div>
    </div>
  </div>
</footer>`))
	footerLinks = []FooterLink{
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
)

type footerTemplatePayload struct {
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
	Links             []FooterLink
}

func RenderFooterHTML(config FooterConfig) (template.HTML, error) {
	payload := footerTemplatePayload{
		ElementID:         config.ElementID,
		InnerElementID:    config.InnerElementID,
		BaseClass:         config.BaseClass,
		InnerClass:        config.InnerClass,
		WrapperClass:      config.WrapperClass,
		BrandWrapperClass: config.BrandWrapperClass,
		MenuWrapperClass:  config.MenuWrapperClass,
		PrefixClass:       config.PrefixClass,
		PrefixText:        config.PrefixText,
		ToggleButtonID:    config.ToggleButtonID,
		ToggleButtonClass: config.ToggleButtonClass,
		ToggleLabel:       config.ToggleLabel,
		MenuClass:         config.MenuClass,
		MenuItemClass:     config.MenuItemClass,
		PrivacyLinkClass:  config.PrivacyLinkClass,
		PrivacyLinkHref:   footerPrivacyLinkHref,
		PrivacyLinkLabel:  footerPrivacyLinkLabel,
		Links:             footerLinks,
	}
	var buffer bytes.Buffer
	if err := footerTemplate.Execute(&buffer, payload); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}
