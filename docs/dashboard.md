# Web and mobile dashboards

[Project overview](../README.md)

## Dashboard (`/app`)

The Bootstrap frontend consumes the [HTTP API](api.md). Features include:

- Account Settings modal with avatar, email, role badge, reports, and inactivity controls
- Site creation available to every authenticated user, with an owner email supplied at creation.
- Site metadata and ownership changes available to site managers. Global administrators also see all sites
- Owner/creator/admin editor for site metadata, with per-site team member emails managed from the Admin dashboard section
- Selected-site traffic report scheduling with recipient selection for only the manager, the whole site team, or checked team members
- Selected-site health monitoring with enablement, target URL, interval, timeout, failure threshold, manual check, and alert-recipient controls
- Widget appearance controls that persist the bubble’s accent color, side (left/right), and bottom offset without code changes
- Feedback table with human-readable timestamps
- Subscribers panel with per-site subscriber counts, table, spreadsheet-formula-safe CSV export, and a copyable `subscribe.js` snippet
- Section selector tabs to switch between Feedback, Subscriptions, Traffic, Health, LA Sentry, and manager-only Admin tools
- Subscriber deletion via a confirmation modal
- Traffic card with visit and unique visitor counts, recent visits, and a copyable `pixel.js` snippet
- Real-time favicon refresh notifications delivered through the SSE stream
- Sign-out button provided by the shared `mpr-ui`/TAuth shell
- Inactivity prompt appears after the configured delay (defaults to 60 seconds) and logs out automatically after the configured timeout (defaults to 120 seconds) if unanswered

Site traffic emails support daily, weekly, and monthly schedules.
A background scheduler refreshes site favicons and sends live dashboard notifications through the favicon SSE stream.

The dashboard automatically redirects unauthenticated visitors to `/login`.

## Mobile site cards

Each site card shows its collected favicon to the left of the name.
The icon is 16 pixels wide and stays inside the current card dimensions.
The role stays on its current line.
The mobile client loads the stored image through the authenticated API.

It loads up to four site images at a time and cancels obsolete requests.
Images load again when the app returns to the foreground.
When an image is absent or cannot be shown, the card shows the site name.

## Mobile site filter

Select the search icon on the right of `SITES` to open the site filter.
Enter part of a site name or configured origin between `SITES` and the search icon.
The field uses the same row. The site cards keep their vertical position when the field opens.

Matches update automatically without regard to letter case, including secondary domains.

An empty field shows all authorized site cards. A query without matches shows an explicit empty state.
The filter preserves the selected site and its dashboard when its card is hidden.

Use the clear control to restore all cards and keep the field active.
Select a matching site card to open its dashboard and dismiss the keyboard.
Select the search icon again to close the filter and clear the query.
The query clears when the dashboard view ends, including settings navigation and sign-out.

## Web reports

Select the statistics icon immediately left of the account avatar to open the reports view.
In Reports, this control shows a dashboard icon with the label `Dashboard`.
Select the dashboard icon to open the selected site and tab.
The automatic schedule write completes before the dashboard opens.

The report library keeps custom reports and email schedules.
The header icon provides navigation between the dashboard and reports.

## Mobile statistics

Select the statistics icon immediately left of the account avatar to open `Statistics`.
The report shows traffic for all sites that the account can view, including team sites.
Global administrators see all sites.
Statistics remains available when the selected site has no dashboard data.

Select a site row to open that site's overview. The traffic interval remains selected.
Select the dashboard icon left of the avatar to open the previous site and tab.
Statistics loads current data on entry and when the app returns from the background.

While open, it updates every 30 seconds and recovers automatically after request failures.
The per-site `Reports` tab contains only the selected-site email schedule.

The mobile report requests `GET /api/reports/traffic/portfolio?days=30&scope=visible`.
The `visible` scope uses the same access rules as `GET /api/sites`.
The default `owned` scope, custom reports, and email schedules keep their owner and creator rules.
The API rejects unknown scopes and the combination of `scope=visible` with a custom report ID.

## Traffic intervals

Web and mobile offer all traffic, 30 days, seven days, and one day.
Select `All`, `30d`, `7d`, or `1d` on the mobile Traffic tab.
Select `All`, `30 days`, `7 days`, or `1 day` on web.
The web interval also applies to traffic CSV exports.

The selected interval applies to all traffic panels and remains selected after a site change.
`7d` includes the current UTC day and the preceding six UTC days.
Aggregate sites show daily request counts. Their unique visitor, engagement, device, and location metrics remain unavailable.

## Location maps

Web and mobile use the same Natural Earth world geography and equirectangular projection.
The map shows location bubbles and labels. Mobile also shows location counts as native text.
Mobile bubble size shows unique visitors. Web bubble size shows page views.

The selected traffic interval applies to the map and location counts.

On mobile, select a bubble or location control to show its source and confidence.
The map fits the available width up to 720 points. Location selection remains after rotation.
Unknown locations appear below the map.

Light and dark themes apply automatically. Aggregate sites keep location metrics unavailable.
The map uses local assets and does not request external map tiles.

## Mobile traffic metrics

The overview traffic card and detailed Traffic panels use unique visitors as their primary count.
Top pages, sources, and devices use that count for values, bar lengths, and rank.
Locations use that count for bubble size and rank.

The Trend card follows Locations and shows a chronological line graph for all returned dates.
Its X axis shows `Date (UTC)`. Its Y axis shows `Unique visitors`.
Aggregate sites show the same graph under Daily counts, with `Requests` on the Y axis.

The graph includes zero counts and single-day data. Accessibility services can read the exact daily counts.

Page views appear as explicitly labeled supporting information and include repeat views.
`Returning` measures visitors with more than one page view in the selected interval.
`Pages/visitor` remains the average page count per tracked visitor.
The mobile client requests `sort=unique_visitors` for detailed breakdowns.

Web and email reports keep their current page-view ranking.

## Location map signals

Browser language regions and country-only signals appear as counts below the map.
These signals do not produce map points.
Unknown and local network signals also appear below the map.

Timezone points are estimates. Edge coordinates require a confidence value of at least 88.
Country-only edge signals have confidence values from 80 through 84.
Each row keeps its unique visitor and page view counts.

## Mobile account settings

Open the avatar menu. Select `Settings` to see the account email, account role, and site access roles.
The avatar uses the account image, or an email initial when no image is present.
Select the close icon to open the selected site, tab, and traffic interval.

The `Sign out` action is on the settings screen.

Choose `Light`, `Dark`, or `System` under `Appearance`.
Changes apply immediately and save automatically on this app installation.
`System` follows device appearance changes, including after the app returns from the background.
The preference remains after sign-out and is restored when the app opens again.

Storage errors remain visible. Select a theme again to retry a failed write.

The account photo loads through the authenticated mobile client.
The email initial remains visible while the photo loads or cannot be shown.
Account changes cancel the previous photo request.
The photo loads again when the app returns to the foreground.

## Mobile overview navigation

Select an overview card to open its detail tab:

- `Feedback` opens `Feedback`.
- `Subscribers` opens `Subscribers`.
- `Unique visitors` for detailed sites and `Requests` for aggregate sites open `Traffic`.
- `Health` opens `Health`.
- `Sentry` opens `LA Sentry`.

The complete card is a button, including when its count is zero.
The selected site and traffic interval remain selected.
The destination tab becomes visible. Accessibility focus moves to its content heading.
