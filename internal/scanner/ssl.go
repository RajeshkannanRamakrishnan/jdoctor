package scanner

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type CertificateSummary struct {
	Subject string
	Issuer  string
	Expiry  time.Time
}

type SSLInfo struct {
	CommonName      string
	Issuer          string
	Expiry          time.Time
	DaysRemaining   int
	TrustedByGo     bool
	TrustedByJava   bool
	JavaError       string
	GoError         string
	
	// New Fields
	Chain          []CertificateSummary
	RootCAMissing  bool
	Expired        bool
	MITMDetected   bool
	MITMDetails    string
}

func ScanSSL(host string) (*SSLInfo, error) {
	info := &SSLInfo{}
	
	// Ensure host defines a port, default to 443
	target := host
	if !hasPort(host) {
		target = host + ":443"
	}

	// 1. Go Native Check
	// We want to capture the chain even if verification fails, so we might need InsecureSkipVerify for a second pass
	// but let's try standard dial first.
	
	conn, err := tls.Dial("tcp", target, nil)
	if err != nil {
		info.GoError = err.Error()
		info.TrustedByGo = false
		
		// Analyze specific errors
		var unknownAuthErr x509.UnknownAuthorityError
		if errors.As(err, &unknownAuthErr) {
			info.RootCAMissing = true
		}
		
		var certInvalidErr x509.CertificateInvalidError
		if errors.As(err, &certInvalidErr) {
			// This can cover expired, etc. We probably want to inspect the cert to be sure.
			info.Expired = true 
		}

		// Try to connect insecurely just to get the cert chain for diagnosis
		connInsecure, errInsecure := tls.Dial("tcp", target, &tls.Config{InsecureSkipVerify: true})
		if errInsecure == nil {
			defer connInsecure.Close()
			populateCertInfo(info, connInsecure)
		}
	} else {
		defer conn.Close()
		populateCertInfo(info, conn)
		info.TrustedByGo = true 
	}

	// 2. Java Trust Check
	trusted, javaErr := checkJavaTrust(host)
	info.TrustedByJava = trusted
	if javaErr != nil {
		info.JavaError = javaErr.Error()
	}

	// 3. Heuristic MITM Detection
	detectMITM(info)

	return info, nil
}

func populateCertInfo(info *SSLInfo, conn *tls.Conn) {
	state := conn.ConnectionState()
	if len(state.PeerCertificates) > 0 {
		cert := state.PeerCertificates[0]
		info.CommonName = cert.Subject.CommonName
		info.Issuer = cert.Issuer.CommonName
		info.Expiry = cert.NotAfter
		info.DaysRemaining = int(time.Until(cert.NotAfter).Hours() / 24)
		
		if time.Now().After(cert.NotAfter) {
			info.Expired = true
		}
		
		// Populate Chain
		for _, c := range state.PeerCertificates {
			info.Chain = append(info.Chain, CertificateSummary{
				Subject: c.Subject.CommonName,
				Issuer:  c.Issuer.CommonName,
				Expiry:  c.NotAfter,
			})
		}
	}
}

func detectMITM(info *SSLInfo) {
	// Heuristic 1: Common Proxy Issuers
	proxyKeywords := []string{"Zscaler", "BlueCoat", "Netskope", "Fortinet", "Cisco", "Proxy", "Gateway"}
	for _, keyword := range proxyKeywords {
		if strings.Contains(info.Issuer, keyword) {
			info.MITMDetected = true
			info.MITMDetails = fmt.Sprintf("Issuer '%s' contains proxy keyword '%s'", info.Issuer, keyword)
			return
		}
	}

	// Heuristic 2: Trusted by OS but NOT by Java
	// This usually means the corp cert is in the System Keychain (MacOS/Windows) but not in the Java KeyStore.
	if info.TrustedByGo && !info.TrustedByJava {
		info.MITMDetected = true
		info.MITMDetails = "Trusted by OS (Go) but not locally installed Java. Likely a corporate intercepting proxy missing from Java truststore."
		return
	}
}

func hasPort(s string) bool {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return true
		}
	}
	return false
}

func checkJavaTrust(host string) (bool, error) {
	// Simple Java program to connect to the URL
	javaCode := `
import javax.net.ssl.HttpsURLConnection;
import java.net.URL;

public class CheckSSL {
    public static void main(String[] args) {
        try {
            URL url = new URL("https://" + args[0]);
            HttpsURLConnection conn = (HttpsURLConnection) url.openConnection();
            conn.connect();
            System.out.println("OK");
        } catch (Exception e) {
            e.printStackTrace();
            System.exit(1);
        }
    }
}
`
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, "CheckSSL.java")
	if err := os.WriteFile(tmpFile, []byte(javaCode), 0644); err != nil {
		return false, fmt.Errorf("failed to write temp java file: %w", err)
	}
	// defer os.Remove(tmpFile) // Optional: keep for debug? clean up usually good.

	// Run with java 11+ single-file source code support
	// or compile and run for older versions. 
	// Let's assume 'java' is in path and supports single file (Java 11+) which is common now.
	// We can check java version first or just try. 
	
	// Note: We need to strip port if provided for URL construction in Java code above if we strictly use https://host
	// But host argument to this function might be "google.com:443". 
	// URL("https://google.com:443") is valid.
	
	cmd := exec.Command("java", tmpFile, host)
	output, err := cmd.CombinedOutput()
	
	if err != nil {
		// If java fails to run (e.g. older java), we might want to try javac + java
		// For now, let's treat execution failure as "not trusted" or "error"
		return false, fmt.Errorf("java check failed: %s, output: %s", err, string(output))
	}
	
	return true, nil
}
