package usecase

import (
	"client/gateway"
	"context"
	"fmt"
	"net"
	"shared/core"
	"sync"
	"time"
)

type ScanDevicesReq struct {
	IPRange string `json:"ip_range"`
	Workers int
	TimeOut time.Duration
}

type ScanDevicesRes struct {
	// ScanResults []gateway.ScanICMPRes
}

type ScanDevices = core.ActionHandler[ScanDevicesReq, ScanDevicesRes]

func ImplScanDevices(
	ScanICMP gateway.ScanICMP,
	CallServer gateway.CallServer,
) ScanDevices {
	return func(ctx context.Context, req ScanDevicesReq) (*ScanDevicesRes, error) {

		var result []gateway.ScanICMPRes

		ipList, err := expandIPRange(req.IPRange)
		if err != nil {
			return nil, err
		}

		fmt.Printf("Memulai scan network untuk %d IP dengan %d workers\n", len(ipList), req.Workers)

		var wg sync.WaitGroup
		ipChan := make(chan string, len(ipList))

		// Menambahkan semua IP ke channel
		for _, ip := range ipList {
			ipChan <- ip
		}
		close(ipChan)

		// Memulai worker
		for i := range req.Workers {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				fmt.Printf("Worker %d dimulai\n", id)

				for ip := range ipChan {
					resultScan, err := ScanICMP(ctx, gateway.ScanICMPReq{
						IP:      ip,
						Timeout: req.TimeOut,
					})
					if err != nil {

						fmt.Printf("IP %s error\n", ip)

						result = append(result, *resultScan)

						continue
					}

					result = append(result, *resultScan)

				}

				fmt.Printf("Worker %d selesai\n", id)
			}(i)
		}

		wg.Wait()
		fmt.Printf("Scan network selesai\n")

		if _, err = CallServer(ctx, gateway.CallServerReq{
			Method:  "POST",
			Path:    "/api/scan-devices-result",
			Payload: result,
		}); err != nil {
			return nil, err
		}

		return &ScanDevicesRes{}, nil
	}
}

// Memperluas range IP seperti "192.168.1.1-192.168.1.254" menjadi list IP
func expandIPRange(ipRange string) ([]string, error) {
	var startIP, endIP net.IP

	// Memeriksa apakah input adalah CIDR
	if _, network, err := net.ParseCIDR(ipRange); err == nil {
		fmt.Printf("Memproses range CIDR: %s", ipRange)
		return expandCIDR(network)
	}

	// Memeriksa apakah input adalah IP tunggal
	if ip := net.ParseIP(ipRange); ip != nil {
		fmt.Printf("IP tunggal terdeteksi: %s", ipRange)
		return []string{ipRange}, nil
	}

	// Memeriksa apakah input adalah range IP (format: start-end)
	if _, err := fmt.Sscanf(ipRange, "%s-%s", &startIP, &endIP); err == nil {
		fmt.Printf("Range IP terdeteksi: %s ke %s", startIP, endIP)
		return expandIPStartEnd(startIP.String(), endIP.String())
	}

	return nil, fmt.Errorf("format IP tidak didukung: %s", ipRange)
}

// Memperluas CIDR menjadi list IP
func expandCIDR(network *net.IPNet) ([]string, error) {
	var ipList []string

	// Mendapatkan IP awal
	ip := network.IP.To4()
	if ip == nil {
		return nil, fmt.Errorf("hanya mendukung IPv4")
	}

	// Konversi IP ke uint32 untuk memudahkan increment
	startIP := uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])

	// Mendapatkan mask
	mask := network.Mask
	ones, _ := mask.Size()

	// Menghitung jumlah IP
	hostBits := 32 - ones
	numIPs := uint32(1 << hostBits)

	fmt.Printf("Memperluas CIDR menjadi %d IP", numIPs)

	// Memperluas range
	for i := uint32(0); i < numIPs; i++ {
		currentIP := startIP + i
		ip := net.IPv4(byte(currentIP>>24), byte(currentIP>>16), byte(currentIP>>8), byte(currentIP))
		ipList = append(ipList, ip.String())
	}

	return ipList, nil
}

// Memperluas range IP dari start ke end
func expandIPStartEnd(startIPStr, endIPStr string) ([]string, error) {
	startIP := net.ParseIP(startIPStr).To4()
	endIP := net.ParseIP(endIPStr).To4()

	if startIP == nil || endIP == nil {
		return nil, fmt.Errorf("format IP tidak valid")
	}

	// Konversi IP ke uint32
	start := uint32(startIP[0])<<24 | uint32(startIP[1])<<16 | uint32(startIP[2])<<8 | uint32(startIP[3])
	end := uint32(endIP[0])<<24 | uint32(endIP[1])<<16 | uint32(endIP[2])<<8 | uint32(endIP[3])

	if end < start {
		return nil, fmt.Errorf("IP akhir harus lebih besar dari IP awal")
	}

	numIPs := end - start + 1
	fmt.Printf("Memperluas range IP menjadi %d IP", numIPs)

	var ipList []string
	for i := start; i <= end; i++ {
		ip := net.IPv4(byte(i>>24), byte(i>>16), byte(i>>8), byte(i))
		ipList = append(ipList, ip.String())
	}

	return ipList, nil
}
