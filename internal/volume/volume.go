//go:build darwin && cgo

// Package volume provides direct system audio volume control via macOS CoreAudio.
package volume

/*
#cgo LDFLAGS: -framework CoreAudio

#include <CoreAudio/CoreAudio.h>

static OSStatus defaultOutputDevice(AudioDeviceID *deviceID) {
	UInt32 size = sizeof(*deviceID);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		0
	};
	return AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, deviceID);
}

static Boolean hasVolumeProperty(AudioDeviceID device, UInt32 element) {
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioObjectPropertyScopeOutput,
		element
	};
	return AudioObjectHasProperty(device, &addr);
}

// volumeElement returns the element (channel) that supports volume control.
// Tries master (0) first, then channel 1.
static OSStatus volumeElement(AudioDeviceID device, UInt32 *element) {
	if (hasVolumeProperty(device, 0)) {
		*element = 0;
		return noErr;
	}
	if (hasVolumeProperty(device, 1)) {
		*element = 1;
		return noErr;
	}
	return kAudioHardwareUnknownPropertyError;
}

static OSStatus getVolume(Float32 *volume) {
	AudioDeviceID device = 0;
	OSStatus status = defaultOutputDevice(&device);
	if (status != noErr) return status;

	UInt32 element = 0;
	status = volumeElement(device, &element);
	if (status != noErr) return status;

	UInt32 size = sizeof(*volume);
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioObjectPropertyScopeOutput,
		element
	};
	return AudioObjectGetPropertyData(device, &addr, 0, NULL, &size, volume);
}

static OSStatus setVolume(Float32 volume) {
	if (volume < 0.0f) volume = 0.0f;
	if (volume > 1.0f) volume = 1.0f;

	AudioDeviceID device = 0;
	OSStatus status = defaultOutputDevice(&device);
	if (status != noErr) return status;

	UInt32 elem = 0;
	status = volumeElement(device, &elem);
	if (status != noErr) return status;

	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioObjectPropertyScopeOutput,
		elem
	};
	status = AudioObjectSetPropertyData(device, &addr, 0, NULL, sizeof(volume), &volume);
	if (status != noErr) return status;

	// If no master channel, set both stereo channels when channel 2 exists.
	if (elem != 0 && hasVolumeProperty(device, 2)) {
		addr.mElement = 2;
		status = AudioObjectSetPropertyData(device, &addr, 0, NULL, sizeof(volume), &volume);
		if (status != noErr) return status;
	}

	return noErr;
}
*/
import "C"

import "fmt"

// volumeStep is the fraction of full volume changed per key press (5%).
const volumeStep = 0.05

// osStatusError converts a CoreAudio OSStatus code to a Go error, returning nil for noErr.
func osStatusError(op string, status C.OSStatus) error {
	if status == C.OSStatus(C.noErr) {
		return nil
	}
	return fmt.Errorf("%s failed: OSStatus %d", op, int32(status))
}

// Get returns the current system output volume (0.0–1.0).
func Get() (float32, error) {
	var vol C.Float32
	status := C.getVolume(&vol)
	if err := osStatusError("get volume", status); err != nil {
		return 0, err
	}
	return float32(vol), nil
}

// Set sets the system output volume to an exact value (0.0–1.0).
func Set(v float32) error {
	status := C.setVolume(C.Float32(v))
	return osStatusError("set volume", status)
}

// Up increases the system output volume by 5%.
func Up() error {
	v, err := Get()
	if err != nil {
		return err
	}
	return Set(v + volumeStep)
}

// Down decreases the system output volume by 5%.
func Down() error {
	v, err := Get()
	if err != nil {
		return err
	}
	return Set(v - volumeStep)
}
