import { useCallback, useEffect, useRef, useState } from "react";
import { Platform } from "react-native";

type Destination = "avatar" | "settings";
type DismissalCompletion = "native" | "frames";

function dismissalCompletion(): DismissalCompletion {
  switch (Platform.OS) {
    case "ios": return "native";
    case "android": return "frames";
    default: throw new Error(`Account menu requires iOS or Android: ${Platform.OS}`);
  }
}

export function useAccountMenu(onSettings: () => void, focusAvatar: () => void): {
  visible: boolean;
  open: () => void;
  close: (destination: Destination) => void;
  onDismiss: () => void;
} {
  const [visible, setVisible] = useState(false);
  const pending = useRef<Destination | null>(null);
  const callbacks = useRef({ onSettings, focusAvatar });
  callbacks.current = { onSettings, focusAvatar };
  const frame = useRef<ReturnType<typeof requestAnimationFrame> | null>(null);
  const completion = dismissalCompletion();

  const cancelFrames = useCallback(() => {
    if (frame.current !== null) {
      cancelAnimationFrame(frame.current);
      frame.current = null;
    }
  }, []);

  const finishDismissal = useCallback(() => {
    const destination = pending.current;
    pending.current = null;
    if (destination === "avatar") callbacks.current.focusAvatar();
    if (destination === "settings") callbacks.current.onSettings();
  }, []);

  const open = useCallback(() => {
    cancelFrames();
    pending.current = null;
    setVisible(true);
  }, [cancelFrames]);

  const close = useCallback((destination: Destination) => {
    pending.current = destination;
    setVisible(false);
  }, []);

  const onDismiss = useCallback(() => {
    if (completion === "native") finishDismissal();
  }, [completion, finishDismissal]);

  useEffect(() => {
    if (!visible && completion === "frames" && pending.current !== null) {
      frame.current = requestAnimationFrame(() => {
        frame.current = requestAnimationFrame(() => {
          frame.current = null;
          finishDismissal();
        });
      });
    }
    return cancelFrames;
  }, [visible, completion, cancelFrames, finishDismissal]);

  useEffect(() => () => {
    pending.current = null;
    cancelFrames();
  }, [cancelFrames]);

  return { visible, open, close, onDismiss };
}
