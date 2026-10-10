# Dangerous Dave in the Haunted Mansion

[Русская версия](README.ru.md)

<img src="icon.png" alt="Dangerous Dave Go icon" width="128">

An unofficial native reconstruction of **Dangerous Dave in the Haunted Mansion** (Dangerous Dave II, 1991), written in Go. Explore the mansion, collect its treasure, defeat its monsters and bosses, and rescue Dave's brother Delbert.

The game uses the eight original maps, EGA artwork, character sprites, title screen, death portraits, and directional tile collision properties recovered from the supplied DOS copy. Movement, combat, enemy behavior, interface, sound effects, and saving are implemented in Go. It runs without MS-DOS or DOSBox; all required game data is embedded into the executable.

This is a playable recreation, not a recovered copy of the original source or an instruction-by-instruction port. Physics, enemy timing, boss behavior, synthesized sound effects, and the interface differ from the DOS release.

## Features

- Eight original levels, including the two boss chambers.
- Original 16-color EGA artwork at a logical resolution of 320 × 200.
- Scrolling levels, jumps, and one-way platforms with drop-through movement.
- Eight-direction shotgun aiming, eight-shell magazine, automatic reloading while standing still.
- Zombies, knife-throwing witches, ghosts, slimes, spiders, flames, wolves, falling traps, Frank, and the Mansion Master.
- Treasure, extra lives, closets, level exits, and linked teleport doors.
- Original illustrated death sequences.
- Pause, keyboard help, fullscreen, sound toggle, and manual save/load.
- A new desktop icon in PNG, ICO, and ICNS formats.
- Native builds for Windows, Linux, and macOS on x64 and ARM64.

## Screenshots

Actual captures from the Go version, enlarged 3× using nearest-neighbor scaling.

### Outside the mansion

![The mansion entrance](screenshots/entrance.png)

### Inside the mansion

![Exploring the mansion](screenshots/mansion.png)

### Frank's chamber

![The first boss](screenshots/frank.png)

## How to play

Move through the mansion, jump between floors, and fire at enemies. Standing still on a surface without firing reloads the shotgun one shell at a time. Up opens a nearby closet, enters a teleport door, or uses an exit. A closet can be looted only once per attempt.

The regular treasure values are 100, 200, 400, 800, and 1,600 points. A 1UP collectible awards a life, and every 10,000 points awards another life. You start with three lives. Death restarts the current level; at game over you can retry that level. Defeating Frank advances to level five; defeating the Mansion Master completes the game and rescues Delbert.

### Controls

| Action | Keys |
| --- | --- |
| Move | Left / Right or A / D |
| Aim upward / downward | Up / Down or W / S |
| Aim diagonally | Hold a horizontal and a vertical direction |
| Jump | Ctrl or Space |
| Drop through a one-way platform | Down + Ctrl / Space |
| Fire | Alt or X |
| Open / enter a door | Up, W, or E |
| Start a new game / confirm | Enter |
| Continue a saved game from the title | C |
| Pause / resume | Esc or P |
| Help | F1 |
| Sound on / off | F2 |
| Save | F5 |
| Load | F9 |
| Fullscreen | F11 |
| Restart current level | Pause, then R, then Enter |
| Return to title | Pause, then M |
| Quit | Esc on the title screen, or close the window |

The save file keeps the current position, map, enemies, treasure, doors, score, lives, and ammunition. Saving is available during active play, outside death and level-transition animations. There is one manual save slot.

Default location:

- Windows: `%AppData%\DangerousDaveGo\save.json`
- Linux: `$XDG_CONFIG_HOME/DangerousDaveGo/save.json`, or `~/.config/DangerousDaveGo/save.json`
- macOS: `~/Library/Application Support/DangerousDaveGo/save.json`

## Requirements

- **Go 1.25 or newer**.
- Internet access for the first dependency download; `go.mod` and `go.sum` pin the versions.
- A desktop session and a working graphics driver.
- Ebitengine 2.10.5 uses pure Go for these desktop targets: no C compiler, CGO, SDL SDK, or cross-compiler is required.
- Linux needs X11 or XWayland and OpenGL runtime libraries. Audio uses a PulseAudio-compatible service (including PipeWire's PulseAudio service), with ALSA as a fallback.

On Debian / Ubuntu, install runtime libraries if your desktop does not already provide them:

```sh
sudo apt install libx11-6 libgl1 libglx-mesa0 libxcursor1 libxi6 libxinerama1 libxrandr2 libxrender1 libxext6 libasound2t64
```

On Debian 12 / Ubuntu 22.04, use `libasound2` instead of `libasound2t64`. Keep an existing working graphics driver when using a proprietary driver.

## Build and run

Run commands from the project root (the directory containing `main.go`).

```sh
go mod download
go run .
```

A regular native build embeds the game data and window icon:

```sh
go build -trimpath -o dangerous-dave .
```

On Windows, use `dangerous-dave.exe` as the output filename. A regular `go build` does not by itself add an ICO resource to the Windows executable or create a macOS application bundle. Use the included build helper for those formats.

### Windows — executable with icon

```sh
go run ./tools/build -os windows -arch amd64
```

Output: `dist/dangerous-dave-windows-amd64.exe`.

For Windows on ARM:

```sh
go run ./tools/build -os windows -arch arm64
```

The helper reads `icon.ico`, creates a temporary architecture-specific `.syso` resource, invokes `go build` with `-H windowsgui`, and removes the temporary resource when finished. No resource compiler or additional Go tool is required. The icon is embedded in the EXE and visible in Explorer.

### Linux

```sh
go run ./tools/build -os linux -arch amd64
```

For ARM64:

```sh
go run ./tools/build -os linux -arch arm64
```

Outputs: `dist/dangerous-dave-linux-amd64` or `dist/dangerous-dave-linux-arm64`.

```sh
./dist/dangerous-dave-linux-amd64
```

The embedded PNG is used as the window icon. Linux launchers use a desktop entry and an external icon, rather than a Windows-style executable icon resource. `icon.png` can be used for such a launcher.

### macOS — application bundle with icon

Apple Silicon:

```sh
go run ./tools/build -os darwin -arch arm64 -app
```

Intel:

```sh
go run ./tools/build -os darwin -arch amd64 -app
```

Output: `dist/Dangerous Dave.app`. The helper copies `icon.icns`, writes `Info.plist`, and places the executable in `Contents/MacOS`. Open the app from Finder or run:

```sh
open "dist/Dangerous Dave.app"
```

Build an ordinary macOS executable by omitting `-app`:

```sh
go run ./tools/build -os darwin -arch arm64
```

### Cross-compilation

All commands above also work from another desktop OS. The helper sets `GOOS`, `GOARCH`, and `CGO_ENABLED=0` internally. You can build Linux ARM64 from Windows or x64 Linux without an ARM C compiler.

For an explicit output path:

```sh
go run ./tools/build -os windows -arch amd64 -out dist/dave.exe
```

The helper builds the selected architecture. For macOS, build the Intel and Apple Silicon editions separately; `-app` writes the same bundle path, so retain one output before building the other. Code signing and notarization are separate distribution steps.

## Command-line options

```sh
dangerous-dave --level 2 --fullscreen
dangerous-dave --mute --scale 4
dangerous-dave --save /path/to/save.json
```

| Option | Meaning |
| --- | --- |
| `--level 1..8` | Start a particular level directly; default opens the title |
| `--fullscreen` | Start fullscreen |
| `--scale 1..6` | Initial window scale; default 3 |
| `--mute` | Disable sound |
| `--save PATH` | Override the save slot location |
| `--version` | Print the version |
| `--capture PATH` | Capture the rendered game frame as PNG and exit |
| `--ticks N` | Frames before capture; default 1 |
| `--demo` | Scripted movement for screenshot capture |

Example screenshot command (requires a running desktop):

```sh
go run . --mute --level 1 --ticks 120 --capture entrance.png
```

## Project layout

```text
main.go                 Window, input, command-line options
render.go               Rendering, menus, HUD, screenshot capture
audio.go                Synthesized sound effects
internal/game/          Simulation, collisions, combat, saving, regression tests
assets/                 Embedded PNG artwork, maps, sprite layout, tile properties
tools/build/            Cross-platform build and icon resource helper
screenshots/            Screenshots used in the README
licenses/               Original-asset notice and dependency licenses
icon.png / .ico / .icns  Desktop icon
LICENSE                 License for newly written Go source
README.md / README.ru.md English / Russian documentation
RELEASE.md              Suggested English GitHub release description
```

The archive contains the source project and its assets. It does not contain the old DOS program, prebuilt game binaries, an emulator, or a vendor directory.

## Validation

```sh
go test ./...
go vet ./...
```

The simulation tests cover level loading and safe spawns, jumping and landing, wall collisions, dropping through platforms, shotgun hits and reloading, teleport doors, single-use closets, extra lives, saving/loading, and the final boss outcome.

During preparation, Linux x64 was launched and all eight levels were rendered; Windows, Linux, and macOS builds were checked for x64 and ARM64. Windows resource directories were inspected for the embedded icon. Windows and macOS binaries were cross-compiled; native execution on those systems has not been verified. Automated geometry exploration checked access to exits and boss firing positions; a complete manual playthrough has not been performed.

## Licenses and credits

- Newly written Go source: [MIT License](LICENSE).
- Original game artwork, maps, portraits, and other recovered data: [asset rights notice](licenses/ORIGINAL-ASSETS.md). These assets are **not licensed by this project's MIT license**. The supplied archive establishes their source, not permission for public redistribution.
- New icon and synthesized effects: see [asset rights notice](licenses/ORIGINAL-ASSETS.md).
- Ebitengine and other dependencies: their respective licenses in [licenses/](licenses/).

Original game: id Software / Softdisk, 1991. Dangerous Dave and its original characters and artwork belong to their respective rights holders. This project is unofficial and is not endorsed by them.

Format research acknowledgments:

- [Gil Megidish's Dangerous Dave reverse-engineering notes](https://github.com/gmegidish/dangerous-dave-re).
- [ModdingWiki: Dangerous Dave 2](https://moddingwiki.shikadi.net/wiki/Dangerous_Dave_2), including work credited to Andrew Durdin, Levellass, and the Keen modding community.
- [Ebitengine desktop installation documentation](https://ebitengine.org/en/documents/install.html).
