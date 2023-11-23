package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"image"
	"image/color"
	"io/fs"
	"log"
	"math"
	"path"
	"strings"
	"time"
)

//go:embed assets/*
var embeddedFiles embed.FS

type TowerType int

const (
	None TowerType = iota - 1
	ArcherTower
	MagicTower
	StoneTower
	IceTower
	PlayerTower
)

type Tower struct {
	Type     TowerType
	Level    int
	Range    float64
	Damage   float64
	Health   int
	Cost     int
	Freeze   float64
	Position image.Point
	Images   [3]*ebiten.Image
}

type Player struct {
	Health   int
	Currency int
}

type TowerDefenseGame struct {
	Level               *tiled.Map
	tileHash            map[uint32]*ebiten.Image
	TowerDefenseMap     []string
	Player              Player
	Towers              []Tower
	enemies             []Enemy
	pathMap             *paths.Grid
	path                *paths.Path
	archerTowerImages   [3]*ebiten.Image
	magicTowerImages    [3]*ebiten.Image
	iceTowerImages      [3]*ebiten.Image
	stoneTowerImages    [3]*ebiten.Image
	playerTowerImages   [3]*ebiten.Image
	SelectedTowerType   TowerType
	UpgradeAvailable    bool
	TowerManagementOpen bool
	TowerToPlace        TowerType
	lastUpdateTime      time.Time
	currentWave         int
	waveEnemies         []Enemy
	waveEnemyIndex      int
	waveInProgress      bool
	lastEnemySpawnTime  float64
	waveEndTime         float64
}

type EnemyType int

const (
	Goblin EnemyType = iota
	Knight
	Wizard
	Berserker
)

type Enemy struct {
	Type        EnemyType
	pict        *ebiten.Image
	xloc        float64
	yloc        float64
	Strength    int
	Currency    int
	Health      int
	attackTimer float64
}

func (game *TowerDefenseGame) Update() error {
	currentTime := time.Now()
	deltaTime := currentTime.Sub(game.lastUpdateTime).Seconds()
	game.lastUpdateTime = currentTime

	game.checkMouse()

	// Update enemy position
	for i := range game.enemies {
		fmt.Println("Updating enemy position:", game.enemies[i].xloc, game.enemies[i].yloc)
		if game.enemies[i].yloc > float64(60) {
			game.enemies[i].yloc -= .5
		} else {
			game.EnemyAttack(deltaTime, &game.enemies[i])
		}
	}

	if game.waveInProgress {
		fmt.Println("Wave in progress, enemy index:", game.waveEnemyIndex, "of", len(game.waveEnemies))
		if game.waveEnemyIndex < len(game.waveEnemies) {
			fmt.Println("Wave in progress:", game.waveInProgress)

			// Check if it's time to spawn the next enemy
			if game.lastEnemySpawnTime >= 4 { // 1 second interval
				game.enemies = append(game.enemies, game.waveEnemies[game.waveEnemyIndex])
				game.waveEnemyIndex++
				game.lastEnemySpawnTime = 0 // Reset the spawn timer
			} else {
				game.lastEnemySpawnTime += deltaTime
			}
		} else {
			// All enemies in the wave have been spawned
			if game.waveEndTime >= 15 {
				// Start the next wave
				game.startWave(game.currentWave + 1)
				game.waveEndTime = 0
			} else {
				game.waveEndTime += deltaTime
			}
		}

		for i := range game.Towers {
			tower := &game.Towers[i]
			for j := 0; j < len(game.enemies); {
				enemy := &game.enemies[j]
				if tower.isEnemyInRange(enemy) {
					enemy.Health -= int(tower.Damage)
					if enemy.Health <= 0 {
						// Update player's currency
						game.Player.Currency += enemy.Currency

						// Remove enemy from slice
						game.enemies = append(game.enemies[:j], game.enemies[j+1:]...)

					} else {

						j++
					}
				} else {

					j++
				}
			}
		}
	}

	return nil
}

func (game *TowerDefenseGame) checkMouse() {

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		tileX, tileY := mouseX/game.Level.TileWidth, mouseY/game.Level.TileHeight

		// Place a tower only if a tower is selected, the tower/upgrade window is closed, and the tile is valid
		if !game.TowerManagementOpen && game.TowerToPlace != None && game.isTileValidForTower(tileX, tileY) {
			game.placeTower(tileX, tileY)
			game.TowerToPlace = -1 // Reset the selected tower after placing it
		}
	}
}

func (game *TowerDefenseGame) isTileValidForTower(x, y int) bool {

	return true // Placeholder implementation
}

func (game *TowerDefenseGame) selectTower(towerType TowerType) {
	if game.TowerToPlace == towerType {
		game.TowerToPlace = -1 // Deselect if the same tower type is clicked again
	} else {
		game.TowerToPlace = towerType
	}
	game.TowerManagementOpen = false // Close the window after selection
}

func (game *TowerDefenseGame) placeTower(x, y int) {
	if game.TowerToPlace == -1 {
		return // No tower selected, so don't place anything
	}

	// Calculate the center position of the tile
	centerX := (x * game.Level.TileWidth) + (game.Level.TileWidth / 2) - (64 / 2)
	centerY := (y * game.Level.TileHeight) + (game.Level.TileHeight / 2) - (192 / 2)

	if game.TowerToPlace == StoneTower || game.TowerToPlace == MagicTower {
		centerY -= 64 // Adjust centerY for Stone and Magic towers
	}

	var newTower Tower
	switch game.TowerToPlace {
	case ArcherTower:
		if game.Player.Currency < 20 {
			fmt.Println("Not enough currency to place Archer Tower")
			return
		}
		newTower = Tower{
			Type:     ArcherTower,
			Level:    1,
			Range:    100,
			Damage:   10,
			Position: image.Point{X: centerX, Y: centerY},
			Images:   game.archerTowerImages,
			Cost:     20,
		}
		game.Player.Currency -= 20

	case MagicTower:
		if game.Player.Currency < 40 {
			fmt.Println("Not enough currency to place Lightning Tower")
			return
		}
		newTower = Tower{
			Type:     MagicTower,
			Level:    1,
			Range:    150,
			Damage:   15,
			Position: image.Point{X: centerX, Y: centerY},
			Images:   game.magicTowerImages,
			Cost:     40,
		}
		game.Player.Currency -= 40

	case IceTower:
		if game.Player.Currency < 60 {
			fmt.Println("Not enough currency to place Ice Tower")
			return
		}
		newTower = Tower{
			Type:     IceTower,
			Level:    1,
			Range:    120,
			Damage:   10,
			Position: image.Point{X: centerX, Y: centerY},
			Images:   game.iceTowerImages,
			Cost:     60,
			Freeze:   5,
		}
		game.Player.Currency -= 60

	case StoneTower:
		if game.Player.Currency < 100 {
			fmt.Println("Not enough currency to place Stone Tower")
			return
		}
		newTower = Tower{
			Type:     StoneTower,
			Level:    1,
			Range:    80,
			Damage:   25,
			Position: image.Point{X: centerX, Y: centerY},
			Images:   game.stoneTowerImages,
			Cost:     100,
		}
		game.Player.Currency -= 100

	default:
		fmt.Println("Unknown tower type")
		return
	}

	game.Towers = append(game.Towers, newTower)

}

func (t *Tower) isEnemyInRange(enemy *Enemy) bool {
	distance := math.Sqrt(math.Pow(enemy.xloc-float64(t.Position.X), 2) + math.Pow(enemy.yloc-float64(t.Position.Y), 2))
	return distance <= t.Range
}

func (game *TowerDefenseGame) drawTowerManagementButton(screen *ebiten.Image) {

	buttonX, buttonY, buttonWidth, buttonHeight := 10, 10, 64, 64

	// Draw button background
	ebitenutil.DrawRect(screen, float64(buttonX), float64(buttonY), float64(buttonWidth), float64(buttonHeight), color.Black)

	// Check if the mouse button is just pressed
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		if mouseX >= buttonX && mouseX <= buttonX+buttonWidth && mouseY >= buttonY && mouseY <= buttonY+buttonHeight {
			game.TowerManagementOpen = !game.TowerManagementOpen // Toggle the window
		}
	}

	// Draw the tower management window if open
	if game.TowerManagementOpen {
		game.drawTowerManagementWindow(screen)
	}
}

const (
	buttonSize = 40
)

func (game *TowerDefenseGame) drawTowerManagementWindow(screen *ebiten.Image) {
	// Define the window size and position
	screenWidth, screenHeight := screen.Size()
	windowWidth, windowHeight := 500, 250 // Adjust size as needed
	windowX, windowY := (screenWidth-windowWidth)/2, (screenHeight-windowHeight)/2

	// Draw the window background
	ebitenutil.DrawRect(screen, float64(windowX), float64(windowY), float64(windowWidth), float64(windowHeight), color.RGBA{R: 50, G: 50, B: 50, A: 200})

	// Draw a title for the window
	title := "Tower Management"
	titleX := windowX + (windowWidth-len(title)*8)/2 // Center the title
	titleY := windowY + 10
	ebitenutil.DebugPrintAt(screen, title, titleX, titleY)

	// Draw icons for each tower type
	towerTypes := []TowerType{ArcherTower, MagicTower, StoneTower, IceTower}
	for i, towerType := range towerTypes {
		x := windowX + 64 + i*(buttonSize+64) // Adjust position as needed
		y := windowY + 64                     // Adjust position as needed

		if towerType == StoneTower || towerType == MagicTower {
			y -= 64
		}

		// Draw tower image
		var towerImage *ebiten.Image
		switch towerType {
		case ArcherTower:
			towerImage = game.archerTowerImages[0]
		case MagicTower:
			towerImage = game.magicTowerImages[0]
		case IceTower:
			towerImage = game.iceTowerImages[0]
		case StoneTower:
			towerImage = game.stoneTowerImages[0]
		}

		if towerImage != nil {
			opts := &ebiten.DrawImageOptions{}
			opts.GeoM.Translate(float64(x), float64(y))
			screen.DrawImage(towerImage, opts)
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		for i, towerType := range towerTypes {
			x := windowX + 64 + i*(buttonSize+64)
			y := windowY + 128
			if mouseX >= x && mouseX <= x+buttonSize && mouseY >= y && mouseY <= y+buttonSize {
				game.selectTower(towerType)
				break
			}
		}
	}
}

func (game *TowerDefenseGame) Draw(screen *ebiten.Image) {
	drawOptions := ebiten.DrawImageOptions{}
	// Draw the map
	for y := 0; y < game.Level.Height; y++ {
		for x := 0; x < game.Level.Width; x++ {
			tile := game.Level.Layers[0].Tiles[y*game.Level.Width+x]
			if tileImage, ok := game.tileHash[tile.ID]; ok {
				opts := &ebiten.DrawImageOptions{}
				opts.GeoM.Translate(float64(x*game.Level.TileWidth), float64(y*game.Level.TileHeight))
				screen.DrawImage(tileImage, opts)
			}
		}
	}

	game.drawHeader(screen)
	game.drawTowerManagementButton(screen)

	for _, enemy := range game.enemies {
		if enemy.pict != nil {
			drawOptions.GeoM.Reset()
			drawOptions.GeoM.Translate(enemy.xloc, enemy.yloc)
			screen.DrawImage(enemy.pict, &drawOptions)
		} else {
			fmt.Println("Enemy picture is nil for enemy type:", enemy.Type)
		}
	}

	if game.TowerManagementOpen {
		game.drawTowerManagementWindow(screen)
	}

	// Draw the towers
	for _, tower := range game.Towers {
		opts := &ebiten.DrawImageOptions{}
		opts.GeoM.Translate(float64(tower.Position.X), float64(tower.Position.Y))
		screen.DrawImage(tower.Images[tower.Level-1], opts)
	}

}

func (game *TowerDefenseGame) drawHeader(screen *ebiten.Image) {
	headerText := fmt.Sprintf("Health: %d Currency: %d", game.Player.Health, game.Player.Currency)

	// Load the font face for larger text
	tt, err := opentype.Parse(goregular.TTF)
	if err != nil {
		log.Fatal(err)
	}

	const dpi = 72
	face, err := opentype.NewFace(tt, &opentype.FaceOptions{
		Size:    24, // Adjust the size as needed
		DPI:     dpi,
		Hinting: font.HintingFull,
	})
	if err != nil {
		log.Fatal(err)
	}

	// Measure the size of the text
	bounds := text.BoundString(face, headerText)
	textWidth := bounds.Max.X - bounds.Min.X
	x := 400 - textWidth
	y := 40

	bg := image.Rect(x-10, y-24, x+textWidth+10, y+10)
	ebitenutil.DrawRect(screen, float64(bg.Min.X), float64(bg.Min.Y), float64(bg.Dx()), float64(bg.Dy()), color.RGBA{0, 0, 0, 128})

	text.Draw(screen, headerText, face, x, y, color.White)

	err = face.Close()
	if err != nil {
		return
	}
}

func (game TowerDefenseGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func makeEnemies(game *TowerDefenseGame, enemyType EnemyType) Enemy {
	var picture *ebiten.Image
	var err error
	var strength, currency, health int

	switch enemyType {
	case Goblin:
		picture, err = LoadEmbeddedImage("enemies/goblins", "goblin.png")
		strength = 5
		currency = 10
		health = 10
	case Knight:
		picture, err = LoadEmbeddedImage("enemies/knights", "knight.png")
		strength = 10
		currency = 20
		health = 20
	case Wizard:
		picture, err = LoadEmbeddedImage("enemies/wizards", "wizard.png")
		strength = 15
		currency = 30
		health = 30
	case Berserker:
		picture, err = LoadEmbeddedImage("enemies/berserkers", "berserker.png")
		strength = 20
		currency = 40
		health = 40

	}

	if err != nil {
		log.Fatalf("Failed to load enemy image: %v", err)
	}

	xloc := float64(game.Level.TileWidth*game.Level.Width)/2 - 32
	yloc := float64(game.Level.TileHeight*game.Level.Height) - float64(picture.Bounds().Dy())

	return Enemy{
		Type:        enemyType,
		pict:        picture,
		xloc:        xloc,
		yloc:        yloc,
		Strength:    strength,
		Currency:    currency,
		Health:      health,
		attackTimer: 3,
	}
}

// Check if the enemy has reached the player tower
func (game *TowerDefenseGame) isEnemyAtPlayerTower() bool {

	middleColumnX := game.Level.TileWidth * 7
	topRowY := 0 + 64

	// Calculate the horizontal range of the tile
	tileLeft := middleColumnX
	tileRight := middleColumnX + game.Level.TileWidth

	for _, enemy := range game.enemies {
		enemyWithinHorizontalRange := enemy.xloc >= float64(tileLeft) && enemy.xloc+float64(enemy.pict.Bounds().Dx()) <= float64(tileRight)
		enemyReachedTileY := enemy.yloc <= float64(topRowY)
		if enemyWithinHorizontalRange && enemyReachedTileY {
			return true
		}
	}
	return false
}

// Handle the enemy's attack on the player tower
func (game *TowerDefenseGame) EnemyAttack(deltaTime float64, enemy *Enemy) {

	attackInterval := 1.0

	enemy.attackTimer += deltaTime
	if enemy.attackTimer >= attackInterval {
		game.Player.Health -= enemy.Strength
		enemy.attackTimer = 0

		if game.Player.Health <= 0 {
			// Placeholder for future logic
		}

	}
}

func (game *TowerDefenseGame) startWave(waveNumber int) {
	fmt.Println("Starting wave:", waveNumber)
	game.currentWave = waveNumber
	game.waveInProgress = true
	game.waveEnemyIndex = 0
	game.waveEnemies = []Enemy{}

	// Define the enemies for each wave
	switch waveNumber {
	case 1:

		for i := 0; i < 10; i++ {
			game.waveEnemies = append(game.waveEnemies, makeEnemies(game, Goblin))
		}
	case 2:

		for i := 0; i < 8; i++ {
			game.waveEnemies = append(game.waveEnemies, makeEnemies(game, Knight))
		}
	case 3:

		for i := 0; i < 6; i++ {
			game.waveEnemies = append(game.waveEnemies, makeEnemies(game, Wizard))
		}
	case 4:

		for i := 0; i < 5; i++ {
			game.waveEnemies = append(game.waveEnemies, makeEnemies(game, Berserker))
		}

	}
}

func main() {
	fmt.Println("Main function started")
	gameMap := loadMapFromEmbedded(path.Join("assets", "MapForPaths.tmx"))
	pathMap := makeSearchMap(gameMap)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('2', false)
	searchablePathMap.SetWalkable('3', false)
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Tower Defense Game")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	oneLevelGame := TowerDefenseGame{
		Level:             gameMap,
		tileHash:          ebitenImageMap,
		TowerDefenseMap:   pathMap,
		SelectedTowerType: -1,
		TowerToPlace:      -1,
		UpgradeAvailable:  true,
		Player: Player{
			Health:   100,
			Currency: 50,
		},
		Towers:  []Tower{},
		pathMap: searchablePathMap,
	}

	loadTowerImages(&oneLevelGame)

	placePlayerTowers(&oneLevelGame)

	oneLevelGame.enemies = []Enemy{}

	oneLevelGame.startWave(1)

	// Run the game
	err := ebiten.RunGame(&oneLevelGame)
	if err != nil {
		log.Fatalf("Couldn't run game: %v", err)
	}
}

func loadTowerImages(game *TowerDefenseGame) {
	var err error

	// Load archer tower images
	game.archerTowerImages, err = loadTowerImageSet("archertower")
	if err != nil {
		log.Fatalf("Failed to load archer tower images: %v", err)
	}

	// Load ice tower images
	game.iceTowerImages, err = loadTowerImageSet("icetower")
	if err != nil {
		log.Fatalf("Failed to load ice tower images: %v", err)
	}

	// Load magic tower images
	game.magicTowerImages, err = loadTowerImageSet("magictower")
	if err != nil {
		log.Fatalf("Failed to load magic tower images: %v", err)
	}

	// Load stone tower images
	game.stoneTowerImages, err = loadTowerImageSet("stonetower")
	if err != nil {
		log.Fatalf("Failed to load stone tower images: %v", err)
	}

	// Load player tower images
	game.playerTowerImages, err = loadTowerImageSet("playertower")
	if err != nil {
		log.Fatalf("Failed to load player tower images: %v", err)
	}
}

func loadTowerImageSet(folderName string) ([3]*ebiten.Image, error) {
	var images [3]*ebiten.Image
	fullImage, err := LoadEmbeddedImage(folderName, folderName+".png")
	if err != nil {
		return images, err
	}

	for i := 0; i < 3; i++ {
		subImage := fullImage.SubImage(image.Rect(i*64, 0, (i+1)*64, 192)).(*ebiten.Image)
		images[i] = subImage
	}

	return images, nil
}

func placePlayerTowers(game *TowerDefenseGame) {
	
	centerX := (game.Level.TileWidth * game.Level.Width / 2) - (64 / 2)

	centerY := (game.Level.TileHeight / 2) - (192 / 2)

	game.Towers = append(game.Towers, Tower{
		Type:     PlayerTower,
		Level:    1,
		Health:   100, // Health for level 1
		Position: image.Point{X: centerX, Y: centerY},
		Images:   game.playerTowerImages,
	})
}

func loadMapFromEmbedded(name string) *tiled.Map {
	embeddedMap, err := tiled.LoadFile(name, tiled.WithFileSystem(embeddedFiles))
	if err != nil {
		log.Fatalf("Error loading embedded map: %v", err)
	}
	return embeddedMap
}

func makeSearchMap(tiledMap *tiled.Map) []string {
	mapAsStringSlice := make([]string, 0, tiledMap.Height)
	row := strings.Builder{}
	for position, tile := range tiledMap.Layers[0].Tiles {
		if position%tiledMap.Width == 0 && position > 0 {
			mapAsStringSlice = append(mapAsStringSlice, row.String())
			row = strings.Builder{}
		}
		row.WriteString(fmt.Sprintf("%d", tile.ID))
	}
	mapAsStringSlice = append(mapAsStringSlice, row.String())
	return mapAsStringSlice
}

func makeEbitenImagesFromMap(tiledMap tiled.Map) map[uint32]*ebiten.Image {
	idToImage := make(map[uint32]*ebiten.Image)
	for _, tile := range tiledMap.Tilesets[0].Tiles {
		embeddedFile, err := embeddedFiles.Open(path.Join("assets", tile.Image.Source))
		if err != nil {
			log.Fatal("failed to load embedded image ", embeddedFile, err)
		}
		ebitenImageTile, _, err := ebitenutil.NewImageFromReader(embeddedFile)
		if err != nil {
			fmt.Println("Error loading tile image:", tile.Image.Source, err)
		}
		idToImage[tile.ID] = ebitenImageTile
	}
	return idToImage
}

func LoadEmbeddedImage(folderName string, imageName string) (*ebiten.Image, error) {
	embeddedFile, err := embeddedFiles.Open(path.Join("assets", folderName, imageName))
	if err != nil {
		return nil, fmt.Errorf("failed to load embedded image %s: %w", imageName, err)
	}
	defer func(embeddedFile fs.File) {
		err := embeddedFile.Close()
		if err != nil {

		}
	}(embeddedFile)

	img, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create image from embedded file %s: %w", imageName, err)
	}

	return img, nil
}
