import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type KeyboardEvent,
} from "react";
import { createPortal } from "react-dom";
import { Icon } from "../Icon";
import type { MenuAction } from "./DropdownMenu";
import { useOverlayContainer } from "./Overlay";
import "./ui.css";

export function ContextMenu({
  x,
  y,
  items,
  onClose,
  label = "操作菜单",
}: {
  x: number;
  y: number;
  items: MenuAction[];
  onClose: () => void;
  label?: string;
}) {
  const container = useOverlayContainer();
  const menu = useRef<HTMLDivElement>(null);
  const previousFocus = useRef<HTMLElement | null>(null);
  const close = useRef(onClose);
  close.current = onClose;
  const [position, setPosition] = useState({ left: x, top: y });
  useLayoutEffect(() => {
    const node = menu.current;
    if (!node) return;
    const rect = node.getBoundingClientRect();
    setPosition({
      left: Math.max(8, Math.min(x, window.innerWidth - rect.width - 8)),
      top: Math.max(8, Math.min(y, window.innerHeight - rect.height - 8)),
    });
    if (!node.contains(document.activeElement))
      previousFocus.current = document.activeElement as HTMLElement;
    node
      .querySelector<HTMLButtonElement>("button:not(:disabled)")
      ?.focus({ preventScroll: true });
  }, [x, y]);
  useEffect(() => {
    const dismiss = (event: PointerEvent) => {
      if (!menu.current?.contains(event.target as Node)) close.current();
    };
    const resize = () => close.current();
    const scroll = (event: Event) => {
      if (!menu.current?.contains(event.target as Node)) close.current();
    };
    document.addEventListener("pointerdown", dismiss);
    window.addEventListener("resize", resize);
    document.addEventListener("scroll", scroll, true);
    return () => {
      document.removeEventListener("pointerdown", dismiss);
      window.removeEventListener("resize", resize);
      document.removeEventListener("scroll", scroll, true);
    };
  }, []);
  function keyDown(event: KeyboardEvent<HTMLDivElement>) {
    event.stopPropagation();
    if (event.key === "Escape" || event.key === "Tab") {
      if (event.key === "Escape") event.preventDefault();
      previousFocus.current?.focus({ preventScroll: true });
      onClose();
      return;
    }
    const buttons = [
      ...event.currentTarget.querySelectorAll<HTMLButtonElement>(
        "button:not(:disabled)",
      ),
    ];
    const index = buttons.indexOf(document.activeElement as HTMLButtonElement);
    let next: number;
    if (event.key === "Home") next = 0;
    else if (event.key === "End") next = buttons.length - 1;
    else if (event.key === "ArrowDown") next = (index + 1) % buttons.length;
    else if (event.key === "ArrowUp")
      next = (index - 1 + buttons.length) % buttons.length;
    else return;
    event.preventDefault();
    buttons[next]?.focus({ preventScroll: true });
  }
  return createPortal(
    <div
      ref={menu}
      className="ui-popover ui-context-menu"
      role="menu"
      aria-label={label}
      style={position}
      onKeyDown={keyDown}
      onContextMenu={(event) => event.preventDefault()}
    >
      {items.map((item, index) => (
        <div role="none" key={`${item.label}-${index}`}>
          {item.separator && <hr className="ui-menu-separator" />}
          <button
            type="button"
            role="menuitem"
            className={`ui-menu-item${item.danger ? " ui-menu-danger" : ""}`}
            disabled={item.disabled}
            onClick={() => {
              previousFocus.current?.focus({ preventScroll: true });
              onClose();
              item.onSelect();
            }}
          >
            {item.icon && <Icon name={item.icon} size={16} />}
            <span>{item.label}</span>
            {item.shortcut && <kbd>{item.shortcut}</kbd>}
          </button>
        </div>
      ))}
    </div>,
    container || document.body,
  );
}
