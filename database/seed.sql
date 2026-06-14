-- Seed: Dummy Tenant
INSERT INTO tenants (name, domain, is_active)
VALUES ('Demo Business', 'demo.centrachannel.com', TRUE)
ON CONFLICT (domain) DO NOTHING;

-- Seed: Default Roles for the dummy tenant
INSERT INTO roles (tenant_id, name)
SELECT id, 'super-admin' FROM tenants WHERE domain = 'demo.centrachannel.com'
UNION ALL
SELECT id, 'admin' FROM tenants WHERE domain = 'demo.centrachannel.com'
UNION ALL
SELECT id, 'agent' FROM tenants WHERE domain = 'demo.centrachannel.com'
ON CONFLICT (tenant_id, name) DO NOTHING;

-- Seed: Global Channels
INSERT INTO channels (name, type)
VALUES ('Facebook', 'facebook'),
       ('Instagram', 'instagram'),
       ('WhatsApp Business', 'whatsapp_business'),
       ('WhatsApp', 'whatsapp')
ON CONFLICT (type) DO NOTHING;
