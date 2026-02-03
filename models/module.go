package models

import (
	"go.viam.com/rdk/components/camera"
	"go.viam.com/rdk/components/camera/videosource"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

var (
	WebcamDiscovery = resource.NewModel("viam", "find-webcams", "webcam-discovery")
	Webcam          = resource.NewModel("viam", "find-webcams", "webcam")
)

func init() {
	resource.RegisterService(discovery.API, WebcamDiscovery,
		resource.Registration[discovery.Service, resource.NoNativeConfig]{
			Constructor: newFindWebcamsWebcamDiscovery,
		},
	)
	resource.RegisterComponent(camera.API, Webcam,
		resource.Registration[camera.Camera, *videosource.WebcamConfig]{
			Constructor: videosource.NewWebcam,
		},
	)
}
