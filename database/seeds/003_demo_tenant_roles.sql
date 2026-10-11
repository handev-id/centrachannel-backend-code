-- Seed: Default Roles for demo tenant
INSERT INTO roles (tenant_id, name)
SELECT id, 'super-admin' FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001'
UNION ALL
SELECT id, 'admin' FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001'
UNION ALL
SELECT id, 'agent' FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001'
ON CONFLICT (tenant_id, name) DO NOTHING;
