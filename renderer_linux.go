package main

import (
	"os"
	"path/filepath"
	"regexp"
)

// configureRenderer chooses WebKitGTK's renderer before the app starts.
//
// MyGo turns WebKitGTK's DMA-BUF renderer off everywhere, because it shows
// blank windows on some setups (NVIDIA's own driver, virtual machines,
// containers). Without it WebKit composites in software and draws no
// backdrop-filter, so every material of the interface turns into plain
// see-through. The app turns the renderer back on where it is known to work, and
// tells the page when it is off, which then draws its materials opaque.
func configureRenderer() (blur bool) {
	if v, ok := os.LookupEnv("WEBKIT_DISABLE_DMABUF_RENDERER"); ok {
		return v == "0" // the user decided
	}
	if risky() {
		return false // MyGo's default stays
	}
	_ = os.Setenv("WEBKIT_DISABLE_DMABUF_RENDERER", "0")
	return true
}

var vmProduct = regexp.MustCompile(`(?i)virtualbox|vmware|qemu|kvm|bochs|parallels|hyper-v|virtual machine|xen`)

func risky() bool {
	// NVIDIA's own driver is the trouble when it is the only GPU; on a laptop that pairs it with an integrated
	// GPU, WebKit renders on the integrated one, which works
	drivers := renderDrivers()
	if len(drivers) == 0 {
		return true // no GPU to render with
	}
	if _, err := os.Stat("/proc/driver/nvidia/version"); err == nil {
		other := false
		for _, d := range drivers {
			if d != "nvidia" {
				other = true
			}
		}
		if !other {
			return true
		}
	}
	for _, f := range []string{"/.dockerenv", "/run/.containerenv"} {
		if _, err := os.Stat(f); err == nil {
			return true
		}
	}
	if b, err := os.ReadFile("/sys/class/dmi/id/product_name"); err == nil && vmProduct.Match(b) {
		return true
	}
	if b, err := os.ReadFile("/sys/class/dmi/id/sys_vendor"); err == nil && vmProduct.Match(b) {
		return true
	}
	return false
}

// renderDrivers lists the kernel drivers of the GPUs WebKit could render with.
func renderDrivers() []string {
	nodes, _ := filepath.Glob("/sys/class/drm/renderD*")
	var out []string
	for _, n := range nodes {
		if d, err := filepath.EvalSymlinks(filepath.Join(n, "device", "driver")); err == nil {
			out = append(out, filepath.Base(d))
		}
	}
	return out
}
