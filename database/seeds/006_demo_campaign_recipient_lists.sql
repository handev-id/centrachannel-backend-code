-- Seed: Sample Campaign Recipient Lists for demo tenant
INSERT INTO campaign_recipient_lists (tenant_id, name, source, channel_id)
SELECT t.id, 'All Contacts', 'manual', c.id
FROM tenants t
CROSS JOIN channels c
WHERE t.domain = 'demo.centrachannel.local'
  AND c.type = 'whatsapp_business'
ON CONFLICT DO NOTHING;
