package main

import (
	"embed"
	"fmt"
	"image/color"
	_ "image/png"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/text"
	logging "github.com/tsujio/game-logging-server/client"
	"github.com/tsujio/game-util/dotutil"
	"github.com/tsujio/game-util/resourceutil"
	"github.com/tsujio/game-util/touchutil"
)

const (
	gameName               = "maxwells-demon"
	screenWidth            = 640
	screenHeight           = 480
	frameTop               = 70
	frameLeft              = 23
	frameRight             = screenWidth - frameLeft
	frameBottom            = screenHeight - 30
	frameThickness         = 15
	partitionSlitSize      = 50
	partitionSlitSpeed     = 1
	moleculeRadius         = 10
	moleculeSpeedHighValue = 4
	moleculeSpeedLowValue  = 3
)

//go:embed resources/*.ttf resources/*.dat
var resources embed.FS

func loadAudioData(name string, audioContext *audio.Context) []byte {
	f, err := resources.Open(name)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

var (
	largeFont, mediumFont, smallFont = (func() (l, m, s *resourceutil.Font) {
		l, m, s, err := resourceutil.LoadFont(resources, "resources/PressStart2P-Regular.ttf", nil)
		if err != nil {
			log.Fatal(err)
		}
		return
	})()
	audioContext      = audio.NewContext(48000)
	ngAudioData       = loadAudioData("resources/魔王魂 効果音 システム19.mp3.dat", audioContext)
	okAudioData       = loadAudioData("resources/魔王魂 効果音 ワンポイント26.mp3.dat", audioContext)
	startAudioData    = loadAudioData("resources/魔王魂 効果音 システム49.mp3.dat", audioContext)
	openAudioData     = loadAudioData("resources/魔王魂 効果音 物音05.mp3.dat", audioContext)
	completeAudioData = loadAudioData("resources/魔王魂 効果音 物音15.mp3.dat", audioContext)
)

type MoleculeSpeedType int

const (
	MoleculeSpeedHigh MoleculeSpeedType = iota
	MoleculeSpeedLow
)

var moleculeDrawingPattern = [][]int{
	{0, 0, 1, 1, 1, 0, 0},
	{0, 1, 1, 1, 1, 1, 0},
	{1, 1, 1, 1, 1, 1, 1},
	{1, 1, 1, 1, 1, 1, 1},
	{1, 1, 1, 1, 1, 1, 1},
	{0, 1, 1, 1, 1, 1, 0},
	{0, 0, 1, 1, 1, 0, 0},
}

var highSpeedMoleculeDrawingPattern = dotutil.CreatePatternImage(moleculeDrawingPattern, &dotutil.CreatePatternImageOption{
	Color: color.RGBA{0xfa, 0x25, 0x58, 0xff},
})

var lowSpeedMoleculeDrawingPattern = dotutil.CreatePatternImage(moleculeDrawingPattern, &dotutil.CreatePatternImageOption{
	Color: color.RGBA{0x7a, 0xa4, 0xff, 0xff},
})

type Molecule struct {
	speedType MoleculeSpeedType
	x, y      float64
	vx, vy    float64
	r         float64
}

func (m *Molecule) move() {
	m.x += m.vx
	m.y += m.vy
}

func (m *Molecule) draw(dst *ebiten.Image) {
	var img *ebiten.Image
	switch m.speedType {
	case MoleculeSpeedHigh:
		img = highSpeedMoleculeDrawingPattern
	case MoleculeSpeedLow:
		img = lowSpeedMoleculeDrawingPattern
	}
	w, _ := img.Size()
	dotutil.DrawImage(dst, img, m.x, m.y, &dotutil.DrawImageOption{
		Scale:        m.r * 2 / float64(w),
		BasePosition: dotutil.DrawImagePositionCenter,
	})
}

func (m *Molecule) reboundIfCollide(game *Game) {
	if m.x < frameLeft {
		m.x = frameLeft
		m.vx *= -1
	}
	if m.x > frameRight {
		m.x = frameRight
		m.vx *= -1
	}
	if m.y < frameTop {
		m.y = frameTop
		m.vy *= -1
	}
	if m.y > frameBottom {
		m.y = frameBottom
		m.vy *= -1
	}
	if m.x > screenWidth/2-frameThickness/2 && m.x-m.vx < screenWidth/2-frameThickness/2 {
		if !game.isPartitionSlitOpen || m.y <= game.partitionSlitTop || m.y >= game.partitionSlitBottom {
			m.x = screenWidth/2 - frameThickness/2
			m.vx *= -1
		}
	}
	if m.x < screenWidth/2+frameThickness/2 && m.x-m.vx > screenWidth/2+frameThickness/2 {
		if !game.isPartitionSlitOpen || m.y <= game.partitionSlitTop || m.y >= game.partitionSlitBottom {
			m.x = screenWidth/2 + frameThickness/2
			m.vx *= -1
		}
	}
}

type GameMode int

const (
	GameModeTitle GameMode = iota
	GameModeLevelStart
	GameModePlaying
	GameModeComplete
)

type Game struct {
	playID                                                             string
	mode                                                               GameMode
	ticksFromModeStart                                                 uint64
	touchContext                                                       *touchutil.TouchContext
	level                                                              int
	molecules                                                          []Molecule
	partitionSlitTop, partitionSlitBottom                              float64
	partitionSlitVec                                                   float64
	isPartitionSlitOpen                                                bool
	playStartTime                                                      time.Time
	playingSec                                                         int
	lowSpeedMoleculeCount, highSpeedMoleculeCount                      int
	lowSpeedMoleculeCountInLeftSide, highSpeedMoleculeCountInRightSide int
}

func (g *Game) updateMoleculeCountInCorrectArea(playAudio bool) {
	var lowCount, highCount int
	for i := 0; i < len(g.molecules); i++ {
		m := &g.molecules[i]
		if m.speedType == MoleculeSpeedLow && m.x < screenWidth/2 {
			lowCount++
		}
		if m.speedType == MoleculeSpeedHigh && m.x > screenWidth/2 {
			highCount++
		}
	}

	if playAudio {
		if lowCount > g.lowSpeedMoleculeCountInLeftSide || highCount > g.highSpeedMoleculeCountInRightSide {
			audio.NewPlayerFromBytes(audioContext, okAudioData).Play()
		}
		if lowCount < g.lowSpeedMoleculeCountInLeftSide || highCount < g.highSpeedMoleculeCountInRightSide {
			audio.NewPlayerFromBytes(audioContext, ngAudioData).Play()
		}
	}

	g.lowSpeedMoleculeCountInLeftSide = lowCount
	g.highSpeedMoleculeCountInRightSide = highCount
}

func (g *Game) Update() error {
	g.ticksFromModeStart++

	g.touchContext.Update()

	switch g.mode {
	case GameModeTitle:
		if g.touchContext.IsJustTouched() {
			g.mode = GameModeLevelStart
			g.ticksFromModeStart = 0
			g.playingSec = 0

			audio.NewPlayerFromBytes(audioContext, startAudioData).Play()

			logging.LogAsync(gameName, map[string]interface{}{
				"play_id": g.playID,
				"action":  "start_game",
			})
		}
	case GameModeLevelStart:
		if g.ticksFromModeStart > 2*uint64(ebiten.CurrentTPS()) {
			g.mode = GameModePlaying
			g.ticksFromModeStart = 0
			g.playStartTime = time.Now()
			g.playingSec = 0
		}
	case GameModePlaying:
		if g.ticksFromModeStart%600 == 0 {
			logging.LogAsync(gameName, map[string]interface{}{
				"play_id": g.playID,
				"action":  "playing",
				"level":   g.level,
				"ticks":   g.ticksFromModeStart,
				"sec":     time.Now().Unix() - g.playStartTime.Unix(),
				"low":     g.lowSpeedMoleculeCountInLeftSide,
				"high":    g.highSpeedMoleculeCountInRightSide,
			})
		}

		g.playingSec = int(time.Now().Unix() - g.playStartTime.Unix())

		if g.touchContext.IsJustTouched() {
			g.isPartitionSlitOpen = true

			audio.NewPlayerFromBytes(audioContext, openAudioData).Play()
		}
		if g.touchContext.IsJustReleased() {
			g.isPartitionSlitOpen = false
		}

		// Molecules move
		for i := 0; i < len(g.molecules); i++ {
			g.molecules[i].move()
		}

		// Partition slit move
		if !g.isPartitionSlitOpen {
			g.partitionSlitTop += g.partitionSlitVec
			g.partitionSlitBottom += g.partitionSlitVec

			if g.partitionSlitTop < frameTop {
				g.partitionSlitTop = frameTop
				g.partitionSlitBottom = g.partitionSlitTop + partitionSlitSize
				g.partitionSlitVec *= -1
			}
			if g.partitionSlitBottom > frameBottom {
				g.partitionSlitBottom = frameBottom
				g.partitionSlitTop = g.partitionSlitBottom - partitionSlitSize
				g.partitionSlitVec *= -1
			}
		}

		// Molecules and frame collision
		for i := 0; i < len(g.molecules); i++ {
			g.molecules[i].reboundIfCollide(g)
		}

		g.updateMoleculeCountInCorrectArea(true)

		// Complete
		if !g.isPartitionSlitOpen &&
			g.highSpeedMoleculeCountInRightSide == g.highSpeedMoleculeCount &&
			g.lowSpeedMoleculeCountInLeftSide == g.lowSpeedMoleculeCount {
			g.mode = GameModeComplete
			g.ticksFromModeStart = 0

			audio.NewPlayerFromBytes(audioContext, completeAudioData).Play()

			logging.LogAsync(gameName, map[string]interface{}{
				"play_id": g.playID,
				"action":  "complete",
				"level":   g.level,
				"sec":     g.playingSec,
			})
		}
	case GameModeComplete:
		for i := 0; i < len(g.molecules); i++ {
			g.molecules[i].move()
			g.molecules[i].reboundIfCollide(g)
		}

		if g.touchContext.IsJustTouched() {
			g.level++
			g.setUpField()
			g.mode = GameModeLevelStart
			g.ticksFromModeStart = 0
			g.playingSec = 0

			audio.NewPlayerFromBytes(audioContext, startAudioData).Play()
		}
	}

	return nil
}

var demonPattern = dotutil.CreatePatternImage([][]int{
	{1, 0, 0, 0, 1},
	{1, 1, 1, 1, 1},
	{1, 0, 1, 0, 1},
	{1, 1, 1, 1, 1},
	{1, 0, 1, 0, 1},
}, &dotutil.CreatePatternImageOption{
	Color: color.RGBA{0xfa, 0x64, 0x81, 0xff},
})

func (g *Game) Draw(screen *ebiten.Image) {
	// Background
	backgroundColor := color.RGBA{0xf3, 0xf3, 0xf3, 0xff}
	screen.Fill(backgroundColor)

	// Outer frame
	drawFrameOpt := &dotutil.DrawLineOption{
		DotSize:     frameThickness,
		Interval:    2,
		DotPosition: dotutil.LineDotPositionRightSide,
		Color:       color.RGBA{0xe3, 0xa3, 0x66, 0xff},
	}
	dotutil.DrawLine(screen, frameLeft, frameTop, frameLeft, frameBottom, drawFrameOpt)
	dotutil.DrawLine(screen, frameLeft, frameBottom, frameRight, frameBottom, drawFrameOpt)
	dotutil.DrawLine(screen, frameRight, frameBottom, frameRight, frameTop, drawFrameOpt)
	dotutil.DrawLine(screen, frameRight, frameTop, frameLeft, frameTop, drawFrameOpt)

	// Center partition
	drawPartitionOpt := &dotutil.DrawLineOption{
		DotSize:  frameThickness,
		Interval: 2,
		Color:    color.RGBA{0xe3, 0xa3, 0x66, 0xff},
	}
	dotutil.DrawLine(screen, screenWidth/2, frameTop, screenWidth/2, g.partitionSlitTop, drawPartitionOpt)
	dotutil.DrawLine(screen, screenWidth/2, frameBottom, screenWidth/2, g.partitionSlitBottom, drawPartitionOpt)

	// Demon
	demonY := g.partitionSlitTop + partitionSlitSize/2
	if g.isPartitionSlitOpen {
		demonY -= partitionSlitSize
	}
	w, _ := demonPattern.Size()
	dotutil.DrawImage(screen, demonPattern, screenWidth/2, demonY, &dotutil.DrawImageOption{
		Scale:        float64(partitionSlitSize-10) / float64(w),
		BasePosition: dotutil.DrawImagePositionCenter,
	})

	drawMolecules := func() {
		for i := 0; i < len(g.molecules); i++ {
			g.molecules[i].draw(screen)
		}
	}

	drawTextCenter := func(t string, y int, f *resourceutil.Font) {
		fontSize := f.FaceOptions.Size
		x := (screenWidth - len(t)*int(fontSize)) / 2
		ebitenutil.DrawRect(screen, float64(x), float64(y)-fontSize-10, float64(len(t))*fontSize, fontSize+10, backgroundColor)
		text.Draw(screen, t, f.Face, x, y, color.Black)
	}

	drawMolecureCount := func() {
		y := frameTop + 20
		lowCountText := fmt.Sprintf("%d/%d", g.lowSpeedMoleculeCountInLeftSide, g.lowSpeedMoleculeCount)
		highCountText := fmt.Sprintf("%d/%d", g.highSpeedMoleculeCountInRightSide, g.highSpeedMoleculeCount)
		lowTextX := (screenWidth/2+frameLeft-frameThickness/2)/2 - float64(len(lowCountText))*smallFont.FaceOptions.Size/2
		highTextX := (frameRight+screenWidth/2+frameThickness/2)/2 - float64(len(highCountText))*smallFont.FaceOptions.Size/2
		text.Draw(screen, lowCountText, smallFont.Face, int(lowTextX), y, color.Black)
		text.Draw(screen, highCountText, smallFont.Face, int(highTextX), y, color.Black)

		r := 7.0
		(&Molecule{speedType: MoleculeSpeedLow, x: lowTextX - 15, y: float64(y) - smallFont.FaceOptions.Size/2, r: r}).draw(screen)
		(&Molecule{speedType: MoleculeSpeedHigh, x: highTextX - 15, y: float64(y) - smallFont.FaceOptions.Size/2, r: r}).draw(screen)
	}

	drawLevel := func() {
		levelText := fmt.Sprintf("Lv.%d", g.level)
		text.Draw(screen, levelText, smallFont.Face, frameLeft, 20+int(smallFont.FaceOptions.Size), color.Black)
	}

	drawPlayingTime := func() {
		timeText := strconv.Itoa(g.playingSec)
		text.Draw(screen, timeText, smallFont.Face, frameRight-len(timeText)*int(smallFont.FaceOptions.Size), 20+int(smallFont.FaceOptions.Size), color.Black)
	}

	switch g.mode {
	case GameModeTitle:
		(&Molecule{speedType: MoleculeSpeedLow, x: 70, y: 300, r: moleculeRadius}).draw(screen)
		(&Molecule{speedType: MoleculeSpeedLow, x: 100, y: 100, r: moleculeRadius}).draw(screen)
		(&Molecule{speedType: MoleculeSpeedHigh, x: 200, y: 350, r: moleculeRadius}).draw(screen)
		(&Molecule{speedType: MoleculeSpeedHigh, x: 500, y: 200, r: moleculeRadius}).draw(screen)
		(&Molecule{speedType: MoleculeSpeedLow, x: 550, y: 400, r: moleculeRadius}).draw(screen)
		(&Molecule{speedType: MoleculeSpeedHigh, x: 600, y: 90, r: moleculeRadius}).draw(screen)

		drawTextCenter("MAXWELL'S DEMON", 170, largeFont)
		drawTextCenter("CLICK TO START", 290, mediumFont)
		licenseTexts := []string{"CREATOR: NAOKI TSUJIO", "FONT: Press Start 2P by CodeMan38", "SOUND: MaouDamashii"}
		for i, s := range licenseTexts {
			drawTextCenter(s, 400+i*int(smallFont.FaceOptions.Size*1.8), smallFont)
		}
	case GameModeLevelStart:
		drawMolecureCount()
		drawMolecules()
		drawTextCenter(fmt.Sprintf("LEVEL%d", g.level), 170, largeFont)
		drawLevel()
		drawPlayingTime()
	case GameModePlaying:
		drawMolecureCount()
		drawMolecules()
		drawLevel()
		drawPlayingTime()
	case GameModeComplete:
		drawMolecureCount()
		drawMolecules()
		drawLevel()
		drawPlayingTime()
		drawTextCenter("COMPLETED", 170, largeFont)
		drawTextCenter("YOUR RECORD IS", 275, mediumFont)
		drawTextCenter(fmt.Sprintf("%d SECONDS!", g.playingSec), 275+int(mediumFont.FaceOptions.Size*2), mediumFont)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return screenWidth, screenHeight
}

func (g *Game) setUpField() {
	logging.LogAsync(gameName, map[string]interface{}{
		"play_id": g.playID,
		"action":  "setup",
		"level":   g.level,
	})

	moleculeCount := g.level * 5

	g.lowSpeedMoleculeCount = 0
	g.highSpeedMoleculeCount = 0

	// Molecules
	molecules := []Molecule{}
	for i := 0; i < moleculeCount; i++ {
		var speedType MoleculeSpeedType
		var s float64
		if i%2 == 0 {
			speedType = MoleculeSpeedHigh
			s = moleculeSpeedHighValue
			g.highSpeedMoleculeCount++
		} else {
			speedType = MoleculeSpeedLow
			s = moleculeSpeedLowValue
			g.lowSpeedMoleculeCount++
		}
		var x float64
		if i/2%2 == 0 {
			x = float64(frameLeft + moleculeRadius/2 + rand.Int()%(screenWidth/2-frameThickness/2-moleculeRadius-frameLeft))
		} else {
			x = float64(frameRight - moleculeRadius/2 - rand.Int()%(frameRight-screenWidth/2-frameThickness/2-moleculeRadius))
		}
		y := float64(frameTop + moleculeRadius/2 + rand.Int()%(frameBottom-frameTop-moleculeRadius))
		angle := math.Pi * 2 * float64(rand.Int()%360) / 360
		if math.Pow(math.Pi/2-angle, 2) < math.Pow(math.Pi/6, 2) {
			angle += math.Pi / 3
		}
		if math.Pow(3*math.Pi/2-angle, 2) < math.Pow(math.Pi/6, 2) {
			angle += math.Pi / 3
		}
		m := Molecule{
			speedType: speedType,
			x:         x,
			y:         y,
			vx:        s * math.Cos(angle),
			vy:        s * math.Sin(angle),
			r:         moleculeRadius,
		}
		molecules = append(molecules, m)
	}
	g.molecules = molecules

	g.updateMoleculeCountInCorrectArea(false)

	// Partition slit
	g.partitionSlitTop = (frameBottom - partitionSlitSize + frameTop) * 0.4
	g.partitionSlitBottom = g.partitionSlitTop + partitionSlitSize
	g.partitionSlitVec = partitionSlitSpeed
	g.isPartitionSlitOpen = false
}

func (g *Game) initialize() {
	g.mode = GameModeTitle
	g.ticksFromModeStart = 0
	g.touchContext = touchutil.CreateTouchContext()
	g.level = 1

	g.setUpField()
}

func main() {
	if os.Getenv("GAME_LOGGING") != "1" {
		logging.Disable()
	}
	if seed, err := strconv.Atoi(os.Getenv("GAME_RAND_SEED")); err == nil {
		rand.Seed(int64(seed))
	} else {
		rand.Seed(time.Now().Unix())
	}

	ebiten.SetWindowSize(screenWidth, screenHeight)
	ebiten.SetWindowTitle("Maxwell's Demon")

	playIDObj, err := uuid.NewRandom()
	var playID string
	if err != nil {
		playID = "?"
	} else {
		playID = playIDObj.String()

	}

	game := &Game{playID: playID}
	game.initialize()

	if err := ebiten.RunGame(game); err != nil {
		log.Fatal(err)
	}
}
