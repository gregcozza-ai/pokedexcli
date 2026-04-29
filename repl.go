package main

import "strings"

func cleanInput(text string) []string {
    // Trim leading/trailing whitespace and convert to lowercase
    trimmed := strings.TrimSpace(text)
    if trimmed == "" {
        return []string{}
    }

    // Split by whitespace and lowercase each word
    words := strings.Fields(strings.ToLower(trimmed))
    return words
}