import React, { createContext, useCallback, useContext, useEffect, useRef, useState, useSyncExternalStore } from "react";
import * as SecureStore from "expo-secure-store";
import { Appearance, AppState } from "react-native";

export type ThemePreference = "light" | "dark" | "system";
type ThemeAppearance = "light" | "dark";

export interface ThemeColors {
  background: string;
  ink: string;
  muted: string;
  border: string;
  surface: string;
  surfaceAlt: string;
  teal: string;
  tealDark: string;
  amber: string;
  red: string;
  green: string;
  accentSurface: string;
  roleSurface: string;
  errorSurface: string;
  errorBorder: string;
  onAccent: string;
}

export const palettes: Record<ThemeAppearance, ThemeColors> = {
  light: {
    background: "#f5f7f6", ink: "#17211f", muted: "#65726e", border: "#d8e0dd",
    surface: "#fff", surfaceAlt: "#eef3f1", teal: "#157f78", tealDark: "#0e5e59",
    amber: "#b7791f", red: "#b42318", green: "#287d3c", accentSurface: "#e6f4f1",
    roleSurface: "#d9f0ec", errorSurface: "#fff2f1", errorBorder: "#f2b8b5", onAccent: "#fff",
  },
  dark: {
    background: "#101816", ink: "#eef5f2", muted: "#a7b7b0", border: "#34453f",
    surface: "#192620", surfaceAlt: "#24362e", teal: "#62d4c6", tealDark: "#8de5da",
    amber: "#eabd6c", red: "#ffb4ab", green: "#8dd9a1", accentSurface: "#173c35",
    roleSurface: "#214b42", errorSurface: "#401f20", errorBorder: "#8d4645", onAccent: "#092f2b",
  },
};

const storageKey = "loopaware.mobile.theme";

interface ThemeState {
  preference: ThemePreference | null;
  loading: boolean;
  saving: boolean;
  error: string | null;
}

interface ThemeValue extends ThemeState {
  appearance: ThemeAppearance;
  colors: ThemeColors;
  select: (preference: ThemePreference) => void;
}

const ThemeContext = createContext<ThemeValue | null>(null);

function systemAppearance(): ThemeAppearance {
  return Appearance.getColorScheme() === "dark" ? "dark" : "light";
}

function subscribeAppearance(notify: () => void): () => void {
  const appearance = Appearance.addChangeListener(notify);
  const appState = AppState.addEventListener("change", (state) => {
    if (state === "active") notify();
  });
  return () => {
    appearance.remove();
    appState.remove();
  };
}

function storedPreference(value: string | null): ThemePreference {
  if (value === null) return "system";
  if (value === "light" || value === "dark" || value === "system") return value;
  throw new Error("The stored theme choice is invalid.");
}

function errorDetail(error: unknown): string {
  return error instanceof Error ? error.message : String(error);
}

export function ThemeProvider({ children }: { children: React.ReactNode }): React.ReactElement {
  const osAppearance = useSyncExternalStore(subscribeAppearance, systemAppearance, systemAppearance);
  const [state, setState] = useState<ThemeState>({ preference: null, loading: true, saving: false, error: null });
  const mounted = useRef(false);
  const choiceVersion = useRef(0);
  const readVersion = useRef(0);
  const readFailed = useRef(false);
  const failedWrite = useRef<{ version: number; preference: ThemePreference } | null>(null);
  const writes = useRef<Promise<void>>(Promise.resolve());

  const restore = useCallback(async () => {
    const version = ++readVersion.current;
    const initialChoice = choiceVersion.current;
    setState((current) => ({ ...current, loading: true, error: null }));
    try {
      const preference = storedPreference(await SecureStore.getItemAsync(storageKey));
      if (!mounted.current || version !== readVersion.current || initialChoice !== choiceVersion.current) return;
      readFailed.current = false;
      setState({ preference, loading: false, saving: false, error: null });
    } catch (error) {
      if (!mounted.current || version !== readVersion.current || initialChoice !== choiceVersion.current) return;
      readFailed.current = true;
      setState({
        preference: null, loading: false, saving: false,
        error: `Could not load theme preference: ${errorDetail(error)} System appearance is active. Choose a theme to store a new preference.`,
      });
    }
  }, []);

  const select = useCallback((preference: ThemePreference) => {
    if (!mounted.current) return;
    const version = ++choiceVersion.current;
    readFailed.current = false;
    failedWrite.current = null;
    setState({ preference, loading: false, saving: true, error: null });
    writes.current = writes.current.then(async () => {
      try {
        await SecureStore.setItemAsync(storageKey, preference);
        if (!mounted.current || version !== choiceVersion.current) return;
        setState({ preference, loading: false, saving: false, error: null });
      } catch (error) {
        if (!mounted.current || version !== choiceVersion.current) return;
        failedWrite.current = { version, preference };
        setState({
          preference, loading: false, saving: false,
          error: `Could not save theme preference: ${errorDetail(error)} Your choice is applied for this session. Choose it again to retry.`,
        });
      }
    });
  }, []);

  useEffect(() => {
    mounted.current = true;
    void restore();
    const appState = AppState.addEventListener("change", (nextState) => {
      if (nextState !== "active") return;
      if (readFailed.current) void restore();
      const failed = failedWrite.current;
      if (failed && failed.version === choiceVersion.current) select(failed.preference);
    });
    return () => {
      mounted.current = false;
      readVersion.current += 1;
      appState.remove();
    };
  }, [restore, select]);

  useEffect(() => {
    Appearance.setColorScheme(state.preference === "light" || state.preference === "dark" ? state.preference : "unspecified");
  }, [state.preference]);

  useEffect(() => () => Appearance.setColorScheme("unspecified"), []);

  const appearance = state.preference === "light" || state.preference === "dark" ? state.preference : osAppearance;
  const value: ThemeValue = { ...state, appearance, colors: palettes[appearance], select };
  return React.createElement(ThemeContext.Provider, { value }, children);
}

export function useTheme(): ThemeValue {
  const theme = useContext(ThemeContext);
  if (theme === null) throw new Error("useTheme requires ThemeProvider.");
  return theme;
}
