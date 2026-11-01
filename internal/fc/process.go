package fc

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/M1ngdaXie/mini-e2b/internal/config"
)

const (
	baseurl         = "http://localhost/"
	BootArgsKey     = "console=ttyS0 reboot=k panic=1 pci=off ip=10.200.0.2::10.200.0.1:255.255.255.0::eth0:off"
	DriveIdKey      = "rootfs"
	IsRootDeviceKey = true
	IsReadOnlyKey   = false
	VcpuCountKey    = 2
	MemoryMibKey    = 256
)

var homeDir string

type VM struct {
	Cmd             *exec.Cmd
	SandboxID       string
	SockPath        string
	RootfsPath      string
	SnapShotPath    string
	MemSnapshotPath string
	client          *http.Client
	done            chan struct{}
	GuestMac        string
	Host            string
	Iface_id        string
	IP              string
	exitErr         error
}

func NewVM(cfg config.Config) (*VM, error) {
	id := NewID()
	home, err := os.UserHomeDir()
	homeDir = home
	if err != nil {
		return nil, err
	}
	sockPath := fmt.Sprintf("/tmp/fc-%s.sock", id)
	rootfsPath := filepath.Join(home, "mini-e2b-data", "vms", id)
	logsPath := filepath.Join(home, "mini-e2b-data", "logs", id)
	snapShotpath := filepath.Join(cfg.SnapshotDir, id)
	memSnapshotPath := filepath.Join(cfg.SnapshotDir, id)
	os.Remove(sockPath)
	os.MkdirAll(logsPath, 0755)

	stdoutFile, err := os.OpenFile(filepath.Join(logsPath, "stdout.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	stderrFile, err := os.OpenFile(filepath.Join(logsPath, "stderr.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		stdoutFile.Close()
		return nil, err
	}

	cmd := exec.Command(cfg.FirecrackerBin, "--api-sock", sockPath)
	timeout := 5 * time.Second
	client := NewUDSClient(sockPath, timeout)
	cmd.Stdout = stdoutFile
	cmd.Stderr = stderrFile

	err = cmd.Start()
	if err != nil {
		log.Printf("Error starting firecracker: %v", err)
		stdoutFile.Close()
		stderrFile.Close()
		return nil, err
	}
	done := make(chan struct{})
	v := &VM{
		Cmd:             cmd,
		SandboxID:       id,
		RootfsPath:      rootfsPath,
		SockPath:        sockPath,
		SnapShotPath:    snapShotpath,
		MemSnapshotPath: memSnapshotPath,
		client:          client,
		GuestMac:        "06:00:0A:C8:00:02",
		Host:            "tap0",
		IP:              "10.200.0.2",
		Iface_id:        "my_network0",
		done:            done,
		exitErr:         nil,
	}
	go func() {
		err := cmd.Wait()
		v.exitErr = err
		stdoutFile.Close()
		stderrFile.Close()
		close(done)
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
		case <-done:
			log.Printf("Error connecting to socket: %v", v.exitErr)
			return nil, v.exitErr
		}
	}

	return v, nil
}

func (v *VM) Boot(cfg config.Config) error {
	bootSource := &boot_source{
		Kernel_image_path: cfg.KernelPath,
		Boot_args:         BootArgsKey,
	}
	b, err := json.Marshal(bootSource)
	if err != nil {
		log.Printf("Error setting boot source err: %v", err)
		return err
	}
	log.Printf("Marshaled boot source : %s", b)
	var body []byte
	code, hdr, body, err := UDSRequest(v.client,
		"PUT",
		baseurl+"boot-source",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil {
		log.Printf("Error set boot source : %v, code : %v, body : %s", err, code, string(body))
		return err
	}
	log.Printf("Boot source set successfully code : %v, hdr : %v", code, hdr)
	//step two
	// cp rootfs to v.RootfsPath
	if err := os.MkdirAll(v.RootfsPath, 0755); err != nil {
		log.Printf("Error creating rootfs path : %v", err)
		return err
	}
	dest := filepath.Join(v.RootfsPath, "rootfs.ext4")
	cmd := exec.Command("cp", cfg.BaseRootfsPath, dest)
	if err := cmd.Run(); err != nil {
		log.Printf("Error copying rootfs : %v", err)
		return err
	}
	os.Chmod(dest, 0644)
	drive := &drive{
		Drive_id:       DriveIdKey,
		Path_on_host:   dest,
		Is_root_device: IsRootDeviceKey,
		Is_read_only:   IsReadOnlyKey,
	}
	b, err = json.Marshal(drive)
	if err != nil {
		log.Printf("Error marshal drive : %v", err)
		return err
	}
	code, hdr, body, err = UDSRequest(v.client,
		"PUT",
		baseurl+"drives/rootfs",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error set drive : %v, code : %v, body : %s", err, code, string(body))
	}
	log.Printf("Drive set successfully code : %v, hdr : %v", code, hdr)
	//step three
	machineConfig := &machine_config{
		VcpuCount: VcpuCountKey,
		MemoryMib: MemoryMibKey,
	}
	b, err = json.Marshal(machineConfig)
	if err != nil {
		log.Printf("Error marshal machine config : %v", err)
		return err
	}
	code, hdr, body, err = UDSRequest(v.client,
		"PUT",
		baseurl+"machine-config",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error set machine config : %v, body : %s", err, string(body))
	}
	log.Printf("Machine config set successfully code : %v, hdr : %v", code, hdr)
	ninterface := &NetworkInterface{
		Guest_mac:     v.GuestMac,
		Host_dev_name: v.Host,
		Iface_id:      v.Iface_id,
	}
	b, err = json.Marshal(ninterface)
	if err != nil {
		log.Printf("Error marshal network config : %v", err)
		return err
	}
	code, hdr, body, err = UDSRequest(v.client,
		"PUT",
		baseurl+"network-interfaces/"+v.Iface_id,
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error set network interface : %v, body : %s", err, string(body))
	}
	//step four
	action := &action{
		ActionType: "InstanceStart",
	}
	b, err = json.Marshal(action)
	if err != nil {
		log.Printf("Error marshal action : %v", err)
		return err
	}
	code, hdr, body, err = UDSRequest(v.client,
		"PUT",
		baseurl+"actions",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error set action : %d, body : %s", code, string(body))
	}
	log.Printf("Machine started successfully code : %v, hdr : %v", code, hdr)
	return nil
}

func (v *VM) Stop() error {
	if v.Cmd != nil {
		err := v.Cmd.Process.Signal(syscall.SIGKILL)
		if errors.Is(err, os.ErrProcessDone) {
			log.Println("Process already done")
			return nil
		}
		if err != nil {
			log.Printf("Error kill process : %v", err)
			return err
		}
	}

	return nil
}
func (v *VM) PauseVm() error {
	state := &State{State: "Paused"}
	b, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshal state : %v", err)
		return err
	}
	code, headers, body, err := UDSRequest(v.client,
		"PATCH",
		baseurl+"vm",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error Pause VM : %d, body : %s", code, string(body))
	}
	log.Printf("Machine paused successfully code : %v, hdr : %v", code, headers)
	return nil
}
func (v *VM) ResumeVm() error {
	state := &State{State: "Resumed"}
	b, err := json.Marshal(state)
	if err != nil {
		log.Printf("Error marshal state : %v", err)
		return err
	}
	code, headers, body, err := UDSRequest(v.client,
		"PATCH",
		baseurl+"vm",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error Resume VM : %d, body : %s", code, string(body))
	}
	log.Printf("Machine Resumed successfully code : %v, hdr : %v", code, headers)
	return nil
}
func (v *VM) CreateSnapshot() error {
	if err := os.MkdirAll(v.SnapShotPath, 0755); err != nil {
		log.Printf("Error create snapshot path : %v", err)
		return err
	}
	cmd := exec.Command("cp", filepath.Join(v.RootfsPath, "rootfs.ext4"), v.SnapShotPath)
	if err := cmd.Run(); err != nil {
		log.Printf("Error copy rootfs to snapshot path : %v", err)
		return err
	}
	req := &SnapshotCreateReq{
		Mem_file_path: filepath.Join(v.MemSnapshotPath, "mem"),
		Snapshot_path: filepath.Join(v.SnapShotPath, "state"),
	}
	b, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error marshal snapshot request : %v", err)
		return err
	}
	code, headers, body, err := UDSRequest(v.client,
		"PUT",
		baseurl+"snapshot/create",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error Create Snapshot : %d, body : %s, err : %w ", code, string(body), err)
	}
	log.Printf("Snapshot created successfully code : %v, hdr : %v, body : %s", code, headers, string(body))
	return nil
}
func (v *VM) LoadSnapshot(id string) error {
	memSnapshotPath := filepath.Join(homeDir, "mini-e2b-data", "snapshots", id)
	rootfsPath := filepath.Join(homeDir, "mini-e2b-data", "vms", id)
	log.Printf("SnapshotPath is %s", memSnapshotPath)
	if err := os.MkdirAll(rootfsPath, 0755); err != nil {
		log.Printf("Error create snapshot path : %v", err)
		return err
	}
	dest := filepath.Join(rootfsPath, "rootfs.ext4")
	cmd := exec.Command("cp", filepath.Join(memSnapshotPath, "rootfs.ext4"), dest)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("copy rootfs from snapshot %s: %w (%s)", id, err, output)
	}
	v.RootfsPath = rootfsPath
	req := &SnapshotLoadReq{
		Mem_file_path: filepath.Join(memSnapshotPath, "mem"),
		Snapshot_path: filepath.Join(memSnapshotPath, "state"),
	}
	b, err := json.Marshal(req)
	if err != nil {
		log.Printf("Error marshal snapshot request : %v", err)
		return err
	}
	code, headers, body, err := UDSRequest(v.client,
		"PUT",
		baseurl+"snapshot/load",
		map[string]string{"Content-Type": "application/json"},
		b)
	if err != nil || code != 204 {
		return fmt.Errorf("Error Load Snapshot : %d, body : %s", code, string(body))
	}
	log.Printf("Snapshot loaded successfully code : %v, hdr : %v, body : %s", code, headers, string(body))
	return nil
}
func (v *VM) Done() <-chan struct{} {
	return v.done
}

func (v *VM) ExitErr() error {
	return v.exitErr
}
func NewID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func (v *VM) Cleanup() error {
	err := os.Remove(v.SockPath)
	if err != nil {
		log.Printf("Error remove sock path : %v", err)
		return err
	}
	err = os.RemoveAll(v.RootfsPath)
	if err != nil {
		log.Printf("Error remove rootfs path : %v", err)
		return err
	}
	return nil
}
