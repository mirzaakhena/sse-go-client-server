package wiring

import (
	"client/controller"
	"client/gateway"
	"client/usecase"
	"shared/utility"
)

func SetupDependency(sseClient *utility.SSEClient) {

	// gateways
	scanICMPImpl := gateway.ImplScanICMP()
	callServerImpl := gateway.ImplCallServer()
	// ...other gateways here...

	// use cases
	scanDevicesImpl := usecase.ImplScanDevices(scanICMPImpl, callServerImpl)
	// ...other usecases here...

	c := controller.Controller{
		SSEClient: sseClient,
	}

	// controllers
	c.HandleScanDevices(scanDevicesImpl)
	// ...other controllers here...

}
