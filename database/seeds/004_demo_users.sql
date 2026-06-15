-- Seed: Demo Users for demo tenant
-- Password for all users: demo1234
INSERT INTO users (tenant_id, first_name, last_name, username, email, password)
SELECT id, 'Super', 'Admin', 'superadmin', 'superadmin@demo.centrachannel.local', '$2a$10$Hv45itlXfYsEpLLJJKETn.h3sy2GfblADwBUGRtrNqOuWBaAxlcZW'
FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'Admin', 'User', 'admin', 'admin@demo.centrachannel.local', '$2a$10$Hv45itlXfYsEpLLJJKETn.h3sy2GfblADwBUGRtrNqOuWBaAxlcZW'
FROM tenants WHERE domain = 'demo.centrachannel.local'
UNION ALL
SELECT id, 'Agent', 'User', 'agent', 'agent@demo.centrachannel.local', '$2a$10$Hv45itlXfYsEpLLJJKETn.h3sy2GfblADwBUGRtrNqOuWBaAxlcZW'
FROM tenants WHERE domain = 'demo.centrachannel.local'
ON CONFLICT (tenant_id, email) DO NOTHING;
