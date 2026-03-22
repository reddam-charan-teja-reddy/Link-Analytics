DROP INDEX IF EXISTS idx_links_category_id;

ALTER TABLE links
DROP COLUMN IF EXISTS category_id;

DROP TABLE IF EXISTS categories;
