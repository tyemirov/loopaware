declare module "react" {
  export type ReactNode = JSX.Element | string | number | boolean | null | undefined | ReactNode[];

  export type SetStateAction<State> = State | ((previousState: State) => State);

  export type Dispatch<Value> = (value: Value) => void;

  export type Context<Value> = {
    Provider: (props: { value: Value; children?: ReactNode }) => JSX.Element;
  };

  export function createContext<Value>(defaultValue: Value): Context<Value>;

  export function useContext<Value>(context: Context<Value>): Value;

  export function useMemo<Value>(factory: () => Value, dependencies: readonly unknown[]): Value;

  export function useState<State>(initialState: State | (() => State)): [State, Dispatch<SetStateAction<State>>];

  export function useState<State = undefined>(): [State | undefined, Dispatch<SetStateAction<State | undefined>>];

  const React: {
    createElement: (...args: unknown[]) => JSX.Element;
  };

  export default React;
}

declare namespace JSX {
  interface Element {}

  interface ElementChildrenAttribute {
    children: {};
  }

  interface IntrinsicAttributes {
    key?: string | number;
  }
}
