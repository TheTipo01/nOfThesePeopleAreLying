package main

import "github.com/bwmarrin/discordgo"

type Gamer struct {
	// User id
	id string
	// Points of the user
	points int
	// Nickname of the gamer
	username string
	// Article
	article string
}

type Game struct {
	// Message sent by the bot
	m *discordgo.Message
	// Maps of the people playing
	players map[string]*Gamer
	// Has the game started?
	started bool
	// Who guesses this round
	guesser string
	// Round
	round int
	// Channel
	channel string
	// Guild
	guild string
}
