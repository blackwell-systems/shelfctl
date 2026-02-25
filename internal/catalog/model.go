package catalog

// Book is one entry in a shelf's catalog.yml.
type Book struct {
	ID        string   `yaml:"id"`
	Title     string   `yaml:"title"`
	Author    string   `yaml:"author,omitempty"`
	Year      int      `yaml:"year,omitempty"`
	Tags      []string `yaml:"tags,omitempty"`
	Format    string   `yaml:"format"`
	Cover     string   `yaml:"cover,omitempty"`
	Checksum  Checksum `yaml:"checksum,omitempty"`
	SizeBytes int64    `yaml:"size_bytes,omitempty"`
	Source    Source   `yaml:"source"`
	Meta      Meta     `yaml:"meta,omitempty"`
}

// Checksum holds content hashes.
type Checksum struct {
	SHA256 string `yaml:"sha256,omitempty"`
}

// Source describes where the asset is stored.
type Source struct {
	Type    string `yaml:"type"`
	Owner   string `yaml:"owner"`
	Repo    string `yaml:"repo"`
	Release string `yaml:"release"`
	Asset   string `yaml:"asset"`
}

// Meta holds optional provenance data.
type Meta struct {
	AddedAt      string `yaml:"added_at,omitempty"`
	MigratedFrom string `yaml:"migrated_from,omitempty"`
}
