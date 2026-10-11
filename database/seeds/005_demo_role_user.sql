-- Seed: Assign roles to demo users
INSERT INTO role_user (tenant_id, user_id, role_id)
SELECT u.tenant_id, u.id, r.id
FROM users u
JOIN roles r ON r.tenant_id = u.tenant_id
WHERE u.tenant_id = (SELECT id FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001')
  AND u.email = 'superadmin@demo.centrachannel.local'
  AND r.name = 'super-admin'
UNION ALL
SELECT u.tenant_id, u.id, r.id
FROM users u
JOIN roles r ON r.tenant_id = u.tenant_id
WHERE u.tenant_id = (SELECT id FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001')
  AND u.email = 'admin@demo.centrachannel.local'
  AND r.name = 'admin'
UNION ALL
SELECT u.tenant_id, u.id, r.id
FROM users u
JOIN roles r ON r.tenant_id = u.tenant_id
WHERE u.tenant_id = (SELECT id FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001')
  AND u.email = 'agent@demo.centrachannel.local'
  AND r.name = 'agent'
ON CONFLICT (tenant_id, user_id, role_id) DO NOTHING;
