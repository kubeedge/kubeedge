package httpserver

import "net/http"

func (rs *RestServer) InitRouter() {
	// Common
	rs.Router.HandleFunc(APIPing, rs.Ping).Methods(http.MethodGet)

	// Device
	rs.Router.HandleFunc(APIDeviceReadRoute, rs.DeviceRead).Methods(http.MethodGet)

	// Meta
	rs.Router.HandleFunc(APIMetaGetModelRoute, rs.MetaGetModel).Methods(http.MethodGet)

	// DataBase
	rs.Router.HandleFunc(APIDataBaseGetDataByID, rs.DataBaseGetDataByID).Methods(http.MethodGet)
}
