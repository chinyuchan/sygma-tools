package cmd

import (
	"fmt"
	"log"
	"os"

	"github.com/binance-chain/tss-lib/ecdsa/keygen"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sygma-tools",
	Short: "sygma tools",
	Long:  "sygma tools",
}

var ecdsaCmd = &cobra.Command{
	Use:   "ecdsa",
	Short: "Generate ECDSA local party",
	Long:  "Generate ECDSA local party",
	Run: func(cmd *cobra.Command, args []string) {
		p, err := cmd.Flags().GetInt("participants")
		if err != nil {
			log.Println(err.Error())
			return
		}
		t, err := cmd.Flags().GetInt("threshold")
		if err != nil {
			log.Println(err.Error())
			return
		}
		fmt.Println("Generating local parties...")
		keygen.GenECDSA(t, p)
		fmt.Println("Complete.")
	},
}

func init() {
	rootCmd.AddCommand(ecdsaCmd)
	ecdsaCmd.Flags().IntP("participants", "p", 2, "participants")
	ecdsaCmd.Flags().IntP("threshold", "t", 1, "threshold")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
