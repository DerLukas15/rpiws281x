package rpiws281x

import (
	"fmt"
	"time"

	"github.com/DerLukas15/rpigpio"
	"github.com/DerLukas15/rpihardware"
	"github.com/pkg/errors"
)

//Config is the main struct which holds all information.
/*
A Config can only contain one driver type (PWM, PCM, or SPI). However you can define multiple configurations with different driver types. Be aware
that only one Config per driver type can be initialized and used at a time.

There are only some methods possible once the Config has been initialized. Use method Stop to deinitialize the Config.
*/
type Config struct {
	driverType DriverType // DriverType for this config. Only on active config per driverType allowed
	// DMA channel to use. Use different channels if you use multiple drivers simultaniously.
	// You should be able to use the same channel if you don't run multiple renders at the same time (i.e. `go config.Render()`)
	dmaChannel uint32
	//Frequency for communication
	frequency uint32
	//PWM allowes two strips as it has two channels. Thus channels is a slice
	channels []ledChannel

	initialized bool

	// Timing for next render
	renderWaitTime     int64
	previousRenderTime time.Time
}

type ledChannel struct {
	stripType  StripType    // StripType
	strip      LEDs         // Actual LEDs
	pin        *rpigpio.Pin // Pin to use for the strip
	invert     bool         // Set output inverse
	active     bool         // Set when this channel is configured to be used
	brightness uint32       // Brightness of the strip

	wshift uint8 //White shift value
	rshift uint8 //Red shift value
	gshift uint8 //Green shift value
	bshift uint8 //Blue shift value
}

//New returns a new Config for driverType.
/*
Default Frequency: 800 kHz

Default DMAChannel: 10
*/
func New(driverType DriverType) (*Config, error) {
	c := &Config{
		driverType:  driverType,
		initialized: false,
		dmaChannel:  10,
		frequency:   800000,
	}
	switch c.driverType {
	case DriverPWM:
		c.channels = make([]ledChannel, 2, 2)
	default:
		return nil, errors.Wrap(ErrDriverNotSupported, "New")
	}
	return c, nil
}

//Initialize activates a Config. If another Config for the same driverType is already active, an error is returned
func (c *Config) Initialize() error {
	if c.initialized {
		return nil
	}
	var oneActive bool
	for _, curChannel := range c.channels {
		if curChannel.active {
			oneActive = true
			break
		}
	}
	if !oneActive {
		return errors.Wrap(ErrNoActiveChannel, "config initialize")
	}
	//Initialize GPIO. Does not matter if already done.
	logOutput("Initializing GPIO package")
	err := rpigpio.Initialize()
	if err != nil {
		return errors.Wrap(err, "config initialize")
	}
	logOutput("Done with GPIO package")
	curHardware, err = rpihardware.Check()
	if err != nil {
		return errors.Wrap(err, "config initialize")
	}
	logOutput("Initializing DMA peripheral")
	err = initializeDMA()
	if err != nil {
		return errors.Wrap(err, "config initialize")
	}
	logOutput("Done with DMA peripheral")
	err = enableDMA(c.dmaChannel)
	if err != nil {
		return errors.Wrap(err, "config initialize")
	}
	logOutput("Enabled DMA channel")
	switch c.driverType {
	case DriverPWM:
		if pwmActive {
			return errors.Wrap(ErrDriverAlreadyUsed, "pwm")
		}
		pwmActive = true
		//Initialize and start PWM. This will also setup the clock
		logOutput("Initializing PWM")
		err := initializePWM(c.channels, c.frequency)
		if err != nil {
			return errors.Wrap(err, "config initialize")
		}
		logOutput("Done PWM")
		//Initialize gpio pin and set mode per channel
		for curChannelID, curChannel := range c.channels {
			if curChannel.active {
				//err was checked during SetStrip
				altMode, _ := pwmChannels.getAltMode(curChannelID, curChannel.pin.UInt32())
				curChannel.pin.Mode(altMode)
			}
		}
	default:
		return errors.Wrap(ErrDriverNotSupported, "config initialize")
	}
	c.initialized = true
	return nil
}

//Stop will dsable the Config so that another Config of same driverType can be initialized.
//Stopping is also needed if changing of fundamental settings is desired.
func (c *Config) Stop() error {
	if !c.initialized {
		return nil
	}
	switch c.driverType {
	case DriverPWM:
		pwmActive = false
		err := cleanupPWM()
		if err != nil {
			return errors.Wrap(err, "config Stop")
		}
	case DriverPCM:
		pcmActive = false
	case DriverSPI:
		spiActive = false
	}
	//Don't stop DMA as other config might use it.
	c.initialized = false
	for _, curChan := range c.channels {
		if curChan.active {
			fmt.Println("Setting pinmode")
			curChan.pin.Mode(rpigpio.ModeOut)
			curChan.pin.Set(0)
		}
	}
	return nil
}

//SetDMAChannel sets the DMAChannel to use. Default is 10.
/*
If you want to use multiple Config, you can use the same DMAChannel IF you don't render the Config at the same time.

There are some DMAChannels used by the system. Please check only if you want to use another channel as 10.
*/
func (c *Config) SetDMAChannel(channel uint32) error {
	if c.initialized {
		return errors.Wrap(ErrConfigInitialized, "config SetDMAChannel")
	}
	c.dmaChannel = channel
	return nil
}

//SetFrequency sets the output frequency to use. Valid values are 400000 and 800000
func (c *Config) SetFrequency(frequency uint32) error {
	if c.initialized {
		return errors.Wrap(ErrConfigInitialized, "config SetFrequency")
	}
	if frequency != 400000 && frequency != 800000 {
		return errors.Wrap(ErrWrongFrequency, "config SetFrequency")
	}
	c.frequency = frequency
	return nil
}

//SetBrightness sets the output brightness to use for the strip with index stripIndex. Valid values are between 0 and 255.
//This method can be called once the Config is initialized.
func (c *Config) SetBrightness(brightness uint32, stripIndex int) error {
	if stripIndex < 0 {
		return errors.Wrap(ErrConfigWrongIndex, "config SetBrightness")
	}
	switch c.driverType {
	case DriverPWM:
		if stripIndex >= 2 {
			return errors.Wrap(ErrConfigWrongIndex, "config SetBrightness")
		}
		c.channels[stripIndex].brightness = brightness
		logOutput(fmt.Sprintf("Setting brightness of strip: %d\n", c.channels[stripIndex].brightness))
	}
	return nil
}

//SetStrip adds LEDs to the Config.
/*
*****

A word about channels:

A channel represents a physical signal output on the Raspberry Pi.

There are two simultaniously available channels when using PWM as a driver. Otherwise there is only one channel available.

*****

For PWM the stripIndex can be 0 or 1. All other drivers require a stripIndex of 0.
The pin is checked if it is suitable for the driverType.
*/
func (c *Config) SetStrip(ledStrip LEDs, pin uint32, stripType StripType, stripIndex int, invertSignal bool) error {
	if c.initialized {
		return errors.Wrap(ErrConfigInitialized, "config SetStrip")
	}
	if stripIndex < 0 {
		return errors.Wrap(ErrConfigWrongIndex, "config SetStrip")
	}
	switch c.driverType {
	case DriverPWM:
		if stripIndex >= 2 {
			return errors.Wrap(ErrConfigWrongIndex, "config SetStrip")
		}
		//PWM may have two channels. Use stripIndex
		curChannel := &c.channels[stripIndex]
		//Checking if pin is allowed for this channel
		_, err := pwmChannels.getAltMode(stripIndex, pin)
		if err != nil {
			return err
		}
		curChannel.pin, err = rpigpio.NewPin(pin)
		if err != nil {
			return err
		}
		curChannel.strip = ledStrip
		curChannel.stripType = stripType
		curChannel.invert = invertSignal
		curChannel.active = true
		curChannel.wshift = uint8((stripType >> 24) & 0xff)
		curChannel.rshift = uint8((stripType >> 16) & 0xff)
		curChannel.gshift = uint8((stripType >> 8) & 0xff)
		curChannel.bshift = uint8((stripType >> 0) & 0xff)
	}
	return nil
}

//stripIndex -1 renders all stripes for that driver
func (c *Config) Render(stripIndex int) error {
	if stripIndex < -1 {
		return errors.Wrap(ErrConfigWrongIndex, "")
	}
	switch c.driverType {
	case DriverPWM:
		if stripIndex >= 2 {
			return errors.Wrap(ErrConfigWrongIndex, "")
		}
		if c.renderWaitTime != 0 && !c.previousRenderTime.IsZero() {
			timeDiff := time.Now().Sub(c.previousRenderTime)
			if timeDiff.Microseconds() < c.renderWaitTime {
				time.Sleep(time.Duration((c.renderWaitTime - timeDiff.Microseconds()) * 1000))
			}
		}
		var err error
		if stripIndex == -1 {
			c.renderWaitTime, err = renderPWM(c.channels, c.frequency)
		} else {
			c.renderWaitTime, err = renderPWM(c.channels[stripIndex:stripIndex+1], c.frequency)
		}
		if err != nil {
			return err
		}
		err = startDMA(c.dmaChannel, dmaCBRegisterMemPWM.BusAddr())
		if err != nil {
			return err
		}
		c.previousRenderTime = time.Now()
	}

	return nil
}
