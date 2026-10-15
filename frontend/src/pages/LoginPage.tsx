import { useRef, useState, type FormEvent } from "react";
import { api, type AuthSession, type AuthStatus } from "../lib/api";
import { Icon } from "../components/Icon";
import "./login.css";

export function LoginPage({
  status,
  connectionError,
  onRetry,
  onLogin,
}: {
  status: AuthStatus | null;
  connectionError: string;
  onRetry: () => void;
  onLogin: (session: AuthSession) => void;
}) {
  const [username, setUsername] = useState(
    () => localStorage.getItem("camo-login-account") || "",
  );
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [remember, setRemember] = useState(
    () => !!localStorage.getItem("camo-login-account"),
  );
  const [showPassword, setShowPassword] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [help, setHelp] = useState(false);
  const submitting = useRef(false);
  const ready = !!status?.configured;

  async function submit(event: FormEvent) {
    event.preventDefault();
    if (submitting.current || !ready) return;
    submitting.current = true;
    setBusy(true);
    setError("");
    try {
      const result = await api.auth.login(username.trim(), password, code);
      if (remember) localStorage.setItem("camo-login-account", username.trim());
      else localStorage.removeItem("camo-login-account");
      setPassword("");
      setCode("");
      onLogin({ actor: result.username, role: result.role });
    } catch (err) {
      setError(err instanceof Error ? err.message : "登录失败，请稍后重试");
    } finally {
      submitting.current = false;
      setBusy(false);
    }
  }

  return (
    <main className="login-shell">
      <section className="login-story" aria-label="CamoPanel">
        <a className="login-brand" href="/login" aria-label="CamoPanel 登录">
          <span className="login-logo">C</span>
          <span>
            CamoPanel<small>SERVER CONTROL</small>
          </span>
        </a>
        <div className="login-story-content">
          <p className="login-eyebrow">
            <span />
            YOUR SERVER. YOUR WORKSPACE.
          </p>
          <h1>
            掌控服务，
            <br />
            从容运维。
          </h1>
          <p className="login-story-copy">
            让服务器、网站与应用井然有序。
            <br />
            一个工作空间，处理日常运维。
          </p>
          <div className="login-visual" aria-hidden="true">
            <div className="login-orbit login-orbit-one" />
            <div className="login-orbit login-orbit-two" />
            <div className="login-rack">
              <div className="login-rack-label">
                <span>CAMOPANEL</span>
                <Icon name="server" size={15} />
              </div>
              {[0, 1, 2].map((index) => (
                <div className="login-rack-unit" key={index}>
                  <span className="login-rack-dot" />
                  <div className="login-rack-lines">
                    <i />
                    <i />
                  </div>
                  <span className="login-rack-slots">···</span>
                </div>
              ))}
              <div className="login-rack-footer">
                <span>SERVER / WORKSPACE</span>
                <span>01</span>
              </div>
            </div>
            <div className="login-visual-tile login-tile-folder">
              <Icon name="folder" size={21} />
            </div>
            <div className="login-visual-tile login-tile-globe">
              <Icon name="globe" size={22} />
            </div>
          </div>
        </div>
        <footer className="login-story-footer">
          <span>为更从容的运维而构建</span>
          <span>© {new Date().getFullYear()} CamoPanel</span>
        </footer>
      </section>
      <section className="login-form-panel" aria-label="账号登录">
        <div className="login-mobile-brand">
          <span className="login-logo">C</span>
          <strong>CamoPanel</strong>
        </div>
        <div className="login-form-content">
          <div className="login-entry-label">
            <Icon name="lock" size={14} />
            <span>工作空间入口</span>
          </div>
          <h2>欢迎回来</h2>
          <p className="login-description">
            登录 CamoPanel，进入你的服务器工作空间。
          </p>
          {(error || connectionError) && (
            <div className="login-alert" role="alert">
              <Icon name="alertCircle" size={16} />
              <span>{error || connectionError}</span>
              {!error && (
                <button type="button" onClick={onRetry}>
                  重新连接
                </button>
              )}
            </div>
          )}
          {status && !ready && !connectionError && (
            <div className="login-alert" role="alert">
              <Icon name="info" size={16} />
              <span>此工作空间尚未配置登录账号，请联系管理员完成配置。</span>
              <button type="button" onClick={onRetry}>
                刷新
              </button>
            </div>
          )}
          <form onSubmit={submit}>
            <fieldset className="login-fields" disabled={busy || !ready}>
              <label className="login-field">
                <span>账号</span>
                <div className="login-input-group">
                  <Icon name="user" size={17} />
                  <input
                    name="username"
                    autoFocus
                    required
                    autoComplete="username"
                    autoCapitalize="none"
                    spellCheck={false}
                    placeholder="输入登录账号"
                    value={username}
                    onChange={(event) => {
                      setUsername(event.target.value);
                      setError("");
                    }}
                  />
                </div>
              </label>
              <label className="login-field">
                <span>密码</span>
                <div className="login-input-group">
                  <Icon name="lock" size={16} />
                  <input
                    name="password"
                    type={showPassword ? "text" : "password"}
                    required
                    autoComplete="current-password"
                    placeholder="输入登录密码"
                    value={password}
                    onChange={(event) => {
                      setPassword(event.target.value);
                      setError("");
                    }}
                  />
                  <button
                    className="login-password-toggle"
                    type="button"
                    aria-label={showPassword ? "隐藏密码" : "显示密码"}
                    aria-pressed={showPassword}
                    onClick={() => setShowPassword((previous) => !previous)}
                  >
                    <Icon name={showPassword ? "eyeOff" : "eye"} size={17} />
                  </button>
                </div>
              </label>
              {status?.totp_enabled && (
                <label className="login-field">
                  <span>动态验证码</span>
                  <div className="login-input-group">
                    <Icon name="shield" size={16} />
                    <input
                      name="code"
                      required
                      inputMode="numeric"
                      autoComplete="one-time-code"
                      maxLength={6}
                      pattern="[0-9]{6}"
                      placeholder="输入验证器中的 6 位验证码"
                      value={code}
                      onChange={(event) => setCode(event.target.value)}
                    />
                  </div>
                </label>
              )}
              <div className="login-options">
                <label className="login-remember">
                  <input
                    type="checkbox"
                    checked={remember}
                    onChange={(event) => setRemember(event.target.checked)}
                  />
                  <span>记住账号</span>
                </label>
                <button
                  className="login-help-button"
                  type="button"
                  onClick={() => setHelp((previous) => !previous)}
                >
                  忘记密码？
                </button>
              </div>
              {help && (
                <p className="login-help" role="status">
                  请联系服务器管理员重置登录密码，再使用新密码登录。
                </p>
              )}
              <button className="login-submit" type="submit">
                {busy ? (
                  <>
                    <span className="login-spinner" />
                    正在登录…
                  </>
                ) : (
                  <>
                    <span>登录工作空间</span>
                    <Icon name="arrow" size={17} />
                  </>
                )}
              </button>
            </fieldset>
          </form>
          <div className="login-trust">
            <Icon name="shield" size={14} />
            <span>仅授权用户可访问此工作空间</span>
          </div>
        </div>
        <footer className="login-form-footer">
          <span>CamoPanel</span>
          <span>v0.1.0</span>
        </footer>
      </section>
    </main>
  );
}
