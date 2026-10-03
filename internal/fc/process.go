package fc

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/exec"
	"syscall"
	"time"
)

const (
	baseurl            = "http://localhost/"
	KernelImagePathKey = "/home/mingda/firecracker/vmlinux.bin"
	BootArgsKey        = "console=ttyS0 reboot=k panic=1 pci=off"
	DriveIdKey         = "rootfs"
	PathOnHostKey      = "/home/mingda/firecracker/rootfs.ext4"
	IsRootDeviceKey    = true
	IsReadOnlyKey      = false
	VcpuCountKey       = 2
	MemoryMibKey       = 256
)

type VM struct {
	Cmd      *exec.Cmd
	SockPath string
	client   *http.Client
	errCh    chan error
}

func NewVM(sockPath string) (*VM, error) {
	os.Remove(sockPath)
	cmd := exec.Command("/home/mingda/firecracker/firecracker", "--api-sock", sockPath)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	err := cmd.Start()
	if err != nil {
		log.Printf("Error starting firecracker: %v", err)
		return nil, err
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
			return nil, err
		}
	}

	timeout := 5 * time.Second
	client := NewUDSClient(sockPath, timeout)
	return &VM{
		Cmd:      cmd,
		SockPath: sockPath,
		client:   client,
		errCh:    errCh,
	}, nil
}

func (v *VM) Boot() error {
	bootSource := &boot_source{
		Kernel_image_path: KernelImagePathKey,
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
	drive := &drive{
		Drive_id:       DriveIdKey,
		Path_on_host:   PathOnHostKey,
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
		log.Printf("Error set drive : %v, code : %v, body : %s", err, code, string(body))
		return err
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
		log.Printf("Error set machine config : %v, body : %s", err, string(body))
		return err
	}
	log.Printf("Machine config set successfully code : %v, hdr : %v", code, hdr)
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
		log.Printf("Error set action : %v, body : %s", err, string(body))
		return err
	}
	log.Printf("Machine started successfully code : %v, hdr : %v", code, hdr)
	return nil
}

func (v *VM) Stop() error {
	defer os.Remove(v.SockPath)
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

func (v *VM) Done() <-chan error {
	return v.errCh
}
