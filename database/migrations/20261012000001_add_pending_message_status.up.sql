ALTER TABLE messages DROP CONSTRAINT chk_messages_status;
ALTER TABLE messages ADD CONSTRAINT chk_messages_status CHECK (status IN ('pending', 'sent', 'delivered', 'read', 'unread', 'failed'));
