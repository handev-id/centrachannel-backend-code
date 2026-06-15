-- Seed: Sample Campaign Templates for demo tenant
INSERT INTO campaign_templates (tenant_id, name, type, template_type, category, language, content, variables)
SELECT
  id,
  'Welcome Message',
  'broadcast',
  'text',
  'Greeting',
  'id',
  '{"body": "Hello {{1}}, welcome to our service! We are excited to have you on board."}'::jsonb,
  '[{"key": "1", "label": "Customer Name"}]'::jsonb
FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT
  id,
  'Monthly Promotion',
  'broadcast',
  'text',
  'Promotion',
  'id',
  '{"body": "Hi {{1}}, check out our monthly special offer! Visit {{2}} for more details."}'::jsonb,
  '[{"key": "1", "label": "Customer Name"}, {"key": "2", "label": "Promo Link"}]'::jsonb
FROM tenants WHERE domain = 'demo.centrachannel.local'
ON CONFLICT DO NOTHING;
