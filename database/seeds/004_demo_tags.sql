-- Seed: Default Tags for demo tenant
INSERT INTO tags (tenant_id, name, color)
SELECT id, 'VIP', '#FFD700' FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'New Lead', '#3498DB' FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'Follow Up', '#E67E22' FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'Urgent', '#E74C3C' FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'Support', '#2ECC71' FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'Feedback', '#9B59B6' FROM tenants WHERE domain = 'demo.centrachannel.local'
ON CONFLICT (tenant_id, name) DO NOTHING;
