package main

import (
	"github.com/bwmarrin/discordgo"
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

// Returns a random player, and if ignoreGuesser is true, ignores the currently guessing person
func getRand(guild string, ignoreGuesser bool) *gamer {
	// produce a pseudo-random number between 0 and len(a)-1
retry:

	i := int(float32(len(games[guild].players)) * rand.Float32())
	for _, v := range games[guild].players {
		if i == 0 {
			if ignoreGuesser {
				// If we need to ignore the guesser, we check if the user has an empty article, and then try again
				if v.article == "" {
					goto retry
				}
			} else {
				// Else, we ignore the previous guesser (because yes)
				if v.id == games[guild].previousGuesser {
					goto retry
				}
			}
			return v
		}
		i--

	}
	panic("impossible")
}

// Tries to get guild and other info from a user string
func getGuildFromUser(user string) string {
	for _, g := range games {
		for _, p := range g.players {
			if p.id == user {
				return g.guild
			}
		}
	}

	return ""
}

// Checks if all the people have added an article
func haveWeFinished(guild string) bool {
	var i int

	for _, p := range games[guild].players {
		if p.article != "" {
			i++
		}
	}

	return len(games[guild].players)-i == 1
}

// Checks if you have guessed the user who sent the article
func didYoUGuess(guild, username string) bool {
	return strings.ToLower(games[guild].players[games[guild].chosenOne].username) == strings.ToLower(username)
}

// Returns a leaderboard for the current game
func leaderboard(guild string) string {
	// Sort the players
	var players []gamer
	for _, p := range games[guild].players {
		players = append(players, *p)
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].points > players[j].points
	})

	// Create string
	var message string
	for _, p := range players {
		message += p.username + " - " + strconv.Itoa(p.points) + "\n"
	}

	return message
}

// Skips to the next round
func updatePoint(guild, chosen string) {
	for _, g := range games[guild].players {
		g.article = ""
	}

	if chosen == "" {
		games[guild].players[games[guild].chosenOne].points++
		games[guild].players[games[guild].guesser].points++
	} else {
		games[guild].players[chosen].points++
	}
}

// Removes the provided messages, ignoring errors
func removeMessages(s *discordgo.Session, messages []*discordgo.Message) {
	for _, m := range messages {
		_ = s.ChannelMessageDelete(m.ChannelID, m.ID)
	}
}

func searchUser(nickname string) string {
	nickname = strings.ToLower(nickname)

	for _, g := range games {
		for _, p := range g.players {
			if strings.ToLower(p.username) == nickname {
				return p.id
			}
		}
	}

	return ""
}
