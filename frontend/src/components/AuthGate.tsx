import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import { api, ApiError, type AuthSession, type AuthStatus } from "../lib/api";
import { LoginPage } from "../pages/LoginPage";

function returnPath() {
  const requested = new URLSearchParams(window.location.search).get("next");
  return requested && /^\/[a-z-]*$/.test(requested) && requested !== "/login"
    ? requested
    : "/";
}

function showLogin() {
  if (window.location.pathname !== "/login") {
    const next = window.location.pathname;
    window.history.replaceState(
      null,
      "",
      `/login${next !== "/" ? `?next=${encodeURIComponent(next)}` : ""}`,
    );
  }
}

export function AuthGate({
  children,
}: {
  children: (session: AuthSession, logout: () => Promise<void>) => ReactNode;
}) {
  const [session, setSession] = useState<AuthSession | null>(null);
  const [status, setStatus] = useState<AuthStatus | null>(null);
  const [checking, setChecking] = useState(true);
  const [error, setError] = useState("");
  const revision = useRef(0);

  const check = useCallback(async () => {
    const current = ++revision.current;
    setChecking(true);
    setError("");
    const results = await Promise.allSettled([
      api.auth.status(),
      api.auth.session(),
    ]);
    if (current !== revision.current) return;
    const [config, account] = results;
    if (config.status === "fulfilled") setStatus(config.value);
    else setError("登录服务暂不可用，请确认后端已启动后重试。");
    if (
      config.status === "fulfilled" &&
      config.value.configured &&
      account.status === "fulfilled" &&
      !["local", "anonymous", "token", ""].includes(account.value.actor)
    ) {
      setSession(account.value);
      if (window.location.pathname === "/login")
        window.history.replaceState(null, "", returnPath());
    } else {
      setSession(null);
      showLogin();
      if (
        account.status === "rejected" &&
        !(
          account.reason instanceof ApiError && account.reason.status === 401
        ) &&
        config.status === "fulfilled"
      )
        setError("无法验证登录会话，请检查服务连接后重试。");
    }
    setChecking(false);
  }, []);

  useEffect(() => {
    void check();
    return () => {
      revision.current++;
    };
  }, [check]);
  useEffect(() => {
    const expired = () => {
      revision.current++;
      setSession(null);
      setChecking(false);
      setError("登录会话已失效，请重新登录。");
      showLogin();
    };
    window.addEventListener("camo:unauthorized", expired);
    return () => window.removeEventListener("camo:unauthorized", expired);
  }, []);

  useEffect(() => {
    const navigate = () => {
      if (!checking && !session) showLogin();
      else if (session && window.location.pathname === "/login")
        window.history.replaceState(null, "", returnPath());
    };
    window.addEventListener("popstate", navigate);
    return () => window.removeEventListener("popstate", navigate);
  }, [checking, session]);

  async function logout() {
    try {
      await api.auth.logout();
    } catch (err) {
      if (!(err instanceof ApiError && err.status === 401)) throw err;
    }
    sessionStorage.removeItem("camo-token");
    revision.current++;
    setSession(null);
    setError("");
    showLogin();
  }
  function loggedIn(account: AuthSession) {
    sessionStorage.removeItem("camo-token");
    window.history.replaceState(null, "", returnPath());
    setSession(account);
    setError("");
  }

  if (checking)
    return (
      <div className="login-checking" role="status">
        <span className="login-logo">C</span>
        <span className="login-spinner" />
        正在连接工作空间…
      </div>
    );
  if (!session)
    return (
      <LoginPage
        status={status}
        connectionError={error}
        onRetry={() => void check()}
        onLogin={loggedIn}
      />
    );
  return <>{children(session, logout)}</>;
}
