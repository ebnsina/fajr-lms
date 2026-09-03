// Command api runs the Fajr LMS HTTP API.
package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ebnsina/fajr-lms/internal/ai"
	"github.com/ebnsina/fajr-lms/internal/api"
	"github.com/ebnsina/fajr-lms/internal/config"
	"github.com/ebnsina/fajr-lms/internal/database"
	"github.com/ebnsina/fajr-lms/internal/httpx"
	"github.com/ebnsina/fajr-lms/internal/identity"
	"github.com/ebnsina/fajr-lms/internal/media"
	"github.com/ebnsina/fajr-lms/internal/notify"
	"github.com/ebnsina/fajr-lms/internal/payment"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel})))

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	store, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer store.Close()

	providers := []media.Provider{media.Embed{AllowedHosts: cfg.MediaHosts}}
	if cfg.S3.Enabled() {
		files, err := media.NewObjectStore(ctx, media.ObjectStoreConfig(cfg.S3))
		if err != nil {
			return err
		}
		providers = append(providers, files)
		slog.Info("file media provider enabled", "bucket", cfg.S3.Bucket)
	}

	registry, err := media.NewRegistry("embed", providers...)
	if err != nil {
		return err
	}

	methods := []payment.Provider{payment.BankTransfer{
		AccountName: cfg.Bank.AccountName, AccountNumber: cfg.Bank.AccountNumber,
		BankName: cfg.Bank.BankName, BranchName: cfg.Bank.BranchName,
	}}
	if cfg.SSLCommerz.Enabled() {
		methods = append(methods, payment.SSLCommerz{
			StoreID: cfg.SSLCommerz.StoreID, StorePasswd: cfg.SSLCommerz.StorePasswd,
			Sandbox:    cfg.SSLCommerz.Sandbox,
			SuccessURL: cfg.PublicURL + "/pay/done", FailURL: cfg.PublicURL + "/pay/failed",
			CancelURL: cfg.PublicURL + "/pay/cancelled",
			IPNURL:    cfg.PublicURL + "/v1/payment/{tenant}/sslcommerz/callback",
		})
		slog.Info("sslcommerz enabled", "sandbox", cfg.SSLCommerz.Sandbox)
	}

	if cfg.BKash.Enabled() {
		methods = append(methods, &payment.BKash{
			Username: cfg.BKash.Username, Password: cfg.BKash.Password,
			AppKey: cfg.BKash.AppKey, AppSecret: cfg.BKash.AppSecret, Sandbox: cfg.BKash.Sandbox,
			CallbackURL: cfg.PublicURL + "/v1/payment/{tenant}/bkash/callback",
		})
		slog.Info("bkash enabled", "sandbox", cfg.BKash.Sandbox)
	}

	payments, err := payment.NewRegistry("bank_transfer", methods...)
	if err != nil {
		return err
	}

	// The same channel carries login codes and announcements.
	primary := notify.Channel(notify.LogChannel{})
	if cfg.SMS.Enabled() {
		primary = notify.HTTPChannel{
			ChannelName: "sms", Method: cfg.SMS.Method, URL: cfg.SMS.URL, Body: cfg.SMS.Body,
			Sender: cfg.SMS.Sender, Encoding: cfg.SMS.Encoding, SuccessContains: cfg.SMS.SuccessContains,
			Headers: cfg.SMS.Headers,
		}
		slog.Info("sms channel enabled", "url", cfg.SMS.URL)
	}

	server := api.NewServer(store, identity.New(store, primary), registry, payments, cfg.PublicURL)
	server.UseNotifier(notify.NewService(server, primary.Name()))

	if cfg.AI.Enabled() {
		server.UseAI(ai.Anthropic{Key: cfg.AI.Key, Model: cfg.AI.Model})
		slog.Info("fajr ai enabled", "model", cfg.AI.Model)
	}

	dispatcher := notify.NewDispatcher(server, primary)
	go dispatcher.Run(ctx)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           httpx.Chain(server.Routes(), httpx.RequestID, httpx.Recover, httpx.Log),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	return httpx.Serve(ctx, srv, cfg.ShutdownTimeout)
}
