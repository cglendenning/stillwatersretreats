package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Review struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	Location string `json:"location"`
	Text     string `json:"text"`
}

func main() {
	wd, _ := os.Getwd()
	inputFile := filepath.Join(wd, "..", "assets", "data", "bvc-raw.txt")
	outputFile := filepath.Join(wd, "..", "assets", "data", "bvc-reviews.json")

	file, err := os.Open(inputFile)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		l := scanner.Text()
		parts := strings.SplitN(l, "|", 2)
		if len(parts) == 2 {
			lines = append(lines, strings.TrimSpace(parts[1]))
		} else {
			lines = append(lines, strings.TrimSpace(l))
		}
	}

	var reviews []Review
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			i++
			continue
		}

		// Skip response lines and host names
		if strings.HasPrefix(line, "Response from") || strings.HasPrefix(line, "Translated from") || strings.HasPrefix(line, "Show more") || line == "Craig & Mercedes" || line == "Craig" {
			i++
			continue
		}

		// Skip rating lines and separators
		if strings.HasPrefix(line, "Rating,") || line == ", ·" {
			i++
			continue
		}

		// This should be a reviewer name
		reviewer := line
		i++

		// Next line should be location or time on Airbnb
		var location string
		if i < len(lines) {
			line = strings.TrimSpace(lines[i])
			if line != "" && line != reviewer && !strings.HasPrefix(line, "Rating,") && !strings.HasPrefix(line, "Response from") {
				// Check if it looks like a location (contains comma and not time periods)
				if strings.Contains(line, ",") && !strings.Contains(line, "on Airbnb") && !strings.Contains(line, "years") && !strings.Contains(line, "months") {
					location = line
				}
				i++
			}
		}

		// Skip duplicate reviewer name
		if i < len(lines) && strings.TrimSpace(lines[i]) == reviewer {
			i++
		}

		// Skip rating line
		if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "Rating,") {
			i++
		}

		// Skip ", ·"
		if i < len(lines) && strings.TrimSpace(lines[i]) == ", ·" {
			i++
		}

		// Get date
		var date string
		if i < len(lines) {
			date = strings.TrimSpace(lines[i])
			i++
		}

		// Skip ", ·"
		if i < len(lines) && strings.TrimSpace(lines[i]) == ", ·" {
			i++
		}

		// Skip stay type
		if i < len(lines) {
			i++
		}

		// Collect review text
		var reviewLines []string
		for i < len(lines) {
			line = strings.TrimSpace(lines[i])
			if line == "" {
				i++
				continue
			}
			// Stop at response or next review
			if strings.HasPrefix(line, "Response from") || strings.HasPrefix(line, "Translated from") || strings.HasPrefix(line, "Show more") {
				break
			}
			// Stop if we see a pattern that looks like a new review starting
			if len(strings.Split(line, " ")) <= 4 && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				if strings.Contains(nextLine, ",") && !strings.Contains(nextLine, "·") && !strings.Contains(nextLine, "on Airbnb") {
					break
				}
			}
			reviewLines = append(reviewLines, line)
			i++
		}

		// Skip response if present
		if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "Response from") {
			for i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Rating,") {
				i++
			}
		}

		// Skip "Show more" if present
		if i < len(lines) && strings.TrimSpace(lines[i]) == "Show more" {
			i++
		}

		// Add review if we have meaningful text
		reviewText := strings.Join(reviewLines, " ")
		if len(reviewText) > 0 && reviewText != "." && !strings.HasPrefix(reviewText, "Thank you") {
			reviews = append(reviews, Review{
				Name:     reviewer,
				Date:     date,
				Location: location,
				Text:     reviewText,
			})
		}
	}

	jsonData, err := json.MarshalIndent(reviews, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		fmt.Printf("Error writing file: %v\n", err)
		return
	}

	fmt.Println("Parsing complete. Output written to", outputFile)
}
