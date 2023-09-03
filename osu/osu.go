package osu

import (
	"bufio"
	"github.com/Nerinyan/nerinyan-download-apiv1/logger"
	"io"
	"regexp"
	"strings"
)

var _HIT_OBJECT_FN, _ = regexp.Compile(`(?:[0-9]+?:){4,}(.+?)(?:$|,|:)`)

type OsuFile struct {
	AudioFilename string
	Version       string
	Background    []string
	Video         []string
	StoryBoard    []string
	HitSound      []string
}

func ParseOsuFileInfo(reader io.Reader) (res OsuFile) {
	scanner := bufio.NewScanner(reader)
	background := map[string]bool{}
	video := map[string]bool{}
	storyBoard := map[string]bool{}
	hitSound := map[string]bool{}

	var section = "General"
	for scanner.Scan() {
		line := scanner.Text()

		if shouldSkipLine(line) {
			continue
		}

		if section != "Metadata" {
			line = stripComments(line)
		}

		line = strings.TrimRight(line, " \t")

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.Trim(line, "[]")
			continue
		}
		switch section {
		case "General": //AudioFilename
			split := strings.Split(line, ":")
			if len(split) < 2 {
				continue
			}
			switch split[0] {
			case "AudioFilename":
				res.AudioFilename = strings.TrimSpace(split[1])
			}

		case "Metadata":
			split := strings.Split(line, ":")
			if len(split) < 2 {
				continue
			}
			switch split[0] {
			case "Version":
				res.Version = strings.TrimSpace(split[1])
			}

		case "Events": // .osb 가 여기에 해당
			split := strings.Split(line, ",")
			if len(split) < 1 {
				continue
			}
			switch split[0] {
			case "0", "Background":
				background[cleanFilename(split[2])] = true
			case "1", "Video":
				video[cleanFilename(split[2])] = true
			case "4", "Sprite":
				storyBoard[cleanFilename(split[3])] = true
			}

		case "HitObjects":
			group := _HIT_OBJECT_FN.FindStringSubmatch(line) // (?:[0-9]+?:){4,}(.+?)(?:$|,|:)
			if len(group) > 0 && group[0] != "" {
				hitSound[cleanFilename(group[0])] = true
			}
		}
	}
	res.StoryBoard = func() (res []string) {
		for k := range storyBoard {
			res = append(res, k)
		}
		return
	}()
	res.Video = func() (res []string) {
		for k := range video {
			res = append(res, k)
		}
		return
	}()
	res.HitSound = func() (res []string) {
		for k := range hitSound {
			res = append(res, k)
		}
		return
	}()
	res.Background = func() (res []string) {
		for k := range background {
			res = append(res, k)
		}
		return
	}()
	if err := scanner.Err(); err != nil {
		logger.Errorf("An error occurred: %v", err)
	}
	return
}

func shouldSkipLine(line string) bool {
	line = strings.TrimSpace(line)
	return line == "" || strings.HasPrefix(line, "//")
}

func stripComments(line string) string {
	index := strings.Index(line, "//")
	if index > 0 {
		return line[:index]
	}
	return line
}

func cleanFilename(path string) string {
	return strings.ReplaceAll(strings.Trim(path, `"`), `\`, `/`)
}
