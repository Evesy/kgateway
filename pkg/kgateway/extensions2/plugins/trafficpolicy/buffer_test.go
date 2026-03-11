package trafficpolicy

import (
	"testing"

	bufferv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/buffer/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/kgateway-dev/kgateway/v2/api/v1alpha1/kgateway"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/filters"
	"github.com/kgateway-dev/kgateway/v2/pkg/pluginsdk/ir"
)

func TestBufferIREquals(t *testing.T) {
	tests := []struct {
		name string
		a, b *kgateway.Buffer
		want bool
	}{
		{
			name: "both nil are equal",
			want: true,
		},
		{
			name: "non-nil and not equal",
			a: &kgateway.Buffer{
				MaxRequestSize: new(resource.MustParse("1Ki")),
			},
			b: &kgateway.Buffer{
				MaxRequestSize: new(resource.MustParse("2Ki")),
			},
			want: false,
		},
		{
			name: "non-nil and equal",
			a: &kgateway.Buffer{
				MaxRequestSize: new(resource.MustParse("1Ki")),
			},
			b: &kgateway.Buffer{
				MaxRequestSize: new(resource.MustParse("1Ki")),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := assert.New(t)

			aOut := &trafficPolicySpecIr{}
			constructBuffer(kgateway.TrafficPolicySpec{
				Buffer: tt.a,
			}, aOut)

			bOut := &trafficPolicySpecIr{}
			constructBuffer(kgateway.TrafficPolicySpec{
				Buffer: tt.b,
			}, bOut)

			a.Equal(tt.want, aOut.buffer.Equals(bOut.buffer))
		})
	}
}

func TestBufferFilterRunsBeforeTransformation(t *testing.T) {
	t.Run("classic transformation", func(t *testing.T) {
		plugin := &trafficPolicyPluginGwPass{
			setTransformationInChain: map[string]bool{
				"test-filter-chain": true,
			},
			bufferInChain: map[string]*bufferv3.Buffer{
				"test-filter-chain": {
					MaxRequestBytes: &wrapperspb.UInt32Value{Value: 1024},
				},
			},
		}

		fcc := ir.FilterChainCommon{FilterChainName: "test-filter-chain"}
		httpFilters, err := plugin.HttpFilters(ir.HttpFiltersContext{}, fcc)
		require.NoError(t, err)

		bufferIdx := -1
		transformationIdx := -1
		var bufferStage filters.FilterStage[filters.WellKnownFilterStage]
		var transformationStage filters.FilterStage[filters.WellKnownFilterStage]
		for i, stagedFilter := range httpFilters {
			switch stagedFilter.Filter.GetName() {
			case bufferFilterName:
				bufferIdx = i
				bufferStage = stagedFilter.Stage
			case transformationFilterNamePrefix:
				transformationIdx = i
				transformationStage = stagedFilter.Stage
			}
		}

		require.NotEqual(t, -1, bufferIdx, "buffer filter should be present in the chain")
		require.NotEqual(t, -1, transformationIdx, "transformation filter should be present in the chain")
		assert.Equal(t, -1, filters.FilterStageComparison(bufferStage, transformationStage), "buffer stage must be earlier than transformation stage")
		assert.Less(t, bufferIdx, transformationIdx, "buffer filter must run before transformation")
	})

	t.Run("rustformation", func(t *testing.T) {
		previousUseRustformations := useRustformations
		useRustformations = true
		defer func() {
			useRustformations = previousUseRustformations
		}()

		plugin := &trafficPolicyPluginGwPass{
			setTransformationInChain: map[string]bool{
				"test-filter-chain": true,
			},
			bufferInChain: map[string]*bufferv3.Buffer{
				"test-filter-chain": {
					MaxRequestBytes: &wrapperspb.UInt32Value{Value: 1024},
				},
			},
		}

		fcc := ir.FilterChainCommon{FilterChainName: "test-filter-chain"}
		httpFilters, err := plugin.HttpFilters(ir.HttpFiltersContext{}, fcc)
		require.NoError(t, err)

		bufferIdx := -1
		transformationIdx := -1
		var bufferStage filters.FilterStage[filters.WellKnownFilterStage]
		var transformationStage filters.FilterStage[filters.WellKnownFilterStage]
		for i, stagedFilter := range httpFilters {
			switch stagedFilter.Filter.GetName() {
			case bufferFilterName:
				bufferIdx = i
				bufferStage = stagedFilter.Stage
			case transformationFilterNamePrefix:
				transformationIdx = i
				transformationStage = stagedFilter.Stage
			}
		}

		require.NotEqual(t, -1, bufferIdx, "buffer filter should be present in the chain")
		require.NotEqual(t, -1, transformationIdx, "transformation filter should be present in the chain")
		assert.Equal(t, -1, filters.FilterStageComparison(bufferStage, transformationStage), "buffer stage must be earlier than transformation stage")
		assert.Less(t, bufferIdx, transformationIdx, "buffer filter must run before transformation/rustformation")
	})
}
