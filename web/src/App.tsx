import { Spin } from "antd";
import { Suspense, lazy, useEffect } from "react";
import { useAuthStore } from "./modules/auth/store";

const AppRouter = lazy(async () => ({
  default: (await import("./app/router/AppRouter")).AppRouter,
}));

const LoginPage = lazy(async () => ({
  default: (await import("./pages/LoginPage")).LoginPage,
}));

export default function App() {
  const user = useAuthStore((state) => state.user);
  const checking = useAuthStore((state) => state.checking);
  const loadMe = useAuthStore((state) => state.loadMe);

  useEffect(() => {
    void loadMe();
  }, [loadMe]);

  if (checking) {
    return <Spin size="large" style={{ margin: "20vh auto", display: "block" }} />;
  }

  return (
    <Suspense fallback={<Spin size="large" style={{ margin: "20vh auto", display: "block" }} />}>
      {user ? <AppRouter /> : <LoginPage />}
    </Suspense>
  );
}
