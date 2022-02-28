package rpiws281x

import (
	"fmt"
	"os"
	"time"

	"github.com/DerLukas15/rpigpio"
	"github.com/DerLukas15/rpihardware"
	"github.com/DerLukas15/rpimemmap"
	"github.com/pkg/errors"
)

/*
 * Pin map of alternate pin configuration for PWM
 * GPIO    PWM0   PWM1
 *  12      0
 *  13             0
 *  18      5
 *  19             5
 *  40      0
 *  41             0
 *  45             0
 *  52      1
 *  53             1
 */

const (
	pwmBitsPerOutputBit         = 3
	registerPWMBusOffset uint32 = 0x0020c000

	//Register Offsets
	registerOffsetPWMCtl  uint32 = 0x00 // Control
	registerOffsetPWMSta  uint32 = 0x04 // Status
	registerOffsetPWMDmac uint32 = 0x08 // DMA control
	registerOffsetPWMRng1 uint32 = 0x10 // Channel 1 range
	registerOffsetPWMDat1 uint32 = 0x14 // Channel 1 data
	registerOffsetPWMFif1 uint32 = 0x18 // Channel 1 fifo
	registerOffsetPWMRng2 uint32 = 0x20 // Channel 2 range
	registerOffsetPWMDat2 uint32 = 0x24 // Channel 2 data

	//PWM register values
	registerValuePWMDmacEnab uint32 = (1 << 31) // Start PWM DMA

	registerValuePWMCtlPwen1 uint32 = (1 << 0)  // Chan 1: enable
	registerValuePWMCtlMode1 uint32 = (1 << 1)  // Chan 1: mode
	registerValuePWMCtlRptl1 uint32 = (1 << 2)  // Chan 1: repeat last data when Fifo empty
	registerValuePWMCtlPola1 uint32 = (1 << 4)  // Chan 1: polarisation
	registerValuePWMCtlUsef1 uint32 = (1 << 5)  // Chan 1: use Fifo
	registerValuePWMCtlClrf1 uint32 = (1 << 6)  // Chan 1: clear fifo
	registerValuePWMCtlMsen1 uint32 = (1 << 7)  // Chan 1: m/s enable
	registerValuePWMCtlPwen2 uint32 = (1 << 8)  // Chan 2: enable
	registerValuePWMCtlMode2 uint32 = (1 << 9)  // Chan 2: mode
	registerValuePWMCtlRptl2 uint32 = (1 << 10) // Chan 2: repeat last data when Fifo empty
	registerValuePWMCtlPola2 uint32 = (1 << 12) // Chan 2: polarisation
	registerValuePWMCtlUsef2 uint32 = (1 << 13) // Chan 2: use Fifo
	registerValuePWMCtlMsen2 uint32 = (1 << 15) // Chan 2: m/s enable

	registerValuePWMStaSta1 uint32 = (1 << 9)  // Chan 1: status
	registerValuePWMStaSta2 uint32 = (1 << 10) // Chan 2: status
	registerValuePWMStaBerr uint32 = (1 << 8)  // Bus Error bit
)

var (
	registerValuePWMDmacPanic = func(val uint32) uint32 { return ((val & 0xff) << 8) }
	registerValuePWMDmacDreq  = func(val uint32) uint32 { return ((val & 0xff) << 0) }
)

var pwmRegisterMem rpimemmap.MemMap
var pwmDataMem rpimemmap.MemMap
var activePWMChannels uint32

type pwmPinDefinition struct {
	pinNum  *rpigpio.Pin
	altMode rpigpio.Mode
}

type pwmPinTable []pwmPinDefinition

type pwmChannelType []pwmPinTable

var (
	pin12, _    = rpigpio.NewPin(12)
	pin18, _    = rpigpio.NewPin(18)
	pin40, _    = rpigpio.NewPin(40)
	pin13, _    = rpigpio.NewPin(13)
	pin19, _    = rpigpio.NewPin(19)
	pin41, _    = rpigpio.NewPin(41)
	pin45, _    = rpigpio.NewPin(45)
	pwmChannels = pwmChannelType{
		pwmPinTable{ // Mapping of Pin to alternate function for PWM channel 0
			{
				pinNum:  pin12,
				altMode: rpigpio.ModeAlternate0,
			},
			{
				pinNum:  pin18,
				altMode: rpigpio.ModeAlternate5,
			},
			{
				pinNum:  pin40,
				altMode: rpigpio.ModeAlternate0,
			},
		},
		pwmPinTable{ // Mapping of Pin to alternate function for PWM channel 1
			{
				pinNum:  pin13,
				altMode: rpigpio.ModeAlternate0,
			},
			{
				pinNum:  pin19,
				altMode: rpigpio.ModeAlternate5,
			},
			{
				pinNum:  pin41,
				altMode: rpigpio.ModeAlternate0,
			},
			{
				pinNum:  pin45,
				altMode: rpigpio.ModeAlternate0,
			},
		},
	}
)

func (pt pwmPinDefinition) getAltMode() (rpigpio.Mode, error) {
	return pt.altMode, nil
}

func (ptl pwmPinTable) getAltMode(pinNum uint32) (rpigpio.Mode, error) {
	for _, curEntry := range ptl {
		if curEntry.pinNum.Is(pinNum) {
			return curEntry.getAltMode()
		}
	}
	return 0, errors.Wrap(ErrPinNotAllowed, "")
}

func (pc pwmChannelType) getAltMode(chanNum int, pinNum uint32) (rpigpio.Mode, error) {
	return pc[chanNum].getAltMode(pinNum)
}

//initializes pwm specific stuff like clock, pwm data, pwm device, dma cb
func initializePWM(channels []ledChannel, frequency uint32) error {
	var err error
	logOutput("Initializing clock peripheral")
	err = initializeClock()
	if err != nil {
		return errors.Wrap(err, "PWM init")
	}
	logOutput("Done clock")
	logOutput("Setup PWM clock")
	err = clockSetupPwm(frequency)
	if err != nil {
		return err
	}
	logOutput("Done PWM clock")

	if pwmRegisterMem == nil {
		logOutput("Initializing PWM")
		pwmRegisterMem = rpimemmap.NewPeripheral(uint32(os.Getpagesize()))
		err := pwmRegisterMem.Map(registerPWMBusOffset, rpimemmap.MemDevDefault, 0)
		if err != nil {
			return err
		}
		logOutput("Done pwm")
		logOutput("PWM: " + pwmRegisterMem.String())
	}
	*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMRng1) = 32 //32-bits per word to serialize
	*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMRng2) = 32 //32-bits per word to serialize
	time.Sleep(10 * time.Microsecond)
	*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl) = registerValuePWMCtlClrf1 // Clear Fifo
	time.Sleep(10 * time.Microsecond)
	*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMDmac) = registerValuePWMDmacEnab | registerValuePWMDmacPanic(7) | registerValuePWMDmacDreq(3)
	time.Sleep(10 * time.Microsecond)
	var ctlSettings uint32
	if channels[0].active || PWMAlwaysUseTwoChannel {
		ctlSettings |= registerValuePWMCtlUsef1 | registerValuePWMCtlMode1
		if channels[0].invert {
			ctlSettings |= registerValuePWMCtlPola1
		}
	}
	if channels[1].active || PWMAlwaysUseTwoChannel {
		ctlSettings |= registerValuePWMCtlUsef2 | registerValuePWMCtlMode2
		if channels[1].invert {
			ctlSettings |= registerValuePWMCtlPola1
		}
	}
	*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl) = ctlSettings
	time.Sleep(10 * time.Microsecond)
	if channels[0].active || PWMAlwaysUseTwoChannel {
		*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl) |= registerValuePWMCtlPwen1
	}
	if channels[1].active || PWMAlwaysUseTwoChannel {
		*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl) |= registerValuePWMCtlPwen2
	}
	time.Sleep(10 * time.Microsecond)

	if pwmDataMem != nil {
		logOutput("Clearing PWM data storage")
		err = pwmDataMem.Unmap()
		if err != nil {
			return err
		}
		logOutput("Done clearing")
	}

	var dataSize uint32
	for _, curChannel := range channels {
		if curChannel.active {
			ledColors := 3 //Assume 3 colors per LED
			// If our shift mask includes the highest nibble, then we have 4 LEDs, RBGW.
			if (uint(curChannel.stripType) & sk6812ShiftMask) != 0 {
				ledColors = 4
			}
			ledBitCount := curChannel.strip.TotalCount() * ledColors * 8 * pwmBitsPerOutputBit // Each LED has 8 Bit per Color which are each mapped to pwmBitsPerOutputBit Bits
			thisByteCount := (uint32(ledBitCount>>3) & ^uint32(0x7)) + 8
			//Larger channel will be the reference
			thisByteCount += 32 //Spacing so that leds render. Don't know yet how to calculate
			if thisByteCount > dataSize {
				dataSize = thisByteCount
			}
			activePWMChannels++
		}
	}
	if PWMAlwaysUseTwoChannel {
		activePWMChannels = 2
	}
	dataSize *= activePWMChannels
	logOutput(fmt.Sprintf("Initializing PWM data storage. size: %d bytes", dataSize))
	pwmDataMem = rpimemmap.NewUncached(dataSize)
	allocationFlags := rpimemmap.UncachedMemFlagDirect
	if curHardware.RPiType == rpihardware.RPiType1 {
		allocationFlags = 0xc
	}
	err = pwmDataMem.Map(0, "", allocationFlags)
	if err != nil {
		return err
	}
	logOutput("Done PWM data storage")
	logOutput("PWM storage: " + pwmDataMem.String())

	//Initialize dma control block for pwm
	logOutput("Initializing DmaCB")
	err = initializeDmaCBPWM(dataSize)
	if err != nil {
		return err
	}
	logOutput("Done DmaCB")
	if Debug {
		logOutput(statusPWM())
	}
	return nil
}

//stops pwm and deallocates memory
func cleanupPWM() error {
	err := stopPWM()
	if err != nil {
		return errors.Wrap(err, "cleanup pwm")
	}
	if pwmRegisterMem != nil {
		err := pwmRegisterMem.Unmap()
		if err != nil {
			return errors.Wrap(err, "cleanup pwm")
		}
		pwmRegisterMem = nil
	}
	if pwmDataMem != nil {
		err := pwmDataMem.Unmap()
		if err != nil {
			return errors.Wrap(err, "cleanup pwm data")
		}
		pwmDataMem = nil
	}
	err = cleanupDmaCBPWM()
	if err != nil {
		return errors.Wrap(err, "cleanup pwm dma cb")
	}
	activePWMChannels = 0
	return nil
}

func stopPWM() error {
	if pwmRegisterMem == nil {
		return nil
	}
	*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl) = 0
	//Stop PWM clock
	err := stopClockPWM()
	if err != nil {
		return err
	}
	return nil
}

//prints status of pwm device
func statusPWM() string {
	if pwmRegisterMem == nil {
		return ""
	}
	var res string
	res += fmt.Sprintf("PWM Status:\n")
	res += fmt.Sprintf("\tChan1: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMSta)&registerValuePWMStaSta1 != 0))
	res += fmt.Sprintf("\tChan2: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMSta)&registerValuePWMStaSta2 != 0))
	res += fmt.Sprintf("\tBuserror: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMSta)&registerValuePWMStaBerr != 0))
	res += fmt.Sprintf("PWM Chan 1:\n")
	res += fmt.Sprintf("\tEnabled: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlPwen1 != 0))
	res += fmt.Sprintf("\tUse Serialiser: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlMode1 != 0))
	res += fmt.Sprintf("\tRepeat: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlRptl1 != 0))
	res += fmt.Sprintf("\tInverse: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlPola1 != 0))
	res += fmt.Sprintf("\tUse Fifo: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlUsef1 != 0))
	res += fmt.Sprintf("\tUse M/S: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlMsen1 != 0))
	res += fmt.Sprintf("PWM Chan 2:\n")
	res += fmt.Sprintf("\tEnabled: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlPwen2 != 0))
	res += fmt.Sprintf("\tUse Serialiser: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlMode2 != 0))
	res += fmt.Sprintf("\tRepeat: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlRptl2 != 0))
	res += fmt.Sprintf("\tInverse: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlPola2 != 0))
	res += fmt.Sprintf("\tUse Fifo: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlUsef2 != 0))
	res += fmt.Sprintf("\tUse M/S: %t\n", (*rpimemmap.Reg32(pwmRegisterMem, registerOffsetPWMCtl)&registerValuePWMCtlMsen2 != 0))
	return res
}

//outputs signals with PWM for the given channels
func renderPWM(channels []ledChannel, frequency uint32) (int64, error) {
	var protocolTime uint32
	var bitPos int
	bitPos = 31
	for curChanID, curChannel := range channels {
		if !curChannel.active {
			//In case -1 was supplied during render call
			continue
		}
		var wordPos uint32
		wordPos = uint32(curChanID)
		scale := uint32((curChannel.brightness & 0xff)) + 1

		//2.5 uS per bit to led @ 400000
		//1.25 uS per bit to led @ 800000
		bitTime := float32(2.5)
		if frequency == 800000 {
			bitTime = float32(1.25)
		}
		ledColors := 3 //Assume 3 colors per LED
		// If our shift mask includes the highest nibble, then we have 4 LEDs, RBGW.
		if (uint(curChannel.stripType) & sk6812ShiftMask) != 0 {
			ledColors = 4
		}
		channelProtocolTime := uint32(float32(curChannel.strip.TotalCount()*ledColors*8) * bitTime)
		if channelProtocolTime > protocolTime {
			protocolTime = channelProtocolTime
		}
		for i := 0; i < curChannel.strip.TotalCount(); i++ {
			curLED := curChannel.strip.UInt32(i)
			var color = []uint8{
				gammaTable[(((curLED>>curChannel.rshift)&0xff)*scale)>>8],
				gammaTable[(((curLED>>curChannel.gshift)&0xff)*scale)>>8],
				gammaTable[(((curLED>>curChannel.bshift)&0xff)*scale)>>8],
				gammaTable[(((curLED>>curChannel.wshift)&0xff)*scale)>>8],
			}
			for j := 0; j < ledColors; j++ {
				curColor := color[j]
				for k := 7; k >= 0; k-- { // Bit per Color
					// Inversion is handled by hardware for PWM, otherwise by software here
					symbol := symbolLow
					if (curColor & (1 << k)) != 0 {
						symbol = symbolHigh
					}
					for l := 2; l >= 0; l-- { // Bit per Symbol
						*rpimemmap.Reg32(pwmDataMem, wordPos*4) &= ^(1 << bitPos)
						if (symbol & (1 << l)) != 0 {
							*rpimemmap.Reg32(pwmDataMem, wordPos*4) |= (1 << bitPos)
						}
						bitPos--
						if bitPos < 0 {
							// Every other word is on the same channel for PWM if two channels are active and with fifo
							if activePWMChannels == 2 {
								wordPos += 2
							} else {
								wordPos++
							}
							bitPos = 31
						}
					}
				}
			}
		}
	}
	//fmt.Println(rpimemmap.Dump(pwmDataMem, 0))
	// 300: Minimum time to wait for reset to occur in microseconds
	return int64(protocolTime + 300), nil
}
