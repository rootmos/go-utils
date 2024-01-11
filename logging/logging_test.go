package logging

import (
	"os"
	"syscall"
	"testing"
	"bufio"
	"reflect"
	"math/rand"
	"time"
	"io"
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

func lines(r io.Reader) (ls []string, err error) {
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		ls = append(ls, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return
}

func run(f func()) (st *os.ProcessState, stdout, stderr []string, err error) {
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}

	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		return nil, nil, nil, err
	}

	proc, err := fork(func() {
		os.Stdout = stdoutW
		if err := stdoutR.Close(); err != nil {
			panic(err)
		}

		os.Stderr = stderrW
		if err := stderrR.Close(); err != nil {
			panic(err)
		}

		f()

		if err := stdoutW.Close(); err != nil {
			panic(err)
		}
		if err := stderrW.Close(); err != nil {
			panic(err)
		}
	})
	if err != nil {
		return nil, nil, nil, err
	}

	if err = stdoutW.Close(); err != nil {
		return nil, nil, nil, err
	}
	if err = stderrW.Close(); err != nil {
		return nil, nil, nil, err
	}

	st, err = proc.Wait()
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err = lines(stdoutR)
	if err != nil {
		return nil, nil, nil, err
	}
	if err = stdoutR.Close(); err != nil {
		return nil, nil, nil, err
	}

	stderr, err = lines(stderrR)
	if err != nil {
		return nil, nil, nil, err
	}
	if err = stderrR.Close(); err != nil {
		return nil, nil, nil, err
	}

	return st, stdout, stderr, nil
}

func TestExit(t *testing.T) {
	// For portability, the status code should be in the range [0, 125].
	ec := prng.Intn(125+1)

	st, stdout, stderr, err := run(func() {
		cfg := Config {
			HumanWriter: os.Stdout,
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
		logger.ExitWriter = os.Stderr

		logger.Exit(ec, "oops!")
	})
	if err != nil {
		t.Fatal(err)
	}

	if st.ExitCode() != ec {
		t.Fatalf("unexpected exit code: %d", st.ExitCode())
	}

	expectedStdout := []string {
		"rootmos.io/go-utils/logging.TestExit.func1:logging_test.go:130:ERROR oops!",
	}
	if !reflect.DeepEqual(stdout, expectedStdout) {
		t.Fatalf("unexpected stdout: %v != %v", stdout, expectedStdout)
	}

	expectedstderr := []string {
		"oops!",
	}
	if !reflect.DeepEqual(stderr, expectedstderr) {
		t.Fatalf("unexpected stderr: %v != %v", stderr, expectedstderr)
	}
}

func TestExitWithNilLogger(t *testing.T) {
	// For portability, the status code should be in the range [0, 125].
	ec := prng.Intn(125+1)

	st, stdout, stderr, err := run(func() {
		var logger *Logger
		logger.Exitf(ec, "really bad: %d", 7)
	})
	if err != nil {
		t.Fatal(err)
	}

	if st.ExitCode() != ec {
		t.Fatalf("unexpected exit code: %d", st.ExitCode())
	}

	if stdout != nil {
		t.Fatalf("unexpected stdout: %v", stdout)
	}

	expectedstderr := []string {
		"really bad: 7",
	}
	if !reflect.DeepEqual(stderr, expectedstderr) {
		t.Fatalf("unexpected stderr: %v != %v", stderr, expectedstderr)
	}
}
