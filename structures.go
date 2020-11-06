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
	// Id of who guesses in this round
	guesser string
	// Round
	round int
	// Text channel for where to send messages
	channel string
	// Guild of the game
	guild string
	// Do we except a response from the guesser?
	response bool
	// Selected user for the article
	chosenOne string
	// Old messages, to delete the next round
	messages []*discordgo.Message
	// Id of the previous guesser
	previousGuesser string
}
