DELETE FROM conversations c
USING conversations keep
WHERE c.artist_id = keep.artist_id
  AND c.client_email = keep.client_email
  AND (
    c.created_at > keep.created_at
    OR (c.created_at = keep.created_at AND c.id > keep.id)
  );

CREATE UNIQUE INDEX idx_conversations_artist_client_email
    ON conversations (artist_id, client_email);
