package cli

import (
	"fmt"
	"jdoctor/internal/scanner"

	"github.com/spf13/cobra"
)

var sslCmd = &cobra.Command{
	Use:   "ssl [host]",
	Short: "Check SSL certificate for a host",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		successCount := 0
		failCount := 0
		
		for i, host := range args {
			if i > 0 {
				fmt.Println("\n========================================")
			}
			fmt.Printf("🔍 Scanning SSL for %s...\n", host)

			info, err := scanner.ScanSSL(host)
			if err != nil {
				fmt.Printf("Error scanning SSL: %v\n", err)
				failCount++
				continue
			}

			// Report Go Native Status
			if info.TrustedByGo {
				fmt.Println("✔ TLS Connection established (Go Native)")
				fmt.Printf("  Subject: %s\n", info.CommonName)
				fmt.Printf("  Issuer:  %s\n", info.Issuer)
				fmt.Printf("  Expires: %s (%d days remaining)\n", info.Expiry.Format("2006-01-02"), info.DaysRemaining)
			} else {
				fmt.Printf("❌ TLS Connection failed (Go Native): %v\n", info.GoError)
			}

			fmt.Println("-----------")

			// Report Java Status
			if info.TrustedByJava {
				fmt.Println("✔ Trusted by local Java Environment")
				successCount++
			} else {
				fmt.Printf("❌ Not trusted by local Java: %v\n", info.JavaError)
				failCount++ // Considered fail if Java trust fails
			}
		}

		if len(args) > 1 {
			fmt.Println("\n========================================")
			fmt.Printf("Summary: %d passed, %d failed\n", successCount, failCount)
		}
	},
}

func init() {
	rootCmd.AddCommand(sslCmd)
}
