-- Seed: Assign roles to demo users
INSERT INTO role_user (user_id, role_id)
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.tenant_id = u.tenant_id
WHERE u.tenant_id = (SELECT id FROM tenants WHERE domain = 'demo.centrachannel.local')
  AND u.email = 'superadmin@demo.centrachannel.local'
  AND r.name = 'super-admin'
UNION ALL
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.tenant_id = u.tenant_id
WHERE u.tenant_id = (SELECT id FROM tenants WHERE domain = 'demo.centrachannel.local')
  AND u.email = 'admin@demo.centrachannel.local'
  AND r.name = 'admin'
UNION ALL
SELECT u.id, r.id
FROM users u
JOIN roles r ON r.tenant_id = u.tenant_id
WHERE u.tenant_id = (SELECT id FROM tenants WHERE domain = 'demo.centrachannel.local')
  AND u.email = 'agent@demo.centrachannel.local'
  AND r.name = 'agent'
ON CONFLICT DO NOTHING;
