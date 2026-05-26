-- Drop marketplace-era tables that backed the bookings, matching,
-- messaging, payments, and notifications modules. Those modules have
-- been removed from the API as the project narrows scope to the artist
-- dashboard + open-booking-link flow. Tables are dropped in reverse
-- dependency order so foreign-key constraints don't block the drops.

DROP TABLE IF EXISTS payment_webhook_events;
DROP TABLE IF EXISTS payments;

DROP TABLE IF EXISTS conversation_read_cursors;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS conversations;

DROP TABLE IF EXISTS matches;
DROP TABLE IF EXISTS match_requests;

DROP TABLE IF EXISTS booking_status_history;
DROP TABLE IF EXISTS bookings;

DROP TABLE IF EXISTS notification_deliveries;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS notifications;
