package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Example of how I ran it before...
// go run parse_reviews.go -input ../assets/data/vp-raw.txt -output ../assets/data/vp-test.json

type Review struct {
	Name     string `json:"name"`
	Date     string `json:"date"`
	Location string `json:"location"`
	Text     string `json:"text"`
}

func isLocation(line string) bool {
	return strings.Contains(line, ",") &&
		!strings.Contains(line, "on Airbnb") &&
		!strings.Contains(line, "years") &&
		!strings.Contains(line, "months") &&
		line != ", ·"
}

func parseReviews(inputFile, outputFile string, overwrite bool) error {
	// Check if output file exists and overwrite is not set
	if !overwrite {
		if _, err := os.Stat(outputFile); err == nil {
			return fmt.Errorf("output file '%s' already exists. Use -overwrite flag to overwrite", outputFile)
		}
	}

	// Open input file
	file, err := os.Open(inputFile)
	if err != nil {
		return fmt.Errorf("error opening input file: %v", err)
	}
	defer file.Close()

	// Read all lines
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

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input file: %v", err)
	}

	var reviews []Review
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])

		// Skip empty lines
		if line == "" {
			i++
			continue
		}

		// Skip host responses and metadata
		if strings.HasPrefix(line, "Response from") ||
			strings.HasPrefix(line, "Translated from") ||
			line == "Show more" ||
			line == "Craig & Mercedes" ||
			line == "Craig" ||
			strings.HasPrefix(line, "Rating,") ||
			line == ", ·" {
			i++
			continue
		}

		// Start of a new review - this is the reviewer name
		reviewerName := line
		i++

		if i >= len(lines) {
			break
		}

		// Next line is either location or "X on Airbnb" or empty
		location := ""
		if i < len(lines) {
			nextLine := strings.TrimSpace(lines[i])
			if nextLine != "" && nextLine != reviewerName && !strings.HasPrefix(nextLine, "Rating,") {
				if isLocation(nextLine) {
					location = nextLine
				}
				i++
			}
		}

		// Skip duplicate reviewer name or empty line
		if i < len(lines) {
			line := strings.TrimSpace(lines[i])
			if line == reviewerName || line == "" {
				i++
			}
		}

		// Must have rating line
		if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "Rating,") {
			i++
		} else {
			// Not a valid review, skip this entry
			continue
		}

		// Skip ", ·"
		if i < len(lines) && strings.TrimSpace(lines[i]) == ", ·" {
			i++
		}

		// Get date
		date := ""
		if i < len(lines) && strings.TrimSpace(lines[i]) != "" && !strings.HasPrefix(strings.TrimSpace(lines[i]), "Rating,") {
			date = strings.TrimSpace(lines[i])
			i++
		}

		// Skip ", ·"
		if i < len(lines) && strings.TrimSpace(lines[i]) == ", ·" {
			i++
		}

		// Skip stay type
		if i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			i++
		}

		// Collect review text until we hit a response or next reviewer
		var reviewTextLines []string
		for i < len(lines) {
			line = strings.TrimSpace(lines[i])

			if line == "" {
				i++
				continue
			}

			// Stop at response
			if strings.HasPrefix(line, "Response from") ||
				strings.HasPrefix(line, "Translated from") ||
				line == "Show more" {
				break
			}

			// Check if this looks like a new reviewer name
			if len(strings.Split(line, " ")) <= 4 && i+1 < len(lines) {
				nextLine := strings.TrimSpace(lines[i+1])
				// If next line is location OR "X on Airbnb", this is likely a new reviewer
				if isLocation(nextLine) || strings.Contains(nextLine, "on Airbnb") {
					// Check if there's a duplicate name after that
					if i+2 < len(lines) && strings.TrimSpace(lines[i+2]) == line {
						// This is definitely a new reviewer
						break
					}
				}
			}

			reviewTextLines = append(reviewTextLines, line)
			i++
		}

		// Skip response section if present
		if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "Response from") {
			// Skip until we find a line that's not part of the response
			i++
			for i < len(lines) {
				line = strings.TrimSpace(lines[i])
				if strings.HasPrefix(line, "Rating,") {
					break
				}
				// Check if this looks like the start of a new review
				if line != "" && len(strings.Split(line, " ")) <= 4 && i+1 < len(lines) {
					nextLine := strings.TrimSpace(lines[i+1])
					if isLocation(nextLine) || strings.Contains(nextLine, "on Airbnb") {
						break
					}
				}
				i++
			}
		}

		// Skip "Show more" if present
		if i < len(lines) && strings.TrimSpace(lines[i]) == "Show more" {
			i++
		}

		// Add review if we have meaningful text
		reviewText := strings.Join(reviewTextLines, " ")
		if reviewText != "" && reviewText != "." {
			reviews = append(reviews, Review{
				Name:     reviewerName,
				Date:     date,
				Location: location,
				Text:     reviewText,
			})
		}
	}

	// Write to output file
	jsonData, err := json.MarshalIndent(reviews, "", "  ")
	if err != nil {
		return fmt.Errorf("error marshaling JSON: %v", err)
	}

	err = os.WriteFile(outputFile, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("error writing output file: %v", err)
	}

	fmt.Printf("Parsing complete. Output written to %s\n", outputFile)
	fmt.Printf("Total reviews parsed: %d\n", len(reviews))

	return nil
}

func main() {
	// Define command-line flags
	inputFile := flag.String("input", "", "Input file path (required)")
	outputFile := flag.String("output", "", "Output file path (required)")
	overwrite := flag.Bool("overwrite", false, "Overwrite output file if it exists")
	help := flag.Bool("help", false, "Show usage information")

	flag.Parse()

	// Show help if requested
	if *help {
		fmt.Println("Usage: parse_reviews -input <input_file> -output <output_file> [-overwrite]")
		fmt.Println("\nOptions:")
		flag.PrintDefaults()
		fmt.Println("\nExample:")
		fmt.Println("  parse_reviews -input assets/data/bvc-raw.txt -output assets/data/bvc-reviews.json")
		fmt.Println("  parse_reviews -input assets/data/bvc-raw.txt -output assets/data/bvc-reviews.json -overwrite")
		os.Exit(0)
	}

	// Validate required flags
	if *inputFile == "" || *outputFile == "" {
		fmt.Println("Error: Both -input and -output flags are required")
		fmt.Println("\nUsage: parse_reviews -input <input_file> -output <output_file> [-overwrite]")
		fmt.Println("Run 'parse_reviews -help' for more information")
		os.Exit(1)
	}

	// Parse reviews
	err := parseReviews(*inputFile, *outputFile, *overwrite)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
