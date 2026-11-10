CREATE TABLE IF NOT EXISTS abuse_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_number VARCHAR(32) NOT NULL UNIQUE,
    file_id UUID REFERENCES files(id) ON DELETE SET NULL,
    target_url TEXT NOT NULL,
    category VARCHAR(64) NOT NULL,
    reporter_name TEXT NOT NULL,
    reporter_email TEXT NOT NULL,
    description TEXT NOT NULL,
    evidence_url TEXT,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    admin_notes TEXT,
    reviewed_by UUID REFERENCES users(id) ON DELETE SET NULL,
    reviewed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_abuse_reports_status ON abuse_reports(status);
CREATE INDEX IF NOT EXISTS idx_abuse_reports_file_id ON abuse_reports(file_id);
CREATE INDEX IF NOT EXISTS idx_abuse_reports_ticket ON abuse_reports(ticket_number);
CREATE INDEX IF NOT EXISTS idx_abuse_reports_created ON abuse_reports(created_at DESC);

---- create above / drop below ----

DROP INDEX IF EXISTS idx_abuse_reports_created;
DROP INDEX IF EXISTS idx_abuse_reports_ticket;
DROP INDEX IF EXISTS idx_abuse_reports_file_id;
DROP INDEX IF EXISTS idx_abuse_reports_status;
DROP TABLE IF EXISTS abuse_reports;
