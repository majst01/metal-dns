package main

import (
	"fmt"
	"strings"

	"github.com/majst01/metal-dns/pkg/server"

	"github.com/metal-stack/v"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	moduleName = "metal-dns"
)

var (
	logger *zap.Logger
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
	rootCmd.Flags().StringP("host", "", "localhost", "the host/ip to serve on")
	rootCmd.Flags().IntP("port", "", 50051, "the port to serve on")

	rootCmd.Flags().StringP("ca", "", "certs/ca.pem", "ca path")
	rootCmd.Flags().StringP("cert", "", "certs/server.pem", "server certificate path")
	rootCmd.Flags().StringP("certkey", "", "certs/server-key.pem", "server key path")

	rootCmd.Flags().StringP("secret", "", "secret", "jwt signing secret")

	rootCmd.Flags().StringP("pdns-api-url", "", "http://localhost:8081", "powerdns api url")
	rootCmd.Flags().StringP("pdns-api-password", "", "apipw", "powerdns api password")
	rootCmd.Flags().StringP("pdns-api-vhost", "", "localhost", "powerdns vhost")

	err := viper.BindPFlags(rootCmd.Flags())
	if err != nil {
		logger.Error("unable to construct root command", zap.Error(err))
	}
}

func run() {
	logger, _ = zap.NewProduction()
	defer func() {
		err := logger.Sync() // flushes buffer, if any
		if err != nil {
			fmt.Printf("unable to sync logger buffers:%v", err)
		}
	}()

	config := server.DialConfig{
		Host:   viper.GetString("host"),
		Port:   viper.GetInt("port"),
		Secret: viper.GetString("secret"),

		CA:      viper.GetString("ca"),
		Cert:    viper.GetString("cert"),
		CertKey: viper.GetString("certkey"),

		PdnsApiUrl:      viper.GetString("pdns-api-url"),
		PdnsApiPassword: viper.GetString("pdns-api-password"),
		PdnsApiVHost:    viper.GetString("pdns-api-vhost"),
	}
	s, err := server.NewServer(logger, config)
	if err != nil {
		logger.Fatal("failed to create server %v", zap.Error(err))
	}

	s.StartMetricsAndPprof()
	if err := s.Serve(); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}
}
