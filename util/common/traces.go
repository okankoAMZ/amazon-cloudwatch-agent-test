package common

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/aws/amazon-cloudwatch-agent-test/util/awsservice"
	"github.com/aws/aws-sdk-go-v2/service/xray/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)



type TraceConfig struct {
	Interval    time.Duration
	Annotations map[string]interface{}
	Metadata    map[string]map[string]interface{}
	Attributes  []attribute.KeyValue
}
type TracesTestInterface interface {
	StartSendingTraces(ctx context.Context) error
	StopSendingTraces()
	Generate(ctx context.Context) error
	validateSegments(t *testing.T, segments []types.Segment, config *TraceConfig)
	GetTestCount() (int, int)
	GetAgentConfigPath() string
	GetGeneratorConfig() *TraceConfig
	GetContext() context.Context
	GetAgentRuntime() time.Duration
	GetName() string
}

func TraceTest(t *testing.T, traceTest TracesTestInterface) error {
	startTime := time.Now()
	t.Logf("AgentConfigpath %s", traceTest.GetAgentConfigPath())
	CopyFile(traceTest.GetAgentConfigPath(), ConfigOutputPath)
	require.NoError(t, StartAgent(ConfigOutputPath, true, false), "Couldn't copy file")
	t.Logf("Successfully copied file.")
	t.Log("Starting to send traces")
	go func() {
		require.NoError(t, traceTest.StartSendingTraces(context.Background()), "load generator exited with error")
	}()
	time.Sleep(traceTest.GetAgentRuntime())
	traceTest.StopSendingTraces()
	t.Logf("Stopped Traces")
	StopAgent()
	t.Logf("Stopped Agent")
	testsGenerated, testsEnded := traceTest.GetTestCount()
	t.Logf("For %s , Test Cases Generated %d | Test Cases Ended: %d", traceTest.GetName(), testsGenerated, testsEnded)
	endTime := time.Now()
	t.Logf("Agent has been running for %s", endTime.Sub(startTime))
	time.Sleep(5 * time.Second)

	traceIDs, err := awsservice.GetTraceIDs(startTime, endTime, awsservice.FilterExpression(
		traceTest.GetGeneratorConfig().Annotations))
	require.NoError(t, err, "unable to get trace IDs")
	segments, err := awsservice.GetSegments(traceIDs)
	require.NoError(t, err, "unable to get segments")

	assert.True(t, len(segments) >= testsGenerated,
		"FAILED: Not enough segments, expected %d but got %d",
		testsGenerated, len(segments))
	require.NoError(t, SegmentValidationTest(t, traceTest, segments), "Segment Validation Failed")
	return nil
}

func SegmentValidationTest(t *testing.T, traceTest TracesTestInterface, segments []types.Segment) error {
	// t.Helper()
	cfg := traceTest.GetGeneratorConfig()
	for _, segment := range segments {
		var result map[string]interface{}
		require.NoError(t, json.Unmarshal([]byte(*segment.Document), &result))
		if _, ok := result["parent_id"]; ok {
			// skip subsegments
			continue
		}
		annotations, ok := result["annotations"]
		assert.True(t, ok, "missing annotations")
		assert.True(t, reflect.DeepEqual(annotations, cfg.Annotations), "mismatching annotations")
		metadataByNamespace, ok := result["metadata"].(map[string]interface{})
		assert.True(t, ok, "missing metadata")
		for namespace, wantMetadata := range cfg.Metadata {
			var gotMetadata map[string]interface{}
			gotMetadata, ok = metadataByNamespace[namespace].(map[string]interface{})
			assert.Truef(t, ok, "missing metadata in namespace: %s", namespace)
			for key, wantValue := range wantMetadata {
				var gotValue interface{}
				gotValue, ok = gotMetadata[key]
				assert.Truef(t, ok, "missing expected metadata key: %s", key)
				assert.Truef(t, reflect.DeepEqual(gotValue, wantValue), "mismatching values for key (%s):\ngot\n\t%v\nwant\n\t%v", key, gotValue, wantValue)
			}
		}
	}
	return nil

}
