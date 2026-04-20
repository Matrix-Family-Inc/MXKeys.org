/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Thu 06 Feb 2026 UTC
 * Status: Created
 */

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/lib/pq"

	"mxkeys/internal/config"
	"mxkeys/internal/server"
	"mxkeys/internal/version"
	"mxkeys/internal/zero/log"
)

func main() {
	configPath := flag.String("config", "", "path to config.yaml (optional; falls back to config.yaml and /etc/mxkeys/config.yaml)")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: mxkeys [flags]\n\nFlags:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVersion {
		fmt.Println(version.Full())
		return
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	configureLogging(cfg)

	log.Info("MXKeys Federation Trust Infrastructure starting",
		"server", cfg.Server.Name,
		"port", cfg.Server.Port,
		"version", version.Version,
	)

	// Create server
	srv, err := server.New(cfg)
	if err != nil {
		log.Error("Failed to create server", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := srv.Close(); err != nil {
			log.Warn("server close failed", "error", err)
		}
	}()

	// Signal handling:
	//   * First SIGINT/SIGTERM triggers graceful shutdown (ctx cancel).
	//   * Second signal forces exit(130) so operators can interrupt a
	//     stuck shutdown. A stuck shutdown usually indicates an HTTP
	//     client holding open connections past shutdown_timeout or a
	//     peer that never closes its raft connection.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		first := <-sigChan
		log.Info("Received shutdown signal", "signal", first.String())
		cancel()

		second := <-sigChan
		log.Warn("Received second shutdown signal, forcing exit",
			"signal", second.String(),
			"exit_code", 130,
		)
		os.Exit(130)
	}()

	if err := srv.Run(ctx); err != nil {
		log.Error("Server error", "error", err)
		os.Exit(1)
	}

	log.Info("MXKeys stopped")
}

func configureLogging(cfg *config.Config) {
	if cfg.Logging.Format == "json" {
		log.SetJSONWithLevel(cfg.Logging.Level)
	} else {
		log.SetLevel(cfg.Logging.Level)
	}
}
