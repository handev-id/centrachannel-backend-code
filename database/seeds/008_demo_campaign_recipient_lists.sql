-- Seed: Sample Campaign Recipient Lists for demo tenant
INSERT INTO campaign_recipient_lists (tenant_id, name, source, channel_id)
SELECT t.id, 'All Contacts', 'manual', c.id
FROM tenants t
CROSS JOIN channels c
WHERE t.id = '00000000-0000-4000-8000-000000000001'
  AND c.type = 'whatsapp_business'
ON CONFLICT DO NOTHING;
