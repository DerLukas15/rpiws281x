package rpiws281x

import (
	"os"
	"time"

	"github.com/DerLukas15/rpimemmap"
)

const (
	registerClockBusOffset uint32 = 0x00101000

	//register offsets
	registerOffsetClkCtl    uint32 = 0x00 // Control
	registerOffsetClkPwmCtl uint32 = 0xa0 // PWM Control
	registerOffsetClkPcmCtl uint32 = 0x98 // PCM Control
	registerOffsetClkPwmDiv uint32 = 0xa4 // PWM Div
	registerOffsetClkPcmDiv uint32 = 0x9c // PCM Div

	// Clock register values
	registerValueClkPasswd uint32 = 0x5a000000 // Password used by some registers

	//Ctl register
	registerValueClkCtlSrcOsc uint32 = (1 << 0) // Set Source from oscillator
	registerValueClkCtlEnab   uint32 = (1 << 4) // Enable clock
	registerValueClkCtlKill   uint32 = (1 << 5) // Kill and reset
	registerValueClkCtlBusy   uint32 = (1 << 7) // Set if running
)

//helper functions for the clock register
var (
	registerValueClkCtlMash = func(val uint32) uint32 { return ((val & 0x3) << 9) }
	registerValueClkDivDivi = func(val uint32) uint32 { return ((val & 0xfff) << 12) } // Integer part of devisor
	registerValueClkDivDivf = func(val uint32) uint32 { return ((val & 0xfff) << 0) }  // Fractional part of devisor
)

var clockRegisterMem rpimemmap.MemMap //stores reference to clock device

//Setup the pwm clock for correct frequency
func clockSetupPwm(frequency uint32) error {
	stopClockPWM()
	*rpimemmap.Reg32(clockRegisterMem, registerOffsetClkPwmDiv) = registerValueClkPasswd | registerValueClkDivDivi(curHardware.OscFreq/(pwmBitsPerOutputBit*frequency))
	*rpimemmap.Reg32(clockRegisterMem, registerOffsetClkPwmCtl) = registerValueClkPasswd | registerValueClkCtlSrcOsc
	*rpimemmap.Reg32(clockRegisterMem, registerOffsetClkPwmCtl) = registerValueClkPasswd | registerValueClkCtlSrcOsc | registerValueClkCtlEnab
	time.Sleep(10 * time.Microsecond)
	//Wait for clock to setup
	logOutput("Waiting for clock to start")
	for (*rpimemmap.Reg32(clockRegisterMem, registerOffsetClkPwmCtl) & registerValueClkCtlBusy) == 0 {
		time.Sleep(1 * time.Microsecond)
	}
	logOutput("Done waiting")
	return nil
}

//stops the pwm clock
func stopClockPWM() error {
	if clockRegisterMem == nil {
		return nil
	}
	*rpimemmap.Reg32(clockRegisterMem, registerOffsetClkPwmCtl) = registerValueClkPasswd | registerValueClkCtlKill
	time.Sleep(10 * time.Microsecond)
	logOutput("Waiting for clock to stop")
	for (*rpimemmap.Reg32(clockRegisterMem, registerOffsetClkPwmCtl) & registerValueClkCtlBusy) != 0 {
		time.Sleep(1 * time.Microsecond)
	}
	logOutput("Done waiting")
	return nil
}

//Initialize the clock device and get virtual address
func initializeClock() error {
	if clockRegisterMem != nil {
		//Already initialized
		logOutput("Clock already initialized. Skipping")
		return nil
	}
	clockRegisterMem = rpimemmap.NewPeripheral(uint32(os.Getpagesize()))
	err := clockRegisterMem.Map(registerClockBusOffset, rpimemmap.MemDevDefault, 0)
	if err != nil {
		return err
	}
	logOutput("Clock: " + clockRegisterMem.String())
	return nil
}

//cleanup the mapping to the clock device
func cleanupClock() error {
	if clockRegisterMem == nil {
		return nil
	}
	err := clockRegisterMem.Unmap()
	if err != nil {
		return err
	}
	clockRegisterMem = nil
	return nil
}
