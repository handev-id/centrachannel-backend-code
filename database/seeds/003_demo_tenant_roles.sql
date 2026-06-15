-- Seed: Default Roles for demo tenant
INSERT INTO roles (tenant_id, name)
SELECT id, 'super-admin' FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'admin' FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'agent' FROM tenants WHERE domain = 'demo.centrachannel.local'
ON CONFLICT (tenant_id, name) DO NOTHING;
