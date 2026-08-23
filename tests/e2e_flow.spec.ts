import { test, expect } from '@playwright/test'

test('login and open route book', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByRole('heading', { name: '夜徒出发前' })).toBeVisible()
  await page.getByRole('button', { name: '进入营地' }).click()
  await expect(page.getByRole('heading', { name: '路书' })).toBeVisible()
  await page.getByRole('button', { name: '编辑路书' }).first().click()
  await expect(page.getByText('点击地图落点')).toBeVisible()
})
