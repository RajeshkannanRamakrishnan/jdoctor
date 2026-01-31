package scanner

import (
	"crypto/tls"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type SSLInfo struct {
	CommonName      string
	Issuer          string
	Expiry          time.Time
	DaysRemaining   int
	TrustedByGo     bool
	TrustedByJava   bool
	JavaError       string
	GoError         string
}

func ScanSSL(host string) (*SSLInfo, error) {
	info := &SSLInfo{}
	
	// Ensure host defines a port, default to 443
	target := host
	if !hasPort(host) {
		target = host + ":443"
	}

	// 1. Go Native Check
	conn, err := tls.Dial("tcp", target, nil)
	if err != nil {
		info.GoError = err.Error()
		info.TrustedByGo = false
	} else {
		defer conn.Close()
		state := conn.ConnectionState()
		cert := state.PeerCertificates[0]
		
		info.CommonName = cert.Subject.CommonName
		info.Issuer = cert.Issuer.CommonName
		info.Expiry = cert.NotAfter
		info.DaysRemaining = int(time.Until(cert.NotAfter).Hours() / 24)
		info.TrustedByGo = true // If Dial succeeded without InsecureSkipVerify
	}

	// 2. Java Trust Check
	trusted, javaErr := checkJavaTrust(host)
	info.TrustedByJava = trusted
	if javaErr != nil {
		info.JavaError = javaErr.Error()
	}

	return info, nil
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
