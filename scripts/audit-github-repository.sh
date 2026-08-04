#!/usr/bin/env bash
set -euo pipefail

api_version="2026-03-10"
repository="$(gh repo view --json nameWithOwner --jq '.nameWithOwner')"
repository_api="repos/${repository}"

fail() {
  echo "github_repository_audit_failed: $1" >&2
  exit 1
}

assert_json() {
  local document="$1"
  local expression="$2"
  local message="$3"
  jq -e "${expression}" >/dev/null <<<"${document}" || fail "${message}"
}

actions_permissions="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/actions/permissions")"
assert_json "${actions_permissions}" '.enabled == true and .allowed_actions == "selected" and .sha_pinning_required == true' "Actions must allow only selected, full-SHA-pinned workflows"

selected_actions="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/actions/permissions/selected-actions")"
assert_json "${selected_actions}" '.github_owned_allowed == true and .verified_allowed == false and (.patterns_allowed | length) == 0' "only GitHub-owned actions may run"

workflow_permissions="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/actions/permissions/workflow")"
assert_json "${workflow_permissions}" '.default_workflow_permissions == "read" and .can_approve_pull_request_reviews == false' "workflow tokens must default to read-only and may not approve pull requests"

repository_settings="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}")"
assert_json "${repository_settings}" '.security_and_analysis.dependabot_security_updates.status == "enabled" and .security_and_analysis.secret_scanning.status == "enabled" and .security_and_analysis.secret_scanning_push_protection.status == "enabled"' "Dependabot security updates, secret scanning, and push protection must be enabled"

gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/vulnerability-alerts" >/dev/null
automated_fixes="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/automated-security-fixes")"
assert_json "${automated_fixes}" '.enabled == true and .paused == false' "Dependabot automated security fixes must be enabled and active"

codeql="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/code-scanning/default-setup")"
assert_json "${codeql}" '.state == "configured" and .query_suite == "extended" and .threat_model == "remote"' "CodeQL extended default setup must be configured for the remote threat model"
assert_json "${codeql}" '([.languages[]] | sort) == (["actions", "go", "javascript-typescript", "python"] | sort)' "CodeQL must analyze Actions, Go, JavaScript/TypeScript, and Python"

master_protection="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/branches/master/protection")"
assert_json "${master_protection}" '.required_status_checks.strict == true and .required_status_checks.contexts == ["test"] and .required_pull_request_reviews.required_approving_review_count == 1 and .required_pull_request_reviews.dismiss_stale_reviews == true and .required_pull_request_reviews.require_last_push_approval == true and .required_linear_history.enabled == true and .required_conversation_resolution.enabled == true and .allow_force_pushes.enabled == false and .allow_deletions.enabled == false' "master protection must require current CI, review, linear history, resolved conversations, and safe updates"

pages_protection="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/branches/gh-pages/protection")"
assert_json "${pages_protection}" '.enforce_admins.enabled == true and .required_status_checks == null and .required_pull_request_reviews == null and .required_linear_history.enabled == true and .allow_force_pushes.enabled == true and .allow_deletions.enabled == false' "gh-pages must prevent deletion while permitting only the gateway-required force-with-lease activation contract"

pages="$(gh api -H "X-GitHub-Api-Version: ${api_version}" "${repository_api}/pages")"
assert_json "${pages}" '.build_type == "legacy" and .source.branch == "gh-pages" and .source.path == "/" and .https_enforced == true and .cname == "loopaware.mprlab.com"' "Pages must remain the HTTPS-enforced legacy gh-pages resource owned by the gateway"

echo "github-security-audit OK"
