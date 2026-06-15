-- Seed: Global Channels
INSERT INTO channels (name, type)
VALUES ('Facebook', 'facebook'),
       ('Instagram', 'instagram'),
       ('WhatsApp Business', 'whatsapp_business'),
       ('WhatsApp', 'whatsapp')
ON CONFLICT (type) DO NOTHING;
