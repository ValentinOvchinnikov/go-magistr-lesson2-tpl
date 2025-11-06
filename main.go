package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: %s <yamlfile>\n", os.Args[0])
		os.Exit(1)
	}
	file := os.Args[1]

	data, err := os.ReadFile(file)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", file, err)
		os.Exit(1)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		fmt.Fprintf(os.Stderr, "%s: %v\n", file, err)
		os.Exit(1)
	}

	if len(root.Content) == 0 {
		fmt.Fprintf(os.Stderr, "%s: empty yaml\n", file)
		os.Exit(1)
	}

	doc := root.Content[0]
	if doc.Kind != yaml.MappingNode {
		fmt.Fprintf(os.Stderr, "%s: invalid yaml structure\n", file)
		os.Exit(1)
	}

	var hasErr bool

	req := map[string]bool{"apiVersion": false, "kind": false, "metadata": false, "spec": false}

	for i := 0; i < len(doc.Content); i += 2 {
		key := doc.Content[i]
		val := doc.Content[i+1]
		switch key.Value {
		case "apiVersion":
			req["apiVersion"] = true
			if val.Value != "v1" {
				printErr(file, val.Line, fmt.Sprintf("apiVersion has unsupported value '%s'", val.Value))
				hasErr = true
			}
		case "kind":
			req["kind"] = true
			if val.Value != "Pod" {
				printErr(file, val.Line, fmt.Sprintf("kind has unsupported value '%s'", val.Value))
				hasErr = true
			}
		case "metadata":
			req["metadata"] = true
			hasErr = validateMetadata(file, val) || hasErr
		case "spec":
			req["spec"] = true
			hasErr = validateSpec(file, val) || hasErr
		}
	}

	for k, ok := range req {
		if !ok {
			fmt.Fprintf(os.Stderr, "%s is required\n", k)
			hasErr = true
		}
	}

	if hasErr {
		os.Exit(1)
	}
	os.Exit(0)
}

func printErr(file string, line int, msg string) {
	fmt.Fprintf(os.Stderr, "%s:%d %s\n", file, line, msg)
}

func validateMetadata(file string, n *yaml.Node) bool {
	hasErr := false
	name := findMapKey(n, "name")
	if name == nil {
		fmt.Fprintln(os.Stderr, "metadata.name is required")
		hasErr = true
	}
	return hasErr
}

func validateSpec(file string, n *yaml.Node) bool {
	hasErr := false
	containers := findMapKey(n, "containers")
	if containers == nil {
		fmt.Fprintln(os.Stderr, "spec.containers is required")
		return true
	}
	if containers.Kind != yaml.SequenceNode {
		printErr(file, containers.Line, "spec.containers must be array")
		return true
	}
	for _, c := range containers.Content {
		hasErr = validateContainer(file, c) || hasErr
	}
	return hasErr
}

func validateContainer(file string, c *yaml.Node) bool {
	hasErr := false
	name := findMapKey(c, "name")
	if name == nil {
		fmt.Fprintln(os.Stderr, "containers.name is required")
		hasErr = true
	} else if !isSnake(name.Value) {
		printErr(file, name.Line, fmt.Sprintf("containers.name has invalid format '%s'", name.Value))
		hasErr = true
	}

	image := findMapKey(c, "image")
	if image == nil {
		fmt.Fprintln(os.Stderr, "containers.image is required")
		hasErr = true
	} else if !strings.HasPrefix(image.Value, "registry.bigbrother.io/") || !strings.Contains(image.Value, ":") {
		printErr(file, image.Line, fmt.Sprintf("containers.image has invalid format '%s'", image.Value))
		hasErr = true
	}

	res := findMapKey(c, "resources")
	if res == nil {
		fmt.Fprintln(os.Stderr, "containers.resources is required")
		hasErr = true
	}
	return hasErr
}

func findMapKey(n *yaml.Node, key string) *yaml.Node {
	if n == nil || n.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i < len(n.Content)-1; i += 2 {
		if n.Content[i].Value == key {
			return n.Content[i+1]
		}
	}
	return nil
}

func isSnake(s string) bool {
	for i, r := range s {
		if !(r == '_' || r >= 'a' && r <= 'z' || r >= '0' && r <= '9') {
			return false
		}
		if i == 0 && r == '_' {
			return false
		}
	}
	return true
}

func toInt(s string) (int, error) {
	return strconv.Atoi(s)
}
