-- Seed: Demo Tenant and its default local domain. Fixed ID keeps repeated seeds idempotent.
INSERT INTO tenants (id, name, is_active)
VALUES ('00000000-0000-4000-8000-000000000001', 'Demo Business', TRUE)
ON CONFLICT (id) DO NOTHING;

INSERT INTO domains (tenant_id, name, is_default, is_active)
SELECT '00000000-0000-4000-8000-000000000001', 'demo.centrachannel.local', TRUE, TRUE
WHERE NOT EXISTS (
    SELECT 1 FROM domains WHERE tenant_id = '00000000-0000-4000-8000-000000000001'
      AND name = 'demo.centrachannel.local'
);
