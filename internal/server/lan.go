package server

import (
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"rsc.io/qr"
)

// GetLANIPs returns non-loopback IPv4 addresses on physical LAN interfaces,
// filtering out VPN/tunnel/virtual interfaces (e.g. utun, docker, vmnet).
func GetLANIPs() []string {
	var primary, secondary []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if isVirtualInterface(iface.Name) {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP.To4()
			if ip == nil || ip.IsLoopback() {
				continue
			}
			s := ip.String()
			if isPrivateLAN(ip) {
				if isPhysicalLANName(iface.Name) {
					primary = append(primary, s)
				} else {
					secondary = append(secondary, s)
				}
			}
		}
	}
	return append(primary, secondary...)
}

func isVirtualInterface(name string) bool {
	low := strings.ToLower(name)
	prefixes := []string{"utun", "tun", "tap", "ppp", "wg", "tailscale", "docker", "veth", "br-", "vmnet", "vboxnet", "bridge", "awdl", "llw", "gif", "stf"}
	for _, p := range prefixes {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}

func isPhysicalLANName(name string) bool {
	low := strings.ToLower(name)
	if low == "en0" || low == "en1" {
		return true
	}
	for _, p := range []string{"eth", "wlan", "wi-fi", "wifi", "ethernet"} {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return false
}

func isPrivateLAN(ip net.IP) bool {
	if ip[0] == 10 {
		return true
	}
	if ip[0] == 192 && ip[1] == 168 {
		return true
	}
	if ip[0] == 172 && ip[1] >= 16 && ip[1] <= 31 {
		return true
	}
	return false
}

// PrintBanner displays the local/LAN URLs and a terminal QR code for zero-install phone joining.
func PrintBanner(port int, ips []string) {
	fmt.Println("==========================================")
	fmt.Println("  GO-STOP LAN SERVER")
	fmt.Println("==========================================")
	fmt.Printf("  Local:   http://localhost:%d\n", port)
	if len(ips) > 0 {
		for i, ip := range ips {
			label := "LAN:    "
			if i > 0 {
				label = "        "
			}
			fmt.Printf("  %s http://%s:%d\n", label, ip, port)
		}
		primaryURL := fmt.Sprintf("http://%s:%d", ips[0], port)
		fmt.Println("\n  Scan with your phone camera to join:")
		qrterminal.GenerateHalfBlock(primaryURL, qrterminal.L, os.Stdout)
	} else {
		fmt.Println("  LAN:     (no LAN IP detected; check Wi-Fi)")
	}
	fmt.Println("==========================================")
}

// ServeQR serves a PNG QR code for the given default URL (or ?url= override).
func ServeQR(defaultURL string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		target := r.URL.Query().Get("url")
		if target == "" {
			target = defaultURL
		}
		code, err := qr.Encode(target, qr.L)
		if err != nil {
			http.Error(w, "QR unavailable", http.StatusInternalServerError)
			return
		}
		code.Scale = 6
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write(code.PNG())
	}
}

// OpenBrowser launches the default web browser to the given URL across macOS, Windows, and Linux.
func OpenBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
