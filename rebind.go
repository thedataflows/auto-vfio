package main

import (
	"bufio"
	"io/fs"
	"os"
	"path"
	"regexp"
	"strings"
)

const (
	PATH_SYS_BUS_PCI_DRIVERS_VFIO_PCI = "/sys/bus/pci/drivers/vfio-pci"
	PATH_VFIO_CONF                    = "/etc/modprobe.d/vfio.conf"
)

type _rebind struct {
	Bus     []string `short:"b" required:"" help:"Comma separated lisf of Bus addresses. Use 'list' command to get them. Example: 0000:07:00.0,0000:07:00.1" placeholder:"bus-address1"`
	Persist bool     `short:"p" help:"Persist binding to vfio-pci across reboots"`
}

// persistDeviceVfio persists the device to vfio
func (cmd *_rebind) persistDeviceVfio(venDevId string) error {
	// Check if device is already persisted
	file, err := os.OpenFile(PATH_VFIO_CONF, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// create buffer that will hold the file content
	buff := make([]byte, 0)
	// read file line by line
	scanner := bufio.NewScanner(file)
	spaceTabRegex := regexp.MustCompile(`[\s\t]+`)
	added := false
	for scanner.Scan() {
		// already persisted
		if strings.Contains(scanner.Text(), venDevId) && scanner.Text()[0] != '#' {
			return nil
		}
		parts := spaceTabRegex.Split(scanner.Text(), -1)
		if len(parts) < 3 || parts[0] != "options" || parts[1] != "vfio-pci" {
			buff = append(buff, scanner.Text()+"\n"...)
			continue
		}
		ids := append(strings.Split(parts[2], ","), venDevId)
		buff = append(buff, "options vfio-pci "+strings.Join(ids, ",")+"\n"...)
		added = true
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	if len(buff) == 0 || !added {
		buff = append(buff, "options vfio-pci ids="+venDevId+"\n"...)
	}

	// write to file
	if _, err := file.WriteAt(buff, 0); err != nil {
		return err
	}

	return nil
}

type RebindCmd struct {
	Rebind _rebind `cmd:"" aliases:"r" help:"Rebind a device from its driver to vfio-pci"`
}

// Run executes the command
func (cmd *_rebind) Run(globals *Globals) error {
	log := globals.config.Logger()

	// Re-run elevated
	if err := reRunElevated(); err != nil {
		return err
	}

	for _, dev := range cmd.Bus {
		// Check device
		driverPath := PATH_SYS_BUS_PCI_DEVICES + "/" + dev + "/driver"
		files, err := listFiles(driverPath, fs.ModeSymlink)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to list files in %q", driverPath)
			continue
		}
		if len(files) == 0 {
			log.Error().Err(err).Msgf("Driver for device %q not found", dev)
			continue
		}

		vendorId, err := os.ReadFile(PATH_SYS_BUS_PCI_DEVICES + "/" + dev + "/vendor")
		if err != nil {
			log.Error().Err(err).Msgf("Failed to read vendor id for device %q", dev)
			continue
		}
		vendorId = vendorId[2 : len(vendorId)-1]
		deviceId, err := os.ReadFile(PATH_SYS_BUS_PCI_DEVICES + "/" + dev + "/device")
		if err != nil {
			log.Error().Err(err).Msgf("Failed to read device id for device %q", dev)
			continue
		}
		deviceId = deviceId[2 : len(deviceId)-1]

		// persist
		if cmd.Persist {
			err := cmd.persistDeviceVfio(string(vendorId) + ":" + string(deviceId))
			if err != nil {
				log.Error().Err(err).Msgf("Failed to persist device %q to vfio", dev)
				continue
			}
			log.Info().Msgf("Device %q persisted to vfio-pci in %q", dev, PATH_VFIO_CONF)
		}

		driver, err := os.Readlink(files[0])
		if err != nil {
			log.Error().Err(err).Msgf("Failed to readlink %q", files[0])
			continue
		}
		driverName := path.Base(driver)
		switch driverName {
		case "vfio-pci":
			log.Warn().Msgf("Device %q is already bound to vfio-pci", dev)
			continue
		case "nvidia":
			// Check modeset
			modeset := "/sys/module/nvidia_drm/parameters/modeset"
			modesetValue, err := os.ReadFile(modeset)
			if err != nil {
				log.Error().Err(err).Msgf("Failed to read %q", modeset)
				continue
			}
			if string(modesetValue) == "Y\n" {
				log.Info().Msg("Disabling nvidia_drm modeset")
				err = writeSysfsFileWithTimeout(modeset, "N")
				if err != nil {
					log.Error().Err(err).Msg("Failed to disable nvidia_drm modeset")
					continue
				}
			}
		}
		// Unbind device from current driver
		log.Info().Msgf("Unbinding device %q from driver %q", dev, driverName)
		err = writeSysfsFileWithTimeout(files[0]+"/unbind", dev)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to unbind device %q", dev)
			continue
		}

		// Bind to vfio
		log.Info().Msgf("Binding device %q to vfio-pci", dev)
		id := string(vendorId) + " " + string(deviceId)
		err = writeSysfsFileWithTimeout(PATH_SYS_BUS_PCI_DRIVERS_VFIO_PCI+"/new_id", id)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to add id %q of device %q to vfio-pci", id, dev)
			continue
		}
		err = writeSysfsFileWithTimeout(PATH_SYS_BUS_PCI_DRIVERS_VFIO_PCI+"/bind", dev)
		if err != nil {
			log.Error().Err(err).Msgf("Failed to bind device %q to vfio-pci", dev)
			continue
		}

		log.Info().Msgf("Device %q bound successfully", dev)
	}

	return nil
}
