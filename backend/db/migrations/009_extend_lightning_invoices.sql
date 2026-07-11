ALTER TABLE lightning_invoices ADD COLUMN status VARCHAR(32) NOT NULL DEFAULT 'pending';
ALTER TABLE lightning_invoices ADD COLUMN created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP;
ALTER TABLE lightning_invoices ADD COLUMN expires_at TIMESTAMP;
ALTER TABLE lightning_invoices ADD COLUMN add_index INTEGER NOT NULL DEFAULT 0;
ALTER TABLE lightning_invoices ADD COLUMN settle_index INTEGER NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_lightning_invoices_status ON lightning_invoices(status);
CREATE INDEX IF NOT EXISTS idx_lightning_invoices_settle_index ON lightning_invoices(settle_index);
