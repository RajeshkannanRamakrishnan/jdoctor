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

var sslDiagnoseCmd = &cobra.Command{
	Use:   "diagnose [host]",
	Short: "Run a detailed SSL diagnosis for a host",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		host := args[0]
		fmt.Printf("🩺 Diagnosing SSL connection to %s...\n\n", host)

		info, err := scanner.ScanSSL(host)
		if err != nil {
			fmt.Printf("❌ Fatal error during scan: %v\n", err)
			return
		}

		// 1. Certificate Chain
		fmt.Println("📜 Certificate Chain:")
		if len(info.Chain) == 0 {
			fmt.Println("   (No certificates received)")
		}
		for i, cert := range info.Chain {
			prefix := "   ├─"
			if i == len(info.Chain)-1 {
				prefix = "   └─"
			}
			fmt.Printf("%s Subject: %s\n", prefix, cert.Subject)
			fmt.Printf("      Issuer:  %s\n", cert.Issuer)
			fmt.Printf("      Expires: %s\n", cert.Expiry.Format("2006-01-02"))
		}
		fmt.Println("")

		// 2. Trust Status
		fmt.Println("🔒 Trust Status:")
		if info.TrustedByGo {
			fmt.Println("   ✔ Trusted by System (Go)")
		} else {
			fmt.Println("   ❌ NOT Trusted by System (Go)")
			fmt.Printf("      Reason: %s\n", info.GoError)
		}

		if info.TrustedByJava {
			fmt.Println("   ✔ Trusted by Java")
		} else {
			fmt.Println("   ❌ NOT Trusted by Java")
			fmt.Printf("      Reason: %s\n", info.JavaError)
		}
		fmt.Println("")

		// 3. Issue Detection
		fmt.Println("🚩 Diagnosis Report:")
		issuesFound := false

		if info.Expired {
			fmt.Println("   [CRITICAL] Certificate is EXPIRED.")
			issuesFound = true
		}
		if info.RootCAMissing {
			fmt.Println("   [CRITICAL] Root CA is missing or unknown.")
			issuesFound = true
		}
		if info.MITMDetected {
			fmt.Println("   [WARNING] Potential Corporate Proxy / MITM Detected!")
			fmt.Printf("   Details: %s\n", info.MITMDetails)
			issuesFound = true
		}

		if !issuesFound {
			if info.TrustedByGo && info.TrustedByJava {
				fmt.Println("   ✔ No issues detected. Connection looks healthy.")
			} else {
				fmt.Println("   ⚠ No specificMITM/Expiration signatures found, but connection is untrusted.")
			}
		}
	},
}

func init() {
	sslCmd.AddCommand(sslDiagnoseCmd)
	rootCmd.AddCommand(sslCmd)
}
