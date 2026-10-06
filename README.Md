# Webcam Discovery Module for Viam

This module provides a viam discovery service for automatically detecting and configuring webcams for use with the [Viam platform](https://www.viam.com).

## Overview

The [`webcam-discovery`](models/module.go) discovery service automatically:
- Scans for connected USB webcam devices
- Detects their capabilities and configuration
- Generates the appropriate Viam resource configurations

## Attributes
No models in this module have any attributes. Just add the model to your machine, expand the test card or look at the discovery control card, and copy/paste the video path to the `video_path` attribute on a `rdk:builtin:webcam` or click the add component model.

The discovery service detects the highest available resolution of the first format it finds for a usb webcam connected to the machine running viam-server.

## Discovery output

Each discovered camera is returned as a `camera` component config with the standard `webcam` attributes (`video_path`, `format`, `width_px`, `height_px`, `frame_rate`). On Linux `video_path` is set to the stable `/dev/v4l/by-id` (or `by-path`) name rather than the volatile `/dev/videoN` node, so it survives reboots and replugs.

## Windows support

Discovered webcams use the built-in `rdk:builtin:webcam` model on every platform. On Windows, that model requires **viam-server v1.11.0 or later**; older versions of viam-server cannot run the configurations this module returns on Windows. If you previously used the `viam-labs:modular-webcam` module or the `windows-webcam` fragment, upgrade viam-server and switch your camera components to `rdk:builtin:webcam`.

## Example config

Add this module to your Viam robot's configuration:

```json
{
    {
  "services": [
    {
      "name": "webcam-discovery",
      "api": "rdk:service:discovery",
      "model": "viam:find-webcams:webcam-discovery",
      "attributes" : {}
    }
  ]
}
  "modules": [
    {
      "name": "find-webcams",
      "module_id": "viam:find-webcams",
      "type": "registry"
    }
  ]
}
```
