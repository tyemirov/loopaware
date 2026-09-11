// @ts-check

const publicThemeStorageKey = "loopaware_public_theme";
const landingThemeStorageKey = "loopaware_landing_theme";
const lightTheme = "light";
const darkTheme = "dark";

/**
 * @param {string | null} rawTheme
 * @returns {"light" | "dark"}
 */
function normalizeTheme(rawTheme) {
  return rawTheme === lightTheme ? lightTheme : darkTheme;
}

/**
 * @param {"light" | "dark"} theme
 * @returns {void}
 */
function applyPublicTheme(theme) {
  const rootElement = document.body;
  const documentRoot = document.documentElement;

  rootElement.setAttribute("data-bs-theme", theme);
  rootElement.setAttribute("data-mpr-theme", theme);
  documentRoot.setAttribute("data-bs-theme", theme);
  documentRoot.setAttribute("data-mpr-theme", theme);
  rootElement.classList.toggle("bg-body", true);
  rootElement.classList.toggle("text-body", true);
}

/**
 * @param {"light" | "dark"} theme
 * @returns {void}
 */
function updateFooterThemeConfig(theme) {
  const footerElement = document.querySelector("mpr-footer");
  if (!footerElement) {
    throw new Error("resource_theme_missing_footer");
  }

  const rawThemeConfig = footerElement.getAttribute("theme-config");
  const themeConfig = rawThemeConfig ? JSON.parse(rawThemeConfig) : {};
  themeConfig.attribute = themeConfig.attribute || "data-bs-theme";
  themeConfig.initialMode = theme;
  footerElement.setAttribute("theme-config", JSON.stringify(themeConfig));
}

/**
 * @returns {string | null}
 */
function loadPublicTheme() {
  const storedTheme = localStorage.getItem(publicThemeStorageKey);
  if (storedTheme !== null) {
    return storedTheme;
  }

  const landingStoredTheme = localStorage.getItem(landingThemeStorageKey);
  if (landingStoredTheme === lightTheme || landingStoredTheme === darkTheme) {
    localStorage.setItem(publicThemeStorageKey, landingStoredTheme);
    return landingStoredTheme;
  }

  return null;
}

/**
 * @param {"light" | "dark"} theme
 * @returns {void}
 */
function persistPublicTheme(theme) {
  localStorage.setItem(publicThemeStorageKey, theme);
  localStorage.setItem(landingThemeStorageKey, theme);
}

const selectedTheme = normalizeTheme(loadPublicTheme());
applyPublicTheme(selectedTheme);
updateFooterThemeConfig(selectedTheme);
persistPublicTheme(selectedTheme);

document.querySelector("mpr-footer")?.addEventListener("mpr-footer:theme-change", (event) => {
  const customEvent = /** @type {CustomEvent<{ theme?: string }>} */ (event);
  const nextTheme = normalizeTheme(customEvent.detail?.theme || null);
  applyPublicTheme(nextTheme);
  persistPublicTheme(nextTheme);
});
