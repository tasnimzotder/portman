//go:build darwin

package cli

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
)

var testBinary string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "portman-cli-tests-")
	if err != nil {
		panic(err)
	}
	testBinary = filepath.Join(dir, "portman")
	build := exec.Command("go", "build", "-o", testBinary, "../../cmd/portman")
	if out, err := build.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "build: %v\n%s", err, out)
		os.RemoveAll(dir)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func runCLI(t *testing.T, env []string, args ...string) (string, string, error) {
	t.Helper()
	cmd := exec.Command(testBinary, args...)
	cmd.Env = append(os.Environ(), env...)
	var out, diagnostic bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diagnostic
	err := cmd.Run()
	return out.String(), diagnostic.String(), err
}
func fakeLsof(t *testing.T, fixture, status string) []string {
	t.Helper()
	dir := t.TempDir()
	data := filepath.Join(dir, "data")
	if err := os.WriteFile(data, []byte(fixture), 0600); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\n/bin/cat '" + data + "'\n" + status + "\n"
	if err := os.WriteFile(filepath.Join(dir, "lsof"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return []string{"PATH=" + dir + ":" + os.Getenv("PATH")}
}

func TestEmptyJSONAndDelimitedAcrossCommands(t *testing.T) {
	env := fakeLsof(t, "", "exit 1")
	for _, args := range [][]string{{"--json"}, {"find", "missing", "--json"}, {"pid", "123", "--json"}, {"conflicts", "--json"}, {"34567-34568", "--json"}} {
		out, stderr, err := runCLI(t, env, args...)
		var result model.ScanResult
		if err != nil || json.Unmarshal([]byte(out), &result) != nil || result.Listeners == nil || len(result.Listeners) != 0 {
			t.Fatalf("%v output=%q stderr=%q err=%v", args, out, stderr, err)
		}
	}
	for _, args := range [][]string{{"--format", "csv"}, {"find", "missing", "--format", "csv"}, {"pid", "123", "--format", "csv"}, {"conflicts", "--format", "csv"}, {"34567", "--format", "csv"}} {
		out, stderr, err := runCLI(t, env, args...)
		rows, parseErr := csv.NewReader(strings.NewReader(out)).ReadAll()
		if err != nil || parseErr != nil || len(rows) != 1 || rows[0][0] != "PORT" {
			t.Fatalf("%v rows=%v stderr=%q err=%v", args, rows, stderr, err)
		}
	}
}

func TestWaitDoesNotExecuteAfterTimeoutOrScannerFailure(t *testing.T) {
	for _, status := range []string{"exit 1", "exit 2"} {
		env := fakeLsof(t, "", status)
		marker := filepath.Join(t.TempDir(), "executed")
		args := []string{"wait", "34567", "--quiet", "--timeout", "50ms", "--interval", "10ms", "--exec", "touch '" + marker + "'"}
		if status == "exit 2" {
			args = append(args, "--invert")
		}
		out, stderr, err := runCLI(t, env, args...)
		if err == nil || out != "" || stderr != "" {
			t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
		}
		if _, err := os.Stat(marker); !os.IsNotExist(err) {
			t.Fatal("executed command after failed wait")
		}
	}
}

func TestWaitSuccessPreservesExecExitCode(t *testing.T) {
	env := fakeLsof(t, "", "exit 1")
	out, stderr, err := runCLI(t, env, "wait", "34567", "--invert", "--quiet", "--exec", "exit 7")
	if out != "" || stderr != "" {
		t.Fatalf("quiet exec failure emitted output: %q %q", out, stderr)
	}
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 7 {
		t.Fatalf("exit=%v", err)
	}
}

func TestFlagValidation(t *testing.T) {
	for _, args := range [][]string{{"--tcp", "--udp"}, {"--format", "invalid"}, {"--sort", "invalid"}, {"--watch", "--interval", "0s"}, {"wait", "34567", "--timeout", "0s"}, {"wait", "34567", "--interval", "-1s"}, {"kill", "34567", "--timeout", "0s"}, {"--json", "--format", "csv"}} {
		if _, _, err := runCLI(t, nil, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestDuplicateOwnerDetection(t *testing.T) {
	env := fakeLsof(t, "p111\x00cone\x00Luser\x00\nf3\x00tIPv4\x00PTCP\x00n127.0.0.1:34567\x00TST=LISTEN\x00\np222\x00ctwo\x00Luser\x00\nf3\x00tIPv4\x00PTCP\x00n127.0.0.2:34567\x00TST=LISTEN\x00\n", "exit 0")
	out, stderr, err := runCLI(t, env, "conflicts", "--json")
	var result model.ScanResult
	if err != nil || json.Unmarshal([]byte(out), &result) != nil || len(result.Listeners) != 2 {
		t.Fatalf("out=%q stderr=%q err=%v", out, stderr, err)
	}
	_, stderr, err = runCLI(t, env, "kill", "34567", "--yes")
	if err == nil || !strings.Contains(stderr, "multiple owners") {
		t.Fatalf("ambiguous kill: stderr=%q err=%v", stderr, err)
	}
}

// A disposable server lets CLI tests signal a real process without touching services.
func TestServerHelper(t *testing.T) {
	if os.Getenv("PORTMAN_TEST_SERVER") != "1" {
		return
	}
	signal.Ignore(syscall.SIGTERM, syscall.SIGHUP)
	l, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		os.Exit(2)
	}
	fmt.Println(l.Addr().(*net.TCPAddr).Port)
	for {
		c, err := l.Accept()
		if err != nil {
			os.Exit(0)
		}
		c.Close()
	}
}
func startServer(t *testing.T) (int, *exec.Cmd) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "-test.run=^TestServerHelper$")
	cmd.Env = append(os.Environ(), "PORTMAN_TEST_SERVER=1")
	pipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stderr = os.Stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(pipe)
	line, err := reader.ReadString('\n')
	if err != nil {
		cmd.Process.Kill()
		cmd.Wait()
		t.Fatal(err)
	}
	port, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { cmd.Wait(); close(done) }()
	t.Cleanup(func() { cmd.Process.Kill(); <-done })
	return port, cmd
}
func TestKillReloadTimeoutAndEscalation(t *testing.T) {
	port, server := startServer(t)
	p := strconv.Itoa(port)
	if out, stderr, err := runCLI(t, nil, "kill", p, "--signal", "HUP", "--yes"); err != nil {
		t.Fatalf("reload out=%q stderr=%q err=%v", out, stderr, err)
	}
	if err := server.Process.Signal(syscall.Signal(0)); err != nil {
		t.Fatal("reload killed server")
	}
	start := time.Now()
	if _, _, err := runCLI(t, nil, "kill", p, "--yes", "--timeout", "150ms"); err == nil {
		t.Fatal("ignored TERM reported success")
	}
	if elapsed := time.Since(start); elapsed > 3*time.Second {
		t.Fatalf("kill ignored configured timeout: %s", elapsed)
	}
	if out, stderr, err := runCLI(t, nil, "kill", p, "--force", "--yes", "--timeout", "150ms"); err != nil {
		t.Fatalf("escalation out=%q stderr=%q err=%v", out, stderr, err)
	}
	if err := server.Process.Signal(syscall.Signal(0)); err == nil {
		t.Fatal("force did not kill server")
	}
}
