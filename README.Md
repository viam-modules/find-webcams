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

## Windows Support

On Windows, discovered webcams' returned configurations use the `viam-labs:modular-webcam:webcam` model instead of the built-in webcam component `rdk:builtin:webcam`. This dependency ([github.com/viam-labs/modular-webcam](https://github.com/viam-labs/modular-webcam)) provides Windows-compatible webcam support.

We recommend using the viam-labs fragment `windows-webcam`, which will automatically add the find-webcams module, the modular-webcam module, as well as a discovery resource automatically for you. You can do this by navigating to your machine on app.viam.com, clicking the create button `+`, clicking `Insert fragment`, then typing `windows-webcam`.

Alternatively, you can manually add the `viam-labs:modular-webcam` module to your machine configuration:

```json
{
  "modules": [
    {
      "name": "modular-webcam",
      "module_id": "viam-labs:modular-webcam",
      "type": "registry"
    }
  ]
}
```
