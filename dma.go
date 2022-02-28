package rpiws281x

import (
	"os"
	"time"

	"github.com/DerLukas15/rpimemmap"
)

const (
	//Register Offsets
	registerOffsetDmaCs       uint32 = 0x00
	registerOffsetDmaConblkAd uint32 = 0x04
	registerOffsetDmaTi       uint32 = 0x08
	registerOffsetDmaSrcAd    uint32 = 0x0c
	registerOffsetDmaDestAd   uint32 = 0x10
	registerOffsetDmaTxfrLen  uint32 = 0x14
	registerOffsetDmaStride   uint32 = 0x18
	registerOffsetDmaNexConbk uint32 = 0x1c
	registerOffsetDmaDebug    uint32 = 0x20
	registerOffsetDmaEnable   uint32 = 0xff0

	//Register values Cs
	registerValueDmaCsReset                    uint32 = (1 << 31)
	registerValueDmaCsAbort                    uint32 = (1 << 30)
	registerValueDmaCsDisdebug                 uint32 = (1 << 29)
	registerValueDmaCsWaitOutstandingWrites    uint32 = (1 << 28)
	registerValueDmaCsError                    uint32 = (1 << 8)
	registerValueDmaCsWaitingOutstandingWrites uint32 = (1 << 6)
	registerValueDmaCsDreqStopsDma             uint32 = (1 << 5)
	registerValueDmaCsPaused                   uint32 = (1 << 4)
	registerValueDmaCsDreq                     uint32 = (1 << 3)
	registerValueDmaCsInt                      uint32 = (1 << 2)
	registerValueDmaCsEnd                      uint32 = (1 << 1)
	registerValueDmaCsActive                   uint32 = (1 << 0)

	registerDMABusOffset uint32 = 0x00007000
)

//helper functions for dma register
var (
	registerValueDmaCsPanicPriority = func(val uint32) uint32 { return ((val & 0xf) << 20) }
	registerValueDmaCsPriority      = func(val uint32) uint32 { return ((val & 0xf) << 16) }

	registerValueDmaTxfrLenYlength = func(val uint32) uint32 { return ((val & 0xffff) << 16) }
	registerValueDmaTxfrLenXlength = func(val uint32) uint32 { return ((val & 0xffff) << 0) }

	registerValueDmaStrideDStride = func(val uint32) uint32 { return ((val & 0xffff) << 16) }
	registerValueDmaStrideSStride = func(val uint32) uint32 { return ((val & 0xffff) << 0) }

	registerOffsetDmaChannel = func(ch uint32, r uint32) uint32 { //Adds channel specific offset to address
		return uint32(ch*0x100) + r
	}
)

var dmaRegisterMem rpimemmap.MemMap //stores reference to dma device

//Initialize the dma device
func initializeDMA() error {
	if dmaRegisterMem != nil {
		//Already initialized
		logOutput("DMA already initialized. Skipping")
		return nil
	}
	dmaRegisterMem = rpimemmap.NewPeripheral(uint32(os.Getpagesize()))
	err := dmaRegisterMem.Map(registerDMABusOffset, rpimemmap.MemDevDefault, 0)
	if err != nil {
		return err
	}
	logOutput("DMA: " + dmaRegisterMem.String())
	return nil
}

//stop the dma channel
func stopDMA(channel uint32) error {
	if dmaRegisterMem == nil {
		return nil
	}
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaChannel(channel, registerOffsetDmaCs)) = registerValueDmaCsReset
	return nil
}

//enable a dma channel
func enableDMA(channel uint32) error {
	if dmaRegisterMem == nil {
		return nil
	}
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaEnable) |= (1 << channel)
	return nil
}

//disable a dma channel
func disableDMA(channel uint32) error {
	if dmaRegisterMem == nil {
		return nil
	}
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaEnable) &= ^(1 << channel)
	return nil
}

//deallocate dma device
func cleanupDMA() error {
	if dmaRegisterMem == nil {
		return nil
	}
	err := dmaRegisterMem.Unmap()
	if err != nil {
		return err
	}
	dmaRegisterMem = nil
	return nil
}

//starts a dma transfer with channel and dmaCBAddress
func startDMA(channel uint32, dmaCBAddress uint32) error {
	err := stopDMA(channel)
	if err != nil {
		return err
	}
	//logOutput(fmt.Sprintf("DMA CB bus address: 0x%x", dmaCBAddress))
	time.Sleep(10 * time.Microsecond)
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaChannel(channel, registerOffsetDmaCs)) = registerValueDmaCsInt | registerValueDmaCsEnd
	time.Sleep(10 * time.Microsecond)
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaChannel(channel, registerOffsetDmaConblkAd)) = dmaCBAddress
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaChannel(channel, registerOffsetDmaDebug)) = 7
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaChannel(channel, registerOffsetDmaCs)) = registerValueDmaCsWaitOutstandingWrites |
		registerValueDmaCsPanicPriority(15) | registerValueDmaCsPriority(15)
	*rpimemmap.Reg32(dmaRegisterMem, registerOffsetDmaChannel(channel, registerOffsetDmaCs)) |= registerValueDmaCsActive
	time.Sleep(20 * time.Microsecond)
	return nil
}
