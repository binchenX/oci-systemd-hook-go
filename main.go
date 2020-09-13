package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

const (
	// LOG is the log file for the oci-systemd-hook plugin
	LOG = "/var/log/hook.log"

	// CgroupRoot is the root cgroup
	CgroupRoot = "/sys/fs/cgroup"

	// CgroupSystemd is the cgroup for systemd
	CgroupSystemd = CgroupRoot + "/systemd"
)

func main() {
	f, err := os.OpenFile(LOG, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0755)
	if err != nil {
		log.Fatal("Fail to create log file")
	}
	logrus.SetOutput(f)
	logrus.SetLevel(logrus.DebugLevel)

	var hookdata specs.State
	var spec specs.Spec

	if hookdata, err = getHookData(); err != nil {
		log.Fatalf("Fail to get hook data %s", err)
	}

	if spec, err = getSpec(hookdata.Bundle); err != nil {
		log.Fatalf("Fail to get spec %s", err)
	}
	enableSystemd(hookdata, spec)
}

// getHookData read the stdio for the hook data (state) passed from the runtime
func getHookData() (hook specs.State, err error) {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return hook, err
	}

	if err := json.Unmarshal(b, &hook); err != nil {
		return hook, err
	}

	return hook, nil
}

// getSpec returns the Spec by reading the config.json in the bundle
func getSpec(bundleDir string) (spec specs.Spec, err error) {
	configFile := path.Join(bundleDir, "config.json")
	f, err := os.Open(configFile)
	if err != nil {
		return spec, err
	}

	d, err := ioutil.ReadAll(f)
	if err != nil {
		return spec, err
	}

	if err := json.Unmarshal(d, &spec); err != nil {
		return spec, err
	}

	return spec, nil
}

// enableSystemd setup the stuff needed for systemd to run
// reference:
// https://developers.redhat.com/blog/2016/09/13/running-systemd-in-a-non-privileged-container/
// https://github.com/projectatomic/oci-systemd-hook
func enableSystemd(hook specs.State, spec specs.Spec) {
	remountCgroupSystemdRW(spec)
	createAndMountAsTmpFs(spec.Root.Path, "run")
	createAndMountAsTmpFs(spec.Root.Path, "run/lock")
	createAndMountAsTmpFs(spec.Root.Path, "tmp")
	createMachineID(spec.Root.Path, hook.ID)
}

// remountCgroupSystemdRW remount /sys/fs/cgroup/systemd as rw
func remountCgroupSystemdRW(spec specs.Spec) error {
	cgroupSystemdPath := path.Join(spec.Root.Path, CgroupSystemd)
	var stat syscall.Statfs_t
	if err := syscall.Statfs(cgroupSystemdPath, &stat); err != nil {
		logrus.Fatal(err)
	}
	if stat.Type != unix.CGROUP_SUPER_MAGIC && stat.Type != unix.CGROUP2_SUPER_MAGIC {
		logrus.Fatalf("%s is NOT moutned as cgroup yet\n", cgroupSystemdPath)
	}
	flags := stat.Flags &^ unix.MS_RDONLY
	flags |= unix.MS_BIND | unix.MS_REMOUNT

	if err := unix.Mount("cgroup", cgroupSystemdPath, "", uintptr(flags), "name=systemd"); err != nil {
		logrus.Fatalf("err %s when bind mount systemd to writable %s to %s \n", err, CgroupSystemd, cgroupSystemdPath)
	}
	return nil
}

// createAndMountAsTmpFs create the path if it not exsit and mount it as tmpfs
func createAndMountAsTmpFs(rootfs, pathInContainer string) error {
	options := "mode=1777"
	flags := unix.MS_NOSUID | unix.MS_NOEXEC | unix.MS_NODEV
	hostPath := path.Join(rootfs, pathInContainer)
	if _, err := os.Stat(hostPath); os.IsNotExist(err) {
		if err := os.Mkdir(hostPath, 0755); err != nil {
			logrus.Fatal(err)
		}
	}
	if err := unix.Mount("tmpfs", hostPath, "tmpfs", uintptr(flags), options); err != nil {
		log.Fatal(err)
	}
	return nil
}

// createMachineID create the machine-id file needed by systemd from container uuid
func createMachineID(rootfs, containerUUID string) error {
	machineID := fmt.Sprintf("%.32s", containerUUID)
	f := path.Join(rootfs, "etc/machine-id")
	if err := ioutil.WriteFile(f, []byte(machineID), 0644); err != nil {
		logrus.Fatalf("fail to write machine-id")
	}
	return nil
}
