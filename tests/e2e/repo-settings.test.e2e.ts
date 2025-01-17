// @watch start
// templates/webhook/shared-settings.tmpl
// templates/repo/settings/**
// web_src/css/{form,repo}.css
// web_src/css/modules/grid.css
// web_src/js/features/comp/WebHookEditor.js
// @watch end

import {expect} from '@playwright/test';
import {test, save_visual} from './utils_e2e.ts';
import {validate_form} from './shared/forms.ts';

test.use({user: 'user2'});

test('repo webhook settings', async ({page}, workerInfo) => {
  test.skip(workerInfo.project.name === 'Mobile Safari', 'Cannot get it to work - as usual');
  const response = await page.goto('/user2/repo1/settings/hooks/forgejo/new');
  expect(response?.status()).toBe(200);

  await page.locator('input[name="events"][value="choose_events"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeVisible();

  // check accessibility including the custom events (now visible) part
  await validate_form({page}, 'fieldset');
  await save_visual(page);

  await page.locator('input[name="events"][value="push_only"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeHidden();
  await page.locator('input[name="events"][value="send_everything"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeHidden();
  await save_visual(page);
});

test.describe('repo branch protection settings', () => {
  test('form', async ({page}, {project}) => {
    test.skip(project.name === 'Mobile Safari', 'Cannot get it to work - as usual');
    const response = await page.goto('/user2/repo1/settings/branches/edit');
    expect(response?.status()).toBe(200);

    await validate_form({page}, 'fieldset');

    // verify header is new
    await expect(page.locator('h4')).toContainText('new');
    await page.locator('input[name="rule_name"]').fill('testrule');
    await save_visual(page);
    await page.getByText('Save rule').click();
    // verify header is in edit mode
    await page.waitForLoadState('domcontentloaded');
    await save_visual(page);
    await page.getByText('Edit').click();
    await expect(page.locator('h4')).toContainText('Protection rules for branch');
    await save_visual(page);
  });

  test.afterEach(async ({page}) => {
    // delete the rule for the next test
    await page.goto('/user2/repo1/settings/branches/');
    await page.waitForLoadState('domcontentloaded');
    test.skip(await page.getByText('Delete rule').isHidden(), 'Nothing to delete at this time');
    await page.getByText('Delete rule').click();
    await page.getByText('Yes').click();
    await page.waitForLoadState('load');
  });
});

test('repo settings - units - pulls', async ({page}/*, workerInfo*/) => {
  //test.skip(workerInfo.project.name === 'Mobile Safari', 'Cannot get it to work - as usual');
  const response = await page.goto('/user2/repo1/settings/units#pulls');
  expect(response?.status()).toBe(200);

  await validate_form({page}, 'fieldset');
  await save_visual(page);

  await page.locator('input[name="enable_pulls"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeHidden();
  await page.locator('input[name="enable_pulls"]').click();
  await expect(page.locator('.hide-unless-checked')).toBeHidden();
  await save_visual(page);
});
