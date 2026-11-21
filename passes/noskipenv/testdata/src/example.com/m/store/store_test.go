package store

import (
	"os"
	"os/exec"
	"runtime"
	"testing"
)

func TestDirect(t *testing.T) {
	if os.Getenv("STORE_DSN") == "" {
		t.Skip("no database") // want `silent test skip`
	}
}

func TestViaVariable(t *testing.T) {
	dsn := os.Getenv("STORE_DSN")
	if dsn == "" {
		t.Skipf("no database %s", dsn) // want `silent test skip`
	}
}

func TestLookup(t *testing.T) {
	_, ok := os.LookupEnv("STORE_DSN")
	if !ok {
		t.SkipNow() // want `silent test skip`
	}
}

func TestElse(t *testing.T) {
	if dsn := os.Getenv("STORE_DSN"); dsn != "" {
		_ = dsn
	} else {
		t.Skip("no database") // want `silent test skip`
	}
}

func helper(tb testing.TB) {
	if os.Getenv("STORE_DSN") == "" {
		tb.Skip("no database") // want `silent test skip`
	}
}

func TestHelper(t *testing.T) { helper(t) }

func TestFails(t *testing.T) {
	if os.Getenv("STORE_DSN") == "" {
		t.Fatal("STORE_DSN is required")
	}
}

func TestShort(t *testing.T) {
	if testing.Short() || os.Getenv("STORE_DSN") == "" {
		t.Skip("short mode")
	}
}

func TestOptOut(t *testing.T) {
	if os.Getenv("STORE_SKIP_INTEGRATION") != "" {
		t.Skip("opted out")
	}
}

const skipKey = "STORE_SKIP"

func TestOptOutConst(t *testing.T) {
	if os.Getenv(skipKey) == "1" {
		t.Skip("opted out")
	}
}

func TestPlatform(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("linux only")
	}
}

func TestSkipperWord(t *testing.T) {
	if os.Getenv("STORE_SKIPPER") != "" {
		t.Skip("not an opt-out token") // want `silent test skip`
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		t.Skip("runs only as a child process")
	}
}

func TestUnconditional(t *testing.T) {
	t.Skip("flaky") // want `silent test skip`
}

func TestMissingTool(t *testing.T) {
	if _, err := exec.LookPath("psql"); err != nil {
		t.Skipf("psql is not installed: %v", err) // want `silent test skip`
	}
}

func TestOptOutVariable(t *testing.T) {
	skip := os.Getenv("STORE_SKIP_TOOLS")
	if skip != "" {
		t.Skip("opted out")
	}
}

func TestNestedUnderShort(t *testing.T) {
	if testing.Short() {
		if _, err := exec.LookPath("psql"); err != nil {
			t.Skip("short mode without psql")
		}
	}
}

func TestArch(t *testing.T) {
	if runtime.GOARCH == "arm" {
		t.Skip("32-bit arm")
	}
}
