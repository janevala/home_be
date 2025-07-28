// config/config.go
package config

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Sites    SitesConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
	Environment  string // "development", "production"
}

type DatabaseConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

type SitesConfig struct {
	Time  int
	Title string
	Sites []Site
}

type Site struct {
	Uuid  string
	Title string
	Url   string
}
