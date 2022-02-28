package rpiws281x

import "image/color"

//SingleLED describes one LED with color values as 0xWWRRGGBB which can be used as an LEDs.
type SingleLED uint32 //0xWWRRGGBB

//ColorToSingleLED turns a color.Color to a SingleLED.
func ColorToSingleLED(c color.Color) *SingleLED {
	// A color's RGBA method returns values in the range [0, 65535]
	red, green, blue, alpha := c.RGBA()
	res := SingleLED((alpha>>8)<<24 | (red>>8)<<16 | (green>>8)<<8 | blue>>8)
	return &res
}

//RGBAtoSingleLED turns rgba valued to a SingleLED.
func RGBAtoSingleLED(r, g, b, a uint32) *SingleLED {
	res := SingleLED(a<<24 | r<<16 | g<<8 | b)
	return &res
}

//ToColor returns a color.Color object of the SingleLED.
func (l *SingleLED) ToColor() color.Color {
	return color.RGBA{
		uint8(*l>>16) & 255,
		uint8(*l>>8) & 255,
		uint8(*l>>0) & 255,
		uint8(*l>>24) & 255,
	}
}

//SetColor sets the color from color.Color.
func (l *SingleLED) SetColor(c color.Color) {
	red, green, blue, alpha := c.RGBA()
	*l = SingleLED((alpha>>8)<<24 | (red>>8)<<16 | (green>>8)<<8 | blue>>8)
}

//SetRGBA sets the color value by r, g, b, and a values.
func (l *SingleLED) SetRGBA(r, g, b, a uint32) {
	*l = SingleLED(a<<24 | r<<16 | g<<8 | b)
}

//SetDirect sets the color value directly. Format 0xWWRRGGBB
func (l *SingleLED) SetDirect(val uint32) {
	*l = SingleLED(val)
}

//Red returns the red color amount.
func (l *SingleLED) Red(unused int) uint8 {
	return uint8(*l>>16) & 255
}

//Green returns the green color amount.
func (l *SingleLED) Green(unused int) uint8 {
	return uint8(*l>>8) & 255
}

//Blue returns the blue color amount.
func (l *SingleLED) Blue(unused int) uint8 {
	return uint8(*l) & 255
}

//White returns the white color amount.
func (l *SingleLED) White(unused int) uint8 {
	return uint8(*l>>24) & 255
}

//UInt32 returns the color as uint32. Format 0xWWRRGGBB
func (l *SingleLED) UInt32(unused int) uint32 {
	return uint32(*l)
}

//TotalCount returns the number of LEDs.
//
//In this case this will always be 1.
func (l *SingleLED) TotalCount() int {
	return 1
}
