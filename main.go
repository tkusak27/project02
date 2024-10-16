package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

// Global variables and structures
var (
	funcMap = template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
	}
	templates = template.Must(template.New("").Funcs(funcMap).ParseFiles(
		"./templates/base.html",
		"./templates/body.html",
		"./templates/game.html",
	))
	categories []Category
	sessions   = make(map[string]*Session)
)

type Category struct {
	Category string   `json:"category"`
	KeyWord  string   `json:"key_word"`
	Hints    []string `json:"hints"`
}

type Session struct {
	Category   string
	KeyWord    string
	Hints      []string
	HintIndex  int
	Guesses    []string
	Won        bool
	ExtraHints []string
}

// Middleware for logging requests and response times
func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		req := fmt.Sprintf("%s %s", r.Method, r.URL)
		log.Println(req)
		next.ServeHTTP(w, r)
		log.Println(req, "completed in", time.Since(start))
	})
}

// Handler for serving static files
func public() http.Handler {
	return http.StripPrefix("/public/", http.FileServer(http.Dir("./public")))
}

// Handler for the homepage
func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b := struct {
			Title        template.HTML
			BusinessName string
			Slogan       string
		}{
			Title:        template.HTML("Business &verbar; Landing"),
			BusinessName: "Business,",
			Slogan:       "we get things done.",
		}
		err := templates.ExecuteTemplate(w, "base", &b)
		if err != nil {
			http.Error(w, fmt.Sprintf("index: couldn't parse template: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

// Function to generate a clue for the target word
func getClue(word string) string {
	// Show the first and last letters, and underscores for the rest
	clue := string(word[0])
	for i := 1; i < len(word)-1; i++ {
		clue += "_"
	}
	clue += string(word[len(word)-1])
	return clue
}

// Function to get or create a session ID
func getSessionID(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil {
		sessionID := fmt.Sprintf("%d", rand.Int())
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: sessionID,
			Path:  "/",
		})
		return sessionID
	}
	return cookie.Value
}

// Initialize a session
func getSession(sessionID string) *Session {
	session, exists := sessions[sessionID]
	if !exists || session.Won || len(session.Guesses) >= 20 {
		// Start a new session
		category := categories[rand.Intn(len(categories))]
		session = &Session{
			Category:   category.Category,
			KeyWord:    category.KeyWord,
			Hints:      category.Hints,
			HintIndex:  0,
			Guesses:    []string{},
			Won:        false,
			ExtraHints: []string{},
		}
		sessions[sessionID] = session
	}
	return session
}

// Handler for the game logic
func game() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSessionID(w, r)
		session := getSession(sessionID)

		if r.Method == http.MethodPost {
			// Handle the user's guess
			err := r.ParseForm()
			if err != nil {
				http.Error(w, "Failed to parse form", http.StatusBadRequest)
				return
			}
			userGuess := r.FormValue("guess")
			if userGuess == "" {
				http.Error(w, "Guess cannot be empty", http.StatusBadRequest)
				return
			}
			session.Guesses = append(session.Guesses, userGuess)
			if userGuess == session.KeyWord {
				session.Won = true
			} else {
				if session.HintIndex < len(session.Hints) {
					session.HintIndex++
				}
				// Provide extra hints at certain attempts
				if len(session.Guesses) == 3 {
					session.ExtraHints = append(session.ExtraHints, fmt.Sprintf("Category: %s", session.Category))
				}
				if len(session.Guesses) == 5 {
					session.ExtraHints = append(session.ExtraHints, fmt.Sprintf("Word Length: %d letters", len(session.KeyWord)))
				}
			}
			sessions[sessionID] = session
		}

		// Ensure HintIndex is at least 1
		if session.HintIndex == 0 {
			session.HintIndex = 1
		}

		// Prepare data for the template
		data := struct {
			Title       string
			CurrentHint string
			AllHints    []string
			ExtraHints  []string
			Guesses     []string
			Attempts    int
			MaxAttempts int
			Won         bool
			KeyWord     string
		}{
			Title:       "Word Guessing Game",
			CurrentHint: session.Hints[session.HintIndex-1],
			AllHints:    session.Hints[:session.HintIndex],
			ExtraHints:  session.ExtraHints,
			Guesses:     session.Guesses,
			Attempts:    len(session.Guesses),
			MaxAttempts: 20,
			Won:         session.Won,
			KeyWord:     session.KeyWord,
		}

		err := templates.ExecuteTemplate(w, "game", data)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
			return
		}
	})
}

func main() {
	// Load the categories from words.json
	file, err := os.Open("words.json")
	if err != nil {
		log.Fatalf("Failed to open words.json: %v", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	var data struct {
		Categories []Category `json:"categories"`
	}
	err = decoder.Decode(&data)
	if err != nil {
		log.Fatalf("Failed to decode words.json: %v", err)
	}
	categories = data.Categories

	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	// Set up the HTTP server and routes
	mux := http.NewServeMux()
	mux.Handle("/public/", logging(public()))
	mux.Handle("/", logging(index()))
	mux.Handle("/game", logging(game()))

	// Port configuration
	port, ok := os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	addr := fmt.Sprintf(":%s", port)
	server := http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}

	log.Println("main: running simple server on port", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("main: couldn't start simple server: %v\n", err)
	}
}
