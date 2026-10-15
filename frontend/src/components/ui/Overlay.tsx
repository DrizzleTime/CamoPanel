import { createContext, useContext } from "react";

// Keep floating content inside a native dialog's top layer and focus boundary.
export const OverlayContainer = createContext<HTMLElement | null>(null);
export function useOverlayContainer() {
  return useContext(OverlayContainer);
}
