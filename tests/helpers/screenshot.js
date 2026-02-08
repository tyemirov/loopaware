// @ts-check
import * as fs from 'node:fs';
import * as path from 'node:path';
import { PNG } from 'pngjs';
import { resolveRepositoryRoot } from './config.js';

function sanitizeSegment(value) {
  if (!value) {
    return 'unnamed';
  }
  return String(value)
    .split('')
    .map((char) => {
      if (/[a-zA-Z0-9_-]/.test(char)) {
        return char;
      }
      return '_';
    })
    .join('');
}

export function ensureScreenshotDirectory(testInfo) {
  const root = resolveRepositoryRoot();
  const dateSegment = new Date().toISOString().slice(0, 10);
  const titleSegments = Array.isArray(testInfo?.titlePath) ? testInfo.titlePath : [testInfo?.title || 'test'];
  const nameSegment = sanitizeSegment(titleSegments.join('_'));
  const directory = path.join(root, 'tests', 'artifacts', dateSegment, nameSegment);
  fs.mkdirSync(directory, { recursive: true });
  return directory;
}

export function saveScreenshot(directory, name, buffer) {
  const filename = `${sanitizeSegment(name)}.png`;
  const filePath = path.join(directory, filename);
  fs.writeFileSync(filePath, buffer);
  return filePath;
}

export function decodePNG(buffer) {
  return PNG.sync.read(buffer);
}

export function computeRegionLuminanceVariance(png, bounds, viewport) {
  const rectangle = convertBoundsToRectangle(bounds, png, viewport);
  const pixelCount = rectangle.width * rectangle.height;
  if (pixelCount <= 0) {
    return 0;
  }
  let luminanceSum = 0;
  let luminanceSquaredSum = 0;
  for (let y = rectangle.y; y < rectangle.y + rectangle.height; y += 1) {
    for (let x = rectangle.x; x < rectangle.x + rectangle.width; x += 1) {
      const index = (png.width * y + x) * 4;
      const red = png.data[index];
      const green = png.data[index + 1];
      const blue = png.data[index + 2];
      const luminance = computeLuminance(red, green, blue);
      luminanceSum += luminance;
      luminanceSquaredSum += luminance * luminance;
    }
  }
  const mean = luminanceSum / pixelCount;
  const variance = luminanceSquaredSum / pixelCount - mean * mean;
  return variance < 0 ? 0 : variance;
}

export function computeColorPresenceRatio(png, bounds, viewport, target, tolerance) {
  const rectangle = convertBoundsToRectangle(bounds, png, viewport);
  const pixelCount = rectangle.width * rectangle.height;
  if (pixelCount <= 0) {
    return 0;
  }
  let matching = 0;
  for (let y = rectangle.y; y < rectangle.y + rectangle.height; y += 1) {
    for (let x = rectangle.x; x < rectangle.x + rectangle.width; x += 1) {
      const index = (png.width * y + x) * 4;
      const red = png.data[index];
      const green = png.data[index + 1];
      const blue = png.data[index + 2];
      if (Math.abs(red - target.red) <= tolerance && Math.abs(green - target.green) <= tolerance && Math.abs(blue - target.blue) <= tolerance) {
        matching += 1;
      }
    }
  }
  return matching / pixelCount;
}

function computeLuminance(red, green, blue) {
  const normalizedRed = red / 255;
  const normalizedGreen = green / 255;
  const normalizedBlue = blue / 255;
  return 0.2126 * normalizedRed + 0.7152 * normalizedGreen + 0.0722 * normalizedBlue;
}

function convertBoundsToRectangle(bounds, png, viewport) {
  const scaleX = png.width / viewport.width;
  const scaleY = png.height / viewport.height;
  let minX = Math.floor(bounds.x * scaleX);
  let minY = Math.floor(bounds.y * scaleY);
  let maxX = Math.ceil((bounds.x + bounds.width) * scaleX);
  let maxY = Math.ceil((bounds.y + bounds.height) * scaleY);
  minX = clamp(minX, 0, png.width - 1);
  minY = clamp(minY, 0, png.height - 1);
  maxX = clamp(maxX, minX + 1, png.width);
  maxY = clamp(maxY, minY + 1, png.height);
  return { x: minX, y: minY, width: maxX - minX, height: maxY - minY };
}

function clamp(value, min, max) {
  if (value < min) {
    return min;
  }
  if (value > max) {
    return max;
  }
  return value;
}
