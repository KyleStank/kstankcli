package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
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

func TestOSRunnerBackgroundDoesNotWaitForExitStatus(t *testing.T) {
	runner := &OSRunner{Stdin: bytes.NewReader(nil), Stdout: io.Discard, Stderr: io.Discard}
	err := runner.Run(context.Background(), Command{
		Name:       os.Args[0],
		Args:       []string{"-test.run=TestOSRunnerHelperProcess"},
		Env:        []string{"KSTANKCLI_TEST_HELPER=exit"},
		Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestOSRunnerBackgroundReportsStartFailure(t *testing.T) {
	name := filepath.Join(t.TempDir(), "missing-executable")
	err := NewOSRunner().Run(context.Background(), Command{Name: name, Background: true})
	if err == nil || !strings.Contains(err.Error(), "run "+name) {
		t.Fatalf("expected contextual start failure, got %v", err)
	}
}

func TestOSRunnerHelperProcess(t *testing.T) {
	switch os.Getenv("KSTANKCLI_TEST_HELPER") {
	case "exit":
		os.Exit(1)
	case "1":
	default:
		return
	}
	_, _ = fmt.Fprintln(os.Stdout, "stdout")
	_, _ = fmt.Fprintln(os.Stderr, "stderr")
}
