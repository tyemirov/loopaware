# LoopAware React Native Feedback

Native feedback button for Expo and React Native apps. This client submits customer feedback to `/public/mobile-feedback`. It is not part of LA Sentry error capture.

## Maintainer Publication

The repository lifecycle publishes this package from its prepared artifact.
Gateway captures `NPM_API_KEY` from the caller environment or the selected `.mprlab/deploy/.env` file.
The selected manifest owns that input source.
See [npm publication credentials](../../docs/npm-publication.md) for the loading contract and authentication diagnosis.

## Installation

```bash
npm install @loopaware/react-native
```

The package ships built ESM JavaScript and TypeScript declarations from `dist/`. React and React Native are peer
dependencies and must come from the host app.

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
      apiOrigin="https://loopaware-api.mprlab.com"
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

Keep only safe product state in the `context` object. Do not include passwords, payment data, tokens, or free-form private user content.

## Manual submission

Apps with their own UI can call `submitLoopAwareFeedback` directly:

```tsx
await submitLoopAwareFeedback(
  {
    siteId: "SITE_ID",
    mobileClientId: "MOBILE_CLIENT_ID",
    apiOrigin: "https://loopaware-api.mprlab.com",
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

## Non-React Native apps

For Swift, Kotlin, and other native clients, call the same `/public/mobile-feedback` REST endpoint directly.
Register the app with `/api/sites/{site_id}/mobile-apps`.
Store the public `client_id` and submit the JSON shape from `submitLoopAwareFeedback`.

## Package validation

Repository CI runs `make client-react-native-check`. That target typechecks the source, builds `dist/`, and packs the npm tarball.
It verifies the package contents and installs the tarball into a temporary consumer project.
It then typechecks a real `@loopaware/react-native` import.
