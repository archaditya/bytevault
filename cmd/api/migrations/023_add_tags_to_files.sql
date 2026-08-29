-- Add tags column to files table and update the search vector trigger to include tags.

ALTER TABLE files ADD COLUMN IF NOT EXISTS tags TEXT[] DEFAULT '{}';

-- GIN index on tags for direct tag-based filtering queries
CREATE INDEX IF NOT EXISTS idx_files_tags ON files USING GIN (tags);

-- Update the search vector trigger to include tags alongside filename and content_text
CREATE OR REPLACE FUNCTION files_search_vector_update() RETURNS trigger AS $$
BEGIN
  NEW.search_vector := to_tsvector('english',
    coalesce(NEW.filename, '') || ' ' ||
    coalesce(NEW.content_text, '') || ' ' ||
    coalesce(array_to_string(NEW.tags, ' '), '')
  );
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Recreate trigger with the updated function (includes tags in trigger columns)
DROP TRIGGER IF EXISTS files_search_vector_trigger ON files;

CREATE TRIGGER files_search_vector_trigger
  BEFORE INSERT OR UPDATE OF filename, content_text, tags ON files
  FOR EACH ROW
  EXECUTE FUNCTION files_search_vector_update();

-- Backfill search_vector for existing files that have tags
UPDATE files SET search_vector = to_tsvector('english',
  coalesce(filename, '') || ' ' ||
  coalesce(content_text, '') || ' ' ||
  coalesce(array_to_string(tags, ' '), '')
)
WHERE tags != '{}' OR content_text IS NOT NULL;

---- create above / drop below ----

-- Revert trigger to previous version (without tags)
DROP TRIGGER IF EXISTS files_search_vector_trigger ON files;

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

DROP INDEX IF EXISTS idx_files_tags;
ALTER TABLE files DROP COLUMN IF EXISTS tags;
