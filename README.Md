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

Each discovered camera is returned as a `camera` component config. In addition to the standard `webcam` attributes (`video_path`, `format`, `width_px`, `height_px`, `frame_rate`), the service populates a `device_id` attribute intended as the most stable identifier the OS can provide for the physical device. It is informational — the webcam component itself opens the camera via `video_path`.

### Stability of `device_id` across reboots

The meaning and stability of `device_id` depends on the host platform:

| Platform | Source | Stable across reboots? | Stable across USB port changes? | Caveats |
| --- | --- | --- | --- | --- |
| **Linux** (`by-id`) | `/dev/v4l/by-id/usb-<vendor>_<product>_<serial>-video-indexN` | Yes | Yes | Only generated when the device exposes a non-empty USB serial descriptor. Two identical devices without serials can collide. |
| **Linux** (`by-path`) | `/dev/v4l/by-path/pci-…-usb-<hub>:<port>…-video-indexN` | Yes | **No** — changes if you move the cable | Used as a fallback when `by-id` is unavailable. Distinguishes identical devices by physical port. |
| **Linux** (fallback) | `videoN` | **No** | No | Used only when neither `by-id` nor `by-path` exists. `N` is assigned in probe order and can flip between reboots. |
| **macOS** | `AVCaptureDevice.uniqueID` | Yes for built-in cameras; yes for USB cameras **on the same port** | **No** for USB (Apple derives the ID from USB `locationID` + vendor/product) | |
| **Windows** | DirectShow friendly name (e.g. `Logitech HD Pro Webcam C920`) | Yes | Yes | Collides for two identical devices. Can change if the driver renames the device. |

On Linux, the service prefers the most stable identifier available: `by-id` → `by-path` → `videoN`. If you need a serial-number-level identifier for tracking a physical camera across reboots *and* replugs, you must rely on `by-id`, which in turn requires the device to expose a USB serial.

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
