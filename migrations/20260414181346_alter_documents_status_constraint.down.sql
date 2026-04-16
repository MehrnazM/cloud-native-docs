-- Drop the new constraint
ALTER TABLE documents.documents DROP CONSTRAINT documents_status_check;

-- Restore the old constraint
ALTER TABLE documents.documents ADD CONSTRAINT documents_status_check 
    CHECK (status IN ('draft', 'published', 'archived'));
