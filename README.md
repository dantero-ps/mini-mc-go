# Mini-MC

A simple Minecraft clone developed using Go, OpenGL, and GLFW.

## Prerequisites

- **Go:** [golang.org](https://golang.org/dl/) (v1.24+)
- **C Compiler:** Required for CGO (GCC or Clang).
- **OpenGL Drivers:** Up-to-date graphics drivers.

## Installation and Running

### Windows
1. Ensure [Mingw-w64](https://www.mingw-w64.org/) or a similar C compiler is installed.
2. Run the following command in the project root:
   ```bash
   go run ./cmd/mini-mc
   ```

### macOS
1. Xcode Command Line Tools must be installed (`xcode-select --install`).
2. Run the following command in the project root:
   ```bash
   go run ./cmd/mini-mc
   ```

### Linux (Ubuntu/Debian)
1. Install the required libraries:
   ```bash
   sudo apt-get update
   sudo apt-get install libgl1-mesa-dev xorg-dev
   ```
2. Run the following command in the project root:
   ```bash
   go run ./cmd/mini-mc
   ```
