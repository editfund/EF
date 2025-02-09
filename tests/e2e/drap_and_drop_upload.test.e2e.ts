// @watch start
// templates/user/auth/**
// web_src/js/features/user-**
// modules/{user,auth}/**
// @watch end

/// <reference lib="dom"/>

import { expect } from '@playwright/test';
import { test } from './utils_e2e.ts';

test('drap and drop upload test', async ({ page }, workerInfo) => {
  page.setDefaultTimeout(0);
  // Create user
  const response = await page.goto('/user/sign_up');
  expect(response?.status()).toBe(200); // Status OK
  await page.fill('input[name=user_name]', `e2e-test-${workerInfo.workerIndex}`);
  await page.fill('input[name=email]', `e2e-test-${workerInfo.workerIndex}@test.com`);
  await page.fill('input[name=password]', 'test123test123');
  await page.fill('input[name=retype]', 'test123test123');
  await page.click('form button.ui.primary.button:visible');

  // It looks like that I am automatically logged in if I create the account
  // // Log-in into account
  // const response2 = await page.goto('/user/login');
  // expect(response2?.status()).toBe(200); // Status OK
  // await page.fill('input[name=user_name]', `e2e-test-${workerInfo.workerIndex}`);
  // await page.fill('input[name=password]', 'test123test123');
  // await page.getByRole('button', { name: 'Sign in' }).click();

  // Create a new repos
  const response3 = await page.goto('/repo/create');
  expect(response3?.status()).toBe(200); // Status OK
  await page.fill('input[name=repo_name]', 'e2e-test');
  await page.getByRole('button', { name: 'Create repository' }).click();

  // Go into the repo
  const response4 = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/_upload/master`);
  expect(response4?.status()).toBe(200); // Status OK

  // Find the drop zone
  const dropzone = page.getByRole('button', { name: 'Drop files or click here to upload.' });

  // Create file1.txt in the current directory
  const buffer = Buffer.from('Test File 1', 'utf-8');

  // Create the DataTransfer and File
  const dataTransfer = await page.evaluateHandle((data) => {
    const dt = new DataTransfer();

    const file = new File([data], 'dir2/file_2.txt', { type: 'text/plain' });

    dt.items.add(file);
    return dt;
  }, buffer);

  // Drop the file
  await dropzone.dispatchEvent('drop', { dataTransfer });

  // Commit the file
  await page.getByRole('button', { name: 'Commit changes' }).click();

  // Go into the repo
  const response6 = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2`);
  expect(response6?.status()).toBe(200); // Status OK

  await expect(page.getByRole('link', { name: 'file_2.txt' })).toBeVisible();
});
