"""Exercise LoopAware's publication policy through the installed Gateway action."""

from __future__ import annotations

import importlib.util
import json
import os
from collections.abc import Callable
from datetime import datetime, timedelta, timezone
from pathlib import Path
from types import ModuleType
from unittest.mock import Mock

import pytest
import yaml
from ansible.parsing.dataloader import DataLoader
from ansible.playbook.play_context import PlayContext
from ansible.playbook.task import Task
from ansible.template import Templar


MANIFEST_PATH = Path(__file__).resolve().parents[1] / "resources.yml"
RESOURCE = "react-native-client"
VERIFY_OPERATION = "app_publish.npm.version_verify"
CAPTURE_VARIABLE = "MPRLAB_LISTENER_EXECUTION_POLICY"


class Clock:
    """Advance simulated time without a wall-clock wait."""

    def __init__(self) -> None:
        self.elapsed = 0.0

    def monotonic(self) -> float:
        return self.elapsed

    def sleep(self, seconds: float) -> None:
        self.elapsed += seconds


@pytest.fixture(scope="module")
def gateway() -> tuple[ModuleType, Path]:
    executable = Path(os.environ["MPRLAB_GATEWAY_EXECUTABLE"]).resolve(strict=True)
    ansible_root = executable.parents[1] / "deploy/ansible"
    spec = importlib.util.spec_from_file_location(
        "npm_visibility_action", ansible_root / "action_plugins/mprlab_network_operation.py"
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module, ansible_root


def normalized_policy(raw: dict, contract: ModuleType) -> dict:
    """Adapt YAML durations to the controller capture format, without policy resolution."""
    result = {}
    if "timeouts" in raw:
        result["timeouts"] = {
            name: contract.mprlab_go_duration_nanoseconds(value)
            for name, value in raw["timeouts"].items()
        }
    if "polling" in raw:
        polling = raw["polling"]
        result["polling"] = {
            name + "_ns": 0 if polling[name] == "0s" else contract.mprlab_go_duration_nanoseconds(polling[name])
            for name in ("initial", "maximum", "increment")
        } | {name: polling[name] for name in ("multiplier", "attempt_limit")}
    if "adapter" in raw:
        result["adapter"] = raw["adapter"]
    return result


@pytest.fixture
def capture(gateway: tuple[ModuleType, Path], tmp_path: Path) -> dict:
    action_module, _ = gateway
    manifest = yaml.safe_load(MANIFEST_PATH.read_text())["mprlab_resources"]
    contract = action_module._contract
    execution = {
        "defaults": normalized_policy(manifest["defaults"], contract),
        "operations": {name: normalized_policy(value, contract) for name, value in manifest["operations"].items()},
        "resources": {
            name: {operation: normalized_policy(value, contract) for operation, value in resource.get("operations", {}).items()}
            for name, resource in manifest["resources"].items()
        },
    }
    capture_path = tmp_path / "mprlab-captured-manifest-test.json"
    capture_path.write_text(json.dumps({"schema_version": 1, "manifest": manifest, "execution": execution}))
    return {"schema_version": 1, "path": str(capture_path), "operation": "app-publish", "resource": RESOURCE}


@pytest.fixture
def observe(gateway: tuple[ModuleType, Path], capture: dict, monkeypatch: pytest.MonkeyPatch) -> Callable:
    action_module, ansible_root = gateway
    manifest = action_module._contract.mprlab_captured_manifest(json.dumps(capture))
    retry_policy = yaml.safe_load((ansible_root / "playbooks/vars/app-lifecycle-contract.yml").read_text())["mprlab_network_retry_policy"]
    resource = manifest["resources"][RESOURCE]
    command = [os.environ["MPRLAB_GATEWAY_EXECUTABLE"], "owned-command-operation", "--", "npm", "view",
               resource["package"] + "@0.0.0-test", "--json", "--registry", resource["registry"]]

    def run(*, ready_after: float = 78, error: str = "E404", operation: str = VERIFY_OPERATION,
            parent_seconds: float | None = None) -> tuple[dict, Clock, list[float]]:
        descriptor = capture.copy()
        if parent_seconds is not None:
            descriptor["parent_deadline"] = (datetime.now(timezone.utc) + timedelta(seconds=parent_seconds)).isoformat()
        monkeypatch.setenv(CAPTURE_VARIABLE, json.dumps(descriptor))
        clock = Clock()
        calls = []

        def provider(*, module_name: str, module_args: dict, **_: object) -> dict:
            assert module_name == "ansible.builtin.command"
            assert module_args == {"argv": command}
            calls.append(clock.monotonic())
            if clock.monotonic() < ready_after:
                return {"rc": 1, "failed": True, "stderr": "npm error code " + error}
            return {"rc": 0, "stdout": '[{"version":"0.0.0-test"}]'}

        loader = DataLoader()
        task = Task.load({"action": "mprlab_network_operation", "args": {
            "operation": operation, "classification": "retry_safe", "module": "ansible.builtin.command",
            "module_args": {"argv": command},
        }}, loader=loader)
        connection = Mock()
        connection._shell.tmpdir = None
        action = action_module.ActionModule(task, connection, PlayContext(), loader, Templar(loader=loader), Mock(), clock=clock)
        monkeypatch.setattr(action, "_execute_module", provider)
        result = action.run(task_vars={"mprlab_network_retry_policy": retry_policy})
        return result, clock, calls

    return run


def test_accepted_version_becomes_readable_after_processing(observe: Callable) -> None:
    result, _, calls = observe()
    assert result["rc"] == 0, result
    assert len(calls) > 4
    assert 78 <= calls[-1] < 88


def test_absent_version_stops_at_publication_deadline(observe: Callable) -> None:
    result, clock, calls = observe(ready_after=float("inf"))
    assert result["failed"]
    assert "deadline_exhausted" in result.get("msg", ""), result
    assert 4 * 60 * 60 - 10 <= clock.elapsed < 4 * 60 * 60
    assert calls[-1] == clock.elapsed


def test_parent_deadline_stops_observation_earlier(observe: Callable) -> None:
    result, clock, calls = observe(parent_seconds=20)
    assert result["failed"]
    assert "deadline_exhausted" in result.get("msg", ""), result
    assert clock.elapsed < 20
    assert calls[-1] == clock.elapsed


@pytest.mark.parametrize("error", ["E401", "E403"])
def test_authentication_errors_stop_immediately(observe: Callable, error: str) -> None:
    result, clock, calls = observe(error=error)
    assert result["rc"] == 1
    assert error in result["stderr"]
    assert calls == [0]
    assert clock.elapsed == 0


def test_initial_version_read_keeps_the_existing_attempt_limit(observe: Callable) -> None:
    result, _, calls = observe(operation="app_publish.npm.version_read", error="connection reset")
    assert result["rc"] == 1
    assert len(calls) == 4


def test_publish_mutation_keeps_the_existing_retry_policy(gateway: tuple[ModuleType, Path], capture: dict) -> None:
    action_module, _ = gateway
    contract = action_module._contract
    policy = contract.mprlab_execution_policy(json.dumps(capture), "app_publish.npm.publish")
    plan = action_module._retry.compile_plan(contract.mprlab_execution_polling(policy))
    assert [entry["delay_seconds"] for entry in plan.entries(limit=5)] == [0, 2, 5, 10]

