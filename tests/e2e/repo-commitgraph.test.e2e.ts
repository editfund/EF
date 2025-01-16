// @watch start
// templates/repo/graph.tmpl
// web_src/css/features/gitgraph.css
// web_src/js/features/repo-graph.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test('Commit graph overflow', async ({page}) => {
  await page.goto('/user2/diff-test/graph');
  await expect(page.getByRole('button', {name: 'Mono'})).toBeInViewport({ratio: 1});
  await expect(page.getByRole('button', {name: 'Color'})).toBeInViewport({ratio: 1});
  await expect(page.locator('.selection.search.dropdown')).toBeInViewport({ratio: 1});
});

test('Switch branch', async ({page}) => {
  const response = await page.goto('/user2/repo1/graph');
  expect(response?.status()).toBe(200);

  await page.click('#flow-select-refs-dropdown');
  const input = page.locator('#flow-select-refs-dropdown');
  await input.pressSequentially('develop', {delay: 50});
  await input.press('Enter');

  await page.waitForLoadState();

  await expect(page.locator('#loading-indicator')).toBeHidden();
  await expect(page.locator('#rel-container')).toBeVisible();
  await expect(page.locator('#rev-container')).toBeVisible();
  await save_visual(page);
});
