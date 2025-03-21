package gateway

import (
	"context"
	"shared/core"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

type ScanICMPReq struct {
	IP      string
	Timeout time.Duration
}

type ScanICMPRes struct {
	IP           string
	Timestamp    time.Time
	Protocol     string
	Status       string
	ResponseTime float64
	SNMPData     string
}

type ScanICMP = core.ActionHandler[ScanICMPReq, ScanICMPRes]

func ImplScanICMP() ScanICMP {
	return func(ctx context.Context, req ScanICMPReq) (*ScanICMPRes, error) {

		result := ScanICMPRes{
			IP:        req.IP,
			Timestamp: time.Now(),
			Protocol:  "ICMP",
			Status:    "Failed",
		}

		// logger.Info("Memulai scan ICMP untuk %s", ip)

		pinger, err := probing.NewPinger(req.IP)
		if err != nil {
			// logger.Error("Gagal membuat pinger untuk %s: %v", ip, err)
			return nil, err
		}

		pinger.Count = 3
		pinger.Timeout = req.Timeout
		pinger.SetPrivileged(true)

		if err = pinger.Run(); err != nil {
			return nil, err
		}

		stats := pinger.Statistics()
		if stats.PacketsRecv > 0 {
			result.Status = "Online"
			result.ResponseTime = float64(stats.AvgRtt) / float64(time.Millisecond)
			// logger.Info("ICMP berhasil untuk %s: response time %.2f ms", ip, result.ResponseTime)
		} else {
			// logger.Info("ICMP gagal untuk %s: tidak ada respons", ip)
		}

		return &result, nil

	}
}
