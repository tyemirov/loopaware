import React, { createContext, useContext, useMemo, useState } from "react";
import type { ReactNode } from "react";
import {
  Modal,
  Platform,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from "react-native";

export type LoopAwareSentiment = "sad" | "neutral" | "happy";

export type LoopAwareContextValue =
  | string
  | number
  | boolean
  | null
  | LoopAwareContextValue[]
  | { [key: string]: LoopAwareContextValue };

export type LoopAwareFeedbackContext = { [key: string]: LoopAwareContextValue };

export type LoopAwareScreen = {
  name: string;
  path?: string;
};

export type LoopAwareAppMetadata = {
  platform?: "ios" | "android";
  applicationId: string;
  version?: string;
  build?: string;
  environment?: string;
};

export type LoopAwareConfig = {
  siteId: string;
  mobileClientId: string;
  apiOrigin: string;
  app: LoopAwareAppMetadata;
  defaultContext?: LoopAwareFeedbackContext;
};

export type LoopAwareProviderProps = LoopAwareConfig & {
  children: ReactNode;
};

export type LoopAwareFeedbackInput = {
  contact: string;
  message: string;
  sentiment?: LoopAwareSentiment;
  screen: LoopAwareScreen;
  context?: LoopAwareFeedbackContext;
};

export type LoopAwareFeedbackButtonProps = {
  screen: LoopAwareScreen;
  context?: LoopAwareFeedbackContext;
  buttonLabel?: string;
};

const LoopAwareContext = createContext<LoopAwareConfig | null>(null);

export function LoopAwareProvider({
  children,
  siteId,
  mobileClientId,
  apiOrigin,
  app,
  defaultContext,
}: LoopAwareProviderProps) {
  const value = useMemo(
    () => ({
      siteId,
      mobileClientId,
      apiOrigin: apiOrigin.replace(/\/+$/, ""),
      app,
      defaultContext,
    }),
    [siteId, mobileClientId, apiOrigin, app, defaultContext]
  );

  return <LoopAwareContext.Provider value={value}>{children}</LoopAwareContext.Provider>;
}

export function useLoopAware() {
  const config = useContext(LoopAwareContext);
  if (!config) {
    throw new Error("loopaware_provider_missing");
  }
  return config;
}

export async function submitLoopAwareFeedback(config: LoopAwareConfig, input: LoopAwareFeedbackInput) {
  const mergedContext = {
    ...(config.defaultContext ?? {}),
    ...(input.context ?? {}),
  };
  const platform = config.app.platform ?? (Platform.OS === "ios" || Platform.OS === "android" ? Platform.OS : "ios");
  const response = await fetch(`${config.apiOrigin.replace(/\/+$/, "")}/public/mobile-feedback`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      site_id: config.siteId,
      mobile_client_id: config.mobileClientId,
      contact: input.contact,
      message: input.message,
      sentiment: input.sentiment ?? "",
      screen: {
        name: input.screen.name,
        path: input.screen.path ?? "",
      },
      app: {
        platform,
        application_id: config.app.applicationId,
        version: config.app.version ?? "",
        build: config.app.build ?? "",
        environment: config.app.environment ?? "",
      },
      context: mergedContext,
    }),
  });
  if (!response.ok) {
    throw new Error(`loopaware_feedback_failed:${response.status}`);
  }
}

export function LoopAwareFeedbackButton({ screen, context, buttonLabel = "Feedback" }: LoopAwareFeedbackButtonProps) {
  const config = useLoopAware();
  const [open, setOpen] = useState(false);
  const [contact, setContact] = useState("");
  const [message, setMessage] = useState("");
  const [sentiment, setSentiment] = useState<LoopAwareSentiment | undefined>();
  const [status, setStatus] = useState("");
  const [submitting, setSubmitting] = useState(false);

  async function handleSubmit() {
    const normalizedContact = contact.trim();
    const normalizedMessage = message.trim();
    if (!normalizedContact || (!normalizedMessage && !sentiment)) {
      setStatus("Add contact and feedback.");
      return;
    }
    setSubmitting(true);
    setStatus("Sending...");
    try {
      await submitLoopAwareFeedback(config, {
        contact: normalizedContact,
        message: normalizedMessage,
        sentiment,
        screen,
        context,
      });
      setContact("");
      setMessage("");
      setSentiment(undefined);
      setStatus("Sent.");
      setOpen(false);
    } catch {
      setStatus("Unable to send feedback.");
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <>
      <Pressable accessibilityRole="button" onPress={() => setOpen(true)} style={styles.floatingButton}>
        <Text style={styles.floatingButtonText}>{buttonLabel}</Text>
      </Pressable>
      <Modal animationType="slide" transparent visible={open} onRequestClose={() => setOpen(false)}>
        <View style={styles.modalBackdrop}>
          <View style={styles.sheet}>
            <Text style={styles.title}>Send feedback</Text>
            <TextInput
              autoCapitalize="none"
              keyboardType="email-address"
              onChangeText={setContact}
              placeholder="Email or phone"
              style={styles.input}
              value={contact}
            />
            <View style={styles.sentimentRow}>
              {(["sad", "neutral", "happy"] as LoopAwareSentiment[]).map((value) => (
                <Pressable
                  accessibilityRole="button"
                  key={value}
                  onPress={() => setSentiment(value)}
                  style={[styles.sentimentButton, sentiment === value ? styles.sentimentButtonActive : null]}
                >
                  <Text style={styles.sentimentText}>{value}</Text>
                </Pressable>
              ))}
            </View>
            <TextInput
              multiline
              onChangeText={setMessage}
              placeholder="Your message"
              style={[styles.input, styles.messageInput]}
              value={message}
            />
            {status ? <Text style={styles.status}>{status}</Text> : null}
            <View style={styles.actions}>
              <Pressable accessibilityRole="button" onPress={() => setOpen(false)} style={styles.secondaryButton}>
                <Text style={styles.secondaryButtonText}>Cancel</Text>
              </Pressable>
              <Pressable
                accessibilityRole="button"
                disabled={submitting}
                onPress={handleSubmit}
                style={[styles.primaryButton, submitting ? styles.disabledButton : null]}
              >
                <Text style={styles.primaryButtonText}>Send</Text>
              </Pressable>
            </View>
          </View>
        </View>
      </Modal>
    </>
  );
}

const styles = StyleSheet.create({
  actions: {
    flexDirection: "row",
    gap: 12,
    justifyContent: "flex-end",
  },
  disabledButton: {
    opacity: 0.6,
  },
  floatingButton: {
    alignItems: "center",
    backgroundColor: "#0d6efd",
    borderRadius: 24,
    bottom: 24,
    elevation: 4,
    paddingHorizontal: 18,
    paddingVertical: 12,
    position: "absolute",
    right: 20,
  },
  floatingButtonText: {
    color: "#ffffff",
    fontWeight: "700",
  },
  input: {
    borderColor: "#ced4da",
    borderRadius: 8,
    borderWidth: 1,
    marginBottom: 12,
    paddingHorizontal: 12,
    paddingVertical: 10,
  },
  messageInput: {
    minHeight: 110,
    textAlignVertical: "top",
  },
  modalBackdrop: {
    backgroundColor: "rgba(0, 0, 0, 0.35)",
    flex: 1,
    justifyContent: "flex-end",
  },
  primaryButton: {
    backgroundColor: "#0d6efd",
    borderRadius: 8,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  primaryButtonText: {
    color: "#ffffff",
    fontWeight: "700",
  },
  secondaryButton: {
    borderColor: "#6c757d",
    borderRadius: 8,
    borderWidth: 1,
    paddingHorizontal: 16,
    paddingVertical: 10,
  },
  secondaryButtonText: {
    color: "#343a40",
    fontWeight: "700",
  },
  sentimentButton: {
    borderColor: "#ced4da",
    borderRadius: 8,
    borderWidth: 1,
    paddingHorizontal: 12,
    paddingVertical: 8,
  },
  sentimentButtonActive: {
    backgroundColor: "#e7f1ff",
    borderColor: "#0d6efd",
  },
  sentimentRow: {
    flexDirection: "row",
    gap: 8,
    marginBottom: 12,
  },
  sentimentText: {
    textTransform: "capitalize",
  },
  sheet: {
    backgroundColor: "#ffffff",
    borderTopLeftRadius: 16,
    borderTopRightRadius: 16,
    padding: 20,
  },
  status: {
    color: "#495057",
    marginBottom: 12,
  },
  title: {
    fontSize: 18,
    fontWeight: "700",
    marginBottom: 16,
  },
});
