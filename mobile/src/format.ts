export function formatCount(value: number | undefined): string {
  return new Intl.NumberFormat("en-US").format(value || 0);
}

export function formatPercent(value: number | undefined): string {
  return `${Math.round((value || 0) * 100)}%`;
}

export function formatDateTime(unixSeconds: number | undefined): string {
  if (!unixSeconds) {
    return "Never";
  }
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  }).format(new Date(unixSeconds * 1000));
}

export function formatDate(unixSeconds: number | undefined): string {
  if (!unixSeconds) {
    return "Not set";
  }
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
  }).format(new Date(unixSeconds * 1000));
}

export function compactText(value: string | undefined, fallback = "Unknown"): string {
  const normalized = String(value || "").trim();
  return normalized || fallback;
}

export function sentenceCase(value: string | undefined): string {
  const normalized = compactText(value, "").replace(/_/g, " ");
  if (!normalized) {
    return "Unknown";
  }
  return normalized.charAt(0).toUpperCase() + normalized.slice(1);
}
