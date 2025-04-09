-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Create plaid_items table
CREATE TABLE IF NOT EXISTS plaid_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id TEXT NOT NULL, -- This will store the Supabase auth.uid()
    access_token TEXT NOT NULL,
    item_id TEXT NOT NULL UNIQUE,
    institution_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS user_info (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id TEXT NOT NULL,
    income FLOAT,
    savings_goal FLOAT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes
CREATE INDEX IF NOT EXISTS idx_plaid_items_user_id ON plaid_items(user_id);
CREATE INDEX IF NOT EXISTS idx_plaid_items_item_id ON plaid_items(item_id);
CREATE INDEX IF NOT EXISTS idx_conversations_user_id ON conversations(user_id);
CREATE INDEX IF NOT EXISTS idx_user_info_user_id ON user_info(user_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create trigger for updated_at
CREATE TRIGGER update_plaid_items_updated_at
    BEFORE UPDATE ON plaid_items
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_conversations_updated_at
    BEFORE UPDATE ON conversations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_info_updated_at
    BEFORE UPDATE ON user_info
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- Add RLS (Row Level Security) policies
ALTER TABLE plaid_items ENABLE ROW LEVEL SECURITY;

-- Add RLS policies to conversations table
ALTER TABLE conversations ENABLE ROW LEVEL SECURITY;

ALTER TABLE user_info ENABLE ROW LEVEL SECURITY;

-- Plaid items policies
CREATE POLICY IF NOT EXISTS "Users can view their own Plaid items"
    ON plaid_items FOR SELECT
    USING (auth.uid()::text = user_id);

CREATE POLICY IF NOT EXISTS "Users can insert their own Plaid items"
    ON plaid_items FOR INSERT
    WITH CHECK (auth.uid()::text = user_id);

CREATE POLICY IF NOT EXISTS "Users can update their own Plaid items"
    ON plaid_items FOR UPDATE
    USING (auth.uid()::text = user_id);

CREATE POLICY IF NOT EXISTS "Users can delete their own Plaid items"
    ON plaid_items FOR DELETE
    USING (auth.uid()::text = user_id); 

-- Conversations policies
CREATE POLICY IF NOT EXISTS "Users can view their own conversations"
    ON conversations FOR SELECT
    USING (auth.uid()::text = user_id);

CREATE POLICY IF NOT EXISTS "Users can insert their own conversations"
    ON conversations FOR INSERT
    WITH CHECK (auth.uid()::text = user_id);

CREATE POLICY IF NOT EXISTS "Users can delete their own conversations"
    ON conversations FOR DELETE
    USING (auth.uid()::text = user_id);

-- User info policies
CREATE POLICY IF NOT EXISTS "Users can view their own user info"
    ON user_info FOR SELECT
    USING (auth.uid()::text = user_id);

CREATE POLICY IF NOT EXISTS "Users can insert their own user info"
    ON user_info FOR INSERT
    WITH CHECK (auth.uid()::text = user_id);    