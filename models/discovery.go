package models

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/pion/mediadevices/pkg/driver"
	mdcam "github.com/pion/mediadevices/pkg/driver/camera"
	"github.com/pion/mediadevices/pkg/prop"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/videosource"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

type findCams struct {
	resource.Named
	resource.TriviallyCloseable
	resource.AlwaysRebuild

	logger logging.Logger
}

func newFindWebcamsWebcamDiscovery(_ context.Context, _ resource.Dependencies, conf resource.Config, logger logging.Logger) (discovery.Service, error) {
	s := &findCams{
		Named:  conf.ResourceName().AsNamed(),
		logger: logger,
	}
	return s, nil
}

// DiscoverResources implements discovery.Service.
func (s *findCams) DiscoverResources(ctx context.Context, extra map[string]any) ([]resource.Config, error) {
	return findCameras(ctx, getVideoDrivers, s.logger)
}

// getVideoDrivers is a helper callback passed to the registered Discover func to get all video drivers.
func getVideoDrivers() []driver.Driver {
	return driver.GetManager().Query(driver.FilterVideoRecorder())
}

// getProperties is a helper func for webcam discovery that returns the Media properties of a specific driver.
// It is NOT related to the GetProperties camera proto API.
func getProperties(d driver.Driver) (_ []prop.Media, err error) {
	// Need to open driver to get properties
	if d.Status() == driver.StateClosed {
		errOpen := d.Open()
		if errOpen != nil {
			return nil, errOpen
		}
		defer func() {
			if errClose := d.Close(); errClose != nil {
				err = errClose
			}
		}()
	}
	return d.Properties(), err
}

func fixName(name string) string {
	// First replace semicolons with hyphens
	name = strings.ReplaceAll(name, ";", "-")
	// remove all non-alphanumeric characters
	reg := regexp.MustCompile(`[^a-zA-Z0-9]`)
	return reg.ReplaceAllString(name, "")
}

// Discover webcam attributes.
func findCameras(ctx context.Context, getDrivers func() []driver.Driver, logger logging.Logger) ([]resource.Config, error) {
	if runtime.GOOS == "linux" || runtime.GOOS == "windows" {
		// Clear all registered camera devices before calling Initialize to prevent duplicates.
		// If first initalize call, this will be a noop.
		manager := driver.GetManager()
		for _, d := range manager.Query(driver.FilterVideoRecorder()) {
			manager.Delete(d.ID())
		}
		// TODO(RSDK-12789): Separate discover() calls from Initialize() calls.
		// So we can call Initialize() only once, and call discover() as many times as we need.
		mdcam.Initialize()
	}
	webcams := []resource.Config{}

	drivers := getDrivers()
	for _, d := range drivers {
		driverInfo := d.Info()

		// Skip Broadcom/BCM devices (typically Raspberry Pi camera-related devices)
		if strings.HasPrefix(driverInfo.Name, "bcm") {
			logger.CDebugw(ctx, "skipping Broadcom/BCM device", "driver", driverInfo.Label)
			continue
		}

		props, err := getProperties(d)
		if len(props) == 0 {
			logger.CDebugw(ctx, "no properties detected for driver, skipping discovery...", "driver", driverInfo.Label)
			continue
		} else if err != nil {
			logger.CDebugw(ctx, "cannot access driver properties, skipping discovery...", "driver", driverInfo.Label, "error", err)
			continue
		}

		if d.Status() == driver.StateRunning {
			logger.CDebugw(ctx, "driver is in use, skipping discovery...", "driver", driverInfo.Label)
			continue
		}

		labelParts := strings.Split(driverInfo.Label, mdcam.LabelSeparator)

		// For macOS and Windows: Label is a single identifier (no separator)
		// For Linux: Label is "name;devicePath" so we need the second part (devicePath)
		label := labelParts[0]
		if len(labelParts) > 1 {
			label = labelParts[1]
		}

		logger.Debugf("found camera drivers with info  %#v", driverInfo)
		logger.Debugf("found %d properties for driver %s", len(props), driverInfo.Name)

		// Create a webcam resource config for every property option
		for i, p := range props {
			logger.Debugf("property %d: %dx%d @ %dfps, format: %s",
				i, p.Video.Width, p.Video.Height, p.Video.FrameRate, p.Video.FrameFormat)
			var result map[string]interface{}
			attributes := videosource.WebcamConfig{
				Path:      label,
				Format:    string(p.Video.FrameFormat),
				Width:     p.Video.Width,
				Height:    p.Video.Height,
				FrameRate: p.Video.FrameRate,
			}

			jsonBytes, err := json.Marshal(attributes)
			if err != nil {
				return nil, err
			}
			if err = json.Unmarshal(jsonBytes, &result); err != nil {
				return nil, err
			}

			// Create unique name for each property option
			name := fixName(driverInfo.Name)
			if len(props) > 1 {
				name = name + "-" + fmt.Sprintf("%d", i)
			}

			wc := resource.Config{
				Name:                name,
				API:                 camera.API,
				Model:               Webcam,
				Attributes:          result,
				ConvertedAttributes: attributes,
			}

			webcams = append(webcams, wc)
		}
	}

	return webcams, nil
}
