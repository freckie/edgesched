package edgescoring

type metricsResp struct {
	CPU metricsRespItem `json:"cpu"`
	Mem metricsRespItem `json:"mem"`
}

type metricsRespItem struct {
	Current float64 `json:"current"`
	Future  float64 `json:"future"`
}

type EdgeMetric struct {
	CPUCurrent float64
	CPUFuture  float64
	MemCurrent float64
	MemFuture  float64
}
