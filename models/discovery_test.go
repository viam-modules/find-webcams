package models

import (
	"context"
	"fmt"
	"testing"

	"go.viam.com/rdk/logging"
	"go.viam.com/test"

	"github.com/pion/mediadevices/pkg/driver"
	"github.com/pion/mediadevices/pkg/prop"
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/videosource"
)

const someName = "USB223;44 Somecam^"

// fakeDriver is a driver has a label and media properties.
type fakeDriver struct {
	label string
	name  string
	props []prop.Media
}

func (d *fakeDriver) Open() error              { return nil }
func (d *fakeDriver) Properties() []prop.Media { return d.props }
func (d *fakeDriver) ID() string               { return d.label }
func (d *fakeDriver) Info() driver.Info        { return driver.Info{Label: d.label, Name: d.name} }
func (d *fakeDriver) Status() driver.State     { return "some state" }
func (d *fakeDriver) Close() error             { return nil }

func newFakeDriver(label, name string, props []prop.Media) driver.Driver {
	return &fakeDriver{label: label, name: name, props: props}
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
	withProps := newFakeDriver("some_label", someName, props)
	withoutProps := newFakeDriver("another label", someName, []prop.Media{})
	return []driver.Driver{withProps, withoutProps}
}

func TestDiscoveryWebcam(t *testing.T) {
	logger := logging.NewTestLogger(t)
	resp, err := findCameras(context.Background(), testGetDrivers, logger)

	test.That(t, err, test.ShouldBeNil)
	test.That(t, resp, test.ShouldHaveLength, 3)

	for i, config := range resp {
		test.That(t, config.API, test.ShouldResemble, camera.API)
		test.That(t, config.Model, test.ShouldResemble, Webcam)

		// Names should be unique with index suffix
		expectedName := fixName(someName) + "-" + fmt.Sprintf("%d", i)
		test.That(t, config.Name, test.ShouldEqual, expectedName)

		cfg, ok := config.ConvertedAttributes.(videosource.WebcamConfig)
		test.That(t, ok, test.ShouldBeTrue)
		test.That(t, cfg, test.ShouldHaveSameTypeAs, videosource.WebcamConfig{})
	}

	cfg0, _ := resp[0].ConvertedAttributes.(videosource.WebcamConfig)
	test.That(t, cfg0.Width, test.ShouldEqual, 1920)
	test.That(t, cfg0.Height, test.ShouldEqual, 1080)
	test.That(t, cfg0.Format, test.ShouldEqual, "MJPEG")
	test.That(t, cfg0.FrameRate, test.ShouldEqual, 30.0)

	cfg1, _ := resp[1].ConvertedAttributes.(videosource.WebcamConfig)
	test.That(t, cfg1.Width, test.ShouldEqual, 1280)
	test.That(t, cfg1.Height, test.ShouldEqual, 720)
	test.That(t, cfg1.Format, test.ShouldEqual, "MJPEG")
	test.That(t, cfg1.FrameRate, test.ShouldEqual, 60.0)

	cfg2, _ := resp[2].ConvertedAttributes.(videosource.WebcamConfig)
	test.That(t, cfg2.Width, test.ShouldEqual, 640)
	test.That(t, cfg2.Height, test.ShouldEqual, 480)
	test.That(t, cfg2.Format, test.ShouldEqual, "YUYV")
	test.That(t, cfg2.FrameRate, test.ShouldEqual, 30.0)
}

func TestDiscoveryWebcamEmptyName(t *testing.T) {
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

		cfg0, _ := resp[0].ConvertedAttributes.(videosource.WebcamConfig)
		test.That(t, cfg0.Width, test.ShouldEqual, 1920)
		test.That(t, cfg0.Format, test.ShouldEqual, "MJPEG")

		cfg2, _ := resp[2].ConvertedAttributes.(videosource.WebcamConfig)
		test.That(t, cfg2.Width, test.ShouldEqual, 640)
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
			// Driver 2: Empty name, 1 property -> webcam-2
			d2 := newFakeDriver("label2", "", []prop.Media{
				{Video: prop.Video{Width: 640}},
			})
			return []driver.Driver{d0, d1, d2}
		}

		resp, err := findCameras(context.Background(), getDrivers, logger)
		test.That(t, err, test.ShouldBeNil)
		test.That(t, resp, test.ShouldHaveLength, 5)

		// Verify d0 (Empty name, driver index 0)
		test.That(t, resp[0].Name, test.ShouldEqual, "webcam-0-0")
		test.That(t, resp[1].Name, test.ShouldEqual, "webcam-0-1")

		// Verify d1 (Valid name "cam", driver index 1 - index not used for valid names)
		test.That(t, resp[2].Name, test.ShouldEqual, "cam-0")
		test.That(t, resp[3].Name, test.ShouldEqual, "cam-1")

		// Verify d2 (Empty name, driver index 2)
		test.That(t, resp[4].Name, test.ShouldEqual, "webcam-2")
	})
}
