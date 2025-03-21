package wiring

import (
	"net/http"
	"server/controller"
	"server/gateway"
	"server/usecase"
	"shared/utility"

	"gorm.io/gorm"
)

func SetupDependency(mux *http.ServeMux, sseServer *utility.SSEServer, apiPrinter *utility.ApiPrinter, db *gorm.DB) {

	// gateways
	// saveClientGw := gateway.ImplClientSaveWithSQlite(db)
	sendSSEMessageGw := gateway.ImplSendSSEMessage(sseServer)
	// ...other gateways here...

	// use cases
	scanDevicesTriggerImpl := usecase.ImplScanICMPTrigger(sendSSEMessageGw)
	// ...other usecases here...

	c := controller.Controller{
		Mux: mux,
	}

	// controllers
	apiPrinter.
		Add(c.ScanDevicesTriggerHandler(scanDevicesTriggerImpl))

	// ...other controllers here...

}
