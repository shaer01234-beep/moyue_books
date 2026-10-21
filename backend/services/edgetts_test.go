package services

import "testing"

func TestNormalizeEdgeTTSVoice(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", EdgeTTSDefaultVoiceShort},
		{"zh-CN-XiaoxiaoNeural", "zh-CN-XiaoxiaoNeural"},
		{"Microsoft Server Speech Text to Speech Voice (zh-CN, XiaoxiaoNeural)", "zh-CN-XiaoxiaoNeural"},
		{"  Microsoft Server Speech Text to Speech Voice (zh-CN-liaoning, XiaobeiNeural)  ", "zh-CN-liaoning-XiaobeiNeural"},
	}
	for _, c := range cases {
		got := normalizeEdgeTTSVoice(c.in)
		if got != c.want {
			t.Errorf("normalizeEdgeTTSVoice(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
