package monitor

import (
	"testing"
)

func TestTrendRangeAliasesMatchGeneratedValues(t *testing.T) {
	t.Parallel()

	cases := []GetMonitorServerStatusParamsTrendRange{
		GetMonitorServerStatusParamsTrendRangeN10m,
		GetMonitorServerStatusParamsTrendRangeN30m,
		GetMonitorServerStatusParamsTrendRangeN1h,
	}

	for _, value := range cases {
		switch value {
		case GetMonitorServerStatusParamsTrendRangeN10m,
			GetMonitorServerStatusParamsTrendRangeN30m,
			GetMonitorServerStatusParamsTrendRangeN1h:
		default:
			t.Fatalf("unexpected generated monitor trend range alias: %q", value)
		}
	}
}
