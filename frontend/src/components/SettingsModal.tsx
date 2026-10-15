import { useEffect, useState, type FormEvent } from 'react';
import * as stylex from '@stylexjs/stylex';
import { Icon } from './Icon';
import { Button } from './ui/Button';
import { IconButton } from './ui/IconButton';
import { Badge } from './ui/Badge';
import { SegmentedControl } from './ui/SegmentedControl';
import { Input } from './ui/Input';
import { FormField } from './ui/FormField';
import { api, formatBytes, type Snapshot } from '../lib/api';
import { ACCENT_COLORS, applyThemeAccent, applyThemeBg, applyTypography, readTypography, type Density } from '../theme/tokens';

const styles = stylex.create({
  container: {
    display: 'grid',
    gridTemplateColumns: { default: '180px 1fr', '@media (max-width: 650px)': '1fr' },
    height: 'min(640px, calc(100dvh - 42px))',
    gridTemplateRows: { default: 'minmax(0, 1fr)', '@media (max-width: 650px)': 'auto minmax(0, 1fr)' },
    overflow: 'hidden',
  },
  sidebar: {
    backgroundColor: '#fafbfc',
    borderRight: '1px solid #e5e7eb',
    padding: '18px 12px',
    display: { default: 'flex', '@media (max-width: 650px)': 'grid' },
    gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
    flexDirection: 'column',
    gap: 4,
  },
  sidebarTitle: {
    gridColumn: '1 / -1',
    fontSize: 11.5,
    fontWeight: 650,
    color: '#64748b',
    padding: '0 10px 8px',
    textTransform: 'uppercase',
    letterSpacing: '0.6px',
  },
  sidebarItem: {
    display: 'flex',
    alignItems: 'center',
    gap: 9,
    padding: '8px 11px',
    borderRadius: 7,
    fontSize: 13,
    fontWeight: 500,
    color: '#334155',
    cursor: 'pointer',
    borderWidth: 0,
    backgroundColor: { default: 'transparent', ':hover': '#f1f5f9' },
    transition: 'all .14s',
    textAlign: 'left',
  },
  sidebarItemActive: {
    backgroundColor: { default: 'var(--accent-light, #fef2f0)', ':hover': 'color-mix(in srgb, var(--accent-light) 85%, var(--accent) 15%)' },
    color: 'var(--accent-text, #b93826)',
    fontWeight: 650,
  },
  content: {
    padding: { default: '24px 28px', '@media (max-width: 650px)': '20px' },
    minHeight: 0,
    minWidth: 0,
    overflowY: 'auto',
    overscrollBehaviorY: 'contain',
    scrollbarGutter: 'stable',
    display: 'flex',
    flexDirection: 'column',
    gap: 18,
  },
  contentHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    paddingBottom: 12,
    borderBottom: '1px solid #f1f5f9',
    flexShrink: 0,
  },
  headerTitle: {
    fontSize: 15,
    fontWeight: 700,
    color: '#0f172a',
  },
  row: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    gap: 18,
    flexWrap: 'wrap',
    padding: '9px 0',
  },
  label: {
    fontSize: 13,
    fontWeight: 600,
    color: '#0f172a',
  },
  desc: {
    fontSize: 12,
    color: '#64748b',
    marginTop: 2,
    lineHeight: 1.45,
  },
  colorPalette: {
    display: 'flex',
    alignItems: 'center',
    gap: 9,
    flexWrap: 'wrap',
  },
  colorCircle: {
    width: 28,
    height: 28,
    borderRadius: '50%',
    cursor: 'pointer',
    display: 'grid',
    placeItems: 'center',
    color: '#ffffff',
    border: '2px solid transparent',
    padding: 0,
    transition: 'transform .15s cubic-bezier(0.16, 1, 0.3, 1)',
    transform: { default: 'none', ':hover': 'scale(1.12)' },
  },
  colorCircleActive: {
    transform: { default: 'scale(1.12)', ':hover': 'scale(1.18)' },
    boxShadow: '0 0 0 2px #ffffff, 0 0 0 4px currentColor',
  },
  sliderWrapper: {
    display: 'flex',
    alignItems: 'center',
    gap: 14,
  },
  sliderInput: {
    width: 150,
    accentColor: 'var(--accent, #d85c49)',
    height: 5,
    cursor: 'pointer',
  },
  sliderValue: {
    fontSize: 12.5,
    fontFamily: 'ui-monospace, monospace',
    color: '#475569',
    minWidth: 42,
    textAlign: 'right',
    fontWeight: 500,
  },
  previewBox: {
    backgroundColor: '#f8fafc',
    border: '1px solid #e2e8f0',
    borderRadius: 9,
    padding: '16px 18px',
    marginTop: 6,
  },
  previewTitle: {
    fontSize: 14,
    fontWeight: 700,
    color: '#0f172a',
    marginBottom: 7,
  },
  previewText: {
    fontSize: 13,
    lineHeight: 1.7,
    color: '#334155',
  },
  previewHighlight: {
    color: 'var(--accent, #d85c49)',
    fontWeight: 650,
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: 16,
  },
  insight: {
    backgroundColor: '#f8fafc',
    padding: '11px 13px',
    borderRadius: 7,
    border: '1px solid #e2e8f0',
    fontSize: 12.5,
    lineHeight: 1.6,
    color: '#1e293b',
    display: 'flex',
    gap: 8,
  },
  actions: {
    display: 'flex',
    justifyContent: 'flex-end',
    gap: 9,
    marginTop: 10,
  },
  hostTitle: {
    display: 'flex',
    alignItems: 'center',
    gap: 10,
    marginBottom: 14,
  },
  hostIcon: {
    width: 34,
    height: 34,
    borderRadius: 8,
    backgroundColor: '#f1f5f9',
    display: 'grid',
    placeItems: 'center',
    color: '#475569',
    flexShrink: 0,
    border: '1px solid #e2e8f0',
  },
  hostRow: {
    display: 'flex',
    justifyContent: 'space-between',
    gap: 14,
    color: '#475569',
    fontSize: 12.5,
    padding: '7px 0',
    borderBottom: '1px solid #f8fafc',
    lineHeight: 1.45,
  },
  hostValue: {
    color: '#0f172a',
    fontWeight: 600,
    textAlign: 'right',
  },
});

export function SettingsModal({
  token,
  onSaveToken,
  data,
  onClose,
}: {
  token: string;
  onSaveToken: (token: string) => void;
  data: Snapshot | null;
  onClose: () => void;
}) {
  const [tab, setTab] = useState<'appearance' | 'connection' | 'engine' | 'shortcuts'>('appearance');
  const [inputToken, setInputToken] = useState(token);
  const [exposure, setExposure] = useState('公网暴露面板需要 HTTPS');
  useEffect(() => {
    if (tab !== 'connection') return;
    void api.exposure().then(info => setExposure(info.https ? info.message : '公网暴露面板需要 HTTPS')).catch(() => setExposure('公网暴露面板需要 HTTPS'));
  }, [tab]);

  // 外观设置状态
  const [activeAccent, setActiveAccent] = useState(() => localStorage.getItem('camo-accent') || 'coral');
  const [bgColor, setBgColor] = useState<'warm' | 'white'>(
    () => (localStorage.getItem('camo-bg') as 'warm' | 'white') || 'warm'
  );
  const [typography] = useState(readTypography);
  const [density, setDensity] = useState<Density>(typography.density);
  const [fontSize, setFontSize] = useState(typography.fontSize);
  const [lineHeight, setLineHeight] = useState(typography.lineHeight);

  useEffect(() => {
    applyTypography(fontSize, lineHeight, density);
  }, [fontSize, lineHeight, density]);

  function handleSelectAccent(colorId: string) {
    setActiveAccent(colorId);
    applyThemeAccent(colorId);
  }

  function handleSelectBg(bg: 'warm' | 'white') {
    setBgColor(bg);
    applyThemeBg(bg);
  }

  function handleSubmitToken(e: FormEvent) {
    e.preventDefault();
    onSaveToken(inputToken.trim());
    onClose();
  }

  return (
    <div {...stylex.props(styles.container)}>
      {/* 弹窗左侧导航栏 */}
      <aside {...stylex.props(styles.sidebar)}>
        <p {...stylex.props(styles.sidebarTitle)}>设置面板</p>
        <button
          {...stylex.props(styles.sidebarItem, tab === 'appearance' && styles.sidebarItemActive)}
          onClick={() => setTab('appearance')}
        >
          <Icon name="palette" size={16} />
          <span>外观风格</span>
        </button>
        <button
          {...stylex.props(styles.sidebarItem, tab === 'connection' && styles.sidebarItemActive)}
          onClick={() => setTab('connection')}
        >
          <Icon name="server" size={16} />
          <span>连接与令牌</span>
        </button>
        <button
          {...stylex.props(styles.sidebarItem, tab === 'engine' && styles.sidebarItemActive)}
          onClick={() => setTab('engine')}
        >
          <Icon name="cpu" size={16} />
          <span>Docker 引擎</span>
        </button>
        <button
          {...stylex.props(styles.sidebarItem, tab === 'shortcuts' && styles.sidebarItemActive)}
          onClick={() => setTab('shortcuts')}
        >
          <Icon name="keyboard" size={16} />
          <span>快捷键</span>
        </button>
      </aside>

      {/* 弹窗右侧设置详情 */}
      <div {...stylex.props(styles.content)}>
        {tab === 'appearance' && (
          <>
            <div {...stylex.props(styles.contentHeader)}>
              <h3 {...stylex.props(styles.headerTitle)}>外观设置</h3>
              <IconButton aria-label="关闭" onClick={onClose}>
                <Icon name="close" size={16} />
              </IconButton>
            </div>

            {/* 强调色 */}
            <div {...stylex.props(styles.row)}>
              <div>
                <p {...stylex.props(styles.label)}>强调色</p>
                <p {...stylex.props(styles.desc)}>应用于按钮、高亮标签与活动状态</p>
              </div>
              <div {...stylex.props(styles.colorPalette)}>
                {ACCENT_COLORS.map(color => {
                  const isSelected = activeAccent === color.id;
                  return (
                    <button
                      key={color.id}
                      type="button"
                      {...stylex.props(styles.colorCircle, isSelected && styles.colorCircleActive)}
                      style={{ backgroundColor: color.hex, color: color.hex }}
                      onClick={() => handleSelectAccent(color.id)}
                      title={color.name}
                      aria-label={color.name}
                      aria-pressed={isSelected}
                    >
                      {isSelected && <Icon name="check" size={13} style={{ color: '#ffffff' }} />}
                    </button>
                  );
                })}
              </div>
            </div>

            {/* 背景色 */}
            <div {...stylex.props(styles.row)}>
              <div>
                <p {...stylex.props(styles.label)}>背景色调</p>
                <p {...stylex.props(styles.desc)}>选择暖纸色或清爽的浅色背景</p>
              </div>
              <SegmentedControl
                label="背景色调"
                value={bgColor}
                onChange={handleSelectBg}
                options={[
                  { value: 'warm', label: '暖纸色' },
                  { value: 'white', label: '浅色' },
                ]}
              />
            </div>

            {/* 界面密度 */}
            <div {...stylex.props(styles.row)}>
              <div>
                <p {...stylex.props(styles.label)}>界面密度</p>
                <p {...stylex.props(styles.desc)}>控制列表、表格及卡片的间距紧凑度</p>
              </div>
              <SegmentedControl
                label="界面密度"
                value={density}
                onChange={setDensity}
                options={[
                  { value: 'comfort', label: '舒适' },
                  { value: 'compact', label: '紧凑' },
                ]}
              />
            </div>

            {/* 正文字号调节 */}
            <div {...stylex.props(styles.row)}>
              <div>
                <p {...stylex.props(styles.label)}>正文字号</p>
                <p {...stylex.props(styles.desc)}>调整界面文字大小，即时生效并保存</p>
              </div>
              <div {...stylex.props(styles.sliderWrapper)}>
                <input
                  type="range"
                  min="12"
                  max="18"
                  step="1"
                  aria-label="正文字号"
                  value={fontSize}
                  onChange={e => setFontSize(Number(e.target.value))}
                  {...stylex.props(styles.sliderInput)}
                />
                <span {...stylex.props(styles.sliderValue)}>{fontSize}px</span>
              </div>
            </div>

            {/* 行高调节 */}
            <div {...stylex.props(styles.row)}>
              <div>
                <p {...stylex.props(styles.label)}>正文行高</p>
                <p {...stylex.props(styles.desc)}>调整正文行距，即时生效并保存</p>
              </div>
              <div {...stylex.props(styles.sliderWrapper)}>
                <input
                  type="range"
                  min="1.3"
                  max="1.8"
                  step="0.05"
                  aria-label="正文行高"
                  value={lineHeight}
                  onChange={e => setLineHeight(Number(e.target.value))}
                  {...stylex.props(styles.sliderInput)}
                />
                <span {...stylex.props(styles.sliderValue)}>{lineHeight.toFixed(2)}</span>
              </div>
            </div>

            {/* 即时排版预览 */}
            <div>
              <p {...stylex.props(styles.label)} style={{ marginBottom: 6 }}>
                排版效果预览
              </p>
              <div {...stylex.props(styles.previewBox)}>
                <p {...stylex.props(styles.previewTitle)}>容器运行状态</p>
                <p {...stylex.props(styles.previewText)} style={{ fontSize, lineHeight }}>
                  服务 camo-web 正在运行，镜像 nginx:alpine。使用搜索快速定位容器，或查看{' '}
                  <span {...stylex.props(styles.previewHighlight)}>运行日志</span> 了解服务状态。
                </p>
              </div>
            </div>
          </>
        )}

        {tab === 'connection' && (
          <form {...stylex.props(styles.form)} onSubmit={handleSubmitToken}>
            <div {...stylex.props(styles.contentHeader)}>
              <h3 {...stylex.props(styles.headerTitle)}>API 连接与令牌</h3>
              <IconButton aria-label="关闭" onClick={onClose}>
                <Icon name="close" size={16} />
              </IconButton>
            </div>
            <p {...stylex.props(styles.desc)}>
              当前通过同源 API 访问后端。生产环境或反向代理需携带{' '}
              <code style={{ color: 'var(--accent, #d85c49)' }}>CAMO_API_TOKEN</code> 进行鉴权。
            </p>
            <p {...stylex.props(styles.desc)}>{exposure}。未同时配置 CAMO_TLS_CERT 与 CAMO_TLS_KEY 时，面板继续使用 HTTP。</p>
            <FormField label="API 访问令牌">
              <Input
                type="password"
                autoComplete="off"
                value={inputToken}
                onChange={e => setInputToken(e.target.value)}
                placeholder="未配置可留空；配置后仅保存在当前标签页"
              />
            </FormField>
            <div {...stylex.props(styles.insight)}>
              <Icon name="info" size={16} />
              <span>
                令牌保存在 sessionStorage，关闭浏览器标签后自动清除。Docker 模式与 Socket 路径由后端 .env 文件指定。
              </span>
            </div>
            <div {...stylex.props(styles.actions)}>
              <Button type="button" onClick={onClose}>
                取消
              </Button>
              <Button variant="primary" type="submit">
                <span>保存并连接</span>
                <Icon name="arrow" size={14} />
              </Button>
            </div>
          </form>
        )}

        {tab === 'engine' && (
          <>
            <div {...stylex.props(styles.contentHeader)}>
              <h3 {...stylex.props(styles.headerTitle)}>Docker 引擎环境</h3>
              <IconButton aria-label="关闭" onClick={onClose}>
                <Icon name="close" size={16} />
              </IconButton>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
              <div {...stylex.props(styles.hostTitle)}>
                <span {...stylex.props(styles.hostIcon)}>
                  <Icon name="server" size={17} />
                </span>
                <div>
                  <p style={{ fontSize: 14, fontWeight: 700, color: '#0f172a' }}>
                    {data?.engine.Name || '本地 Docker 引擎'}
                  </p>
                  <span style={{ fontSize: 12, color: '#64748b' }}>
                    {data?.engine.OperatingSystem || 'Linux / Darwin'}
                  </span>
                </div>
              </div>
              {[
                ['连接模式', 'Unix Socket 实时引擎'],
                ['Docker 版本', data?.engine.ServerVersion || '—'],
                ['处理器核心', data ? `${data.engine.NCPU} 核` : '—'],
                ['内存容量', data ? formatBytes(data.engine.MemTotal) : '—'],
                ['容器总数', data ? `${data.containers.length} 个` : '—'],
                ['本地镜像', data ? `${data.images.length} 个` : '—'],
              ].map(([k, v]) => (
                <div key={k} {...stylex.props(styles.hostRow)}>
                  <span>{k}</span>
                  <span {...stylex.props(styles.hostValue)}>{v}</span>
                </div>
              ))}
            </div>
          </>
        )}

        {tab === 'shortcuts' && (
          <>
            <div {...stylex.props(styles.contentHeader)}>
              <h3 {...stylex.props(styles.headerTitle)}>常用快捷键与操作</h3>
              <IconButton aria-label="关闭" onClick={onClose}>
                <Icon name="close" size={16} />
              </IconButton>
            </div>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {[
                { key: 'Ctrl / ⌘ + K', desc: '聚焦并定位至搜索框' },
                { key: 'Esc', desc: '快速关闭当前弹窗或清除搜索' },
                { key: 'Tab', desc: '在表格与按钮间跳转' },
                { key: 'Enter', desc: '确认创建或提交表单' },
              ].map(item => (
                <div key={item.key} {...stylex.props(styles.row)}>
                  <Badge variant="kbd">{item.key}</Badge>
                  <span style={{ fontSize: 12.5, color: '#64748b' }}>{item.desc}</span>
                </div>
              ))}
            </div>
          </>
        )}
      </div>
    </div>
  );
}
