package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

const envdUnit = `[Unit]
Description=mini envd
After=network.target

[Service]
ExecStart=/usr/local/bin/envd
Restart=always

[Install]
WantedBy=multi-user.target
`

func main() {
	rootfs := flag.String("rootfs", "/home/mingda/rootfs.base.ext4", "rootfs image to modify")
	envd := flag.String("envd", "/home/mingda/mini-e2b/envd", "envd binary to install")
	flag.Parse()

	if err := run(*rootfs, *envd); err != nil {
		log.Printf("build-rootfs failed: %v", err)
		os.Exit(1)
	}
	log.Printf("envd installed into %s", *rootfs)
}

func run(rootfs, envd string) error {
	if _, err := os.Stat(envd); err != nil {
		return fmt.Errorf("envd binary: %w", err)
	}

	// Defers run last-in first-out: umount, then remove the mount dir, then make the base read-only again.
	if err := os.Chmod(rootfs, 0644); err != nil {
		return fmt.Errorf("make rootfs writable: %w", err)
	}
	defer func() {
		if err := os.Chmod(rootfs, 0444); err != nil {
			log.Printf("make rootfs read-only: %v", err)
		}
	}()

	mnt, err := os.MkdirTemp("", "rootfs-*")
	if err != nil {
		return fmt.Errorf("create mount dir: %w", err)
	}
	defer os.Remove(mnt)

	if out, err := exec.Command("mount", "-o", "loop", rootfs, mnt).CombinedOutput(); err != nil {
		return fmt.Errorf("mount: %w: %s", err, out)
	}
	defer func() {
		if out, err := exec.Command("umount", mnt).CombinedOutput(); err != nil {
			log.Printf("umount %s: %v: %s", mnt, err, out)
		}
	}()

	if err := copyFile(envd, filepath.Join(mnt, "usr/local/bin/envd"), 0755); err != nil {
		return fmt.Errorf("install envd: %w", err)
	}

	unitPath := filepath.Join(mnt, "etc/systemd/system/envd.service")
	if err := os.WriteFile(unitPath, []byte(envdUnit), 0644); err != nil {
		return fmt.Errorf("write unit: %w", err)
	}

	// Same as `systemctl enable envd`: link the unit into multi-user.target.wants.
	// The link target is the path as the guest sees it.
	wants := filepath.Join(mnt, "etc/systemd/system/multi-user.target.wants")
	if err := os.MkdirAll(wants, 0755); err != nil {
		return fmt.Errorf("create wants dir: %w", err)
	}
	link := filepath.Join(wants, "envd.service")
	if err := os.Remove(link); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove old link: %w", err)
	}
	if err := os.Symlink("/etc/systemd/system/envd.service", link); err != nil {
		return fmt.Errorf("enable unit: %w", err)
	}

	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}
	// OpenFile only applies perm to new files.
	return os.Chmod(dst, perm)
}
