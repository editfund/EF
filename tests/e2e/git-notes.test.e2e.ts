// @ts-check
import {test, expect} from '@playwright/test';
import {login_user, load_logged_in_context} from './utils_e2e.ts';

test.beforeAll(async ({browser}, workerInfo) => {
  await login_user(browser, workerInfo, 'user2');
});

test('Change git note', async ({browser}, workerInfo) => {
  const context = await load_logged_in_context(browser, workerInfo, 'user2');
  const page = await context.newPage();
  let response = await page.goto('/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d');
  expect(response?.status()).toBe(200);

  await page.locator('#commit-notes-edit-button').click();

  let textarea = page.locator('textarea[name="notes"]');
  await expect(textarea).toBeVisible();
  await textarea.fill('This is a new note');

  await page.locator('#notes-save-button').click();

  expect(response?.status()).toBe(200);

  response = await page.goto('/user2/repo1/commit/65f1bf27bc3bf70f64657658635e66094edbcb4d');
  expect(response?.status()).toBe(200);

  textarea = page.locator('textarea[name="notes"]');
  await expect(textarea).toHaveText('This is a new note');
});
