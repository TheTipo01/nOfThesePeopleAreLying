package main

import (
	"fmt"
	"github.com/spf13/viper"
	"math/rand"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

var (
	token  string
	prefix string
	game   = make(map[string]*Game)
)

const (
	playEmoji = "🎮"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found
			fmt.Println("Config file not found! See example_config.yml")
			return
		}
	} else {
		// Config file found
		token = viper.GetString("token")
		prefix = viper.GetString("prefix")

	}
}

func main() {

	if token == "" {
		fmt.Println("No token provided. Please modify config.yml")
		return
	}

	if prefix == "" {
		fmt.Println("No prefix provided. Please modify config.yml")
		return
	}

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)
	dg.AddHandler(reactionAdd)
	dg.AddHandler(reactionRemove)

	// In this example, we only care about receiving m events.
	dg.Identify.Intents = discordgo.MakeIntent(discordgo.IntentsGuildMessages | discordgo.IntentsGuildMessageReactions | discordgo.IntentsDirectMessages)

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	_ = dg.Close()
}

// This function will be called (due to AddHandler above) every time a new
// m is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself and all the messages from bots
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	if m.Content == prefix+"play" && game[m.GuildID] == nil {
		sntM, err := s.ChannelMessageSend(m.ChannelID, "To play, add a reaction!")
		if err != nil {
			fmt.Println(err)
			return
		}

		err = s.MessageReactionAdd(m.ChannelID, sntM.ID, playEmoji)
		if err != nil {
			fmt.Println(err)
			_ = s.ChannelMessageDelete(m.ChannelID, sntM.ID)
			return
		}

		game[m.GuildID] = &Game{sntM, nil, false, "", 0, m.ChannelID, m.GuildID, false, ""}

		return
	}

	if m.Content == prefix+"start" && game[m.GuildID] != nil {

		_ = s.ChannelMessageDelete(game[m.GuildID].m.ChannelID, game[m.GuildID].m.ID)
		game[m.GuildID].m = nil

		game[m.GuildID].round++

		gamer := getRand(m.GuildID, false)
		game[m.GuildID].guesser = gamer.id

		_, _ = s.ChannelMessageSend(m.ChannelID, "Round "+strconv.Itoa(game[m.GuildID].round)+" started!\n"+"<@"+gamer.id+"> needs to guess! to guess!\nSend your article in private to me!")

		return
	}

	// Private messages
	if m.GuildID == "" {
		guild := getGuildFromUser(m.Author.ID)
		if guild != "" {
			// Check if the guesser guessed something
			if game[guild].response && game[guild].guesser == m.Author.ID {
				game[guild].response = false

				if didYoUGuess(guild, m.Content) {
					updatePoint(guild, true)
					_, _ = s.ChannelMessageSend(game[guild].channel, "Correct!\nUpdated leaderboard: \n"+leaderboard(guild))
				} else {
					updatePoint(guild, false)
					_, _ = s.ChannelMessageSend(game[guild].channel, "Wrong! The correct user was "+game[guild].players[game[guild].ownerArticle].username+"!\nUpdated leaderboard: \n"+leaderboard(guild))
				}

				// New round
				game[guild].round++

				gamer := getRand(guild, false)
				game[guild].guesser = gamer.id

				_, _ = s.ChannelMessageSend(game[guild].channel, "Round "+strconv.Itoa(game[guild].round)+" started!\n"+"<@"+gamer.id+"> needs to guess!\nSend your article in private to me!")

				return
			}

			if game[guild].guesser == m.Author.ID {
				_, _ = s.ChannelMessageSend(m.ChannelID, "You need to guess!\nWait for your friends to finish sending articles in!")
				return
			}

			game[guild].players[m.Author.ID].article = m.Content
			_, _ = s.ChannelMessageSend(m.ChannelID, "Got your article!")

			if haveWeFinished(guild) {
				random := getRand(guild, true)
				game[guild].ownerArticle = random.id

				game[guild].response = true
				_, _ = s.ChannelMessageSend(game[guild].channel, "All articles are in!\nThe selected one is: "+random.article+"\nAnswer in private with only the username!")
			}
		}

		return
	}
}

func reactionAdd(s *discordgo.Session, r *discordgo.MessageReactionAdd) {

	// Ignore all messages created by the bot itself
	if r.UserID == s.State.User.ID {
		return
	}

	if game[r.GuildID] != nil && !game[r.GuildID].started && game[r.GuildID].m != nil && r.MessageID == game[r.GuildID].m.ID && r.Emoji.Name == playEmoji {
		u, err := s.GuildMember(r.GuildID, r.UserID)
		if err != nil {
			fmt.Println(err)
			return
		}

		if game[r.GuildID].players == nil {
			game[r.GuildID].players = make(map[string]*Gamer)
		}

		game[r.GuildID].players[r.UserID] = &Gamer{r.UserID, 0, u.User.Username, ""}
	}

}

func reactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {

	// Ignore all messages created by the bot itself
	if r.UserID == s.State.User.ID {
		return
	}

	if game[r.GuildID] != nil && !game[r.GuildID].started && r.MessageID == game[r.GuildID].m.ID && r.Emoji.Name == playEmoji {
		game[r.GuildID].players[r.UserID] = nil
	}

}
