"""Exercise LoopAware's publication policy through the installed Gateway action."""

from __future__ import annotations

import importlib.util
import json
import os
import shutil
import subprocess
import sys
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


MANIFEST_PATH = Path(__file__).resolve().parents[2] / ".mprlab/deploy/resources.yml"
RESOURCE = "react-native-client"
VERIFY_OPERATION = "app_publish.npm.version_verify"
ARCHIVE_OPERATION = "app_publish.npm.archive_read"
CAPTURE_VARIABLE = "MPRLAB_LISTENER_EXECUTION_POLICY"


class Clock:
    """Advance simulated time without a wall-clock wait."""

    def __init__(self) -> None:
        self.elapsed = 0.0

    def monotonic(self) -> float:
        return self.elapsed

    def sleep(self, seconds: float) -> None:
        self.elapsed += seconds


def prepare_synthetic_application(repository: Path, manifest: bytes) -> dict:
    """Write only declared synthetic inputs for an isolated selected application."""
    manifest_path = repository / ".mprlab/deploy/resources.yml"
    manifest_path.parent.mkdir(parents=True)
    manifest_path.write_bytes(manifest)
    selected_manifest = yaml.safe_load(manifest)["mprlab_resources"]
    environment_relative_path = selected_manifest["environment_file"]
    synthetic_environment = repository / environment_relative_path
    synthetic_environment.parent.mkdir(parents=True, exist_ok=True)
    synthetic_environment.write_text("".join(
        f'{name}="synthetic-policy-input"\n' for name in selected_manifest["inputs"]
    ))
    (repository / ".gitignore").write_text("/" + environment_relative_path + "\n")
    return selected_manifest


def fixture_environment(directory: Path, executable: Path) -> dict[str, str]:
    """Allow only tool search and fixture-owned process context into test children."""
    home = directory / "operator-home"
    temporary = directory / "temporary"
    home.mkdir(exist_ok=True)
    temporary.mkdir(exist_ok=True)
    return {
        "PATH": os.environ["PATH"],
        "HOME": str(home),
        "TMPDIR": str(temporary),
        "TMP": str(temporary),
        "TEMP": str(temporary),
        "LANG": "C",
        "CI": "1",
        "GIT_TERMINAL_PROMPT": "0",
        "GIT_CONFIG_NOSYSTEM": "1",
        "MPRLAB_GATEWAY_EXECUTABLE": str(executable),
    }


def controller_plan(executable: Path, directory: Path, manifest: bytes) -> subprocess.CompletedProcess[str]:
    """Validate real source capture and manifest input before the injected Ansible boundary."""
    directory = directory.resolve()
    repository = directory / "app"
    prepare_synthetic_application(repository, manifest)
    origin = directory / "origin.git"
    operator = directory / "operator"
    inventory = operator / "inventory/hosts.yml"
    inventory.parent.mkdir(parents=True)
    inventory.write_text("all:\n  hosts:\n    localhost:\n      ansible_connection: local\n")
    environment = fixture_environment(directory, executable)

    def git(*arguments: str) -> None:
        subprocess.run(
            ["git", "-C", str(repository), "-c", "core.hooksPath=/dev/null", *arguments],
            stdin=subprocess.DEVNULL, env=environment, capture_output=True, text=True, timeout=30, check=True,
        )

    git("init", "--bare", "-b", "master", str(origin))
    git("init", "-b", "master")
    git("add", ".gitignore", ".mprlab/deploy/resources.yml")
    git("-c", "user.name=Policy fixture", "-c", "user.email=fixture@example.invalid",
        "-c", "commit.gpgsign=false", "commit", "-m", "Publication policy fixture")
    git("remote", "add", "origin", str(origin))
    git("push", "-u", "origin", "master")
    git("remote", "set-head", "origin", "master")

    # Echo the child argv instead of executing the Ansible plan or contacting providers.
    echo = shutil.which("echo")
    assert echo is not None, "The deployment test requires the echo executable"
    return subprocess.run(
        [str(executable), "app-plan-release", "--app-root", str(repository),
         "--operator-root", str(operator), "--ansible-inventory", str(inventory), "--ansible-playbook", echo],
        stdin=subprocess.DEVNULL,
        env=environment,
        capture_output=True,
        text=True,
        timeout=30,
        check=False,
    )


@pytest.fixture(scope="module")
def gateway(tmp_path_factory: pytest.TempPathFactory) -> tuple[ModuleType, Path]:
    executable = Path(os.environ["MPRLAB_GATEWAY_EXECUTABLE"]).resolve(strict=True)
    ansible_root = executable.parents[1] / "deploy/ansible"
    validation = controller_plan(executable, tmp_path_factory.mktemp("valid-controller-plan"), MANIFEST_PATH.read_bytes())
    assert validation.returncode == 0, validation.stderr
    assert str(ansible_root / "playbooks/app-lifecycle-plan.yml") in validation.stdout
    spec = importlib.util.spec_from_file_location(
        "npm_visibility_action", ansible_root / "action_plugins/mprlab_network_operation.py"
    )
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module, ansible_root


def test_controller_rejects_unbounded_attempts(gateway: tuple[ModuleType, Path], tmp_path: Path) -> None:
    _, ansible_root = gateway
    executable = ansible_root.parents[1] / "bin/mprlab-gateway"
    document = yaml.safe_load(MANIFEST_PATH.read_text())
    document["mprlab_resources"]["operations"][VERIFY_OPERATION]["polling"]["attempt_limit"] = 0
    validation = controller_plan(executable, tmp_path, yaml.safe_dump(document).encode())
    assert validation.returncode == 2
    assert f"operation {VERIFY_OPERATION}: polling requires positive intervals and a finite attempt limit" in validation.stderr
    assert str(ansible_root / "playbooks/app-lifecycle-plan.yml") not in validation.stdout



@pytest.mark.parametrize("phase", ["release", "publish", "deploy"])
@pytest.mark.parametrize("scenario", ["caller-precedence", "selected-file", "missing-selected-file"])
def test_lifecycle_make_environment(
    gateway: tuple[ModuleType, Path], tmp_path: Path, phase: str, scenario: str,
) -> None:
    """Resolve synthetic Make inputs through the installed SDK manifest boundary."""
    _, ansible_root = gateway
    executable = ansible_root.parents[1] / "bin/mprlab-gateway"
    repository = tmp_path / "application checkout"
    manifest = prepare_synthetic_application(repository, MANIFEST_PATH.read_bytes())
    shutil.copyfile(MANIFEST_PATH.parents[2] / "Makefile", repository / "Makefile")
    environment = fixture_environment(tmp_path, executable)
    environment["MPRLAB_GATEWAY_EXECUTABLE"] = sys.executable
    environment["FIXTURE_GATEWAY"] = str(executable)
    selected_file = repository / manifest["environment_file"]
    expected = {name: "synthetic-policy-input" for name in manifest["inputs"]}
    if scenario != "selected-file":
        expected = {name: "synthetic-caller-input" for name in manifest["inputs"]}
        environment.update(expected)
    if scenario == "missing-selected-file":
        selected_file.unlink()

    other_file = repository / "configs/.env.loopaware"
    other_file.parent.mkdir()
    extra_read = tmp_path / "other-file-was-sourced"
    other_file.write_text("".join(f'{name}="synthetic-other-file"\n' for name in manifest["inputs"])
                          + f"printf unexpected > '{extra_read}'\n")
    receipt = tmp_path / "sdk-capture.json"
    child = ("import json,pathlib,os,sys; "
             "capture=json.loads(os.environ['MPRLAB_LISTENER_EXECUTION_POLICY']); "
             "envelope=json.loads(pathlib.Path(capture['path']).read_text()); "
             f"pathlib.Path({str(receipt)!r}).write_text(json.dumps("
             "{'inputs':envelope['inputs'],'stdin':sys.stdin.read()}))")
    # Replace only the lifecycle dispatch boundary; SDK manifest resolution stays real.
    (repository / f"app-{phase}").write_text(
        "import os,sys\n"
        f"assert sys.argv[1:] == ['--app-root', {str(repository)!r}]\n"
        "gateway=os.environ['FIXTURE_GATEWAY']\n"
        f"os.execve(gateway, [gateway, 'bootstrap-run', '--repo-root', {str(repository)!r}, "
        f"'--operation', 'app-{phase}', '--', sys.executable, '-c', {child!r}], os.environ)\n"
    )
    for arguments in (("init", "--quiet"), ("add", ".gitignore", ".mprlab/deploy/resources.yml", "Makefile")):
        subprocess.run(["git", "-C", str(repository), *arguments],
                       env=environment, stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=30, check=True)
    result = subprocess.run(["make", phase], cwd=repository, env=environment,
                            input="caller input must not reach Gateway\n", capture_output=True, text=True, timeout=30)
    assert not extra_read.exists(), "The lifecycle wrapper sourced an unselected environment file"
    if scenario == "missing-selected-file":
        assert result.returncode != 0
        assert manifest["environment_file"] in result.stderr
        assert not receipt.exists()
    else:
        assert result.returncode == 0, result.stdout + result.stderr
        observed = json.loads(receipt.read_text())
        assert observed["inputs"] == expected
        assert observed["stdin"] == ""


def run_manifest_envelope(repository: Path, directory: Path, ansible_root: Path) -> subprocess.CompletedProcess[str]:
    """Run the installed surface reader and envelope assertion through real Ansible."""
    lifecycle = yaml.safe_load((ansible_root / "playbooks/app-lifecycle-plan.yml").read_text())
    read_surface = next(task for task in lifecycle[0]["tasks"]
                        if task.get("name") == "Read selected application deployment surface")
    validation = yaml.safe_load((ansible_root / "playbooks/tasks/validate-selected-app-manifest.yml").read_text())
    envelope = next(task for task in validation
                    if task.get("name") == "Assert selected application manifest envelope")
    playbook = directory / "envelope.yml"
    playbook.write_text(yaml.safe_dump([{
        "hosts": "localhost", "connection": "local", "gather_facts": False,
        "vars": {
            "mprlab_application_root_raw": str(repository),
            "mprlab_application_manifest_relative_path": ".mprlab/deploy/resources.yml",
            "mprlab_selected_manifest_document": yaml.safe_load(MANIFEST_PATH.read_text()),
        },
        "tasks": [read_surface, envelope],
    }]))
    environment = fixture_environment(directory, ansible_root.parents[1] / "bin/mprlab-gateway")
    environment["ANSIBLE_NOCOLOR"] = "1"
    return subprocess.run(
        ["ansible-playbook", "-i", "localhost,", str(playbook)],
        env=environment, stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=30, check=False,
    )


def test_tracked_application_manifest_envelope(gateway: tuple[ModuleType, Path], tmp_path: Path) -> None:
    _, ansible_root = gateway
    result = run_manifest_envelope(MANIFEST_PATH.parents[2], tmp_path, ansible_root)
    assert result.returncode == 0, result.stdout + result.stderr


def test_manifest_envelope_rejects_extra_tracked_file(gateway: tuple[ModuleType, Path], tmp_path: Path) -> None:
    _, ansible_root = gateway
    repository = tmp_path / "app"
    forbidden = repository / ".mprlab/deploy/tests/extra.txt"
    forbidden.parent.mkdir(parents=True)
    forbidden.write_text("Forbidden deployment surface fixture\n")
    manifest = repository / ".mprlab/deploy/resources.yml"
    manifest.write_bytes(MANIFEST_PATH.read_bytes())
    environment = fixture_environment(tmp_path, ansible_root.parents[1] / "bin/mprlab-gateway")
    for arguments in (("init",), ("add", ".mprlab/deploy")):
        subprocess.run(["git", "-C", str(repository), *arguments], env=environment,
                       stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=30, check=True)
    result = run_manifest_envelope(repository, tmp_path, ansible_root)
    assert result.returncode == 2, result.stdout + result.stderr
    assert "app_lifecycle.invalid_manifest: selected application must track the single manifest" in result.stdout


def synthetic_capture(gateway: tuple[ModuleType, Path], tmp_path: Path) -> dict:
    action_module, ansible_root = gateway
    executable = ansible_root.parents[1] / "bin/mprlab-gateway"
    repository = tmp_path / "application"
    manifest = prepare_synthetic_application(repository, MANIFEST_PATH.read_bytes())
    environment = fixture_environment(tmp_path, executable)
    child = """import json,os,pathlib
assert "synthetic-ambient-context-must-not-propagate" not in os.environ.values(), "ambient context reached fixture child"
capture=json.loads(os.environ['MPRLAB_LISTENER_EXECUTION_POLICY'])
envelope=json.loads(pathlib.Path(capture['path']).read_text())
print(json.dumps({'descriptor':capture,'envelope':envelope}))
"""
    result = subprocess.run(
        [str(executable), "bootstrap-run", "--repo-root", str(repository.resolve()),
         "--operation", "app-publish", "--", sys.executable, "-c", child],
        env=environment, stdin=subprocess.DEVNULL, capture_output=True, text=True, timeout=30, check=False,
    )
    assert result.returncode == 0, result.stderr
    captured = json.loads(result.stdout)
    envelope = captured["envelope"]
    assert envelope["inputs"] == {name: "synthetic-policy-input" for name in manifest["inputs"]}
    assert all(value == "synthetic-policy-input" for inputs in envelope["host_inputs"].values()
               for value in inputs.values())
    capture_path = tmp_path / "mprlab-captured-manifest-test.json"
    capture_path.write_text(json.dumps(captured["envelope"]))
    descriptor = captured["descriptor"]
    descriptor["path"] = str(capture_path)
    return json.loads(action_module._contract.mprlab_resource_capture(
        json.dumps(descriptor), RESOURCE, "app-publish",
    ))


@pytest.fixture
def capture(gateway: tuple[ModuleType, Path], tmp_path: Path) -> dict:
    return synthetic_capture(gateway, tmp_path)


def test_bootstrap_capture_excludes_ambient_context(
    gateway: tuple[ModuleType, Path], tmp_path: Path, monkeypatch: pytest.MonkeyPatch,
) -> None:
    for name in (
        "NPM_API_KEY", "GH_TOKEN", "GITHUB_TOKEN", "NODE_AUTH_TOKEN",
        "MPRLAB_CAPTURED_INPUT_ROOT", "MPRLAB_LISTENER_EXECUTION_POLICY",
        "MPRLAB_TAUTH_MANAGEMENT_URL", "MPRLAB_TAUTH_PROVISIONING_CREDENTIALS",
        "UNDECLARED_APPLICATION_SECRET",
    ):
        monkeypatch.setenv(name, "synthetic-ambient-context-must-not-propagate")
    synthetic_capture(gateway, tmp_path)



@pytest.fixture
def observe(gateway: tuple[ModuleType, Path], capture: dict, monkeypatch: pytest.MonkeyPatch) -> Callable:
    action_module, ansible_root = gateway
    manifest = action_module._contract.mprlab_captured_manifest(json.dumps(capture))["mprlab_resources"]
    retry_policy = yaml.safe_load((ansible_root / "playbooks/vars/app-lifecycle-contract.yml").read_text())["mprlab_network_retry_policy"]
    resource = manifest["resources"][RESOURCE]

    def run(*, ready_after: float = 78, error: str = "E404", operation: str = VERIFY_OPERATION,
            parent_seconds: float | None = None) -> tuple[dict, Clock, list[float]]:
        package = resource["package"] + "@0.0.0-test"
        npm_arguments = (["pack", package, "--ignore-scripts", "--json", "--pack-destination", "/synthetic/archive",
                          "--cache", "/synthetic/archive/cache", "--registry", resource["registry"],
                          "--userconfig", "/dev/stdin"] if operation == ARCHIVE_OPERATION else
                         ["view", package, "--json", "--registry", resource["registry"]])
        command = [os.environ["MPRLAB_GATEWAY_EXECUTABLE"], "owned-command-operation", "--", "npm", *npm_arguments]
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



def test_archive_policy_matches_version_visibility_without_new_deadlines(
    gateway: tuple[ModuleType, Path], capture: dict,
) -> None:
    action_module, _ = gateway
    contract = action_module._contract
    archive = contract.mprlab_execution_policy(json.dumps(capture), ARCHIVE_OPERATION)
    version = contract.mprlab_execution_policy(json.dumps(capture), VERIFY_OPERATION)
    publish = contract.mprlab_execution_policy(json.dumps(capture), "app_publish.npm.publish")
    assert contract.mprlab_execution_polling(archive) == contract.mprlab_execution_polling(version)
    assert contract.mprlab_execution_polling(archive)["attempt_limit"] == 1443
    for timeout in ("startup", "completion", "readiness", "request", "shutdown"):
        assert archive[timeout] == publish[timeout]
    assert archive["completion"] == 4 * 60 * 60 * 1_000_000_000


def test_archive_transport_recovers_after_four_attempts(observe: Callable) -> None:
    result, _, calls = observe(operation=ARCHIVE_OPERATION, error="connection reset")
    assert result["rc"] == 0, result
    assert len(calls) > 4
    assert 78 <= calls[-1] < 88


def test_archive_transport_exhausts_at_existing_publication_deadline(observe: Callable) -> None:
    result, clock, calls = observe(operation=ARCHIVE_OPERATION, error="connection reset", ready_after=float("inf"))
    assert result["failed"]
    assert "deadline_exhausted" in result.get("msg", ""), result
    assert len(calls) <= 1443
    assert 4 * 60 * 60 - 10 <= clock.elapsed < 4 * 60 * 60


def test_archive_transport_respects_shorter_parent_deadline(observe: Callable) -> None:
    result, clock, calls = observe(operation=ARCHIVE_OPERATION, error="connection reset", parent_seconds=20)
    assert result["failed"]
    assert "deadline_exhausted" in result.get("msg", ""), result
    assert clock.elapsed < 20
    assert calls[-1] == clock.elapsed


@pytest.mark.parametrize("error", ["E401", "E403"])
def test_archive_authentication_errors_stop_immediately(observe: Callable, error: str) -> None:
    result, clock, calls = observe(operation=ARCHIVE_OPERATION, error=error)
    assert result["rc"] == 1
    assert error in result["stderr"]
    assert calls == [0]
    assert clock.elapsed == 0


def test_archive_etarget_recovers_after_four_attempts(observe: Callable) -> None:
    result, _, calls = observe(operation=ARCHIVE_OPERATION, error="ETARGET")
    assert result["rc"] == 0, result
    assert len(calls) > 4
    assert 78 <= calls[-1] < 88


def test_archive_etarget_exhausts_at_existing_publication_deadline(observe: Callable) -> None:
    result, clock, calls = observe(operation=ARCHIVE_OPERATION, error="ETARGET", ready_after=float("inf"))
    assert result["failed"]
    assert "deadline_exhausted" in result.get("msg", ""), result
    assert 4 < len(calls) <= 1443
    assert 4 * 60 * 60 - 10 <= clock.elapsed < 4 * 60 * 60
    assert calls[-1] == clock.elapsed


def test_archive_etarget_respects_shorter_parent_deadline(observe: Callable) -> None:
    result, clock, calls = observe(operation=ARCHIVE_OPERATION, error="ETARGET", parent_seconds=20)
    assert result["failed"]
    assert "deadline_exhausted" in result.get("msg", ""), result
    assert 1 < len(calls) <= 4
    assert clock.elapsed < 20
    assert calls[-1] == clock.elapsed
