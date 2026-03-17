import { Spin } from "antd";
import { useEffect } from "react";
import { Navigate, Route, Routes } from "react-router-dom";
import { ShellLayout } from "./components/ShellLayout";
import { ContainersPage } from "./pages/ContainersPage";
import { CopilotPage } from "./pages/CopilotPage";
import { DatabasesPage } from "./pages/DatabasesPage";
import { DashboardPage } from "./pages/DashboardPage";
import { FilesPage } from "./pages/FilesPage";
import { LoginPage } from "./pages/LoginPage";
import { SchedulesPage } from "./pages/SchedulesPage";
import { StorePage } from "./pages/StorePage";
import { WebsitesPage } from "./pages/WebsitesPage";
import { useAuthStore } from "./store/auth";

function ProtectedRoutes() {
  return (
    <Routes>
      <Route element={<ShellLayout />}>
        <Route path="/app/dashboard" element={<DashboardPage />} />
        <Route path="/app/store" element={<StorePage />} />
        <Route path="/app/websites" element={<WebsitesPage />} />
        <Route path="/app/databases" element={<DatabasesPage />} />
        <Route path="/app/schedules" element={<SchedulesPage />} />
        <Route path="/app/files" element={<FilesPage />} />
        <Route path="/app/containers" element={<ContainersPage />} />
        <Route path="/app/copilot" element={<CopilotPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/app/dashboard" replace />} />
    </Routes>
  );
}

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

  return user ? <ProtectedRoutes /> : <LoginPage />;
}
