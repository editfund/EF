// @watch start
// web_src/css/markup/**
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';

test('markup with #xyz-mode-only', async ({page}) => {
  const response = await page.goto('/user2/repo1/issues/1');
  expect(response?.status()).toBe(200);

  const comment = page.locator('.comment-body>.markup', {hasText: 'test markup light/dark-mode-only'});
  await expect(comment).toBeVisible();
  await expect(comment.locator('[src$="#gh-light-mode-only"]')).toBeVisible();
  await expect(comment.locator('[src$="#gh-dark-mode-only"]')).toBeHidden();
  await save_visual(page);
});
