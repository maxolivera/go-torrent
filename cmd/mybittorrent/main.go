package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/commands"
)

// global variables, set during init(), used in main()
var debugLevel DebugType
var totalWorkers int

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

	case "download_piece":
		commandFlags := flag.NewFlagSet(command, flag.ExitOnError)
		output := commandFlags.String("o", "", "Output file")
		err := commandFlags.Parse(args[1:])
		if err != nil {
			fmt.Println(err)
			return
		}

		commandArgs := commandFlags.Args()
		if len(commandArgs) < 2 {
			fmt.Println("Not enough arguments for command", "command", command)
			return
		}

		file := commandArgs[0]
		pieceNumber, err := strconv.Atoi(commandArgs[1])
		if err != nil {
			fmt.Println(err)
			return
		}

		err = commands.DownloadPiece(file, *output, pieceNumber, totalWorkers)
		if err != nil {
			fmt.Println(err)
			return
		}
	}
}

// LOGGING

func init() {
	// get log level from flags
	flag.Var(&debugLevel, "debug", "Debug level (info, debug, warning)")
	flag.IntVar(&totalWorkers, "w", 5, "Total amount of concurrent block downloads")
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
		logLevel = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{Level: logLevel},
	))
	slog.SetDefault(logger)

	slog.Info("set log level", "level", logLevel)
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
