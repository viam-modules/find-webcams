package main

import (
	"find-webcams/models"
	"os"
	"runtime"

	mdcam "github.com/pion/mediadevices/pkg/driver/camera"
	"go.viam.com/rdk/logging"
	"go.viam.com/rdk/module"
	"go.viam.com/rdk/resource"
	"go.viam.com/rdk/services/discovery"
)

var logger = logging.NewDebugLogger("find-webcams-entrypoint")

func main() {
	if runtime.GOOS == "darwin" {
		if err := mdcam.StartObserver(); err != nil {
			logger.Errorw("failed to start camera observer", "error", err)
			os.Exit(1)
		}
		defer func() {
			if err := mdcam.DestroyObserver(); err != nil {
				logger.Errorw("failed to destroy camera observer", "error", err)
			}
		}()
	}

	module.ModularMain(
		resource.APIModel{
			API:   discovery.API,
			Model: models.WebcamDiscovery,
		},
	)
}
