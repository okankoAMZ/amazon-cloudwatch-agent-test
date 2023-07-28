package xray

import (
	"context"
	"errors"
	"time"

	"github.com/aws/amazon-cloudwatch-agent-test/util/common"
	"github.com/aws/aws-xray-sdk-go/xray"
)

var generatorError = errors.New("Generator error")

type XrayTracesGenerator struct {
	common.TracesTestInterface
	cfg                     *common.TraceConfig
	testCasesGeneratedCount int
	testCasesEndedCount     int
	agentConfigPath         string
	agentRuntime            time.Duration
	name                    string
	done                    chan struct{}
}

func (g *XrayTracesGenerator) StartSendingTraces(ctx context.Context) error {
	ticker := time.NewTicker(g.cfg.Interval)
	for {
		select {
		case <-g.done:
			ticker.Stop()
			return nil
		case <-ticker.C:
			if err := g.Generate(ctx); err != nil {
				return err
			}
		}
	}
}
func (g *XrayTracesGenerator) StopSendingTraces() {
	close(g.done)
}
func newLoadGenerator(cfg *common.TraceConfig) *XrayTracesGenerator {
	return &XrayTracesGenerator{
		cfg:                     cfg,
		done:                    make(chan struct{}),
		testCasesGeneratedCount: 0,
		testCasesEndedCount:     0,
	}
}
func (g *XrayTracesGenerator) Generate(ctx context.Context) error {
	rootCtx, root := xray.BeginSegment(ctx, "load-generator")
	g.testCasesGeneratedCount++
	defer func() {
		root.Close(nil)
		g.testCasesEndedCount++
	}()

	for key, value := range g.cfg.Annotations {
		if err := root.AddAnnotation(key, value); err != nil {
			return err
		}
	}

	for namespace, metadata := range g.cfg.Metadata {
		for key, value := range metadata {
			if err := root.AddMetadataToNamespace(namespace, key, value); err != nil {
				return err
			}
		}
	}

	_, subSeg := xray.BeginSubsegment(rootCtx, "with-error")
	defer subSeg.Close(nil)

	if err := subSeg.AddError(generatorError); err != nil {
		return err
	}

	return nil
}

func (g *XrayTracesGenerator) GetTestCount() (int, int) {
	return g.testCasesGeneratedCount, g.testCasesEndedCount
}

func (g *XrayTracesGenerator) GetAgentConfigPath() string {
	return g.agentConfigPath
}
func (g *XrayTracesGenerator) GetAgentRuntime() time.Duration {
	return g.agentRuntime
}
func (g *XrayTracesGenerator) GetName() string {
	return g.name
}
func (g *XrayTracesGenerator) GetGeneratorConfig() *common.TraceConfig {
	return g.cfg
}
