import { expect, test } from '@playwright/test';

test('admin can manage categories and subcategories from the UI', async ({ page }) => {
  const suffix = Date.now().toString().slice(-6);
  const categoryName = `Codex Category ${suffix}`;
  const updatedCategoryName = `${categoryName} Updated`;
  const subcategoryName = `Codex Subcategory ${suffix}`;
  const updatedSubcategoryName = `${subcategoryName} Updated`;

  await page.goto('/login');

  await page.getByLabel('Email').fill('codex-admin-1943@example.com');
  await page.getByLabel('Password').fill('Passw0rd!123');
  await page.getByRole('button', { name: /continue/i }).click();

  await page.waitForURL('**/dashboard');
  await page.getByRole('link', { name: 'Categories' }).click();
  await page.waitForURL('**/admin/categories');

  await expect(page.getByRole('heading', { name: 'Category Management' })).toBeVisible();

  await page.getByRole('button', { name: 'New Category' }).click();
  await page.getByLabel('Category Name').fill(categoryName);
  await page.getByRole('button', { name: 'Create Category' }).click();

  const categoryRow = page.getByRole('row').filter({ hasText: categoryName });
  await expect(categoryRow).toBeVisible();
  await categoryRow.getByText(categoryName).click();

  await page.getByRole('button', { name: 'New Subcategory' }).click();
  await page.getByLabel('Subcategory Name').fill(subcategoryName);
  await page.getByRole('button', { name: 'Create Subcategory' }).click();

  const subcategoryRow = page.getByRole('row').filter({ hasText: subcategoryName });
  await expect(subcategoryRow).toBeVisible();

  await categoryRow.getByLabel(`Edit category ${categoryName}`).click();
  await page.getByLabel('Category Name').fill(updatedCategoryName);
  await page.getByRole('button', { name: 'Save Changes' }).click();

  const updatedCategoryRow = page.getByRole('row').filter({ hasText: updatedCategoryName });
  await expect(updatedCategoryRow).toBeVisible();
  await updatedCategoryRow.getByText(updatedCategoryName).click();

  await subcategoryRow.getByLabel(`Edit subcategory ${subcategoryName}`).click();
  await page.getByLabel('Subcategory Name').fill(updatedSubcategoryName);
  await page.getByRole('button', { name: 'Save Changes' }).click();

  const updatedSubcategoryRow = page.getByRole('row').filter({ hasText: updatedSubcategoryName });
  await expect(updatedSubcategoryRow).toBeVisible();

  await updatedSubcategoryRow.getByLabel(`Delete subcategory ${updatedSubcategoryName}`).click();
  await page.getByRole('button', { name: /^Delete$/ }).click();
  await expect(page.getByRole('row').filter({ hasText: updatedSubcategoryName })).toHaveCount(0);

  await updatedCategoryRow.getByLabel(`Delete category ${updatedCategoryName}`).click();
  await page.getByRole('button', { name: /^Delete$/ }).click();
  await expect(page.getByRole('row').filter({ hasText: updatedCategoryName })).toHaveCount(0);
});
