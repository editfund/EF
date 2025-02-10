// @watch start
// templates/user/auth/**
// web_src/js/features/user-**
// modules/{user,auth}/**
// @watch end

import { expect } from '@playwright/test';
import { test } from './utils_e2e.ts';

test('delete directory test', async ({ page }, workerInfo) => {
  page.setDefaultTimeout(0);

  // Create user
  await expect(async () => {
    const response = await page.goto('/user/sign_up');
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    await page.fill('input[name=user_name]', `e2e-test-${workerInfo.workerIndex}`);
    await page.fill('input[name=email]', `e2e-test-${workerInfo.workerIndex}@test.com`);
    await page.fill('input[name=password]', 'test123test123');
    await page.fill('input[name=retype]', 'test123test123');
    await page.click('form button.ui.primary.button:visible');
  }).toPass();

  // Remove later because after creating a new account it is logged in
  // // Log-in into account
  // await expect(async () => {
  //   const response = await page.goto('/user/login');
  //   expect(response?.status()).toBe(200);
  // }).toPass();

  // await expect(async () => {
  //   await page.fill('input[name=user_name]', `e2e-test-${workerInfo.workerIndex}`);
  //   await page.fill('input[name=password]', 'test123test123');
  //   await page.getByRole('button', { name: 'Sign in' }).click();
  // }).toPass();

  // Create a new repos
  await expect(async () => {
    const response = await page.goto('/repo/create');
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    await page.fill('input[name=repo_name]', 'e2e-test');
    await page.getByRole('button', { name: 'Create repository' }).click();
  }).toPass();

  // Go into the repo
  // Create file1.txt
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/_new/master`);
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(async () => {
    await page.getByPlaceholder('Name your file…').click();
    await page.getByPlaceholder('Name your file…').fill('file1.txt');
    await page.locator('.view-lines').click();
    await page.getByLabel('Editor content').fill('Test 1');
    await page.getByRole('button', { name: 'Commit changes' }).click();
  }).toPass();
  // Is file there?
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/file1.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  // Create dir2/file2.txt
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/_new/master`);
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(async () => {
    await page.getByPlaceholder('Name your file…').click();
    await page.getByPlaceholder('Name your file…').fill('dir2/file2.txt');
    await page.locator('.view-lines').click();
    await page.getByLabel('Editor content').fill('Test 2');
    await page.getByRole('button', { name: 'Commit changes' }).click();
  }).toPass();
  // Is file there?
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/file2.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();
  // Create dir2/dir3/file3.txt
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/_new/master`);
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(async () => {
    await page.getByPlaceholder('Name your file…').click();
    await page.getByPlaceholder('Name your file…').fill('dir2/dir3/file3.txt');
    await page.locator('.view-lines').click();
    await page.getByLabel('Editor content').fill('Test 3');
    await page.getByRole('button', { name: 'Commit changes' }).click();
  }).toPass();
  // Is file there?
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  // Create dir2/dir3/file3b.txt
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/_new/master`);
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(async () => {
    await page.getByPlaceholder('Name your file…').click();
    await page.getByPlaceholder('Name your file…').fill('dir2/dir3/file3b.txt');
    await page.locator('.view-lines').click();
    await page.getByLabel('Editor content').fill('Test 3');
    await page.getByRole('button', { name: 'Commit changes' }).click();
  }).toPass();
  // Is file there?
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3b.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(page.getByRole('link', { name: 'Delete file' })).toBeVisible();
  // Now we are ready for the test

  // Logout
  await expect(async () => {
    await page.locator('div[aria-label="Profile and settings…"]').click();
    await page.getByText('Sign Out').click();
  }).toPass();

  await expect(async () => {
    const response = await page.goto('/');
    expect(response?.status()).toBe(200);
  }).toPass();

  // Delete shouldn't be able for a logged out person
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/`);
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(page.getByRole('link', { name: 'Delete file' })).toBeHidden();

  // Create a different user
  await expect(async () => {
    const response = await page.goto('/user/sign_up');
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(async () => {
    await page.fill('input[name=user_name]', `oe2e-test-${workerInfo.workerIndex}`);
    await page.fill('input[name=email]', `oe2e-test-${workerInfo.workerIndex}@test.com`);
    await page.fill('input[name=password]', 'test123test123');
    await page.fill('input[name=retype]', 'test123test123');
    await page.click('form button.ui.primary.button:visible');
  }).toPass();
  // Remove later because after creating a new account it is logged in
  // // Log-in into account
  // await expect(async () => {
  //   const response = await page.goto('/user/login');
  //   expect(response?.status()).toBe(200);
  // }).toPass();
  // await expect(async () => {
  //   await page.fill('input[name=user_name]', `oe2e-test-${workerInfo.workerIndex}`);
  //   await page.fill('input[name=password]', 'test123test123');
  //   await page.getByRole('button', { name: 'Sign in' }).click();
  // }).toPass();

  // Delete shouldn't be able for a different person
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/`);
    expect(response?.status()).toBe(200);
  }).toPass();
  await expect(page.getByRole('link', { name: 'Delete file' })).toBeHidden();

  // Logout 
  await expect(async () => {
    await page.locator('div[aria-label="Profile and settings…"]').click();
    await page.getByText('Sign Out').click();
  }).toPass();

  await expect(async () => {
    const response = await page.goto('/');
    expect(response?.status()).toBe(200);
  }).toPass();

  // Log-in of the correct account
  await expect(async () => {
    const response = await page.goto('/user/login');
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    await page.fill('input[name=user_name]', `e2e-test-${workerInfo.workerIndex}`);
    await page.fill('input[name=password]', 'test123test123');
    await page.getByRole('button', { name: 'Sign in' }).click();
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(page.getByRole('link', { name: 'Delete file' })).toBeVisible();
  // Security test done...

  // Still all files there?
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/file1.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/file2.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3b.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  // remove dir3 and content
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(page.getByRole('link', { name: 'Delete file' })).toBeVisible();

  await expect(async () => {
    await page.getByRole('link', { name: 'Delete file' }).click();
    await page.getByRole('button', { name: 'Commit changes' }).click();
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/file1.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/file2.txt`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3.txt`);
    expect(response?.status()).toBe(404);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3b.txt`);
    expect(response?.status()).toBe(404);
  }).toPass();

  // remove root and content
  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/`);
    expect(response?.status()).toBe(200);
  }).toPass();

  await expect(page.getByRole('link', { name: 'Delete file' })).toBeVisible();

  await expect(async () => {
    await page.getByRole('link', { name: 'Delete file' }).click();
    await page.getByRole('button', { name: 'Commit changes' }).click();
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/file1.txt`);
    expect(response?.status()).toBe(404);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/file2.txt`);
    expect(response?.status()).toBe(404);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3.txt`);
    expect(response?.status()).toBe(404);
  }).toPass();

  await expect(async () => {
    const response = await page.goto(`/e2e-test-${workerInfo.workerIndex}/e2e-test/src/branch/master/dir2/dir3/file3b.txt`);
    expect(response?.status()).toBe(404);
  }).toPass();

});
