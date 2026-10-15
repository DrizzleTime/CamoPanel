import * as Menu from "@radix-ui/react-dropdown-menu";
import type { ReactElement } from "react";
import { Icon, type IconName } from "../Icon";
import { useOverlayContainer } from "./Overlay";
import "./ui.css";

export interface MenuAction {
  label: string;
  icon?: IconName;
  shortcut?: string;
  onSelect: () => void;
  disabled?: boolean;
  danger?: boolean;
  separator?: boolean;
}

export function DropdownMenu({
  trigger,
  items,
  label = "操作菜单",
  align = "start",
}: {
  trigger: ReactElement;
  items: MenuAction[];
  label?: string;
  align?: "start" | "end";
}) {
  const container = useOverlayContainer();
  return (
    <Menu.Root modal={false}>
      <Menu.Trigger asChild>{trigger}</Menu.Trigger>
      <Menu.Portal container={container || undefined}>
        <Menu.Content
          className="ui-popover ui-menu"
          aria-label={label}
          sideOffset={6}
          collisionPadding={12}
          align={align}
          onCloseAutoFocus={(event) => {
            if (document.activeElement?.closest("dialog[open]"))
              event.preventDefault();
          }}
        >
          {items.map((item, index) => (
            <MenuEntry key={`${item.label}-${index}`} item={item} />
          ))}
        </Menu.Content>
      </Menu.Portal>
    </Menu.Root>
  );
}

function MenuEntry({ item }: { item: MenuAction }) {
  return (
    <>
      {item.separator && <Menu.Separator className="ui-menu-separator" />}
      <Menu.Item
        className={`ui-menu-item${item.danger ? " ui-menu-danger" : ""}`}
        disabled={item.disabled}
        onSelect={item.onSelect}
      >
        {item.icon && <Icon name={item.icon} size={16} />}
        <span>{item.label}</span>
        {item.shortcut && <kbd>{item.shortcut}</kbd>}
      </Menu.Item>
    </>
  );
}
