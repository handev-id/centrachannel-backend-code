UPDATE messages SET status = 'failed' WHERE status = 'pending';
ALTER TABLE messages DROP CONSTRAINT chk_messages_status;
ALTER TABLE messages ADD CONSTRAINT chk_messages_status CHECK (status IN ('sent', 'delivered', 'read', 'unread', 'failed'));
