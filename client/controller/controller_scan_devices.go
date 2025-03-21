package controller

import (
	"client/usecase"
	"context"
	"encoding/json"
	"fmt"
)

func (c *Controller) HandleScanDevices(u usecase.ScanDevices) {

	c.SSEClient.AddEventHandler("scan_icmp", func(data []byte) error {

		var payload usecase.ScanDevicesReq
		if err := json.Unmarshal(data, &payload); err != nil {
			return fmt.Errorf("error parsing request payload: %v", err)
		}

		if _, err := u(context.Background(), payload); err != nil {
			return err
		}

		return nil
	})

}
