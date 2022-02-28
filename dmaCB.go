package rpiws281x

import (
	"os"

	"github.com/DerLukas15/rpihardware"
	"github.com/DerLukas15/rpimemmap"
)

const (
	//register offsets dmaCB
	registerOffsetDmaCBTi             uint32 = 0 * 4
	registerOffsetDmaCBSrcAddress     uint32 = 1 * 4
	registerOffsetDmaCBDestAddress    uint32 = 2 * 4
	registerOffsetDmaCBTransferLength uint32 = 3 * 4
	registerOffsetDmaCB2DModeStride   uint32 = 4 * 4
	registerOffsetDmaCBNextCBAddress  uint32 = 5 * 4

	//register values dmaCB Ti
	registerValueDmaCBTiInten        uint32 = 1 << 0
	registerValueDmaCBTiTdMode       uint32 = 1 << 1
	registerValueDmaCBTiWaitResp     uint32 = 1 << 3
	registerValueDmaCBTiDestInc      uint32 = 1 << 4
	registerValueDmaCBTiDestWidth    uint32 = 1 << 5
	registerValueDmaCBTiDestDreq     uint32 = 1 << 6
	registerValueDmaCBTiDestIgnore   uint32 = 1 << 7
	registerValueDmaCBTiSrcInc       uint32 = 1 << 8
	registerValueDmaCBTiSrcWidth     uint32 = 1 << 9
	registerValueDmaCBTiSrcDreq      uint32 = 1 << 10
	registerValueDmaCBTiSrcIgnore    uint32 = 1 << 11
	registerValueDmaCBTiNoWideBursts uint32 = 1 << 26
)

//helper functions for the dmaCB register
var (
	registerValueDmaCBTiWaits = func(val uint32) uint32 {
		return ((val & 0x1f) << 21)
	}
	registerValueDmaCBTiPermap = func(val uint32) uint32 {
		return ((val & 0x1f) << 16)
	}
	registerValueDmaCBTiBurstLength = func(val uint32) uint32 {
		return ((val & 0x1f) << 12)
	}
)

var dmaCBRegisterMemPWM rpimemmap.MemMap //stores reference to the one dmaCB

//initialize dmaCB storage for PWM
func initializeDmaCBPWM(transferBytes uint32) error {
	if dmaCBRegisterMemPWM != nil {
		return nil
	}
	dmaCBRegisterMemPWM = rpimemmap.NewUncached(uint32(os.Getpagesize())) // will be rounded to next pageSize anyway
	allocationFlags := rpimemmap.UncachedMemFlagDirect
	if curHardware.RPiType == rpihardware.RPiType1 {
		allocationFlags = 0xc
	}
	err := dmaCBRegisterMemPWM.Map(0, "", allocationFlags)
	if err != nil {
		return err
	}
	logOutput("DMA control block: " + dmaCBRegisterMemPWM.String())
	*rpimemmap.Reg32(dmaCBRegisterMemPWM, registerOffsetDmaCBTi) = registerValueDmaCBTiNoWideBursts | registerValueDmaCBTiWaitResp | registerValueDmaCBTiDestDreq | registerValueDmaCBTiSrcInc | registerValueDmaCBTiPermap(5)
	*rpimemmap.Reg32(dmaCBRegisterMemPWM, registerOffsetDmaCBSrcAddress) = pwmDataMem.BusAddr()
	*rpimemmap.Reg32(dmaCBRegisterMemPWM, registerOffsetDmaCBDestAddress) = pwmRegisterMem.BusAddr() + registerOffsetPWMFif1
	*rpimemmap.Reg32(dmaCBRegisterMemPWM, registerOffsetDmaCBTransferLength) = transferBytes
	*rpimemmap.Reg32(dmaCBRegisterMemPWM, registerOffsetDmaCB2DModeStride) = 0
	*rpimemmap.Reg32(dmaCBRegisterMemPWM, registerOffsetDmaCBNextCBAddress) = 0
	return nil
}

//deallocates dmaCB for pwm
func cleanupDmaCBPWM() error {
	if dmaCBRegisterMemPWM == nil {
		return nil
	}
	err := dmaCBRegisterMemPWM.Unmap()
	if err != nil {
		return err
	}
	dmaCBRegisterMemPWM = nil
	return nil
}
