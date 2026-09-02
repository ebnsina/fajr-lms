// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env             string
	Addr            string
	DatabaseURL     string
	LogLevel        slog.Level
	MediaHosts      []string
	S3              S3Config
	Bank            BankConfig
	SMS             SMSConfig
	SSLCommerz      SSLCommerzConfig
	BKash           BKashConfig
	PublicURL       string
	ShutdownTimeout time.Duration
}

// Load reads configuration, returning an error for any missing or invalid value.
func Load() (Config, error) {
	c := Config{
		Env:             env("FAJR_ENV", "development"),
		Addr:            env("FAJR_ADDR", ":8080"),
		DatabaseURL:     env("FAJR_DATABASE_URL", ""),
		ShutdownTimeout: 15 * time.Second,
	}

	if c.DatabaseURL == "" {
		return Config{}, fmt.Errorf("config: FAJR_DATABASE_URL is required")
	}

	// Extra video hosts an operator trusts, on top of the built-in platforms.
	if hosts := env("FAJR_MEDIA_HOSTS", ""); hosts != "" {
		for _, h := range strings.Split(hosts, ",") {
			if h = strings.TrimSpace(h); h != "" {
				c.MediaHosts = append(c.MediaHosts, h)
			}
		}
	}

	c.S3 = S3Config{
		Endpoint:  env("FAJR_S3_ENDPOINT", ""),
		Bucket:    env("FAJR_S3_BUCKET", "fajr-media"),
		AccessKey: env("FAJR_S3_ACCESS_KEY", ""),
		SecretKey: env("FAJR_S3_SECRET_KEY", ""),
		Region:    env("FAJR_S3_REGION", "us-east-1"),
		UseSSL:    env("FAJR_S3_USE_SSL", "false") == "true",
	}
	if c.S3.Enabled() {
		if c.S3.AccessKey == "" || c.S3.SecretKey == "" {
			return Config{}, fmt.Errorf("config: FAJR_S3_ENDPOINT needs FAJR_S3_ACCESS_KEY and FAJR_S3_SECRET_KEY")
		}
		if raw := env("FAJR_S3_MAX_UPLOAD_BYTES", ""); raw != "" {
			n, err := strconv.ParseInt(raw, 10, 64)
			if err != nil || n <= 0 {
				return Config{}, fmt.Errorf("config: FAJR_S3_MAX_UPLOAD_BYTES must be a positive integer")
			}
			c.S3.MaxBytes = n
		}
	}

	c.PublicURL = strings.TrimSuffix(env("FAJR_PUBLIC_URL", "http://localhost:8080"), "/")

	c.SSLCommerz = SSLCommerzConfig{
		StoreID:     env("FAJR_SSLCOMMERZ_STORE_ID", ""),
		StorePasswd: env("FAJR_SSLCOMMERZ_STORE_PASSWORD", ""),
		Sandbox:     env("FAJR_SSLCOMMERZ_SANDBOX", "true") == "true",
	}
	if c.SSLCommerz.Enabled() && c.SSLCommerz.StorePasswd == "" {
		return Config{}, fmt.Errorf("config: FAJR_SSLCOMMERZ_STORE_ID needs FAJR_SSLCOMMERZ_STORE_PASSWORD")
	}

	c.BKash = BKashConfig{
		Username:  env("FAJR_BKASH_USERNAME", ""),
		Password:  env("FAJR_BKASH_PASSWORD", ""),
		AppKey:    env("FAJR_BKASH_APP_KEY", ""),
		AppSecret: env("FAJR_BKASH_APP_SECRET", ""),
		Sandbox:   env("FAJR_BKASH_SANDBOX", "true") == "true",
	}
	if c.BKash.Enabled() && (c.BKash.Username == "" || c.BKash.Password == "" || c.BKash.AppSecret == "") {
		return Config{}, fmt.Errorf("config: FAJR_BKASH_APP_KEY needs a username, password and app secret")
	}

	c.SMS = SMSConfig{
		URL:             env("FAJR_SMS_URL", ""),
		Method:          env("FAJR_SMS_METHOD", "POST"),
		Body:            env("FAJR_SMS_BODY", "msisdn={to}&sms={message}&sender={sender}"),
		Encoding:        env("FAJR_SMS_ENCODING", "form"),
		Sender:          env("FAJR_SMS_SENDER", "FAJR"),
		SuccessContains: env("FAJR_SMS_SUCCESS_CONTAINS", ""),
	}
	if token := env("FAJR_SMS_AUTH_HEADER", ""); token != "" {
		name, value, found := strings.Cut(token, ":")
		if !found {
			return Config{}, fmt.Errorf("config: FAJR_SMS_AUTH_HEADER must look like Name:value")
		}
		c.SMS.Headers = map[string]string{strings.TrimSpace(name): strings.TrimSpace(value)}
	}

	c.Bank = BankConfig{
		AccountName:   env("FAJR_BANK_ACCOUNT_NAME", ""),
		AccountNumber: env("FAJR_BANK_ACCOUNT_NUMBER", ""),
		BankName:      env("FAJR_BANK_NAME", ""),
		BranchName:    env("FAJR_BANK_BRANCH", ""),
	}

	lvl := env("FAJR_LOG_LEVEL", "info")
	if err := c.LogLevel.UnmarshalText([]byte(lvl)); err != nil {
		return Config{}, fmt.Errorf("config: invalid FAJR_LOG_LEVEL %q: %w", lvl, err)
	}

	switch c.Env {
	case "development", "staging", "production":
	default:
		return Config{}, fmt.Errorf("config: invalid FAJR_ENV %q", c.Env)
	}
	return c, nil
}

// S3Config is optional: without an endpoint the file provider is not registered
// and the API runs on embeds alone.
type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
	MaxBytes  int64
}

func (s S3Config) Enabled() bool { return s.Endpoint != "" }

// SSLCommerzConfig is optional; without a store id the gateway is not offered.
type SSLCommerzConfig struct {
	StoreID     string
	StorePasswd string
	Sandbox     bool
}

func (s SSLCommerzConfig) Enabled() bool { return s.StoreID != "" }

// BKashConfig is optional; without an app key bKash is not offered.
type BKashConfig struct {
	Username  string
	Password  string
	AppKey    string
	AppSecret string
	Sandbox   bool
}

func (b BKashConfig) Enabled() bool { return b.AppKey != "" }

// SMSConfig points the SMS channel at any gateway that speaks HTTP. Local
// providers differ only in field names, so one template covers them.
type SMSConfig struct {
	URL             string
	Method          string
	Body            string
	Encoding        string
	Sender          string
	SuccessContains string
	Headers         map[string]string
}

func (s SMSConfig) Enabled() bool { return s.URL != "" }

// BankConfig is the account a manual transfer is paid into.
type BankConfig struct {
	AccountName   string
	AccountNumber string
	BankName      string
	BranchName    string
}

func (c Config) IsProduction() bool { return c.Env == "production" }

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
