package edgescoring

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/freckie/edgesched/apis/config"
	klog "k8s.io/klog/v2"
)

const (
	urlFormat string = "http://%s:%s%s" // {0}=ip, {1}=port, {2}=endpoint
)

type EdgeScoringConnector struct {
	Targets map[string]config.EdgeScoringTarget
}

func NewEdgeScoringConnector(targets []config.EdgeScoringTarget) *EdgeScoringConnector {
	s := &EdgeScoringConnector{
		Targets: make(map[string]config.EdgeScoringTarget),
	}
	for _, t := range targets {
		s.Targets[t.NodeName] = t
	}

	klog.Infof("[EdgeScoring] Targets in EdgeScoringConnector : %v", s.Targets)

	return s
}

func (s *EdgeScoringConnector) GetEdgeMetric(nodeName string) (EdgeMetric, error) {
	result := EdgeMetric{}

	t, ok := s.Targets[nodeName]
	if !ok {
		return result, fmt.Errorf("nodeName \"%s\" not found in EdgeScoringConnector.", nodeName)
	}

	url := fmt.Sprintf(urlFormat, t.IP, t.Port, "/metrics")
	resp, err := http.Get(url)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return result, err
	}

	var jsonResult metricsResp
	if err = json.Unmarshal(body, &jsonResult); err != nil {
		return result, err
	}

	result.CPUCurrent = jsonResult.CPU.Current
	result.CPUFuture = jsonResult.CPU.Future
	result.MemCurrent = jsonResult.Mem.Current
	result.MemFuture = jsonResult.Mem.Future

	return result, nil
}
