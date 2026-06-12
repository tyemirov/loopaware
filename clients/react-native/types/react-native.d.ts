declare module "react-native" {
  import type { ReactNode } from "react";

  type Component<Props> = (props: Props & { children?: ReactNode }) => JSX.Element;

  type ViewStyle = Record<string, unknown>;

  type TextStyle = Record<string, unknown>;

  type StyleValue = ViewStyle | TextStyle | Array<ViewStyle | TextStyle | null> | null;

  export const Modal: Component<{
    animationType?: "none" | "slide" | "fade";
    transparent?: boolean;
    visible?: boolean;
    onRequestClose?: () => void;
  }>;

  export const Platform: {
    OS: "ios" | "android" | "web" | "macos" | "windows";
  };

  export const Pressable: Component<{
    accessibilityRole?: string;
    disabled?: boolean;
    onPress?: () => void;
    style?: StyleValue;
  }>;

  export const StyleSheet: {
    create<Styles extends Record<string, ViewStyle | TextStyle>>(styles: Styles): Styles;
  };

  export const Text: Component<{
    style?: StyleValue;
  }>;

  export const TextInput: Component<{
    autoCapitalize?: string;
    keyboardType?: string;
    multiline?: boolean;
    onChangeText?: (value: string) => void;
    placeholder?: string;
    style?: StyleValue;
    value?: string;
  }>;

  export const View: Component<{
    style?: StyleValue;
  }>;
}
