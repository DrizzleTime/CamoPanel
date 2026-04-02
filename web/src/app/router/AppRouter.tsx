import { Spin } from "antd";
import { Suspense, lazy } from "react";
import { Navigate, Route, Routes } from "react-router-dom";

const ShellLayout = lazy(async () => ({
  default: (await import("../../widgets/shell/ShellLayout")).ShellLayout,
}));

const DashboardPage = lazy(async () => ({
  default: (await import("../../pages/DashboardPage")).DashboardPage,
}));

const StorePage = lazy(async () => ({
  default: (await import("../../pages/StorePage")).StorePage,
}));

const WebsitesPage = lazy(async () => ({
  default: (await import("../../pages/WebsitesPage")).WebsitesPage,
}));

const DatabasesPage = lazy(async () => ({
  default: (await import("../../pages/DatabasesPage")).DatabasesPage,
}));

const SchedulesPage = lazy(async () => ({
  default: (await import("../../pages/SchedulesPage")).SchedulesPage,
}));

const FilesPage = lazy(async () => ({
  default: (await import("../../pages/FilesPage")).FilesPage,
}));

const ContainersPage = lazy(async () => ({
  default: (await import("../../pages/ContainersPage")).ContainersPage,
}));

const CopilotPage = lazy(async () => ({
  default: (await import("../../pages/CopilotPage")).CopilotPage,
}));

export function AppRouter() {
  return (
    <Suspense fallback={<Spin size="large" style={{ margin: "20vh auto", display: "block" }} />}>
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
    </Suspense>
  );
}
