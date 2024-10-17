# project02
project02 - Categories Game

Our Project02, the Categories game, aims to solve the universal human dilema of boredom by providing a fun and addicting web-based word guessing game utilizing the programming language Go.

The main goal of the game is to challenge players to guess a hidden word from a selected category in under 20 guesses. The player is aided by hints the game will display after each incorrect guess.

The Categories game leverages sessions to keep track of each player's progress and provides clean and interactive user interface via Go's templating system. Truly world changing gameplay humanity might not be ready for.

Key gameplay fetures include:
1. Multiple categories of words with corresponding hints
2. Incremental gameplay as incorrect guesses reveal new hints
3. Session-based tracking of a player's progress, including guesses and hints
4. A simple and sleek web interface using Go's templating system that users are sure to be engrossed by
5. Concurrency for session management and request handling

Unique Go features utilized:
- functional programming elements (a custom function for subtraction is passed to Go's template system to manipulate our game's HTML template)
- Concurrency (utilize goroutines for session cleanup, which runs concurrenetly with the main web server)
- Error Handling with Explicit Patterns (Errors are handled consistently and logged. Example of this is the error check for a failure to parse form data)

Other ways our Categories game takes advantage of Go:
- Utlizing static types, we caught all type mismatch errors at compile while debugging and not while running the game. Avoiding unforseen crashes for players.
- go simplified our web developement for Categories by leveraging Go's standard library that includes everything from HTTP server management, JSON parsing, and session handling
