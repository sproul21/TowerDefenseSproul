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
	"log"
	"path"
	"strings"
	"time"
)

//go:embed assets/*
var embeddedFiles embed.FS

type TowerType int

const (
	ArcherTower TowerType = iota
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
	Position image.Point
	Images   [3]*ebiten.Image // Images for each level
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
	enemies             Enemy
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
}

type Enemy struct {
	pict        *ebiten.Image
	xloc        float64
	yloc        float64
	Strength    int
	attackTimer float64
}

func (game *TowerDefenseGame) Update() error {
	currentTime := time.Now()
	deltaTime := currentTime.Sub(game.lastUpdateTime).Seconds() // deltaTime in seconds
	game.lastUpdateTime = currentTime

	game.checkMouse()

	// Update enemy position
	if game.enemies.yloc > float64(60) {
		game.enemies.yloc -= 1 // Adjust this value to control the speed of the enemy
	} else {
		game.EnemyAttack(deltaTime)
	}

	return nil
}

func (game *TowerDefenseGame) checkMouse() {

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		tileX, tileY := mouseX/game.Level.TileWidth, mouseY/game.Level.TileHeight

		// Place a tower only if a tower is selected, the tower/upgrade window is closed, and the tile is valid
		if !game.TowerManagementOpen && game.TowerToPlace != -1 && game.isTileValidForTower(tileX, tileY) {
			game.placeTower(tileX, tileY)
			game.TowerToPlace = -1 // Reset the selected tower after placing it
		}
	}
}

func (game *TowerDefenseGame) isTileValidForTower(x, y int) bool {
	// Implement logic to check if the tile at (x, y) is valid for placing a tower
	// For example, you might want to check if the tile is not a path tile
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
	centerX := (x * game.Level.TileWidth) + (game.Level.TileWidth / 2) - (64 / 2)    // Center X - half width of the tower
	centerY := (y * game.Level.TileHeight) + (game.Level.TileHeight / 2) - (192 / 2) // Center Y - half height of the tower

	if game.TowerToPlace == StoneTower || game.TowerToPlace == MagicTower {
		centerY -= 64 // Adjust centerY for Stone and Magic towers
	}

	var towerImages [3]*ebiten.Image
	switch game.TowerToPlace {
	case ArcherTower:
		towerImages = game.archerTowerImages
	case MagicTower:
		towerImages = game.magicTowerImages
	case IceTower:
		towerImages = game.iceTowerImages
	case StoneTower:
		towerImages = game.stoneTowerImages
	// Add cases for other tower types
	default:
		return // No tower selected or invalid type
	}

	newTower := Tower{
		Type:     game.TowerToPlace,
		Level:    1,
		Range:    100, // Example values, adjust as needed
		Damage:   10,
		Position: image.Point{X: centerX, Y: centerY},
		Images:   towerImages,
	}
	game.Towers = append(game.Towers, newTower)
}

func (game *TowerDefenseGame) drawTowerManagementButton(screen *ebiten.Image) {

	buttonX, buttonY, buttonWidth, buttonHeight := 10, 10, 64, 64 // Adjust as needed

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
	buttonSize = 40 // Size of the button
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

	// Draw buttons or icons for each tower type
	towerTypes := []TowerType{ArcherTower, MagicTower, StoneTower, IceTower}
	for i, towerType := range towerTypes {
		x := windowX + 64 + i*(buttonSize+64) // Adjust position as needed
		y := windowY + 64                     // Adjust position as needed

		if towerType == StoneTower || towerType == MagicTower {
			y -= 64
		}

		// Draw button background or tower image
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

	// Handle button clicks outside the drawing loop
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

	if game.enemies.pict != nil {
		drawOptions.GeoM.Reset()
		drawOptions.GeoM.Translate(game.enemies.xloc, game.enemies.yloc)
		screen.DrawImage(game.enemies.pict, &drawOptions)
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
	y := 40 // Adjust as needed

	// Draw a semi-transparent rectangle as the background
	bg := image.Rect(x-10, y-24, x+textWidth+10, y+10)
	ebitenutil.DrawRect(screen, float64(bg.Min.X), float64(bg.Min.Y), float64(bg.Dx()), float64(bg.Dy()), color.RGBA{0, 0, 0, 128})

	// Draw the text
	text.Draw(screen, headerText, face, x, y, color.White)

	// Release the font face
	err = face.Close()
	if err != nil {
		return
	}
}

func (game TowerDefenseGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func makeEnemies(game *TowerDefenseGame) Enemy {
	picture, err := LoadEmbeddedImage("", "goblin.png")
	if err != nil {
		log.Fatalf("Failed to load enemy image: %v", err)
	}
	xloc := float64(game.Level.TileWidth*game.Level.Width)/2 - 32
	yloc := float64(game.Level.TileHeight*game.Level.Height) - float64(picture.Bounds().Dy())
	character := Enemy{
		pict:        picture,
		xloc:        xloc,
		yloc:        yloc,
		Strength:    5,
		attackTimer: 3,
	}
	return character
}

// Check if the enemy has reached the player tower
func (game *TowerDefenseGame) isEnemyAtPlayerTower() bool {
	// Calculate the position of the tile in the first row and middle column
	middleColumnX := game.Level.TileWidth * 7 // 7th column in zero-based indexing
	topRowY := 0 + 64                         // Top row

	// Calculate the horizontal range of the tile
	tileLeft := middleColumnX
	tileRight := middleColumnX + game.Level.TileWidth

	// Check if the enemy is within the horizontal range of the tile
	enemyWithinHorizontalRange := game.enemies.xloc >= float64(tileLeft) && game.enemies.xloc+float64(game.enemies.pict.Bounds().Dx()) <= float64(tileRight)

	// Check if the enemy has reached the y position of the tile
	enemyReachedTileY := game.enemies.yloc <= float64(topRowY)

	return enemyWithinHorizontalRange && enemyReachedTileY
}

// Handle the enemy's attack on the player tower
func (game *TowerDefenseGame) EnemyAttack(deltaTime float64) {

	attackInterval := 1.0 // Example: 1 second between attacks

	// Update the attack timer
	game.enemies.attackTimer += deltaTime

	// Check if it's time to attack
	if game.enemies.attackTimer >= attackInterval {
		game.Player.Health -= game.enemies.Strength
		game.enemies.attackTimer = 0 // Reset the timer after an attack

		if game.Player.Health <= 0 {
			// Handle the destruction of the player tower
			// For example, end the game or reduce player's health
			// Implement your logic here
		}

	}
}

func main() {
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

	// Load all tower images including the player tower
	loadTowerImages(&oneLevelGame)

	// Place the player towers at the specified locations
	placePlayerTowers(&oneLevelGame)

	// Load enemies onto the map
	oneLevelGame.enemies = makeEnemies(&oneLevelGame)

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

	// Each tower level image is 64x192 pixels
	for i := 0; i < 3; i++ {
		subImage := fullImage.SubImage(image.Rect(i*64, 0, (i+1)*64, 192)).(*ebiten.Image)
		images[i] = subImage
	}

	return images, nil
}

func placePlayerTowers(game *TowerDefenseGame) {
	// Calculate centerX as the horizontal center of the map minus half the width of the tower
	centerX := (game.Level.TileWidth * game.Level.Width / 2) - (64 / 2)

	// centerY is set to the top of the map minus half the height of the tower
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
		fmt.Println("Error loading embedded map:", err)
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
	ebitenImage, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		return nil, fmt.Errorf("error loading tile image %s: %w", imageName, err)
	}
	return ebitenImage, nil
}
