package tests

import (
	"strings"
	"testing"

	"zckyachmd/lifeline/pkg/jailer"
)

func TestResolverRejectsTraversal(t *testing.T) {
	j, err := jailer.New("/tmp/sandbox")
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if _, err := j.Resolve("../etc/passwd"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestResolverWithin(t *testing.T) {
	j, _ := jailer.New("/tmp/sandbox")
	p, _ := j.Resolve("file.txt")
	if !j.Within(p) {
		t.Fatalf("expected within sandbox")
	}
	if j.Within("/etc/passwd") {
		t.Fatalf("should not be within sandbox")
	}
}

func TestWriteFileLimit(t *testing.T) {
	j, _ := jailer.New(t.TempDir())
	data := strings.NewReader(strings.Repeat("a", 10))
	if _, err := j.WriteFile("ok.txt", data, 5); err == nil {
		t.Fatalf("expected size limit error")
	}
}
