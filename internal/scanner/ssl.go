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
	// Use a unique name to avoid conflicts if running multiple times? 
	// actually for simplicity let's keep CheckSSL.java but maybe in a unique subdir if parallel?
	// For CLI tool, standard temp dir with fixed name is risky if concurrent. 
	// Let's use a temp dir for the file.
	runDir, err := os.MkdirTemp(tmpDir, "jdoctor-ssl-*")
	if err != nil {
		return false, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(runDir) // Clean up everything

	tmpFile := filepath.Join(runDir, "CheckSSL.java")
	if err := os.WriteFile(tmpFile, []byte(javaCode), 0644); err != nil {
		return false, fmt.Errorf("failed to write temp java file: %w", err)
	}

	// 1. Compile: javac CheckSSL.java
	compileCmd := exec.Command("javac", tmpFile)
	if out, err := compileCmd.CombinedOutput(); err != nil {
		return false, fmt.Errorf("javac failed: %s, output: %s", err, string(out))
	}

	// 2. Run: java -cp . CheckSSL <host>
	// Note: We need to strip port from host if provided, but URL constructor handles host:port fine usually.
	runCmd := exec.Command("java", "-cp", runDir, "CheckSSL", host)
	output, err := runCmd.CombinedOutput()
	
	if err != nil {
		return false, fmt.Errorf("java check failed: %s, output: %s", err, string(output))
	}
	
	if !strings.Contains(string(output), "OK") {
		return false, fmt.Errorf("unexpected output from java check: %s", string(output))
	}

	return true, nil
}
