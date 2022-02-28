//Package rpiws281x is used for controlling WS281x LEDs on the Raspberry Pi using plain GO.
package rpiws281x

import (
	"errors"
	"fmt"

	"github.com/DerLukas15/rpihardware"
)

// Errors
var (
	ErrNoClockMap         = errors.New("clock device map not set. Not initialized?")
	ErrNoHardware         = errors.New("no hardware set. Not initialized?")
	ErrDriverAlreadyUsed  = errors.New("driver already initialized")
	ErrConfigInitialized  = errors.New("config already initialized")
	ErrConfigWrongIndex   = errors.New("wrong strip index")
	ErrDriverNotSupported = errors.New("driver not supported")
	ErrPinNotAllowed      = errors.New("selected pin not allowed")
	ErrNoActiveChannel    = errors.New("No active channel")
	ErrWrongFrequency     = errors.New("Wrong Frequency")
)

var (
	pwmActive   bool                  // Set once a config with PWM as driver is active
	pcmActive   bool                  // Set once a config with PCM as driver is active
	spiActive   bool                  // Set once a config with SPI as driver is active
	curHardware *rpihardware.Hardware // Set during initialize
)

//DriverType defines the hardware type (PWM, PCM, SPI) which is used for communication
type DriverType uint8

//Valid DriverTypes
const (
	DriverPWM DriverType = 1 << iota
	DriverPCM
	DriverSPI
)

//StripType is the layout of the connected strip an can be different for each output signal.
type StripType uint

//Valid StripTypes
const (
	sk6812ShiftMask uint = 0xf0000000

	// 4 color R, G, B and W ordering
	SK6812StripRGBW StripType = 0x18100800
	SK6812StripRBGW           = 0x18100008
	SK6812StripGRBW           = 0x18081000
	SK6812StripGBRW           = 0x18080010
	SK6812StripBRGW           = 0x18001008
	SK6812StripBGRW           = 0x18000810

	// 3 color R, G and B ordering
	WS2811StripRGB StripType = 0x00100800
	WS2811StripRBG           = 0x00100008
	WS2811StripGRB           = 0x00081000
	WS2811StripGBR           = 0x00080010
	WS2811StripBRG           = 0x00001008
	WS2811StripBGR           = 0x00000810
)

// Predefined fixed LED types
const (
	WS2812Strip  = WS2811StripGRB
	SK6812Strip  = WS2811StripGRB
	SK6812WStrip = SK6812StripGRBW
)

const (
	symbolHigh uint8 = 0b110
	symbolLow  uint8 = 0b100

	symbolHighInv = ^symbolHigh
	symbolLowInv  = ^symbolLow
)

var gammaTable = make([]uint8, 256, 256)

//Enable Debug output
var Debug bool

//Enable two channel mode for PWM no matter the configuration
var PWMAlwaysUseTwoChannel bool

func init() {
	for x := 0; x < 256; x++ {
		gammaTable[x] = uint8(x)
	}
}

func logOutput(msg string) {
	if Debug {
		fmt.Println(msg)
	}
}
