package article

type Article struct {
	Title    string `yaml:"title"`
	Path     string `yaml:"path"`
	UUID     string `yaml:"uuid"`
	Content  string
	FilePath string
}

type HatenaEntry struct {
	ID       string
	Title    string
	Content  string
	URL      string
	EditURL  string
	Updated  string
	IsDraft  bool
}