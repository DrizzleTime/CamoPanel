import { Typography } from "antd";
import { createContext, useContext, useEffect, type Dispatch, type ReactNode, type SetStateAction } from "react";

type ShellHeaderSetter = Dispatch<SetStateAction<ReactNode | null>>;

export const ShellHeaderContext = createContext<ShellHeaderSetter | null>(null);

export function useShellHeader(content: ReactNode | null) {
  const setHeaderContent = useContext(ShellHeaderContext);

  useEffect(() => {
    if (!setHeaderContent) {
      return;
    }

    setHeaderContent(content);
  }, [content, setHeaderContent]);

  useEffect(() => {
    if (!setHeaderContent) {
      return;
    }

    return () => setHeaderContent(null);
  }, [setHeaderContent]);
}

type ShellPageMetaProps = {
  title: string;
  description?: string;
};

export function ShellPageMeta({ title, description }: ShellPageMetaProps) {
  return (
    <div className="shell-page-meta">
      <Typography.Text className="shell-page-title">{title}</Typography.Text>
      {description ? (
        <Typography.Text type="secondary" className="shell-page-description">
          {description}
        </Typography.Text>
      ) : null}
    </div>
  );
}
