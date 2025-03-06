// @watch start
// templates/repo/editor/**
// web_src/js/features/common-global.js
// routers/web/web.go
// services/repository/files/upload.go
// @watch end

/// <reference lib="dom"/>

import {expect} from '@playwright/test';
import {test, dynamic_id, save_visual} from './utils_e2e.ts';

test.use({user: 'user2'});

test('drap and drop upload', async ({page}) => {
  const response = await page.goto(`/user2/file-uploads/_upload/main/`);
  expect(response?.status()).toBe(200); // Status OK

  const testID = dynamic_id();
  const dropzone = page.getByRole('button', {name: 'Drop files or click here to upload.'});

  // create the virtual files
  const dataTransfer = await page.evaluateHandle(() => {
    const dt = new DataTransfer();

    // add items in different folders
    dt.items.add(new File(['Filecontent (dir1/file1.txt)'], 'dir1/file1.txt', {type: 'text/plain'}));
    dt.items.add(new File(["Another file's content (double/nested/file.txt)"], 'double/nested/file.txt', {type: 'text/plain'}));
    dt.items.add(new File(['Root file (root_file.txt)'], 'root_file.txt', {type: 'text/plain'}));

    return dt;
  });
  // and drop them to the upload area
  await dropzone.dispatchEvent('drop', {dataTransfer});

  await page.getByText('new branch').click();
  await save_visual(page);
  await page.getByRole('textbox', {name: 'Name the new branch for this'}).fill(testID);
  await page.getByRole('button', {name: 'Propose file change'}).click();

  // check that nested file structure is preserved
  await expect(page.getByRole('link', {name: 'dir1/file1.txt'})).toBeVisible();
  await expect(page.getByRole('link', {name: 'double/nested/file.txt'})).toBeVisible();
  await expect(page.locator('#diff-file-boxes').getByRole('link', {name: 'root_file.txt', exact: true})).toBeVisible();
  await save_visual(page);
});
