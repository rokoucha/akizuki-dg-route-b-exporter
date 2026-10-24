package MB_RL7023_11

import "testing"

func TestNewEVENT(t *testing.T) {
	for _, tc := range []struct {
		line   string
		num    EVENTNum
		result string
	}{
		{"EVENT 25 FE80:0000:0000:0000:0000:0000:0000:0001", EVENTNumPANAConnected, ""},
		{"EVENT 21 FE80:0000:0000:0000:0000:0000:0000:0001 00", EVENTNumUDPSent, EVENTUDPSentSuccess},
		{"EVENT 21 FE80:0000:0000:0000:0000:0000:0000:0001 00 01", EVENTNumUDPSent, EVENTUDPSentFailure},
		{"EVENT 32 FE80:0000:0000:0000:0000:0000:0000:0001", EVENTNumTransmissionRateLimitExceeded, ""},
	} {
		e, err := NewEVENT(tc.line)
		if err != nil {
			t.Fatalf("NewEVENT(%q): %v", tc.line, err)
		}
		if e.Num != tc.num || e.Result() != tc.result {
			t.Errorf("NewEVENT(%q) = %+v, result %q", tc.line, e, e.Result())
		}
	}
}
