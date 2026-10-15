import { test, expect } from "./fixtures";

test("下拉框支持键盘、长列表和禁用选项", async ({ page }) => {
  await page.goto("/__ui");
  const policy = page.getByRole("combobox", { name: "访问策略", exact: true });
  await policy.focus();
  await page.keyboard.press("Space");
  await page.keyboard.press("End");
  await expect(
    page.getByRole("option", { name: "拒绝访问", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(policy).toHaveText("拒绝访问");
  await expect(policy).toBeFocused();
  await page.getByRole("combobox", { name: "目标服务器", exact: true }).click();
  await expect(page.getByRole("option", { name: "服务器 03" })).toHaveAttribute(
    "data-disabled",
    "",
  );
  await page.keyboard.press("Home");
  await expect(
    page.getByRole("option", { name: "服务器 01", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await expect(
    page.getByRole("option", { name: "服务器 02", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("ArrowDown");
  await expect(
    page.getByRole("option", { name: "服务器 04", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("combobox", { name: "目标服务器", exact: true }),
  ).toHaveText("服务器 04");
  await page.getByRole("combobox", { name: "目标服务器", exact: true }).click();
  await page.keyboard.press("End");
  await expect(
    page.getByRole("option", { name: "服务器 40", exact: true }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("combobox", { name: "目标服务器", exact: true }),
  ).toHaveText("服务器 40");
});

test("弹窗内下拉框与菜单可点击，Esc 逐层关闭并恢复焦点", async ({ page }) => {
  await page.goto("/__ui");
  const trigger = page.getByRole("button", { name: "打开测试弹窗" });
  await trigger.click();
  const dialog = page.getByRole("dialog", { name: "弹窗内的组件" });
  const select = dialog.getByRole("combobox", {
    name: "弹窗服务器",
    exact: true,
  });
  await select.click();
  const option = page.getByRole("option", { name: "服务器 02", exact: true });
  await expect(option).toBeVisible();
  expect(
    await option.evaluate((node) => {
      const rect = node.getBoundingClientRect();
      return node.contains(
        document.elementFromPoint(
          rect.x + rect.width / 2,
          rect.y + rect.height / 2,
        ),
      );
    }),
  ).toBe(true);
  await option.click();
  await expect(select).toHaveText("服务器 02");
  await select.click();
  await page.keyboard.press("Escape");
  await expect(page.getByRole("listbox")).toHaveCount(0);
  await expect(dialog).toBeVisible();
  await expect(select).toBeFocused();
  await dialog.getByRole("button", { name: "弹窗菜单", exact: true }).click();
  await page.getByRole("menuitem", { name: "测试菜单操作" }).click();
  await expect(dialog).toBeVisible();
  await dialog.getByRole("switch", { name: "模拟处理中" }).check();
  await expect(
    dialog.getByRole("combobox", { name: "受表单禁用控制的下拉框" }),
  ).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(dialog).toHaveCount(0);
  await expect(trigger).toBeFocused();
});

test("必填下拉框保留原生表单校验，主题与密度即时生效", async ({ page }) => {
  await page.goto("/__ui");
  await page.getByRole("button", { name: "提交表单" }).click();
  await expect(
    page.getByRole("status").filter({ hasText: "已提交" }),
  ).toHaveCount(0);
  await page.getByRole("combobox", { name: "目标服务器", exact: true }).click();
  await page.getByRole("option", { name: "服务器 01", exact: true }).click();
  await page.getByRole("button", { name: "提交表单" }).click();
  await expect(
    page.getByRole("status").filter({ hasText: "已提交" }),
  ).toContainText("服务器 1");
  const port = page.getByRole("spinbutton", { name: "端口", exact: true });
  const comfortableHeight = (await port.boundingBox())!.height;
  await page.getByRole("combobox", { name: "密度", exact: true }).click();
  await page.getByRole("option", { name: "紧凑", exact: true }).click();
  expect((await port.boundingBox())!.height).toBeLessThan(comfortableHeight);
  await page.getByRole("combobox", { name: "主题色", exact: true }).click();
  await page.getByRole("option", { name: "经典蓝", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "主要操作", exact: true }),
  ).toHaveCSS("background-color", "rgb(37, 99, 235)");
});

test("防火墙策略切换保留面板端口确认与请求参数", async ({ page }) => {
  await page.goto("/firewall");
  await page
    .getByRole("spinbutton", { name: "端口", exact: true })
    .fill("8080");
  await page.getByRole("combobox", { name: "访问策略", exact: true }).click();
  await page.getByRole("option", { name: "拒绝访问" }).click();
  const submit = page.getByRole("button", { name: "添加规则", exact: true });
  await expect(submit).toBeDisabled();
  await page
    .getByRole("checkbox", { name: "我确认此规则可能关闭面板访问端口" })
    .check();
  const request = page.waitForRequest((request) =>
    request.url().endsWith("/firewall/rules"),
  );
  await submit.click();
  expect((await request).postDataJSON()).toEqual({
    port: 8080,
    policy: "deny",
    confirm: true,
  });
  await expect(page.getByRole("status")).toContainText("端口规则已添加");
});

test("文件管理支持多选、新建、重命名、右键删除和网格切换", async ({ page }) => {
  await page.goto("/files");
  await page
    .getByRole("checkbox", { name: "选择 README.md", exact: true })
    .check();
  const all = page.getByRole("checkbox", { name: "选择当前目录全部项目" });
  await expect(all).toBeChecked({ indeterminate: true });
  await all.check();
  await expect(all).toBeChecked();
  await all.uncheck();
  await page.getByRole("button", { name: "新建", exact: true }).click();
  await page.getByRole("menuitem", { name: "文件夹", exact: true }).click();
  let dialog = page.getByRole("dialog");
  await expect(dialog).toBeVisible();
  await expect(dialog.getByRole("textbox", { name: "名称" })).toBeFocused();
  await dialog.getByRole("textbox", { name: "名称" }).fill("release");
  await dialog.getByRole("button", { name: "确认", exact: true }).click();
  await expect(dialog).toHaveCount(0);
  const row = page.getByRole("row").filter({
    has: page.getByRole("checkbox", { name: "选择 release", exact: true }),
  });
  await expect(row).toBeVisible();
  await row.getByRole("button", { name: "release 操作菜单" }).click();
  await page.getByRole("menuitem", { name: /重命名/ }).click();
  dialog = page.getByRole("dialog");
  await dialog.getByRole("textbox", { name: "名称" }).fill("release-v2");
  await dialog.getByRole("button", { name: "确认", exact: true }).click();
  const renamed = page.getByRole("row").filter({
    has: page.getByRole("checkbox", { name: "选择 release-v2", exact: true }),
  });
  await expect(renamed).toBeVisible();
  await renamed.click({ button: "right" });
  await page.getByRole("menuitem", { name: /^删除/ }).click();
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "确认删除", exact: true })
    .click();
  await expect(
    page.getByRole("checkbox", { name: "选择 release-v2", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("radio", { name: "图标视图" }).click();
  await expect(page.getByRole("radio", { name: "图标视图" })).toBeChecked();
  await page.getByRole("switch", { name: "显示隐藏文件" }).check();
  await expect(
    page.getByRole("checkbox", { name: "选择 .hidden", exact: true }),
  ).toBeVisible();
});

test("文件编辑器保存与权限复选框保留原有行为", async ({ page }) => {
  await page.goto("/files");
  await page.getByRole("button", { name: "README.md", exact: true }).dblclick();
  let dialog = page.getByRole("dialog");
  await dialog.getByRole("textbox", { name: "文件内容" }).fill("# Updated");
  const saveRequest = page.waitForRequest(
    (request) => request.url().endsWith("/files") && request.method() === "PUT",
  );
  await dialog.getByRole("button", { name: "保存文件" }).click();
  expect((await saveRequest).postDataJSON().content).toBe("# Updated");
  await dialog
    .getByRole("button", { name: "关闭", exact: true })
    .last()
    .click();
  const row = page.getByRole("row").filter({
    has: page.getByRole("checkbox", { name: "选择 README.md", exact: true }),
  });
  await row.getByRole("button", { name: "0644" }).click();
  dialog = page.getByRole("dialog");
  await dialog.getByRole("checkbox", { name: "所有者执行" }).check();
  await expect(dialog.getByRole("textbox", { name: "权限代码" })).toHaveValue(
    "0744",
  );
});

for (const width of [390, 1024, 1920]) {
  test(`文件与防火墙在 ${width}px 宽度下无页面横向溢出`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    for (const path of ["/files", "/firewall", "/__ui"]) {
      await page.goto(path);
      await expect(
        page.locator(
          path === "/files"
            ? ".fe-explorer"
            : path === "/firewall"
              ? ".ops-content"
              : ".ui-preview",
        ),
      ).toBeVisible();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
      ).toBe(true);
      if (width !== 1024)
        await page.screenshot({
          path: test.info().outputPath(`${path.slice(1)}-${width}.png`),
          fullPage: true,
        });
      if (width === 1920 && path === "/firewall") {
        await page
          .getByRole("combobox", { name: "访问策略", exact: true })
          .click();
        await page.screenshot({
          path: test.info().outputPath("firewall-select.png"),
          fullPage: false,
        });
      }
    }
  });
}
