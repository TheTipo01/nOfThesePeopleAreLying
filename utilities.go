package main

import (
	"math/rand"
	"sort"
	"strconv"
	"strings"
)

func getRand(guild string, ignoreGuesser bool) *gamer {
	// produce a pseudo-random number between 0 and len(a)-1
retry:

	i := int(float32(len(games[guild].players)) * rand.Float32())
	for _, v := range games[guild].players {
		if i == 0 {
			if ignoreGuesser {
				if v.article == "" {
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
	return strings.ToLower(games[guild].players[games[guild].ownerArticle].username) == strings.ToLower(username)
}

func leaderboard(guild string) string {
	// Sort the players
	var players []gamer
	for _, p := range games[guild].players {
		players = append(players, *p)
	}

	sort.Slice(players, func(i, j int) bool {
		return players[i].points < players[j].points
	})

	// Create string
	var message string
	for _, p := range players {
		message += p.username + " - " + strconv.Itoa(p.points) + "\n"
	}

	return message
}

func updatePoint(guild string, didYouWin bool) {
	for _, g := range games[guild].players {
		g.article = ""
	}

	if didYouWin {
		games[guild].players[games[guild].ownerArticle].points++
		games[guild].players[games[guild].guesser].points++
	} else {
		games[guild].players[games[guild].ownerArticle].points++
	}
}
