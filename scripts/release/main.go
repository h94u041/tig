// Command release builds what a release of tigwx hands out, for every system:
// the setup program, an archive with the program alone, and their checksums.
//
//	go run ./scripts/release 0.1.0
//
// Everything lands in dist/. It needs Go and nothing else.
package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

var targets = [][2]string{
	{"windows", "amd64"}, {"windows", "arm64"},
	{"linux", "amd64"}, {"linux", "arm64"},
	{"darwin", "amd64"}, {"darwin", "arm64"},
}

var versionForm = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+([-+][0-9A-Za-z.+-]+)?$`)

const (
	dist       = "dist"
	payloadDir = "cmd/tigwx-setup/payload"
	programPkg = "./cmd/tigwx"
	setupPkg   = "./cmd/tigwx-setup"
)

func main() {
	log.SetFlags(0)
	if len(os.Args) != 2 {
		log.Fatal("usage: go run ./scripts/release <version>")
	}
	version := strings.TrimPrefix(os.Args[1], "v")
	if !versionForm.MatchString(version) {
		log.Fatalf("%q is not a version like 0.1.0", os.Args[1])
	}
	if _, err := os.Stat(payloadDir); err != nil {
		log.Fatal("run this from the root of the tig repository: ", err)
	}
	if err := os.RemoveAll(dist); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(dist, 0o755); err != nil {
		log.Fatal(err)
	}
	for _, t := range targets {
		if err := buildFor(version, t[0], t[1]); err != nil {
			log.Fatalf("%s/%s: %v", t[0], t[1], err)
		}
		fmt.Printf("built %s/%s\n", t[0], t[1])
	}
	if err := writeSums(); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s holds release %s\n", dist, version)
}

func buildFor(version, goos, goarch string) error {
	ext := ""
	if goos == "windows" {
		ext = ".exe"
	}
	program := filepath.Join(payloadDir, "tigwx"+ext)
	defer os.Remove(program) // the source tree holds no program
	if err := goBuild(version, goos, goarch, program, programPkg); err != nil {
		return err
	}
	base := fmt.Sprintf("%s-%s-%s", version, goos, goarch)
	if goos == "windows" {
		if err := writeZip(filepath.Join(dist, "tigwx-"+base+".zip"), program, "tigwx"+ext); err != nil {
			return err
		}
	} else if err := writeTarGz(filepath.Join(dist, "tigwx-"+base+".tar.gz"), program, "tigwx"); err != nil {
		return err
	}
	return goBuild(version, goos, goarch, filepath.Join(dist, "tigwx-setup-"+base+ext), setupPkg)
}

func goBuild(version, goos, goarch, out, pkg string) error {
	cmd := exec.Command("go", "build", "-trimpath", "-buildvcs=false", "-ldflags", "-s -w -X main.version="+version, "-o", out, pkg)
	cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch, "CGO_ENABLED=0")
	cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
	return cmd.Run()
}

func writeZip(file, program, name string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	z := zip.NewWriter(f)
	fi, err := os.Stat(program)
	if err != nil {
		return err
	}
	h, err := zip.FileInfoHeader(fi)
	if err != nil {
		return err
	}
	h.Name, h.Method = name, zip.Deflate
	w, err := z.CreateHeader(h)
	if err != nil {
		return err
	}
	src, err := os.Open(program)
	if err != nil {
		return err
	}
	defer src.Close()
	if _, err := io.Copy(w, src); err != nil {
		return err
	}
	return z.Close()
}

func writeTarGz(file, program, name string) error {
	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	tw := tar.NewWriter(gz)
	fi, err := os.Stat(program)
	if err != nil {
		return err
	}
	h, err := tar.FileInfoHeader(fi, "")
	if err != nil {
		return err
	}
	h.Name, h.Mode = name, 0o755
	if err := tw.WriteHeader(h); err != nil {
		return err
	}
	src, err := os.Open(program)
	if err != nil {
		return err
	}
	defer src.Close()
	if _, err := io.Copy(tw, src); err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	return gz.Close()
}

func writeSums() error {
	entries, err := os.ReadDir(dist)
	if err != nil {
		return err
	}
	var lines []string
	for _, e := range entries {
		f, err := os.Open(filepath.Join(dist, e.Name()))
		if err != nil {
			return err
		}
		h := sha256.New()
		_, err = io.Copy(h, f)
		f.Close()
		if err != nil {
			return err
		}
		lines = append(lines, hex.EncodeToString(h.Sum(nil))+"  "+e.Name())
	}
	slices.Sort(lines)
	return os.WriteFile(filepath.Join(dist, "SHA256SUMS"), []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}
