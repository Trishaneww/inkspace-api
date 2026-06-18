INSERT INTO stripe_events (id, type)
VALUES (@id, @type)
ON CONFLICT (id) DO NOTHING;
