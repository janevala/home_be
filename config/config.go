// config/config.go
package config

type Config struct {
	Server ServerConfig
	Ollama Ollama
	Sites  SitesConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout  int
	WriteTimeout int
}

type Ollama struct {
	Host  string
	Port  string
	Model string // "mistral:7b", "qwen2.5-coder:14b"
}

type SitesConfig struct {
	Title string
	Sites []Site
}

type Site struct {
	Title string
	Url   string
}
