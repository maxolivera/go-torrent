package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/commands"
)

func main() {
	args := flag.Args()

	if len(args) < 2 {
		slog.Error("not enough arguments")
		return
	}

	command := args[0]
	file := args[1]

	switch command {
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)

	case "decode":
		err := commands.Decode([]byte(file))
		if err != nil {
			fmt.Println(err)
			return
		}

	case "info":
		err := commands.Info(file)
		if err != nil {
			fmt.Println(err)
			return
		}

	case "peers":
		err := commands.Peers(file)
		if err != nil {
			fmt.Println(err)
			return
		}

	case "handshake":
		connection := args[2]
		slog.Info("connection to be used", "connection", connection)
		err := commands.Handshake(file, connection)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

// LOGGING

func init() {
	// get log level from flags
	var debugLevel DebugType
	flag.Var(&debugLevel, "debug", "Debug level (info, debug, warning)")
	flag.Parse()

	// configure logger
	var logLevel slog.Level
	switch debugLevel {
	case DebugDebug:
		logLevel = slog.LevelDebug
	case DebugInfo:
		logLevel = slog.LevelInfo
	case DebugWarning:
		logLevel = slog.LevelWarn
	default:
		logLevel = slog.LevelWarn
	}

	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: logLevel},
	))
	slog.SetDefault(logger)
}

type DebugType int

const (
	DebugInfo DebugType = iota
	DebugDebug
	DebugWarning
)

func (dt *DebugType) String() string {
	switch *dt {
	case DebugInfo:
		return "info"
	case DebugDebug:
		return "debug"
	case DebugWarning:
		return "warning"
	default:
		return "unknown"
	}
}

func (dt *DebugType) Set(s string) error {
	switch s {
	case "info":
		*dt = DebugInfo
	case "debug":
		*dt = DebugDebug
	case "warning", "warn":
		*dt = DebugWarning
	default:
		return fmt.Errorf("invalid debug type: %s", s)
	}
	return nil
}
