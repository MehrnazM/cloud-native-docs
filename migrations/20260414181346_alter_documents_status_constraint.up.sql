-- Drop the old constraint (PostgreSQL auto-names it as tablename_columnname_check)
ALTER TABLE documents.documents DROP CONSTRAINT documents_status_check;

-- Add the new constraint with updated values
ALTER TABLE documents.documents ADD CONSTRAINT documents_status_check 
    CHECK (status IN ('pending', 'processing', 'done', 'failed'));
