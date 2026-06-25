-- Roles table (scalable — permissions stored as JSONB)
CREATE TABLE IF NOT EXISTS roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    permissions JSONB NOT NULL DEFAULT '{}',
    is_system_role BOOLEAN NOT NULL DEFAULT false,
    created_by UUID,
    updated_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- User <-> Role mapping (many-to-many)
-- A user can have multiple roles, each role has its own permissions
CREATE TABLE IF NOT EXISTS user_roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, role_id)
);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);

-- User devices for Firebase push notifications
CREATE TABLE IF NOT EXISTS user_devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    fcm_token TEXT NOT NULL,
    device_type VARCHAR(20) NOT NULL,
    device_id VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_user_devices_user_id ON user_devices(user_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_devices_fcm_token ON user_devices(fcm_token);

-- Activity logs — tracks every important action in the system
CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id UUID,
    metadata JSONB,
    ip_address VARCHAR(45),
    user_agent TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_activity_logs_user_id ON activity_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_activity_logs_action ON activity_logs(action);
CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at ON activity_logs(created_at);

-- Seed default roles with permissions
INSERT INTO roles (name, description, permissions, is_system_role) VALUES
('user', 'Default user role', '{
    "user:read": true, "user:write": true, "user:delete": false,
    "file:read": true, "file:write": true, "file:delete": true,
    "admin:users": false, "admin:activity": false, "admin:sessions": false, "admin:roles": false
}', true),
('super_admin', 'Full system access', '{
    "user:read": true, "user:write": true, "user:delete": true,
    "file:read": true, "file:write": true, "file:delete": true,
    "admin:users": true, "admin:activity": true, "admin:sessions": true, "admin:roles": true
}', true)
ON CONFLICT (name) DO NOTHING;

---- create above / drop below ----

DROP TABLE IF EXISTS activity_logs;
DROP TABLE IF EXISTS user_devices;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS roles;