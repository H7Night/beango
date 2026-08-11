package main

import "testing"

func TestParseArgs(t *testing.T) {
	cases := []struct {
		name       string
		raw        []string
		wantType   string
		wantOutput string
		wantMerge  bool
		wantArgs   []string
		wantErr    bool
	}{
		{
			name:       "flags before positional",
			raw:        []string{"-type", "alipay", "-output", "./out", "-merge", "file.csv"},
			wantType:   "alipay",
			wantOutput: "./out",
			wantMerge:  true,
			wantArgs:   []string{"file.csv"},
		},
		{
			name:       "flags after positional (bug fix)",
			raw:        []string{"-type", "alipay", "file.csv", "-output", "./out", "-merge"},
			wantType:   "alipay",
			wantOutput: "./out",
			wantMerge:  true,
			wantArgs:   []string{"file.csv"},
		},
		{
			name:       "flags interleaved with positional",
			raw:        []string{"-merge", "-type", "wechat", "a.xlsx", "-output", "./x"},
			wantType:   "wechat",
			wantOutput: "./x",
			wantMerge:  true,
			wantArgs:   []string{"a.xlsx"},
		},
		{
			name:       "equals form",
			raw:        []string{"-type=alipay", "-output=./x", "f.csv"},
			wantType:   "alipay",
			wantOutput: "./x",
			wantArgs:   []string{"f.csv"},
		},
		{
			name:      "no flags",
			raw:       []string{},
			wantArgs:  []string{},
		},
		{
			name:      "missing value",
			raw:       []string{"-type"},
			wantErr:   true,
		},
		{
			name:    "unknown flag",
			raw:     []string{"-badopt"},
			wantErr: true,
		},
		{
			name:    "help",
			raw:     []string{"-h"},
			wantErr: true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gotType, gotOutput, gotMerge, gotArgs, err := parseArgs(c.raw)
			if c.wantErr != (err != nil) {
				t.Fatalf("parseArgs(%v) err = %v, wantErr = %v", c.raw, err, c.wantErr)
			}
			if err != nil {
				return
			}
			if gotType != c.wantType || gotOutput != c.wantOutput || gotMerge != c.wantMerge {
				t.Fatalf("parseArgs(%v) = (%q, %q, %v), want (%q, %q, %v)",
					c.raw, gotType, gotOutput, gotMerge, c.wantType, c.wantOutput, c.wantMerge)
			}
			if len(gotArgs) != len(c.wantArgs) {
				t.Fatalf("parseArgs(%v) args = %v, want %v", c.raw, gotArgs, c.wantArgs)
			}
			for i := range gotArgs {
				if gotArgs[i] != c.wantArgs[i] {
					t.Fatalf("parseArgs(%v) args = %v, want %v", c.raw, gotArgs, c.wantArgs)
				}
			}
		})
	}
}
