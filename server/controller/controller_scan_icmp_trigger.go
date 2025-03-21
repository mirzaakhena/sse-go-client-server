package controller

import (
	"net/http"
	"server/usecase"
	"shared/utility"
)

func (c Controller) ScanDevicesTriggerHandler(u usecase.ScanICMPTrigger) utility.APIData {

	apiData := utility.APIData{
		// Access:  model.MANAJEMEN_PENGGUNA_DAFTAR_PENGGUNA_CREATE,
		Method:  http.MethodPost,
		Url:     "/api/scan-devices-trigger",
		Body:    usecase.ScanICMPTriggerReq{},
		Summary: "Scan with ICMP By Range",
		Tag:     "Scan",
	}

	handler := func(w http.ResponseWriter, r *http.Request) {

		body, ok := utility.ParseJSON[usecase.ScanICMPTriggerReq](w, r)
		if !ok {
			return
		}

		utility.HandleUsecase(r.Context(), w, u, body)
	}

	// authorizationHandler := Authorization(handler)
	// authenticatedHandler := Authentication(authorizationHandler, c.JWT)
	// c.Mux.HandleFunc(apiData.GetMethodUrl(), authenticatedHandler)
	c.Mux.HandleFunc(apiData.GetMethodUrl(), handler)

	return apiData
}
