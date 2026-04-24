"""Protected LoopAware LA Sentry ingest client for Python services."""

from __future__ import annotations

from dataclasses import dataclass, field
from datetime import UTC, datetime
from http import HTTPStatus
import json
import traceback
from types import TracebackType
from typing import Any, Callable, Mapping, MutableMapping, Protocol, Sequence
from urllib.error import HTTPError, URLError
from urllib.parse import quote, urlparse
from urllib.request import Request, urlopen
from uuid import uuid4

DEFAULT_LEVEL = "error"
DEFAULT_TIMEOUT_SECONDS = 5.0
MAX_EVENT_ID_LENGTH = 100
MAX_FRAME_STRING_LENGTH = 500
MAX_LEVEL_LENGTH = 20
MAX_MESSAGE_LENGTH = 4000
MAX_RELEASE_LENGTH = 200
MAX_SITE_ID_LENGTH = 36
MAX_TAG_LENGTH = 200
MAX_USER_HASH_LENGTH = 200
PLATFORM = "python"
LA_SENTRY_ENDPOINT_PATH = "/sentry/errors"


class InvalidLASentryConfig(ValueError):
    """Raised when client configuration cannot produce valid LA Sentry events."""


class InvalidCaptureAttributes(ValueError):
    """Raised when capture attributes cannot produce valid LA Sentry events."""


class CaptureFailed(RuntimeError):
    """Raised when LoopAware rejects or cannot receive an error event."""


class HTTPResponse(Protocol):
    """Small response protocol used by the injectable transport."""

    status: int

    def read(self) -> bytes:
        """Read the response body."""

    def __enter__(self) -> "HTTPResponse":
        """Enter the response context."""

    def __exit__(
        self,
        exception_type: type[BaseException] | None,
        exception_value: BaseException | None,
        traceback_value: TracebackType | None,
    ) -> bool | None:
        """Exit the response context."""


Transport = Callable[[Request, float], HTTPResponse]


@dataclass(frozen=True)
class LASentryConfig:
    """Validated configuration for token-protected LoopAware LA Sentry ingest."""

    endpoint: str
    site_id: str
    ingest_token: str
    environment: str
    release: str = ""
    default_tags: Mapping[str, str] = field(default_factory=dict)
    timeout_seconds: float = DEFAULT_TIMEOUT_SECONDS

    def __post_init__(self) -> None:
        endpoint = normalize_endpoint(self.endpoint)
        site_id = normalize_required_string(self.site_id, "site_id", MAX_SITE_ID_LENGTH)
        ingest_token = normalize_required_string(self.ingest_token, "ingest_token", MAX_MESSAGE_LENGTH)
        environment = normalize_required_string(self.environment, "environment", MAX_TAG_LENGTH)
        release = normalize_optional_string(self.release, MAX_RELEASE_LENGTH)
        timeout_seconds = float(self.timeout_seconds)
        if timeout_seconds <= 0:
            raise InvalidLASentryConfig("invalid_timeout_seconds")
        object.__setattr__(self, "endpoint", endpoint)
        object.__setattr__(self, "site_id", site_id)
        object.__setattr__(self, "ingest_token", ingest_token)
        object.__setattr__(self, "environment", environment)
        object.__setattr__(self, "release", release)
        object.__setattr__(self, "default_tags", normalize_tags(self.default_tags))
        object.__setattr__(self, "timeout_seconds", timeout_seconds)


@dataclass(frozen=True)
class CaptureAttributes:
    """Validated optional metadata for a captured Python exception."""

    event_id: str = ""
    level: str = DEFAULT_LEVEL
    user_hash: str = ""
    tags: Mapping[str, str] = field(default_factory=dict)
    extra: Mapping[str, Any] = field(default_factory=dict)
    request: Mapping[str, Any] = field(default_factory=dict)

    def __post_init__(self) -> None:
        level = normalize_required_capture_string(self.level, "level", MAX_LEVEL_LENGTH).lower()
        event_id = normalize_optional_string(self.event_id, MAX_EVENT_ID_LENGTH)
        user_hash = normalize_optional_string(self.user_hash, MAX_USER_HASH_LENGTH)
        object.__setattr__(self, "event_id", event_id)
        object.__setattr__(self, "level", level)
        object.__setattr__(self, "user_hash", user_hash)
        object.__setattr__(self, "tags", normalize_tags(self.tags))
        object.__setattr__(self, "extra", normalize_json_mapping(self.extra, "extra"))
        object.__setattr__(self, "request", normalize_json_mapping(self.request, "request"))


class Client:
    """Client for submitting Python exceptions to LoopAware LA Sentry ingest."""

    def __init__(self, config: LASentryConfig, transport: Transport | None = None) -> None:
        self._config = config
        self._transport = transport or default_transport

    def capture_error(self, error: BaseException, attributes: CaptureAttributes | Mapping[str, Any] | None = None) -> Mapping[str, Any]:
        """Submit an explicit Python exception event."""

        capture_attributes = coerce_capture_attributes(attributes)
        payload = self._build_payload(error, capture_attributes)
        return self._submit(payload)

    def _build_payload(self, error: BaseException, attributes: CaptureAttributes) -> Mapping[str, Any]:
        event_id = attributes.event_id or str(uuid4())
        tags: dict[str, str] = dict(self._config.default_tags)
        tags.update(attributes.tags)
        return {
            "site_id": self._config.site_id,
            "event_id": event_id,
            "timestamp": datetime.now(UTC).isoformat().replace("+00:00", "Z"),
            "platform": PLATFORM,
            "environment": self._config.environment,
            "release": self._config.release,
            "level": attributes.level,
            "message": normalize_optional_string(str(error), MAX_MESSAGE_LENGTH) or error.__class__.__name__,
            "exception_type": normalize_optional_string(error.__class__.__name__, MAX_FRAME_STRING_LENGTH),
            "stacktrace": stacktrace_from_exception(error),
            "request": attributes.request,
            "user_hash": attributes.user_hash,
            "tags": tags,
            "extra": attributes.extra,
        }

    def _submit(self, payload: Mapping[str, Any]) -> Mapping[str, Any]:
        serialized_payload = json.dumps(payload, separators=(",", ":"), sort_keys=True).encode("utf-8")
        request = Request(
            self._config.endpoint,
            data=serialized_payload,
            method="POST",
            headers={
                "Accept": "application/json",
                "Authorization": "Bearer " + self._config.ingest_token,
                "Content-Type": "application/json",
            },
        )
        try:
            with self._transport(request, self._config.timeout_seconds) as response:
                response_body = response.read()
                if response.status != HTTPStatus.OK:
                    raise CaptureFailed("capture_failed:" + str(response.status) + ":" + response_body.decode("utf-8", errors="replace"))
        except HTTPError as http_error:
            error_body = http_error.read().decode("utf-8", errors="replace")
            raise CaptureFailed("capture_failed:" + str(http_error.code) + ":" + error_body) from http_error
        except URLError as url_error:
            raise CaptureFailed("capture_failed:network:" + str(url_error.reason)) from url_error
        decoded_payload = json.loads(response_body.decode("utf-8"))
        if not isinstance(decoded_payload, dict):
            raise CaptureFailed("capture_failed:invalid_response")
        return decoded_payload


class WSGILASentryMiddleware:
    """WSGI middleware that captures uncaught application exceptions."""

    def __init__(self, app: Callable[..., Any], client: Client) -> None:
        self._app = app
        self._client = client

    def __call__(self, environ: MutableMapping[str, Any], start_response: Callable[..., Any]) -> Any:
        try:
            result = self._app(environ, start_response)
        except Exception as error:
            self._client.capture_error(error, CaptureAttributes(request=request_from_wsgi_environ(environ)))
            raise
        return self._capture_iterable(result, environ)

    def _capture_iterable(self, result: Any, environ: MutableMapping[str, Any]) -> Any:
        try:
            for chunk in result:
                yield chunk
        except Exception as error:
            self._client.capture_error(error, CaptureAttributes(request=request_from_wsgi_environ(environ)))
            raise
        finally:
            close = getattr(result, "close", None)
            if callable(close):
                close()


class ASGILASentryMiddleware:
    """ASGI middleware that captures uncaught application exceptions."""

    def __init__(self, app: Callable[..., Any], client: Client) -> None:
        self._app = app
        self._client = client

    async def __call__(self, scope: Mapping[str, Any], receive: Callable[..., Any], send: Callable[..., Any]) -> Any:
        try:
            return await self._app(scope, receive, send)
        except Exception as error:
            self._client.capture_error(error, CaptureAttributes(request=request_from_asgi_scope(scope)))
            raise


def default_transport(request: Request, timeout_seconds: float) -> HTTPResponse:
    """Submit an HTTP request with urllib."""

    return urlopen(request, timeout=timeout_seconds)


def coerce_capture_attributes(attributes: CaptureAttributes | Mapping[str, Any] | None) -> CaptureAttributes:
    """Convert user metadata into validated capture attributes."""

    if attributes is None:
        return CaptureAttributes()
    if isinstance(attributes, CaptureAttributes):
        return attributes
    if not isinstance(attributes, Mapping):
        raise InvalidCaptureAttributes("invalid_attributes")
    return CaptureAttributes(
        event_id=str(attributes.get("event_id", "")),
        level=str(attributes.get("level", DEFAULT_LEVEL)),
        user_hash=str(attributes.get("user_hash", "")),
        tags=coerce_mapping(attributes.get("tags", {}), "tags"),
        extra=coerce_mapping(attributes.get("extra", {}), "extra"),
        request=coerce_mapping(attributes.get("request", {}), "request"),
    )


def stacktrace_from_exception(error: BaseException) -> Sequence[Mapping[str, Any]]:
    """Build LoopAware stack frames from a Python exception traceback."""

    extracted_frames = traceback.extract_tb(error.__traceback__)
    frames: list[Mapping[str, Any]] = []
    for frame in extracted_frames:
        frames.append(
            {
                "filename": normalize_optional_string(frame.filename, MAX_FRAME_STRING_LENGTH),
                "function": normalize_optional_string(frame.name, MAX_FRAME_STRING_LENGTH),
                "module": "",
                "line": int(frame.lineno or 0),
                "column": int(getattr(frame, "colno", 0) or 0),
                "in_app": is_in_app_frame(frame.filename),
            }
        )
    return frames


def is_in_app_frame(filename: str) -> bool:
    """Return whether a Python traceback frame belongs to application code."""

    normalized_filename = filename.replace("\\", "/").lower()
    if "/site-packages/" in normalized_filename or "/dist-packages/" in normalized_filename:
        return False
    return not normalized_filename.endswith("/la_sentry/client.py")


def request_from_wsgi_environ(environ: Mapping[str, Any]) -> Mapping[str, Any]:
    """Build request metadata from WSGI environ."""

    scheme = str(environ.get("wsgi.url_scheme", "http"))
    host = str(environ.get("HTTP_HOST") or environ.get("SERVER_NAME") or "")
    path = quote(str(environ.get("PATH_INFO", "")))
    url = scheme + "://" + host + path if host else path
    return {
        "method": str(environ.get("REQUEST_METHOD", "")),
        "url": url,
        "user_agent": str(environ.get("HTTP_USER_AGENT", "")),
    }


def request_from_asgi_scope(scope: Mapping[str, Any]) -> Mapping[str, Any]:
    """Build request metadata from ASGI scope."""

    scheme = str(scope.get("scheme", "http"))
    server = scope.get("server") or ("", "")
    host = ""
    if isinstance(server, tuple) and server:
        host = str(server[0])
        if len(server) > 1 and server[1]:
            host += ":" + str(server[1])
    path = quote(str(scope.get("path", "")))
    url = scheme + "://" + host + path if host else path
    return {
        "method": str(scope.get("method", "")),
        "url": url,
        "user_agent": header_value(scope.get("headers", []), b"user-agent"),
    }


def header_value(headers: Any, name: bytes) -> str:
    """Read a decoded ASGI header value."""

    if not isinstance(headers, list):
        return ""
    for header_name, header_body in headers:
        if header_name == name:
            return bytes(header_body).decode("utf-8", errors="replace")
    return ""


def normalize_endpoint(endpoint: str) -> str:
    """Validate and normalize the LoopAware LA Sentry endpoint URL."""

    trimmed_endpoint = str(endpoint or "").strip()
    parsed_endpoint = urlparse(trimmed_endpoint)
    if parsed_endpoint.scheme not in {"http", "https"} or not parsed_endpoint.netloc:
        raise InvalidLASentryConfig("invalid_endpoint")
    if not parsed_endpoint.path.endswith(LA_SENTRY_ENDPOINT_PATH):
        raise InvalidLASentryConfig("invalid_endpoint_path")
    return trimmed_endpoint


def normalize_required_string(value: str, field_name: str, max_length: int) -> str:
    """Normalize a required bounded string."""

    normalized = normalize_optional_string(value, max_length)
    if not normalized:
        raise InvalidLASentryConfig("missing_" + field_name)
    return normalized


def normalize_required_capture_string(value: str, field_name: str, max_length: int) -> str:
    """Normalize a required bounded capture string."""

    normalized = normalize_optional_string(value, max_length)
    if not normalized:
        raise InvalidCaptureAttributes("missing_" + field_name)
    return normalized


def normalize_optional_string(value: Any, max_length: int) -> str:
    """Normalize an optional bounded string."""

    normalized = str(value or "").strip()
    if len(normalized) > max_length:
        return normalized[:max_length]
    return normalized


def normalize_tags(tags: Mapping[str, str]) -> Mapping[str, str]:
    """Normalize string tags."""

    normalized_tags: dict[str, str] = {}
    for key, value in tags.items():
        normalized_key = normalize_optional_string(key, MAX_TAG_LENGTH)
        if normalized_key:
            normalized_tags[normalized_key] = normalize_optional_string(value, MAX_TAG_LENGTH)
    return normalized_tags


def normalize_json_mapping(value: Mapping[str, Any], field_name: str) -> Mapping[str, Any]:
    """Validate that a mapping can be serialized as JSON."""

    try:
        json.dumps(value)
    except TypeError as error:
        raise InvalidCaptureAttributes("invalid_" + field_name) from error
    return dict(value)


def coerce_mapping(value: Any, field_name: str) -> Mapping[str, Any]:
    """Require mapping metadata at the client edge."""

    if not isinstance(value, Mapping):
        raise InvalidCaptureAttributes("invalid_" + field_name)
    return value
