import { test, expect } from './fixtures';

for (const viewport of [
  { width: 1440, height: 1000 },
  { width: 390, height: 844 },
  { width: 844, height: 390 },
]) {
  test(`设置切换时保持高度和位置，内容在内部滚动 ${viewport.width}×${viewport.height}`, async ({ page }) => {
    await page.setViewportSize(viewport);
    await page.goto('/containers');
    await page.getByRole('button', { name: '外观与设置', exact: true }).click();
    const dialog = page.getByRole('dialog', { name: '外观与连接设置' });
    await expect(dialog).toBeVisible();
    const initial = (await dialog.boundingBox())!;
    expect(initial.height).toBeLessThanOrEqual(viewport.height - 40);
    expect(initial.y).toBeGreaterThanOrEqual(0);

    for (const tab of ['连接与令牌', 'Docker 引擎', '快捷键', '外观风格']) {
      await dialog.getByRole('button', { name: tab, exact: true }).click();
      const box = (await dialog.boundingBox())!;
      expect(box.height).toBeCloseTo(initial.height, 0);
      expect(box.y).toBeCloseTo(initial.y, 0);
      expect(box.width).toBeCloseTo(initial.width, 0);
    }

    if (viewport.height < 500 || viewport.width < 650) {
      const content = dialog.locator('aside + div');
      expect(await content.evaluate(node => node.scrollHeight > node.clientHeight)).toBe(true);
      await dialog.getByText('排版效果预览', { exact: true }).scrollIntoViewIfNeeded();
      expect(await content.evaluate(node => node.scrollTop)).toBeGreaterThan(0);
      // Scrolling the content must keep the navigation visible.
      await expect(dialog.getByRole('button', { name: '连接与令牌', exact: true })).toBeInViewport();
    }
    await page.screenshot({ path: test.info().outputPath('settings-appearance.png') });
    await dialog.getByRole('button', { name: '连接与令牌', exact: true }).click();
    await dialog.getByLabel('API 访问令牌', { exact: true }).scrollIntoViewIfNeeded();
    await page.screenshot({ path: test.info().outputPath('settings-connection.png') });
  });
}

test('令牌输入框使用统一控件样式并保留保存行为', async ({ page }) => {
  await page.goto('/containers');
  await page.getByRole('button', { name: '外观与设置', exact: true }).click();
  const dialog = page.getByRole('dialog', { name: '外观与连接设置' });
  await dialog.getByRole('button', { name: '经典蓝', exact: true }).click();
  await dialog.getByRole('button', { name: '连接与令牌', exact: true }).click();
  const input = dialog.getByLabel('API 访问令牌', { exact: true });
  await expect(input).toHaveAttribute('type', 'password');
  await expect(input).toHaveCSS('border-top-width', '1px');
  await expect(input).toHaveCSS('border-radius', '8px');
  await input.focus();
  await expect(input).toHaveCSS('outline-style', 'none');
  await expect(input).toHaveCSS('border-top-color', 'rgb(37, 99, 235)');
  await expect(input).toHaveCSS('box-shadow', 'rgb(239, 246, 255) 0px 0px 0px 3px');
  await input.fill('  example-api-token  ');
  await dialog.getByRole('button', { name: '保存并连接', exact: true }).click();
  await expect(dialog).toHaveCount(0);
  expect(await page.evaluate(() => sessionStorage.getItem('camo-token'))).toBe('example-api-token');
  await page.getByRole('button', { name: '外观与设置', exact: true }).click();
  await dialog.getByRole('button', { name: '连接与令牌', exact: true }).click();
  await expect(dialog.getByLabel('API 访问令牌', { exact: true })).toHaveValue('example-api-token');
});
