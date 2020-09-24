package main

import "math/rand"

func getRand(guild string, ignoreGuesser bool) *Gamer {
	// produce a pseudo-random number between 0 and len(a)-1
retry:

	i := int(float32(len(game[guild].players)) * rand.Float32())
	for _, v := range game[guild].players {
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
	for _, g := range game {
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

	for _, p := range game[guild].players {
		if p.article != "" {
			i++
		}
	}

	return len(game[guild].players)-i == 1
}
