package main

import (
	"container/list"
	"encoding/json"
	"fmt"
	"slices"
	"unicode"
)

type _list struct {
	Tree         bool   `short:"t" help:"Hierarchical output"`
	OutputFormat string `short:"o" help:"Output format. One of: ${enum}" enum:"json, yaml, xml, toml, props, shell, csv, tsv," default:""`
	YQ           string `short:"y" help:"YQ expression to apply to the output. Ignored if output format is not specified"`
}

type ListCmd struct {
	List _list `cmd:"" aliases:"l" help:"List IOMMU groups and PCI devices"`
}

// Run executes the command
func (cmd *_list) Run(globals *Globals) error {
	pciDevices, err := ParsePciDevices()
	if err != nil {
		return err
	}

	groups := map[string][]PciDevice{}
	for _, dev := range pciDevices {
		if groups[dev.IommuGroup] == nil {
			groups[dev.IommuGroup] = []PciDevice{dev}
			continue
		}
		groups[dev.IommuGroup] = append(groups[dev.IommuGroup], dev)
	}

	// Print the output in specified format
	var jsonObject []byte
	// Apply YQ expression
	if slices.Contains([]string{"csv", "tsv"}, cmd.OutputFormat) {
		jsonObject, _ = json.Marshal(pciDevices)
	} else {
		jsonObject, _ = json.Marshal(groups)
	}
	yqResult, err := yq(globals, cmd.YQ, jsonObject)
	if err != nil {
		return fmt.Errorf("error applying YQ expression: %w", err)
	}
	if len(cmd.OutputFormat) > 0 {
		out, err := yqEncode(yqResult, cmd.OutputFormat, true)
		if err != nil {
			return fmt.Errorf("error encoding output: %w", err)
		}
		fmt.Println(string(out))
		return nil
	}

	// Pretty print all devices in each iommu group
	err = prettyPrintDevices(yqResult, cmd)
	if err != nil {
		return fmt.Errorf("error pretty printing devices: %w", err)
	}

	return nil
}

// NaturalCompare performs natural string comparison.
// Returns:
//   - negative if a < b
//   - zero if a == b
//   - positive if a > b
func NaturalCompare(a, b string) int {
	aRunes := []rune(a)
	bRunes := []rune(b)
	aLen, bLen := len(aRunes), len(bRunes)

	for i, j := 0, 0; i < aLen && j < bLen; {
		// Skip leading zeros in numbers
		if unicode.IsDigit(aRunes[i]) && unicode.IsDigit(bRunes[j]) {
			// Skip leading zeros in both strings
			for i < aLen && aRunes[i] == '0' {
				i++
			}
			for j < bLen && bRunes[j] == '0' {
				j++
			}
			// Find number length in both strings
			aNumStart, bNumStart := i, j
			for i < aLen && unicode.IsDigit(aRunes[i]) {
				i++
			}
			for j < bLen && unicode.IsDigit(bRunes[j]) {
				j++
			}
			// Compare number lengths
			aNumLen := i - aNumStart
			bNumLen := j - bNumStart
			if aNumLen != bNumLen {
				return aNumLen - bNumLen
			}
			// Compare numbers digit by digit
			for k := 0; k < aNumLen; k++ {
				if diff := int(aRunes[aNumStart+k]) - int(bRunes[bNumStart+k]); diff != 0 {
					return diff
				}
			}
			continue
		}
		// Compare non-digit characters
		if diff := int(aRunes[i]) - int(bRunes[j]); diff != 0 {
			return diff
		}
		i++
		j++
	}
	// Handle different lengths
	return len(a) - len(b)
}

func prettyPrintDevices(yqResult *list.List, cmd *_list) error {
	out, err := yqEncode(yqResult, "json", false)
	if err != nil {
		return fmt.Errorf("error encoding output: %w", err)
	}

	// Sort map by key
	groups := map[string][]PciDevice{}
	err = json.Unmarshal(out, &groups)
	if err != nil {
		return fmt.Errorf("error unmarshalling JSON: %w", err)
	}

	keys := make([]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	slices.SortFunc(keys, NaturalCompare)

	for _, g := range keys {
		if cmd.Tree {
			fmt.Printf("IOMMU Group %s:\n", g)
		}
		prevClass := ""
		for _, dev := range groups[g] {
			nl := ""
			spacer := " "
			if cmd.Tree {
				spacer = "  "
				nl = "\n"
			} else {
				fmt.Printf("IOMMU Group %s:", g)
			}
			class := fmt.Sprintf("%s%s [%s]:%s", spacer, dev.DeviceClass, dev.Class, nl)
			if cmd.Tree {
				if class == prevClass {
					class = ""
				}
				spacer = "    "
			}
			if class != "" {
				prevClass = class
			}
			fmt.Printf(
				"%s%s%s %s %s [%s:%s] (rev %s) driver: %s\n",
				class, spacer, dev.Bus, dev.VendorName, dev.DeviceName, dev.VendorID, dev.DeviceID, dev.Revision, dev.KernelDriver,
			)
		}
	}
	return nil
}
