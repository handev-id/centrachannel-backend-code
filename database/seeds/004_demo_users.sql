-- Seed: Demo Users for demo tenant
-- Password for all users: demo1234
INSERT INTO users (tenant_id, first_name, last_name, username, email, password)
SELECT id, 'Super', 'Admin', 'superadmin', 'superadmin@demo.centrachannel.local', '$2a$10$Hv45itlXfYsEpLLJJKETn.h3sy2GfblADwBUGRtrNqOuWBaAxlcZW'
FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001'
UNION ALL
SELECT id, 'Admin', 'User', 'admin', 'admin@demo.centrachannel.local', '$2a$10$Hv45itlXfYsEpLLJJKETn.h3sy2GfblADwBUGRtrNqOuWBaAxlcZW'
FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001'
UNION ALL
SELECT id, 'Agent', 'User', 'agent', 'agent@demo.centrachannel.local', '$2a$10$Hv45itlXfYsEpLLJJKETn.h3sy2GfblADwBUGRtrNqOuWBaAxlcZW'
FROM tenants WHERE id = '00000000-0000-4000-8000-000000000001'
ON CONFLICT (tenant_id, email) DO NOTHING;
