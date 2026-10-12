CREATE INDEX IF NOT EXISTS idx_messages_provider ON messages (tenant_id, conversation_id, webhook_message_id) WHERE webhook_message_id IS NOT NULL;
