package httpapi

import (
	"bytes"
	"html/template"
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	adminTemplateName                  = "admin_page"
	adminHTMLContentType               = "text/html; charset=utf-8"
	adminPageTitle                     = "LoopAware Admin Dashboard"
	adminBearerTokenInputIdentifier    = "admin-bearer-token"
	adminSiteIdentifierInputIdentifier = "admin-site-identifier"
	adminMessagesContainerIdentifier   = "admin-messages"
	adminStatusContainerIdentifier     = "admin-status"
	adminSaveTokenButtonIdentifier     = "save-token-button"
	adminLoadMessagesButtonIdentifier  = "load-messages-button"
	adminMessagesEndpointPrefix        = "/api/admin/sites/"
	adminMessagesEndpointSuffix        = "/messages"
	adminBearerTokenStorageKey         = "loopaware_admin_bearer"
	adminStatusSuccessMessage          = "Messages loaded."
	adminStatusFailureMessage          = "Failed to load messages."
	adminTokenSavedMessage             = "Bearer token saved in browser storage."
	adminMissingTokenMessage           = "Admin bearer token is required."
	adminMissingSiteMessage            = "Site identifier is required."
)

const adminPageTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <title>{{.PageTitle}}</title>
    <style>
      body { font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; margin: 24px; }
      h1 { font-size: 24px; }
      label { display: block; margin-top: 16px; font-weight: 600; }
      input { width: 360px; max-width: 100%; padding: 8px; margin-top: 4px; }
      button { margin-top: 8px; padding: 8px 16px; }
      pre { background: #0f172a; color: #e2e8f0; padding: 16px; border-radius: 8px; overflow: auto; }
      #{{.StatusContainerID}} { margin-top: 16px; font-weight: 600; }
    </style>
  </head>
  <body>
    <h1>{{.PageTitle}}</h1>
    <section>
      <label for="{{.BearerTokenInputID}}">Admin Bearer Token</label>
      <input id="{{.BearerTokenInputID}}" type="password" autocomplete="off" placeholder="Enter bearer token" />
      <div>
        <button id="{{.SaveTokenButtonID}}" type="button">Save Token</button>
      </div>
    </section>
    <section>
      <label for="{{.SiteIdentifierInputID}}">Site Identifier</label>
      <input id="{{.SiteIdentifierInputID}}" type="text" autocomplete="off" placeholder="example-site-id" />
      <div>
        <button id="{{.LoadMessagesButtonID}}" type="button">Load Messages</button>
      </div>
    </section>
    <p id="{{.StatusContainerID}}"></p>
    <pre id="{{.MessagesContainerID}}">Messages will appear here.</pre>
    <script>
      (function() {
        var storageKey = '{{.BearerTokenStorageKey}}';
        var messagesEndpointPrefix = '{{.MessagesEndpointPrefix}}';
        var messagesEndpointSuffix = '{{.MessagesEndpointSuffix}}';
        var statusElement = document.getElementById('{{.StatusContainerID}}');
        var messagesElement = document.getElementById('{{.MessagesContainerID}}');
        var tokenInput = document.getElementById('{{.BearerTokenInputID}}');
        var siteInput = document.getElementById('{{.SiteIdentifierInputID}}');
        var saveTokenButton = document.getElementById('{{.SaveTokenButtonID}}');
        var loadMessagesButton = document.getElementById('{{.LoadMessagesButtonID}}');

        function updateStatus(message, isError) {
          statusElement.textContent = message;
          statusElement.style.color = isError ? '#b91c1c' : '#047857';
        }

        function getBearerToken() {
          var token = tokenInput.value.trim();
          if (!token) {
            token = window.localStorage.getItem(storageKey) || '';
            if (token) {
              tokenInput.value = token;
            }
          }
          if (!token) {
            updateStatus('{{.MissingTokenMessage}}', true);
          }
          return token;
        }

        saveTokenButton.addEventListener('click', function() {
          var token = tokenInput.value.trim();
          if (!token) {
            updateStatus('{{.MissingTokenMessage}}', true);
            return;
          }
          window.localStorage.setItem(storageKey, token);
          updateStatus('{{.TokenSavedMessage}}', false);
        });

        loadMessagesButton.addEventListener('click', function() {
          var token = getBearerToken();
          if (!token) {
            return;
          }
          var siteIdentifier = siteInput.value.trim();
          if (!siteIdentifier) {
            updateStatus('{{.MissingSiteMessage}}', true);
            return;
          }

          var endpoint = messagesEndpointPrefix + encodeURIComponent(siteIdentifier) + messagesEndpointSuffix;
          window.fetch(endpoint, {
            method: 'GET',
            headers: {
              'Authorization': 'Bearer ' + token,
              'Accept': 'application/json'
            }
          }).then(function(response) {
            if (!response.ok) {
              throw new Error('Request failed: ' + response.status);
            }
            return response.json();
          }).then(function(payload) {
            messagesElement.textContent = JSON.stringify(payload, null, 2);
            updateStatus('{{.StatusSuccessMessage}}', false);
            window.localStorage.setItem(storageKey, token);
          }).catch(function(error) {
            console.error(error);
            updateStatus('{{.StatusFailureMessage}}', true);
          });
        });

        var storedToken = window.localStorage.getItem(storageKey);
        if (storedToken) {
          tokenInput.value = storedToken;
        }
      })();
    </script>
  </body>
</html>
`

type adminTemplateData struct {
	PageTitle              string
	BearerTokenInputID     string
	SiteIdentifierInputID  string
	MessagesContainerID    string
	StatusContainerID      string
	SaveTokenButtonID      string
	LoadMessagesButtonID   string
	BearerTokenStorageKey  string
	MessagesEndpointPrefix string
	MessagesEndpointSuffix string
	MissingTokenMessage    string
	MissingSiteMessage     string
	TokenSavedMessage      string
	StatusSuccessMessage   string
	StatusFailureMessage   string
}

// AdminWebHandlers serves the HTML admin interface that drives the API.
type AdminWebHandlers struct {
	logger   *zap.Logger
	template *template.Template
}

// NewAdminWebHandlers builds an AdminWebHandlers instance with the compiled template.
func NewAdminWebHandlers(logger *zap.Logger) *AdminWebHandlers {
	compiledTemplate := template.Must(template.New(adminTemplateName).Parse(adminPageTemplate))
	return &AdminWebHandlers{
		logger:   logger,
		template: compiledTemplate,
	}
}

// RenderAdminInterface responds with the admin HTML page that drives API interactions requiring authentication.
func (adminWebHandlers *AdminWebHandlers) RenderAdminInterface(context *gin.Context) {
	data := adminTemplateData{
		PageTitle:              adminPageTitle,
		BearerTokenInputID:     adminBearerTokenInputIdentifier,
		SiteIdentifierInputID:  adminSiteIdentifierInputIdentifier,
		MessagesContainerID:    adminMessagesContainerIdentifier,
		StatusContainerID:      adminStatusContainerIdentifier,
		SaveTokenButtonID:      adminSaveTokenButtonIdentifier,
		LoadMessagesButtonID:   adminLoadMessagesButtonIdentifier,
		BearerTokenStorageKey:  adminBearerTokenStorageKey,
		MessagesEndpointPrefix: adminMessagesEndpointPrefix,
		MessagesEndpointSuffix: adminMessagesEndpointSuffix,
		MissingTokenMessage:    adminMissingTokenMessage,
		MissingSiteMessage:     adminMissingSiteMessage,
		TokenSavedMessage:      adminTokenSavedMessage,
		StatusSuccessMessage:   adminStatusSuccessMessage,
		StatusFailureMessage:   adminStatusFailureMessage,
	}

	var buffer bytes.Buffer
	if executeErr := adminWebHandlers.template.Execute(&buffer, data); executeErr != nil {
		adminWebHandlers.logger.Error("render_admin_page", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "render_failed"})
		return
	}

	context.Data(http.StatusOK, adminHTMLContentType, buffer.Bytes())
}
