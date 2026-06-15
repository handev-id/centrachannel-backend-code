-- Seed: Demo Tenant (local)
INSERT INTO tenants (name, domain, is_active)
VALUES ('Demo Business', 'demo.centrachannel.local', TRUE)
ON CONFLICT (domain) DO NOTHING;
