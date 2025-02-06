CREATE TABLE IF NOT EXISTS tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    artist VARCHAR(255) NOT NULL,
    album VARCHAR(255),
    isrc VARCHAR(12),
    iswc VARCHAR(11),
    year INTEGER,
    label VARCHAR(255),
    publisher VARCHAR(255),
    
    -- AI-generated metadata
    bpm DECIMAL(5,2),
    key VARCHAR(10),
    mood VARCHAR(50),
    genre VARCHAR(50),
    
    -- AI processing metadata
    ai_confidence DECIMAL(4,3),
    model_version VARCHAR(50),
    needs_review BOOLEAN DEFAULT false,
    
    -- Compliance and auditing
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    territory VARCHAR(2)
);

-- Indexes for common queries
CREATE INDEX idx_tracks_isrc ON tracks(isrc) WHERE deleted_at IS NULL;
CREATE INDEX idx_tracks_artist ON tracks(artist) WHERE deleted_at IS NULL;
CREATE INDEX idx_tracks_label ON tracks(label) WHERE deleted_at IS NULL;
CREATE INDEX idx_tracks_created_at ON tracks(created_at) WHERE deleted_at IS NULL;

-- Trigger to automatically update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tracks_updated_at
    BEFORE UPDATE ON tracks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 