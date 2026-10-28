#!/usr/bin/env python3
from pathlib import Path
from collections import Counter
import re
import sys


repository_root = Path(__file__).resolve().parent.parent
workflow_source = (repository_root / ".github/workflows/ci.yml").read_text(encoding="utf-8")
dependabot_source = (repository_root / ".github/dependabot.yml").read_text(encoding="utf-8")
makefile_source = (repository_root / "Makefile").read_text(encoding="utf-8")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


try:
    ci_target = re.search(r"(?m)^ci: (.+)$", makefile_source)
    if ci_target is None:
        raise ValueError("github_workflow_audit_failed: missing canonical CI target")
    require(
        set(re.findall(r"(?m)^\s+target: (ci-[a-z-]+)$", workflow_source)) == set(ci_target[1].split()),
        "github_workflow_audit_failed: matrix must cover every canonical CI target",
    )
    require(
        re.findall(r"(?m)^\s+shard: (\d+/\d+)$", workflow_source) == ["1/3", "2/3", "3/3"],
        "github_workflow_audit_failed: integration shards must cover all three partitions",
    )
    require(
        re.search(r"(?m)^    strategy:\n      fail-fast: true\n      matrix:$", workflow_source) is not None,
        "github_workflow_audit_failed: matrix must stop sibling jobs after failure",
    )
    for contract in (
        "LOOPAWARE_TEST_SHARD: ${{ matrix.shard }}",
        "run: make ${{ matrix.target }}",
        "  test:\n    if: always()\n    needs: checks\n",
        "CHECKS_RESULT: ${{ needs.checks.result }}",
        'run: test "$CHECKS_RESULT" = success',
        "GOCACHE: ${{ github.workspace }}/.cache/go-build",
        "NPM_CONFIG_CACHE: ${{ github.workspace }}/.cache/npm",
        "cache: gradle",
        "mobile/prepared/package-lock.json",
    ):
        require(contract in workflow_source, f"github_workflow_audit_failed: missing CI contract {contract}")
    require(
        re.search(r"(?m)^permissions:\n  contents: read$", workflow_source) is not None,
        "github_workflow_audit_failed: top-level permissions must grant only contents read",
    )
    require(
        re.search(r"(?m)^\s+[a-z_-]+: write(?:\s|$)", workflow_source) is None,
        "github_workflow_audit_failed: workflow must not grant write permissions",
    )
    require(
        re.search(
            r"(?m)^  pull_request:\n    branches:\n      - master\n(?=\n\S|\Z)",
            workflow_source,
        ) is not None,
        "github_workflow_audit_failed: required CI must run for every pull request to master without path or event filters",
    )
    require(
        workflow_source.count("      - '.github/**'") == 1,
        "github_workflow_audit_failed: push CI must include all GitHub control files",
    )
    require(
        workflow_source.count("      - 'Dockerfile'") == 1,
        "github_workflow_audit_failed: push CI must include the production Dockerfile",
    )

    uv_step = (
        "      - name: Install uv for deployment policy checks\n"
        "        if: matrix.target == 'ci-mobile'\n"
        "        run: |\n"
        "          python -m pip install --disable-pip-version-check uv==0.12.19\n"
        "          uv --version\n"
    )
    require(
        workflow_source.count(uv_step) == 1,
        "github_workflow_audit_failed: install pinned uv for the deployment policy checks",
    )
    require(
        workflow_source.index("      - name: Set up Python for Gateway installation")
        < workflow_source.index(uv_step) < workflow_source.index("      - name: Run checks"),
        "github_workflow_audit_failed: install uv after Python and before checks",
    )
    require(
        "ci-mobile: mobile-release-check deployment-npm-visibility-check" in makefile_source,
        "github_workflow_audit_failed: Mobile CI must include npm visibility policy checks",
    )

    for contract in (
        "ci-checks: first-failure-node-check first-failure-go-check ",
        "deployment-npm-visibility-check: first-failure-pytest-check",
        "ci-integration: first-failure-browser-check ",
        "first-failure-browser-check: tests-browser-install",
    ):
        require(contract in makefile_source, f"github_workflow_audit_failed: missing first-failure CI path {contract}")

    bundletool_step = (
        "      - name: Install bundletool for Mobile checks\n"
        "        if: matrix.target == 'ci-mobile'\n"
        "        run: make mobile-ci-bundletool\n"
    )
    require(
        workflow_source.count(bundletool_step) == 1,
        "github_workflow_audit_failed: install bundletool for the Mobile job",
    )
    require(
        workflow_source.index("      - name: Set up Java") < workflow_source.index(bundletool_step)
        < workflow_source.index("      - name: Run checks"),
        "github_workflow_audit_failed: qualify bundletool after Java and before checks",
    )

    action_uses = re.findall(r"(?m)^\s+uses:\s+([^@\s]+)@([^\s#]+)", workflow_source)
    require(len(action_uses) == 6, "github_workflow_audit_failed: expected six declared actions")
    approved_actions = {
        "actions/checkout",
        "actions/setup-go",
        "actions/setup-node",
        "actions/setup-python",
        "actions/setup-java",
    }
    require(
        {action for action, _ in action_uses} == approved_actions,
        "github_workflow_audit_failed: workflow uses an unapproved action",
    )
    require(
        Counter(action for action, _ in action_uses) == Counter({
            "actions/checkout": 1,
            "actions/setup-go": 1,
            "actions/setup-node": 1,
            "actions/setup-python": 2,
            "actions/setup-java": 1,
        }),
        "github_workflow_audit_failed: workflow action declaration counts differ",
    )
    for action, revision in action_uses:
        require(
            re.fullmatch(r"[0-9a-f]{40}", revision) is not None,
            f"github_workflow_audit_failed: {action} must use an immutable commit SHA",
        )

    require(
        dependabot_source.count("  - package-ecosystem:") == 6,
        "github_workflow_audit_failed: Dependabot must cover all six shipped dependency roots",
    )
    for contract in (
        "package-ecosystem: gomod",
        "package-ecosystem: docker",
        "package-ecosystem: github-actions",
        "directory: /tests",
        "directory: /clients/react-native",
        "directory: /mobile",
    ):
        require(contract in dependabot_source, f"github_workflow_audit_failed: missing Dependabot contract {contract}")
except ValueError as error:
    print(error, file=sys.stderr)
    raise SystemExit(1) from error
