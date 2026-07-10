CREATE INDEX idx_orders_user_uploaded ON orders(user_id, uploaded_at DESC);
CREATE INDEX idx_withdrawals_user_processed ON withdrawals(user_id, processed_at DESC);