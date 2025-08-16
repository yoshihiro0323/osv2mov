package main

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	command := os.Args[1]
	switch command {
	case "inspect", "i":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: osv2mov inspect <input.osv>")
			fmt.Fprintln(os.Stderr, "   or: osv2mov i <input.osv>")
			os.Exit(2)
		}
		if err := cmdInspect(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	case "extract", "e":
		cmdExtractWithFlags()
	case "help", "h", "--help", "-h":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Available commands: inspect, extract, help")
		os.Exit(2)
	}
}

func cmdExtractWithFlags() {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)

	outputDir := fs.String("o", "", "Output directory (default: same as input file)")
	outputDirLong := fs.String("output", "", "Output directory (default: same as input file)")

	metaMode := fs.String("m", "decode", "Metadata processing mode: raw|decode|both")
	metaModeLong := fs.String("meta", "decode", "Metadata processing mode: raw|decode|both")

	movMode := fs.Bool("mov", true, "Burn audio to MOV file")
	separateMode := fs.Bool("s", false, "Separate files")
	separateModeLong := fs.Bool("separate", false, "Separate files")

	csvMode := fs.Bool("c", false, "Output IMU data in CSV format")
	csvModeLong := fs.Bool("csv", false, "Output IMU data in CSV format")

	forceMode := fs.Bool("f", false, "Overwrite existing files")
	forceModeLong := fs.Bool("force", false, "Overwrite existing files")

	verboseMode := fs.Bool("v", false, "Show detailed output")
	verboseModeLong := fs.Bool("verbose", false, "Show detailed output")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: osv2mov extract [options] <input.osv> or <input_directory>\n")
		fmt.Fprintf(os.Stderr, "   or: osv2mov e [options] <input.osv> or <input_directory>\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		fmt.Fprintf(os.Stderr, "  -o, -output string\n")
		fmt.Fprintf(os.Stderr, "         Output directory (default: same as input file)\n")
		fmt.Fprintf(os.Stderr, "  -m, -meta string\n")
		fmt.Fprintf(os.Stderr, "         Metadata processing mode: raw|decode|both (default: decode)\n")
		fmt.Fprintf(os.Stderr, "  -mov\n")
		fmt.Fprintf(os.Stderr, "         Burn audio to MOV file (default: enabled)\n")
		fmt.Fprintf(os.Stderr, "  -s, -separate\n")
		fmt.Fprintf(os.Stderr, "         Separate files\n")
		fmt.Fprintf(os.Stderr, "  -c, -csv\n")
		fmt.Fprintf(os.Stderr, "         Output IMU data in CSV format\n")
		fmt.Fprintf(os.Stderr, "  -f, -force\n")
		fmt.Fprintf(os.Stderr, "         Overwrite existing files\n")
		fmt.Fprintf(os.Stderr, "  -v, -verbose\n")
		fmt.Fprintf(os.Stderr, "         Show detailed output\n")
		fmt.Fprintf(os.Stderr, "  -h, -help\n")
		fmt.Fprintf(os.Stderr, "         Show this help\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  osv2mov extract input.osv\n")
		fmt.Fprintf(os.Stderr, "  osv2mov extract -o output_dir input.osv\n")
		fmt.Fprintf(os.Stderr, "  osv2mov e -s -c input_directory\n")
		fmt.Fprintf(os.Stderr, "  osv2mov extract --separate --csv input_directory\n")
	}

	if err := fs.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}

	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "Error: Input file or directory not specified")
		fs.Usage()
		os.Exit(2)
	}

	input := args[0]

	outdir := *outputDir
	if outdir == "" {
		outdir = *outputDirLong
	}

	if outdir == "" {
		if filepath.IsAbs(input) {
			outdir = filepath.Dir(input)
		} else {
			outdir = "."
		}
	}

	meta := *metaMode
	if meta == "" {
		meta = *metaModeLong
	}

	separate := *separateMode || *separateModeLong
	csv := *csvMode || *csvModeLong
	force := *forceMode || *forceModeLong
	verbose := *verboseMode || *verboseModeLong

	if verbose {
		fmt.Printf("Input: %s\n", input)
		fmt.Printf("Output directory: %s\n", outdir)
		fmt.Printf("Metadata mode: %s\n", meta)
		fmt.Printf("MOV output: %v\n", *movMode)
		fmt.Printf("Separate files: %v\n", separate)
		fmt.Printf("CSV output: %v\n", csv)
		fmt.Printf("Force overwrite: %v\n", force)
		fmt.Println()
	}

	if err := processInput(input, outdir, meta, *movMode, separate, csv, force, verbose); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func processInput(input, outdir, metaMode string, movMode, separateMode, csvMode, force, verbose bool) error {
	fileInfo, err := os.Stat(input)
	if err != nil {
		return fmt.Errorf("failed to check input path: %v", err)
	}

	if fileInfo.IsDir() {
		return processDirectory(input, outdir, metaMode, movMode, separateMode, csvMode, force, verbose)
	} else {
		return cmdExtract(input, outdir, metaMode, movMode, separateMode, csvMode, force, verbose)
	}
}

func processDirectory(inputDir, outdir, metaMode string, movMode, separateMode, csvMode, force, verbose bool) error {
	if verbose {
		fmt.Printf("Searching for OSV files in directory: %s\n", inputDir)
	}

	osvFiles, err := findOSVFiles(inputDir)
	if err != nil {
		return fmt.Errorf("failed to search for OSV files: %v", err)
	}

	if len(osvFiles) == 0 {
		return fmt.Errorf("no OSV files found in directory: %s", inputDir)
	}

	if verbose {
		fmt.Printf("Found %d OSV files\n", len(osvFiles))
		fmt.Println()
	}

	for i, osvFile := range osvFiles {
		if verbose {
			fmt.Printf("Processing (%d/%d): %s\n", i+1, len(osvFiles), filepath.Base(osvFile))
			fmt.Println(strings.Repeat("-", 50))
		}

		fileOutdir := outdir
		if fileOutdir == "" {
			if filepath.IsAbs(osvFile) {
				fileOutdir = filepath.Dir(osvFile)
			} else {
				fileOutdir = "."
			}
		}

		if err := cmdExtract(osvFile, fileOutdir, metaMode, movMode, separateMode, csvMode, force, verbose); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to process %s: %v\n", filepath.Base(osvFile), err)
			if verbose {
				fmt.Println()
			}
			continue
		}

		if verbose {
			fmt.Printf("Completed: %s\n", filepath.Base(osvFile))
			fmt.Println()
		}
	}

	if verbose {
		fmt.Printf("All OSV files processed (%d files)\n", len(osvFiles))
	}

	return nil
}

func findOSVFiles(dir string) ([]string, error) {
	var osvFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".osv" {
			osvFiles = append(osvFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Strings(osvFiles)
	return osvFiles, nil
}

func usage() {
	fmt.Println("osv2mov - CLI tool to parse and convert OSV files")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  osv2mov <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  inspect, i     Parse and display the content of an OSV file")
	fmt.Println("  extract, e     Extract videos, audio, and metadata from an OSV file")
	fmt.Println("  help, h         Show this help")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  osv2mov inspect input.osv")
	fmt.Println("  osv2mov extract input.osv")
	fmt.Println("  osv2mov extract -o output_dir input.osv")
	fmt.Println("  osv2mov e -s -c input.osv")
	fmt.Println()
	fmt.Println("Detailed help:")
	fmt.Println("  osv2mov extract -h")
	fmt.Println("  osv2mov e -h")
}

func cmdInspect(path string) error {
	raw, err := run("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", "-show_format", path)
	if err != nil {
		return err
	}
	var p probe
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	sum := summarize(&p)
	b, _ := json.MarshalIndent(sum, "", "  ")
	fmt.Println(string(b))
	return nil
}

func cmdExtract(input, outdir, metaMode string, movMode, separateMode, csvMode, force, verbose bool) error {
	if verbose {
		fmt.Printf("Creating output directory: %s\n", outdir)
	}
	if err := os.MkdirAll(outdir, 0o755); err != nil {
		return err
	}
	base := strings.TrimSuffix(filepath.Base(input), filepath.Ext(input))
	subdir := filepath.Join(outdir, base)
	if verbose {
		fmt.Printf("Creating subdirectory: %s\n", subdir)
	}
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		return err
	}
	if verbose {
		fmt.Printf("Parsing OSV file: %s\n", input)
	}
	raw, err := run("ffprobe", "-v", "error", "-print_format", "json", "-show_streams", input)
	if err != nil {
		return err
	}
	var p probe
	if err := json.Unmarshal(raw, &p); err != nil {
		return err
	}
	if verbose {
		fmt.Printf("Number of streams: %d\n", len(p.Streams))
	}
	var vids []int
	var auds []int
	var thumbs []int
	var djmd []int
	var dbgi []int
	for _, s := range p.Streams {
		switch s.CodecType {
		case "video":
			if s.CodecName == "hevc" {
				vids = append(vids, s.Index)
			}
			if s.Disposition.AttachedPic == 1 || s.CodecName == "mjpeg" {
				thumbs = append(thumbs, s.Index)
			}
		case "audio":
			auds = append(auds, s.Index)
		case "data":
			if s.CodecTagString == "djmd" {
				djmd = append(djmd, s.Index)
			}
			if s.CodecTagString == "dbgi" {
				dbgi = append(dbgi, s.Index)
			}
		}
	}
	sort.Ints(vids)
	sort.Ints(auds)
	sort.Ints(thumbs)
	sort.Ints(djmd)
	sort.Ints(dbgi)

	if verbose {
		fmt.Printf("Video streams: %v\n", vids)
		fmt.Printf("Audio streams: %v\n", auds)
		fmt.Printf("Thumbnails: %v\n", thumbs)
		fmt.Printf("DJMD data: %v\n", djmd)
		fmt.Printf("DBGI data: %v\n", dbgi)
		fmt.Println()
	}

	if movMode {
		if verbose {
			fmt.Println("Creating MOV files...")
		}
		if err := createMOVFiles(input, subdir, base, vids, auds, verbose, force); err != nil {
			return err
		}
	}

	if separateMode {
		if verbose {
			fmt.Println("Creating separate files...")
		}
		if err := createSeparateFiles(input, subdir, base, vids, auds, thumbs, djmd, dbgi, metaMode, verbose, force); err != nil {
			return err
		}
	}

	if !movMode && !separateMode {
		if verbose {
			fmt.Println("Creating MOV files (default)...")
		}
		if err := createMOVFiles(input, subdir, base, vids, auds, verbose, force); err != nil {
			return err
		}
	}

	if csvMode && (metaMode == "decode" || metaMode == "both") {
		if len(djmd) > 0 {
			out := filepath.Join(subdir, base+"_djmd.csv")
			if verbose {
				fmt.Printf("Outputting IMU data to CSV: %s\n", out)
			}
			if err := decodeDataTrackToCSVCombined(input, djmd, out); err != nil {
				return err
			}
			if verbose {
				fmt.Printf("CSV output completed: %s\n", out)
			}
		}
	}

	return nil
}

func createMOVFiles(input, subdir, base string, vids, auds []int, verbose, force bool) error {
	if len(vids) == 0 {
		return fmt.Errorf("no video streams found")
	}
	if len(auds) == 0 {
		return fmt.Errorf("no audio streams found")
	}

	for i, vidIdx := range vids {
		if i >= 2 {
			break
		}
		audioIdx := auds[0]
		if len(auds) > i {
			audioIdx = auds[i]
		}

		var suffix string
		if i == 0 {
			suffix = "front"
		} else {
			suffix = "rear"
		}
		out := filepath.Join(subdir, fmt.Sprintf("%s_%s.mov", base, suffix))
		if !force {
			if _, err := os.Stat(out); err == nil {
				return fmt.Errorf("file already exists: %s (use -f to overwrite)", out)
			}
		}
		if verbose {
			fmt.Printf("Creating MOV file: %s (Video:%d, Audio:%d)\n", out, vidIdx, audioIdx)
		}
		args := []string{
			"-y", "-i", input,
			"-map", fmt.Sprintf("0:%d", vidIdx),
			"-map", fmt.Sprintf("0:%d", audioIdx),
			"-c:v", "copy",
			"-c:a", "copy",
			"-f", "mov",
			out,
		}
		if _, err := run("ffmpeg", args...); err != nil {
			return fmt.Errorf("MOV file creation error (v%d): %v", i, err)
		}
		if verbose {
			fmt.Printf("Completed: %s\n", out)
		}
	}
	return nil
}

func createSeparateFiles(input, subdir, base string, vids, auds, thumbs, djmd, dbgi []int, metaMode string, verbose, force bool) error {
	if len(vids) > 0 {
		out := filepath.Join(subdir, base+"_front.hevc.mp4")
		if !force {
			if _, err := os.Stat(out); err == nil {
				return fmt.Errorf("file already exists: %s (use -f to overwrite)", out)
			}
		}
		if verbose {
			fmt.Printf("Creating video file: %s\n", out)
		}
		if _, err := run("ffmpeg", "-y", "-i", input, "-map", "0:"+strconv.Itoa(vids[0]), "-c", "copy", out); err != nil {
			return err
		}
	}
	if len(vids) > 1 {
		out := filepath.Join(subdir, base+"_rear.hevc.mp4")
		if !force {
			if _, err := os.Stat(out); err == nil {
				return fmt.Errorf("file already exists: %s (use -f to overwrite)", out)
			}
		}
		if verbose {
			fmt.Printf("Creating second video file: %s\n", out)
		}
		if _, err := run("ffmpeg", "-y", "-i", input, "-map", "0:"+strconv.Itoa(vids[1]), "-c", "copy", out); err != nil {
			return err
		}
	}
	if len(auds) > 0 {
		out := filepath.Join(subdir, base+".aac.m4a")
		if !force {
			if _, err := os.Stat(out); err == nil {
				return fmt.Errorf("file already exists: %s (use -f to overwrite)", out)
			}
		}
		if verbose {
			fmt.Printf("Creating audio file: %s\n", out)
		}
		if _, err := run("ffmpeg", "-y", "-i", input, "-map", "0:"+strconv.Itoa(auds[0]), "-c", "copy", out); err != nil {
			return err
		}
	}
	if len(thumbs) > 0 {
		out := filepath.Join(subdir, base+"_thumb.jpg")
		if !force {
			if _, err := os.Stat(out); err == nil {
				return fmt.Errorf("file already exists: %s (use -f to overwrite)", out)
			}
		}
		if verbose {
			fmt.Printf("Creating thumbnail: %s\n", out)
		}
		if _, err := run("ffmpeg", "-y", "-i", input, "-map", "0:"+strconv.Itoa(thumbs[0]), "-frames:v", "1", out); err != nil {
			return err
		}
	}
	if metaMode == "raw" || metaMode == "both" {
		for i, idx := range djmd {
			out := filepath.Join(subdir, base+"_djmd_"+strconv.Itoa(i)+".bin")
			if !force {
				if _, err := os.Stat(out); err == nil {
					return fmt.Errorf("file already exists: %s (use -f to overwrite)", out)
				}
			}
			if verbose {
				fmt.Printf("Creating DJMD data file: %s\n", out)
			}
			if _, err := run("ffmpeg", "-y", "-i", input, "-map", "0:"+strconv.Itoa(idx), "-c", "copy", "-f", "data", out); err != nil {
				return err
			}
		}
		for i, idx := range dbgi {
			out := filepath.Join(subdir, base+"_dbgi_"+strconv.Itoa(i)+".bin")
			if !force {
				if _, err := os.Stat(out); err == nil {
					return fmt.Errorf("file already exists: %s (use -f to overwrite)", out)
				}
			}
			if verbose {
				fmt.Printf("Creating DBGI data file: %s\n", out)
			}
			if _, err := run("ffmpeg", "-y", "-i", input, "-map", "0:"+strconv.Itoa(idx), "-c", "copy", "-f", "data", out); err != nil {
				return err
			}
		}
	}

	if verbose {
		fmt.Println("All processing completed")
	}
	return nil
}

func run(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s failed: %v\n%s", name, err, string(out))
	}
	return out, nil
}

type probe struct {
	Streams []stream `json:"streams"`
	Format  format   `json:"format"`
}

type stream struct {
	Index          int            `json:"index"`
	CodecName      string         `json:"codec_name"`
	CodecType      string         `json:"codec_type"`
	CodecTagString string         `json:"codec_tag_string"`
	Width          int            `json:"width"`
	Height         int            `json:"height"`
	RFrameRate     string         `json:"r_frame_rate"`
	Disposition    disposition    `json:"disposition"`
	Tags           map[string]any `json:"tags"`
}

type disposition struct {
	AttachedPic int `json:"attached_pic"`
}

type format struct {
	Duration string            `json:"duration"`
	Tags     map[string]string `json:"tags"`
}

type summary struct {
	Container map[string]string `json:"container"`
	Video     []map[string]any  `json:"video"`
	Audio     []map[string]any  `json:"audio"`
	Data      []map[string]any  `json:"data"`
	Thumb     []map[string]any  `json:"thumb"`
}

func summarize(p *probe) *summary {
	sum := &summary{
		Container: map[string]string{},
	}
	if p.Format.Tags != nil {
		for k, v := range p.Format.Tags {
			sum.Container[k] = v
		}
	}
	for _, s := range p.Streams {
		m := map[string]any{
			"index": s.Index,
			"codec": s.CodecName,
			"type":  s.CodecType,
		}
		switch s.CodecType {
		case "video":
			m["w"] = s.Width
			m["h"] = s.Height
			m["r_frame_rate"] = s.RFrameRate
			if s.Disposition.AttachedPic == 1 || s.CodecName == "mjpeg" {
				sum.Thumb = append(sum.Thumb, m)
			} else {
				sum.Video = append(sum.Video, m)
			}
		case "audio":
			sum.Audio = append(sum.Audio, m)
		case "data":
			m["tag"] = s.CodecTagString
			sum.Data = append(sum.Data, m)
		}
	}
	return sum
}

type packetDump struct {
	Packets []packet `json:"packets"`
}

type packet struct {
	PtsTime string `json:"pts_time"`
	Data    string `json:"data"`
}

func decodeHexDump(s string) []byte {
	var nibbles []byte
	lines := strings.Split(s, "\n")
	for _, ln := range lines {
		idx := strings.Index(ln, ":")
		if idx < 0 {
			continue
		}
		hexpart := ln[idx+1:]
		for i := 0; i < len(hexpart); i++ {
			c := hexpart[i]
			if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
				nibbles = append(nibbles, c)
			}
		}
	}
	// pair nibbles into bytes
	out := make([]byte, 0, len(nibbles)/2)
	for i := 0; i+1 < len(nibbles); i += 2 {
		var b [1]byte
		if _, err := hex.Decode(b[:], []byte{nibbles[i], nibbles[i+1]}); err == nil {
			out = append(out, b[0])
		}
	}
	return out
}

func readVarint(b []byte, i int) (uint64, int) {
	var v uint64
	var shift uint
	start := i
	for i < len(b) {
		c := b[i]
		v |= uint64(c&0x7F) << shift
		i++
		if c < 0x80 {
			return v, i - start
		}
		shift += 7
		if shift > 63 {
			break
		}
	}
	return 0, 0
}

func decodeDataTrackToCSVCombined(input string, streamIndices []int, out string) error {
	var allIMUData []IMURecord
	var sampleRate float32 = 800.0
	var globalSampleIndex int = 0

	for _, streamIndex := range streamIndices {
		raw, err := run("ffprobe", "-v", "error", "-print_format", "json", "-show_packets", "-show_data", "-select_streams", strconv.Itoa(streamIndex), input)
		if err != nil {
			return err
		}
		var dump packetDump
		if err := json.Unmarshal(raw, &dump); err != nil {
			return err
		}

		for _, p := range dump.Packets {
			b := decodeHexDump(p.Data)
			imuData := decodeIMUData(b, sampleRate)

			// サンプルインデックスを連続させる
			for i := range imuData {
				imuData[i].SampleIndex = globalSampleIndex
				globalSampleIndex++
			}

			allIMUData = append(allIMUData, imuData...)
		}
	}

	if len(allIMUData) == 0 {
		return fmt.Errorf("IMU data not found")
	}

	// 時間を再計算（連続したサンプルインデックスに基づく）
	for i := range allIMUData {
		allIMUData[i].Timestamp = float64(allIMUData[i].SampleIndex) / float64(sampleRate)
	}

	return writeIMUCSV(allIMUData, out)
}

type IMURecord struct {
	Timestamp                                        float64
	SampleIndex                                      int
	Ch0, Ch1, Ch2, Ch3, Ch4, Ch5, Ch6, Ch7, Ch8, Ch9 int16
}

func decodeIMUData(b []byte, sampleRate float32) []IMURecord {
	var records []IMURecord
	i := 0
	for i < len(b) {
		key, n := readVarint(b, i)
		if n == 0 {
			break
		}
		i += n
		fieldNum := int(key >> 3)
		wireType := int(key & 0x7)

		if fieldNum == 3 && wireType == 2 {
			l, m := readVarint(b, i)
			if m == 0 || int(l) < 0 || i+m+int(l) > len(b) {
				break
			}
			i += m
			payload := b[i : i+int(l)]
			i += int(l)

			imuRecords := parseIMUPayload(payload, sampleRate)
			records = append(records, imuRecords...)
		} else {
			switch wireType {
			case 0:
				_, m := readVarint(b, i)
				if m == 0 {
					i = len(b)
					break
				}
				i += m
			case 1:
				if i+8 > len(b) {
					i = len(b)
					break
				}
				i += 8
			case 2:
				l, m := readVarint(b, i)
				if m == 0 || int(l) < 0 || i+m+int(l) > len(b) {
					i = len(b)
					break
				}
				i += m + int(l)
			case 5:
				if i+4 > len(b) {
					i = len(b)
					break
				}
				i += 4
			default:
				i = len(b)
			}
		}
	}
	return records
}

func parseIMUPayload(payload []byte, sampleRate float32) []IMURecord {
	var records []IMURecord
	i := 0

	for i < len(payload) {
		key, n := readVarint(payload, i)
		if n == 0 {
			break
		}
		i += n
		fieldNum := int(key >> 3)
		wireType := int(key & 0x7)

		if fieldNum == 2 && wireType == 2 {
			l, m := readVarint(payload, i)
			if m == 0 || int(l) < 0 || i+m+int(l) > len(payload) {
				break
			}
			i += m
			headerData := payload[i : i+int(l)]
			i += int(l)

			if len(headerData) >= 20 {
				sampleRate = float32(binary.LittleEndian.Uint32(headerData[0:4]))
			}
		} else if fieldNum == 3 && wireType == 2 {
			l, m := readVarint(payload, i)
			if m == 0 || int(l) < 0 || i+m+int(l) > len(payload) {
				break
			}
			i += m
			imuData := payload[i : i+int(l)]
			i += int(l)

			records = parseIMURecords(imuData, sampleRate)
		} else {
			switch wireType {
			case 0:
				_, m := readVarint(payload, i)
				if m == 0 {
					i = len(payload)
					break
				}
				i += m
			case 1:
				if i+8 > len(payload) {
					i = len(payload)
					break
				}
				i += 8
			case 2:
				l, m := readVarint(payload, i)
				if m == 0 || int(l) < 0 || i+m+int(l) > len(payload) {
					i = len(payload)
					break
				}
				i += m + int(l)
			case 5:
				if i+4 > len(payload) {
					i = len(payload)
					break
				}
				i += 4
			default:
				i = len(payload)
			}
		}
	}
	return records
}

func parseIMURecords(imuData []byte, sampleRate float32) []IMURecord {
	var records []IMURecord
	recordSize := 24

	for i := 0; i+recordSize <= len(imuData); i += recordSize {
		record := imuData[i : i+recordSize]

		_ = binary.LittleEndian.Uint32(record[0:4])
		ch0 := int16(binary.LittleEndian.Uint16(record[4:6]))
		ch1 := int16(binary.LittleEndian.Uint16(record[6:8]))
		ch2 := int16(binary.LittleEndian.Uint16(record[8:10]))
		ch3 := int16(binary.LittleEndian.Uint16(record[10:12]))
		ch4 := int16(binary.LittleEndian.Uint16(record[12:14]))
		ch5 := int16(binary.LittleEndian.Uint16(record[14:16]))
		ch6 := int16(binary.LittleEndian.Uint16(record[16:18]))
		ch7 := int16(binary.LittleEndian.Uint16(record[18:20]))
		ch8 := int16(binary.LittleEndian.Uint16(record[20:22]))
		ch9 := int16(binary.LittleEndian.Uint16(record[22:24]))

		sampleIndex := len(records)
		timeSec := float64(sampleIndex) / float64(sampleRate)

		imuRecord := IMURecord{
			Timestamp:   timeSec,
			SampleIndex: sampleIndex,
			Ch0:         ch0,
			Ch1:         ch1,
			Ch2:         ch2,
			Ch3:         ch3,
			Ch4:         ch4,
			Ch5:         ch5,
			Ch6:         ch6,
			Ch7:         ch7,
			Ch8:         ch8,
			Ch9:         ch9,
		}

		records = append(records, imuRecord)
	}

	return records
}

func writeIMUCSV(records []IMURecord, out string) error {
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	header := "Timestamp(s),SampleIndex,Ch0,Ch1,Ch2,Ch3,Ch4,Ch5,Ch6,Ch7,Ch8,Ch9\n"
	if _, err := w.WriteString(header); err != nil {
		return err
	}

	for _, record := range records {
		line := fmt.Sprintf("%.6f,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d,%d\n",
			record.Timestamp,
			record.SampleIndex,
			record.Ch0, record.Ch1, record.Ch2, record.Ch3, record.Ch4,
			record.Ch5, record.Ch6, record.Ch7, record.Ch8, record.Ch9)

		if _, err := w.WriteString(line); err != nil {
			return err
		}
	}

	return nil
}
