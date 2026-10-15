import * as Primitive from "@radix-ui/react-tooltip";
import type { ReactElement } from "react";
import { useOverlayContainer } from "./Overlay";
import "./ui.css";

export function Tooltip({
  content,
  children,
}: {
  content: string;
  children: ReactElement;
}) {
  const container = useOverlayContainer();
  return (
    <Primitive.Provider delayDuration={450} skipDelayDuration={150}>
      <Primitive.Root>
        <Primitive.Trigger asChild>{children}</Primitive.Trigger>
        <Primitive.Portal container={container || undefined}>
          <Primitive.Content
            className="ui-tooltip"
            sideOffset={7}
            collisionPadding={12}
          >
            {content}
            <Primitive.Arrow className="ui-tooltip-arrow" />
          </Primitive.Content>
        </Primitive.Portal>
      </Primitive.Root>
    </Primitive.Provider>
  );
}
