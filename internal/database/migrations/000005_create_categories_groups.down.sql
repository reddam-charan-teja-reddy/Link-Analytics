DROP TABLE IF EXISTS link_groups;
DROP TABLE IF EXISTS groups;

DROP INDEX IF EXISTS idx_links_category_id;
ALTER TABLE links DROP COLUMN IF EXISTS category_id;

DROP TABLE IF EXISTS categories;