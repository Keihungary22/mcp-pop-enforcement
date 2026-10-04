package config

import "os"

const (
	defaultListenAddr       = ":9100"
	defaultUpstreamMCPURL   = "http://127.0.0.1:9000/mcp"
	defaultExpectedIssuer   = "https://issuer.example"
	defaultExpectedAudience = "mcp://resource"
	defaultRequiredScope    = "mcp:invoke"
	defaultExpectedDPoPHTU  = "http://127.0.0.1:9100/mcp"
)

// Config contains runtime configuration.
// Configは実行時設定を保持する。
type Config struct {
	ListenAddr               string
	UpstreamMCPURL           string
	AccessTokenPublicKeyFile string
	ExpectedIssuer           string
	ExpectedAudience         string
	RequiredScope            string
	ExpectedDPoPHTU          string
}

// Load reads configuration from environment variables.
// Loadは環境変数から設定を読み込む。
func Load() Config {
	return Config{
		ListenAddr: getEnv(
			"LISTEN_ADDR",
			defaultListenAddr,
		),
		UpstreamMCPURL: getEnv(
			"UPSTREAM_MCP_URL",
			defaultUpstreamMCPURL,
		),
		AccessTokenPublicKeyFile: os.Getenv(
			"ACCESS_TOKEN_PUBLIC_KEY_FILE",
		),
		ExpectedIssuer: getEnv(
			"EXPECTED_ISSUER",
			defaultExpectedIssuer,
		),
		ExpectedAudience: getEnv(
			"EXPECTED_AUDIENCE",
			defaultExpectedAudience,
		),
		RequiredScope: getEnv(
			"REQUIRED_SCOPE",
			defaultRequiredScope,
		),
		ExpectedDPoPHTU: getEnv(
			"EXPECTED_DPOP_HTU",
			defaultExpectedDPoPHTU,
		),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
