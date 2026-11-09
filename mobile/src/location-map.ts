import type { LocationPoint } from "./types";
import type { ThemeColors } from "./theme";
import { locationWorldLandPath, locationMapWidth, locationMapHeight, projectLocationCoordinates } from "./location-world";

function escapeXML(value: string): string {
  return value.replace(/[&<>"']/g, character => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&apos;" })[character]!);
}

// Expo Image's Android data URL loader requires base64 bytes.
function base64UTF8(value: string): string {
  const bytes = encodeURIComponent(value).replace(/%([0-9A-F]{2})/g, (_, hex: string) => String.fromCharCode(parseInt(hex, 16)));
  const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";
  let result = "";
  for (let index = 0; index < bytes.length; index += 3) {
    const group = (bytes.charCodeAt(index) << 16) | (bytes.charCodeAt(index + 1) << 8) | bytes.charCodeAt(index + 2);
    result += alphabet[(group >>> 18) & 63] + alphabet[(group >>> 12) & 63];
    result += index + 1 < bytes.length ? alphabet[(group >>> 6) & 63] : "=";
    result += index + 2 < bytes.length ? alphabet[group & 63] : "=";
  }
  return result;
}

export function locationMapImage(locations: LocationPoint[], colors: ThemeColors): string {
  const maximum = Math.max(...locations.map(location => location.unique_visitor_count), 1);
  const labelBoxes: Array<{left: number; right: number; top: number; bottom: number}> = [];
  const grid = [-120, -60, 0, 60, 120].map(longitude => {
    const { x } = projectLocationCoordinates({ latitude: 0, longitude });
    return `<line x1="${x}" y1="0" x2="${x}" y2="${locationMapHeight}"/>`;
  }).concat([-45, 0, 45].map(latitude => {
    const { y } = projectLocationCoordinates({ latitude, longitude: 0 });
    return `<line x1="0" y1="${y}" x2="${locationMapWidth}" y2="${y}"/>`;
  })).join("");
  const markers = locations.map(location => {
    const { x, y } = projectLocationCoordinates(location);
    const radius = 5 + Math.sqrt(location.unique_visitor_count / maximum) * 17;
    const cx = Math.max(radius + 16, Math.min(locationMapWidth - radius - 16, x));
    const cy = Math.max(radius + 16, Math.min(locationMapHeight - radius - 16, y));
    const label = escapeXML(location.label);
    const width = Math.max(58, Math.min(130, location.label.length * 6.5 + 18));
    const left = Math.max(8, Math.min(locationMapWidth - width - 8, cx - width / 2));
    let top = Math.max(10, Math.min(locationMapHeight - 32, cy < locationMapHeight * 0.52 ? cy + radius + 10 : cy - radius - 32));
    for (const box of labelBoxes) {
      if (left < box.right + 10 && left + width > box.left - 10 && top < box.bottom + 6 && top + 22 > box.top - 6) {
        top = Math.min(locationMapHeight - 32, box.bottom + 6);
      }
    }
    labelBoxes.push({left, right: left + width, top, bottom: top + 22});
    const characters = Array.from(location.label);
    const shortLabel = escapeXML(characters.length > 16 ? `${characters.slice(0, 15).join("")}…` : location.label);
    return `<g><title>${label}: ${location.unique_visitor_count} unique visitors, ${location.visit_count} page views</title>
      <line x1="${cx}" y1="${cy}" x2="${left + width / 2}" y2="${top + 11}" stroke="${colors.muted}"/>
      <circle class="traffic-map__bubble" cx="${cx.toFixed(2)}" cy="${cy.toFixed(2)}" r="${radius.toFixed(2)}" fill="${colors.teal}" fill-opacity="0.35" stroke="${colors.tealDark}" stroke-width="1.5"/>
      <text x="${cx.toFixed(2)}" y="${cy.toFixed(2)}" text-anchor="middle" dominant-baseline="central" fill="${colors.ink}" font-size="14" font-family="sans-serif" font-weight="600">${location.unique_visitor_count}</text>
      <rect x="${left}" y="${top}" width="${width}" height="22" rx="11" fill="${colors.surface}" stroke="${colors.border}"/>
      <text class="traffic-map__bubble-label" x="${left + width / 2}" y="${top + 11}" text-anchor="middle" dominant-baseline="central" fill="${colors.ink}" font-size="11" font-family="sans-serif">${shortLabel}</text>
    </g>`;
  }).join("");
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${locationMapWidth}" height="${locationMapHeight}" viewBox="0 0 ${locationMapWidth} ${locationMapHeight}" data-map-source="natural-earth-110m" data-projection="equirectangular">
    <rect width="${locationMapWidth}" height="${locationMapHeight}" fill="${colors.surface}"/>
    <path class="traffic-map__land" d="${locationWorldLandPath}" fill="${colors.surfaceAlt}" stroke="${colors.muted}" stroke-opacity="0.5" stroke-width="0.75"/>
    <g stroke="${colors.border}" stroke-width="1">${grid}</g>${markers}
  </svg>`;
  return `data:image/svg+xml;base64,${base64UTF8(svg)}`;
}
