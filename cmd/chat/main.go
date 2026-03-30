// Package main — точка входа P2P gRPC чата.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/nznyx/se-control/internal/app"
)

func main() {
	var config app.Config

	rootCmd := &cobra.Command{
		Use:   "chat",
		Short: "P2P gRPC Chat",
		Long:  "Консольное приложение для обмена текстовыми сообщениями между двумя пирами по протоколу gRPC.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if config.Username == "" {
				return fmt.Errorf("укажите имя пользователя через флаг --name")
			}

			application := app.New(config)
			return application.Run()
		},
	}

	rootCmd.Flags().StringVarP(&config.Username, "name", "n", "", "Имя пользователя (обязательно)")
	_ = rootCmd.MarkFlagRequired("name")
	rootCmd.Flags().StringVarP(&config.PeerAddress, "address", "a", "", "Адрес peer-а для подключения, включая порт (например: localhost:50051) (режим клиента)")
	rootCmd.Flags().IntVarP(&config.Port, "port", "p", 50051, "Порт для запуска сервера (режим сервера)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
