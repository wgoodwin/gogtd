package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/wgoodwin/govotl"
)

type Config struct {
	GTDPath string `envconfig:"GTDPATH" default:"."`
}

func main() {
	log.SetPrefix("")

	// Load Config
	var config Config
	err := envconfig.Process("gogtd", &config)
	if err != nil {
		log.Fatal("Failed to load environment: ", err.Error())
	}

	// Parse the arguments
	var command string = "unknown"

	if len(os.Args) > 1 {
		command = os.Args[1]
	}

	switch command {
	case "archive":
		archiver(config, os.Args[2:])
	default:
		log.Fatal("Unknown command: ", command)
	}
}

func archiver(config Config, args []string) {
	var infile string = "next_actions.otl"
	var outfile string = "archive.otl"

	if len(args) > 0 {
		infile = args[0]
	}

	if len(args) > 1 {
		outfile = args[1]
	}

	// Load Infile
	fmt.Printf("Starting archiving from %s to %s...\n", infile, outfile)
	inDoc, err := govotl.LoadFile(config.GTDPath + "/" + infile)
	if err != nil {
		log.Fatalf("Unable to load input file [%s]: %s\n", infile, err.Error())
	}

	// Stat outfile in case we expect an error other than non existant
	if _, err = os.Stat(config.GTDPath + "/" + outfile); !(err == nil || errors.Is(err, os.ErrNotExist)) {
		log.Fatalf("Unable to open archive file [%s].\n", outfile)
	}

	// If malformed, we assume we need to start fresh
	var outDoc govotl.VOTLDoc
	outDoc, err = govotl.LoadFile(config.GTDPath + "/" + outfile)
	if err != nil {
		fmt.Printf("No previous archive file found [%s] or file is malformed, starting fresh.\n", outfile)
	}

	/* TODO Uncomment this when I'm able to add a child element
	archiveTag := govotl.NewVOTLElement("Archived")
	_ = archiveTag.AddChild(time.Now().Format(time.DateOnly))
	*/
	for inIndex, element := range inDoc {
		// We only parse heading root objects
		if element.Type == govotl.Heading {
			var keep, move []govotl.VOTLElement

			// We only filter checked task elements
			for _, child := range element.Children {
				if child.Type == govotl.Checkbox {
					if child.Checked {
						// TODO The archive tagging should change when I'm able to add by element
						child.AddChild("Archived")
						child.AddChild("\t" + time.Now().Format(time.DateOnly))
						move = append(move, child)
						continue
					}
				}
				keep = append(keep, child)
			}

			// Look for the heading in the outDoc and if it's not found add it
			fmt.Printf("Archiving %d entries in header: %s\n", len(move), element.Value)
			var found bool = false
			for outIndex, outElement := range outDoc {
				if outElement.Type == govotl.Heading {
					if outElement.Value == element.Value {
						found = true
						outElement.Children = append(outElement.Children, move...)
						outDoc[outIndex] = outElement
					}
				}
			}

			if !found {
				newHead := govotl.NewVOTLElement(element.Value)
				newHead.Children = move
				outDoc = append(outDoc, newHead)
			}

			element.Children = keep
			inDoc[inIndex] = element
		}
	}

	fmt.Println("Writing updated input file:", infile)
	if err = govotl.WriteFile(inDoc, config.GTDPath+"/"+infile); err != nil {
		log.Fatalf("Failed to write new input file: %s\n", err.Error())
	}
	fmt.Println("Writing updated archive file:", outfile)
	if err = govotl.WriteFile(outDoc, config.GTDPath+"/"+outfile); err != nil {
		log.Fatalf("Failed to write new output file: %s\n", err.Error())
	}

	fmt.Println("Archiving Complete.")
}
