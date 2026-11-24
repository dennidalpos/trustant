package main

import (
	"reflect"
	"testing"
)

func TestOpsDevelCommandRunsWithoutShell(t *testing.T) {
	hostile := "/tmp/x$(touch pwned)`id`\nrm"
	for _, tc := range []struct {
		fast bool
		want []string
	}{
		{false, []string{"ops", "ide", "devel"}},
		{true, []string{"ops", "ide", "devel", "--fast"}},
	} {
		cmd := opsDevelCommand(hostile, tc.fast)
		if !reflect.DeepEqual(cmd.Args, tc.want) {
			t.Fatalf("fast=%v: args %q, want %q", tc.fast, cmd.Args, tc.want)
		}
		if cmd.Dir != hostile {
			t.Fatalf("fast=%v: Dir %q, want the workbench path", tc.fast, cmd.Dir)
		}
	}
}
