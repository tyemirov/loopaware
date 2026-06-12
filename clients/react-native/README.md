# LoopAware React Native Feedback

Native feedback button for Expo and React Native apps. This client submits customer feedback to `/public/mobile-feedback`; it is not part of LA Sentry error capture.

## Expo setup

Register the mobile app for a LoopAware site through the authenticated API:

```http
POST /api/sites/{site_id}/mobile-apps
```

```json
{
  "platform": "ios",
  "app_identifier": "com.example.app",
  "display_name": "Example iOS"
}
```

The response includes a public `client_id`. Ship that value with the app alongside the site ID.

```tsx
import * as Application from "expo-application";
import { LoopAwareFeedbackButton, LoopAwareProvider } from "@loopaware/react-native";

export function App() {
  return (
    <LoopAwareProvider
      siteId="SITE_ID"
      mobileClientId="MOBILE_CLIENT_ID"
      apiOrigin="https://loopaware.mprlab.com"
      app={{
        applicationId: Application.applicationId ?? "com.example.app",
        version: Application.nativeApplicationVersion ?? "",
        build: Application.nativeBuildVersion ?? "",
        environment: "production",
      }}
    >
      <CheckoutScreen />
    </LoopAwareProvider>
  );
}

function CheckoutScreen() {
  return (
    <>
      <CheckoutContent />
      <LoopAwareFeedbackButton
        screen={{ name: "Checkout", path: "/checkout/payment" }}
        context={{ step: "payment", plan: "pro" }}
      />
    </>
  );
}
```

The `context` object should contain only safe product state. Do not include passwords, payment data, tokens, or free-form private user content.

## Manual submission

Apps with their own UI can call `submitLoopAwareFeedback` directly:

```tsx
await submitLoopAwareFeedback(
  {
    siteId: "SITE_ID",
    mobileClientId: "MOBILE_CLIENT_ID",
    apiOrigin: "https://loopaware.mprlab.com",
    app: {
      platform: "android",
      applicationId: "com.example.app",
      version: "1.2.3",
      build: "44",
      environment: "production",
    },
  },
  {
    contact: "person@example.com",
    message: "The checkout button is confusing.",
    sentiment: "sad",
    screen: { name: "Checkout", path: "/checkout/payment" },
    context: { step: "payment" },
  }
);
```
