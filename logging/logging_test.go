package logging

import (
	"os"
	"syscall"
	"testing"
	"path/filepath"
	"bufio"
	"reflect"
	"math/rand"
	"time"
)

var seed = time.Now().UnixNano()
var prng = rand.New(rand.NewSource(seed))

func fork(f func()) (*os.Process, error) {
	r1, _, err := syscall.Syscall(syscall.SYS_FORK, 0, 0, 0)
	if r1 < 0 {
		return nil, err
	}
	if r1 == 0 {
		f()
		os.Exit(0)
	}

	return os.FindProcess(int(r1))
}

func lines(path string) (ls []string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ls = append(ls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return
}

func TestExit(t *testing.T) {
	tmp := t.TempDir()
	stdoutPath := filepath.Join(tmp, "stdout")
	stderrPath := filepath.Join(tmp, "stderr")

	// For portability, the status code should be in the range [0, 125].
	ec := prng.Intn(125+1)

	proc, err := fork(func() {
		stdout, err := os.Create(stdoutPath)
		if err != nil {
			panic(err)
		}

		stderr, err := os.Create(stderrPath)
		if err != nil {
			panic(err)
		}

		cfg := Config {
			HumanWriter: stdout,
			HumanFields: HumanHandlerFields {
				OmitTime: true,
				OmitPID: true,
			},
		}

		logger, _, err := cfg.SetupLogger()
		if err != nil {
			panic(err)
		}

		logger.ExitLevel = LevelError
		logger.ExitWriter = stderr

		logger.Exit(ec, "oops!")
	})
	if err != nil {
		t.Fatal(err)
	}

	st, err := proc.Wait()
	if err != nil {
		t.Fatal(err)
	}

	if st.ExitCode() != ec {
		t.Fatalf("unexpected exit code: %d", st.ExitCode())
	}


	stdout, err := lines(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedStdout := []string {
		"rootmos.io/go-utils/logging.TestExit.func1:logging_test.go:84:ERROR oops!",
	}

	if !reflect.DeepEqual(stdout, expectedStdout) {
		t.Fatalf("unexpected stdout: %v != %v", stdout, expectedStdout)
	}

	stderr, err := lines(stderrPath)
	if err != nil {
		t.Fatal(err)
	}

	expectedstderr := []string {
		"oops!",
	}

	if !reflect.DeepEqual(stderr, expectedstderr) {
		t.Fatalf("unexpected stderr: %v != %v", stderr, expectedstderr)
	}
}
