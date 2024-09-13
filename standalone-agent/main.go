package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"
    	"github.com/BL00DY-C0D3/evr-game-wrapper/wrapper" // Adjust the path as necessary

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var version string = "v0.0.0"

type Flags struct {
	Targets         map[string][]int
	Frequency       int
	Format          string
	OutputDirectory string
	LogPath         string
	Debug           bool
	WorkerCount     int
	QueueSize       int
}

var opts = Flags{}

var logger *zap.Logger

func setupLogging() {
	level := zap.InfoLevel
	if opts.Debug {
		level = zap.DebugLevel
	}
	// Log to a file
	if opts.LogPath != "" {
		// Create a new logger that logs to a file
		cfg := zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
		cfg.OutputPaths = []string{opts.LogPath}
		cfg.ErrorOutputPaths = []string{opts.LogPath}

		cfg.Level.SetLevel(level)
		fileLogger, _ := cfg.Build()

		defer fileLogger.Sync() // flushes buffer, if any

		// Create a new logger that logs to the console
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
		cfg.OutputPaths = []string{"stdout"}
		cfg.ErrorOutputPaths = []string{"stderr"}

		cfg.Level.SetLevel(level)

		consoleLogger, _ := cfg.Build()
		defer consoleLogger.Sync() // flushes buffer, if any

		// Create a new logger that logs to both the file and the console
		core := zapcore.NewTee(
			fileLogger.Core(),
			consoleLogger.Core(),
		)
		logger = zap.New(core)
	} else {
		cfg := zap.NewProductionConfig()
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.RFC3339)
		cfg.Level.SetLevel(level)
		logger, _ = cfg.Build()
	}

	defer logger.Sync() // flushes buffer, if any
}

func parseFlags() {
	flag.IntVar(&opts.Frequency, "frequency", 10, "Frequency in Hz")
	flag.BoolVar(&opts.Debug, "debug", false, "Enable debug logging")
	flag.StringVar(&opts.LogPath, "log", "", "Log file path")
	// Output options
	outputGroup := flag.NewFlagSet("output", flag.ContinueOnError)
	outputGroup.StringVar(&opts.Format, "format", "replay", "Output format")
	outputGroup.StringVar(&opts.OutputDirectory, "output", "output", "Output directory")
	flag.IntVar(&opts.WorkerCount, "workers", 2, "Number of workers")
	flag.IntVar(&opts.QueueSize, "queue", 128, "Queue size")

	// Advanced options
	advancedGroup := flag.NewFlagSet("advanced", flag.ContinueOnError)
	advancedGroup.IntVar(&opts.WorkerCount, "workers", 10, "Number of workers")
	advancedGroup.IntVar(&opts.QueueSize, "queue", 100, "Queue size")
	// Set usage
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), "Usage: %s [options] host:port[-endPort] [host:port[-endPort]...]\n", os.Args[0])
		fmt.Fprintf(flag.CommandLine.Output(), "Version: %s\n", version)
		flag.PrintDefaults()
		// include version

	}

	flag.Parse()

	// Parse N arguments as host:port or host:startPort-endPort
	if flag.NArg() != 1 {
		// Show help
		flag.Usage()
		// Exit
		os.Exit(1)
	}

	opts.Targets = make(map[string][]int)
	for _, hostPort := range flag.Args() {
		host, ports, err := parseHostPort(hostPort)
		if err != nil {
			log.Fatalf("Failed to parse host and port: %v", err)
		}
		opts.Targets[host] = ports
	}
}

func parseHostPort(s string) (string, []int, error) {
	components := strings.Split(s, ":")
	if len(components) != 2 {
		return "", nil, errors.New("invalid format, expected host:port or host:startPort-endPort")
	}

	host := components[0]

	ports, err := parsePortRange(components[1])
	if err != nil {
		return "", nil, err
	}

	return host, ports, nil
}

func main() {

	parseFlags()
	setupLogging()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	if opts.Frequency <= 0 {
		log.Fatalf("Frequency must be greater than 0")
	}

	agent := wrapper.NewSessionAgent(logger, opts.WorkerCount, opts.QueueSize)

	urls := make([]string, 0)
	for host, ports := range opts.Targets {
		for _, port := range ports {
			urls = append(urls, fmt.Sprintf("http://%s:%d/session", host, port))
		}
	}

	logger.Debug("Starting agent", zap.Int("frequency", opts.Frequency), zap.Int("target_count", len(urls)))

	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// For each port in the target list, check if the port is open, then start polling
		activePollers := make(map[string]struct{})
		for {
			// Search for open ports
			for _, url := range urls {
				logger.Debug("Checking url", zap.String("url", url))
				select {
				case <-ctx.Done():
					return
				default:
				}
				<-time.After(10 * time.Millisecond)
				if _, ok := activePollers[url]; ok {
					logger.Debug("Poller already active, skipping...", zap.String("url", url))
					continue
				}
				logger.Debug("Adding poller", zap.String("url", url))
				activePollers[url] = struct{}{}
				go func() {
					agent.Poll(url, opts.Frequency)
					delete(activePollers, url)
				}()
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(3 * time.Second):
			}
		}
	}()
	go agent.Consume()
	// Wait for signal interrupt
	<-interrupt
	logger.Info("Shutting down")
}

func parsePortRange(port string) ([]int, error) {

	// 1234,3456,7890-10111
	portRanges := strings.Split(port, ",")

	ports := make([]int, 0)

	for _, rangeStr := range portRanges {
		rangeStr = strings.TrimSpace(rangeStr)
		if rangeStr == "" {
			continue
		}
		parts := strings.SplitN(rangeStr, "-", 2)
		if len(parts) > 2 {
			return nil, fmt.Errorf("invalid port range `%s`", rangeStr)
		}

		if len(parts) == 1 {
			port, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port `%s`: %v", rangeStr, err)
			}
			ports = append(ports, port)
		} else {
			startPort, err := strconv.Atoi(parts[0])
			if err != nil {
				return nil, fmt.Errorf("invalid port `%s`: %v", port, err)
			}
			endPort, err := strconv.Atoi(parts[1])
			if err != nil {
				return nil, fmt.Errorf("invalid port `%s`: %v", port, err)
			}
			if startPort > endPort {
				return nil, fmt.Errorf("invalid port range `%s`: startPort must be less than or equal to endPort", rangeStr)
			}

			for i := startPort; i <= endPort; i++ {
				ports = append(ports, i)
			}
		}

		for _, port := range ports {
			if port < 0 || port > 65535 {
				return nil, fmt.Errorf("invalid port `%d`: port must be between 0 and 65535", port)
			}
		}
	}
	return ports, nil
}

func generateFilename(sessionID string) string {
	currentTime := time.Now().Format("2006-01-02_15:04:05")

	return fmt.Sprintf("%s_%s", currentTime, sessionID)
}
