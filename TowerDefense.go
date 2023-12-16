package main

import (
	"embed"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/lafriks/go-tiled"
	"github.com/solarlune/paths"
	"image"
	"log"
	"math"
	"path"
	"strings"
)

//go:embed assets/*
var embeddedFiles embed.FS

type TowerDefenseGame struct {
	Level          *tiled.Map
	tileHash       map[uint32]*ebiten.Image
	pathFindingMap []string
	playertower    playertower
	enemy          enemy
	pathMap        *paths.Grid
	path           *paths.Path
	frameCounter   int
}

type playertower struct {
	spritesheet *ebiten.Image
	frame       int
	row         int
	column      int
}

type enemy struct {
	pict *ebiten.Image
	xloc float64
	yloc float64
}

func (game *TowerDefenseGame) Update() error {
	checkMouse(game)
	if game.path != nil {
		pathCell := game.path.Current()
		if math.Abs(float64(pathCell.X*game.Level.TileWidth)-(game.enemy.xloc)) <= 2 &&
			math.Abs(float64(pathCell.Y*game.Level.TileHeight)-(game.enemy.yloc)) <= 2 {
			game.path.Advance()
		}
		direction := 0.0
		if pathCell.X*game.Level.TileWidth > int(game.enemy.xloc) {
			direction = 1.0
		} else if pathCell.X*game.Level.TileWidth < int(game.enemy.xloc) {
			direction = -1.0
		}
		Ydirection := 0.0
		if pathCell.Y*game.Level.TileHeight > int(game.enemy.yloc) {
			Ydirection = 1.0
		} else if pathCell.Y*game.Level.TileHeight < int(game.enemy.yloc) {
			Ydirection = -1.0
		}
		game.enemy.xloc += direction * 2
		game.enemy.yloc += Ydirection * 2
	}

	// Update the player tower animation frame every 100 frames
	if game.frameCounter%100 == 0 {
		game.playertower.frame = (game.playertower.frame + 1) % 3
	}
	game.frameCounter++

	return nil
}

func checkMouse(game *TowerDefenseGame) {
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		mouseX, mouseY := ebiten.CursorPosition()
		game.enemy.xloc = float64(mouseX)
		game.enemy.yloc = float64(mouseY)
		startRow := int(game.enemy.yloc) / game.Level.TileHeight
		startCol := int(game.enemy.xloc) / game.Level.TileWidth
		startCell := game.pathMap.Get(startCol, startRow)
		endCell := game.pathMap.Get(game.playertower.column, game.playertower.row)
		game.path = game.pathMap.GetPathFromCells(startCell, endCell, false, false)
	}
}

func (game *TowerDefenseGame) Draw(screen *ebiten.Image) {
	// Draw the map
	for tileY := 0; tileY < game.Level.Height; tileY += 1 {
		for tileX := 0; tileX < game.Level.Width; tileX += 1 {
			drawOptions := ebiten.DrawImageOptions{}
			drawOptions.GeoM.Reset()
			TileXpos := float64(game.Level.TileWidth * tileX)
			TileYpos := float64(game.Level.TileHeight * tileY)
			drawOptions.GeoM.Translate(TileXpos, TileYpos)
			tileToDraw := game.Level.Layers[0].Tiles[tileY*game.Level.Width+tileX]
			ebitenTileToDraw := game.tileHash[tileToDraw.ID]
			screen.DrawImage(ebitenTileToDraw, &drawOptions)
		}
	}

	// Draw the player tower using the sprite sheet
	spriteWidth := game.playertower.spritesheet.Bounds().Dx() / 3
	spriteHeight := game.playertower.spritesheet.Bounds().Dy()
	frameWidth := spriteWidth
	frameHeight := spriteHeight
	frameX := frameWidth * game.playertower.frame
	frameY := 0

	drawOptions := ebiten.DrawImageOptions{}
	drawOptions.GeoM.Reset()
	drawOptions.GeoM.Translate(float64(game.playertower.column*game.Level.TileWidth), float64(game.playertower.row*game.Level.TileHeight))
	srcRect := image.Rect(frameX, frameY, frameX+frameWidth, frameY+frameHeight)
	screen.DrawImage(game.playertower.spritesheet.SubImage(srcRect).(*ebiten.Image), &drawOptions)

	// Draw the enemy
	drawOptions.GeoM.Reset()
	drawOptions.GeoM.Translate(game.enemy.xloc, game.enemy.yloc)
	screen.DrawImage(game.enemy.pict, &drawOptions)
}

func (game *TowerDefenseGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

func main() {
	gameMap := loadMapFromEmbedded(path.Join("assets", "MapForPaths.tmx"))
	pathMap := makeSearchMap(gameMap)
	searchablePathMap := paths.NewGridFromStringArrays(pathMap, gameMap.TileWidth, gameMap.TileHeight)
	searchablePathMap.SetWalkable('2', false)
	searchablePathMap.SetWalkable('3', false)
	ebiten.SetWindowSize(gameMap.TileWidth*gameMap.Width, gameMap.TileHeight*gameMap.Height)
	ebiten.SetWindowTitle("Maps Embedded")
	ebitenImageMap := makeEbitenImagesFromMap(*gameMap)
	playerTowerSpriteSheet := LoadEmbeddedImage("playertower", "playertower.png")

	// Get the top center tile coordinates
	topCenterTileX := gameMap.Width / 2
	topCenterTileY := 0

	oneLevelGame := TowerDefenseGame{
		Level:          gameMap,
		tileHash:       ebitenImageMap,
		pathFindingMap: pathMap,
		playertower: playertower{
			spritesheet: playerTowerSpriteSheet,
			frame:       0,
			row:         topCenterTileY, // Set to top center tile Y coordinate
			column:      topCenterTileX, // Set to top center tile X coordinate
		},
		enemy: enemy{
			pict: LoadEmbeddedImage("enemies/goblins", "goblin.png"),
			xloc: -100, // Put the NPC off-screen initially
			yloc: -100,
		},
		pathMap: searchablePathMap,
	}
	err := ebiten.RunGame(&oneLevelGame)
	if err != nil {
		fmt.Println("Couldn't run the game:", err)
	}
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
			log.Fatal("Failed to load embedded image ", embeddedFile, err)
		}
		ebitenImageTile, _, err := ebitenutil.NewImageFromReader(embeddedFile)
		if err != nil {
			fmt.Println("Error loading tile image:", tile.Image.Source, err)
		}
		idToImage[tile.ID] = ebitenImageTile
	}
	return idToImage
}

func LoadEmbeddedImage(folderName string, imageName string) *ebiten.Image {
	embeddedFile, err := embeddedFiles.Open(path.Join("assets", folderName, imageName))
	if err != nil {
		log.Fatal("Failed to load embedded image ", imageName, err)
	}
	ebitenImage, _, err := ebitenutil.NewImageFromReader(embeddedFile)
	if err != nil {
		fmt.Println("Error loading tile image:", imageName, err)
	}
	return ebitenImage
}

func loadMapFromEmbedded(name string) *tiled.Map {
	embeddedMap, err := tiled.LoadFile(name, tiled.WithFileSystem(embeddedFiles))
	if err != nil {
		fmt.Println("Error loading embedded map:", err)
	}
	return embeddedMap
}
