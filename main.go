package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"
)

var (
	funcMap = template.FuncMap{
		"sub": func(a, b int) int {
			return a - b
		},
		"until": func(count int) []int {
			nums := make([]int, count)
			for i := 0; i < count; i++ {
				nums[i] = i
			}
			return nums
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
	Category  string
	KeyWord   string
	Hints     []string
	HintIndex int
	Guesses   []string
	Won       bool
	ExpiresAt time.Time
}

// Load word categories from words.json
func loadCategories() {
	file, err := os.Open("words.json")
	if err != nil {
		log.Fatalf("Failed to open words.json: %v", err)
	}
	defer file.Close()

	var data struct {
		Categories []Category `json:"categories"`
	}
	if err := json.NewDecoder(file).Decode(&data); err != nil {
		log.Fatalf("Failed to decode words.json: %v", err)
	}
	categories = data.Categories
}

// Preload 5 random sessions from the categories
func preloadSessions(count int) {
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator

	// Shuffle the categories to ensure uniqueness
	rand.Shuffle(len(categories), func(i, j int) {
		categories[i], categories[j] = categories[j], categories[i]
	})

	// Use the first few categories to create sessions
	for i := 0; i < count && i < len(categories); i++ {
		category := categories[i]
		sessionID := fmt.Sprintf("preloaded-%d", i+1)

		sessions[sessionID] = &Session{
			Category:  category.Category,
			KeyWord:   category.KeyWord,
			Hints:     category.Hints,
			HintIndex: 0,
			Guesses:   []string{},
			Won:       false,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}

		log.Printf("Preloaded session %s: %s - %s", sessionID, category.Category, category.KeyWord)
	}
}

// Middleware for logging requests
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
		data := struct {
			Title        template.HTML
			BusinessName string
			Slogan       string
		}{
			Title:        template.HTML("Word Guessing Game"),
			BusinessName: "Welcome to the Word Guessing Game!",
			Slogan:       "Test your vocabulary and have fun!",
		}
		if err := templates.ExecuteTemplate(w, "base", &data); err != nil {
			http.Error(w, fmt.Sprintf("index: couldn't parse template: %v", err), http.StatusInternalServerError)
		}
	})
}

// Function to get or create a session ID
func getSessionID(w http.ResponseWriter, r *http.Request) string {
	cookie, err := r.Cookie("session_id")
	if err != nil { // No session ID, assign a preloaded session if available
		sessionID := assignPreloadedSessionID()
		if sessionID == "" { // No more preloaded sessions, generate a new one
			sessionID = fmt.Sprintf("%d", rand.Int())
		}
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: sessionID,
			Path:  "/",
		})
		return sessionID
	}
	return cookie.Value
}

func assignPreloadedSessionID() string {
	for id, session := range sessions {
		if strings.HasPrefix(id, "preloaded-") && session.ExpiresAt.After(time.Now()) {
			return id // Return the first available preloaded session
		}
	}
	return "" // No valid preloaded session found
}

// Initialize a session
func getSession(sessionID string) *Session {
	session, exists := sessions[sessionID]

	// If session doesn't exist, expired, or exhausted, assign a new one
	if !exists || session.ExpiresAt.Before(time.Now()) || session.Won || len(session.Guesses) >= 5 {
		category := categories[rand.Intn(len(categories))]
		session = &Session{
			Category:  category.Category,
			KeyWord:   category.KeyWord,
			Hints:     category.Hints,
			HintIndex: 0,
			Guesses:   []string{},
			Won:       false,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		sessions[sessionID] = session
	}
	return session
}

// Cleanup expired sessions
func startSessionCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	for {
		<-ticker.C
		now := time.Now()
		for id, session := range sessions {
			if session.ExpiresAt.Before(now) {
				delete(sessions, id)
				log.Printf("Session %s expired and removed", id)
			}
		}
	}
}

// Handler for the game logic
func game() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSessionID(w, r)
		session := getSession(sessionID)

		if r.Method == http.MethodPost {
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

			if strings.EqualFold(userGuess, session.KeyWord) {
				session.Won = true
			} else if session.HintIndex < len(session.Hints) {
				session.HintIndex++
			}

			sessions[sessionID] = session
		}

		if session.HintIndex == 0 {
			session.HintIndex = 1
		}

		data := struct {
			Title       string
			CurrentHint string
			AllHints    []string
			Guesses     []string
			Attempts    int
			MaxAttempts int
			Won         bool
			KeyWord     string
		}{
			Title:       "Word Guessing Game",
			CurrentHint: session.Hints[session.HintIndex-1],
			AllHints:    session.Hints[:session.HintIndex],
			Guesses:     session.Guesses,
			Attempts:    len(session.Guesses),
			MaxAttempts: 5,
			Won:         session.Won,
			KeyWord:     session.KeyWord,
		}

		if err := templates.ExecuteTemplate(w, "game", data); err != nil {
			http.Error(w, fmt.Sprintf("Failed to render template: %v", err), http.StatusInternalServerError)
		}
	})
}

func main() {
	loadCategories()   // Load categories from words.json
	preloadSessions(5) // Preload 5 random sessions

	go startSessionCleanup(1 * time.Minute)

	mux := http.NewServeMux()
	mux.Handle("/public/", logging(public()))
	mux.Handle("/", logging(index()))
	mux.Handle("/game", logging(game()))

	port := os.Getenv("PORT")
	if port == "" {
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

	log.Printf("main: running word guessing game server on port %s", port)
	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("main: couldn't start server: %v", err)
	}
}
