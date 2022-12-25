package main

import (
	"fmt"
	"strings"

	"github.com/majst01/metal-dns/pkg/server"

	"github.com/metal-stack/v"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	moduleName = "metal-dns"
)

var (
	logger *zap.SugaredLogger
)

var rootCmd = &cobra.Command{
	Use:     moduleName,
	Short:   "an api manage dns for metal cloud components",
	Version: v.V.String(),
	Run: func(cmd *cobra.Command, args []string) {
		run()
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.Error("failed executing root command", zap.Error(err))
	}
}

func initConfig() {
	viper.SetEnvPrefix("DNS API")
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.Flags().StringP("http-endpoint", "", "localhost:8080", "the host/ip to serve on")

	rootCmd.Flags().StringP("secret", "", "secret", "jwt signing secret")

	rootCmd.Flags().StringP("pdns-api-url", "", "http://localhost:8081", "powerdns api url")
	rootCmd.Flags().StringP("pdns-api-password", "", "apipw", "powerdns api password")
	rootCmd.Flags().StringP("pdns-api-vhost", "", "localhost", "powerdns vhost")

	rootCmd.Flags().StringP("log-level", "", "info", "log level to use")

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		logger.Error("unable to construct root command", zap.Error(err))
	}
}

func run() {
	logger, _ = createLogger()
	defer func() {
		err := logger.Sync() // flushes buffer, if any
		if err != nil {
			fmt.Printf("unable to sync logger buffers:%v", err)
		}
	}()

	config := server.DialConfig{
		HttpServerEndpoint: viper.GetString("http-endpoint"),
		Secret:             viper.GetString("secret"),

		PdnsApiUrl:      viper.GetString("pdns-api-url"),
		PdnsApiPassword: viper.GetString("pdns-api-password"),
		PdnsApiVHost:    viper.GetString("pdns-api-vhost"),
	}
	s, err := server.New(logger, config)
	if err != nil {
		logger.Fatal("failed to create server %v", zap.Error(err))
	}
	if err := s.Serve(); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}
}

func createLogger() (*zap.SugaredLogger, error) {
	cfg := zap.NewProductionConfig()
	level, err := zap.ParseAtomicLevel(viper.GetString("log-level"))
	if err != nil {
		return nil, err
	}
	cfg.Level = level
	cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zlog, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	logger := zlog.Sugar()
	return logger, nil
}
