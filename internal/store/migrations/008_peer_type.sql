ALTER TABLE peers ADD COLUMN type TEXT NOT NULL DEFAULT 'client' CHECK(type IN ('client', 'site'));
