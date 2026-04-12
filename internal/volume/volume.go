//go:build darwin && cgo

// Package volume provides direct system audio volume control via macOS CoreAudio.
package volume

/*
#cgo LDFLAGS: -framework CoreAudio

#include <CoreAudio/CoreAudio.h>

static AudioDeviceID defaultOutputDevice() {
	AudioDeviceID deviceID = 0;
	UInt32 size = sizeof(deviceID);
	AudioObjectPropertyAddress addr = {
		kAudioHardwarePropertyDefaultOutputDevice,
		kAudioObjectPropertyScopeGlobal,
		0
	};
	AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &deviceID);
	return deviceID;
}

// volumeElement returns the element (channel) that supports volume control.
// Tries master (0) first, then channel 1.
static UInt32 volumeElement() {
	AudioDeviceID device = defaultOutputDevice();
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioObjectPropertyScopeOutput,
		0
	};
	if (AudioObjectHasProperty(device, &addr)) return 0;
	return 1;
}

static float getVolume() {
	AudioDeviceID device = defaultOutputDevice();
	Float32 volume = 0;
	UInt32 size = sizeof(volume);
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioObjectPropertyScopeOutput,
		volumeElement()
	};
	AudioObjectGetPropertyData(device, &addr, 0, NULL, &size, &volume);
	return volume;
}

static void setVolume(float volume) {
	if (volume < 0.0f) volume = 0.0f;
	if (volume > 1.0f) volume = 1.0f;
	AudioDeviceID device = defaultOutputDevice();
	UInt32 elem = volumeElement();
	AudioObjectPropertyAddress addr = {
		kAudioDevicePropertyVolumeScalar,
		kAudioObjectPropertyScopeOutput,
		elem
	};
	AudioObjectSetPropertyData(device, &addr, 0, NULL, sizeof(volume), &volume);
	// If no master channel, set both stereo channels.
	if (elem != 0) {
		addr.mElement = 2;
		AudioObjectSetPropertyData(device, &addr, 0, NULL, sizeof(volume), &volume);
	}
}
*/
import "C"

// volumeStep is the fraction of full volume changed per key press (5%).
const volumeStep = 0.05

// Get returns the current system output volume (0.0–1.0).
func Get() float32 {
	return float32(C.getVolume())
}

// Set sets the system output volume to an exact value (0.0–1.0).
func Set(v float32) {
	C.setVolume(C.float(v))
}

// Up increases the system output volume by 5%.
func Up() {
	Set(Get() + volumeStep)
}

// Down decreases the system output volume by 5%.
func Down() {
	Set(Get() - volumeStep)
}
