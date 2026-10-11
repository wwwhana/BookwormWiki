-- +goose Up
-- The project was renamed from Stash to BookwormWiki. Work resources that
-- the server itself owns carried the authority (and default source) "stash";
-- they now carry "bookwormwiki". The server still accepts "stash" from older
-- clients and stores it as "bookwormwiki".
ALTER TABLE work_resources DROP CONSTRAINT IF EXISTS work_resources_authority_check;
UPDATE work_resources SET authority = 'bookwormwiki' WHERE authority = 'stash';
UPDATE work_resources SET source = 'bookwormwiki' WHERE source = 'stash';
ALTER TABLE work_resources ALTER COLUMN authority SET DEFAULT 'bookwormwiki';
ALTER TABLE work_resources ALTER COLUMN source SET DEFAULT 'bookwormwiki';
ALTER TABLE work_resources
    ADD CONSTRAINT work_resources_authority_check CHECK (authority IN ('bookwormwiki', 'external'));

-- +goose Down
ALTER TABLE work_resources DROP CONSTRAINT IF EXISTS work_resources_authority_check;
UPDATE work_resources SET authority = 'stash' WHERE authority = 'bookwormwiki';
UPDATE work_resources SET source = 'stash' WHERE source = 'bookwormwiki';
ALTER TABLE work_resources ALTER COLUMN authority SET DEFAULT 'stash';
ALTER TABLE work_resources ALTER COLUMN source SET DEFAULT 'stash';
ALTER TABLE work_resources
    ADD CONSTRAINT work_resources_authority_check CHECK (authority IN ('stash', 'external'));
