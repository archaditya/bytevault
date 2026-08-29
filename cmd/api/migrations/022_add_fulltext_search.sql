-- Add full-text search support to files table.
-- Stores extracted document text and a tsvector for fast full-text search.

ALTER TABLE files ADD COLUMN IF NOT EXISTS content_text TEXT;
ALTER TABLE files ADD COLUMN IF NOT EXISTS search_vector TSVECTOR;

-- GIN index for fast full-text queries
CREATE INDEX IF NOT EXISTS idx_files_search_vector ON files USING GIN (search_vector);

-- Trigger function: auto-generates search_vector from filename + content_text
CREATE OR REPLACE FUNCTION files_search_vector_update() RETURNS trigger AS $$
BEGIN
  NEW.search_vector := to_tsvector('english',
    coalesce(NEW.filename, '') || ' ' || coalesce(NEW.content_text, '')
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER files_search_vector_trigger
  BEFORE INSERT OR UPDATE OF filename, content_text ON files
  FOR EACH ROW
  EXECUTE FUNCTION files_search_vector_update();

-- Backfill search_vector for existing files (filename only, content_text will be populated by workers)
UPDATE files SET search_vector = to_tsvector('english', coalesce(filename, ''))
WHERE search_vector IS NULL;

---- create above / drop below ----

DROP TRIGGER IF EXISTS files_search_vector_trigger ON files;
DROP FUNCTION IF EXISTS files_search_vector_update();
DROP INDEX IF EXISTS idx_files_search_vector;
ALTER TABLE files DROP COLUMN IF EXISTS search_vector;
ALTER TABLE files DROP COLUMN IF EXISTS content_text;
