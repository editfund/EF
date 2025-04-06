// @watch start
// templates/org/team/new.tmpl
// web_src/css/form.css
// web_src/js/features/org-team.js
// @watch end

import {expect} from '@playwright/test';
import {save_visual, test} from './utils_e2e.ts';
import {validate_form} from './shared/forms.ts';

test.use({user: 'user2'});

test('org team settings', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Cannot get it to work - as usual');
  const response = await page.goto('/org/org3/teams/team1/edit');
  expect(response?.status()).toBe(200);

  await page.locator('input[name="permission"][value="admin"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeHidden();
  await save_visual(page);

  await page.locator('input[name="permission"][value="read"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeVisible();
  await save_visual(page);

  // we are validating the form here to include the part that could be hidden
  await validate_form({page});
});
