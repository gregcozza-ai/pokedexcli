package main

import (
 "fmt"
 "bufio"
 "os" 
 "encoding/json"
 "net/http"
 "io"
 "time"
 "math/rand"
 "github.com/gregcozza-ai/pokedexcli/internal/pokecache"
)

type config struct {
	NextURL		string
	PrevURL		string
	CaughtPokemon	map[string]PokemonInfo
}

type PokemonInfo struct {
	Name 	string
	Height 	int
	Weight 	int
	Stats	[]Stat
	Types	[]string
}

type Stat struct {
	Name	string
	Value	int
}

type LocationAreaResponse struct {
	Results     []struct {
		Name string `json:"name"`
	} `json:"results"`
	Next     string `json:"next"`
	Previous string `json:"previous"`
}

type FullPokemon struct {
	Name		string `json:"name"`
	Height		int `json:"height"`
	Weight		int `json:"weight"`
	BaseExperience	int `json."base_experience"`
	Stats		[]struct {
		BaseStat int `json:"base_stat"`
		Stat	struct {
			Name string `json:"name"`
		} `json:"stat"`
	} `json:"stats"`
	Types []struct {
		Type struct {
			Name string `json:"name"`
		} `json:"type"`
	} `json:"types"`
} 

type PokemonResponse struct {
	BaseExperience int `json:"base_experience"`
	Name string `json:"name"`
}

type PokemonEncounter struct {
	Pokemon struct {
		Name string `json:"name"`
	}
}

type LocationArea struct {
	PokemonEncounters []struct {
		Pokemon struct {
			Name string `json:"name"`
		} `json:"pokemon"`
	} `json:"pokemon_encounters"`
}

type cliCommand struct {
	name		string
	description 	string
	callback	func(*config, *pokecache.Cache, []string) error
}

var cache *pokecache.Cache

func main() {
    scanner := bufio.NewScanner(os.Stdin)
    cache = pokecache.NewCache(5 * time.Minute)
    cfg := &config{
	CaughtPokemon: make(map[string]PokemonInfo),
    }
    commands := makeCommands(cfg)
    // Initalize random seed
    rand.Seed(time.Now().UnixNano())

    for {
	fmt.Print("Pokedex > ")
	if !scanner.Scan() {
		break
	}
	input := scanner.Text()
	words := cleanInput(input)
	if len(words) == 0 {
		continue
	}
	
	cmdName := words[0]
	args := words[1:]
	cmd, exists := commands[cmdName]
	if !exists {
		fmt.Println("Unknown command")
		continue
	}

	if err := cmd.callback(cfg, cache, args); err != nil {
		fmt.Println(err)
	}
    } 
}

func makeCommands(cfg *config) map[string]cliCommand {
	return map[string]cliCommand {
		"exit": {
			name:		"exit",
			description:	"Exit the Pokedex", 
			callback:	commandExit,
		},
		"help": {
			name:		"help",
			description:	"Displays a help message",
			callback:	commandHelp,
		},
		"map": {
			name: 		"map",
			description:	"Displays the next 20 location areas",
			callback:	commandMap,
		},
		"mapb": {
			name:		"mapb",
			description:	"Displays the previous 20 location areas",
			callback:	commandMapb,
		},
		"explore": {
			name:		"explore",
			description: 	"Explore a location area and see Pokemon encounters",
			callback:	commandExplore,
		},
		"catch": {
			name:		"catch",
			description:	"Attempt to catch a Pokemon",
			callback:	commandCatch,
		},
		"inspect": {
			name:		"inspect",
			description:	"Inspect a caught Pokemon",
			callback:	commandInspect,
		},
		"pokedex": {
			name:		"podedex",
			description:	"List all caught Pokemon",
			callback:	commandPokedex,
		},
	}
}

func commandExit(cfg *config, cache *pokecache.Cache, args []string) error {
	fmt.Println("Closing the Pokedex... Goodbye!")
	os.Exit(0)
	return nil
}

func commandHelp(cfg *config, cache *pokecache.Cache, args []string) error {
	fmt.Println("Welcome to the Pokedex!")
	fmt.Println("Usage:")
	for _, cmd := range makeCommands(cfg) {
		fmt.Printf("%s: %s\n", cmd.name, cmd.description)
	}
	return nil
}

func commandMap(cfg *config, cache *pokecache.Cache, args []string) error {
	url := "https://pokeapi.co/api/v2/location-area"
	if cfg.NextURL != "" {
		url = cfg.NextURL
	}

	// Check cache first
	if cachedData, ok := cache.Get(url); ok {
		fmt.Println("Using cached data for", url)
		var locResp LocationAreaResponse
		if err := json.Unmarshal(cachedData, &locResp); err != nil {
			fmt.Println("Cache corrupted, making new request")
		} else {
			for _, area := range locResp.Results {
				fmt.Println(area.Name)
			}
			cfg.NextURL = locResp.Next
			cfg.PrevURL = locResp.Previous
			return nil
		}
	}

	// Make request if not in cache
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch location areas: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Cache the response
	cache.Add(url, body)

	// Parse and print results
	var locResp LocationAreaResponse
	if err := json.Unmarshal(body, &locResp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	for _, area := range locResp.Results {
		fmt.Println(area.Name)
	}

	cfg.NextURL = locResp.Next
	cfg.PrevURL = locResp.Previous
	return nil
}

func commandMapb(cfg *config, cache *pokecache.Cache, args []string) error {
	if cfg.PrevURL == "" {
		return fmt.Errorf("you're on the first page")
	}

	// Check cache for previous URL
	if cachedData, ok := cache.Get(cfg.PrevURL); ok {
		fmt.Println("Using cached data for", cfg.PrevURL)
		var locResp LocationAreaResponse
		if err := json.Unmarshal(cachedData, &locResp); err != nil {
			fmt.Println("Cache corrupted, making new request")
		} else {
			for _, area := range locResp.Results {
				fmt.Println(area.Name)
			}
			cfg.NextURL = locResp.Next
			cfg.PrevURL = locResp.Previous
			return nil
		}
	}

	// Make request for previous URL
	resp, err := http.Get(cfg.PrevURL)
	if err != nil {
		return fmt.Errorf("failed to fetch location areas: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	// Cache the response
	cache.Add(cfg.PrevURL, body)

	// Parse and print results
	var locResp LocationAreaResponse
	if err := json.Unmarshal(body, &locResp); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	for _, area := range locResp.Results {
		fmt.Println(area.Name)
	}
	
	cfg.NextURL = locResp.Next
	cfg.PrevURL = locResp.Previous
	return nil 
}

func commandExplore(cfg *config, cache *pokecache.Cache, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("explore command requires a location area name")
	}
	areaName := args[0]
	url := "https://pokeapi.co/api/v2/location-area/" + areaName

	if cachedData, ok := cache.Get(url); ok {
		fmt.Println("Using cached data for", url)
		var locArea LocationArea
		if err := json.Unmarshal(cachedData, &locArea); err != nil {
			return fmt.Errorf("cache corrupted: %v", err)
		}
		return printPokemonEncounters(locArea.PokemonEncounters)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch location area %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)
	}

	cache.Add(url, body)

	var locArea LocationArea
	if err := json.Unmarshal(body, &locArea); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	return printPokemonEncounters(locArea.PokemonEncounters)
}

func printPokemonEncounters(encounters []struct {
	Pokemon struct {
		Name string `json:"name"`
	} `json:"pokemon"`
}) error{
	if len(encounters) == 0 {
		return fmt.Errorf("no Pokemon found in location area")
	}
	fmt.Printf("Exploring %s...\n", encounters[0].Pokemon.Name) //Use the first Pokemon's name for the area name
	fmt.Println("Found Pokemon:")
	for _, encounter := range encounters {
		fmt.Printf(" - %s\n", encounter.Pokemon.Name)
	}
	return nil
}

func commandCatch(cfg *config, cache *pokecache.Cache, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("catch command requires a Pokemon name")
	}
	pokemonName := args[0]
	url := "https://pokeapi.co/api/v2/pokemon/" + pokemonName

	// Check cache first
	if cachedData, ok := cache.Get(url); ok {
		fmt.Println("Using cached data for", url)
		var fullPokemon FullPokemon
		if err := json.Unmarshal(cachedData, &fullPokemon); err != nil {
			return fmt.Errorf("cache corrupted: %v", err)
		}
		return attemptCatch(cfg, fullPokemon, pokemonName)
	}
	// Make request if not in cache
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch Pokemon: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, resp.Status)
	}
	
	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %v", err)	
	}

	// Cache the response
	cache.Add(url, body)

	// Parse response
	var fullPokemon FullPokemon
	if err := json.Unmarshal(body, &fullPokemon); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}
	
	return attemptCatch(cfg, fullPokemon, pokemonName)
}

func attemptCatch(cfg *config, fullPokemon FullPokemon, pokemonName string) error {
	fmt.Printf("Throwing a Pokeball at %s...\n", pokemonName)

	// Calculate catch chance (100 - base_experience/10)
	chance := 100 - (fullPokemon.BaseExperience / 10)
	if chance < 0 {
		chance = 0
	}

	// Generate random number (0-99)
	if rand.Intn(100) < chance { 
		// Create PokemonInfo from full data
		var pokemonInfo PokemonInfo
		pokemonInfo.Name = fullPokemon.Name
		pokemonInfo.Height = fullPokemon.Height
		pokemonInfo.Weight = fullPokemon.Weight

		// Process stats
		for _, stat := range fullPokemon.Stats {
			pokemonInfo.Stats = append(pokemonInfo.Stats, Stat{
				Name:	stat.Stat.Name,
				Value:	stat.BaseStat,
			})
		}
		// Process types
		for _, t := range fullPokemon.Types {
			pokemonInfo.Types = append(pokemonInfo.Types, t.Type.Name)
		}
		
		// Store in Pokedex
		cfg.CaughtPokemon[pokemonName] = pokemonInfo
		fmt.Printf("%s was caught!\n", pokemonName)
		fmt.Println("You may now inspect it with the inspect command.")
	} else {
		fmt.Printf("%s escaped!\n", pokemonName)
	}
	return nil
}

func commandInspect(cfg *config, cache *pokecache.Cache, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("inspect command requires a Pokemon name")
	}
	pokemonName := args[0]

	// Check if caught
	pokemon, exists := cfg.CaughtPokemon[pokemonName]
	if !exists {
		return fmt.Errorf("you have not caught that pokemon")
	}

	// Print Pokemon info
	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %d\n", pokemon.Height)
	fmt.Printf("Weight: %d\n", pokemon.Weight)
	fmt.Println("Stats:")
	for _, stat := range pokemon.Stats {
		fmt.Printf("  -%s: %d\n", stat.Name, stat.Value)
	}
	fmt.Println("Types:")
	for _, t := range pokemon.Types {
		fmt.Printf("  -%s\n", t)
	}
	return nil 
}

func commandPokedex(cfg *config, cache *pokecache.Cache, args []string) error {
	if len(cfg.CaughtPokemon) == 0 {
		fmt.Println("Your Pokedex is empty.")
		return nil
	}

	fmt.Println("Your Pokedex:")
	for name := range cfg.CaughtPokemon {
		fmt.Printf(" -%s\n", name)
	}
	return nil
}



