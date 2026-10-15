# 界面组件与验收

首批统一设计变量、基础组件、防火墙与文件管理，并将全部业务页面的原生下拉框迁移到公共 Select。

## 设计变量

`frontend/src/reset.css` 定义颜色、文字层级、间距、圆角、控件尺寸、阴影与状态颜色。`frontend/src/theme/tokens.ts` 将保存的主题色、背景、字号与密度应用到变量。

| 项目 | 舒适 | 紧凑 |
| --- | --- | --- |
| 常规控件最小高度 | 40px | 36px |
| 小尺寸控件最小高度 | 34px | 30px |
| 文件表格行高 | 52px | 42px |

文字使用 `--text-xs`、`--text-sm`、`--text-md`、`--text-lg`，随用户字号设置调整。卡片、表单、弹窗和菜单分别使用对应的语义变量。文件类型图标仍保留各自的识别颜色。

公共组件采用 StyleX 和 `components` CSS 层。复杂状态选择器集中在 `components/ui/ui.css`，页面 CSS 负责工作区、目录树、表格列宽、上传队列等业务布局。继续迁移页面时，应删除原先的控件样式，避免重复覆盖。

## 公共组件

路径：`frontend/src/components/ui/`。

| 组件 | 用途 |
| --- | --- |
| `Input`、`Textarea` | 表单输入，保留原生类型、约束校验、禁用和 ref |
| `FormField` | 标签、提示、错误及其无障碍关联；单个子控件自动分配 ID |
| `Select` | 统一的触发器与选项列表，支持键盘、长列表、禁用项、必填校验 |
| `Checkbox`、`Switch` | 复选框、半选状态、开关；保留原生表单行为 |
| `Button`、`IconButton` | 主要、次要、辅助、危险操作和图标按钮 |
| `DropdownMenu`、`ContextMenu` | 下拉操作与指定位置的右键菜单，包含分隔线、快捷键与危险操作 |
| `Tooltip`、`Alert` | 操作说明，以及成功、错误、警告和普通提示 |
| `Panel`、`PanelHeader` | 卡片与标题，支持图标、描述和附加操作 |
| `SearchInput`、`SegmentedControl` | 搜索和互斥视图选择 |

示例：

```tsx
<FormField label="访问策略" hint="设置端口的访问权限">
  <Select
    value={policy}
    onValueChange={setPolicy}
    options={[
      { value: 'allow', label: '允许访问' },
      { value: 'deny', label: '拒绝访问' },
    ]}
  />
</FormField>
```

Select 的选项值使用非空字符串，未选择时使用 `value=""` 与 `placeholder`。如果需要“无”选项，为它指定明确的值，再在业务边界转换，例如容器重启策略中的 `unset`。

`components/Modal.tsx` 统一普通弹窗与文件工作弹窗。`structured` 模式提供标准标题栏，正文由业务布局，底部使用 `ModalFooter`。Modal 会提供浮层挂载容器，避免 Select、Tooltip、DropdownMenu 被原生 dialog 顶层遮住。输入组件的 `autoFocus` 会在弹窗打开后生效，关闭后恢复原焦点。

Select、DropdownMenu、Tooltip 的交互基于 Radix Primitives；ContextMenu 为项目公共组件。菜单共用同一组视觉样式。

## 预览与检查

启动 `bun run dev:web`，访问 `http://127.0.0.1:5173/__ui` 查看组件状态。该路径只在 Vite 开发模式开放，不进入生产构建。可以切换主题色、背景、密度和字号，检查普通、选中、禁用、错误、半选、弹窗及长列表状态。预览切换会保存到当前浏览器的外观设置。

```bash
bun run test:web
bun run --cwd frontend build

# 首次运行浏览器测试时安装 Chromium
cd frontend
bunx playwright install chromium
cd ..
bun run test:ui
```

浏览器测试使用独立的 5180 端口，拦截所有 `/api/` 请求，使用模拟数据，不修改实际文件或防火墙。测试覆盖键盘选择、禁用项、长列表、必填校验、主题与密度、弹窗浮层和焦点、面板端口确认，以及文件多选、新建、重命名、删除、编辑、权限与视图切换。响应式检查覆盖 390px、1024px、1920px；桌面与手机截图输出到 `frontend/test-results/`。

后续可继续将网站、应用商店、Docker 页面中剩余的独立输入样式、业务卡片和反馈布局迁移到这些组件。
