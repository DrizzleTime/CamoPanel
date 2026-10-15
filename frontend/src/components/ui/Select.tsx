import * as SelectPrimitive from "@radix-ui/react-select";
import { type ComponentPropsWithoutRef } from "react";
import { Icon } from "../Icon";
import { useOverlayContainer } from "./Overlay";
import "./ui.css";

export interface SelectOption {
  value: string;
  label: string;
  disabled?: boolean;
}
export interface SelectProps extends Omit<
  ComponentPropsWithoutRef<typeof SelectPrimitive.Trigger>,
  "children" | "onChange" | "defaultValue"
> {
  options: SelectOption[];
  value?: string;
  defaultValue?: string;
  onValueChange?: (value: string) => void;
  placeholder?: string;
  name?: string;
  required?: boolean;
}

export function Select({
  options,
  value,
  defaultValue,
  onValueChange,
  placeholder = "请选择",
  name,
  required,
  disabled,
  className = "",
  ...props
}: SelectProps) {
  const container = useOverlayContainer();
  return (
    <SelectPrimitive.Root
      value={value}
      defaultValue={defaultValue}
      onValueChange={onValueChange}
      name={name}
      required={required}
      disabled={disabled}
    >
      <SelectPrimitive.Trigger
        {...props}
        className={`ui-input ui-select ${className}`}
      >
        <SelectPrimitive.Value placeholder={placeholder} />
        <SelectPrimitive.Icon className="ui-select-chevron">
          <Icon name="down" size={16} />
        </SelectPrimitive.Icon>
      </SelectPrimitive.Trigger>
      <SelectPrimitive.Portal container={container || undefined}>
        <SelectPrimitive.Content
          className="ui-popover ui-select-content"
          position="popper"
          sideOffset={6}
          collisionPadding={12}
        >
          <SelectPrimitive.ScrollUpButton className="ui-select-scroll">
            <Icon name="up" size={14} />
          </SelectPrimitive.ScrollUpButton>
          <SelectPrimitive.Viewport className="ui-select-viewport">
            {options.map((option) => (
              <SelectPrimitive.Item
                className="ui-menu-item ui-select-item"
                key={option.value}
                value={option.value}
                disabled={option.disabled}
                textValue={option.label}
              >
                <SelectPrimitive.ItemText>
                  {option.label}
                </SelectPrimitive.ItemText>
                <SelectPrimitive.ItemIndicator className="ui-select-indicator">
                  <Icon name="check" size={16} />
                </SelectPrimitive.ItemIndicator>
              </SelectPrimitive.Item>
            ))}
          </SelectPrimitive.Viewport>
          <SelectPrimitive.ScrollDownButton className="ui-select-scroll">
            <Icon name="down" size={14} />
          </SelectPrimitive.ScrollDownButton>
        </SelectPrimitive.Content>
      </SelectPrimitive.Portal>
    </SelectPrimitive.Root>
  );
}
