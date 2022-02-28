package rpiws281x

//LEDs can be a single LED or a slice depending on the implementation. Position defines the physical position on the LED strip starting at 0 to the total number of LEDs on that strip.
type LEDs interface {
	Red(position int) uint8
	Green(position int) uint8
	Blue(position int) uint8
	White(position int) uint8
	UInt32(position int) uint32 //Format 0xWWRRGGBB
	TotalCount() int
}
