package config

import "os"

const (
	defaultListenAddr     = ":9100"
	defaultUpstreamMCPURL = "http://127.0.0.1:9000/mcp"
)

// Config contains runtime configuration.
// Configは実行時設定を保持する。
type Config struct {
	ListenAddr     string
	UpstreamMCPURL string
}

// Load reads configuration from environment variables.
// Loadは環境変数から設定を読み込む。
func Load() Config {
	return Config{
		ListenAddr:     getEnv("LISTEN_ADDR", defaultListenAddr),
		UpstreamMCPURL: getEnv("UPSTREAM_MCP_URL", defaultUpstreamMCPURL),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
