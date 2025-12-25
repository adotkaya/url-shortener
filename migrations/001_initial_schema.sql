-- Migration: Initial Schema
-- This creates the database tables for our URL shortener
-- Migrations allow us to version control our database schema

-- Create URLs table
CREATE TABLE IF NOT EXISTS urls (
    -- UUID (Universally Unique Identifier) is better than auto-increment for distributed systems
    -- gen_random_uuid() is a PostgreSQL function that generates a random UUID
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),

    -- short_code is the shortened identifier (e.g., "abc123")
    -- UNIQUE constraint ensures no duplicates
    -- NOT NULL means this field is required
    short_code VARCHAR(20) UNIQUE NOT NULL,

    -- TEXT type can store URLs of any length (vs VARCHAR which has a limit)
    original_url TEXT NOT NULL,

    -- Custom alias is optional, hence no NOT NULL
    -- UNIQUE ensures no two URLs can have the same custom alias
    custom_alias VARCHAR(50) UNIQUE,

    -- Timestamps for tracking
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP,

    -- BIGINT for large numbers (can store up to 9,223,372,036,854,775,807)
    clicks BIGINT DEFAULT 0,

    -- Track who created the URL (could be API key, user ID, etc.)
    created_by VARCHAR(255),

    -- Soft delete flag - we don't actually delete data, just mark it inactive
    is_active BOOLEAN DEFAULT true
);

-- Create indexes for faster queries
-- Index on short_code because we'll query by it frequently (every redirect)
-- This makes lookups O(log n) instead of O(n)
CREATE INDEX IF NOT EXISTS idx_short_code ON urls(short_code);

-- Partial index: only index rows where expires_at is not null
-- This is more efficient than indexing all rows
CREATE INDEX IF NOT EXISTS idx_expires_at ON urls(expires_at) WHERE expires_at IS NOT NULL;

-- Index for finding active URLs
CREATE INDEX IF NOT EXISTS idx_is_active ON urls(is_active) WHERE is_active = true;

-- Create URL clicks table for analytics
CREATE TABLE IF NOT EXISTS url_clicks (
    -- BIGSERIAL is auto-incrementing BIGINT (1, 2, 3, ...)
    id BIGSERIAL PRIMARY KEY,

    -- Foreign key to urls table
    -- ON DELETE CASCADE means if a URL is deleted, all its clicks are deleted too
    url_id UUID NOT NULL REFERENCES urls(id) ON DELETE CASCADE,

    clicked_at TIMESTAMP NOT NULL DEFAULT NOW(),

    -- INET is a PostgreSQL type for storing IP addresses
    ip_address INET,

    user_agent TEXT,
    referer TEXT,

    -- Geolocation data
    country_code VARCHAR(2),  -- ISO country code (e.g., "US", "UK")
    city VARCHAR(100)
);

-- Index on url_id for fast lookups of clicks for a specific URL
CREATE INDEX IF NOT EXISTS idx_url_clicks_url_id ON url_clicks(url_id);

-- Index on clicked_at for time-based queries (e.g., "clicks in last 7 days")
CREATE INDEX IF NOT EXISTS idx_url_clicks_clicked_at ON url_clicks(clicked_at);

-- Composite index for common query pattern: clicks for a URL in a time range
CREATE INDEX IF NOT EXISTS idx_url_clicks_url_time ON url_clicks(url_id, clicked_at);
