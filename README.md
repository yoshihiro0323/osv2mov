# osv2mov

A CLI tool for extracting various files from OSV files created by DJI OSMO 360.

## Features

- OSV is ISO-BMFF(MP4/MOV) compatible, containing two HEVC fisheye videos, AAC audio, thumbnails, and multiple data tracks
- **Default**: Outputs MOV files with audio embedded without quality degradation
- **Optional**: Extract video/audio separately without recompression (`-c copy`)
- Data track extraction (`djmd`/`dbgi`) with 2 modes:
  - raw: Raw binary (.bin)
  - decode: Generic Protobuf wire format decoding → CSV (channel-specific time series data)
  - both: Both formats
- **Folder support**: Specify a directory to automatically search and batch process all OSV files
- **Flexible output**: Output path is optional, defaults to the same directory as input files

## Installation Guide

#### For macOS Users:
1. **Download the tool:**
   - Go to the [Releases page](https://github.com/yoshihiro0323/osv2mov/releases)
   - Click on the [latest version](https://github.com/yoshihiro0323/osv2mov/releases#latest)
   - Download `osv2mov_darwin_amd64` (for Intel Macs) or `osv2mov_darwin_arm64` (for Apple Silicon Macs)

2. **Install the tool:**
   ```bash
   # Open Terminal (Applications > Utilities > Terminal)
   # Navigate to Downloads folder
   cd ~/Downloads
   
   # Make the file executable
   chmod +x osv2mov_darwin_amd64
   
   # Move to a directory in your PATH (optional but recommended)
   sudo mv osv2mov_darwin_amd64 /usr/local/bin/osv2mov
   
   # Test the installation
   osv2mov help
   ```

#### For Linux Users:
1. **Download the tool:**
   - Go to the [Releases page](https://github.com/yoshihiro0323/osv2mov/releases)
   - Click on the latest version
   - Download `osv2mov_linux_amd64`

2. **Install the tool:**
   ```bash
   # Open Terminal
   # Navigate to Downloads folder
   cd ~/Downloads
   
   # Make the file executable
   chmod +x osv2mov_linux_amd64
   
   # Move to a directory in your PATH
   sudo mv osv2mov_linux_amd64 /usr/local/bin/osv2mov
   
   # Test the installation
   osv2mov help
   ```

### Installing FFmpeg (Required)

**Important:** FFmpeg and FFprobe are required dependencies for this tool to work. You must install them before using osv2mov.

#### macOS:
```bash
# Using Homebrew (recommended)
brew install ffmpeg

# Or download from https://ffmpeg.org/download.html
```

#### Linux:
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install ffmpeg

# CentOS/RHEL
sudo yum install ffmpeg

# Or download from https://ffmpeg.org/download.html
```

#### Verify FFmpeg Installation:
```bash
# Check if FFmpeg is installed
ffmpeg -version

# Check if FFprobe is installed
ffprobe -version
```

If you see version information, FFmpeg is properly installed. If you get "command not found", you need to install FFmpeg first.

## Quick Start Guide for Beginners

### What is OSV?
OSV (OSMO Video) files are created by DJI OSMO 360 cameras. They contain:
- Two fisheye video streams (front and rear cameras)
- Audio recording
- Motion sensor data (IMU - gyroscope, accelerometer)
- Camera calibration data
- Thumbnail images

### First Steps
1. **Install the tool**: Follow the installation guide above
2. **Check your OSV file**: Use the inspect command to see what's inside
3. **Extract videos**: Convert to standard MOV format for easy playback
4. **Get sensor data**: Export IMU data to CSV for analysis

### Basic Workflow
```bash
# 1. Check what's in your OSV file
osv2mov inspect "your_file.OSV"

# 2. Convert to MOV (most common use case)
osv2mov extract "your_file.OSV"

# 3. Get sensor data as CSV
osv2mov extract -c "your_file.OSV"
```

### Your First OSV File

1. **Find your OSV file:**
   - Look for files ending in `.OSV` on your computer
   - Common locations: Downloads folder, Camera/DCIM folders

2. **Open Terminal/Command Prompt:**
   - **macOS:** Applications > Utilities > Terminal
   - **Linux:** Press Ctrl+Alt+T or search for "Terminal"

3. **Navigate to your file:**
   ```bash
   # Example: if your file is in Downloads
   cd ~/Downloads
   
   # List files to see your OSV file
   ls *.OSV
   ```

4. **Run your first command:**
   ```bash
   # Check what's in the file
   osv2mov inspect "CAM_20241201_123456.OSV"
   ```

## Usage

### Basic Commands

```bash
# Show help
./osv2mov help

# Inspect file contents
./osv2mov inspect "/path/to/CAM_....OSV"
# or short form
./osv2mov i "/path/to/CAM_....OSV"
```

### File Extraction

```bash
# Basic extraction (MOV output, same directory as input)
./osv2mov extract "/path/to/CAM_....OSV"
# or short form
./osv2mov e "/path/to/CAM_....OSV"

# Specify output directory
./osv2mov extract -o "/path/to/output" "/path/to/CAM_....OSV"
# or short form
./osv2mov e -o "/path/to/output" "/path/to/CAM_....OSV"

# Verbose output
./osv2mov extract -v "/path/to/CAM_....OSV"

# Extract as separate files
./osv2mov extract -s "/path/to/CAM_....OSV"
# or long form
./osv2mov extract --separate "/path/to/CAM_....OSV"

# Export IMU data as CSV
./osv2mov extract -c "/path/to/CAM_....OSV"

# Combine multiple options
./osv2mov extract -s -c -v "/path/to/CAM_....OSV"

# Overwrite existing files
./osv2mov extract -f "/path/to/CAM_....OSV"

# Specify metadata processing mode
./osv2mov extract -m decode "/path/to/CAM_....OSV"
# or long form
./osv2mov extract --meta both "/path/to/CAM_....OSV"
```

### Batch Processing

```bash
# Process all OSV files in a directory (output to same directory as each file)
./osv2mov extract "/path/to/osv_directory"

# Specify output directory for batch processing
./osv2mov extract -o "/path/to/output" "/path/to/osv_directory"

# Verbose batch processing (show progress)
./osv2mov extract -v "/path/to/osv_directory"

# Extract as separate files for all OSV files in directory
./osv2mov extract -s "/path/to/osv_directory"

# Export IMU data as CSV for all OSV files in directory
./osv2mov extract -c "/path/to/osv_directory"
```

**Batch Processing Features:**
- Recursively searches specified directory
- Automatically detects files with `.osv` extension
- Processes found OSV files sequentially
- Continues processing even if individual files fail (shows warnings)
- Shows progress and detailed results in verbose mode
- Outputs to same directory as each OSV file if no output directory specified

### Options Reference

| Short | Long | Description | Default |
|-------|------|-------------|---------|
| `-o` | `--output` | Output directory (optional) | Same directory as input file |
| `-m` | `--meta` | Metadata processing mode: raw\|decode\|both | decode |
| `-s` | `--separate` | Extract as separate files | false |
| `-c` | `--csv` | Export IMU data as CSV | false |
| `-f` | `--force` | Overwrite existing files | false |
| `-v` | `--verbose` | Verbose output | false |
| `-h` | `--help` | Show help | - |

**Output Directory Behavior:**
- When no output directory is specified:
  - Absolute path: Same directory as input file
  - Relative path: Current directory (`.`)
- When output directory is specified: Use specified directory

### Detailed Help

```bash
# Detailed help for extract command
./osv2mov extract -h
# or
./osv2mov e -h
```

## Output Examples

**MOV Output (default):**
- `<basename>_front.mov` … Front fisheye video + audio
- `<basename>_rear.mov` … Rear fisheye video + audio

**Separate Files Output (with -separate flag):**
- `<basename>_front.hevc.mp4`, `<basename>_rear.hevc.mp4` … Front/rear fisheye HEVC 10bit
- `<basename>.aac.m4a` … AAC 48kHz stereo
- `<basename>_thumb.jpg` … Thumbnail (MJPEG attached_pic)
- `<basename>_djmd_*.bin`, `<basename>_dbgi_*.bin` … Metadata raw binary (optional)

**CSV Output (with -csv flag):**
- `<basename>_djmd.csv` … IMU data (CSV time series data, all streams integrated)

## OSV Track Structure

- Video: HEVC Main10, 3000x3000, ~29.97fps ×2
- Audio: AAC LC 48kHz stereo ×1
- Data: `djmd`, `dbgi`
- Thumbnail: MJPEG attached_pic ×1

`djmd`: Gyroscope/accelerometer data

`dbgi`: Lens calibration and detailed IMU/debug metadata

## IMU Data CSV Output Specification

IMU data extracted from `djmd` data tracks is output as CSV in the following format:

**Basic Format:**
- 1 row = 1 sample (800Hz sampling)
- Header: `Timestamp(s),SampleIndex,Ch0,Ch1,Ch2,Ch3,Ch4,Ch5,Ch6,Ch7,Ch8,Ch9`
- Time calculated from continuous sample number (`t_sec = sample_number / 800`)
- When multiple djmd streams exist, all are integrated into one CSV file

## Common Use Cases

### 1. Video Editing
```bash
# Convert to standard MOV format for editing software
osv2mov extract "video.OSV"
```

### 2. Data Analysis
```bash
# Get motion sensor data for analysis
osv2mov extract -c "video.OSV"
```

### 3. Archival
```bash
# Extract all components separately
osv2mov extract -s "video.OSV"
```

### 4. Batch Processing
```bash
# Process entire folder of recordings
osv2mov extract -v "/path/to/recordings/"
```

## Troubleshooting

### Common Issues
- **"FFmpeg not found"**: Install FFmpeg and ensure it's in your PATH
- **"Permission denied"**: Check file permissions and output directory access
- **"File already exists"**: Use `-f` flag to overwrite or specify different output directory

### Getting Help
```bash
# Show all available commands
osv2mov help

# Show detailed help for specific command
osv2mov extract -h
```

## License

See `LICENSE` file for this repository's license information.