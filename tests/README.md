# Tests

## Visual Captures

LoopAware keeps visual capture tooling as an on-demand workflow, not as checked-in dated screenshot dumps.

Current pieces:

- `tests/helpers/screenshot.js`: small helper for creating dated artifact directories and saving page or locator screenshots.
- `tests/scripts/capture-visual.mjs`: manual entry point for capturing screenshots against a running LoopAware instance.
- `tests/artifacts/`: generated output directory for captures. This path is ignored by Git.

### When to use it

Use visual captures when you need a quick UI record while debugging widget, subscribe, traffic, or dashboard rendering changes, or when you want a human-reviewable screenshot before and after a UI change.

### What not to do

- Do not commit ad hoc screenshots under `tests/YYYY-MM-DD/...`.
- Do not treat captures in `tests/artifacts/` as golden baselines unless the repo adopts an explicit snapshot policy.

### Requirements

The target app must already be running and reachable at `LOOPAWARE_BASE_URL` or the default `http://localhost:8090`.

Examples:

```bash
npm --prefix tests run capture:visual -- --path /widget-integration?site_id=demo --name widget-bubble
```

```bash
npm --prefix tests run capture:visual -- --path /widget-integration?site_id=demo --name widget-panel --selector '#mp-feedback-panel'
```

```bash
npm --prefix tests run capture:visual -- --path /app --name dashboard --admin --wait-for '#user-name'
```

### Output

Each run writes PNG files under:

```text
tests/artifacts/YYYY-MM-DD/<label>/<name>.png
```

If `--label` is omitted, the script uses the screenshot name as the directory label.

### Supported flags

- `--path <path>`: open a relative path on `LOOPAWARE_BASE_URL`
- `--url <url>`: open an absolute URL instead of `--path`
- `--name <name>`: required screenshot filename without extension
- `--label <label>`: optional directory label under the dated artifact folder
- `--selector <css>`: capture a specific locator instead of the full page
- `--wait-for <css>`: wait for a selector before capture
- `--delay-ms <ms>`: wait a fixed delay before capture
- `--admin`: add an authenticated admin session cookie before navigation
- `--full-page`: capture the full page instead of the viewport

### Notes

- `--admin` uses the test config from `tests/configs/loopaware.env`, the same way the Playwright integration suite does.
- The script runs headless Chromium through the same Playwright dependency used by the test suite.
