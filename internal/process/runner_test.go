package process

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"
)

func TestMergeEnvironment(t *testing.T) {
	got := mergeEnvironment([]string{"PATH=/bin", "Mode=old", "KEEP=yes"}, []string{"MODE=new", "NEW=value"})
	want := []string{"PATH=/bin", "KEEP=yes", "MODE=new", "NEW=value"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestOSRunnerQuietDiscardsChildOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	runner := &OSRunner{Stdout: &stdout, Stderr: &stderr}
	err := runner.Run(context.Background(), Command{
		Name:  os.Args[0],
		Args:  []string{"-test.run=TestOSRunnerHelperProcess"},
		Env:   []string{"KSTANKCLI_TEST_HELPER=1"},
		Quiet: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("quiet command produced stdout %q and stderr %q", stdout.String(), stderr.String())
	}
}

func TestOSRunnerHelperProcess(t *testing.T) {
	if os.Getenv("KSTANKCLI_TEST_HELPER") != "1" {
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, "stdout")
	_, _ = fmt.Fprintln(os.Stderr, "stderr")
}
