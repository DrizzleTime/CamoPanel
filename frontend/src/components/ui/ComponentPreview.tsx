import { useState } from "react";
import { Button } from "./Button";
import { IconButton } from "./IconButton";
import { Input, Textarea } from "./Input";
import { Select } from "./Select";
import { FormField } from "./FormField";
import { Checkbox, Switch } from "./Checkbox";
import { DropdownMenu } from "./DropdownMenu";
import { Alert } from "./Alert";
import { Tooltip } from "./Tooltip";
import { Panel, PanelHeader } from "./Panel";
import { Modal, ModalFooter } from "../Modal";
import { Icon } from "../Icon";
import {
  ACCENT_COLORS,
  applyThemeAccent,
  applyThemeBg,
  applyTypography,
  readTypography,
} from "../../theme/tokens";
import "./preview.css";

const policyOptions = [
  { value: "allow", label: "允许访问" },
  { value: "deny", label: "拒绝访问" },
];
const longOptions = Array.from({ length: 40 }, (_, index) => ({
  value: String(index + 1),
  label: `服务器 ${String(index + 1).padStart(2, "0")}`,
  disabled: index === 2,
}));

export default function ComponentPreview() {
  const [accent, setAccent] = useState(
    localStorage.getItem("camo-accent") || "coral",
  );
  const [density, setDensity] = useState(readTypography().density);
  const [background, setBackground] = useState(
    localStorage.getItem("camo-bg") || "warm",
  );
  const [fontSize, setFontSize] = useState(String(readTypography().fontSize));
  const [policy, setPolicy] = useState("allow");
  const [server, setServer] = useState("");
  const [modalServer, setModalServer] = useState("1");
  const [checked, setChecked] = useState(true);
  const [enabled, setEnabled] = useState(false);
  const [modal, setModal] = useState(false);
  const [message, setMessage] = useState("");
  const [busy, setBusy] = useState(false);
  return (
    <main className="ui-preview">
      <header className="ui-preview-heading">
        <div>
          <span className="ui-preview-eyebrow">CAMOPANEL / UI</span>
          <h1>基础组件</h1>
          <p>统一的表单、操作和反馈。此预览仅在开发模式开放。</p>
        </div>
        <Button onClick={() => setModal(true)}>
          打开测试弹窗
          <Icon name="external" size={15} />
        </Button>
      </header>
      <section className="ui-preview-settings" aria-label="外观预览">
        <FormField label="主题色">
          <Select
            value={accent}
            onValueChange={(value) => {
              setAccent(value);
              applyThemeAccent(value);
            }}
            options={ACCENT_COLORS.map((color) => ({
              value: color.id,
              label: color.name,
            }))}
          />
        </FormField>
        <FormField label="背景">
          <Select
            value={background}
            onValueChange={(value) => {
              setBackground(value);
              applyThemeBg(value as "warm" | "white");
            }}
            options={[
              { value: "warm", label: "暖色背景" },
              { value: "white", label: "冷白背景" },
            ]}
          />
        </FormField>
        <FormField label="密度">
          <Select
            value={density}
            onValueChange={(value) => {
              const next = value as "comfort" | "compact";
              setDensity(next);
              applyTypography(Number(fontSize), 1.6, next);
            }}
            options={[
              { value: "comfort", label: "舒适" },
              { value: "compact", label: "紧凑" },
            ]}
          />
        </FormField>
        <FormField label="字号">
          <Select
            value={fontSize}
            onValueChange={(value) => {
              setFontSize(value);
              applyTypography(Number(value), 1.6, density);
            }}
            options={[12, 14, 16, 18].map((size) => ({
              value: String(size),
              label: `${size} px`,
            }))}
          />
        </FormField>
      </section>
      {message && (
        <Alert tone="success" onDismiss={() => setMessage("")}>
          {message}
        </Alert>
      )}
      <div className="ui-preview-grid">
        <Panel>
          <PanelHeader title="表单组件" icon={<Icon name="edit" />} />
          <form
            className="ui-preview-body"
            onSubmit={(event) => {
              event.preventDefault();
              setMessage(
                `已提交 · ${policy === "allow" ? "允许访问" : "拒绝访问"} · 服务器 ${server}`,
              );
            }}
          >
            <FormField label="端口" hint="端口范围 1–65535">
              <Input
                type="number"
                required
                min={1}
                max={65535}
                defaultValue="22"
              />
            </FormField>
            <FormField label="访问策略">
              <Select
                value={policy}
                onValueChange={setPolicy}
                options={policyOptions}
                name="policy"
              />
            </FormField>
            <FormField label="目标服务器" hint="支持键盘选择，服务器 03 不可选">
              <Select
                required
                value={server}
                onValueChange={setServer}
                options={longOptions}
                placeholder="请选择服务器"
                name="server"
              />
            </FormField>
            <FormField label="备注">
              <Textarea placeholder="输入备注" />
            </FormField>
            <Button variant="primary" type="submit">
              提交表单
            </Button>
          </form>
        </Panel>
        <div className="ui-preview-stack">
          <Panel>
            <PanelHeader title="操作与选择" icon={<Icon name="grid" />} />
            <div className="ui-preview-body">
              <div className="ui-preview-row">
                <Button variant="primary">主要操作</Button>
                <Button>次要操作</Button>
                <Button variant="ghost">辅助操作</Button>
                <Button variant="danger">删除</Button>
              </div>
              <div className="ui-preview-row">
                <Button size="sm">小尺寸</Button>
                <Button disabled>不可用</Button>
                <Tooltip content="刷新数据">
                  <IconButton aria-label="刷新数据">
                    <Icon name="refresh" size={16} />
                  </IconButton>
                </Tooltip>
                <DropdownMenu
                  trigger={
                    <Button size="sm">
                      更多操作
                      <Icon name="down" size={14} />
                    </Button>
                  }
                  items={[
                    {
                      label: "复制",
                      icon: "copy",
                      shortcut: "Ctrl+C",
                      onSelect: () => setMessage("已复制"),
                    },
                    {
                      label: "不可用操作",
                      icon: "edit",
                      disabled: true,
                      onSelect: () => {},
                    },
                    {
                      label: "删除",
                      icon: "trash",
                      danger: true,
                      separator: true,
                      onSelect: () => setMessage("已选择删除"),
                    },
                  ]}
                />
              </div>
              <div className="ui-preview-row">
                <label className="ui-preview-check">
                  <Checkbox
                    checked={checked}
                    onChange={(event) => setChecked(event.target.checked)}
                  />
                  显示隐藏文件
                </label>
                <label className="ui-preview-check">
                  <Checkbox indeterminate aria-label="部分选中" />
                  部分选中
                </label>
                <label className="ui-preview-check">
                  <Checkbox disabled />
                  不可用
                </label>
              </div>
              <Switch
                label="启用通知"
                checked={enabled}
                onChange={(event) => setEnabled(event.target.checked)}
              />
            </div>
          </Panel>
          <Panel>
            <PanelHeader title="状态与反馈" icon={<Icon name="info" />} />
            <div className="ui-preview-body">
              <FormField label="无效字段" error="请输入有效的目录名称">
                <Input defaultValue="invalid/path" />
              </FormField>
              <FormField label="禁用字段">
                <Select disabled options={policyOptions} value="allow" />
              </FormField>
              <Alert tone="info">操作提示与辅助说明</Alert>
              <Alert tone="warning">此操作需要确认</Alert>
              <Alert tone="error">保存失败，请重试</Alert>
            </div>
          </Panel>
        </div>
      </div>
      {modal && (
        <Modal structured title="弹窗内的组件" onClose={() => setModal(false)}>
          <div className="ui-preview-body">
            <FormField label="弹窗服务器">
              <Select
                value={modalServer}
                onValueChange={setModalServer}
                options={longOptions}
              />
            </FormField>
            <FormField label="弹窗备注">
              <Input placeholder="测试焦点恢复" />
            </FormField>
            <DropdownMenu
              trigger={<Button>弹窗菜单</Button>}
              items={[
                {
                  label: "测试菜单操作",
                  icon: "check",
                  onSelect: () => setMessage("弹窗菜单已选择"),
                },
              ]}
            />
            <Switch
              label="模拟处理中"
              checked={busy}
              onChange={(event) => setBusy(event.target.checked)}
            />
            <fieldset disabled={busy} className="ui-preview-fieldset">
              <FormField label="受表单禁用控制的下拉框">
                <Select options={policyOptions} defaultValue="allow" />
              </FormField>
            </fieldset>
          </div>
          <ModalFooter>
            <span />
            <Button onClick={() => setModal(false)}>完成</Button>
          </ModalFooter>
        </Modal>
      )}
    </main>
  );
}
