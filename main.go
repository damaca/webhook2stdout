package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/gofiber/fiber/v3"
)

type Source string

const (
	SourceBody    Source = "body"
	SourceHeaders Source = "headers"
	SourceQuery   Source = "query"
	SourceParams  Source = "params"
	SourceMethod  Source = "method"
	SourcePath    Source = "path"
	SourceIP      Source = "ip"
)

type FieldMapping struct {
	From Source `json:"from" yaml:"from"`
	To   string `json:"to" yaml:"to"`
	Root bool   `json:"root" yaml:"root"`
}

type Config struct {
	Port      int            `json:"port" yaml:"port"`
	Route     string         `json:"route" yaml:"route"`
	Pretty    bool           `json:"pretty" yaml:"pretty"`
	LogJSON   bool           `json:"log_json" yaml:"log_json"`
	LogLevel  string         `json:"log_level" yaml:"log_level"`
	AckStatus int            `json:"ack_status" yaml:"ack_status"`
	AckBody   map[string]any `json:"ack_body" yaml:"ack_body"`
	Mappings  []FieldMapping `json:"mappings" yaml:"mappings"`
}

func defaultConfig() Config {
	return Config{
		Port:      8080,
		Route:     "/",
		Pretty:    false,
		LogJSON:   true,
		LogLevel:  "info",
		AckStatus: 200,
		AckBody: map[string]any{
			"ok": true,
		},
		Mappings: []FieldMapping{
			{From: SourceBody, To: "body"},
			{From: SourceHeaders, To: "headers"},
			{From: SourceQuery, To: "query"},
			{From: SourceParams, To: "params"},
			{From: SourceMethod, To: "method"},
			{From: SourcePath, To: "path"},
			{From: SourceIP, To: "ip"},
		},
	}
}

func main() {
	configPath := flag.String("config", "config.yaml", "Path to YAML or JSON config file")
	flag.Parse()

	cfg, err := loadConfig(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}
	if err := validateConfig(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "invalid config: %v\n", err)
		os.Exit(1)
	}

	logger, err := newLogger(cfg.LogJSON, cfg.LogLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid log configuration: %v\n", err)
		os.Exit(1)
	}

	app := fiber.New(fiber.Config{
		ServerHeader: "wh-logger",
		AppName:      "Webhook Logger",
	})

	app.All(cfg.Route, func(c fiber.Ctx) error {
		output, err := buildOutput(c, cfg.Mappings)
		if err != nil {
			logger.Error("failed to build output", "error", err)
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}

		if err := printOutput(output, cfg.Pretty); err != nil {
			logger.Error("failed to write output", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to write output"})
		}

		return c.Status(cfg.AckStatus).JSON(cfg.AckBody)
	})

	addr := fmt.Sprintf(":%d", cfg.Port)
	logger.Debug("listening", "address", addr, "route", cfg.Route)
	if err := app.Listen(addr, fiber.ListenConfig{
		DisableStartupMessage: true,
	}); err != nil {
		logger.Error("server exited", "error", err)
		os.Exit(1)
	}
}

func newLogger(jsonOutput bool, level string) (*slog.Logger, error) {
	logLevel, err := parseLogLevel(level)
	if err != nil {
		return nil, err
	}
	opts := &slog.HandlerOptions{Level: logLevel}
	if jsonOutput {
		return slog.New(slog.NewJSONHandler(os.Stdout, opts)), nil
	}
	return slog.New(slog.NewTextHandler(os.Stdout, opts)), nil
}

func parseLogLevel(level string) (slog.Level, error) {
	switch level {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unsupported log_level %q (use debug, info, warn, or error)", level)
	}
}

func buildOutput(c fiber.Ctx, mappings []FieldMapping) (any, error) {
	output := make(map[string]any, len(mappings))
	var (
		rootValue    any
		hasRootValue bool
	)

	for _, m := range mappings {
		value, err := extractValue(c, m.From)
		if err != nil {
			return nil, fmt.Errorf("mapping %q -> %q: %w", m.From, m.To, err)
		}
		if m.Root {
			obj, ok := value.(map[string]any)
			if ok {
				if hasRootValue {
					return nil, fmt.Errorf("mapping %q cannot merge object root when non-object root already set", m.From)
				}
				if err := mergeRootObject(output, obj); err != nil {
					return nil, fmt.Errorf("mapping %q as root: %w", m.From, err)
				}
				continue
			}

			if len(output) > 0 {
				return nil, fmt.Errorf("mapping %q cannot set non-object root when object output already exists", m.From)
			}
			if hasRootValue {
				return nil, fmt.Errorf("mapping %q cannot set non-object root more than once", m.From)
			}
			rootValue = value
			hasRootValue = true
			continue
		}
		if hasRootValue {
			return nil, fmt.Errorf("mapping %q cannot add keyed fields when non-object root is already set", m.From)
		}

		output[m.To] = value
	}

	if hasRootValue {
		return rootValue, nil
	}
	return output, nil
}

func mergeRootObject(dst map[string]any, obj map[string]any) error {
	for k, v := range obj {
		if _, exists := dst[k]; exists {
			return fmt.Errorf("root key collision on %q", k)
		}
		dst[k] = v
	}

	return nil
}

func extractValue(c fiber.Ctx, source Source) (any, error) {
	switch source {
	case SourceBody:
		return parseBody(c.Body())
	case SourceHeaders:
		return c.GetReqHeaders(), nil
	case SourceQuery:
		return c.Queries(), nil
	case SourceParams:
		return routeParams(c), nil
	case SourceMethod:
		return c.Method(), nil
	case SourcePath:
		return c.Path(), nil
	case SourceIP:
		return c.IP(), nil
	default:
		return nil, fmt.Errorf("unsupported source %q", source)
	}
}

func routeParams(c fiber.Ctx) map[string]string {
	params := map[string]string{}
	route := c.Route()
	if route == nil {
		return params
	}

	for _, key := range route.Params {
		params[key] = c.Params(key)
	}

	return params
}

func parseBody(raw []byte) (any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}

	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return string(raw), nil
	}

	return parsed, nil
}

func printOutput(payload any, pretty bool) error {
	var (
		b   []byte
		err error
	)

	if pretty {
		b, err = json.MarshalIndent(payload, "", "  ")
	} else {
		b, err = json.Marshal(payload)
	}
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(os.Stdout, string(b))
	return err
}
