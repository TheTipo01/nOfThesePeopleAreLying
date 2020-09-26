package main

import "github.com/bwmarrin/discordgo"

type gamer struct {
	// User id
	id string
	// Points of the user
	points int
	// Nickname of the gamer
	username string
	// Article
	article string
}

type game struct {
	// Message sent by the bot
	m *discordgo.Message
	// Maps of the people playing
	players map[string]*gamer
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
	// Do we except a response from the guesser?
	response bool
	// Selected user for the article
	ownerArticle string
}
