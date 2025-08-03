// config/config.go
package config

type Config struct {
	Server    ServerConfig
	Database  Database
	McpServer McpServer
	Sites     SitesConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
	Environment  string // "development", "production"
}

type Database struct {
	Postgres string `json:"postgres"`
}

type McpServer struct {
	Host string
	Port string
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
