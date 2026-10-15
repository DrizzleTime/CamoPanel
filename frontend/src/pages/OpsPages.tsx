import { useCallback, useEffect, useState } from "react";
import {
  api,
  type AiReply,
  type AlertRule,
  type BackupItem,
  type Certificate,
  type DatabaseInfo,
  type FirewallStatus,
  type ScheduleJob,
} from "../lib/api";
import { Icon } from "../components/Icon";
import { Input, Textarea } from "../components/ui/Input";
import { Select } from "../components/ui/Select";
import { Checkbox } from "../components/ui/Checkbox";
import { Button } from "../components/ui/Button";
import {
  OpsPage,
  OpsCard,
  OpsField,
  OpsEmpty,
  OpsStats,
  OpsTag,
  OpsLoading,
  opsDate,
  useOpsFeedback,
} from "../components/ui/OpsUI";

function useInitialLoad(
  load: () => Promise<void>,
  setError: (message: string) => void,
) {
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");
    void load()
      .catch((err) => {
        if (active)
          setError(err instanceof Error ? err.message : "加载失败，请重试");
      })
      .finally(() => {
        if (active) setLoading(false);
      });
    return () => {
      active = false;
    };
  }, [load, setError]);
  return loading;
}

const splitValues = (value: string) =>
  value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
const DATABASE_NAMES: Record<string, string> = {
  mysql: "MySQL",
  postgres: "PostgreSQL",
  redis: "Redis",
};
const JOB_NAMES: Record<string, string> = {
  backup_site: "网站备份",
  backup_app: "应用备份",
  backup_database: "数据库备份",
  backup_directory: "目录备份",
  renew_certs: "证书续签",
  rotate_logs: "日志轮转",
  fetch_url: "访问 URL",
  script: "执行脚本",
};

export function CertificatesPage() {
  const feedback = useOpsFeedback();
  const [list, setList] = useState<Certificate[]>([]);
  const [domain, setDomain] = useState("");
  const [selected, setSelected] = useState("");
  const [sites, setSites] = useState("");
  const [provider, setProvider] = useState("cloudflare");
  const [apiBase, setApiBase] = useState("");
  const [zone, setZone] = useState("");
  const [token, setToken] = useState("");
  const [dnsSet, setDnsSet] = useState(false);
  const [txtName, setTxtName] = useState("");
  const [txtValue, setTxtValue] = useState("");
  const [txt, setTxt] = useState("");
  const load = useCallback(async () => {
    const [certs, dns] = await Promise.all([
      api.certificates.list(),
      api.certificates.dns(),
    ]);
    setList(certs || []);
    setDnsSet(dns.token_set);
    setProvider(dns.provider || "cloudflare");
    setApiBase(dns.api_base || "");
    setZone(dns.zone || "");
    setSelected((id) =>
      certs?.some((cert) => cert.id === id) ? id : certs?.[0]?.id || "",
    );
  }, []);
  const loading = useInitialLoad(load, feedback.setError);
  useEffect(() => {
    const certificate = list.find((item) => item.id === selected);
    setSites((certificate?.sites || []).join(", "));
  }, [list, selected]);
  const expired = list.filter(
    (cert) => Date.parse(cert.expiry) <= Date.now(),
  ).length;

  return (
    <OpsPage title="证书" feedback={feedback}>
      <OpsStats
        items={[
          {
            label: "证书总数",
            value: loading ? "—" : list.length,
            icon: "lock",
            hint: "集中管理网站 TLS 证书",
          },
          {
            label: "已绑定网站",
            value: loading
              ? "—"
              : new Set(list.flatMap((cert) => cert.sites || [])).size,
            icon: "globe",
            hint: expired
              ? `${expired} 张证书已过期，请及时更新`
              : "同一证书可供多个网站使用",
          },
          {
            label: "DNS 验证",
            value: loading ? "—" : dnsSet ? "已配置" : "待配置",
            icon: "shield",
            hint: "配置后可使用 DNS-01 签发",
          },
        ]}
      />
      <div className="ops-layout">
        <div className="ops-stack">
          <OpsCard
            title="证书列表"
            description="查看域名、有效期与网站绑定情况"
            icon="lock"
            extra={
              <Button
                size="sm"
                disabled={loading}
                onClick={() => void feedback.run(load)}
              >
                <Icon name="refresh" size={14} />
                刷新
              </Button>
            }
          >
            {loading ? (
              <OpsLoading />
            ) : list.length ? (
              <div className="ops-list">
                {list.map((cert) => (
                  <div className="ops-list-item" key={cert.id}>
                    <div className="ops-item-top">
                      <strong>
                        {(cert.domains || []).join(", ") || cert.id}
                      </strong>
                      <OpsTag
                        tone={
                          cert.attempts?.at(-1)?.status === "failed" ||
                          Date.parse(cert.expiry) <= Date.now()
                            ? "warning"
                            : "success"
                        }
                      >
                        {cert.attempts?.at(-1)?.status === "failed"
                          ? "签发失败"
                          : Date.parse(cert.expiry) <= Date.now()
                            ? "已过期"
                            : cert.attempts?.at(-1)?.status === "submitted"
                              ? "已提交验证"
                              : "已签发"}
                      </OpsTag>
                    </div>
                    <div className="ops-grid">
                      <div className="ops-note">
                        签发来源：{cert.provider || "—"}
                        <br />
                        有效期至：{opsDate(cert.expiry)}
                      </div>
                      <div className="ops-note">
                        绑定网站：{(cert.sites || []).join(", ") || "尚未绑定"}
                        <br />
                        证书 ID：{cert.id}
                      </div>
                    </div>
                    {!!cert.attempts?.length && (
                      <details className="ops-details">
                        <summary>
                          <Icon name="down" size={14} />
                          签发记录（{cert.attempts.length}）
                        </summary>
                        {cert.attempts.map((attempt, index) => (
                          <p className="ops-note" key={index}>
                            {attempt.method} · {attempt.status}{" "}
                            {attempt.error || ""}
                          </p>
                        ))}
                      </details>
                    )}
                    <div className="ops-actions">
                      <Button
                        size="sm"
                        onClick={() => {
                          setSelected(cert.id);
                          setSites((cert.sites || []).join(", "));
                          document
                            .getElementById("certificate-binding")
                            ?.scrollIntoView({
                              block: "center",
                              behavior: "smooth",
                            });
                        }}
                      >
                        管理绑定
                        <Icon name="arrow" size={13} />
                      </Button>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <OpsEmpty
                icon="lock"
                title={feedback.error ? "证书暂时无法加载" : "还没有证书"}
                description={
                  feedback.error
                    ? "请刷新重试，或检查服务连接。"
                    : "从右侧签发第一张证书，再将它绑定到网站。"
                }
              />
            )}
          </OpsCard>
          <OpsCard
            title="绑定到网站"
            description="选择证书后，填写需要绑定的网站 ID"
            icon="link"
          >
            <form
              id="certificate-binding"
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  await api.certificates.bind(selected, splitValues(sites));
                  await load();
                }, "证书绑定已更新");
              }}
            >
              <div className="ops-grid">
                <OpsField label="选择证书">
                  <Select
                    required
                    value={selected}
                    placeholder="请选择证书"
                    onValueChange={(value) => {
                      setSelected(value);
                      setSites(
                        (
                          list.find((cert) => cert.id === value)?.sites || []
                        ).join(", "),
                      );
                    }}
                    options={list.map((cert) => ({
                      value: cert.id,
                      label: cert.domains?.join(", ") || cert.id,
                    }))}
                  />
                </OpsField>
                <OpsField label="网站 ID" hint="多个 ID 用英文逗号分隔">
                  <Input
                    required
                    placeholder="site-id, another-site-id"
                    value={sites}
                    onChange={(event) => setSites(event.target.value)}
                  />
                </OpsField>
              </div>
              <div className="ops-actions ops-actions-end">
                <Button type="submit" disabled={!selected || loading}>
                  <Icon name="link" size={14} />
                  保存绑定
                </Button>
              </div>
            </form>
          </OpsCard>
          <OpsCard
            title="手动 DNS 验证"
            description="记录用于域名验证的 TXT 信息"
            icon="fileText"
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  const result = await api.certificates.manualTxt(
                    txtName,
                    txtValue,
                  );
                  setTxt(`${result.name} ${result.type} ${result.value}`);
                }, "TXT 记录已保存");
              }}
            >
              <div className="ops-grid">
                <OpsField label="TXT 名称">
                  <Input
                    required
                    placeholder="_acme-challenge.example.com"
                    value={txtName}
                    onChange={(event) => setTxtName(event.target.value)}
                  />
                </OpsField>
                <OpsField label="TXT 值">
                  <Input
                    required
                    placeholder="验证记录的值"
                    value={txtValue}
                    onChange={(event) => setTxtValue(event.target.value)}
                  />
                </OpsField>
              </div>
              {txt && <pre className="ops-output">{txt}</pre>}
              <div className="ops-actions ops-actions-end">
                <Button type="submit">记录手动 TXT</Button>
              </div>
            </form>
          </OpsCard>
        </div>
        <div className="ops-stack">
          <OpsCard
            title="签发证书"
            description="为一个或多个域名创建证书"
            icon="plus"
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  const cert = await api.certificates.selfSigned(
                    splitValues(domain),
                  );
                  await load();
                  setSelected(cert.id);
                }, "自签证书已签发");
              }}
            >
              <OpsField label="域名" hint="多个域名用英文逗号分隔">
                <Input
                  required
                  placeholder="example.com, www.example.com"
                  value={domain}
                  onChange={(event) => setDomain(event.target.value)}
                />
              </OpsField>
              <div className="ops-actions">
                <Button
                  variant="primary"
                  type="submit"
                  disabled={!splitValues(domain).length}
                >
                  <Icon name="plus" size={15} />
                  签发自签证书
                </Button>
                <Button
                  disabled={!dnsSet || !splitValues(domain).length}
                  onClick={() =>
                    void feedback.run(async () => {
                      const result = await api.certificates.dnsIssue(
                        splitValues(domain),
                      );
                      await load();
                      if (
                        result.status === "failed" ||
                        result.status === "error"
                      )
                        throw new Error(result.message || "签发失败");
                      feedback.setMessage(result.message || result.status);
                    })
                  }
                >
                  DNS-01 签发
                </Button>
              </div>
              <div className="ops-callout">
                <Icon name="info" size={15} />
                <span>
                  自签证书适合本地和测试环境。公网网站建议配置 DNS
                  验证后签发受信任证书。
                </span>
              </div>
            </form>
          </OpsCard>
          <OpsCard
            title="DNS 服务商"
            description="设置自动验证所需的连接信息"
            icon="globe"
            extra={
              <OpsTag tone={dnsSet ? "success" : "warning"}>
                {dnsSet ? "已配置" : "待配置"}
              </OpsTag>
            }
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  await api.certificates.saveDns({
                    provider,
                    api_base: apiBase,
                    token,
                    zone,
                  });
                  setToken("");
                  await load();
                }, "DNS 配置已保存");
              }}
            >
              <OpsField label="服务商">
                <Input
                  required
                  value={provider}
                  onChange={(event) => setProvider(event.target.value)}
                  placeholder="cloudflare"
                />
              </OpsField>
              <OpsField
                label="API 地址"
                hint="DNS-01 签发需要配置服务商 API 地址"
              >
                <Input
                  placeholder="https://api.example.com"
                  value={apiBase}
                  onChange={(event) => setApiBase(event.target.value)}
                />
              </OpsField>
              <OpsField label="Zone / 区域 ID">
                <Input
                  placeholder="填写域名所属区域"
                  value={zone}
                  onChange={(event) => setZone(event.target.value)}
                />
              </OpsField>
              <OpsField
                label="API 令牌"
                hint={
                  dnsSet
                    ? "密钥已配置；保存配置时请重新填写令牌"
                    : "保存后不会回显密钥"
                }
              >
                <Input
                  type="password"
                  required
                  autoComplete="off"
                  placeholder={dnsSet ? "••••••••（已保存）" : "输入 API 令牌"}
                  value={token}
                  onChange={(event) => setToken(event.target.value)}
                />
              </OpsField>
              <div className="ops-actions ops-actions-end">
                <Button type="submit">保存 DNS 配置</Button>
              </div>
            </form>
          </OpsCard>
        </div>
      </div>
    </OpsPage>
  );
}

export function DatabasesPage({
  onOpenStore,
}: {
  onOpenStore?: (app: string) => void;
}) {
  const feedback = useOpsFeedback();
  const [engines, setEngines] = useState<DatabaseInfo[]>([]);
  const [kind, setKind] = useState("mysql");
  const [name, setName] = useState("");
  const [user, setUser] = useState("");
  const [password, setPassword] = useState("");
  const [sql, setSql] = useState("SELECT 1");
  const [dump, setDump] = useState("");
  const [result, setResult] = useState("");
  const [prefix, setPrefix] = useState("user:");
  const [keys, setKeys] = useState<string[]>([]);
  const [scanned, setScanned] = useState(false);
  const load = useCallback(async () => {
    setEngines((await api.databases.list()) || []);
  }, []);
  const loading = useInitialLoad(load, feedback.setError);
  const current = engines.find((item) => item.kind === kind);
  const unavailable = loading || !current?.installed;
  function selectEngine(next: string) {
    setKind(next);
    setResult("");
    setDump("");
    setKeys([]);
    setScanned(false);
    feedback.setMessage("");
    feedback.setError("");
  }

  return (
    <OpsPage title="数据库" feedback={feedback}>
      <div className="ops-engine-grid">
        {Object.entries(DATABASE_NAMES).map(([id, label]) => {
          const engine = engines.find((item) => item.kind === id);
          return (
            <div
              key={id}
              className={`ops-engine ${kind === id ? "ops-engine-active" : ""}`}
            >
              <button
                type="button"
                className="ops-engine-choice"
                aria-pressed={kind === id}
                onClick={() => selectEngine(id)}
              >
                <Icon name="database" size={21} />
                {label}
              </button>
              <div className="ops-item-top">
                <OpsTag tone={engine?.installed ? "success" : "neutral"}>
                  {loading ? "加载中" : engine?.installed ? "已连接" : "未连接"}
                </OpsTag>
                {!loading && !engine?.installed && onOpenStore && (
                  <Button size="sm" onClick={() => onOpenStore(id)}>
                    前往安装
                    <Icon name="arrow" size={12} />
                  </Button>
                )}
              </div>
            </div>
          );
        })}
      </div>
      <div className="ops-layout">
        <div className="ops-stack">
          {unavailable && (
            <div className="ops-callout">
              <Icon name="info" size={16} />
              <span>
                {loading
                  ? "正在读取数据库状态…"
                  : feedback.error && !current
                    ? "数据库状态暂时无法读取。"
                    : "数据库控制台暂不可用，尚未接入真实数据库。"}
              </span>
            </div>
          )}
          {kind !== "redis" ? (
            <>
              <OpsCard
                title="SQL 控制台"
                description={`在 ${DATABASE_NAMES[kind]} 中执行 SQL 语句`}
                icon="terminal"
                extra={<OpsTag>{DATABASE_NAMES[kind]}</OpsTag>}
              >
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    void feedback.run(async () => {
                      setResult(
                        (await api.databases.exec(kind, sql, password)).result,
                      );
                    }, "SQL 已执行");
                  }}
                >
                  <OpsField label="SQL 语句">
                    <Textarea
                      className="ops-code"
                      rows={8}
                      required
                      spellCheck={false}
                      value={sql}
                      onChange={(event) => setSql(event.target.value)}
                    />
                  </OpsField>
                  <div className="ops-actions ops-actions-end">
                    <span className="ops-note">执行记录会保存到审计日志</span>
                    <Button
                      variant="primary"
                      type="submit"
                      disabled={unavailable}
                    >
                      <Icon name="play" size={14} />
                      执行 SQL
                    </Button>
                  </div>
                </form>
                {result && (
                  <>
                    <h3 className="ops-subheading">执行结果</h3>
                    <pre className="ops-output">{result}</pre>
                  </>
                )}
              </OpsCard>
              <OpsCard
                title="导入与导出"
                description="导出 SQL，或粘贴 SQL 内容进行导入"
                icon="download"
              >
                <OpsField label="SQL 数据">
                  <Textarea
                    className="ops-code"
                    rows={5}
                    placeholder="导出的 SQL 将显示在这里，也可粘贴要导入的内容"
                    value={dump}
                    onChange={(event) => setDump(event.target.value)}
                  />
                </OpsField>
                <div className="ops-actions ops-actions-end">
                  <Button
                    disabled={unavailable}
                    onClick={() =>
                      void feedback.run(async () => {
                        setDump((await api.databases.exportSql(kind)).sql);
                      }, "SQL 已导出")
                    }
                  >
                    导出 SQL
                  </Button>
                  <Button
                    disabled={unavailable || !dump.trim()}
                    onClick={() =>
                      void feedback.run(
                        () => api.databases.importSql(kind, dump),
                        "SQL 已导入",
                      )
                    }
                  >
                    导入 SQL
                  </Button>
                </div>
              </OpsCard>
            </>
          ) : (
            <OpsCard
              title="Redis 键浏览器"
              description="按前缀扫描，每次最多显示 50 个键"
              icon="search"
            >
              <form
                onSubmit={(event) => {
                  event.preventDefault();
                  void feedback.run(async () => {
                    setKeys(
                      (await api.databases.redisKeys(prefix, 50)).keys || [],
                    );
                    setScanned(true);
                  });
                }}
              >
                <div className="ops-toolbar">
                  <OpsField label="键前缀">
                    <Input
                      required
                      value={prefix}
                      onChange={(event) => setPrefix(event.target.value)}
                      placeholder="user:"
                    />
                  </OpsField>
                  <Button
                    variant="primary"
                    type="submit"
                    disabled={unavailable}
                  >
                    <Icon name="search" size={14} />
                    扫描键
                  </Button>
                </div>
              </form>
              {keys.length ? (
                <div className="ops-list">
                  {keys.map((key) => (
                    <div className="ops-list-item" key={key}>
                      <div className="ops-item-top">
                        <strong>{key}</strong>
                        <Button
                          size="sm"
                          variant="danger"
                          disabled={unavailable}
                          onClick={() =>
                            void feedback.run(async () => {
                              await api.databases.redisDel(key);
                              setKeys(
                                (await api.databases.redisKeys(prefix, 50))
                                  .keys || [],
                              );
                            }, "键已删除")
                          }
                        >
                          删除
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <OpsEmpty
                  icon="database"
                  title={scanned ? "没有匹配的键" : "浏览 Redis 数据"}
                  description={
                    scanned
                      ? "调整前缀后重新扫描。"
                      : "输入键前缀，查看匹配的数据。"
                  }
                />
              )}
            </OpsCard>
          )}
        </div>
        <div className="ops-stack">
          {kind !== "redis" && (
            <>
              <OpsCard
                title="创建数据库"
                description={`为 ${DATABASE_NAMES[kind]} 新建数据库`}
                icon="plus"
              >
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    void feedback.run(
                      () => api.databases.createDatabase(kind, name),
                      `数据库 ${name} 已创建`,
                    );
                  }}
                >
                  <OpsField label="数据库名">
                    <Input
                      required
                      placeholder="my_database"
                      value={name}
                      onChange={(event) => setName(event.target.value)}
                    />
                  </OpsField>
                  <div className="ops-actions ops-actions-end">
                    <Button type="submit" disabled={unavailable}>
                      创建数据库
                    </Button>
                  </div>
                </form>
              </OpsCard>
              <OpsCard
                title="数据库用户"
                description="创建访问数据库的账号"
                icon="user"
              >
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    void feedback.run(
                      () => api.databases.createUser(kind, user, password),
                      `用户 ${user} 已创建`,
                    );
                  }}
                >
                  <OpsField label="用户名">
                    <Input
                      required
                      placeholder="db_user"
                      value={user}
                      onChange={(event) => setUser(event.target.value)}
                    />
                  </OpsField>
                  <OpsField label="密码" hint="此密码也用于 SQL 控制台连接">
                    <Input
                      type="password"
                      required
                      autoComplete="new-password"
                      placeholder="输入用户密码"
                      value={password}
                      onChange={(event) => setPassword(event.target.value)}
                    />
                  </OpsField>
                  <div className="ops-actions ops-actions-end">
                    <Button type="submit" disabled={unavailable}>
                      创建用户
                    </Button>
                  </div>
                </form>
              </OpsCard>
            </>
          )}
          <div className="ops-callout">
            <Icon name="shield" size={15} />
            <span>数据库操作会记录审计日志，连接密码不会被记录。</span>
          </div>
          <div className="ops-actions">
            <Button
              size="sm"
              disabled={loading}
              onClick={() => void feedback.run(load)}
            >
              <Icon name="refresh" size={14} />
              刷新数据库状态
            </Button>
          </div>
        </div>
      </div>
    </OpsPage>
  );
}

export function BackupsPage() {
  const feedback = useOpsFeedback();
  const [items, setItems] = useState<BackupItem[]>([]);
  const [siteId, setSiteId] = useState("");
  const [confirm, setConfirm] = useState(false);
  const [endpoint, setEndpoint] = useState("");
  const [bucket, setBucket] = useState("");
  const [access, setAccess] = useState("");
  const [secret, setSecret] = useState("");
  const load = useCallback(async () => {
    setItems((await api.backups.list()) || []);
  }, []);
  const loading = useInitialLoad(load, feedback.setError);
  return (
    <OpsPage title="备份" feedback={feedback}>
      <OpsStats
        items={[
          {
            label: "备份总数",
            value: loading ? "—" : items.length,
            icon: "download",
            hint: "网站与数据的恢复副本",
          },
          {
            label: "对象存储",
            value: loading ? "—" : items.filter((item) => item.remote).length,
            icon: "globe",
            hint: "已登记到远程存储的备份",
          },
          {
            label: "本地副本",
            value: loading ? "—" : items.length,
            icon: "folder",
            hint: "保存在面板数据目录",
          },
        ]}
      />
      <div className="ops-layout">
        <OpsCard
          title="备份记录"
          description="查看备份来源，并按需恢复或登记远程存储"
          icon="download"
          extra={
            <Button
              size="sm"
              disabled={loading}
              onClick={() => void feedback.run(load)}
            >
              <Icon name="refresh" size={14} />
              刷新
            </Button>
          }
        >
          {loading ? (
            <OpsLoading />
          ) : items.length ? (
            <div className="ops-list">
              {items.map((item) => (
                <div className="ops-list-item" key={item.id}>
                  <div className="ops-item-top">
                    <strong>{item.source || item.id}</strong>
                    <OpsTag tone={item.remote ? "success" : "neutral"}>
                      {item.remote ? "已登记远程" : "本地备份"}
                    </OpsTag>
                  </div>
                  <p className="ops-note">
                    {JOB_NAMES[`backup_${item.kind}`] || item.kind} ·{" "}
                    {opsDate(item.created_at)}
                    {item.remote && (
                      <>
                        <br />
                        {item.remote}
                      </>
                    )}
                  </p>
                  <div className="ops-actions">
                    <Button
                      size="sm"
                      disabled={!confirm}
                      onClick={() =>
                        void feedback.run(
                          () => api.backups.restore(item.id, confirm),
                          "备份已恢复",
                        )
                      }
                    >
                      恢复备份
                    </Button>
                    <Button
                      size="sm"
                      disabled={!endpoint || !bucket || !access || !secret}
                      onClick={() =>
                        void feedback.run(async () => {
                          const result = await api.backups.publish(item.id, {
                            endpoint,
                            bucket,
                            access_key: access,
                            secret,
                          });
                          await load();
                          feedback.setMessage(`已登记：${result.remote}`);
                        })
                      }
                    >
                      登记到对象存储
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <OpsEmpty
              icon="download"
              title={feedback.error ? "备份暂时无法加载" : "还没有备份"}
              description={
                feedback.error
                  ? "请刷新重试，或检查服务连接。"
                  : "填写网站 ID 创建第一份备份，为数据保留恢复副本。"
              }
            />
          )}
          <label className="ops-check">
            <Checkbox
              checked={confirm}
              onChange={(event) => setConfirm(event.target.checked)}
            />
            <span>我确认恢复备份会覆盖目标内容</span>
          </label>
          <div className="ops-callout">
            <Icon name="info" size={15} />
            <span>
              恢复前勾选确认。超出保留数量后，系统会清理最旧的本地副本。
            </span>
          </div>
        </OpsCard>
        <div className="ops-stack">
          <OpsCard
            title="创建备份"
            description="为指定网站保存恢复副本"
            icon="plus"
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  await api.backups.site(siteId);
                  await load();
                }, "网站已备份");
              }}
            >
              <OpsField label="网站 ID">
                <Input
                  required
                  placeholder="输入需要备份的网站 ID"
                  value={siteId}
                  onChange={(event) => setSiteId(event.target.value)}
                />
              </OpsField>
              <div className="ops-actions ops-actions-end">
                <Button variant="primary" type="submit">
                  <Icon name="download" size={14} />
                  备份网站
                </Button>
              </div>
            </form>
          </OpsCard>
          <OpsCard
            title="对象存储"
            description="填写信息后，在备份记录中登记"
            icon="globe"
          >
            <OpsField label="S3 兼容地址">
              <Input
                placeholder="https://s3.example.com"
                value={endpoint}
                onChange={(event) => setEndpoint(event.target.value)}
              />
            </OpsField>
            <OpsField label="存储桶">
              <Input
                placeholder="my-backups"
                value={bucket}
                onChange={(event) => setBucket(event.target.value)}
              />
            </OpsField>
            <OpsField label="Access Key">
              <Input
                autoComplete="off"
                placeholder="输入访问密钥"
                value={access}
                onChange={(event) => setAccess(event.target.value)}
              />
            </OpsField>
            <OpsField label="Secret Key">
              <Input
                type="password"
                autoComplete="off"
                placeholder="输入私有密钥"
                value={secret}
                onChange={(event) => setSecret(event.target.value)}
              />
            </OpsField>
          </OpsCard>
        </div>
      </div>
    </OpsPage>
  );
}

export function SchedulesPage() {
  const feedback = useOpsFeedback();
  const [jobs, setJobs] = useState<ScheduleJob[]>([]);
  const [name, setName] = useState("站点备份");
  const [type, setType] = useState("backup_site");
  const [cron, setCron] = useState("0 3 * * *");
  const [payload, setPayload] = useState("");
  const [script, setScript] = useState("echo ok\n");
  const load = useCallback(async () => {
    setJobs((await api.schedules.list()) || []);
  }, []);
  const loading = useInitialLoad(load, feedback.setError);
  const payloadLabel =
    type === "backup_site"
      ? "网站 ID"
      : type === "backup_app"
        ? "应用路径"
        : type === "fetch_url"
          ? "目标 URL"
          : type === "script"
            ? "脚本文件名"
            : type === "backup_database"
              ? "数据库类型"
              : "目标路径";
  return (
    <OpsPage title="计划任务" feedback={feedback}>
      <OpsStats
        items={[
          {
            label: "任务总数",
            value: loading ? "—" : jobs.length,
            icon: "clock",
            hint: "任务配置在重启后保留",
          },
          {
            label: "已启用",
            value: loading ? "—" : jobs.filter((job) => job.enabled).length,
            icon: "play",
            hint: "按 Cron 表达式自动执行",
          },
          {
            label: "已停用",
            value: loading ? "—" : jobs.filter((job) => !job.enabled).length,
            icon: "stop",
            hint: "可随时重新启用",
          },
        ]}
      />
      <div className="ops-layout">
        <OpsCard
          title="任务列表"
          description="管理任务状态，查看最近的运行记录"
          icon="clock"
          extra={
            <Button
              size="sm"
              disabled={loading}
              onClick={() =>
                void feedback.run(async () => {
                  await api.schedules.reload();
                  await load();
                }, "计划任务已重新加载")
              }
            >
              <Icon name="refresh" size={14} />
              重新加载
            </Button>
          }
        >
          {loading ? (
            <OpsLoading />
          ) : jobs.length ? (
            <div className="ops-list">
              {jobs.map((job) => (
                <div className="ops-list-item" key={job.id}>
                  <div className="ops-item-top">
                    <strong>{job.name}</strong>
                    <OpsTag tone={job.enabled ? "success" : "neutral"}>
                      {job.enabled ? "已启用" : "已停用"}
                    </OpsTag>
                  </div>
                  <p className="ops-note">
                    {JOB_NAMES[job.type] || job.type} · <code>{job.cron}</code>
                    {job.missed && " · 已错过执行"}
                  </p>
                  {job.runs?.length ? (
                    <details className="ops-details">
                      <summary>
                        <Icon name="down" size={14} />
                        最近运行记录（{job.runs.length}）
                      </summary>
                      {job.runs
                        .slice(-3)
                        .reverse()
                        .map((run, index) => (
                          <div className="ops-note" key={index}>
                            {opsDate(run.at)} · {run.status}
                            {run.output && (
                              <pre className="ops-output">{run.output}</pre>
                            )}
                          </div>
                        ))}
                    </details>
                  ) : (
                    <p className="ops-note">还没有运行记录</p>
                  )}
                  <div className="ops-actions">
                    <Button
                      size="sm"
                      onClick={() =>
                        void feedback.run(async () => {
                          const run = await api.schedules.run(job.id);
                          await load();
                          feedback.setMessage(`${job.name}：${run.status}`);
                        })
                      }
                    >
                      <Icon name="play" size={13} />
                      立即运行
                    </Button>
                    <Button
                      size="sm"
                      onClick={() =>
                        void feedback.run(
                          async () => {
                            await api.schedules.enabled(job.id, !job.enabled);
                            await load();
                          },
                          job.enabled ? "任务已停用" : "任务已启用",
                        )
                      }
                    >
                      {job.enabled ? "停用任务" : "启用任务"}
                    </Button>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <OpsEmpty
              icon="clock"
              title={feedback.error ? "任务暂时无法加载" : "还没有计划任务"}
              description={
                feedback.error
                  ? "请重新加载任务，或检查服务连接。"
                  : "创建备份、证书续签或脚本任务，让日常维护自动执行。"
              }
            />
          )}
        </OpsCard>
        <div className="ops-stack">
          <OpsCard
            title="新建任务"
            description="设置任务内容与执行周期"
            icon="plus"
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                const key =
                  type === "backup_site"
                    ? "site_id"
                    : type === "fetch_url"
                      ? "url"
                      : type === "script"
                        ? "script"
                        : type === "backup_database"
                          ? "kind"
                          : "path";
                void feedback.run(async () => {
                  await api.schedules.create({
                    name,
                    type,
                    cron,
                    payload: payload ? { [key]: payload } : {},
                    timeout_sec: 30,
                  });
                  await load();
                }, "计划任务已创建");
              }}
            >
              <OpsField label="任务名称">
                <Input
                  required
                  value={name}
                  onChange={(event) => setName(event.target.value)}
                />
              </OpsField>
              <OpsField label="任务类型">
                <Select
                  value={type}
                  onValueChange={setType}
                  options={Object.entries(JOB_NAMES).map(([value, label]) => ({
                    value,
                    label,
                  }))}
                />
              </OpsField>
              <OpsField
                label="Cron 表达式"
                hint="分 时 日 月 周；0 3 * * * 表示每天 03:00"
              >
                <Input
                  className="ops-code"
                  required
                  value={cron}
                  onChange={(event) => setCron(event.target.value)}
                  placeholder="0 3 * * *"
                />
              </OpsField>
              {!["renew_certs", "rotate_logs"].includes(type) && (
                <OpsField label={payloadLabel}>
                  <Input
                    required
                    value={payload}
                    onChange={(event) => setPayload(event.target.value)}
                    placeholder={
                      type === "script" ? "job.sh" : `输入${payloadLabel}`
                    }
                  />
                </OpsField>
              )}
              {type === "script" && (
                <>
                  <OpsField label="脚本内容">
                    <Textarea
                      className="ops-code"
                      rows={5}
                      spellCheck={false}
                      value={script}
                      onChange={(event) => setScript(event.target.value)}
                    />
                  </OpsField>
                  <Button
                    disabled={!payload.trim()}
                    onClick={() =>
                      void feedback.run(
                        () => api.schedules.script(payload, script),
                        "脚本已保存",
                      )
                    }
                  >
                    保存脚本
                  </Button>
                  <p className="ops-note">创建任务前，请先保存脚本。</p>
                </>
              )}
              <div className="ops-actions ops-actions-end">
                <Button variant="primary" type="submit">
                  <Icon name="plus" size={14} />
                  创建任务
                </Button>
              </div>
            </form>
          </OpsCard>
          <div className="ops-callout">
            <Icon name="info" size={15} />
            <span>
              任务默认超时为 30 秒。证书自动续签可通过「证书续签」任务设置。
            </span>
          </div>
        </div>
      </div>
    </OpsPage>
  );
}

export function FirewallPage() {
  const feedback = useOpsFeedback();
  const [status, setStatus] = useState<FirewallStatus | null>(null);
  const [rules, setRules] = useState<AlertRule[]>([]);
  const [port, setPort] = useState("22");
  const [policy, setPolicy] = useState("allow");
  const [confirm, setConfirm] = useState(false);
  const [webhook, setWebhook] = useState("");
  const [threshold, setThreshold] = useState("90");
  const [container, setContainer] = useState("");
  const load = useCallback(async () => {
    const [fw, alerts] = await Promise.all([
      api.firewall.status(),
      api.alerts.list(),
    ]);
    setStatus(fw);
    setRules(alerts || []);
  }, []);
  const loading = useInitialLoad(load, feedback.setError);
  const needsConfirm = policy === "deny" && Number(port) === status?.panel_port;
  return (
    <OpsPage title="防火墙" feedback={feedback}>
      <OpsStats
        items={[
          {
            label: "防火墙状态",
            value: loading ? "—" : status?.available ? "可用" : "不可用",
            icon: "shield",
            hint: status?.backend
              ? `当前后端：${status.backend}`
              : "读取主机防火墙能力",
          },
          {
            label: "端口规则",
            value: loading ? "—" : (status?.rules?.length ?? "—"),
            icon: "network",
            hint: status?.panel_port
              ? `面板端口 ${status.panel_port} 默认放行`
              : "管理端口的访问策略",
          },
          {
            label: "告警规则",
            value: loading ? "—" : rules.length,
            icon: "alertCircle",
            hint: "监测主机磁盘使用情况",
          },
        ]}
      />
      {status?.message && (
        <div className="ops-callout">
          <Icon name="info" size={16} />
          <span>{status.message}</span>
        </div>
      )}
      <div className="ops-layout">
        <div className="ops-stack">
          <OpsCard
            title="端口规则"
            description="查看已配置的端口访问策略"
            icon="shield"
            extra={
              <Button
                size="sm"
                disabled={loading}
                onClick={() => void feedback.run(load)}
              >
                <Icon name="refresh" size={14} />
                刷新
              </Button>
            }
          >
            {loading ? (
              <OpsLoading />
            ) : status?.rules?.length ? (
              <div className="ops-table-wrap">
                <table className="ops-table">
                  <thead>
                    <tr>
                      <th>端口</th>
                      <th>访问策略</th>
                      <th>用途</th>
                    </tr>
                  </thead>
                  <tbody>
                    {status.rules.map((rule, index) => (
                      <tr key={`${rule.port}-${index}`}>
                        <td>
                          <strong>{rule.port}</strong>
                        </td>
                        <td>
                          <OpsTag
                            tone={
                              rule.policy === "allow" ? "success" : "warning"
                            }
                          >
                            {rule.policy === "allow" ? "允许访问" : "拒绝访问"}
                          </OpsTag>
                        </td>
                        <td className="ops-note">
                          {rule.port === status.panel_port
                            ? "面板端口"
                            : rule.port === 22
                              ? "SSH"
                              : rule.port === 80
                                ? "HTTP"
                                : rule.port === 443
                                  ? "HTTPS"
                                  : "自定义端口"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : (
              <OpsEmpty
                icon="shield"
                title={
                  !status || !status.available
                    ? "防火墙暂不可用"
                    : "还没有端口规则"
                }
                description={
                  !status || !status.available
                    ? "请查看防火墙状态，或刷新重试。"
                    : "添加规则，管理服务端口的访问权限。"
                }
              />
            )}
          </OpsCard>
          <OpsCard
            title="磁盘告警"
            description="磁盘使用率达到设定阈值时触发通知"
            icon="alertCircle"
            extra={<OpsTag>{rules.length} 条规则</OpsTag>}
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  await api.alerts.add({
                    kind: "disk",
                    threshold: Number(threshold),
                    webhook,
                  });
                  await load();
                }, "磁盘告警已添加");
              }}
            >
              <OpsField label="磁盘使用率阈值（%）">
                <Input
                  type="number"
                  required
                  min={1}
                  max={100}
                  value={threshold}
                  onChange={(event) => setThreshold(event.target.value)}
                />
              </OpsField>
              <OpsField label="Webhook 地址">
                <Input
                  type="url"
                  required
                  placeholder="https://example.com/webhook"
                  value={webhook}
                  onChange={(event) => setWebhook(event.target.value)}
                />
              </OpsField>
              <div className="ops-actions ops-actions-end">
                <Button
                  disabled={!rules.length}
                  onClick={() =>
                    void feedback.run(async () => {
                      const result = await api.alerts.evaluate({
                        disk_percent: Number(threshold),
                      });
                      feedback.setMessage(`已触发 ${result.sent} 次通知`);
                    })
                  }
                >
                  测试告警通知
                </Button>
                <Button type="submit">添加告警规则</Button>
              </div>
            </form>
            {!!rules.length && (
              <div className="ops-list">
                {rules.map((rule) => (
                  <div key={rule.id} className="ops-list-item">
                    <div className="ops-item-top">
                      <strong>
                        {rule.kind === "disk" ? "磁盘告警" : rule.kind}
                      </strong>
                      <OpsTag>
                        阈值 {rule.threshold}
                        {rule.kind === "disk" ? "%" : ""}
                      </OpsTag>
                    </div>
                    <p className="ops-note">
                      {rule.webhook || rule.smtp_host || "未设置通知地址"}
                    </p>
                  </div>
                ))}
              </div>
            )}
          </OpsCard>
        </div>
        <div className="ops-stack">
          <OpsCard
            title="添加端口规则"
            description="指定端口与访问策略"
            icon="plus"
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  await api.firewall.add(Number(port), policy, confirm);
                  setConfirm(false);
                  await load();
                }, "端口规则已添加");
              }}
            >
              <OpsField label="端口">
                <Input
                  required
                  type="number"
                  min={1}
                  max={65535}
                  value={port}
                  onChange={(event) => {
                    setPort(event.target.value);
                    setConfirm(false);
                  }}
                />
              </OpsField>
              <OpsField label="访问策略">
                <Select
                  value={policy}
                  onValueChange={(value) => {
                    setPolicy(value);
                    setConfirm(false);
                  }}
                  options={[
                    { value: "allow", label: "允许访问" },
                    { value: "deny", label: "拒绝访问" },
                  ]}
                />
              </OpsField>
              {needsConfirm && (
                <label className="ops-check">
                  <Checkbox
                    checked={confirm}
                    onChange={(event) => setConfirm(event.target.checked)}
                  />
                  <span>我确认此规则可能关闭面板访问端口</span>
                </label>
              )}
              <div className="ops-actions ops-actions-end">
                <Button
                  variant="primary"
                  type="submit"
                  disabled={!status?.available || (needsConfirm && !confirm)}
                >
                  <Icon name="plus" size={14} />
                  添加规则
                </Button>
              </div>
            </form>
          </OpsCard>
          <OpsCard
            title="终端会话"
            description="创建主机或容器的终端会话记录"
            icon="terminal"
          >
            <Button
              onClick={() =>
                void feedback.run(async () => {
                  const session = await api.terminals.host();
                  feedback.setMessage(
                    `主机终端 ${session.id} · ${session.account}`,
                  );
                })
              }
            >
              <Icon name="terminal" size={14} />
              创建主机会话
            </Button>
            <hr className="ops-separator" />
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  const session = await api.terminals.container(container);
                  feedback.setMessage(`容器终端 ${session.id}`);
                });
              }}
            >
              <OpsField label="容器 ID">
                <Input
                  required
                  placeholder="输入容器 ID"
                  value={container}
                  onChange={(event) => setContainer(event.target.value)}
                />
              </OpsField>
              <Button type="submit">创建容器会话</Button>
            </form>
            <p className="ops-note">
              审计只记录账号与会话起止时间，不记录按键内容。
            </p>
          </OpsCard>
        </div>
      </div>
    </OpsPage>
  );
}

export function AssistantPage() {
  const feedback = useOpsFeedback();
  const [base, setBase] = useState("https://api.openai.com/v1");
  const [model, setModel] = useState("");
  const [key, setKey] = useState("");
  const [keySet, setKeySet] = useState(false);
  const [headerName, setHeaderName] = useState("");
  const [headerValue, setHeaderValue] = useState("");
  const [message, setMessage] = useState("");
  const [reply, setReply] = useState<AiReply | null>(null);
  const [asked, setAsked] = useState("");
  const load = useCallback(async () => {
    const settings = await api.ai.settings();
    setBase(settings.base_url || "https://api.openai.com/v1");
    setModel(settings.model || "");
    setKeySet(settings.key_set);
  }, []);
  const loading = useInitialLoad(load, feedback.setError);
  const suggestions = [
    "当前容器和证书怎么样？",
    "帮我检查网站运行状态",
    "有哪些需要关注的运维问题？",
  ];
  function ask(confirm: boolean) {
    const question = confirm ? asked : message;
    void feedback.run(async () => {
      const response = await api.ai.chat(question, confirm);
      setAsked(question);
      setReply(response);
    });
  }
  return (
    <OpsPage title="AI 运维助手" feedback={feedback}>
      <div className="ops-layout">
        <div className="ops-stack">
          <OpsCard
            title="运维对话"
            description="结合当前资源状态，获取建议与操作方案"
            icon="sparkle"
            extra={
              <OpsTag tone={keySet && model ? "success" : "neutral"}>
                {loading
                  ? "加载中"
                  : keySet && model
                    ? "供应商已配置"
                    : "快照模式"}
              </OpsTag>
            }
          >
            {reply ? (
              <div className="ops-stack">
                <div className="ops-callout">
                  <Icon name="user" size={16} />
                  <span>{asked}</span>
                </div>
                <pre className="ops-output">{reply.reply}</pre>
              </div>
            ) : (
              <OpsEmpty
                icon="sparkle"
                title="从一次运维检查开始"
                description="询问容器、网站或证书状态，助手会结合当前快照给出建议。"
              />
            )}
            {reply?.pending_write && (
              <div className="ops-list-item">
                <div className="ops-item-top">
                  <strong>待确认操作</strong>
                  <OpsTag tone="warning">需要确认</OpsTag>
                </div>
                <p className="ops-note">{reply.pending_write.name}</p>
                <pre className="ops-output">
                  {reply.pending_write.arguments}
                </pre>
                <div className="ops-actions">
                  <Button variant="primary" onClick={() => ask(true)}>
                    确认执行
                  </Button>
                  <Button
                    onClick={() =>
                      setReply({ reply: "已拒绝，没有执行写入。" })
                    }
                  >
                    拒绝操作
                  </Button>
                </div>
              </div>
            )}
            {!reply?.pending_write && (
              <>
                <div className="ops-suggestions">
                  {suggestions.map((question) => (
                    <button
                      type="button"
                      className="ops-suggestion"
                      key={question}
                      onClick={() => setMessage(question)}
                    >
                      {question}
                    </button>
                  ))}
                </div>
                <form
                  onSubmit={(event) => {
                    event.preventDefault();
                    ask(false);
                  }}
                >
                  <OpsField label="你的问题">
                    <Textarea
                      required
                      rows={4}
                      placeholder="描述你想了解的运行状态，或需要处理的问题…"
                      value={message}
                      onChange={(event) => setMessage(event.target.value)}
                    />
                  </OpsField>
                  <div className="ops-actions ops-actions-end">
                    <span className="ops-note">写入操作会在执行前请求确认</span>
                    <Button
                      variant="primary"
                      type="submit"
                      disabled={!message.trim()}
                    >
                      <Icon name="send" size={14} />
                      发送问题
                    </Button>
                  </div>
                </form>
              </>
            )}
          </OpsCard>
        </div>
        <div className="ops-stack">
          <OpsCard
            title="模型与连接"
            description="配置助手使用的模型服务"
            icon="settings"
          >
            <form
              onSubmit={(event) => {
                event.preventDefault();
                void feedback.run(async () => {
                  await api.ai.save({
                    base_url: base,
                    model,
                    api_key: key,
                    headers: headerName
                      ? { [headerName]: headerValue }
                      : undefined,
                  });
                  setKey("");
                  setHeaderValue("");
                  await load();
                }, "供应商配置已保存");
              }}
            >
              <OpsField label="接口地址">
                <Input
                  required
                  type="url"
                  value={base}
                  onChange={(event) => setBase(event.target.value)}
                />
              </OpsField>
              <OpsField label="模型名称">
                <Input
                  required
                  value={model}
                  onChange={(event) => setModel(event.target.value)}
                  placeholder="填写供应商支持的模型名称"
                />
              </OpsField>
              <OpsField
                label="API 密钥"
                hint={
                  keySet ? "密钥已设置，留空保留现有密钥" : "保存后不会回显密钥"
                }
              >
                <Input
                  type="password"
                  autoComplete="off"
                  value={key}
                  onChange={(event) => setKey(event.target.value)}
                  placeholder={keySet ? "••••••••（已保存）" : "输入 API 密钥"}
                />
              </OpsField>
              <details className="ops-details">
                <summary>
                  <Icon name="down" size={14} />
                  高级设置 · 附加请求头
                </summary>
                <div className="ops-stack">
                  <OpsField label="请求头名称">
                    <Input
                      placeholder="X-Custom-Header"
                      value={headerName}
                      onChange={(event) => setHeaderName(event.target.value)}
                    />
                  </OpsField>
                  <OpsField label="请求头值" hint="保存后不会回显">
                    <Input
                      type="password"
                      autoComplete="off"
                      value={headerValue}
                      onChange={(event) => setHeaderValue(event.target.value)}
                    />
                  </OpsField>
                </div>
              </details>
              <div className="ops-actions ops-actions-end">
                <Button type="submit" disabled={loading}>
                  保存配置
                </Button>
              </div>
            </form>
          </OpsCard>
          <div className="ops-callout">
            <Icon name="info" size={15} />
            <span>
              未配置模型服务时，助手仍可解释当前运行快照。连接服务需支持
              Responses API。
            </span>
          </div>
        </div>
      </div>
    </OpsPage>
  );
}
