#!/usr/bin/env python3
from pathlib import Path
import re
import sys


repository_root = Path(__file__).resolve().parent.parent
workflow_source = (repository_root / ".github/workflows/ci.yml").read_text(encoding="utf-8")
dependabot_source = (repository_root / ".github/dependabot.yml").read_text(encoding="utf-8")


def require(condition: bool, message: str) -> None:
    if not condition:
        raise ValueError(message)


try:
    require(
        re.search(r"(?m)^permissions:\n  contents: read$", workflow_source) is not None,
        "github_workflow_audit_failed: top-level permissions must grant only contents read",
    )
    require(
        re.search(r"(?m)^\s+[a-z_-]+: write(?:\s|$)", workflow_source) is None,
        "github_workflow_audit_failed: workflow must not grant write permissions",
    )
    require(
        workflow_source.count("      - '.github/**'") == 2,
        "github_workflow_audit_failed: every CI event must include all GitHub control files",
    )
    require(
        workflow_source.count("      - 'Dockerfile'") == 2,
        "github_workflow_audit_failed: every CI event must include the production Dockerfile",
    )

    action_uses = re.findall(r"(?m)^\s+uses:\s+([^@\s]+)@([^\s#]+)", workflow_source)
    require(len(action_uses) == 4, "github_workflow_audit_failed: expected four declared actions")
    approved_actions = {
        "actions/checkout",
        "actions/setup-go",
        "actions/setup-node",
        "actions/setup-python",
    }
    require(
        {action for action, _ in action_uses} == approved_actions,
        "github_workflow_audit_failed: workflow uses an unapproved action",
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
