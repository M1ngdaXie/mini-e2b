package fc

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
