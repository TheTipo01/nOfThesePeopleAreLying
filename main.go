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
	games  = make(map[string]*game)
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

	// Create new game
	if m.Content == prefix+"play" && games[m.GuildID] == nil {
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

		games[m.GuildID] = &game{sntM, nil, false, "", 0, m.ChannelID, m.GuildID, false, "", nil, ""}

		return
	}

	// Starts the game
	if m.Content == prefix+"start" && games[m.GuildID] != nil {

		_ = s.ChannelMessageDelete(games[m.GuildID].m.ChannelID, games[m.GuildID].m.ID)
		games[m.GuildID].m = nil

		games[m.GuildID].round++

		gamer := getRand(m.GuildID, false)
		games[m.GuildID].guesser = gamer.id

		mex, _ := s.ChannelMessageSend(m.ChannelID, "Round "+strconv.Itoa(games[m.GuildID].round)+" started!\n"+"<@"+gamer.id+"> needs to guess! to guess!\nSend your article in private to me!")

		// Add the message, to delete it later
		games[m.GuildID].messages = append(games[m.GuildID].messages, *mex)
		return
	}

	// Remove player from the game
	if m.Content == prefix+"remove" && games[m.GuildID] != nil {
		games[m.GuildID].players[m.ID] = nil
		mex, _ := s.ChannelMessageSend(m.GuildID, "You have been removed from the game!")
		// Add the message, to delete it later
		games[m.GuildID].messages = append(games[m.GuildID].messages, *mex)

		return
	}

	// Stop the game
	if m.Content == prefix+"stop" && games[m.GuildID] != nil {
		games[m.GuildID] = nil
		mex, _ := s.ChannelMessageSend(m.GuildID, "Game has been stopped!")

		time.Sleep(time.Second)
		_ = s.ChannelMessageDelete(mex.ChannelID, mex.ID)
	}

	// Private messages, for doing all sorts of things
	if m.GuildID == "" {
		guild := getGuildFromUser(m.Author.ID)
		if guild != "" {
			// Check if the guesser guessed something
			if games[guild].response && games[guild].guesser == m.Author.ID {
				games[guild].response = false

				// Remove old messages
				removeMessages(s, games[guild].messages)
				games[guild].messages = nil

				if didYoUGuess(guild, m.Content) {
					updatePoint(guild, true)
					mex, _ := s.ChannelMessageSend(games[guild].channel, "Correct!\nUpdated leaderboard: \n"+leaderboard(guild))
					// Add the message, to delete it later
					games[m.GuildID].messages = append(games[m.GuildID].messages, *mex)
				} else {
					updatePoint(guild, false)
					mex, _ := s.ChannelMessageSend(games[guild].channel, "Wrong! The correct user was "+games[guild].players[games[guild].choosenOne].username+"!\nUpdated leaderboard: \n"+leaderboard(guild))
					// Add the message, to delete it later
					games[m.GuildID].messages = append(games[m.GuildID].messages, *mex)
				}

				// New round
				games[guild].round++

				gamer := getRand(guild, false)
				games[guild].guesser = gamer.id

				mex, _ := s.ChannelMessageSend(games[guild].channel, "Round "+strconv.Itoa(games[guild].round)+" started!\n"+"<@"+gamer.id+"> needs to guess!\nSend your article in private to me!")
				// Add the message, to delete it later
				games[m.GuildID].messages = append(games[m.GuildID].messages, *mex)

				return
			}

			if games[guild].guesser == m.Author.ID {
				_, _ = s.ChannelMessageSend(m.ChannelID, "You need to guess!\nWait for your friends to finish sending articles in!")
				return
			}

			games[guild].players[m.Author.ID].article = m.Content
			_, _ = s.ChannelMessageSend(m.ChannelID, "Got your article!")

			if haveWeFinished(guild) {
				random := getRand(guild, true)
				games[guild].choosenOne = random.id

				games[guild].response = true
				mex, _ := s.ChannelMessageSend(games[guild].channel, "All articles are in!\nThe selected one is: "+random.article+"\nAnswer in private with only the username!")
				// Add the message, to delete it later
				games[m.GuildID].messages = append(games[m.GuildID].messages, *mex)
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

	if games[r.GuildID] != nil && !games[r.GuildID].started && games[r.GuildID].m != nil && r.MessageID == games[r.GuildID].m.ID && r.Emoji.Name == playEmoji {
		u, err := s.GuildMember(r.GuildID, r.UserID)
		if err != nil {
			fmt.Println(err)
			return
		}

		if games[r.GuildID].players == nil {
			games[r.GuildID].players = make(map[string]*gamer)
		}

		games[r.GuildID].players[r.UserID] = &gamer{r.UserID, 0, u.User.Username, ""}
	}

}

func reactionRemove(s *discordgo.Session, r *discordgo.MessageReactionRemove) {

	// Ignore all messages created by the bot itself
	if r.UserID == s.State.User.ID {
		return
	}

	if games[r.GuildID] != nil && !games[r.GuildID].started && r.MessageID == games[r.GuildID].m.ID && r.Emoji.Name == playEmoji {
		games[r.GuildID].players[r.UserID] = nil
	}

}
