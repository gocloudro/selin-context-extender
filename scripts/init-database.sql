-- Selin Database Schema Initialization
-- This script creates the basic database schema for the Selin learning system

-- Create database (if running manually)
-- CREATE DATABASE selin;

-- Connect to selin database
\c selin;

-- Create content_metadata table as specified in backend_structure_document.mdc
CREATE TABLE IF NOT EXISTS content_metadata (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_url TEXT NOT NULL,
  author TEXT,
  timestamp TIMESTAMP WITH TIME ZONE,
  tags TEXT[],
  content_type TEXT,
  collection_date TIMESTAMP WITH TIME ZONE DEFAULT now(),
  source_platform TEXT,
  language TEXT,
  content_summary TEXT,
  relevance_score REAL,
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_content_source_platform ON content_metadata(source_platform);
CREATE INDEX IF NOT EXISTS idx_content_timestamp ON content_metadata(timestamp);
CREATE INDEX IF NOT EXISTS idx_content_collection_date ON content_metadata(collection_date);
CREATE INDEX IF NOT EXISTS idx_content_relevance_score ON content_metadata(relevance_score);
CREATE INDEX IF NOT EXISTS idx_content_tags ON content_metadata USING GIN(tags);

-- Create learning_progress table to track user learning
CREATE TABLE IF NOT EXISTS learning_progress (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  topic TEXT NOT NULL,
  skill_level TEXT DEFAULT 'beginner',
  progress_score REAL DEFAULT 0.0,
  last_updated TIMESTAMP WITH TIME ZONE DEFAULT now(),
  total_content_consumed INTEGER DEFAULT 0,
  total_queries INTEGER DEFAULT 0,
  mastery_indicators JSONB DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Create query_history table to track all user queries
CREATE TABLE IF NOT EXISTS query_history (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id TEXT DEFAULT 'default_user',
  query_text TEXT NOT NULL,
  response_text TEXT,
  request_id TEXT,
  processing_time_ms INTEGER,
  relevant_content_ids UUID[],
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Create indexes for query_history
CREATE INDEX IF NOT EXISTS idx_query_history_user_id ON query_history(user_id);
CREATE INDEX IF NOT EXISTS idx_query_history_created_at ON query_history(created_at);

-- Create data_sources table to track configured sources
CREATE TABLE IF NOT EXISTS data_sources (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  source_type TEXT NOT NULL, -- 'reddit', 'twitter', 'github', 'file'
  source_name TEXT NOT NULL, -- subreddit name, twitter handle, repo name, etc.
  enabled BOOLEAN DEFAULT true,
  last_collection TIMESTAMP WITH TIME ZONE,
  collection_count INTEGER DEFAULT 0,
  error_count INTEGER DEFAULT 0,
  configuration JSONB DEFAULT '{}',
  created_at TIMESTAMP WITH TIME ZONE DEFAULT now(),
  updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
);

-- Create indexes for data_sources
CREATE INDEX IF NOT EXISTS idx_data_sources_type ON data_sources(source_type);
CREATE INDEX IF NOT EXISTS idx_data_sources_enabled ON data_sources(enabled);

-- Insert initial data sources based on user/sources.yaml
INSERT INTO data_sources (source_type, source_name, configuration) VALUES
  ('reddit', 'golang', '{"collection_interval": "5m", "max_posts_per_run": 50}'),
  ('reddit', 'cosmosdev', '{"collection_interval": "5m", "max_posts_per_run": 50}'),
  ('reddit', 'cryptography', '{"collection_interval": "5m", "max_posts_per_run": 50}'),
  ('twitter', '#cosmos', '{"collection_interval": "10m", "max_tweets_per_run": 100}'),
  ('twitter', '#golang', '{"collection_interval": "10m", "max_tweets_per_run": 100}'),
  ('github', 'cosmos/cosmos-sdk', '{"collection_interval": "30m", "track_releases": true}'),
  ('github', 'golang/go', '{"collection_interval": "30m", "track_releases": true}')
ON CONFLICT DO NOTHING;

-- Insert initial learning progress tracking
INSERT INTO learning_progress (topic, skill_level) VALUES
  ('golang', 'intermediate'),
  ('blockchain', 'beginner'),
  ('cryptography', 'beginner'),
  ('kubernetes', 'intermediate')
ON CONFLICT DO NOTHING;

-- Create a view for recent content
CREATE OR REPLACE VIEW recent_content AS
SELECT 
  id,
  source_url,
  author,
  timestamp,
  source_platform,
  content_type,
  content_summary,
  relevance_score,
  collection_date
FROM content_metadata
WHERE collection_date >= NOW() - INTERVAL '7 days'
ORDER BY collection_date DESC;

-- Create a view for learning analytics
CREATE OR REPLACE VIEW learning_analytics AS
SELECT 
  topic,
  skill_level,
  progress_score,
  total_content_consumed,
  total_queries,
  last_updated,
  EXTRACT(days FROM NOW() - last_updated) as days_since_update
FROM learning_progress
ORDER BY last_updated DESC;

-- Grant permissions (for production, you'd want more restrictive permissions)
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO postgres;

-- Display success message
\echo 'Selin database schema initialized successfully!'
\echo 'Tables created: content_metadata, learning_progress, query_history, data_sources'
\echo 'Views created: recent_content, learning_analytics'
\echo 'Database is ready for Selin services.'
