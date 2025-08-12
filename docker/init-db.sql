-- Initialize WhatsApp Multi-Device Database
-- This script creates the necessary databases and extensions

-- Create the keys database
CREATE DATABASE whatsapp_keys;

-- Grant permissions to the user
GRANT ALL PRIVILEGES ON DATABASE whatsapp_main TO whatsapp_user;
GRANT ALL PRIVILEGES ON DATABASE whatsapp_keys TO whatsapp_user;

-- Connect to main database and create extensions
\c whatsapp_main;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Connect to keys database and create extensions
\c whatsapp_keys;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";

-- Create initial tables for multi-instance management
\c whatsapp_main;

-- Instance management table
CREATE TABLE IF NOT EXISTS instance_registry (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    instance_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    phone VARCHAR(50),
    status VARCHAR(50) DEFAULT 'stopped',
    port INTEGER,
    pid INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP,
    config JSONB,
    metadata JSONB
);

-- Analytics summary table
CREATE TABLE IF NOT EXISTS analytics_summary (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    instance_id VARCHAR(255),
    date DATE DEFAULT CURRENT_DATE,
    total_messages INTEGER DEFAULT 0,
    sent_messages INTEGER DEFAULT 0,
    received_messages INTEGER DEFAULT 0,
    failed_messages INTEGER DEFAULT 0,
    active_users INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(instance_id, date)
);

-- System events table
CREATE TABLE IF NOT EXISTS system_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    instance_id VARCHAR(255),
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,
    severity VARCHAR(20) DEFAULT 'info',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_instance_registry_status ON instance_registry(status);
CREATE INDEX IF NOT EXISTS idx_instance_registry_updated_at ON instance_registry(updated_at);
CREATE INDEX IF NOT EXISTS idx_analytics_summary_date ON analytics_summary(date);
CREATE INDEX IF NOT EXISTS idx_analytics_summary_instance ON analytics_summary(instance_id);
CREATE INDEX IF NOT EXISTS idx_system_events_type ON system_events(event_type);
CREATE INDEX IF NOT EXISTS idx_system_events_created_at ON system_events(created_at);

-- Insert initial system event
INSERT INTO system_events (event_type, event_data, severity) 
VALUES ('system_initialized', '{"message": "WhatsApp Multi-Device system initialized"}', 'info');

COMMIT;