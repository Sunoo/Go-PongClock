package main

import (
	"time"
	"os"
	"net"
	"bytes"
	"image"
	"image/draw"
	"image/color"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"github.com/Sunoo/go-rpi-rgb-led-matrix"
	"github.com/brutella/hc"
	"github.com/brutella/hc/accessory"
	"github.com/Sunoo/hsv"
	"github.com/lmittmann/ppm"
	"github.com/toelsiba/fopix"
)

var (
	clockConfig = ClockConfig{hsv.HSVColor{120, 100, 30}, 0.25, "03:04:05", 40 * time.Millisecond, time.Second / 2, 16, 32, 1, 1, "regular", false, false, false}
	matrix rgbmatrix.Matrix
	canvas *rgbmatrix.Canvas
	power = true
	stopchan chan bool
	stoppedchan chan bool
	font *fopix.Font
)

const (
	bat1_x = 2
	bat2_x = 28
)

type ClockConfig struct {
	ClockColor hsv.HSVColor
	DimAmount float32
	TimeFormat string
	AnimSpeed time.Duration
	ClockReturn time.Duration
	Rows int
	Cols int
	Parallel int
	ChainLength int
	HardwareMapping string
	ShowRefreshRate bool
	InverseColors bool
	DisableHardwarePulsing bool
}

func random(min int, max int) int {
	return min + rand.Intn(max - min)
}

func drawRect(x int, y int, w int, h int, drawColor color.RGBA) {
	for curX := x; curX < x + w; curX++ {
		for curY := y; curY < y + h; curY++ {
			canvas.Set(curX, curY, drawColor)
		}
	}
}

func guessEndpoint(ballpos_x int, ballpos_y float32, ballvel_x int, ballvel_y float32) int {
	for ballpos_x > bat1_x + 1 && ballpos_x < bat2_x {
		ballpos_x += ballvel_x
		ballpos_y += ballvel_y
		if ballpos_y <= 0 || ballpos_y >= 15 {
			ballvel_y *= -1
		}
	}
	return int(ballpos_y)
}

func runClock() {
	ballpos_x := 16
	ballpos_y := float32(8)
	ballvel_x := 0
	ballvel_y := float32(0)
	bat1_y := 5
	bat2_y := 5
	bat1_target_y := 5
	bat2_target_y := 5
	bat1miss := false
	bat2miss := false
	restart := 25
	holdTime := false
	var clock []rune
	bounds := canvas.Bounds()
	
	for {
		colorR, colorG, colorB, _ := clockConfig.ClockColor.RGBA()
		fullColor := color.RGBA{uint8(colorR), uint8(colorG), uint8(colorB), 255}
		dimColor := color.RGBA{uint8(float32(colorR) * clockConfig.DimAmount), uint8(float32(colorG) * clockConfig.DimAmount), uint8(float32(colorB) * clockConfig.DimAmount), 255}
		
		curTime := time.Now().Add(time.Second)
		if !holdTime {
			clock = []rune(curTime.Format(clockConfig.TimeFormat))
		}
		
        mins := curTime.Minute()
        seconds := curTime.Second()
		
		for i := 0; i < 16; i++ {
			canvas.Set(16, i, dimColor)
		}
			
		font.Color(dimColor)
		font.DrawRune(canvas, image.Point{8, 1}, clock[0])
		font.DrawRune(canvas, image.Point{12, 1}, clock[1])
		font.DrawRune(canvas, image.Point{18, 1}, clock[3])
		font.DrawRune(canvas, image.Point{22, 1}, clock[4])
		
		if restart > 0 {
			ballpos_x = 16
			if restart == 1 {
				ballpos_y = float32(random(4,11))
				
				if random(0, 1) > 0 {
					ballvel_x = 1
				} else {
					ballvel_x = -1
				}
				if random(0, 1) > 0 {
					ballvel_y = 0.5
				} else {
					ballvel_y = -0.5
				}
			}
			bat1miss = false
			bat2miss = false
			holdTime = false
			restart--
		}
		
		if seconds == 59 {
			holdTime = true
		}
		
		if seconds == 0 {
			if mins > 0 {
				bat1miss = true
			} else {
				bat2miss = true
			}
		}
		
		if ballpos_x == random(18, 31) {
			bat1_target_y = int(ballpos_y)
		}
		if ballpos_x == random(4, 15) {
			bat2_target_y = int(ballpos_y)
		}
		
		if ballpos_x == 15 && ballvel_x < 0 {
			end_ball_y := guessEndpoint(ballpos_x, ballpos_y, ballvel_x, ballvel_y)
			
			if bat1miss {
				bat1miss = false
				if end_ball_y > 8 {
					bat1_target_y = random(0, 2)
				} else {
					bat1_target_y = 8 + random(0, 2)
				}
			} else {
				bat1_target_y = end_ball_y - random(1, 4)
				if bat1_target_y < 0 {
					bat1_target_y = 0
				}
				if bat1_target_y > 10 {
					bat1_target_y = 10
				}
			}
		}
		
		if ballpos_x == 17 && ballvel_x > 0 {
			end_ball_y := guessEndpoint(ballpos_x, ballpos_y, ballvel_x, ballvel_y)
			
			if bat2miss {
				bat2miss = false
				if end_ball_y > 8 {
					bat2_target_y = random(0, 2)
				} else {
					bat2_target_y = 8 + random(0, 2)
				}
			} else {
				bat2_target_y = end_ball_y - random(1, 4)
				if bat2_target_y < 0 {
					bat2_target_y = 0
				}
				if bat2_target_y > 10 {
					bat2_target_y = 10
				}
			}
		}
		
		if bat1_y > bat1_target_y && bat1_y > 0 {
			bat1_y--
		}
		
		if bat1_y < bat1_target_y && bat1_y < 10 {
			bat1_y++
		}
		
		drawRect(bat1_x - 1, bat1_y, 2, 6, fullColor)
		
		if bat2_y > bat2_target_y && bat2_y > 0 {
			bat2_y--
		}
		
		if bat2_y < bat2_target_y && bat2_y < 10 {
			bat2_y++
		}
		
		drawRect(bat2_x + 1, bat2_y, 2, 6, fullColor)
		
		ballpos_x += ballvel_x
		ballpos_y += ballvel_y
		
		if ballpos_y <= 0 {
			ballvel_y *= -1
			ballpos_y = 0
		}
		
		if ballpos_y >=15 {
			ballvel_y *= -1
			ballpos_y = 15
		}
		
		if ballpos_x == bat1_x + 1 && bat1_y <= int(ballpos_y) && int(ballpos_y) <= bat1_y + 5 {
			if random(0, 2) == 0 {
				ballvel_x *= -1
			} else {
				var flick int
				
				if bat1_y > 1 || bat1_y < 8 {
					flick = random(0, 1)
				}
				
				if bat1_y <= 1 {
					flick = 0
				}
				if bat1_y >= 8 {
					flick = 1
				}
				
				switch flick {
					case 0:
						bat1_target_y += random(1, 2)
						ballvel_x *= -1
						if ballvel_y < 2 {
							ballvel_y += 0.2
						}
						
					case 1:
						bat1_target_y -= random(1, 2)
						ballvel_x *= -1
						if ballvel_y > 0.2 {
							ballvel_y -= 0.2
						}
				}
			}
		}
		
		if ballpos_x == bat2_x && bat2_y <= int(ballpos_y) && int(ballpos_y) <= bat2_y + 5 {
			if random(0, 2) == 0 {
				ballvel_x *= -1
			} else {
				var flick int
				
				if bat2_y > 1 || bat2_y < 8 {
					flick = random(0, 1)
				}
				if bat2_y <= 1 {
					flick = 0
				}
				if bat2_y >= 8 {
					flick = 1
				}
				
				switch flick {
					case 0:
						bat2_target_y += random(0, 2)
						ballvel_x *= -1
						if ballvel_y < 2 {
							ballvel_y += 0.2
						}
					
					case 1:
						bat2_target_y -= random(0, 2)
						ballvel_x *= -1
						if ballvel_y > 0.2 {
							ballvel_y -= 0.2
						}
				}
			}
		}
		
		if restart == 0 {
			if ballpos_x >= bounds.Min.X && ballpos_x <= bounds.Max.X - 1 && int(ballpos_y + 0.5) >= bounds.Min.Y && int(ballpos_y + 0.5) <= bounds.Max.Y - 1 {
				canvas.Set(ballpos_x, int(ballpos_y + 0.5), fullColor)
			}
		}
		
		if ballpos_x < 0 || ballpos_x > 31 {
			restart =  25
			holdTime = false
		}
		
		if (power) {
			canvas.Render()
		}

		select {
			case <-time.After(clockConfig.AnimSpeed):
				//Just keep running
			case <-stopchan:
				canvas.Render()
				stoppedchan <- true
				return
		}
	}
}

func flaschen() {
	pc, err := net.ListenPacket("udp", ":1337")
	if err != nil {
		return
	}
	defer pc.Close()

	doneChan := make(chan error, 1)
	buffer := make([]byte, 65535)
	clockStopped := false

	f := func() {
		clockStopped = false
		matrix.SetBrightness(clockConfig.ClockColor.V)
		go runClock()
	}
	timer := time.AfterFunc(clockConfig.ClockReturn, f)
	timer.Stop()

	go func() {
		for {
			n, _, err := pc.ReadFrom(buffer)
			if err != nil {
				doneChan <- err
				return
			}
			
			if !clockStopped {
				matrix.SetBrightness(100)
				stopchan <- true
				<-stoppedchan
				clockStopped = true
			}
			
			timer.Reset(clockConfig.ClockReturn)
			
			img, err := ppm.Decode(bytes.NewReader(buffer[:n]))
			if err != nil {
				doneChan <- err
				return
			}
			
			draw.Draw(canvas, canvas.Bounds(), img, image.ZP, draw.Src)
			canvas.Render()
		}
	}()

	select {}
}

func main() {
	jsonConfig, _ := ioutil.ReadFile("config.json")
	json.Unmarshal(jsonConfig, &clockConfig)
	
	stopchan = make(chan bool)
	stoppedchan = make(chan bool)
	
	config := &rgbmatrix.DefaultConfig
	config.Rows = clockConfig.Rows
	config.Cols = clockConfig.Cols
	config.Parallel = clockConfig.Parallel
	config.ChainLength = clockConfig.ChainLength
	config.Brightness = clockConfig.ClockColor.V
	config.HardwareMapping = clockConfig.HardwareMapping
	config.ShowRefreshRate = clockConfig.ShowRefreshRate
	config.InverseColors = clockConfig.InverseColors
	config.DisableHardwarePulsing = clockConfig.DisableHardwarePulsing
	
	matrix, _ = rgbmatrix.NewRGBLedMatrix(config)

	canvas = rgbmatrix.NewCanvas(matrix)
	defer canvas.Close()
	
	info := accessory.Info{
		Name:         "Clock",
		SerialNumber: "rpi-rgb-led-matrix",
		Manufacturer: "Sunoo",
		Model:        "Pong Clock",
	}

	acc := accessory.NewLightbulb(info)
	
	acc.Lightbulb.On.SetValue(true)
	acc.Lightbulb.Brightness.SetValue(clockConfig.ClockColor.V)
	acc.Lightbulb.Saturation.SetValue(clockConfig.ClockColor.S)
	acc.Lightbulb.Hue.SetValue(clockConfig.ClockColor.H)
	
	acc.Lightbulb.On.OnValueRemoteUpdate(func(on bool) {
		if power != on {
			power = on;
			if on {
				go runClock()
			} else {
				stopchan <- true
				<-stoppedchan
			}
		}
	})
	
	acc.Lightbulb.Brightness.OnValueRemoteUpdate(func(bright int) {
		clockConfig.ClockColor.V = bright
		matrix.SetBrightness(bright)
		if (power) {
			canvas.Render()
		}
	})
	
	acc.Lightbulb.Saturation.OnValueRemoteUpdate(func(sat float64) {
		clockConfig.ClockColor.S = sat
		if (power) {
			canvas.Render()
		}
	})
	
	acc.Lightbulb.Hue.OnValueRemoteUpdate(func(hue float64) {
		clockConfig.ClockColor.H = hue
		if (power) {
			canvas.Render()
		}
	})

	t, _ := hc.NewIPTransport(hc.Config{Pin: "12312312"}, acc.Accessory)
	
	hc.OnTermination(func() {
		<-t.Stop()
		jsonConfig, _ := json.MarshalIndent(clockConfig, "", "\t")
		ioutil.WriteFile("config.json", jsonConfig, 0666)
		os.Exit(0)
	})

	go t.Start()
	
	rand.Seed(time.Now().UnixNano())
	
	var err error
	font, err = fopix.NewFromFile("digits-3x5.json")
	if err != nil {
		fatal(err)
	}
	
	go runClock()
	
	go flaschen()
	
	select {}
}

func fatal(err error) {
	if err != nil {
		panic(err)
	}
}