package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"
)

const (
	sockPath           = "/tmp/fc.sock"
	timeout            = 5 * time.Second
	KernelImagePathKey = "/home/mingda/firecracker/vmlinux.bin"
	BootArgsKey        = "console=ttyS0 reboot=k panic=1 pci=off"
	DriveIdKey         = "rootfs"
	PathOnHostKey      = "/home/mingda/firecracker/rootfs.ext4"
	IsRootDeviceKey    = true
	IsReadOnlyKey      = false
	VcpuCountKey       = 2
	MemoryMibKey       = 256
)

type boot_source struct {
	Kernel_image_path string `json:"kernel_image_path"`
	Boot_args         string `json:"boot_args"`
}

type drive struct {
	Drive_id       string `json:"drive_id"`
	Path_on_host   string `json:"path_on_host"`
	Is_root_device bool   `json:"is_root_device"`
	Is_read_only   bool   `json:"is_read_only"`
}
type machine_config struct {
	VcpuCount int `json:"vcpu_count"`
	MemoryMib int `json:"mem_size_mib"`
}
type action struct {
	ActionType string `json:"action_type"`
}

func NewUDSClient(sockPath string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", sockPath)
			},
		},
	}
}

func UDSRequest(client *http.Client, method, url string, headers map[string]string, body []byte) (int, http.Header, []byte, error) {

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, nil, err
	}
	// if resp.StatusCode != 204 {
	// 	return 0, nil, nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, respBody)
	// }
	return resp.StatusCode, resp.Header, respBody, nil
}

func main() {
	baseurl := "http://localhost/"
	os.Remove(sockPath)
	cmd := exec.Command("/home/mingda/firecracker/firecracker", "--api-sock", sockPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Start()
	if err != nil {
		log.Fatalf("Error starting firecracker: %v", err)
		return
	}
	errCh := make(chan error, 1)
	go func() {
		err := cmd.Wait()
		errCh <- err
	}()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
outer:
	for {
		select {
		case <-ticker.C:
			if _, err := os.Stat(sockPath); err == nil {
				break outer
			}
		case err := <-errCh:
			log.Printf("Error connecting to socket: %v", err)
			log.Printf("Stderr : %s", stderr.String())
			return
		}
	}

	client := NewUDSClient(sockPath, timeout)

	//step one
	bootSource := &boot_source{
		Kernel_image_path: KernelImagePathKey,
		Boot_args:         BootArgsKey,
	}
	b, err := json.Marshal(bootSource)
	if err != nil {
		log.Fatalf("Error setting boot source err: %v", err)
	}
	log.Printf("Marshaled boot source : %s", b)
	var body []byte
	code, hdr, body, err := UDSRequest(client,
		"PUT",
		baseurl+"boot-source",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil {
		log.Fatalf("Error set boot source : %v, code : %v, body : %s", err, code, string(body))
		return
	}
	log.Printf("Boot source set successfully code : %v, hdr : %v", code, hdr)
	//step two
	drive := &drive{
		Drive_id:       DriveIdKey,
		Path_on_host:   PathOnHostKey,
		Is_root_device: IsRootDeviceKey,
		Is_read_only:   IsReadOnlyKey,
	}
	b, err = json.Marshal(drive)
	if err != nil {
		log.Fatalf("Error marshal drive : %v", err)
		return
	}
	code, hdr, body, err = UDSRequest(client,
		"PUT",
		baseurl+"drives/rootfs",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		log.Fatalf("Error set drive : %v, code : %v, body : %s", err, code, string(body))
		return
	}
	log.Printf("Drive set successfully code : %v, hdr : %v", code, hdr)
	//step three
	machineConfig := &machine_config{
		VcpuCount: VcpuCountKey,
		MemoryMib: MemoryMibKey,
	}
	b, err = json.Marshal(machineConfig)
	if err != nil {
		log.Fatalf("Error marshal machine config : %v", err)
		return
	}
	code, hdr, body, err = UDSRequest(client,
		"PUT",
		baseurl+"machine-config",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		log.Fatalf("Error set machine config : %v, body : %s", err, string(body))
		return
	}
	log.Printf("Machine config set successfully code : %v, hdr : %v", code, hdr)
	//step four
	action := &action{
		ActionType: "InstanceStart",
	}
	b, err = json.Marshal(action)
	if err != nil {
		log.Fatalf("Error marshal action : %v", err)
		return
	}
	code, hdr, body, err = UDSRequest(client,
		"PUT",
		baseurl+"actions",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		log.Fatalf("Error set action : %v, body : %s", err, string(body))
		return
	}
	log.Printf("Machine started successfully code : %v, hdr : %v", code, hdr)
	signCh := make(chan os.Signal, 1)
	signal.Notify(signCh, os.Interrupt, syscall.SIGTERM)
out:
	for {
		select {
		case <-signCh:
			log.Println("Received interrupt signal, stopping...")
			cmd.Process.Signal(syscall.SIGKILL)
			os.Remove(sockPath)
			break out
		case err := <-errCh:
			log.Fatalf("Error : %v", err)
		}
	}
}
