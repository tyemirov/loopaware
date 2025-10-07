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
	dashboardTemplateName            = "dashboard"
	dashboardHTMLContentType         = "text/html; charset=utf-8"
	dashboardPageTitle               = "LoopAware Dashboard"
	dashboardStatusLoadingUser       = "Loading account information..."
	dashboardStatusLoadingSites      = "Loading sites..."
	dashboardStatusLoadFailed        = "Failed to load data."
	dashboardStatusSavingSite        = "Saving site..."
	dashboardStatusSiteSaved         = "Site updated."
	dashboardStatusCreatingSite      = "Creating site..."
	dashboardStatusSiteCreated       = "Site created."
	dashboardStatusSelectSite        = "Select a site to see details."
	dashboardStatusNoMessages        = "No feedback yet."
	dashboardStatusNoSites           = "No sites available yet."
	dashboardRoleAdminLabel          = "Administrator"
	dashboardRoleUserLabel           = "User"
	dashboardFeedbackPlaceholder     = "Select a site to load feedback."
	dashboardWidgetCardTitle         = "Site widget"
	dashboardWidgetInstructions      = "Embed this <script> tag on pages served from the allowed origin."
	dashboardWidgetUnavailable       = "Save the site to generate a widget snippet."
	dashboardStatusWidgetCopied      = "Widget snippet copied."
	dashboardStatusWidgetCopyFailed  = "Unable to copy widget snippet."
	navbarSettingsButtonLabel        = "Account settings"
	navbarLogoutLabel                = "Logout"
	navbarThemeToggleLabel           = "Dark mode"
	newSiteOptionValue               = "__new__"
	newSiteOptionLabel               = "New site"
	siteFormCreateButtonLabel        = "Create site"
	siteFormUpdateButtonLabel        = "Update site"
	siteFormCreateButtonClass        = "btn btn-primary"
	siteFormUpdateButtonClass        = "btn btn-success"
	userNameElementID                = "user-name"
	userEmailElementID               = "user-email"
	userRoleBadgeElementID           = "user-role"
	userAvatarElementID              = "user-avatar"
	statusBannerElementID            = "status-banner"
	siteSelectorElementID            = "site-selector"
	emptySitesMessageElementID       = "empty-sites-message"
	siteFormElementID                = "site-form"
	editSiteNameInputElementID       = "edit-site-name"
	editSiteOriginInputElementID     = "edit-site-origin"
	editSiteOwnerContainerElementID  = "edit-site-owner-container"
	editSiteOwnerInputElementID      = "edit-site-owner"
	saveSiteButtonElementID          = "save-site-button"
	refreshMessagesButtonElementID   = "refresh-messages-button"
	feedbackTableBodyElementID       = "feedback-table-body"
	logoutButtonElementID            = "logout-button"
	widgetSnippetTextareaElementID   = "widget-snippet"
	copyWidgetSnippetButtonElementID = "copy-widget-snippet"
	settingsButtonElementID          = "settings-button"
	settingsMenuElementID            = "settings-menu"
	settingsThemeToggleElementID     = "settings-theme-toggle"
	settingsAvatarImageElementID     = "settings-avatar-image"
	settingsAvatarFallbackElementID  = "settings-avatar-fallback"
	themeStorageKey                  = "loopaware_theme"
)

const dashboardTemplate = `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <title>{{.PageTitle}}</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" integrity="sha384-QWTKZyjpPEjISv5WaRU9OFeRpok6YctnYmDr5pNlyT2bRjXh0JMhjY6hW+ALEwIH" crossorigin="anonymous" />
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css" integrity="sha384-b0OvQu3P5AnbcW2CYwiVdc+GqOR/mdrIW6DCeU44yWiNys8lm2SleqU9jwOpcPfq" crossorigin="anonymous" />
  </head>
  <body class="d-flex flex-column min-vh-100 bg-light">
    <header class="navbar navbar-expand-lg navbar-dark bg-primary fixed-top shadow-sm">
      <div class="container-fluid">
        <span class="navbar-brand fw-semibold">{{.PageTitle}}</span>
        <div class="dropdown ms-auto">
          <button id="{{.SettingsButtonID}}" class="btn btn-outline-light border-0 rounded-circle p-1 d-flex align-items-center justify-content-center" type="button" data-bs-toggle="dropdown" aria-expanded="false">
            <span class="visually-hidden">{{.SettingsButtonLabel}}</span>
            <img id="{{.SettingsAvatarImageID}}" src="" alt="Avatar" class="rounded-circle d-none" width="36" height="36" />
            <i id="{{.SettingsAvatarFallbackID}}" class="bi bi-person-circle fs-3"></i>
          </button>
          <div id="{{.SettingsMenuID}}" class="dropdown-menu dropdown-menu-end">
            <button id="{{.LogoutButtonID}}" class="dropdown-item" type="button">{{.LogoutLabel}}</button>
            <div class="dropdown-item d-flex align-items-center justify-content-between">
              <span>{{.ThemeToggleLabel}}</span>
              <div class="form-check form-switch m-0">
                <input class="form-check-input" type="checkbox" role="switch" id="{{.SettingsThemeToggleID}}" />
              </div>
            </div>
          </div>
        </div>
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
              </div>
            </div>
          </div>
          <div class="col-lg-8">
            <div class="card shadow-sm mb-4">
              <div class="card-header">
                <h5 class="mb-0">Site details</h5>
              </div>
              <div class="card-body">
                <form id="{{.SiteFormID}}">
                  <div class="row g-3 align-items-end">
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
                    <div class="col-md-6 d-flex justify-content-end">
                      <button id="{{.SaveSiteButtonID}}" type="submit" class="btn btn-success">Save changes</button>
                    </div>
                  </div>
                </form>
              </div>
            </div>
            <div class="card shadow-sm mb-4">
              <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="mb-0">{{.WidgetCardTitle}}</h5>
                <button id="{{.CopyWidgetSnippetButtonID}}" type="button" class="btn btn-outline-primary btn-sm">Copy snippet</button>
              </div>
              <div class="card-body">
                <p class="text-muted small mb-3">{{.WidgetInstructions}}</p>
                <textarea id="{{.WidgetSnippetTextareaID}}" class="form-control font-monospace" rows="3" readonly></textarea>
              </div>
            </div>
            <div class="card shadow-sm">
              <div class="card-header d-flex justify-content-between align-items-center">
                <h5 class="mb-0">Feedback messages</h5>
                <button id="{{.RefreshMessagesButtonID}}" class="btn btn-outline-secondary btn-sm">Refresh feedback</button>
              </div>
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
        var themePreferenceStorageKey = '{{.ThemeStorageKey}}';
        var settingsButton = document.getElementById('{{.SettingsButtonID}}');
        var settingsMenu = document.getElementById('{{.SettingsMenuID}}');
        var themeToggle = document.getElementById('{{.SettingsThemeToggleID}}');
        var settingsAvatarImage = document.getElementById('{{.SettingsAvatarImageID}}');
        var settingsAvatarFallback = document.getElementById('{{.SettingsAvatarFallbackID}}');
        var newSiteOptionValue = '{{.NewSiteOptionValue}}';
        var newSiteOptionLabel = '{{.NewSiteOptionLabel}}';
        var createButtonLabel = '{{.CreateButtonLabel}}';
        var updateButtonLabel = '{{.UpdateButtonLabel}}';
        var createButtonClass = '{{.CreateButtonClass}}';
        var updateButtonClass = '{{.UpdateButtonClass}}';
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
          noSites: '{{.StatusNoSites}}',
          widgetCopied: '{{.StatusWidgetCopied}}',
          widgetCopyFailed: '{{.StatusWidgetCopyFailed}}'
        };
        var roleLabels = {
          admin: '{{.RoleAdmin}}',
          user: '{{.RoleUser}}'
        };
        var widgetUnavailableMessage = '{{.WidgetUnavailableMessage}}';

        var banner = document.getElementById('{{.StatusBannerID}}');
        var userName = document.getElementById('{{.UserNameID}}');
        var userEmail = document.getElementById('{{.UserEmailID}}');
        var userAvatar = document.getElementById('{{.UserAvatarID}}');
        var userRole = document.getElementById('{{.UserRoleBadgeID}}');
        var siteSelector = document.getElementById('{{.SiteSelectorID}}');
        var emptySitesMessage = document.getElementById('{{.EmptySitesMessageID}}');
        var siteForm = document.getElementById('{{.SiteFormID}}');
        var editSiteNameInput = document.getElementById('{{.EditSiteNameInputID}}');
        var editSiteOriginInput = document.getElementById('{{.EditSiteOriginInputID}}');
        var editSiteOwnerContainer = document.getElementById('{{.EditSiteOwnerContainerID}}');
        var editSiteOwnerInput = document.getElementById('{{.EditSiteOwnerInputID}}');
        var saveSiteButton = document.getElementById('{{.SaveSiteButtonID}}');
        var refreshMessagesButton = document.getElementById('{{.RefreshMessagesButtonID}}');
        var feedbackTableBody = document.getElementById('{{.FeedbackTableBodyID}}');
        var logoutButton = document.getElementById('{{.LogoutButtonID}}');
        var widgetSnippetTextarea = document.getElementById('{{.WidgetSnippetTextareaID}}');
        var copyWidgetSnippetButton = document.getElementById('{{.CopyWidgetSnippetButtonID}}');

        var state = {
          user: null,
          sites: [],
          selectedSiteId: ''
        };
        var widgetBaseURL = window.location.origin.replace(/\/$/, '');
        widgetSnippetTextarea.value = widgetUnavailableMessage;
        copyWidgetSnippetButton.disabled = true;

        function applyThemePreference(mode) {
          var normalized = mode === 'dark' ? 'dark' : 'light';
          document.documentElement.setAttribute('data-bs-theme', normalized);
          if (normalized === 'dark') {
            document.body.classList.remove('bg-light');
            document.body.classList.add('bg-dark', 'text-light');
          } else {
            document.body.classList.remove('bg-dark', 'text-light');
            if (!document.body.classList.contains('bg-light')) {
              document.body.classList.add('bg-light');
            }
          }
        }

        function loadThemePreference() {
          var stored = localStorage.getItem(themePreferenceStorageKey);
          if (stored !== 'dark' && stored !== 'light') {
            stored = 'light';
          }
          themeToggle.checked = stored === 'dark';
          applyThemePreference(stored);
        }

        function persistThemePreference(mode) {
          localStorage.setItem(themePreferenceStorageKey, mode);
        }

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
            editSiteOwnerContainer.classList.remove('d-none');
          } else {
            editSiteOwnerContainer.classList.add('d-none');
          }

          if (state.user.picture_url) {
            settingsAvatarImage.src = state.user.picture_url;
            settingsAvatarImage.classList.remove('d-none');
            settingsAvatarFallback.classList.add('d-none');
          } else {
            settingsAvatarImage.src = '';
            settingsAvatarImage.classList.add('d-none');
            settingsAvatarFallback.classList.remove('d-none');
          }
        }

        function isNewSiteSelected() {
          return state.selectedSiteId === newSiteOptionValue;
        }

        function buildWidgetSnippet(siteId) {
          return '<script src="' + widgetBaseURL + '/widget.js?site_id=' + siteId + '"></script>';
        }

        function updateWidgetSnippet() {
          if (!state.selectedSiteId || isNewSiteSelected()) {
            widgetSnippetTextarea.value = widgetUnavailableMessage;
            copyWidgetSnippetButton.disabled = true;
            return;
          }
          var site = state.sites.find(function(item) { return item.id === state.selectedSiteId; });
          if (!site) {
            widgetSnippetTextarea.value = widgetUnavailableMessage;
            copyWidgetSnippetButton.disabled = true;
            return;
          }
          widgetSnippetTextarea.value = buildWidgetSnippet(site.id);
          copyWidgetSnippetButton.disabled = false;
        }

        function copyWidgetSnippet() {
          if (copyWidgetSnippetButton.disabled) {
            return;
          }
          var snippet = widgetSnippetTextarea.value;
          if (!snippet || snippet === widgetUnavailableMessage) {
            showStatus(statusMessages.widgetCopyFailed, 'danger');
            return;
          }
          if (navigator.clipboard && navigator.clipboard.writeText) {
            navigator.clipboard.writeText(snippet).then(function() {
              showStatus(statusMessages.widgetCopied, 'success');
            }).catch(function() {
              fallbackCopyWidgetSnippet(snippet);
            });
          } else {
            fallbackCopyWidgetSnippet(snippet);
          }
        }

        function fallbackCopyWidgetSnippet(snippet) {
          var previousSelectionStart = widgetSnippetTextarea.selectionStart;
          var previousSelectionEnd = widgetSnippetTextarea.selectionEnd;
          widgetSnippetTextarea.focus();
          widgetSnippetTextarea.select();
          var copySucceeded = false;
          try {
            copySucceeded = document.execCommand('copy');
          } catch (error) {
            copySucceeded = false;
          }
          widgetSnippetTextarea.selectionStart = previousSelectionStart;
          widgetSnippetTextarea.selectionEnd = previousSelectionEnd;
          widgetSnippetTextarea.blur();
          if (copySucceeded) {
            showStatus(statusMessages.widgetCopied, 'success');
          } else {
            showStatus(statusMessages.widgetCopyFailed, 'danger');
          }
        }

        function renderSites() {
          siteSelector.innerHTML = '';
          var hasSites = state.sites.length > 0;

          var newOption = document.createElement('option');
          newOption.value = newSiteOptionValue;
          newOption.textContent = newSiteOptionLabel;
          siteSelector.appendChild(newOption);

          state.sites.forEach(function(site) {
            var option = document.createElement('option');
            option.value = site.id;
            option.textContent = site.name + ' (' + site.allowed_origin + ')';
            siteSelector.appendChild(option);
          });

          emptySitesMessage.textContent = hasSites ? '' : statusMessages.noSites;
          siteSelector.disabled = false;

          if (!state.selectedSiteId) {
            state.selectedSiteId = hasSites ? state.sites[0].id : newSiteOptionValue;
          }

          var optionExists = false;
          for (var optionIndex = 0; optionIndex < siteSelector.options.length; optionIndex++) {
            if (siteSelector.options[optionIndex].value === state.selectedSiteId) {
              optionExists = true;
              break;
            }
          }
          if (!optionExists) {
            state.selectedSiteId = hasSites ? state.sites[0].id : newSiteOptionValue;
          }

          siteSelector.value = state.selectedSiteId;

          populateSiteForm();
          updateWidgetSnippet();
          if (isNewSiteSelected()) {
            renderFeedbackPlaceholder(statusMessages.selectSite);
          } else {
            loadMessages();
          }
        }

        function clearSiteForm() {
          editSiteNameInput.value = '';
          editSiteOriginInput.value = '';
          editSiteOwnerInput.value = '';
        }

        function populateSiteForm() {
          if (!state.selectedSiteId) {
            clearSiteForm();
            saveSiteButton.disabled = true;
            updateFormMode();
            updateWidgetSnippet();
            return;
          }

          if (isNewSiteSelected()) {
            clearSiteForm();
            saveSiteButton.disabled = false;
            updateFormMode();
            updateWidgetSnippet();
            return;
          }

          var site = state.sites.find(function(item) { return item.id === state.selectedSiteId; });
          if (!site) {
            clearSiteForm();
            saveSiteButton.disabled = true;
            updateFormMode();
            updateWidgetSnippet();
            return;
          }

          editSiteNameInput.value = site.name || '';
          editSiteOriginInput.value = site.allowed_origin || '';
          editSiteOwnerInput.value = site.owner_email || '';
          saveSiteButton.disabled = false;
          updateFormMode();
          updateWidgetSnippet();
        }

        function updateFormMode() {
          if (!state.user) {
            return;
          }
          var isAdmin = state.user.is_admin;
          if (isNewSiteSelected()) {
            saveSiteButton.textContent = createButtonLabel;
            saveSiteButton.className = createButtonClass;
          } else {
            saveSiteButton.textContent = updateButtonLabel;
            saveSiteButton.className = updateButtonClass;
          }
          if (isAdmin) {
            editSiteOwnerInput.disabled = false;
          } else {
            editSiteOwnerInput.disabled = true;
          }
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
          if (!state.selectedSiteId || isNewSiteSelected()) {
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

        function submitSite(event) {
          event.preventDefault();
          if (!state.selectedSiteId) {
            showStatus(statusMessages.selectSite, 'warning');
            return;
          }
          if (isNewSiteSelected()) {
            createSite();
          } else {
            updateSite();
          }
        }

        function updateSite() {
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
            name: editSiteNameInput.value,
            allowed_origin: editSiteOriginInput.value,
            owner_email: editSiteOwnerInput.value
          };
          fetchJSON(apiSitesEndpoint, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json'
            },
            body: JSON.stringify(body)
          }).then(function(payload) {
            showStatus(statusMessages.siteCreated, 'success');
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

        siteForm.addEventListener('submit', submitSite);
        themeToggle.addEventListener('change', function(event) {
          var mode = event.target.checked ? 'dark' : 'light';
          applyThemePreference(mode);
          persistThemePreference(mode);
        });
        copyWidgetSnippetButton.addEventListener('click', function(event) {
          event.preventDefault();
          copyWidgetSnippet();
        });
        refreshMessagesButton.addEventListener('click', function(event) {
          event.preventDefault();
          loadMessages();
        });
        logoutButton.addEventListener('click', function(event) {
          event.preventDefault();
          window.location.href = logoutPath;
        });

        loadThemePreference();
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
	SiteFormID                  string
	EditSiteNameInputID         string
	EditSiteOriginInputID       string
	EditSiteOwnerContainerID    string
	EditSiteOwnerInputID        string
	SaveSiteButtonID            string
	RefreshMessagesButtonID     string
	FeedbackTableBodyID         string
	LogoutButtonID              string
	NewSiteOptionValue          string
	NewSiteOptionLabel          string
	CreateButtonLabel           string
	UpdateButtonLabel           string
	CreateButtonClass           string
	UpdateButtonClass           string
	WidgetCardTitle             string
	WidgetInstructions          string
	WidgetUnavailableMessage    string
	StatusWidgetCopied          string
	StatusWidgetCopyFailed      string
	WidgetSnippetTextareaID     string
	CopyWidgetSnippetButtonID   string
	SettingsButtonID            string
	SettingsButtonLabel         string
	LogoutLabel                 string
	ThemeToggleLabel            string
	SettingsMenuID              string
	SettingsThemeToggleID       string
	ThemeStorageKey             string
	SettingsAvatarImageID       string
	SettingsAvatarFallbackID    string
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
		SiteFormID:                  siteFormElementID,
		EditSiteNameInputID:         editSiteNameInputElementID,
		EditSiteOriginInputID:       editSiteOriginInputElementID,
		EditSiteOwnerContainerID:    editSiteOwnerContainerElementID,
		EditSiteOwnerInputID:        editSiteOwnerInputElementID,
		SaveSiteButtonID:            saveSiteButtonElementID,
		RefreshMessagesButtonID:     refreshMessagesButtonElementID,
		FeedbackTableBodyID:         feedbackTableBodyElementID,
		LogoutButtonID:              logoutButtonElementID,
		NewSiteOptionValue:          newSiteOptionValue,
		NewSiteOptionLabel:          newSiteOptionLabel,
		CreateButtonLabel:           siteFormCreateButtonLabel,
		UpdateButtonLabel:           siteFormUpdateButtonLabel,
		CreateButtonClass:           siteFormCreateButtonClass,
		UpdateButtonClass:           siteFormUpdateButtonClass,
		WidgetCardTitle:             dashboardWidgetCardTitle,
		WidgetInstructions:          dashboardWidgetInstructions,
		WidgetUnavailableMessage:    dashboardWidgetUnavailable,
		StatusWidgetCopied:          dashboardStatusWidgetCopied,
		StatusWidgetCopyFailed:      dashboardStatusWidgetCopyFailed,
		WidgetSnippetTextareaID:     widgetSnippetTextareaElementID,
		CopyWidgetSnippetButtonID:   copyWidgetSnippetButtonElementID,
		SettingsButtonID:            settingsButtonElementID,
		SettingsButtonLabel:         navbarSettingsButtonLabel,
		LogoutLabel:                 navbarLogoutLabel,
		ThemeToggleLabel:            navbarThemeToggleLabel,
		SettingsMenuID:              settingsMenuElementID,
		SettingsThemeToggleID:       settingsThemeToggleElementID,
		ThemeStorageKey:             themeStorageKey,
		SettingsAvatarImageID:       settingsAvatarImageElementID,
		SettingsAvatarFallbackID:    settingsAvatarFallbackElementID,
	}

	var buffer bytes.Buffer
	if executeErr := handlers.template.Execute(&buffer, data); executeErr != nil {
		handlers.logger.Error("render_dashboard", zap.Error(executeErr))
		context.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{jsonKeyError: "render_failed"})
		return
	}

	context.Data(http.StatusOK, dashboardHTMLContentType, buffer.Bytes())
}
