// @watch start
// templates/repo/editor/**
// web_src/js/features/common-global.js
// routers/web/web.go
// services/repository/files/upload.go
// @watch end

// https://codeberg.org/forgejo/forgejo/src/branch/forgejo/tests/e2e/README.md
// lies. Don't use
// make TAGS="sqlite sqlite_unlock_notify" backend
// use
// make TAGS="sqlite sqlite_unlock_notify" build
// otherwise the test fails.


import { expect } from '@playwright/test';
import { test, dynamic_id, save_visual } from './utils_e2e.ts';

test.use({ user: 'user2' });

test('drag and drop upload a', async ({ page }) => {
  const response = await page.goto(`/user2/file-uploads/_upload/main/`);
  expect(response?.status()).toBe(200); // Status OK

  const testID = dynamic_id();
  const dropzone = page.getByRole('button', { name: 'Drop files or click here to upload.' });

  // create the virtual files
  const dataTransferA = await page.evaluateHandle(() => {
    const dt = new DataTransfer();
    // add items in different folders
    dt.items.add(new File(['Filecontent (dir1/file1.txt)'], 'dir1/file1.txt', { type: 'text/plain' }));
    dt.items.add(new File(["Another file's content(double / nested / file.txt)"], 'double / nested / file.txt', { type: 'text / plain' }));
    dt.items.add(new File(['Root file (root_file.txt)'], 'root_file.txt', { type: 'text/plain' }));
    dt.items.add(new File(['Umlaut test'], 'special/äüöÄÜÖß.txt', { type: 'text/plain' }));
    dt.items.add(new File(['Unicode test'], 'special/Ʉ₦ł₵ØĐɆ.txt', { type: 'text/plain' }));
    return dt;
  });
  // and drop them to the upload area
  await dropzone.dispatchEvent('drop', { dataTransfer: dataTransferA });

  await page.getByText('new branch').click();
  await save_visual(page);
  await page.getByRole('textbox', { name: 'Name the new branch for this' }).fill(testID);
  await page.getByRole('button', { name: 'Propose file change' }).click();

  // check that nested file structure is preserved
  await expect(page.getByRole('link', { name: 'dir1/file1.txt' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'double/nested/file.txt' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'special/äüöÄÜÖß.txt' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'special/Ʉ₦ł₵ØĐɆ.txt' })).toBeVisible();
  await expect(page.locator('#diff-file-boxes').getByRole('link', { name: 'root_file.txt', exact: true })).toBeVisible();
  await save_visual(page);
});

test('drag and drop upload b', async ({ page }) => {
  const response = await page.goto(`/user2/file-uploads/_upload/main/`);
  expect(response?.status()).toBe(200); // Status OK

  const testID = dynamic_id();
  const dropzone = page.getByRole('button', { name: 'Drop files or click here to upload.' });

  // create the virtual files
  const dataTransferA = await page.evaluateHandle(() => {
    const dt = new DataTransfer();
    // add items in different folders
    dt.items.add(new File(['1'], '../../dots.txt', { type: 'text/plain' }));
    dt.items.add(new File(['2'], 'special/../../dots_vanish.txt', { type: 'text/plain' }));
    dt.items.add(new File(['3'], '\\windows\\windows_slash.txt', { type: 'text/plain' }));
    dt.items.add(new File(['4'], '/special/badfirstslash.txt', { type: 'text/plain' }));
    dt.items.add(new File(['5'], 'special/S P  A   C   E    !.txt', { type: 'text/plain' }));
    return dt;
  });
  // and drop them to the upload area
  await dropzone.dispatchEvent('drop', { dataTransfer: dataTransferA });

  await page.getByText('new branch').click();
  await save_visual(page);
  await page.getByRole('textbox', { name: 'Name the new branch for this' }).fill(testID);
  await page.getByRole('button', { name: 'Propose file change' }).click();

  // check that nested file structure is preserved
  await expect(page.getByRole('link', { name: 'windows/windows_slash.txt' })).toBeVisible();
  await expect(page.getByRole('link', { name: '_/dots_vanish.txt' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'special/badfirstslash.txt' })).toBeVisible();
  await expect(page.getByRole('link', { name: 'special/S P  A   C   E    !.txt' })).toBeVisible();
  await expect(page.locator('#diff-file-boxes').getByRole('link', { name: '_/_/dots.txt', exact: true })).toBeVisible();
  await save_visual(page);
});