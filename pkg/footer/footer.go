package footer

import (
	"bytes"
	"html/template"
)

// Link describes a navigation entry displayed inside the footer dropdown.
type Link struct {
	Label string
	URL   string
}

// Config captures the markup and style hooks required to render the footer.
type Config struct {
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
	Links             []Link
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
)

// Render returns the footer HTML for the provided configuration.
func Render(config Config) (template.HTML, error) {
	var buffer bytes.Buffer
	if err := footerTemplate.Execute(&buffer, config); err != nil {
		return "", err
	}
	return template.HTML(buffer.String()), nil
}
