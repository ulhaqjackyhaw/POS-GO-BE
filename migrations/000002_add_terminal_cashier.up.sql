-- Add optional assigned_cashier_id to terminals to associate a dedicated cashier with a POS terminal
ALTER TABLE terminals 
ADD COLUMN IF NOT EXISTS assigned_cashier_id UUID REFERENCES users(id) ON DELETE SET NULL;
