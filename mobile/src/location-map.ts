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

export function locationMapPosition(location: LocationPoint, maximum: number): { x: number; y: number; radius: number } {
  const { x, y } = projectLocationCoordinates(location);
  const radius = 5 + Math.sqrt(location.unique_visitor_count / maximum) * 17;
  return {
    x: Math.max(radius + 16, Math.min(locationMapWidth - radius - 16, x)),
    y: Math.max(radius + 16, Math.min(locationMapHeight - radius - 16, y)),
    radius,
  };
}

export function locationMapImage(locations: LocationPoint[], colors: ThemeColors, labelWidth: number, selected?: LocationPoint): string {
  const maximum = Math.max(...locations.map(location => location.unique_visitor_count), 1);
  const grid = [-120, -60, 0, 60, 120].map(longitude => {
    const { x } = projectLocationCoordinates({ latitude: 0, longitude });
    return `<line x1="${x}" y1="0" x2="${x}" y2="${locationMapHeight}"/>`;
  }).concat([-45, 0, 45].map(latitude => {
    const { y } = projectLocationCoordinates({ latitude, longitude: 0 });
    return `<line x1="0" y1="${y}" x2="${locationMapWidth}" y2="${y}"/>`;
  })).join("");
  const bubble = (location: LocationPoint) => {
    const { x, y, radius } = locationMapPosition(location, maximum);
    const active = location === selected;
    return `<g><title>${escapeXML(location.label)}: ${location.unique_visitor_count} unique visitors, ${location.visit_count} page views</title>
      <circle class="traffic-map__bubble" cx="${x.toFixed(2)}" cy="${y.toFixed(2)}" r="${radius.toFixed(2)}" fill="${colors.teal}" fill-opacity="${active ? 0.65 : 0.35}" stroke="${active ? colors.ink : colors.tealDark}" stroke-width="${active ? 2.5 : 1.5}"/>
    </g>`;
  };
  const markers = locations.filter(location => location !== selected).map(bubble).join("");
  const activeMarker = selected ? bubble(selected) + locationMapLabel(selected, maximum, colors, labelWidth) : "";
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="${locationMapWidth}" height="${locationMapHeight}" viewBox="0 0 ${locationMapWidth} ${locationMapHeight}" data-map-source="natural-earth-110m" data-projection="equirectangular">
    <rect width="${locationMapWidth}" height="${locationMapHeight}" fill="${colors.surface}"/>
    <path class="traffic-map__land" d="${locationWorldLandPath}" fill="${colors.surfaceAlt}" stroke="${colors.muted}" stroke-opacity="0.5" stroke-width="0.75"/>
    <g stroke="${colors.border}" stroke-width="1">${grid}</g>${markers}${activeMarker}
  </svg>`;
  return `data:image/svg+xml;base64,${base64UTF8(svg)}`;
}

function locationMapLabel(location: LocationPoint, maximum: number, colors: ThemeColors, canvasWidth: number): string {
  const { x, y, radius } = locationMapPosition(location, maximum);
  const characters = Array.from(location.label);
  const label = characters.length > 16 ? `${characters.slice(0, 15).join("")}…` : location.label;
  const scale = locationMapWidth / canvasWidth;
  const width = Math.min(locationMapWidth - 16, Math.max(58, Math.min(130, characters.length * 6.5 + 18)) * scale);
  const height = 20 * scale;
  const left = Math.max(8, Math.min(locationMapWidth - width - 8, x - width / 2));
  const below = y <= locationMapHeight / 2;
  const top = below ? y + radius + 6 : y - radius - 6 - height;
  return `<line x1="${x}" y1="${below ? y + radius : y - radius}" x2="${left + width / 2}" y2="${below ? top : top + height}" stroke="${colors.muted}"/>
    <rect x="${left}" y="${top}" width="${width}" height="${height}" rx="${4 * scale}" fill="${colors.surface}" stroke="${colors.border}"/>
    <text class="traffic-map__bubble-label" x="${left + width / 2}" y="${top + height / 2}" text-anchor="middle" dominant-baseline="central" fill="${colors.ink}" font-size="${11 * scale}" font-family="sans-serif">${escapeXML(label)}</text>`;
}
