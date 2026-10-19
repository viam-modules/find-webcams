package models

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"

	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/prop"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/videosource"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/utils"
)

const someName = "USB223;44 Somecam^"

// fakeDriver is a driver has a label and media properties. Like a real camera driver, it only
// reports properties while open. It counts Open/Close calls so tests can check that discovery
// leaves drivers the way it found them.
type fakeDriver struct {
	label   string
	name    string
	props   []prop.Media
	status  driver.State
	openErr error
	opens   int
	closes  int
}

func (d *fakeDriver) Open() error {
	d.opens++
	if d.openErr != nil {
		return d.openErr
	}
	d.status = driver.StateOpened
	return nil
}

func (d *fakeDriver) Close() error {
	d.closes++
	d.status = driver.StateClosed
	return nil
}

func (d *fakeDriver) Properties() []prop.Media {
	if d.status == driver.StateClosed {
		return nil
	}
	return d.props
}

func (d *fakeDriver) ID() string           { return d.label }
func (d *fakeDriver) Info() driver.Info    { return driver.Info{Label: d.label, Name: d.name} }
func (d *fakeDriver) Status() driver.State { return d.status }

func newFakeDriver(label, name string, props []prop.Media) *fakeDriver {
	return &fakeDriver{label: label, name: name, props: props, status: driver.StateClosed}
}

func testGetDrivers() []driver.Driver {
	props := []prop.Media{
		{
			Video:    prop.Video{Width: 1920, Height: 1080, FrameFormat: "MJPEG", FrameRate: 30.0},
			DeviceID: "some_device_id;",
		},
		{
			Video:    prop.Video{Width: 1280, Height: 720, FrameFormat: "MJPEG", FrameRate: 60.0},
			DeviceID: "some_device_id;",
		},
		{
			Video:    prop.Video{Width: 640, Height: 480, FrameFormat: "YUYV", FrameRate: 30.0},
			DeviceID: "some_device_id;",
		},
	}
	// Simulate pion's Linux label format "<stable id>;<device node>". We surface
	// the stable first half ("some_device_id") as video_path and intentionally
	// discard the unstable device node ("some_label").
	withProps := newFakeDriver("some_device_id;some_label", someName, props)
	withoutProps := newFakeDriver("another label", someName, []prop.Media{})
	return []driver.Driver{withProps, withoutProps}
}

func TestDiscoveryWebcamNaming(t *testing.T) {
	logger := logging.NewTestLogger(t)

	t.Run("Basic discovery with valid name and multiple properties", func(t *testing.T) {
		resp, err := findCameras(context.Background(), testGetDrivers, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, resp, test.ShouldHaveLength, 3)

		for i, config := range resp {
			test.That(t, config.API, test.ShouldResemble, camera.API)
			test.That(t, config.Model, test.ShouldResemble, videosource.ModelWebcam)

			// Names should be unique with index suffix for multiple properties
			expectedName := fixName(someName) + "-" + fmt.Sprintf("%d", i)
			test.That(t, config.Name, test.ShouldEqual, expectedName)
		}

		cfg0, ok := resp[0].ConvertedAttributes.(videosource.WebcamConfig)
		test.That(t, ok, test.ShouldBeTrue)
		test.That(t, cfg0.Width, test.ShouldEqual, 1920)
		test.That(t, cfg0.Format, test.ShouldEqual, "MJPEG")
		// video_path is the stable first label half, not the device node.
		test.That(t, cfg0.Path, test.ShouldEqual, "some_device_id")

		cfg2, ok := resp[2].ConvertedAttributes.(videosource.WebcamConfig)
		test.That(t, ok, test.ShouldBeTrue)
		test.That(t, cfg2.Width, test.ShouldEqual, 640)

		// Every result should carry the stable identifier as video_path.
		for _, config := range resp {
			test.That(t, config.Attributes["video_path"], test.ShouldEqual, "some_device_id")
		}
	})

	t.Run("Empty name fallback and uniqueness across multiple drivers", func(t *testing.T) {
		getDrivers := func() []driver.Driver {
			// Driver 0: Empty name, 1 prop -> webcam-0
			d1 := newFakeDriver("label1", "", []prop.Media{{Video: prop.Video{Width: 640}}})
			// Driver 1: Empty name, 1 prop -> webcam-1
			d2 := newFakeDriver("label2", "", []prop.Media{{Video: prop.Video{Width: 640}}})
			return []driver.Driver{d1, d2}
		}

		resp, err := findCameras(context.Background(), getDrivers, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, resp, test.ShouldHaveLength, 2)
		test.That(t, resp[0].Name, test.ShouldEqual, "webcam-0")
		test.That(t, resp[1].Name, test.ShouldEqual, "webcam-1")
	})

	t.Run("Complex naming with empty names and multiple properties", func(t *testing.T) {
		getDrivers := func() []driver.Driver {
			// Driver 0: Empty name, 2 properties -> webcam-0-0, webcam-0-1
			d0 := newFakeDriver("label0", "", []prop.Media{
				{Video: prop.Video{Width: 640}},
				{Video: prop.Video{Width: 1280}},
			})
			// Driver 1: Valid name "cam", 2 properties -> cam-0, cam-1
			d1 := newFakeDriver("label1", "cam", []prop.Media{
				{Video: prop.Video{Width: 640}},
				{Video: prop.Video{Width: 1280}},
			})
			// Driver 2: Empty name, 1 property -> webcam-1 (second unnamed camera)
			d2 := newFakeDriver("label2", "", []prop.Media{
				{Video: prop.Video{Width: 640}},
			})
			return []driver.Driver{d0, d1, d2}
		}

		resp, err := findCameras(context.Background(), getDrivers, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, resp, test.ShouldHaveLength, 5)

		// Verify d0 (first unnamed camera)
		test.That(t, resp[0].Name, test.ShouldEqual, "webcam-0-0")
		test.That(t, resp[1].Name, test.ShouldEqual, "webcam-0-1")

		// Verify d1 (unique name "cam", so no camera number)
		test.That(t, resp[2].Name, test.ShouldEqual, "cam-0")
		test.That(t, resp[3].Name, test.ShouldEqual, "cam-1")

		// Verify d2 (second unnamed camera)
		test.That(t, resp[4].Name, test.ShouldEqual, "webcam-1")
	})

	t.Run("Identical names are numbered in label order", func(t *testing.T) {
		// Identical camera models report the same name, e.g. two "USB Camera"s on macOS/Windows,
		// or two capture nodes of one device on Linux.
		usb0 := newFakeDriver("label0", "USB Camera", []prop.Media{{Video: prop.Video{Width: 640}}})
		other := newFakeDriver("label1", "Other Cam", []prop.Media{{Video: prop.Video{Width: 640}}})
		usb1 := newFakeDriver("label2", "USB Camera", []prop.Media{
			{Video: prop.Video{Width: 640}},
			{Video: prop.Video{Width: 1280}},
		})
		usb2 := newFakeDriver("label3", "USB Camera", []prop.Media{{Video: prop.Video{Width: 640}}})
		// A camera named "web cam" shares its fixed name with unnamed cameras.
		webcam := newFakeDriver("label4", "web cam", []prop.Media{{Video: prop.Video{Width: 640}}})
		unnamed := newFakeDriver("label5", "", []prop.Media{{Video: prop.Video{Width: 640}}})

		// The driver manager returns drivers in map order, so the same cameras must get the same
		// names whatever order they arrive in.
		for _, drivers := range [][]driver.Driver{
			{usb0, other, usb1, usb2, webcam, unnamed},
			{unnamed, usb2, webcam, other, usb1, usb0},
		} {
			resp, err := findCameras(context.Background(), func() []driver.Driver { return drivers }, logger)
			test.That(t, err, test.ShouldBeNil)

			names := make([]string, 0, len(resp))
			for _, config := range resp {
				names = append(names, config.Name)
			}
			test.That(t, names, test.ShouldResemble, []string{
				"USBCamera-0", "OtherCam", "USBCamera-1-0", "USBCamera-1-1", "USBCamera-2", "webcam-0", "webcam-1",
			})
		}
	})
}

// TestDiscoverySkipsDrivers checks each reason a driver is left out of discovery. A usable camera
// is listed after the skipped one to show that a skip doesn't stop the scan.
func TestDiscoverySkipsDrivers(t *testing.T) {
	logger := logging.NewTestLogger(t)
	mjpeg := []prop.Media{{Video: prop.Video{Width: 640, Height: 480, FrameFormat: "MJPEG", FrameRate: 30}}}

	for _, tc := range []struct {
		name       string
		driverName string
		status     driver.State
		openErr    error
		props      []prop.Media
	}{
		{name: "Raspberry Pi ISP node", driverName: "bcm2835-isp;platform:bcm2835-isp", props: mjpeg},
		{name: "camera already in use", driverName: "Busy Cam", status: driver.StateRunning, props: mjpeg},
		{name: "camera fails to open", driverName: "Broken Cam", openErr: errors.New("device busy"), props: mjpeg},
		{name: "node with no video formats", driverName: "Metadata Node"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			skipped := newFakeDriver("skipped", tc.driverName, tc.props)
			if tc.status != "" {
				skipped.status = tc.status
			}
			skipped.openErr = tc.openErr
			usable := newFakeDriver("usable", "Usable Cam", mjpeg)

			resp, err := findCameras(context.Background(), func() []driver.Driver {
				return []driver.Driver{skipped, usable}
			}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, resp, test.ShouldHaveLength, 1)
			test.That(t, resp[0].Name, test.ShouldEqual, "UsableCam")
		})
	}
}

// TestDiscoveryLeavesDriversAsFound checks that discovery opens a closed driver only long enough to
// read its properties, and never touches a running one. Leaving a driver open would hold the
// device; closing a running one would cut off the camera component streaming from it.
func TestDiscoveryLeavesDriversAsFound(t *testing.T) {
	logger := logging.NewTestLogger(t)
	mjpeg := []prop.Media{{Video: prop.Video{Width: 640, Height: 480, FrameFormat: "MJPEG", FrameRate: 30}}}

	idle := newFakeDriver("idle", "Idle Cam", mjpeg)
	streaming := newFakeDriver("streaming", "Streaming Cam", mjpeg)
	streaming.status = driver.StateRunning

	_, err := findCameras(context.Background(), func() []driver.Driver {
		return []driver.Driver{idle, streaming}
	}, logger)
	test.That(t, err, test.ShouldBeNil)

	test.That(t, idle.opens, test.ShouldEqual, 1)
	test.That(t, idle.closes, test.ShouldEqual, 1)

	test.That(t, streaming.opens, test.ShouldEqual, 0)
	test.That(t, streaming.closes, test.ShouldEqual, 0)
}

// TestDiscoveredConfigsAreValid checks that every config discovery returns could be added to a
// machine as-is: each one passes rdk's config validation and no two share a name. Driver names
// and labels follow the formats pion reports on each OS.
func TestDiscoveredConfigsAreValid(t *testing.T) {
	logger := logging.NewTestLogger(t)
	formats := []prop.Media{
		{Video: prop.Video{Width: 1920, Height: 1080, FrameFormat: "MJPEG", FrameRate: 30}},
		{Video: prop.Video{Width: 640, Height: 480, FrameFormat: "YUYV", FrameRate: 30}},
	}
	getDrivers := func() []driver.Driver {
		return []driver.Driver{
			// Linux: name is "<card>;<bus_info>", label is "<by-id name>;<device node>".
			newFakeDriver("usb-046d_HD_Pro_Webcam_C920_A1B2C3D4-video-index0;video0",
				"HD Pro Webcam C920;usb-0000:00:14.0-1", formats),
			// macOS: AVFoundation device name and UID.
			newFakeDriver("0x1420000005ac8600", "FaceTime HD Camera", formats),
			// Two identical USB cameras report the same name.
			newFakeDriver("0x1100000c45636b", "USB Camera", formats),
			newFakeDriver("0x1200000c45636b", "USB Camera", formats),
			// Windows: friendly names can include punctuation, or be empty.
			newFakeDriver(`\\?\usb#vid_0bda&pid_58c2&mi_00#6&2f3c9a1&0&0000#{65e8773d-8f56-11d0-a3b9-00a0c9223196}\global`,
				"Integrated Webcam (HD) – Front", formats),
			newFakeDriver(`\\?\usb#vid_046d&pid_0825&mi_00#7&1a2b3c4&0&0000#{65e8773d-8f56-11d0-a3b9-00a0c9223196}\global`,
				"", formats),
		}
	}

	resp, err := findCameras(context.Background(), getDrivers, logger)
	test.That(t, err, test.ShouldBeNil)
	test.That(t, resp, test.ShouldHaveLength, 12)

	seen := map[string]bool{}
	for _, config := range resp {
		_, _, err := config.Validate("test", resource.APITypeComponentName)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, seen, test.ShouldNotContainKey, config.Name)
		seen[config.Name] = true
	}
}

// TestDiscoveryAttributes checks the attribute map a user would add to their config, including
// which part of the driver label becomes video_path.
func TestDiscoveryAttributes(t *testing.T) {
	logger := logging.NewTestLogger(t)
	format := []prop.Media{{Video: prop.Video{Width: 1920, Height: 1080, FrameFormat: "MJPEG", FrameRate: 30}}}

	for _, tc := range []struct {
		name          string
		label         string
		wantVideoPath string
	}{
		{
			// The /dev/videoN half can change across reboots and replugs; the by-id name doesn't.
			name:          "Linux label uses the stable by-id name",
			label:         "usb-046d_HD_Pro_Webcam_C920_A1B2C3D4-video-index0;video0",
			wantVideoPath: "usb-046d_HD_Pro_Webcam_C920_A1B2C3D4-video-index0",
		},
		{
			name:          "macOS and Windows labels are used whole",
			label:         "0x1420000005ac8600",
			wantVideoPath: "0x1420000005ac8600",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := findCameras(context.Background(), func() []driver.Driver {
				return []driver.Driver{newFakeDriver(tc.label, "Cam", format)}
			}, logger)
			test.That(t, err, test.ShouldBeNil)
			test.That(t, resp, test.ShouldHaveLength, 1)
			test.That(t, resp[0].Attributes, test.ShouldResemble, utils.AttributeMap{
				"video_path": tc.wantVideoPath,
				"format":     "MJPEG",
				"width_px":   1920.0,
				"height_px":  1080.0,
				"frame_rate": 30.0,
			})
		})
	}
}
