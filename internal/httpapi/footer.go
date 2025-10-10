package httpapi

import (
	"bytes"
	"fmt"
	"html/template"
)

const (
	footerTemplateName               = "footer"
	footerDropupWrapperClass         = "dropup d-inline-flex align-items-center gap-2"
	footerPrefixTextClass            = "text-muted"
	footerToggleButtonClass          = "btn btn-link dropdown-toggle text-decoration-none p-0 align-baseline"
	footerToggleButtonRole           = "button"
	footerToggleButtonAriaExpanded   = "false"
	footerMenuClass                  = "dropdown-menu dropdown-menu-end shadow"
	footerMenuRole                   = "menu"
	footerMenuItemClass              = "dropdown-item"
	footerMenuItemRole               = "menuitem"
	footerLinkTargetAttribute        = "_blank"
	footerLinkRelAttribute           = "noopener noreferrer"
	footerPlaceholderToken           = "__FOOTER_COMPONENT__"
	footerMenuID                     = "footer-products-menu"
	footerToggleButtonID             = "footer-products-toggle"
	footerInnerContainerDefaultClass = "container d-flex justify-content-end text-end small"
	footerBrandPrimaryLabel          = "Marco Polo Research Lab"
	footerGravityNotesLabel          = "Gravity Notes"
	footerLoopAwareLabel             = "LoopAware"
	footerAllergyWheelLabel          = "Allergy Wheel"
	footerSocialThreaderLabel        = "Social Threader"
	footerRSVPLabel                  = "RSVP"
	footerCountdownCalendarLabel     = "Countdown Calendar"
	footerLLMCrosswordLabel          = "LLM Crossword"
	footerPromptBubblesLabel         = "Prompt Bubbles"
	footerWallpapersLabel            = "Wallpapers"
	footerGravityNotesURL            = "https://gravity.mprlab.com"
	footerLoopAwareURL               = "https://loopaware.mprlab.com"
	footerAllergyWheelURL            = "https://allergy.mprlab.com"
	footerSocialThreaderURL          = "https://threader.mprlab.com"
	footerRSVPURL                    = "https://rsvp.mprlab.com"
	footerCountdownCalendarURL       = "https://countdown.mprlab.com"
	footerLLMCrosswordURL            = "https://llm-crossword.mprlab.com"
	footerPromptBubblesURL           = "https://prompts.mprlab.com"
	footerWallpapersURL              = "https://wallpapers.mprlab.com"
)

const footerTemplateMarkup = `<footer id="{{.ContainerID}}" class="{{.BaseClass}}">
  <div id="{{.InnerContainerID}}" class="{{.InnerContainerClass}}">
    <div class="{{.DropupClass}}">
      <span{{if .PrefixClass}} class="{{.PrefixClass}}"{{end}}>{{.Prefix}} <button id="{{.ToggleID}}" class="{{.ToggleClass}}" type="button" data-bs-toggle="dropdown" aria-expanded="{{.ToggleAriaExpanded}}" role="{{.ToggleRole}}" aria-controls="{{.MenuID}}">{{.TriggerLabel}}</button></span>
      <ul id="{{.MenuID}}" class="{{.MenuClass}}" aria-labelledby="{{.ToggleID}}" role="{{.MenuRole}}">
        {{range .Links}}
        <li><a class="{{$.ItemClass}}" href="{{.URL}}" target="{{$.LinkTarget}}" rel="{{$.LinkRel}}" role="{{$.MenuItemRole}}">{{.Label}}</a></li>
        {{end}}
      </ul>
    </div>
  </div>
</footer>`

// FooterLink describes a link rendered inside the footer dropdown.
type FooterLink struct {
	Label string
	URL   string
}

// FooterConfig captures the configurable attributes for rendering the shared footer component.
type FooterConfig struct {
	ContainerID         string
	InnerContainerID    string
	BaseClass           string
	InnerContainerClass string
	DropupClass         string
	Prefix              string
	PrefixClass         string
	TriggerLabel        string
	ToggleID            string
	ToggleClass         string
	ToggleRole          string
	ToggleAriaExpanded  string
	MenuID              string
	MenuClass           string
	MenuRole            string
	ItemClass           string
	MenuItemRole        string
	LinkTarget          string
	LinkRel             string
	Links               []FooterLink
}

var footerTemplate = template.Must(template.New(footerTemplateName).Parse(footerTemplateMarkup))

// RenderFooterHTML renders the footer component using the provided configuration and returns the resulting HTML.
func RenderFooterHTML(config FooterConfig) (string, error) {
	if config.ContainerID == "" {
		return "", fmt.Errorf("footer container id is required")
	}
	if config.InnerContainerID == "" {
		return "", fmt.Errorf("footer inner container id is required")
	}
	if config.BaseClass == "" {
		return "", fmt.Errorf("footer base class is required")
	}
	if config.InnerContainerClass == "" {
		return "", fmt.Errorf("footer inner container class is required")
	}
	if config.DropupClass == "" {
		return "", fmt.Errorf("footer dropup class is required")
	}
	if config.Prefix == "" {
		return "", fmt.Errorf("footer prefix text is required")
	}
	if config.TriggerLabel == "" {
		return "", fmt.Errorf("footer trigger label is required")
	}
	if config.ToggleID == "" {
		return "", fmt.Errorf("footer toggle id is required")
	}
	if config.ToggleClass == "" {
		return "", fmt.Errorf("footer toggle class is required")
	}
	if config.ToggleRole == "" {
		return "", fmt.Errorf("footer toggle role is required")
	}
	if config.ToggleAriaExpanded == "" {
		return "", fmt.Errorf("footer aria expanded value is required")
	}
	if config.MenuID == "" {
		return "", fmt.Errorf("footer menu id is required")
	}
	if config.MenuClass == "" {
		return "", fmt.Errorf("footer menu class is required")
	}
	if config.MenuRole == "" {
		return "", fmt.Errorf("footer menu role is required")
	}
	if config.ItemClass == "" {
		return "", fmt.Errorf("footer item class is required")
	}
	if config.MenuItemRole == "" {
		return "", fmt.Errorf("footer menu item role is required")
	}
	if config.LinkTarget == "" {
		return "", fmt.Errorf("footer link target is required")
	}
	if config.LinkRel == "" {
		return "", fmt.Errorf("footer link rel is required")
	}
	if len(config.Links) == 0 {
		return "", fmt.Errorf("footer links are required")
	}

	var buffer bytes.Buffer
	executeErr := footerTemplate.Execute(&buffer, config)
	if executeErr != nil {
		return "", fmt.Errorf("render footer template: %w", executeErr)
	}
	return buffer.String(), nil
}

// defaultFooterConfig returns the standard configuration used across LoopAware surfaces.
func defaultFooterConfig(containerID string, innerContainerID string, baseClass string) FooterConfig {
	return FooterConfig{
		ContainerID:         containerID,
		InnerContainerID:    innerContainerID,
		BaseClass:           baseClass,
		InnerContainerClass: footerInnerContainerDefaultClass,
		DropupClass:         footerDropupWrapperClass,
		Prefix:              dashboardFooterBrandPrefix,
		PrefixClass:         footerPrefixTextClass,
		TriggerLabel:        footerBrandPrimaryLabel,
		ToggleID:            footerToggleButtonID,
		ToggleClass:         footerToggleButtonClass,
		ToggleRole:          footerToggleButtonRole,
		ToggleAriaExpanded:  footerToggleButtonAriaExpanded,
		MenuID:              footerMenuID,
		MenuClass:           footerMenuClass,
		MenuRole:            footerMenuRole,
		ItemClass:           footerMenuItemClass,
		MenuItemRole:        footerMenuItemRole,
		LinkTarget:          footerLinkTargetAttribute,
		LinkRel:             footerLinkRelAttribute,
		Links: []FooterLink{
			{
				Label: footerBrandPrimaryLabel,
				URL:   dashboardFooterBrandURL,
			},
			{
				Label: footerGravityNotesLabel,
				URL:   footerGravityNotesURL,
			},
			{
				Label: footerLoopAwareLabel,
				URL:   footerLoopAwareURL,
			},
			{
				Label: footerAllergyWheelLabel,
				URL:   footerAllergyWheelURL,
			},
			{
				Label: footerSocialThreaderLabel,
				URL:   footerSocialThreaderURL,
			},
			{
				Label: footerRSVPLabel,
				URL:   footerRSVPURL,
			},
			{
				Label: footerCountdownCalendarLabel,
				URL:   footerCountdownCalendarURL,
			},
			{
				Label: footerLLMCrosswordLabel,
				URL:   footerLLMCrosswordURL,
			},
			{
				Label: footerPromptBubblesLabel,
				URL:   footerPromptBubblesURL,
			},
			{
				Label: footerWallpapersLabel,
				URL:   footerWallpapersURL,
			},
		},
	}
}
