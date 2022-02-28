# rpiws281x [![GoDoc](https://godoc.org/github.com/DerLukas15/rpiws281x?status.svg)](http://godoc.org/github.com/DerLukas15/rpiws281x)
Package for controlling WS281X LEDs on a Raspberry Pi using only plain GO.

## About

Currently only working with PWM. PCM and SPI are planned to be added later on.

The code is inspired by [github.com/jgarff/rpi_ws281x](https://github.com/jgarff/rpi_ws281x).

Currently only PWM is implemented!

## Installation

```sh
go get github.com/DerLukas15/rpiws281x
```

## Usage
For details please see GoDoc.

This should be the path to follow:
* create a Config for desired driver (PWM, PCM, SPI) (only PWM at the moment)
* create your own object for LED definition (interface LEDs) or use included LEDStrip struct
* set the strip in the Config
* Render the Config

## License

MIT, see [LICENSE](LICENSE)
