import { useEffect, useId, useRef, useState, type ReactNode } from "react";
import * as stylex from "@stylexjs/stylex";
import { Icon } from "./Icon";
import { IconButton } from "./ui/IconButton";
import { OverlayContainer } from "./ui/Overlay";
import "./ui/ui.css";

const styles = stylex.create({
  dialog: {
    borderWidth: 1,
    borderStyle: "solid",
    borderColor: "var(--border-color)",
    padding: 0,
    borderRadius: "var(--radius-modal)",
    width: 760,
    maxWidth: "calc(100vw - 36px)",
    color: "var(--text-primary)",
    backgroundColor: "var(--card-bg)",
    boxShadow: "var(--shadow-modal)",
    maxHeight: "calc(100dvh - 40px)",
    overflow: "hidden",
    "::backdrop": {
      backgroundColor: "rgba(15, 23, 42, 0.4)",
      backdropFilter: "blur(4px)",
    },
  },
  dialogSimple: {
    padding: { default: "24px 28px", "@media (max-width: 600px)": "20px" },
    width: 560,
    overflowY: "auto",
  },
  structured: { overflowY: "auto", width: 520 },
  wide: { width: 1020 },
  structuredHeader: { padding: "16px 20px", marginBottom: 0 },
  header: {
    display: "flex",
    justifyContent: "space-between",
    alignItems: "center",
    padding: "0 0 16px",
    borderBottomWidth: 1,
    borderBottomStyle: "solid",
    borderBottomColor: "var(--border-subtle)",
    marginBottom: 16,
  },
  title: {
    fontSize: "calc(var(--font-size) + 1px)",
    fontWeight: 600,
    color: "var(--text-primary)",
  },
});

export function Modal({
  title,
  onClose,
  bare = false,
  structured = false,
  wide = false,
  children,
}: {
  title?: string;
  onClose: () => void;
  bare?: boolean;
  structured?: boolean;
  wide?: boolean;
  children: ReactNode;
}) {
  const dialog = useRef<HTMLDialogElement>(null);
  const [container, setContainer] = useState<HTMLElement | null>(null);
  const titleId = useId();

  useEffect(() => {
    const node = dialog.current;
    setContainer(node);
    const previousFocus = document.activeElement as HTMLElement | null;
    const previousOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    node?.showModal();
    node?.querySelector<HTMLElement>("[autofocus], [data-autofocus]")?.focus();
    return () => {
      node?.close();
      document.body.style.overflow = previousOverflow;
      previousFocus?.focus();
    };
  }, []);

  return (
    <dialog
      ref={dialog}
      {...stylex.props(
        styles.dialog,
        !bare && !structured && styles.dialogSimple,
        structured && styles.structured,
        wide && styles.wide,
      )}
      aria-labelledby={title ? titleId : undefined}
      aria-label={title ? undefined : "外观与连接设置"}
      onCancel={(event) => {
        event.preventDefault();
        onClose();
      }}
      onClick={(event) => {
        if (event.target === dialog.current) {
          const rect = dialog.current!.getBoundingClientRect();
          if (
            event.clientX < rect.left ||
            event.clientX > rect.right ||
            event.clientY < rect.top ||
            event.clientY > rect.bottom
          ) {
            onClose();
          }
        }
      }}
    >
      <OverlayContainer.Provider value={container}>
        {!bare && (
          <div
            {...stylex.props(
              styles.header,
              structured && styles.structuredHeader,
            )}
          >
            <h2 id={titleId} {...stylex.props(styles.title)}>
              {title}
            </h2>
            <IconButton aria-label="关闭" onClick={onClose}>
              <Icon name="close" size={16} />
            </IconButton>
          </div>
        )}
        {children}
      </OverlayContainer.Provider>
    </dialog>
  );
}

export function ModalFooter({ children }: { children: ReactNode }) {
  return <footer className="ui-modal-footer">{children}</footer>;
}
