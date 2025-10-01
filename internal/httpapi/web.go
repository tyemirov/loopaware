package httpapi

import (
	"bytes"
	"html/template"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/temirov/GAuss/pkg/constants"
	"go.uber.org/zap"
)

const (
	dashboardTemplateName           = "dashboard"
	dashboardHTMLContentType        = "text/html; charset=utf-8"
	dashboardPageTitle              = "LoopAware Dashboard"
	dashboardStatusLoadingUser      = "Loading account information..."
	dashboardStatusLoadingSites     = "Loading sites..."
	dashboardStatusLoadFailed       = "Failed to load data."
	dashboardStatusSavingSite       = "Saving site..."
	dashboardStatusSiteSaved        = "Site updated."
	dashboardStatusCreatingSite     = "Creating site..."
	dashboardStatusSiteCreated      = "Site created."
	dashboardStatusSelectSite       = "Select a site to see details."
	dashboardStatusNoMessages       = "No feedback yet."
	dashboardStatusNoSites          = "No sites available yet."
	dashboardRoleAdminLabel         = "Administrator"
	dashboardRoleUserLabel          = "User"
	dashboardFeedbackPlaceholder    = "Select a site to load feedback."
	userNameElementID               = "user-name"
	userEmailElementID              = "user-email"
	userRoleBadgeElementID          = "user-role"
	userAvatarElementID             = "user-avatar"
	statusBannerElementID           = "status-banner"
	siteSelectorElementID           = "site-selector"
	emptySitesMessageElementID      = "empty-sites-message"
	createSiteCardElementID         = "create-site-card"
	createSiteNameInputElementID    = "create-site-name"
	createSiteOriginInputElementID  = "create-site-origin"
	createSiteOwnerInputElementID   = "create-site-owner"
	createSiteButtonElementID       = "create-site-button"
	siteFormElementID               = "site-form"
	editSiteNameInputElementID      = "edit-site-name"
	editSiteOriginInputElementID    = "edit-site-origin"
	editSiteOwnerContainerElementID = "edit-site-owner-container"
	editSiteOwnerInputElementID     = "edit-site-owner"
	saveSiteButtonElementID         = "save-site-button"
	refreshMessagesButtonElementID  = "refresh-messages-button"
	feedbackTableBodyElementID      = "feedback-table-body"
	logoutButtonElementID           = "logout-button"
)

const dashboardTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>{{.PageTitle}}</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous" />
  </head>
  <body class="d-flex flex-column min-vh-100 bg-light">
    <header class="navbar navbar-expand-lg navbar-dark bg-primary fixed-top shadow-sm">
      <div class="container-fluid">
        <span class="navbar-brand fw-semibold">{{.PageTitle}}</span>
        <button id="{{.LogoutButtonID}}" class="btn btn-outline-light">Logout</button>
      </div>
    </header>
    <main class="flex-grow-1 pt-5 mt-4">
      <div class="container py-4">
        <div id="{{.StatusBannerID}}" class="alert alert-info d-none" role="alert"></div>
        <div class="row g-4">
          <div class="col-lg-4">
            <div class="card shadow-sm">
              <div class="card-header">Account</div>
              <div class="card-body">
                <div class="d-flex align-items-center">
                  <img id="{{.UserAvatarID}}" src="" alt="Avatar" class="rounded-circle me-3 d-none" width="64" height="64" />
                  <div>
                    <div id="{{.UserNameID}}" class="fw-semibold"></div>
                    <div id="{{.UserEmailID}}" class="text-muted small"></div>
                    <span id="{{.UserRoleBadgeID}}" class="badge bg-secondary mt-2"></span>
                  </div>
                </div>
              </div>
            </div>
            <div class="card shadow-sm mt-4">
              <div class="card-header">Sites</div>
              <div class="card-body">
                <label class="form-label" for="{{.SiteSelectorID}}">Select site</label>
                <select id="{{.SiteSelectorID}}" class="form-select"></select>
                <p class="text-muted small mt-3" id="{{.EmptySitesMessageID}}">{{.EmptySitesMessage}}</p>
                <div id="{{.CreateSiteCardID}}" class="mt-4 d-none">
                  <h6>Create site</h6>
                  <div class="mb-3">
                    <label class="form-label" for="{{.CreateSiteNameInputID}}">Name</label>
                    <input id="{{.CreateSiteNameInputID}}" type="text" class="form-control" autocomplete="off" placeholder="Acme Marketing" />
                  </div>
                  <div class="mb-3">
                    <label class="form-label" for="{{.CreateSiteOriginInputID}}">Allowed origin</label>
                    <input id="{{.CreateSiteOriginInputID}}" type="text" class="form-control" autocomplete="off" placeholder="https://example.com" />
                  </div>
                  <div class="mb-3">
                    <label class="form-label" for="{{.CreateSiteOwnerInputID}}">Owner email</label>
                    <input id="{{.CreateSiteOwnerInputID}}" type="email" class="form-control" autocomplete="off" placeholder="owner@example.com" />
                  </div>
                  <button id="{{.CreateSiteButtonID}}" class="btn btn-primary w-100">Create site</button>
                </div>
              </div>
            </div>
          </div>
          <div class="col-lg-8">
            <div class="card shadow-sm mb-4">
              <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="mb-0">Site details</h5>
                <button id="{{.RefreshMessagesButtonID}}" class="btn btn-outline-secondary btn-sm">Refresh feedback</button>
              </div>
              <div class="card-body">
                <form id="{{.SiteFormID}}">
                  <div class="row g-3">
                    <div class="col-md-6">
                      <label class="form-label" for="{{.EditSiteNameInputID}}">Name</label>
                      <input id="{{.EditSiteNameInputID}}" type="text" class="form-control" autocomplete="off" />
                    </div>
                    <div class="col-md-6">
                      <label class="form-label" for="{{.EditSiteOriginInputID}}">Allowed origin</label>
                      <input id="{{.EditSiteOriginInputID}}" type="text" class="form-control" autocomplete="off" />
                    </div>
                    <div class="col-md-6 d-none" id="{{.EditSiteOwnerContainerID}}">
                      <label class="form-label" for="{{.EditSiteOwnerInputID}}">Owner email</label>
                      <input id="{{.EditSiteOwnerInputID}}" type="email" class="form-control" autocomplete="off" />
                    </div>
                  </div>
                  <div class="d-flex justify-content-end mt-3">
                    <button id="{{.SaveSiteButtonID}}" type="submit" class="btn btn-success">Save changes</button>
                  </div>
                </form>
              </div>
            </div>
            <div class="card shadow-sm">
              <div class="card-header">Feedback messages</div>
              <div class="card-body">
                <div class="table-responsive">
                  <table class="table table-striped table-hover align-middle">
                    <thead class="table-light">
                      <tr>
                        <th scope="col">When</th>
                        <th scope="col">Contact</th>
                        <th scope="col">Message</th>
                      </tr>
                    </thead>
                    <tbody id="{{.FeedbackTableBodyID}}">
                      <tr><td colspan="3" class="text-center text-muted">{{.FeedbackPlaceholder}}</td></tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </main>
    <footer class="bg-white mt-auto py-3 fixed-bottom border-top">
      <div class="container text-center text-muted small">
        LoopAware © {{.CurrentYear}} · <a class="text-decoration-none" href="{{.LogoutPath}}">Logout</a>
      </div>
    </footer>
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/js/bootstrap.bundle.min.js" integrity="sha384-YvpcrYf0tY3lHB60NNkmXc5s9fDVZLESaAA55NDzOxhy9GkcIdslK1eN7N6jIeHz" crossorigin="anonymous"></script>
    <script>
      (function() {
        var apiMeEndpoint = '{{.APIMeEndpoint}}';
        var apiSitesEndpoint = '{{.APISitesEndpoint}}';
        var apiSiteUpdatePrefix = '{{.APISiteUpdateEndpointPrefix}}';
        var apiSiteMessagesPrefix = '{{.APIMessagesEndpointPrefix}}';
        var apiSiteMessagesSuffix = '{{.APIMessagesEndpointSuffix}}';
        var logoutPath = '{{.LogoutPath}}';
        var loginPath = '{{.LoginPath}}';
        var statusMessages = {
          loadingUser: '{{.StatusLoadingUser}}',
          loadingSites: '{{.StatusLoadingSites}}',
          loadFailed: '{{.StatusLoadFailed}}',
          savingSite: '{{.StatusSavingSite}}',
          siteSaved: '{{.StatusSiteSaved}}',
          creatingSite: '{{.StatusCreatingSite}}',
          siteCreated: '{{.StatusSiteCreated}}',
          selectSite: '{{.StatusSelectSite}}',
          noMessages: '{{.StatusNoMessages}}',
          noSites: '{{.StatusNoSites}}'
        };
        var roleLabels = {
          admin: '{{.RoleAdmin}}',
          user: '{{.RoleUser}}'
        };

        var banner = document.getElementById('{{.StatusBannerID}}');
        var userName = document.getElementById('{{.UserNameID}}');
        var userEmail = document.getElementById('{{.UserEmailID}}');
        var userAvatar = document.getElementById('{{.UserAvatarID}}');
        var userRole = document.getElementById('{{.UserRoleBadgeID}}');
        var siteSelector = document.getElementById('{{.SiteSelectorID}}');
        var emptySitesMessage = document.getElementById('{{.EmptySitesMessageID}}');
        var createSiteCard = document.getElementById('{{.CreateSiteCardID}}');
        var createSiteNameInput = document.getElementById('{{.CreateSiteNameInputID}}');
        var createSiteOriginInput = document.getElementById('{{.CreateSiteOriginInputID}}');
        var createSiteOwnerInput = document.getElementById('{{.CreateSiteOwnerInputID}}');
        var createSiteButton = document.getElementById('{{.CreateSiteButtonID}}');
        var siteForm = document.getElementById('{{.SiteFormID}}');
        var editSiteNameInput = document.getElementById('{{.EditSiteNameInputID}}');
        var editSiteOriginInput = document.getElementById('{{.EditSiteOriginInputID}}');
        var editSiteOwnerContainer = document.getElementById('{{.EditSiteOwnerContainerID}}');
        var editSiteOwnerInput = document.getElementById('{{.EditSiteOwnerInputID}}');
        var saveSiteButton = document.getElementById('{{.SaveSiteButtonID}}');
        var refreshMessagesButton = document.getElementById('{{.RefreshMessagesButtonID}}');
        var feedbackTableBody = document.getElementById('{{.FeedbackTableBodyID}}');
        var logoutButton = document.getElementById('{{.LogoutButtonID}}');

        var state = {
          user: null,
          sites: [],
          selectedSiteId: ''
        };

        function fetchJSON(url, options) {
          var requestOptions = options || {};
          if (!requestOptions.credentials) {
            requestOptions.credentials = 'same-origin';
          }
          return window.fetch(url, requestOptions).then(function(response) {
            if (response.status === 401) {
              window.location.href = loginPath;
              return Promise.reject(new Error('unauthorized'));
            }
            if (response.status === 403) {
              showStatus(statusMessages.loadFailed, 'danger');
              return Promise.reject(new Error('forbidden'));
            }
            if (!response.ok) {
              return response.json().catch(function() {
                return {};
              }).then(function(body) {
                var message = body.error || body.message || statusMessages.loadFailed;
                throw new Error(message);
              });
            }
            return response.json();
          });
        }

        function showStatus(message, variant) {
          banner.textContent = message;
          banner.className = 'alert alert-' + variant;
          banner.classList.remove('d-none');
        }

        function hideStatus() {
          banner.classList.add('d-none');
          banner.textContent = '';
        }

        function updateUserCard() {
          var displayName = state.user.name || state.user.email;
          userName.textContent = displayName || '';
          userEmail.textContent = state.user.email || '';
          if (state.user.picture_url) {
            userAvatar.src = state.user.picture_url;
            userAvatar.classList.remove('d-none');
          }
          var roleLabel = state.user.is_admin ? roleLabels.admin : roleLabels.user;
          userRole.textContent = roleLabel;
          userRole.className = state.user.is_admin ? 'badge bg-warning text-dark mt-2' : 'badge bg-secondary mt-2';
          if (state.user.is_admin) {
            createSiteCard.classList.remove('d-none');
            editSiteOwnerContainer.classList.remove('d-none');
          } else {
            createSiteCard.classList.add('d-none');
            editSiteOwnerContainer.classList.add('d-none');
          }
        }

        function renderSites() {
          siteSelector.innerHTML = '';
          if (!state.sites.length) {
            emptySitesMessage.textContent = statusMessages.noSites;
            siteSelector.disabled = true;
            clearSiteForm();
            renderFeedbackPlaceholder(statusMessages.selectSite);
            return;
          }
          siteSelector.disabled = false;
          emptySitesMessage.textContent = '';
          state.sites.forEach(function(site) {
            var option = document.createElement('option');
            option.value = site.id;
            option.textContent = site.name + ' (' + site.allowed_origin + ')';
            siteSelector.appendChild(option);
          });
          if (state.selectedSiteId) {
            siteSelector.value = state.selectedSiteId;
          } else {
            state.selectedSiteId = state.sites[0].id;
            siteSelector.value = state.selectedSiteId;
          }
          populateSiteForm();
          loadMessages();
        }

        function clearSiteForm() {
          editSiteNameInput.value = '';
          editSiteOriginInput.value = '';
          editSiteOwnerInput.value = '';
          saveSiteButton.disabled = true;
        }

        function populateSiteForm() {
          var site = state.sites.find(function(item) { return item.id === state.selectedSiteId; });
          if (!site) {
            clearSiteForm();
            return;
          }
          editSiteNameInput.value = site.name || '';
          editSiteOriginInput.value = site.allowed_origin || '';
          editSiteOwnerInput.value = site.owner_email || '';
          saveSiteButton.disabled = false;
        }

        function renderFeedbackPlaceholder(message) {
          feedbackTableBody.innerHTML = '';
          var row = document.createElement('tr');
          var cell = document.createElement('td');
          cell.colSpan = 3;
          cell.className = 'text-center text-muted';
          cell.textContent = message;
          row.appendChild(cell);
          feedbackTableBody.appendChild(row);
        }

        function renderMessages(messages) {
          if (!messages.length) {
            renderFeedbackPlaceholder(statusMessages.noMessages);
            return;
          }
          feedbackTableBody.innerHTML = '';
          messages.forEach(function(message) {
            var row = document.createElement('tr');
            var createdCell = document.createElement('td');
            createdCell.textContent = formatTimestamp(message.created_at);
            var contactCell = document.createElement('td');
            contactCell.textContent = message.contact;
            var bodyCell = document.createElement('td');
            bodyCell.textContent = message.message;
            row.appendChild(createdCell);
            row.appendChild(contactCell);
            row.appendChild(bodyCell);
            feedbackTableBody.appendChild(row);
          });
        }

        function formatTimestamp(unixSeconds) {
          if (!unixSeconds) {
            return '';
          }
          var date = new Date(unixSeconds * 1000);
          return date.toLocaleString();
        }

        function loadUser() {
          showStatus(statusMessages.loadingUser, 'info');
          return fetchJSON(apiMeEndpoint).then(function(payload) {
            state.user = payload;
            updateUserCard();
            hideStatus();
          }).catch(function(error) {
            showStatus(error.message || statusMessages.loadFailed, 'danger');
            throw error;
          });
        }

        function loadSites() {
          showStatus(statusMessages.loadingSites, 'info');
          return fetchJSON(apiSitesEndpoint).then(function(payload) {
            state.sites = payload.sites || [];
            renderSites();
            hideStatus();
          }).catch(function(error) {
            showStatus(error.message || statusMessages.loadFailed, 'danger');
          });
        }

        function loadMessages() {
          if (!state.selectedSiteId) {
            renderFeedbackPlaceholder(statusMessages.selectSite);
            return;
          }
          var endpoint = apiSiteMessagesPrefix + state.selectedSiteId + apiSiteMessagesSuffix;
          renderFeedbackPlaceholder('Loading...');
          fetchJSON(endpoint).then(function(payload) {
            renderMessages(payload.messages || []);
          }).catch(function() {
            renderFeedbackPlaceholder(statusMessages.loadFailed);
          });
        }

        function saveSite(event) {
          event.preventDefault();
          if (!state.selectedSiteId) {
            showStatus(statusMessages.selectSite, 'warning');
            return;
          }
          showStatus(statusMessages.savingSite, 'info');
          var body = {
            name: editSiteNameInput.value,
            allowed_origin: editSiteOriginInput.value
          };
          if (state.user.is_admin) {
            body.owner_email = editSiteOwnerInput.value;
          }
          var endpoint = apiSiteUpdatePrefix + state.selectedSiteId;
          fetchJSON(endpoint, {
            method: 'PATCH',
            headers: {
              'Content-Type': 'application/json'
            },
            body: JSON.stringify(body)
          }).then(function(payload) {
            showStatus(statusMessages.siteSaved, 'success');
            var index = state.sites.findIndex(function(item) { return item.id === payload.id; });
            if (index >= 0) {
              state.sites[index] = payload;
            }
            renderSites();
          }).catch(function(error) {
            showStatus(error.message || statusMessages.loadFailed, 'danger');
          });
        }

        function createSite() {
          showStatus(statusMessages.creatingSite, 'info');
          var body = {
            name: createSiteNameInput.value,
            allowed_origin: createSiteOriginInput.value,
            owner_email: createSiteOwnerInput.value
          };
          fetchJSON(apiSitesEndpoint, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json'
            },
            body: JSON.stringify(body)
          }).then(function(payload) {
            showStatus(statusMessages.siteCreated, 'success');
            createSiteNameInput.value = '';
            createSiteOriginInput.value = '';
            createSiteOwnerInput.value = '';
            state.sites.unshift(payload);
            state.selectedSiteId = payload.id;
            renderSites();
          }).catch(function(error) {
            showStatus(error.message || statusMessages.loadFailed, 'danger');
          });
        }

        siteSelector.addEventListener('change', function(event) {
          state.selectedSiteId = event.target.value;
          populateSiteForm();
          loadMessages();
        });

        siteForm.addEventListener('submit', saveSite);
        createSiteButton.addEventListener('click', function(event) {
          event.preventDefault();
          createSite();
        });
        refreshMessagesButton.addEventListener('click', function(event) {
          event.preventDefault();
          loadMessages();
        });
        logoutButton.addEventListener('click', function(event) {
          event.preventDefault();
          window.location.href = logoutPath;
        });

        loadUser().then(loadSites);
      })();
    </script>
  </body>
</html>
`

type dashboardTemplateData struct {
	PageTitle                   string
	APIMeEndpoint               string
	APISitesEndpoint            string
	APISiteUpdateEndpointPrefix string
	APIMessagesEndpointPrefix   string
	APIMessagesEndpointSuffix   string
	LogoutPath                  string
	LoginPath                   string
	StatusLoadingUser           string
	StatusLoadingSites          string
	StatusLoadFailed            string
	StatusSavingSite            string
	StatusSiteSaved             string
	StatusCreatingSite          string
	StatusSiteCreated           string
	StatusSelectSite            string
	StatusNoMessages            string
	StatusNoSites               string
	RoleAdmin                   string
	RoleUser                    string
	EmptySitesMessage           string
	FeedbackPlaceholder         string
	CurrentYear                 int
	StatusBannerID              string
	UserNameID                  string
	UserEmailID                 string
	UserRoleBadgeID             string
	UserAvatarID                string
	SiteSelectorID              string
	EmptySitesMessageID         string
	CreateSiteCardID            string
	CreateSiteNameInputID       string
	CreateSiteOriginInputID     string
	CreateSiteOwnerInputID      string
	CreateSiteButtonID          string
	SiteFormID                  string
	EditSiteNameInputID         string
	EditSiteOriginInputID       string
	EditSiteOwnerContainerID    string
	EditSiteOwnerInputID        string
	SaveSiteButtonID            string
	RefreshMessagesButtonID     string
	FeedbackTableBodyID         string
	LogoutButtonID              string
}

// DashboardWebHandlers serves the authenticated dashboard UI.
type DashboardWebHandlers struct {
	logger   *zap.Logger
	template *template.Template
}

func NewDashboardWebHandlers(logger *zap.Logger) *DashboardWebHandlers {
	compiledTemplate := template.Must(template.New(dashboardTemplateName).Parse(dashboardTemplate))
	return &DashboardWebHandlers{
		logger:   logger,
		template: compiledTemplate,
	}
}

func (handlers *DashboardWebHandlers) RenderDashboard(context *gin.Context) {
	data := dashboardTemplateData{
		PageTitle:                   dashboardPageTitle,
		APIMeEndpoint:               "/api/me",
		APISitesEndpoint:            "/api/sites",
		APISiteUpdateEndpointPrefix: "/api/sites/",
		APIMessagesEndpointPrefix:   "/api/sites/",
		APIMessagesEndpointSuffix:   "/messages",
		LogoutPath:                  constants.LogoutPath,
		LoginPath:                   constants.LoginPath,
		StatusLoadingUser:           dashboardStatusLoadingUser,
		StatusLoadingSites:          dashboardStatusLoadingSites,
		StatusLoadFailed:            dashboardStatusLoadFailed,
		StatusSavingSite:            dashboardStatusSavingSite,
		StatusSiteSaved:             dashboardStatusSiteSaved,
		StatusCreatingSite:          dashboardStatusCreatingSite,
		StatusSiteCreated:           dashboardStatusSiteCreated,
		StatusSelectSite:            dashboardStatusSelectSite,
		StatusNoMessages:            dashboardStatusNoMessages,
		StatusNoSites:               dashboardStatusNoSites,
		RoleAdmin:                   dashboardRoleAdminLabel,
		RoleUser:                    dashboardRoleUserLabel,
		EmptySitesMessage:           dashboardStatusNoSites,
		FeedbackPlaceholder:         dashboardFeedbackPlaceholder,
		CurrentYear:                 time.Now().Year(),
		StatusBannerID:              statusBannerElementID,
		UserNameID:                  userNameElementID,
		UserEmailID:                 userEmailElementID,
		UserRoleBadgeID:             userRoleBadgeElementID,
		UserAvatarID:                userAvatarElementID,
		SiteSelectorID:              siteSelectorElementID,
		EmptySitesMessageID:         emptySitesMessageElementID,
		CreateSiteCardID:            createSiteCardElementID,
		CreateSiteNameInputID:       createSiteNameInputElementID,
		CreateSiteOriginInputID:     createSiteOriginInputElementID,
		CreateSiteOwnerInputID:      createSiteOwnerInputElementID,
		CreateSiteButtonID:          createSiteButtonElementID,
		SiteFormID:                  siteFormElementID,
		EditSiteNameInputID:         editSiteNameInputElementID,
		EditSiteOriginInputID:       editSiteOriginInputElementID,
		EditSiteOwnerContainerID:    editSiteOwnerContainerElementID,
		EditSiteOwnerInputID:        editSiteOwnerInputElementID,
		SaveSiteButtonID:            saveSiteButtonElementID,
		RefreshMessagesButtonID:     refreshMessagesButtonElementID,
		FeedbackTableBodyID:         feedbackTableBodyElementID,
		LogoutButtonID:              logoutButtonElementID,
	}

	var buffer bytes.Buffer
	if executeErr := handlers.template.Execute(&buffer, data); executeErr != nil {
		handlers.logger.Error("render_dashboard", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{jsonKeyError: "render_failed"})
		return
	}

	context.Data(http.StatusOK, dashboardHTMLContentType, buffer.Bytes())
}
