#!/usr/bin/env python3
"""Enforce the canonical static-browser dependency and CSP contract."""

from __future__ import annotations

from html.parser import HTMLParser
from pathlib import Path
import sys


REPOSITORY_ROOT = Path(__file__).resolve().parent.parent
WEB_ROOT = REPOSITORY_ROOT / "web"

MPR_UI_COMMIT = "97ebeb2df518f91af78aafcb6e14b9691fb20694"
MPR_UI_BASE_URL = (
    "https://cdn.jsdelivr.net/gh/MarcoPoloResearchLab/"
    f"mpr-ui@{MPR_UI_COMMIT}"
)
MPR_UI_STYLE_URL = f"{MPR_UI_BASE_URL}/mpr-ui.css"
MPR_UI_CONFIG_URL = f"{MPR_UI_BASE_URL}/mpr-ui-config.js"
MPR_UI_BUNDLE_URL = f"{MPR_UI_BASE_URL}/mpr-ui.js"
JS_YAML_URL = "https://cdn.jsdelivr.net/npm/js-yaml@4.3.0/dist/js-yaml.min.js"

INTEGRITY_BY_URL = {
    MPR_UI_STYLE_URL: (
        "sha384-WWDM4bNAbnG6m8Lda3m59qcrh8OkdoLPBMl+1LDA+IvCrjszwBgdt3CizK3ayn75"
    ),
    MPR_UI_CONFIG_URL: (
        "sha384-pl32+7hu3Trs6rwm8vbTkVbjEWI7C8+MbeHGwFZA+OpU4qiA2RmZArBA3wRhMak7"
    ),
    JS_YAML_URL: (
        "sha384-0zxS50HhMqXyT0WdkhYMK1yK+EpwgVEIHYc1RW1+JgesjsL7Rwqh0WfQSwEDyDH9"
    ),
}

CSP_META_POLICY = (
    REPOSITORY_ROOT / "configs" / "content-security-policy.txt"
).read_text(encoding="utf-8").strip()
CSP_HEADER_POLICY = f"{CSP_META_POLICY}; frame-ancestors 'none'"

COMPOSE_PATHS = (
    REPOSITORY_ROOT / "docker-compose.yml",
    REPOSITORY_ROOT / "docker-compose.computercat.yml",
    REPOSITORY_ROOT / "tests" / "docker-compose.yml",
)


class BrowserAssetParser(HTMLParser):
    """Collect browser-relevant declarations from one HTML entry point."""

    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.elements: list[tuple[str, dict[str, str], int]] = []

    def handle_starttag(
        self, tag: str, attrs: list[tuple[str, str | None]]
    ) -> None:
        normalized = {name: value or "" for name, value in attrs}
        self.elements.append((tag, normalized, self.getpos()[0]))


def matching_elements(
    parser: BrowserAssetParser, tag: str, attribute: str, value: str
) -> list[dict[str, str]]:
    return [
        attrs
        for current_tag, attrs, _line in parser.elements
        if current_tag == tag and attrs.get(attribute) == value
    ]


def require_exact_element(
    failures: list[str],
    relative_path: Path,
    elements: list[dict[str, str]],
    label: str,
    expected: dict[str, str],
) -> None:
    if len(elements) != 1:
        failures.append(
            f"{relative_path}: expected exactly one {label}; found {len(elements)}"
        )
        return
    actual = elements[0]
    for attribute, expected_value in expected.items():
        if actual.get(attribute) != expected_value:
            failures.append(
                f"{relative_path}: {label} {attribute} must be {expected_value!r}"
            )


def audit_html(path: Path, failures: list[str]) -> bool:
    relative_path = path.relative_to(REPOSITORY_ROOT)
    source = path.read_text(encoding="utf-8")
    parser = BrowserAssetParser()
    parser.feed(source)

    csp_elements = [
        (attrs, line)
        for tag, attrs, line in parser.elements
        if tag == "meta"
        and attrs.get("http-equiv", "").lower() == "content-security-policy"
    ]
    if len(csp_elements) != 1:
        failures.append(
            f"{relative_path}: expected exactly one CSP meta declaration; "
            f"found {len(csp_elements)}"
        )
    else:
        csp_attrs, csp_line = csp_elements[0]
        if csp_attrs.get("content") != CSP_META_POLICY:
            failures.append(f"{relative_path}: CSP meta policy does not match the canonical policy")
        if "frame-ancestors" in csp_attrs.get("content", ""):
            failures.append(
                f"{relative_path}: frame-ancestors is ineffective in CSP meta and must remain edge-owned"
            )
        protected_lines = [
            line
            for tag, _attrs, line in parser.elements
            if tag in {"link", "script", "style"}
        ]
        if protected_lines and csp_line >= min(protected_lines):
            failures.append(
                f"{relative_path}: CSP meta must precede every link, script, and style declaration"
            )

    if "@latest" in source:
        failures.append(f"{relative_path}: mutable @latest CDN selector is forbidden")
    if "js-yaml@4.1.0" in source:
        failures.append(f"{relative_path}: vulnerable js-yaml@4.1.0 selector is forbidden")

    for tag, attrs, _line in parser.elements:
        asset_url = ""
        if tag == "script":
            asset_url = attrs.get("src", "")
        elif tag == "link" and attrs.get("rel") == "stylesheet":
            asset_url = attrs.get("href", "")
        if not asset_url.startswith("https://cdn.jsdelivr.net/"):
            continue
        if attrs.get("integrity", "") == "":
            failures.append(f"{relative_path}: jsDelivr asset lacks integrity: {asset_url}")
        if attrs.get("crossorigin", "") != "anonymous":
            failures.append(
                f"{relative_path}: jsDelivr asset lacks crossorigin=anonymous: {asset_url}"
            )

    uses_mpr_ui = "MarcoPoloResearchLab/mpr-ui@" in source
    if not uses_mpr_ui:
        return False

    require_exact_element(
        failures,
        relative_path,
        matching_elements(parser, "link", "id", "mpr-ui-style"),
        "mpr-ui stylesheet",
        {
            "href": MPR_UI_STYLE_URL,
            "integrity": INTEGRITY_BY_URL[MPR_UI_STYLE_URL],
            "crossorigin": "anonymous",
        },
    )
    require_exact_element(
        failures,
        relative_path,
        matching_elements(parser, "script", "src", JS_YAML_URL),
        "js-yaml script",
        {
            "integrity": INTEGRITY_BY_URL[JS_YAML_URL],
            "crossorigin": "anonymous",
        },
    )
    require_exact_element(
        failures,
        relative_path,
        matching_elements(parser, "script", "src", MPR_UI_CONFIG_URL),
        "mpr-ui config script",
        {
            "integrity": INTEGRITY_BY_URL[MPR_UI_CONFIG_URL],
            "crossorigin": "anonymous",
        },
    )
    require_exact_element(
        failures,
        relative_path,
        matching_elements(parser, "script", "id", "mpr-ui-bundle"),
        "mpr-ui bundle declaration",
        {
            "type": "application/json",
            "data-mpr-ui-bundle-src": MPR_UI_BUNDLE_URL,
        },
    )

    permitted_mpr_ui_urls = {
        MPR_UI_STYLE_URL,
        MPR_UI_CONFIG_URL,
        MPR_UI_BUNDLE_URL,
    }
    for token in source.split('"'):
        if "cdn.jsdelivr.net/gh/MarcoPoloResearchLab/mpr-ui@" in token:
            if token not in permitted_mpr_ui_urls:
                failures.append(f"{relative_path}: noncanonical mpr-ui URL: {token}")

    return True


def audit_compose(failures: list[str]) -> None:
    expected = f'"/=Content-Security-Policy:{CSP_HEADER_POLICY}"'
    for path in COMPOSE_PATHS:
        relative_path = path.relative_to(REPOSITORY_ROOT)
        source = path.read_text(encoding="utf-8")
        if source.count(expected) != 1:
            failures.append(
                f"{relative_path}: expected exactly one canonical Content-Security-Policy response header"
            )


def main() -> int:
    failures: list[str] = []
    html_paths = sorted(WEB_ROOT.rglob("*.html"))
    mpr_ui_entry_count = sum(audit_html(path, failures) for path in html_paths)
    audit_compose(failures)

    if failures:
        for failure in failures:
            print(f"browser_asset_audit.failed: {failure}", file=sys.stderr)
        return 1

    print(
        "browser_asset_audit.ok: "
        f"{len(html_paths)} HTML entry points, "
        f"{mpr_ui_entry_count} immutable mpr-ui declarations, "
        f"{len(COMPOSE_PATHS)} aligned proxy policies"
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
