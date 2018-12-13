package metrics

import "testing"

func TestRunMetricsServices(t *testing.T) {
	type args struct {
		address string
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			RunMetricsServices(tt.args.address)
		})
	}
}
