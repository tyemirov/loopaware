#!/usr/bin/env bash

assert_canonical_github_url() {
  local label="$1"
  local repository="$2"
  local url="$3"
  case "${url}" in
    "git@github.com:${repository}"|"git@github.com:${repository}.git"|"https://github.com/${repository}"|"https://github.com/${repository}.git") ;;
    *) echo "error: ${label} must resolve to the canonical GitHub repository ${repository}, got ${url:-<missing>}" >&2; return 1 ;;
  esac
}

assert_canonical_github_origin() {
  local directory="$1"
  local label="$2"
  local repository="$3"
  local origin_urls
  origin_urls="$(git -C "${directory}" config --get-all remote.origin.url)"
  local origin_url_count
  origin_url_count="$(printf '%s\n' "${origin_urls}" | sed '/^$/d' | wc -l | tr -d ' ')"
  [[ "${origin_url_count}" == "1" ]] || {
    echo "error: ${label} origin must have exactly one fetch URL" >&2
    return 1
  }
  local origin_url="${origin_urls}"
  assert_canonical_github_url "${label} origin fetch URL" "${repository}" "${origin_url}"
  local push_urls
  push_urls="$(git -C "${directory}" config --get-all remote.origin.pushurl || true)"
  [[ -z "${push_urls}" ]] || {
    echo "error: ${label} origin pushurl overrides are not supported by the canonical lifecycle" >&2
    return 1
  }

  local effective_fetch_urls
  effective_fetch_urls="$(git -C "${directory}" remote get-url --all origin)"
  local effective_fetch_url_count
  effective_fetch_url_count="$(printf '%s\n' "${effective_fetch_urls}" | sed '/^$/d' | wc -l | tr -d ' ')"
  [[ "${effective_fetch_url_count}" == "1" ]] || {
    echo "error: ${label} origin must resolve to exactly one effective fetch URL" >&2
    return 1
  }
  assert_canonical_github_url "${label} effective origin fetch URL" "${repository}" "${effective_fetch_urls}"

  local effective_push_urls
  effective_push_urls="$(git -C "${directory}" remote get-url --push --all origin)"
  local effective_push_url_count
  effective_push_url_count="$(printf '%s\n' "${effective_push_urls}" | sed '/^$/d' | wc -l | tr -d ' ')"
  [[ "${effective_push_url_count}" == "1" ]] || {
    echo "error: ${label} origin must resolve to exactly one effective push URL" >&2
    return 1
  }
  assert_canonical_github_url "${label} effective origin push URL" "${repository}" "${effective_push_urls}"
}

assert_no_github_repository_override() {
  [[ -z "${GH_REPO:-}" ]] || {
    echo "error: GH_REPO is not supported by the canonical lifecycle; GitHub operations are pinned explicitly" >&2
    return 1
  }
  [[ -z "${GH_HOST:-}" || "${GH_HOST}" == "github.com" ]] || {
    echo "error: GH_HOST must be github.com for the canonical lifecycle" >&2
    return 1
  }
}

assert_remote_default_and_release_tags() {
  local directory="$1"
  local label="$2"
  local mode="${3:-strict}"
  [[ "${mode}" == "strict" || "${mode}" == "allow-prepared-release" ]] || {
    echo "error: unsupported remote release-state mode: ${mode}" >&2
    return 1
  }
  local remote_refs
  remote_refs="$(git -C "${directory}" ls-remote --symref origin)"
  local remote_default_ref
  local remote_default_sha
  remote_default_ref="$(awk '$1 == "ref:" && $3 == "HEAD" { print $2; exit }' <<<"${remote_refs}")"
  remote_default_sha="$(awk -v ref="${remote_default_ref}" 'NF == 2 && $2 == ref { print $1; exit }' <<<"${remote_refs}")"
  [[ -n "${remote_default_ref}" && -n "${remote_default_sha}" ]] || {
    echo "error: ${label} origin default branch could not be resolved" >&2
    return 1
  }
  local expected_branch="${remote_default_ref#refs/heads/}"
  local current_branch
  local local_head
  current_branch="$(git -C "${directory}" branch --show-current)"
  local_head="$(git -C "${directory}" rev-parse HEAD)"
  [[ "${current_branch}" == "${expected_branch}" ]] || {
    echo "error: ${label} release must run from ${expected_branch}, got ${current_branch:-detached HEAD}" >&2
    return 1
  }
  local remote_tag_names
  remote_tag_names="$(awk '$2 ~ /^refs\/tags\/v/ { name=$2; sub(/\^\{\}$/, "", name); print name }' <<<"${remote_refs}" | LC_ALL=C sort -u)"

  local pending_local_tag=""
  if [[ "${mode}" == "allow-prepared-release" ]]; then
    local pending_parent_line
    local pending_parent_values=()
    pending_parent_line="$(git -C "${directory}" rev-list --parents -n 1 HEAD)"
    read -r -a pending_parent_values <<<"${pending_parent_line}"
    if [[ "${local_head}" == "${remote_default_sha}" \
      || ( "${#pending_parent_values[@]}" -eq 2 && "${pending_parent_values[1]}" == "${remote_default_sha}" ) ]]; then
      local pending_head_tags=()
      local pending_head_tag
      while IFS= read -r pending_head_tag; do
        [[ -n "${pending_head_tag}" ]] || continue
        if [[ "${pending_head_tag}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
          pending_head_tags+=("${pending_head_tag}")
        fi
      done < <(git -C "${directory}" tag --points-at HEAD --list 'v*' --sort=version:refname)
      if [[ "${#pending_head_tags[@]}" -eq 1 ]] && ! grep -Fxq "refs/tags/${pending_head_tags[0]}" <<<"${remote_tag_names}"; then
        local pending_subject
        local pending_changed_files
        pending_subject="$(git -C "${directory}" log -1 --format=%s HEAD)"
        pending_changed_files="$(git -C "${directory}" diff-tree --no-commit-id --name-only -r HEAD)"
        if [[ "${pending_subject}" == "Release ${pending_head_tags[0]}" \
          && "${pending_changed_files}" == "CHANGELOG.md" \
          && "$(git -C "${directory}" cat-file -t "refs/tags/${pending_head_tags[0]}")" == "tag" ]]; then
          pending_local_tag="${pending_head_tags[0]}"
        fi
      fi
    fi
  fi

  local local_tag
  while IFS= read -r local_tag; do
    [[ -n "${local_tag}" ]] || continue
    [[ "${local_tag}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || continue
    if ! grep -Fxq "refs/tags/${local_tag}" <<<"${remote_tag_names}" && [[ "${local_tag}" != "${pending_local_tag}" ]]; then
      git -C "${directory}" tag --delete "${local_tag}" >/dev/null || {
        echo "error: ${label} could not discard unpublished local release tag ${local_tag}" >&2
        return 1
      }
    fi
  done < <(git -C "${directory}" tag --list 'v*' --sort=version:refname)

  local remote_tag_refspecs=()
  local remote_tag_ref
  while IFS= read -r remote_tag_ref; do
    [[ -n "${remote_tag_ref}" ]] || continue
    local tag_name="${remote_tag_ref#refs/tags/}"
    [[ "${tag_name}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || continue
    remote_tag_refspecs+=("+${remote_tag_ref}:${remote_tag_ref}")
  done <<<"${remote_tag_names}"
  if [[ "${#remote_tag_refspecs[@]}" -gt 0 ]]; then
    git -C "${directory}" fetch --force --no-tags --no-write-fetch-head origin "${remote_tag_refspecs[@]}" >/dev/null 2>&1 || {
      echo "error: ${label} could not synchronize remote release tags" >&2
      return 1
    }
  fi

  while IFS= read -r remote_tag_ref; do
    [[ -n "${remote_tag_ref}" ]] || continue
    local tag_name="${remote_tag_ref#refs/tags/}"
    [[ "${tag_name}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || continue
    local remote_tag_commit
    local local_tag_commit
    remote_tag_commit="$(awk -v direct="${remote_tag_ref}" -v peeled="${remote_tag_ref}^{}" '
      $2 == peeled { peeled_sha=$1 }
      $2 == direct { direct_sha=$1 }
      END { if (peeled_sha != "") print peeled_sha; else print direct_sha }
    ' <<<"${remote_refs}")"
    local_tag_commit="$(git -C "${directory}" rev-list -n 1 "${remote_tag_ref}")"
    [[ "${local_tag_commit}" == "${remote_tag_commit}" ]] || {
      echo "error: ${label} local release tag ${tag_name} differs from origin" >&2
      return 1
    }
  done <<<"${remote_tag_names}"

  local extra_local_tags=""
  while IFS= read -r local_tag; do
    [[ -n "${local_tag}" ]] || continue
    [[ "${local_tag}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]] || continue
    if ! grep -Fxq "refs/tags/${local_tag}" <<<"${remote_tag_names}"; then
      extra_local_tags+="${local_tag}"$'\n'
    fi
  done < <(git -C "${directory}" tag --list 'v*' --sort=version:refname)

  if [[ "${local_head}" == "${remote_default_sha}" ]]; then
    if [[ -z "${extra_local_tags}" || "${extra_local_tags}" == "${pending_local_tag}"$'\n' ]]; then
      return 0
    fi
    {
      echo "error: ${label} has local release tags that are not published on origin" >&2
      printf '%s' "${extra_local_tags}" >&2
      return 1
    }
  fi

  if [[ "${mode}" == "allow-prepared-release" ]]; then
    local parent_line
    local parent_values=()
    parent_line="$(git -C "${directory}" rev-list --parents -n 1 HEAD)"
    read -r -a parent_values <<<"${parent_line}"
    local head_stable_tags=()
    while IFS= read -r local_tag; do
      [[ -n "${local_tag}" ]] || continue
      if [[ "${local_tag}" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
        head_stable_tags+=("${local_tag}")
      fi
    done < <(git -C "${directory}" tag --points-at HEAD --list 'v*' --sort=version:refname)
    local extra_local_tag_count
    extra_local_tag_count="$(printf '%s' "${extra_local_tags}" | sed '/^$/d' | wc -l | tr -d ' ')"
    if [[ "${#parent_values[@]}" -eq 2 \
      && "${parent_values[1]}" == "${remote_default_sha}" \
      && "${#head_stable_tags[@]}" -eq 1 \
      && "${extra_local_tag_count}" == "1" \
      && "${extra_local_tags}" == "${head_stable_tags[0]}"$'\n' ]]; then
      return 0
    fi
  fi

  echo "error: ${label} HEAD does not match origin/${expected_branch}, and it is not the one exact locally prepared release" >&2
  return 1
}
