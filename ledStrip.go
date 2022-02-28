package rpiws281x

import "image/color"

//LEDStrip represent a physical continious strip of LEDs.
//Each LED is represented by a SingleLED.
type LEDStrip struct {
	leds  []*SingleLED
	count int
}

//NewLEDStrip returns a LEDStrip with count LEDs. Colors can be set with the methods.
func NewLEDStrip(count int) *LEDStrip {
	l := &LEDStrip{}
	l.leds = make([]*SingleLED, count, count)
	for i := 0; i < count; i++ {
		var led SingleLED
		l.leds[i] = &led
	}
	l.count = count
	return l
}

//TotalCount returns the number of LEDs in the strip.
func (l *LEDStrip) TotalCount() int {
	return l.count
}

//Red returns the red color amount at position.
func (l *LEDStrip) Red(position int) uint8 {
	if position < 0 || position >= l.count {
		return 0
	}
	return l.leds[position].Red(0)
}

//Green returns the green color amount at position.
func (l *LEDStrip) Green(position int) uint8 {
	if position < 0 || position >= l.count {
		return 0
	}
	return l.leds[position].Green(0)
}

//Blue returns the blue color amount at position.
func (l *LEDStrip) Blue(position int) uint8 {
	if position < 0 || position >= l.count {
		return 0
	}
	return l.leds[position].Blue(0)
}

//White returns the white color amount at position.
func (l *LEDStrip) White(position int) uint8 {
	if position < 0 || position >= l.count {
		return 0
	}
	return l.leds[position].White(0)
}

//SetColor sets the color from color.Color at position.
func (l *LEDStrip) SetColor(position int, c color.Color) {
	if position < 0 || position >= l.count {
		return
	}
	l.leds[position].SetColor(c)
}

//SetRGBA sets the color value by r, g, b, and a values at position.
func (l *LEDStrip) SetRGBA(position int, r, g, b, a uint32) {
	if position < 0 || position >= l.count {
		return
	}
	l.leds[position].SetRGBA(r, g, b, a)
}

//SetDirect sets the color value for LED at position directly. Format 0xWWRRGGBB
func (l *LEDStrip) SetDirect(position int, val uint32) {
	if position < 0 || position >= l.count {
		return
	}
	l.leds[position].SetDirect(val)
}

//UInt32 returns the color as uint32. Format 0xWWRRGGBB
func (l *LEDStrip) UInt32(position int) uint32 {
	if position < 0 || position >= l.count {
		return 0
	}
	return l.leds[position].UInt32(0)
}

//ShiftRight shifts the LED colors by shift to the right. Everything leaving on the right wraps around.
//Use ShiftLeft instead of negative shitfs.
func (l *LEDStrip) ShiftRight(shift int) {
	if shift <= 0 || (shift%l.count) == 0 { //Don't need to shift on <=0 and when multiple of count
		return
	}
	actualShift := shift % l.count
	l.leds = append(l.leds[l.count-actualShift:], l.leds[0:l.count-actualShift]...)
}

//ShiftLeft shifts the LED colors by shift to the left. Everything leaving on the left wraps around.
//Use ShiftRight instead of negative shitfs.
func (l *LEDStrip) ShiftLeft(shift int) {
	if shift <= 0 || (shift%l.count) == 0 { //Don't need to shift on <=0 and when multiple of count
		return
	}
	actualShift := shift % l.count
	l.leds = append(l.leds[actualShift+1:], l.leds[0:actualShift]...)
}
