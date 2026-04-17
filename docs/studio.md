# Pixelforge Studio - Game Development Guide

## What is Pixelforge Studio?

Pixelforge Studio is a visual editor for creating retro 2D games using the Pixelforge game engine. It provides a GUI (graphical user interface) where you can:
- Import and view sprite images
- Place sprites on a scene canvas
- Manage game objects
- Generate playable Go code

## Getting Started

### 1. Launch the Studio

```bash
go run ./pixelforge_studio
```

This opens a 1280x800 window with:
- **Top menu bar** - File, Edit, View, Project, Help menus
- **Left panel** - Shows your loaded sprites
- **Center panel** - Scene canvas where you place game objects
- **Right panel** - Properties (coming soon)
- **Bottom status bar**

### 2. Loading Sprites

The studio automatically scans the `pixelforge_examples` folder on startup and loads any PNG images it finds. These include:
- Sprites from the Snake game example
- Sprites from the Shapes example
- Other PNG files in example folders

### 3. Tools Overview

Use keyboard shortcuts or click tool buttons:

| Key | Tool | What it does |
|-----|------|-------------|
| V | Select | Click objects on canvas to select them, then drag to move |
| P | Place | Click on canvas to place the currently selected sprite |
| X | Delete | Click on canvas to delete an object |

### 4. Creating Your First Game Scene

1. **Select a sprite** from the left panel by clicking on it
2. **Switch to Place tool** (press P or click Place button)
3. **Click on the canvas** where you want the sprite to appear
4. **Switch to Select tool** (press V)
5. **Click and drag** to move placed objects around
6. **Press Delete** or use Delete tool to remove objects

### 5. Menu Options

**File Menu:**
- New Project - Reset the studio
- Save - Save project (not implemented yet)
- Export Game - Export to Go project
- Exit - Close the studio

**View Menu:**
- Zoom In / Zoom Out - Change canvas view scale
- Grid - Toggle grid visibility
- Collision - Toggle collision box visibility

**Project Menu:**
- Run Preview - Preview your game (coming soon)
- Settings - Configure game settings

## Building Your First Game

### Step 1: Design Your Scene

1. Think about what sprites you need (player, enemies, items, obstacles)
2. Import or ensure sprites are in the examples folder
3. Place sprites on the canvas using the Place tool
4. Arrange them to create your game scene

### Step 2: Add Game Logic

The exported game will include Update and Draw functions:

```go
func update() {
    // Handle input
    if pixelforge_key.Duration(pixelforge_key.Left) > 0 {
        // Move player left
    }
}

func draw() {
    pixelforge.Screen().Clear(0)
    // Draw your sprites
}
```

### Step 3: Export

From the File menu, select "Export Game". This generates:
- A Go project with `go.mod`
- `main.go` with your game code
- Sprite assets in `assets/sprites/`

### Step 4: Run Your Game

```bash
cd your-exported-game
go run .
```

## Example Projects

### Snake Clone

1. Load snake sprites from examples
2. Use Place tool to place food and snake segments
3. Add collision detection in Update function
4. Export and run!

### Basic Platformer

1. Load player/platform sprites
2. Place platforms on canvas
3. Add gravity and jumping in Update function
4. Add scrolling in Draw function

## Tips for Beginners

1. **Start Simple**: Begin with a game like Pong or Snake before attempting complex games
2. **Use Retro Resolution**: The default 320x180 gives authentic retro feel
3. **Organize Sprites**: Keep your sprite sheets organized (8x8, 16x16 frames)
4. **Test Often**: Export and run frequently to catch issues early
5. **Check Examples**: Look at the pixelforge_examples folder for code patterns

## Keyboard Shortcuts Summary

| Shortcut | Action |
|----------|--------|
| V | Select tool |
| P | Place tool |
| X | Delete tool |
| Delete | Delete selected object |
| Ctrl+O | Open project |
| Ctrl+S | Save project |

## Troubleshooting

**Q: No sprites showing in left panel**
A: Ensure PNG files are in the pixelforge_examples folder

**Q: Can't place sprites**
A: Make sure you have a sprite selected in the left panel, then use Place tool (P)

**Q: Objects not moving**
A: Use Select tool (V), click object, then drag

## Next Steps

- Read the main Pixelforge README for engine documentation
- Check example games in pixelforge_examples folder
- Experiment with different sprite sheets
- Add audio with pixelforge_audio package

Happy game building!