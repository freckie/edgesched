// referenced https://medium.com/@juliorenner123/k8s-creating-a-kube-scheduler-plugin-8a826c486a1

package edgescoring

import (
	"context"
	"fmt"
	"strconv"

	"github.com/freckie/edgesched/apis/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	klog "k8s.io/klog/v2"
	"k8s.io/kubernetes/pkg/scheduler/framework"
)

type EdgeScoring struct {
	handle framework.Handle
	conn   *EdgeScoringConnector
}

const Name = "EdgeScoring"

var _ = framework.ScorePlugin(&EdgeScoring{})
var alpha float64 = 0.5
var theta2 float64 = 0.7
var theta1 float64 = 0.4

func New(obj runtime.Object, h framework.Handle) (framework.Plugin, error) {
	args, ok := obj.(*config.EdgeScoringArgs)
	if !ok {
		return nil, fmt.Errorf("[EdgeScoring] want args to be of type EdgeScoringArgs, got %T", obj)
	}

	klog.Infof("[EdgeScoring] args received. args: %v", args)

	return &EdgeScoring{
		handle: h,
		conn:   NewEdgeScoringConnector(args.Targets),
	}, nil
}

func (s *EdgeScoring) Name() string {
	return Name
}

func (s *EdgeScoring) Score(ctx context.Context, state *framework.CycleState, p *v1.Pod, nodeName string) (int64, *framework.Status) {
	var nodeLabel string
	var score float64
	klog.Infof("[EdgeScoring] Started to edge-node-scoring process")
	klog.Infof("[EdgeScoring] alpha = \"%.3f\"", alpha)
	klog.Infof("[EdgeScoring] theta2 = \"%.3f\"", theta2)
	klog.Infof("[EdgeScoring] theta1 = \"%.3f\"", theta1)

	// getting pod labels
	labels := p.ObjectMeta.Labels
	_cpuRequest, ok := labels["cpu-request"]
	if !ok {
		return 0, framework.NewStatus(framework.Error, "label \"cpu-request\" is not specified.")
	}
	cpuRequest, err := strconv.ParseFloat(_cpuRequest, 64)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, "expected float64 value for \"cpu-request\".")
	}
	_memRequest, ok := labels["mem-request"]
	if !ok {
		return 0, framework.NewStatus(framework.Error, "label \"mem-request\" is not specified.")
	}
	memRequest, err := strconv.ParseFloat(_memRequest, 64)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, "expected float64 value for \"mem-request\".")
	}
	klog.Infof("[EdgeScoring] R_cpu = \"%.2f\"", cpuRequest)
	klog.Infof("[EdgeScoring] R_mem = \"%.2f\"", memRequest)

	// getting node metric
	metric, err := s.conn.GetEdgeMetric(nodeName)
	if err != nil {
		return 0, framework.NewStatus(framework.Error, fmt.Sprintf("error getting edge node metrics: %s", err))
	}
	klog.Infof("[EdgeScoring] system got metrics! \"%v\"", metric)

	// scoring
	value := alpha*metric.CPUFuture + (1-alpha)*metric.MemFuture
	klog.Infof("[EdgeScoring] value for comparison with thresholds : %.5f", value)
	if value >= theta2 {
		nodeLabel = "needs-migration"
		score = 0
	} else if value >= theta1 {
		nodeLabel = "unschedulable"
		score = 0
	} else {
		nodeLabel = "schedulable"
		score = 1 - (alpha * (metric.CPUFuture + cpuRequest)) - (1-alpha)*(metric.MemFuture+memRequest)
		klog.Infof("[EdgeScoring] before-normalized score \"%f\"", score)
	}
	if score < 0 {
		score = 0
	}

	klog.Infof("[EdgeScoring] node \"%s\" label \"%s\"", nodeName, nodeLabel)
	klog.Infof("[EdgeScoring] node \"%s\" final score \"%f\"", nodeName, int64(score*100))
	return int64(score * 100), nil
}

func (s *EdgeScoring) ScoreExtensions() framework.ScoreExtensions {
	return s
}

func (s *EdgeScoring) NormalizeScore(
	ctx context.Context,
	state *framework.CycleState,
	pod *v1.Pod,
	scores framework.NodeScoreList,
) *framework.Status {
	for i, node := range scores {
		scores[i].Score = node.Score
		klog.Infof("[EdgeScoring] Normalizing Scores .. %d", i)
	}
	return nil
}
