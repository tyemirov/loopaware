import { useCallback, useEffect, useRef, useState } from "react";
import { AppState } from "react-native";
import type { LoopAwareApiClient } from "./api";
import type { Account } from "./types";

type ResourceImage =
  | { status: "loading" }
  | { status: "unavailable" }
  | { status: "failed"; message: string }
  | { status: "loaded"; uri: string; generation: number };

export function useAccountPhoto(account: Account, client: LoopAwareApiClient): ResourceImage {
  const load = useCallback((signal: AbortSignal) => client.accountAvatar(account, signal), [account.email, account.avatar?.url, client]);
  return useResourceImage(load);
}

export function useResourceImage(loadImage: (signal: AbortSignal) => Promise<string | null>): ResourceImage {
  const [photo, setPhoto] = useState<ResourceImage>({ status: "loading" });
  const generation = useRef(0);
  useEffect(() => {
    let currentRequest: AbortController | undefined;
    let previousState = AppState.currentState;
    const load = () => {
      currentRequest?.abort();
      const request = new AbortController();
      currentRequest = request;
      const loadGeneration = ++generation.current;
      setPhoto({ status: "loading" });
      void loadImage(request.signal).then(uri => {
        if (!request.signal.aborted) setPhoto(uri ? { status: "loaded", uri, generation: loadGeneration } : { status: "unavailable" });
      }, error => {
        console.error("IMAGE_DIAGNOSTIC", error);
        if (!request.signal.aborted) {
          setPhoto({ status: "failed", message: error instanceof Error ? error.message : "Load image failed." });
        }
      });
    };
    load();
    const subscription = AppState.addEventListener("change", state => {
      if (state === "active" && previousState !== "active") load();
      previousState = state;
    });
    return () => { subscription.remove(); currentRequest?.abort(); };
  }, [loadImage]);
  return photo;
}
