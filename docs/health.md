# Health endpoint

Unauthenticated `GET /healthz` checks local database access with a one-second
deadline. Success returns `200` and `{"status":"ok"}`. A failure returns
`503` and `{"status":"unavailable"}`. Both responses use
`Cache-Control: no-store`.

The probe does not send notifications, record visits, or change customer data.
Successful probes do not produce request events. Failed probes produce an
error event and retain the request event.

Docker probes keep failure output. They use a one-second startup interval,
a 30-second steady interval, and a 30-second startup period.
The website publishes `web/healthz`. Local web servers add the no-store header.
GitHub Pages uses its production cache policy for the static health resource.
The operator approved this exception on 2026-09-04. API and local health
responses still require `Cache-Control: no-store`.
A cached Pages response proves artifact availability, not current API readiness.
